package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pkgtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errorutils "github.com/Scalingo/go-utils/errors"
	"github.com/Scalingo/go-utils/logger"
)

func TestErrorMiddlware(t *testing.T) {
	tests := map[string]struct {
		handlerFunc  HandlerFunc
		assertLogs   func(*testing.T, *pkgtest.Hook)
		statusCode   int
		expectedBody string
	}{
		"it should set the status code to 500 if there is none": {
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				log := logger.Get(r.Context()).WithField("field", "value")
				return errorutils.Wrapf(logger.ToCtx(context.Background(), log), errors.New("error"), "wrapping")
			},
			assertLogs: func(t *testing.T, hook *pkgtest.Hook) {
				require.Equal(t, 1, len(hook.Entries))
				assert.Equal(t, "value", hook.Entries[0].Data["field"])
			},
			statusCode:   500,
			expectedBody: "{\"error\":\"wrapping: error\"}\n",
		},
		"it should set the status code to 422 if this is a ValidationErrors": {
			handlerFunc: func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
				err := (&errorutils.ValidationErrors{
					Errors: map[string][]string{
						"test": []string{"biniou"},
					},
				})

				return pkgerrors.Wrap(err, "biniou")
			},
			statusCode:   422,
			expectedBody: "{\"errors\":{\"test\":[\"biniou\"]}}\n",
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
			w.Header().Set("Content-Type", "application/json")
			r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

			err := handler(w, r, map[string]string{})
			require.Error(t, err)

			if test.assertLogs != nil {
				test.assertLogs(t, hook)
			}
			assert.Equal(t, test.statusCode, w.Code)
			assert.Equal(t, test.expectedBody, w.Body.String())
		})
	}
}
