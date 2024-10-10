// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tracedurationconnector

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/connector"
	conventions "go.opentelemetry.io/collector/semconv/v1.27.0"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/tracedurationconnector/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpersignal"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/pdatautil"
)

const (
	serviceNameKey = conventions.AttributeServiceName
	// TODO(marctc): formalize these constants in the OpenTelemetry specification.
	spanNameKey     = "span.name" // OpenTelemetry non-standard constan
	durationNameKey = "duration"
)

type logsConnector struct {
	config Config
	logger *zap.Logger

	// Additional dimensions to add to logs.
	dimensions   []pdatautil.Dimension
	logsConsumer consumer.Logs
	component.StartFunc

	component.ShutdownFunc

	// TODO:
	telemetryBuilder *metadata.TelemetryBuilder
	// the event machine handling all operations for this processor
	eventMachine *eventMachine

	// the trace storage
	st storage
}

func newLogsConnector(set connector.Settings, config component.Config) *logsConnector {
	cfg := config.(*Config)

	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil
	}

	// the event machine will buffer up to N concurrent events before blocking
	eventMachine := newEventMachine(set.Logger, 10000, cfg.NumWorkers, cfg.NumTraces, telemetryBuilder)

	lc := &logsConnector{
		logger:           set.Logger,
		config:           *cfg,
		telemetryBuilder: telemetryBuilder,
		eventMachine:     eventMachine,
	}

	// register the callbacks
	eventMachine.onTraceReceived = lc.onTraceReceived
	eventMachine.onTraceExpired = lc.onTraceExpired
	eventMachine.onTraceReleased = lc.onTraceReleased
	eventMachine.onTraceRemoved = lc.onTraceRemoved

	return lc
}

// Start is invoked during service startup.
func (c *logsConnector) Start(_ context.Context, _ component.Host) error {
	// start these metrics, as it might take a while for them to receive their first event
	c.telemetryBuilder.ProcessorGroupbytraceTracesEvicted.Add(context.Background(), 0)
	c.telemetryBuilder.ProcessorGroupbytraceIncompleteReleases.Add(context.Background(), 0)
	c.telemetryBuilder.ProcessorGroupbytraceConfNumTraces.Record(context.Background(), int64(c.config.NumTraces))
	c.eventMachine.startInBackground()
	return c.st.start()
}

// Shutdown is invoked during service shutdown.
func (c *logsConnector) Shutdown(_ context.Context) error {
	c.eventMachine.shutdown()
	return c.st.shutdown()
}

// Capabilities implements the consumer interface.
func (c *logsConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (c *logsConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	var errs error
	for _, singleTrace := range batchpersignal.SplitTraces(td) {
		errs = multierr.Append(errs, c.eventMachine.consume(singleTrace))
	}
	return errs
}

// resourceSpans as a single trace's spans
func (c *logsConnector) exportTracesAsLogs(ctx context.Context, resourceSpans []ptrace.ResourceSpans) error {
	ld := plog.NewLogs()
	sl := c.newScopeLogs(ld)
	var minSpanStartTime, maxSpanEndTime int64
	var serviceName, spanName string
	var traceId pcommon.TraceID
	var spanId pcommon.SpanID
	var stmp pcommon.Timestamp
	for i := 0; i < len(resourceSpans); i++ {
		rspans := resourceSpans[i]
		resourceAttr := rspans.Resource().Attributes()
		serviceAttr, ok := resourceAttr.Get(conventions.AttributeServiceName)
		if !ok {
			continue
		}
		serviceName = serviceAttr.Str()
		ilsSlice := rspans.ScopeSpans()
		for j := 0; j < ilsSlice.Len(); j++ {
			ils := ilsSlice.At(j)
			//ils.Scope().CopyTo(sl.Scope())
			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				starTime := int64(span.StartTimestamp().AsTime().Nanosecond())
				endTime := int64(span.EndTimestamp().AsTime().Nanosecond())

				// parentId is empty represent its root span.
				// 1. root span duration represent trace duration in most common case.
				// 2. Asynchronous scenarios need to calculate duration of the minimum start and maximum end times of the entire span
				if span.ParentSpanID().IsEmpty() {
					traceId = span.TraceID()
					spanId = span.SpanID()
					spanName = span.Name()
					stmp = span.StartTimestamp()
				} else {
					if minSpanStartTime == 0 || starTime < minSpanStartTime {
						minSpanStartTime = starTime
					}

					if maxSpanEndTime == 0 || endTime > maxSpanEndTime {
						maxSpanEndTime = endTime
					}

				}
			}
		}
	}
	c.toLogRecord(sl, serviceName, spanName, traceId, spanId, stmp, maxSpanEndTime-minSpanStartTime)

	return c.exportLogs(ctx, ld)
}

