package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

type statusKey struct{}

// captureStatus exposes the response status to httpAttributes. It must run
// outside the otelchi middleware so it sees every write.
func captureStatus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r.WithContext(context.WithValue(r.Context(), statusKey{}, ww)))
	})
}

var knownMethods = map[string]attribute.KeyValue{
	http.MethodGet:     semconv.HTTPRequestMethodGet,
	http.MethodHead:    semconv.HTTPRequestMethodHead,
	http.MethodPost:    semconv.HTTPRequestMethodPost,
	http.MethodPut:     semconv.HTTPRequestMethodPut,
	http.MethodPatch:   semconv.HTTPRequestMethodPatch,
	http.MethodDelete:  semconv.HTTPRequestMethodDelete,
	http.MethodOptions: semconv.HTTPRequestMethodOptions,
	http.MethodConnect: semconv.HTTPRequestMethodConnect,
	http.MethodTrace:   semconv.HTTPRequestMethodTrace,
}

// httpAttributes keeps metric cardinality bounded: the method is normalized,
// http.route is the chi pattern (absent until routing matched), and the
// status code is only known once the handler has written.
func httpAttributes(r *http.Request) []attribute.KeyValue {
	method, ok := knownMethods[r.Method]
	if !ok {
		method = semconv.HTTPRequestMethodOther
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	attrs := []attribute.KeyValue{method, semconv.URLScheme(scheme)}
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if route := rctx.RoutePattern(); route != "" {
			attrs = append(attrs, semconv.HTTPRoute(route))
		}
	}
	if ww, ok := r.Context().Value(statusKey{}).(middleware.WrapResponseWriter); ok && ww.Status() != 0 {
		attrs = append(attrs, semconv.HTTPResponseStatusCode(ww.Status()))
	}
	return attrs
}
