// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package servicegraphprocessor

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/otelcol/otelcoltest"
)

func TestLoadConfig(t *testing.T) {
	// Prepare
	factories, err := otelcoltest.NopFactories()
	require.NoError(t, err)

	factories.Processors[typeStr] = NewFactory()
	factories.Connectors[typeStr] = NewConnectorFactory()

	// Test
	cfg, err := otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "service-graph-config.yaml"), factories)

	// Verify
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t,
		&Config{
			MetricsExporter:         "metrics",
			LatencyHistogramBuckets: []time.Duration{1, 2, 3, 4, 5},
			Dimensions:              []string{"dimension-1", "dimension-2"},
			Store: StoreConfig{
				TTL:      time.Second,
				MaxItems: 10,
			},
			CacheLoop:                 2 * time.Minute,
			StoreExpirationLoop:       10 * time.Second,
			VirtualNodePeerAttributes: []string{"db.name", "rpc.service"},
		},
		cfg.Processors[component.NewID(typeStr)],
	)

	cfg, err = otelcoltest.LoadConfigAndValidate(filepath.Join("testdata", "service-graph-connector-config.yaml"), factories)

	// Verify
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t,
		&Config{
			LatencyHistogramBuckets: []time.Duration{1, 2, 3, 4, 5},
			Dimensions:              []string{"dimension-1", "dimension-2"},
			Store: StoreConfig{
				TTL:      time.Second,
				MaxItems: 10,
			},
			CacheLoop:           time.Minute,
			StoreExpirationLoop: 2 * time.Second,
		},
		cfg.Connectors[component.NewID(typeStr)],
	)

}
