package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/sirupsen/logrus"
)

type LoggingMiddleware struct {
	logger logrus.FieldLogger
}

func NewLoggingMiddleware(logger logrus.FieldLogger) Middleware {
	m := &LoggingMiddleware{logger}
	return m
}

func (l *LoggingMiddleware) Apply(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		logger := l.logger
		before := time.Now()

		id, ok := r.Context().Value("request_id").(string)
		if ok {
			logger = logger.WithField("request_id", id)
		}

		r = r.WithContext(context.WithValue(r.Context(), "logger", logger))

		fields := logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.String(),
			"host":       r.Host,
			"from":       r.RemoteAddr,
			"protocol":   r.Proto,
			"referer":    r.Referer(),
			"user_agent": r.UserAgent(),
		}
		for k, v := range fields {
			if len(v.(string)) == 0 {
				delete(fields, k)
			}
		}
		logger = logger.WithFields(fields)
		logger.Info("starting request")

		rw := negroni.NewResponseWriter(w)
		err := next(rw, r, vars)
		after := time.Now()

		status := rw.Status()
		if status == 0 {
			status = 200
		}

		logger.WithFields(logrus.Fields{
			"status":   status,
			"duration": after.Sub(before).Seconds(),
			"bytes":    rw.Size(),
		}).Info("request completed")

		return err
	}
}
