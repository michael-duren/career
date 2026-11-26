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
				if response.Code != http.StatusOK || response.Body.String() != shell {
					t.Fatalf("development %s: %d", path, response.Code)
				}
				continue // a plain built file outside production, with the asset cache headers
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

// The shared design CSS is public like /_astro/, so pages served before login (/login) and
// the Go-rendered pages can link it.
func TestDesignCSSIsPublic(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "ui"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tokens.css", "primitives.css"} {
		if err := os.WriteFile(filepath.Join(root, "ui", name), []byte(".ui{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	s := &Server{config: config.Config{StaticDir: root, Username: "test", JWTSecret: "test-secret", Production: true}}
	handler := s.RegisterRoutes()
	for _, path := range []string{"/ui/tokens.css", "/ui/primitives.css"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || response.Body.String() != ".ui{}" {
			t.Fatalf("anonymous %s: %d %s", path, response.Code, response.Body.String())
		}
		if got := response.Header().Get("Content-Type"); got != "text/css; charset=utf-8" {
			t.Fatalf("%s content type %q", path, got)
		}
	}
	// Only those two files are public: neighbours still need a login.
	for _, path := range []string{"/ui/other.css", "/dev/ui/index.html"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusSeeOther {
			t.Fatalf("anonymous %s: %d, want a login redirect", path, response.Code)
		}
	}
}

func TestDevAssetsHiddenInProduction(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dev", "ui"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dev", "ui", "x.js"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	for production, want := range map[bool]int{true: http.StatusNotFound, false: http.StatusOK} {
		s := &Server{config: config.Config{StaticDir: root, Username: "test", JWTSecret: "test-secret", Production: production}}
		request := httptest.NewRequest(http.MethodGet, "/dev/ui/x.js", nil)
		request.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		response := httptest.NewRecorder()
		s.RegisterRoutes().ServeHTTP(response, request)
		if response.Code != want {
			t.Fatalf("production=%v /dev/ui/x.js: %d, want %d", production, response.Code, want)
		}
	}
}
