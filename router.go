package handlers

import (
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/Scalingo/go-utils/logger"
)

type Router struct {
	*mux.Router
	middlewares []Middleware
	// otelOptions are the options used for OpenTelemetry instrumentation
	otelOptions []otelhttp.Option
	// otelOperation is the operation name used for OpenTelemetry instrumentation
	// It gives the name of the spans created for each request to which metrics
	// are attachd
	otelOperation string
	// otelEnabled indicates if OpenTelemetry instrumentation is enabled (true by default)
	otelEnabled bool
}

const (
	otelDefaultOperation = "http"
)

type RouterOption func(r *Router)

func WithOtelOptions(opts ...otelhttp.Option) RouterOption {
	return func(r *Router) {
		r.otelOptions = opts
	}
}

func WithOtelOperation(name string) RouterOption {
	return func(r *Router) {
		r.otelOperation = name
	}
}

func WithoutOtelInstrumentation() RouterOption {
	return func(r *Router) {
		r.otelEnabled = false
	}
}

func NewRouter(logger logrus.FieldLogger, options ...RouterOption) *Router {
	r := &Router{
		Router:        mux.NewRouter(),
		otelOperation: otelDefaultOperation,
		otelEnabled:   true,
		middlewares: []Middleware{
			NewLoggingMiddleware(logger),
			MiddlewareFunc(RequestIDMiddleware),
		},
	}
	for _, opt := range options {
		opt(r)
	}
	if r.otelEnabled {
		r.Router.Use(otelhttp.NewMiddleware(r.otelOperation, r.otelOptions...))
	}
	return r
}

func New(options ...RouterOption) *Router {
	return NewRouter(logger.Default(), options...)
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
