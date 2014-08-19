package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
				rollbar.Error(rollbar.CRIT, r.(error))
				w.WriteHeader(500)
				fmt.Fprintln(w, r)
			}
		}()

		rw := negroni.NewResponseWriter(w)
		err := handler(rw, r, vars)

		if err != nil {
			errorLogger.Printf("%v %s (%d): %v\n", time.Now(), r.URL.Path, rw.Status(), errgo.Details(err))
			if rw.Status() == 500 {
				rollbar.Error(rollbar.ERR, err)
			} else if rw.Status()%400 < 100 {
				rollbar.Error(rollbar.WARN, err)
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
