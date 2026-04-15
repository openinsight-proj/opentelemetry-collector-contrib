// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wraperrorprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/wraperrorprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/wraperrorprocessor/internal/metadata"
)

var consumerCapabilities = consumer.Capabilities{MutatesData: false}

// NewFactory returns a new factory for the WrapError processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, metadata.TracesStability),
		processor.WithLogs(createLogsProcessor, metadata.LogsStability),
		processor.WithMetrics(createMetricsProcessor, metadata.MetricsStability))
}

// createDefaultConfig creates the default configuration for the processor.
func createDefaultConfig() component.Config {
	return &Config{}
}

// createTracesProcessor creates a trace processor based on this config.
func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		&tracesSanitizer{next: nextConsumer},
		func(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
			return td, nil
		},
		processorhelper.WithCapabilities(consumerCapabilities))
}

// createLogsProcessor creates a logs processor based on this config.
func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		&logsSanitizer{next: nextConsumer},
		func(_ context.Context, ld plog.Logs) (plog.Logs, error) {
			return ld, nil
		},
		processorhelper.WithCapabilities(consumerCapabilities))
}

// createMetricsProcessor creates a metrics processor based on this config.
func createMetricsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (processor.Metrics, error) {
	return processorhelper.NewMetricsProcessor(
		ctx,
		set,
		cfg,
		&metricsSanitizer{next: nextConsumer},
		func(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
			return md, nil
		},
		processorhelper.WithCapabilities(consumerCapabilities))
}
