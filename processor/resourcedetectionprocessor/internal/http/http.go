// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package http // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor/internal/http"
import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/processor"
	conventions "go.opentelemetry.io/collector/semconv/v1.22.0"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor/internal"
)

const (
	// TypeStr is type of detector.
	TypeStr = "http"
)

type resourceAttribute struct {
	Key   string
	Value string
}

var _ internal.Detector = (*detector)(nil)

type detector struct {
	logger *zap.Logger
	//
	set                   component.TelemetrySettings
	interval              time.Duration
	client                *http.Client
	apiURL                string
	apiKey                configopaque.String
	requestIntervalTicker *time.Ticker
}

// NewDetector returns a detector which can detect resource attributes on Heroku
func NewDetector(set processor.Settings, dcfg internal.DetectorConfig) (internal.Detector, error) {
	cfg := dcfg.(Config)

	if cfg.APIURL == "" {
		return nil, fmt.Errorf("apiUrl could not be empty")
	}

	return &detector{
		apiKey: cfg.APIKey,
		apiURL: cfg.APIURL,
		logger: set.Logger,
		// TODO: interval request
		interval: time.Second * 5,
	}, nil
}

// Detect detects http response metadata and returns a resource with the available ones
func (d detector) Detect(ctx context.Context) (resource pcommon.Resource, schemaURL string, err error) {
	res := pcommon.NewResource()

	detectedResources := d.requestResourceAttributes()

	for _, resAttr := range detectedResources {
		res.Attributes().PutStr(resAttr.Key, resAttr.Value)
	}

	return res, conventions.SchemaURL, nil
}

func (d detector) requestResourceAttributes() []resourceAttribute {
	var resources []resourceAttribute
	resp, err := http.Get(d.apiURL)
	if err != nil {
		d.logger.Warn("Failed to fetch resource", zap.Error(err))
		return resources
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		d.logger.Warn("Failed to fetch resource", zap.Error(err))
		return resources
	}

	err = jsoniter.Unmarshal(body, &resources)
	if err != nil {
		d.logger.Warn("Failed to fetch resource", zap.Error(err))
		return resources
	}

	return resources
}
