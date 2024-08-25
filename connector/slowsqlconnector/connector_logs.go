// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package slowsqlconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/slowsqlconnector"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.6.1"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/traceutil"
)

type logsConnector struct {
	config Config

	// Additional dimensions to add to logs.
	dimensions []dimension

	logsConsumer consumer.Logs
	component.StartFunc
	component.ShutdownFunc

	logger *zap.Logger
}

func newLogsConnector(logger *zap.Logger, config component.Config) *logsConnector {
	cfg := config.(*Config)

	return &logsConnector{
		logger:     logger,
		config:     *cfg,
		dimensions: newDimensions(cfg.Dimensions),
	}
}

// Capabilities implements the consumer interface.
func (c *logsConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces implements the consumer.Traces interface.
// It aggregates the trace data to generate logs.
func (c *logsConnector) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	ld := plog.NewLogs()
	for i := 0; i < traces.ResourceSpans().Len(); i++ {
		rspans := traces.ResourceSpans().At(i)
		resourceAttr := rspans.Resource().Attributes()
		serviceAttr, ok := resourceAttr.Get(conventions.AttributeServiceName)
		if !ok {
			continue
		}

		serviceName := serviceAttr.Str()
		ilsSlice := rspans.ScopeSpans()
		for j := 0; j < ilsSlice.Len(); j++ {
			sl := c.newScopeLogs(ld)
			ils := ilsSlice.At(j)
			ils.Scope().CopyTo(sl.Scope())
			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				switch span.Kind() {
				case ptrace.SpanKindClient:
					// through db.Statement exists represents db client
					if _, dbSystem := findAttributeValue(dbSystemKey, span.Attributes()); dbSystem {
						for _, db := range c.config.DBSystem {
							if db == getValue(span.Attributes(), dbSystemKey) && spanDuration(span) >= c.config.Threshold {
								c.attrToLogRecord(sl, serviceName, span, resourceAttr)
							}
						}
					}
				}
			}
		}
	}
	return c.exportLogs(ctx, ld)
}

// spanDuration returns the duration of the given span in seconds
func spanDuration(span ptrace.Span) float64 {
	return float64(span.EndTimestamp()-span.StartTimestamp()) / float64(time.Millisecond.Nanoseconds())
}

func (c *logsConnector) exportLogs(ctx context.Context, ld plog.Logs) error {
	if err := c.logsConsumer.ConsumeLogs(ctx, ld); err != nil {
		c.logger.Error("failed to convert slow sql to logs", zap.Error(err))
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

	logRecord.SetTimestamp(span.StartTimestamp())
	logRecord.SetSeverityNumber(plog.SeverityNumberError)
	logRecord.SetSeverityText("SLOW")
	logRecord.SetSpanID(span.SpanID())
	logRecord.SetTraceID(span.TraceID())
	spanAttrs := span.Attributes()

	// Copy span attributes to the log record.
	spanAttrs.CopyTo(logRecord.Attributes())

	// Add common attributes to the log record.
	logRecord.Attributes().PutStr(spanKindKey, traceutil.SpanKindStr(span.Kind()))
	logRecord.Attributes().PutStr(statusCodeKey, traceutil.StatusCodeStr(span.Status().Code()))
	logRecord.Attributes().PutStr(serviceNameKey, serviceName)
	logRecord.Attributes().PutStr(dbStatementKey, serviceName)
	logRecord.Attributes().PutDouble(statementExecDuration, spanDuration(span)) // seconds

	// Add configured dimension attributes to the log record.
	for _, d := range c.dimensions {
		if v, ok := getDimensionValue(d, spanAttrs, resourceAttrs); ok {
			logRecord.Attributes().PutStr(d.name, v.Str())
		}
	}

	return logRecord
}

// getValue returns the value of the attribute with the given key.
func getValue(attr pcommon.Map, key string) string {
	if attrVal, ok := attr.Get(key); ok {
		return attrVal.Str()
	}
	return ""
}
