package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/michael-duren/career-strategy/internal/config"
)

func TestDevUIPageOnlyOutsideProduction(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dev", "ui"), 0755); err != nil {
		t.Fatal(err)
	}
	const shell = "<html>Drawing primitives</html>"
	if err := os.WriteFile(filepath.Join(root, "dev", "ui", "index.html"), []byte(shell), 0644); err != nil {
		t.Fatal(err)
	}
	for _, production := range []bool{false, true} {
		s := &Server{config: config.Config{StaticDir: root, Username: "test", JWTSecret: "test-secret", Production: production}}
		handler := s.RegisterRoutes()
		for _, path := range []string{"/dev/ui", "/dev/ui/", "/dev/ui/index.html"} {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if production {
				if response.Code != http.StatusNotFound {
					t.Fatalf("production %s: %d, want 404", path, response.Code)
				}
				continue
			}
			if path == "/dev/ui/index.html" {
				continue // served like any other built file outside production
			}
			if response.Code != http.StatusOK || response.Body.String() != shell {
				t.Fatalf("development %s: %d %s", path, response.Code, response.Body.String())
			}
			if response.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("%s must be served with private cache headers", path)
			}
		}
		if !production {
			request := httptest.NewRequest(http.MethodGet, "/dev/ui", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusSeeOther {
				t.Fatalf("anonymous /dev/ui: %d, want a login redirect", response.Code)
			}
		}
	}
}
