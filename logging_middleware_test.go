package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/Sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware_Apply(t *testing.T) {
	examples := []struct {
		Name           string
		Handler        func(t *testing.T) HandlerFunc
		Path           string
		Method         string
		Host           string
		Headers        map[string]string
		Context        func(context.Context) context.Context
		ExpectedFields []string
	}{
		{
			Name:           "HTTP GET / on example.dev without any additional info",
			Path:           "/",
			Method:         "GET",
			Host:           "example.dev",
			ExpectedFields: []string{"path", "host", "method"},
		}, {
			Name:           "with user agent",
			Path:           "/",
			Method:         "GET",
			Host:           "example.dev",
			Headers:        map[string]string{"User-Agent": "MyUserAgent1.0"},
			ExpectedFields: []string{"path", "host", "method", "user_agent"},
		}, {
			Name:           "with referer",
			Path:           "/",
			Method:         "GET",
			Host:           "example.dev",
			Headers:        map[string]string{"Referer": "http://google.com"},
			ExpectedFields: []string{"path", "host", "method", "referer"},
		}, {
			Name:           "with request_id in context",
			Path:           "/",
			Method:         "GET",
			Host:           "example.dev",
			ExpectedFields: []string{"path", "host", "method", "request_id"},
			Context: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "request_id", "0")
			},
		}, {
			Name:           "with request_id in context",
			Path:           "/",
			Method:         "GET",
			Host:           "example.dev",
			ExpectedFields: []string{"path", "host", "method"},
			Handler: func(t *testing.T) HandlerFunc {
				return HandlerFunc(func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
					logger, ok := r.Context().Value("logger").(logrus.FieldLogger)
					assert.True(t, ok)
					assert.NotNil(t, logger)
					return nil
				})
			},
		},
	}

	handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
		return nil
	})

	logger, hook := test.NewNullLogger()
	middleware := NewLoggingMiddleware(logger)

	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {
			defer hook.Reset()

			reqHandler := handler
			if example.Handler != nil {
				reqHandler = example.Handler(t)
			}
			reqHandler = middleware.Apply(reqHandler)

			w := httptest.NewRecorder()
			r, err := http.NewRequest(example.Method, example.Path, nil)
			assert.NoError(t, err)

			if example.Context != nil {
				r = r.WithContext(example.Context(r.Context()))
			}

			r.Host = example.Host
			if example.Headers != nil {
				for k, v := range example.Headers {
					r.Header.Add(k, v)
				}
			}

			err = reqHandler(w, r, map[string]string{})
			assert.NoError(t, err)

			assert.Equal(t, 2, len(hook.Entries))
			assert.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
			assert.Equal(t, logrus.InfoLevel, hook.Entries[1].Level)

			log1Keys := []string{}
			for k, _ := range hook.Entries[0].Data {
				log1Keys = append(log1Keys, k)
			}
			log2Keys := []string{}
			for k, _ := range hook.Entries[1].Data {
				log2Keys = append(log2Keys, k)
			}

			assert.Subset(t, log1Keys, example.ExpectedFields)
			assert.Subset(t, log1Keys, []string{"protocol"})

			assert.Subset(t, log2Keys, example.ExpectedFields)
			assert.Subset(t, log2Keys, []string{"protocol", "status", "duration", "bytes"})
		})
	}
}
