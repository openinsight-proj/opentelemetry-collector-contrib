// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package http // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor/internal/http"
import (
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
)

type Config struct {
	// APIKey is the authentication token
	APIKey configopaque.String `mapstructure:"api_key"`

	// API URL to use
	APIURL string `mapstructure:"api_url"`

	confighttp.ClientConfig `mapstructure:",squash"`
}

func CreateDefaultConfig() Config {
	return Config{
		APIURL: "http://localhost:8080",
	}
}
