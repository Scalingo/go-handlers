package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/juju/errgo"
	"github.com/stvp/rollbar"
)

var errorLogger = log.New(os.Stderr, "[http-error] ", 0)

func ErrorMiddleware(handler HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		defer func() {
			if r := recover(); r != nil {
				debug.PrintStack()
				err, ok := r.(error)
				if !ok {
					err = errors.New(r.(string))
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
				rollbar.RequestError(rollbar.ERR, r, err)
			} else if rw.Status()%400 < 100 {
				rollbar.RequestError(rollbar.WARN, r, err)
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
	fmt.Fprintln(w, err)
}
