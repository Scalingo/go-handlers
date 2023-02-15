package handlers

import (
	"errors"
	"net/http"
)

func EnforceHTTPSMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		auth := r.Header.Get("X-Forwarded-Proto")
		if auth != "https" {
			w.WriteHeader(401)
			return errors.New("no https header")
		}
		return next(w, r, vars)
	}
}
