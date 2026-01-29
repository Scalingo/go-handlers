package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewRouter_OtelMiddleware(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	metricReader := sdkmetric.NewManualReader()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))

	prevProvider := otel.GetTracerProvider()
	prevMeterProvider := otel.GetMeterProvider()
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	t.Cleanup(func() {
		otel.SetTracerProvider(prevProvider)
		_ = tp.Shutdown(t.Context())
		otel.SetMeterProvider(prevMeterProvider)
		_ = mp.Shutdown(t.Context())
	})

	router := NewRouter(logrus.New())
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	spans := spanRecorder.Ended()
	require.NotEmpty(t, spans)

	var metrics metricdata.ResourceMetrics
	require.NoError(t, metricReader.Collect(t.Context(), &metrics))
	require.NotEmpty(t, metrics.ScopeMetrics)
	require.NotEmpty(t, metrics.ScopeMetrics[0].Metrics)
}
