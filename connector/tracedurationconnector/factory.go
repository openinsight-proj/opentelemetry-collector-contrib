// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tracedurationconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/tracedurationconnector"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/connector"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"

	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/tracedurationconnector/internal/metadata"
)

const (
	defaultWaitDuration   = time.Second
	defaultNumTraces      = 1_000_000
	defaultNumWorkers     = 1
	defaultDiscardOrphans = false
	defaultStoreOnDisk    = false
)

var (
	errDiskStorageNotSupported    = fmt.Errorf("option 'disk storage' not supported in this release")
	errDiscardOrphansNotSupported = fmt.Errorf("option 'discard orphans' not supported in this release")
)

// NewFactory returns a new factory for the Filter processor.
func NewFactory() connector.Factory {

	return connector.NewFactory(
		metadata.Type,
		createDefaultConfig,
		connector.WithTracesToLogs(createTracesToLogsConnector, metadata.TracesToLogsStability),
		//processor.WithTraces(createTracesProcessor, metadata.TracesStability)
	)
}

// createDefaultConfig creates the default configuration for the processor.
func createDefaultConfig() component.Config {
	return &Config{
		NumTraces:    defaultNumTraces,
		NumWorkers:   defaultNumWorkers,
		WaitDuration: defaultWaitDuration,

		// not supported for now
		DiscardOrphans: defaultDiscardOrphans,
		StoreOnDisk:    defaultStoreOnDisk,
	}
}

// createTracesProcessor creates a trace processor based on this config.
//func createTracesProcessor(
//	_ context.Context,
//	params processor.Settings,
//	cfg component.Config,
//	nextConsumer consumer.Traces) (processor.Traces, error) {
//
//	oCfg := cfg.(*Config)
//
//	var st storage
//	if oCfg.StoreOnDisk {
//		return nil, errDiskStorageNotSupported
//	}
//	if oCfg.DiscardOrphans {
//		return nil, errDiscardOrphansNotSupported
//	}
//
//	processor := newGroupByTraceProcessor(params, nextConsumer, *oCfg)
//	// the only supported storage for now
//	st = newMemoryStorage(processor.telemetryBuilder)
//	processor.st = st
//	return processor, nil
//}

func createTracesToLogsConnector(_ context.Context, params connector.Settings, cfg component.Config, nextConsumer consumer.Logs) (connector.Traces, error) {
	oCfg := cfg.(*Config)

	var st storage
	if oCfg.StoreOnDisk {
		return nil, errDiskStorageNotSupported
	}
	if oCfg.DiscardOrphans {
		return nil, errDiscardOrphansNotSupported
	}

	lc := newLogsConnector(params, cfg)
	// the only supported storage for now
	st = newMemoryStorage(lc.telemetryBuilder)
	lc.st = st

	lc.logsConsumer = nextConsumer
	return lc, nil
}
