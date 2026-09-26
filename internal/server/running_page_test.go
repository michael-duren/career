package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/michael-duren/career-strategy/internal/config"
)

func TestRunningPageRoute(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "audio-thoughts"), 0755); err != nil {
		t.Fatal(err)
	}
	const shell = "<html>Audio thoughts recorder shell</html>"
	if err := os.WriteFile(filepath.Join(root, "audio-thoughts", "index.html"), []byte(shell), 0644); err != nil {
		t.Fatal(err)
	}
	s := &Server{config: config.Config{StaticDir: root, Username: "test", JWTSecret: "test-secret"}}
	handler := s.RegisterRoutes()
	for _, path := range []string{"/audio-thoughts", "/audio-thoughts/"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusSeeOther {
			t.Fatalf("anonymous %s: %d", path, response.Code)
		}
		request.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.String() != shell {
			t.Fatalf("authenticated %s: %d %s", path, response.Code, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("audio thoughts page must retain private cache headers")
		}
	}
	for path, want := range map[string]string{"/running": "/audio-thoughts", "/running/": "/audio-thoughts", "/running?id=run-1": "/audio-thoughts?id=run-1"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != want {
			t.Fatalf("legacy %s: %d %s", path, response.Code, response.Header().Get("Location"))
		}
	}
}
