package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/Soulou/errgo-rollbar"
	"github.com/codegangsta/negroni"
	"github.com/stvp/rollbar"
	"gopkg.in/errgo.v1"
)

var errorLogger = log.New(os.Stderr, "[http-error] ", 0)

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

		rw := negroni.NewResponseWriter(w)
		err := handler(rw, r, vars)

		if err != nil {
			errorLogger.Printf("%v %v %s (%d): %v\n", time.Now(), r.Method, r.URL.Path, rw.Status(), errgo.Details(err))
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
	if !w.Written() {
		w.WriteHeader(w.Status())
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/plain")
	}
	if w.Header().Get("Content-Type") == "application/json" {
		json.NewEncoder(w).Encode(&(map[string]string{"error": err.Error()}))
	} else {
		fmt.Fprintln(w, err)
	}
}
