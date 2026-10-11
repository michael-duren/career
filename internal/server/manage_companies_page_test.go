package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/michael-duren/career-strategy/internal/config"
)

// Company cards link to /manage/companies?id=<slug> to edit; the editor opens
// that company itself, so the server must serve the editor rather than redirect.
func TestManageCompaniesServesEditorForCompanyID(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "manage", "companies"), 0755); err != nil {
		t.Fatal(err)
	}
	const shell = "<html>Application targets</html>"
	if err := os.WriteFile(filepath.Join(root, "manage", "companies", "index.html"), []byte(shell), 0644); err != nil {
		t.Fatal(err)
	}
	s := &Server{config: config.Config{StaticDir: root, Username: "test", JWTSecret: "test-secret"}}
	handler := s.RegisterRoutes()
	request := httptest.NewRequest(http.MethodGet, "/manage/companies?id=acme", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != shell {
		t.Fatalf("/manage/companies?id=acme: %d %q", response.Code, response.Body.String())
	}
}
