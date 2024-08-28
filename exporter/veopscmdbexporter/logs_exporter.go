// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package veopscmdbexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/veopscmdbexporter"

import (
	"context"
	"fmt"
	"net/http"
	"runtime"

	"github.com/tidwall/gjson"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
)

type marker struct {
	logBoolExpr expr.BoolExpr[ottllog.TransformContext]
}

const (
	AddciUrl = "/api/v0.1/ci"
)

type cmdbLogsExporter struct {
	set                component.TelemetrySettings
	client             *http.Client
	httpClientSettings confighttp.ClientConfig
	apiAddress         string
	apiKey             configopaque.String
	apiSecret          configopaque.String
	userAgentHeader    string
	clusterType        int64
}

func newVeopsCMDBExporter(set exporter.Settings, config *Config) (*cmdbLogsExporter, error) {
	if config == nil {
		return nil, fmt.Errorf("unable to create VeopsCMDBExporter without config")
	}

	telemetrySettings := set.TelemetrySettings

	logsExp := &cmdbLogsExporter{
		set:                telemetrySettings,
		httpClientSettings: config.ClientConfig,
		apiAddress:         config.APIAddress,
		apiKey:             config.APIKey,
		apiSecret:          config.APISecret,
		clusterType:        config.KubernetesClusterCIType,
		userAgentHeader:    fmt.Sprintf("%s/%s (%s/%s)", set.BuildInfo.Description, set.BuildInfo.Version, runtime.GOOS, runtime.GOARCH),
	}
	return logsExp, nil
}

func (e *cmdbLogsExporter) exportResources(ctx context.Context, ld plog.Logs) error {

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rlogs := ld.ResourceLogs().At(i)
		for j := 0; j < rlogs.ScopeLogs().Len(); j++ {
			slogs := rlogs.ScopeLogs().At(j)
			logs := slogs.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				logRecord := logs.At(k)

				kind := gjson.Get(logRecord.Body().AsString(), "kind").String()
				// 1. heart beat with cluster_uid
				if kind == "Namespace" && gjson.Get(logRecord.Body().AsString(), "metadata.name").String() == "kube-system" {
					e.set.Logger.Info("kind == namespace && namespace == kube-system")

					k8s_uid := gjson.Get(logRecord.Body().AsString(), "metadata.uid").String()
					params := map[string]interface{}{
						"ci_type":             e.clusterType,
						"no_attribute_policy": "ignore",
						"exist_policy":        "reject",
						"uuid":                k8s_uid,
						"cluster_name":        "changeme-" + k8s_uid,
					}
					if err := e.addCI(e.set.Logger, params); err != nil {
						return err
					}
				}
				e.set.Logger.Debug(fmt.Sprintf("logs body: %s", logRecord.Body().AsString()))
			}
		}
	}
	return nil
}

func (e *cmdbLogsExporter) start(ctx context.Context, host component.Host) (err error) {
	client, err := e.httpClientSettings.ToClient(ctx, host, e.set)

	if err != nil {
		return err
	}

	e.client = client

	return nil
}

func (e *cmdbLogsExporter) addCI(logger *zap.Logger, payload map[string]interface{}) error {
	url := fmt.Sprintf("%s%s", e.apiAddress, AddciUrl)
	payload = buildAPIKey(urlPath(url), string(e.apiKey), string(e.apiSecret), payload)

	logger.Debug("add ci request with url", zap.String("url", url), zap.Any("payload", payload))

	// Perform POST request
	return makeRequest(http.MethodPost, url, payload)
}
