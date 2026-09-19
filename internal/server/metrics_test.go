package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
)

func TestHTTPAttributes(t *testing.T) {
	var got attribute.Set
	r := chi.NewRouter()
	r.Use(captureStatus, func(next http.Handler) http.Handler {
		// Mirrors otelchi: attributes are read after the handler returns.
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			got = attribute.NewSet(httpAttributes(r)...)
		})
	})
	r.Get("/api/entries/{kind}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })

	for _, tc := range []struct {
		method, path, route, wantMethod string
		status                          int
	}{
		{"GET", "/api/entries/note", "/api/entries/{kind}", "GET", http.StatusTeapot},
		{"GET", "/nope/123", "", "GET", http.StatusNotFound},
		{"BREW", "/api/entries/note", "", "_OTHER", http.StatusMethodNotAllowed},
	} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tc.method, tc.path, nil))
		if v, _ := got.Value("http.request.method"); v.AsString() != tc.wantMethod {
			t.Errorf("%s %s: method %q, want %q", tc.method, tc.path, v.AsString(), tc.wantMethod)
		}
		if v, _ := got.Value("http.route"); v.AsString() != tc.route {
			t.Errorf("%s %s: route %q, want %q", tc.method, tc.path, v.AsString(), tc.route)
		}
		if v, _ := got.Value("http.response.status_code"); v.AsInt64() != int64(tc.status) {
			t.Errorf("%s %s: status %d, want %d", tc.method, tc.path, v.AsInt64(), tc.status)
		}
	}
}
