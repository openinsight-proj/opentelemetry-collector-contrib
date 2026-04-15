// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sanitizeerrorprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/sanitizeerrorprocessor"

import (
	"context"
	"fmt"
	"reflect"
	"unsafe"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// iface mirrors the runtime representation of an interface value (type, data).
type iface struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

// sanitizeError detects typed nil errors and converts them to plain nil.
// This prevents panics in upstream code (e.g., obsreport.endOp) that calls
// err.Error() without checking whether the concrete value is nil.
func sanitizeError(err error, logger *zap.Logger) error {
	if err == nil {
		return nil
	}
	v := reflect.ValueOf(err)
	// error is an interface; if it holds a typed nil, unwrap to the concrete value.
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		if v.IsNil() {
			if logger != nil {
				i := *(*iface)(unsafe.Pointer(&err))
				logger.Info("Detected typed nil error", zap.String("address", fmt.Sprintf("{%p, %p}", i.typ, i.data)))
			}
			return nil
		}
	}
	return err
}

// tracesSanitizer wraps a consumer.Traces and sanitizes returned errors.
type tracesSanitizer struct {
	next   consumer.Traces
	logger *zap.Logger
}

func (s *tracesSanitizer) Capabilities() consumer.Capabilities {
	return consumerCapabilities
}

func (s *tracesSanitizer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return sanitizeError(s.next.ConsumeTraces(ctx, td), s.logger)
}

// logsSanitizer wraps a consumer.Logs and sanitizes returned errors.
type logsSanitizer struct {
	next   consumer.Logs
	logger *zap.Logger
}

func (s *logsSanitizer) Capabilities() consumer.Capabilities {
	return consumerCapabilities
}

func (s *logsSanitizer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return sanitizeError(s.next.ConsumeLogs(ctx, ld), s.logger)
}

// metricsSanitizer wraps a consumer.Metrics and sanitizes returned errors.
type metricsSanitizer struct {
	next   consumer.Metrics
	logger *zap.Logger
}

func (s *metricsSanitizer) Capabilities() consumer.Capabilities {
	return consumerCapabilities
}

func (s *metricsSanitizer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return sanitizeError(s.next.ConsumeMetrics(ctx, md), s.logger)
}
