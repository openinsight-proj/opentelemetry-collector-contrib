package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/processor/processortest"
	conventions "go.opentelemetry.io/collector/semconv/v1.22.0"
)

type mockMetadata struct {
	mock.Mock
}

func TestDetect(t *testing.T) {
	handler := http.NotFound
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	defer ts.Close()
	handler = func(w http.ResponseWriter, r *http.Request) {
		outPut := `[{"key":"attributes_1","value":"foo"},{"key":"attributes_2","value":"bar"}]`
		_, _ = w.Write([]byte(outPut))
	}
	defaultCfg := CreateDefaultConfig()
	defaultCfg.APIURL = ts.URL

	d, err := NewDetector(processortest.NewNopSettings(), defaultCfg)
	require.NoError(t, err)
	res, schemaURL, err := d.Detect(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, res.Attributes().Len())
	require.NotNil(t, res)
	assert.Equal(t, conventions.SchemaURL, schemaURL)
}