func (c *logsConnector) exportLogs(ctx context.Context, ld plog.Logs) error {
	if err := c.logsConsumer.ConsumeLogs(ctx, ld); err != nil {
		c.logger.Error("failed to convert exceptions to logs", zap.Error(err))
		return err
	}
	return nil
}

func (c *logsConnector) newScopeLogs(ld plog.Logs) plog.ScopeLogs {
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	return sl
}

func (c *logsConnector) attrToLogRecord(sl plog.ScopeLogs, serviceName string, span ptrace.Span, resourceAttrs pcommon.Map) plog.LogRecord {
	logRecord := sl.LogRecords().AppendEmpty()
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
	logRecord.SetSpanID(span.SpanID())
	logRecord.SetTraceID(span.TraceID())
	spanAttrs := span.Attributes()

	// Copy span attributes to the log record.
	// spanAttrs.CopyTo(logRecord.Attributes())

	// Add common attributes to the log record.
	logRecord.Attributes().PutStr(spanNameKey, span.Name())
	logRecord.Attributes().PutStr(serviceNameKey, serviceName)

	// Add configured dimension attributes to the log record.
	for _, d := range c.dimensions {
		if v, ok := pdatautil.GetDimensionValue(d, spanAttrs, resourceAttrs); ok {
			logRecord.Attributes().PutStr(d.Name, v.Str())
		}
	}
	return logRecord
}

func (c *logsConnector) toLogRecord(sl plog.ScopeLogs, serviceName, spanName string, traceId pcommon.TraceID, spanId pcommon.SpanID, stmp pcommon.Timestamp, traceDuration int64) plog.LogRecord {
	logRecord := sl.LogRecords().AppendEmpty()
	logRecord.SetSeverityNumber(plog.SeverityNumberInfo)
	logRecord.SetSpanID(spanId)
	logRecord.SetTraceID(traceId)
	logRecord.SetTimestamp(stmp)

	// Add common attributes to the log record.
	logRecord.Attributes().PutStr(spanNameKey, spanName)
	logRecord.Attributes().PutStr(serviceNameKey, serviceName)
	// Unix Nano
	logRecord.Attributes().PutInt(durationNameKey, traceDuration)
	return logRecord
}

func (c *logsConnector) onTraceReceived(trace tracesWithID, worker *eventMachineWorker) error {
	traceID := trace.id
	if worker.buffer.contains(traceID) {
		c.logger.Debug("trace is already in memory storage")

		// it exists in memory already, just append the spans to the trace in the storage
		if err := c.addSpans(traceID, trace.td); err != nil {
			return fmt.Errorf("couldn't add spans to existing trace: %w", err)
		}

		// we are done with this trace, move on
		return nil
	}

	// at this point, we determined that we haven't seen the trace yet, so, record the
	// traceID in the map and the spans to the storage

	// place the trace ID in the buffer, and check if an item had to be evicted
	evicted := worker.buffer.put(traceID)
	if !evicted.IsEmpty() {
		// delete from the storage
		worker.fire(event{
			typ:     traceRemoved,
			payload: evicted,
		})

		rs, err := c.st.get(evicted)
		if err != nil || rs == nil {
			c.logger.Error("failed to retrieve trace from storage", zap.Error(err), zap.Stringer("traceID", evicted))
		}

		err = c.exportTracesAsLogs(context.Background(), rs)
		if err != nil {
			c.logger.Error("failed to export traces", zap.Error(err), zap.Stringer("traceID", evicted))
		}

		c.telemetryBuilder.ProcessorGroupbytraceTracesEvicted.Add(context.Background(), 1)

		c.logger.Info("trace evicted: in order to avoid this in the future, adjust the wait duration and/or number of traces to keep in memory",
			zap.Stringer("traceID", evicted))
	}

	// we have the traceID in the memory, place the spans in the storage too
	if err := c.addSpans(traceID, trace.td); err != nil {
		return fmt.Errorf("couldn't add spans to existing trace: %w", err)
	}

	c.logger.Debug("scheduled to release trace", zap.Duration("duration", c.config.WaitDuration))

	time.AfterFunc(c.config.WaitDuration, func() {
		// if the event machine has stopped, it will just discard the event
		worker.fire(event{
			typ:     traceExpired,
			payload: traceID,
		})
	})
	return nil
}

