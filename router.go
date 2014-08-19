package handlers

import (
	"fmt"
	"log"
	"os"

	"github.com/gorilla/mux"
)

type Router struct {
	*mux.Router
	middlewares []Middleware
}

func NewRouter(prefix string) *Router {
	r := &Router{}
	r.Router = mux.NewRouter()
	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", prefix), 0)
	r.Use(NewLoggingMiddleware(logger))
	return r
}

func (r *Router) HandleFunc(pattern string, f HandlerFunc) *mux.Route {
	for _, m := range r.middlewares {
		f = m.Apply(f)
	}

	stdHandler := ToHTTPHandler(f)
	return r.Router.Handle(pattern, stdHandler)
}

func (r *Router) Handle(pattern string, h Handler) *mux.Route {
	return r.HandleFunc(pattern, h.ServeHTTP)
}

func (r *Router) Use(m Middleware) {
	// Add at the beginning of the middleware stack
	// The last middleware is called first
	middlewares := r.middlewares
	r.middlewares = append([]Middleware(nil), m)
	r.middlewares = append(r.middlewares, middlewares...)
}
