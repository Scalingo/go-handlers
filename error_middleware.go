package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/Sirupsen/logrus"
	"github.com/Soulou/errgo-rollbar"
	"github.com/codegangsta/negroni"
	"github.com/stvp/rollbar"
)

func ErrorMiddleware(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		defer func() {
			if rec := recover(); rec != nil {
				debug.PrintStack()
				err, ok := rec.(error)
				if !ok {
					err = errors.New(rec.(string))
				}
				rollbar.RequestError(rollbar.CRIT, r, err)
				w.WriteHeader(500)
				fmt.Fprintln(w, err)
			}
		}()

		logger, ok := r.Context().Value("logger").(logrus.FieldLogger)
		if !ok {
			logger = logrus.New()
		}

		rw := negroni.NewResponseWriter(w)
		err := handler(rw, r, vars)

		if err != nil {
			logger.WithField("error", err).Error("request error")
			if rw.Status() == 500 {
				rollbar.RequestErrorWithStack(rollbar.ERR, r, err, errgorollbar.BuildStack(err))
			} else if rw.Status()%400 < 100 {
				rollbar.RequestErrorWithStack(rollbar.WARN, r, err, errgorollbar.BuildStack(err))
			}
			writeError(rw, err)
		}

		return err
	}
}

func writeError(w negroni.ResponseWriter, err error) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain")
	}

	// If the status is 0, In means WriteHeader has not been called
	// and we've to write it, otherwise it has been done in the handler
	// with another response code.
	if w.Status() == 0 {
		w.WriteHeader(500)
	}

	if w.Header().Get("Content-Type") == "application/json" {
		json.NewEncoder(w).Encode(&(map[string]string{"error": err.Error()}))
	} else {
		fmt.Fprintln(w, err)
	}
}
