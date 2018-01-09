package handlers

import (
	"context"
	"net/http"

	uuid "github.com/satori/go.uuid"
	errgo "gopkg.in/errgo.v1"
)

func RequestIDMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		id := r.Header.Get("X-Request-ID")
		if len(id) == 0 {
			id, err = uuid.NewV4().String()
			if err != nil {
				return errgo.Notef(err, "fail to generate a new UUID")
			}
			r.Header.Set("X-Request-ID", id)
		}
		r = r.WithContext(context.WithValue(r.Context(), "request_id", id))
		return next(w, r, vars)
	}
}
