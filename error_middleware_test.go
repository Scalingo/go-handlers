package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pkgtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2errors "github.com/Scalingo/go-utils/errors/v2"
	errorutils "github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
	"github.com/Scalingo/go-utils/security"
)

func TestErrorMiddlware(t *testing.T) {
	tests := map[string]struct {
		contentType        string
		accept             string
		handlerFunc        HandlerFunc
		assertLogs         func(*testing.T, *pkgtest.Hook)
		expectedStatusCode int
		expectedBody       string
	}{
		"it should set the status code to 500 if there is none": {
			contentType: "application/json",
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				log := logger.Get(r.Context()).WithField("field", "value")
				return errorutils.Wrapf(logger.ToCtx(context.Background(), log), errors.New("error"), "wrapping")
			},
			assertLogs: func(t *testing.T, hook *pkgtest.Hook) {
				require.Equal(t, 1, len(hook.Entries))
				assert.Equal(t, "value", hook.Entries[0].Data["field"])
			},
			expectedStatusCode: 500,
			expectedBody:       "{\"error\":\"wrapping: error\"}\n",
		},
		"it should set the status code to 422 if this is a ValidationErrors": {
			contentType: "application/json",
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				err := (&errorutils.ValidationErrors{
					Errors: map[string][]string{
						"test": {"biniou"},
					},
				})

				return pkgerrors.Wrap(err, "biniou")
			},
			expectedStatusCode: 422,
			expectedBody:       "{\"errors\":{\"test\":[\"biniou\"]}}\n",
		},
		"it should set the status code to 422 if this is a ValidationErrors from go-utils/errors/v2": {
			contentType: "application/json",
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				err := (&v2errors.ValidationErrors{
					Errors: map[string][]string{
						"test": {"biniou"},
					},
				})

				return pkgerrors.Wrap(err, "biniou")
			},
			expectedStatusCode: 422,
			expectedBody:       "{\"errors\":{\"test\":[\"biniou\"]}}\n",
		},
		"it should set the status code to 400 if this is a BadRequestError": {
			contentType: "application/json",
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				err := (&BadRequestError{
					Errors: map[string][]string{
						"per_page": {"must be greater than 0"},
					},
				})

				return pkgerrors.Wrap(err, "biniou")
			},
			expectedStatusCode: 400,
			expectedBody:       "{\"error\":\"biniou: * per_page → must be greater than 0\"}\n",
		},
		"it should set the status code to 401 if this is an error due to an invalidity token": {
			contentType: "application/json",
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				err := security.ErrFutureTimestamp
				return pkgerrors.Wrap(err, "biniou")
			},
			expectedStatusCode: 401,
			expectedBody:       "{\"error\":\"biniou: invalid timestamp in the future\"}\n",
		},
		"it should detect any Content-Type ending with +json as JSON": {
			contentType: "application/vnd.docker.plugins.v1.1+json",
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				log := logger.Get(r.Context()).WithField("field", "value")
				return errorutils.Wrapf(logger.ToCtx(context.Background(), log), errors.New("error"), "wrapping")
			},
			assertLogs: func(t *testing.T, hook *pkgtest.Hook) {
				require.Equal(t, 1, len(hook.Entries))
				assert.Equal(t, "value", hook.Entries[0].Data["field"])
			},
			expectedStatusCode: 500,
			expectedBody:       "{\"error\":\"wrapping: error\"}\n",
		},
		"it should add the Content-Type plain text if none is specified": {
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				log := logger.Get(r.Context()).WithField("field", "value")
				return errorutils.Wrapf(logger.ToCtx(context.Background(), log), errors.New("error"), "wrapping")
			},
			assertLogs: func(t *testing.T, hook *pkgtest.Hook) {
				require.Equal(t, 1, len(hook.Entries))
				assert.Equal(t, "value", hook.Entries[0].Data["field"])
			},
			expectedStatusCode: 500,
			expectedBody:       "wrapping: error\n",
		},
		"it should use JSON if the user accepts JSON": {
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				log := logger.Get(r.Context()).WithField("field", "value")
				return errorutils.Wrapf(logger.ToCtx(context.Background(), log), errors.New("error"), "wrapping")
			},
			assertLogs: func(t *testing.T, hook *pkgtest.Hook) {
				require.Equal(t, 1, len(hook.Entries))
				assert.Equal(t, "value", hook.Entries[0].Data["field"])
			},
			accept:             "application/json",
			expectedStatusCode: 500,
			expectedBody:       "{\"error\":\"wrapping: error\"}\n",
		},
		"it should not write anything in the body if it has already been written": {
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				w.WriteHeader(500)
				// My handler writes something in the body but returns an error
				fmt.Fprintln(w, "biniou")
				return errors.New("my error")
			},
			expectedStatusCode: 500,
			expectedBody:       "biniou\n",
		},
		"it should log the error for all 5xx status code": {
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				w.WriteHeader(502)
				log := logger.Get(r.Context()).WithField("field", "value")
				return errorutils.Wrapf(logger.ToCtx(context.Background(), log), errors.New("error"), "wrapping")
			},
			assertLogs: func(t *testing.T, hook *pkgtest.Hook) {
				require.Equal(t, 1, len(hook.Entries))
				assert.Equal(t, "value", hook.Entries[0].Data["field"])
			},
			expectedStatusCode: 502,
		},
	}

	for msg, test := range tests {
		t.Run(msg, func(t *testing.T) {
			handler := ErrorMiddleware(test.handlerFunc)

			log, hook := pkgtest.NewNullLogger()
			log.SetLevel(logrus.DebugLevel)
			defer hook.Reset()

			ctx := logger.ToCtx(context.Background(), log)
			w := httptest.NewRecorder()
			w.Header().Set("Content-Type", test.contentType)
			r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
			r.Header.Set("Accept", test.accept)

			err := handler(w, r, map[string]string{})
			require.Error(t, err)

			if test.assertLogs != nil {
				test.assertLogs(t, hook)
			}
			if test.expectedBody != "" {
				assert.Equal(t, test.expectedBody, w.Body.String())
			}
			assert.Equal(t, test.expectedStatusCode, w.Code)
		})
	}
}