func (c *logsConnector) onTraceExpired(traceID pcommon.TraceID, worker *eventMachineWorker) error {
	c.logger.Debug("processing expired", zap.Stringer("traceID", traceID))

	if !worker.buffer.contains(traceID) {
		// we likely received multiple batches with spans for the same trace
		// and released this trace already
		c.logger.Debug("skipping the processing of expired trace", zap.Stringer("traceID", traceID))
		c.telemetryBuilder.ProcessorGroupbytraceIncompleteReleases.Add(context.Background(), 1)
		return nil
	}

	// delete from the map and erase its memory entry
	worker.buffer.delete(traceID)

	// this might block, but we don't need to wait
	c.logger.Debug("marking the trace as released", zap.Stringer("traceID", traceID))
	go func() {
		_ = c.markAsReleased(traceID, worker.fire)
	}()

	return nil
}

func (c *logsConnector) markAsReleased(traceID pcommon.TraceID, fire func(...event)) error {
	// #get is a potentially blocking operation
	trace, err := c.st.get(traceID)
	if err != nil {
		return fmt.Errorf("couldn't retrieve trace %q from the storage: %w", traceID, err)
	}

	if trace == nil {
		return fmt.Errorf("the trace %q couldn't be found at the storage", traceID)
	}

	// signal that the trace is ready to be released
	c.logger.Debug("trace marked as released", zap.Stringer("traceID", traceID))

	// atomically fire the two events, so that a concurrent shutdown won't leave
	// an orphaned trace in the storage
	fire(event{
		typ:     traceReleased,
		payload: trace,
	}, event{
		typ:     traceRemoved,
		payload: traceID,
	})
	return nil
}

func (c *logsConnector) onTraceReleased(rss []ptrace.ResourceSpans) error {
	trace := ptrace.NewTraces()
	for _, rs := range rss {
		trs := trace.ResourceSpans().AppendEmpty()
		rs.CopyTo(trs)
	}

	c.telemetryBuilder.ProcessorGroupbytraceSpansReleased.Add(context.Background(), int64(trace.SpanCount()))
	c.telemetryBuilder.ProcessorGroupbytraceTracesReleased.Add(context.Background(), 1)

	err := c.exportTracesAsLogs(context.Background(), rss)
	if err != nil {
		c.logger.Error("failed to export traces", zap.Error(err))
	}

	return nil
}

func (c *logsConnector) onTraceRemoved(traceID pcommon.TraceID) error {
	trace, err := c.st.delete(traceID)
	if err != nil {
		return fmt.Errorf("couldn't delete trace %q from the storage: %w", traceID, err)
	}

	if trace == nil {
		return fmt.Errorf("trace %q not found at the storage", traceID)
	}

	return nil
}

func (c *logsConnector) addSpans(traceID pcommon.TraceID, trace ptrace.Traces) error {
	c.logger.Debug("creating trace at the storage", zap.Stringer("traceID", traceID))
	return c.st.createOrAppend(traceID, trace)
}
