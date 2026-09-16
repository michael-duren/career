package server

import (
	"github.com/michael-duren/career-strategy/internal/config"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthAndOrigin(t *testing.T) {
	h, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	s := &Server{config: config.Config{Username: "admin", PasswordHash: string(h), JWTSecret: "test-secret", PublicOrigin: "https://example.com"}, attempts: map[string]attempt{}}
	handler := s.RegisterRoutes()
	request := func(method, path, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/api/entries/note", "", "", nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := request("GET", "/companies/acme?q=one", "", "", nil); w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login?redirect=%2Fcompanies%2Facme%3Fq%3Done" {
		t.Fatal(w.Code, w.Header().Get("Location"))
	}
	if w := request("GET", "/login", "", "", nil); w.Code != http.StatusNotFound {
		t.Fatal(w.Code)
	}
	if w := request("POST", "/api/auth/login", `{"username":"admin","password":"password"}`, "https://evil.com", nil); w.Code != 403 {
		t.Fatal(w.Code)
	}
	w := request("POST", "/api/auth/login", `{"username":"admin","password":"password"}`, "https://example.com", nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("unsafe cookie")
	}
	if w = request("GET", "/api/auth/verify", "", "", cookies[0]); w.Code != 200 {
		t.Fatal(w.Code)
	}
	cookies[0].Value += "tampered"
	if w = request("GET", "/api/auth/verify", "", "", cookies[0]); w.Code != 401 {
		t.Fatal(w.Code)
	}
	s.attempts["192.0.2.1"] = attempt{10, time.Now().Add(time.Minute)}
	if w = request("POST", "/api/auth/login", `{}`, "https://example.com", nil); w.Code != 429 {
		t.Fatal(w.Code)
	}
}
