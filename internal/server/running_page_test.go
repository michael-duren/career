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
	if err := os.Mkdir(filepath.Join(root, "running"), 0755); err != nil {
		t.Fatal(err)
	}
	const shell = "<html>Running recorder shell</html>"
	if err := os.WriteFile(filepath.Join(root, "running", "index.html"), []byte(shell), 0644); err != nil {
		t.Fatal(err)
	}
	s := &Server{config: config.Config{StaticDir: root, Username: "test", JWTSecret: "test-secret"}}
	handler := s.RegisterRoutes()
	for _, path := range []string{"/running", "/running/"} {
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
			t.Fatal("running page must retain private cache headers")
		}
	}
}
