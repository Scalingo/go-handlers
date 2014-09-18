package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/codegangsta/negroni"
)

type LoggingMiddleware struct {
	logger *log.Logger
}

func NewLoggingMiddleware(logger *log.Logger) Middleware {
	m := &LoggingMiddleware{logger}
	return m
}

func (l *LoggingMiddleware) Apply(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		before := time.Now()
		l.logger.Printf("%v %v %v - %v", before, r.Method, r.URL, r.RemoteAddr)
		rw := negroni.NewResponseWriter(w)
		err := next(rw, r, vars)
		after := time.Now()

		status := rw.Status()
		if status == 0 {
			status = 200
		}
		l.logger.Printf("%v %v %v - %v - %d - %0.4f", after, r.Method, r.URL, r.RemoteAddr, status, after.Sub(before).Seconds())
		return err
	}
}
