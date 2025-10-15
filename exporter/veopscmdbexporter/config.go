// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package veopscmdbexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/veopscmdbexporter"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configretry"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for the CMDB exporter.
type Config struct {
	// APIKey is the authentication token associated with the Honeycomb account.
	APIKey    configopaque.String `mapstructure:"api_key"`
	APISecret configopaque.String `mapstructure:"api_secret"`

	// API URL to use (defaults to http://localhost:5000)
	APIAddress string `mapstructure:"api_address"`

	// CIMatches is the list of CIs to create
	KubernetesClusterCIType int64 `mapstructure:"kubernetes_cluster_ci_type"`

	confighttp.ClientConfig   `mapstructure:",squash"`
	QueueSettings             exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
	configretry.BackOffConfig `mapstructure:"retry_on_failure"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	if cfg.APIKey == "" || cfg.APISecret == "" {
		return fmt.Errorf("invalid API Key or API Secret")
	}

	if cfg.KubernetesClusterCIType == 0 {
		return fmt.Errorf("no KubernetesClusterCIType supplied")
	}

	if cfg.APIAddress == "" {
		cfg.APIAddress = "http://localhost:5000"
	}

	return nil
}

var _ component.Config = (*Config)(nil)
