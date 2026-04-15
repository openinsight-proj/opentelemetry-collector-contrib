// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wraperrorprocessor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func TestSanitizeErrorNil(t *testing.T) {
	assert.Nil(t, sanitizeError(nil))
}

func TestSanitizeErrorPlain(t *testing.T) {
	err := errors.New("plain error")
	assert.Equal(t, err, sanitizeError(err))
}

func TestSanitizeErrorTypedNil(t *testing.T) {
	var err *customError
	assert.Nil(t, sanitizeError(err))
}

func TestSanitizeErrorTypedNilViaInterface(t *testing.T) {
	var typedNil *customError
	var err error = typedNil
	assert.Nil(t, sanitizeError(err))
}

func TestSanitizeErrorNonNilPointer(t *testing.T) {
	err := &customError{msg: "hello"}
	assert.Equal(t, err, sanitizeError(err))
}

func TestCreateTracesProcessor(t *testing.T) {
	tp, err := createTracesProcessor(context.Background(), processortest.NewNopSettings(), createDefaultConfig(), consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, tp)
	assert.False(t, tp.Capabilities().MutatesData)
}

func TestCreateLogsProcessor(t *testing.T) {
	lp, err := createLogsProcessor(context.Background(), processortest.NewNopSettings(), createDefaultConfig(), consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, lp)
	assert.False(t, lp.Capabilities().MutatesData)
}

func TestCreateMetricsProcessor(t *testing.T) {
	mp, err := createMetricsProcessor(context.Background(), processortest.NewNopSettings(), createDefaultConfig(), consumertest.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, mp)
	assert.False(t, mp.Capabilities().MutatesData)
}

func TestTracesSanitizerSanitizesTypedNil(t *testing.T) {
	next := consumertest.NewErr(&customError{})
	// Force the error to be a typed nil by using a variable
	var typedNil *customError
	next = consumertest.NewErr(typedNil)

	s := &tracesSanitizer{next: next}
	err := s.ConsumeTraces(context.Background(), ptrace.NewTraces())
	assert.NoError(t, err)
}

func TestLogsSanitizerSanitizesTypedNil(t *testing.T) {
	var typedNil *customError
	next := consumertest.NewErr(typedNil)

	s := &logsSanitizer{next: next}
	err := s.ConsumeLogs(context.Background(), plog.NewLogs())
	assert.NoError(t, err)
}

func TestMetricsSanitizerSanitizesTypedNil(t *testing.T) {
	var typedNil *customError
	next := consumertest.NewErr(typedNil)

	s := &metricsSanitizer{next: next}
	err := s.ConsumeMetrics(context.Background(), pmetric.NewMetrics())
	assert.NoError(t, err)
}
