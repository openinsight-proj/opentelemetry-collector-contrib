// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package veopscmdbexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/veopscmdbexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/veopscmdbexporter/internal/metadata"
)

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		APIAddress: "http://localhost:5000",
	}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Logs, error) {
	cf := cfg.(*Config)

	logsExp, err := newVeopsCMDBExporter(set, cf)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		logsExp.exportResources,
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
		exporterhelper.WithRetry(cf.BackOffConfig),
		exporterhelper.WithQueue(cf.QueueSettings),
		exporterhelper.WithStart(logsExp.start),
	)
}
