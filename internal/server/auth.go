package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func publicPagePath(path string) bool {
	return path == "/login" || path == "/login/" || path == "/offline.html" || path == "/manifest.webmanifest" || path == "/favicon.svg" || path == "/apple-touch-icon.png" || path == "/sw.js" || path == "/.well-known/oauth-protected-resource" || path == "/.well-known/oauth-protected-resource/api/mcp" || strings.HasPrefix(path, "/_astro/") || strings.HasPrefix(path, "/icons/")
}

func (s *Server) pageAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/health" || path == "/healthz" || path == "/readyz" || publicPagePath(path) || s.authenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		target := path
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, "/login?redirect="+url.QueryEscape(target), http.StatusSeeOther)
	})
}

func (s *Server) token() string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	b, _ := json.Marshal(map[string]any{"username": s.config.Username, "exp": time.Now().Add(24 * time.Hour).Unix()})
	payload := header + "." + base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(s.config.JWTSecret))
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *Server) authenticated(r *http.Request) bool {
	c, err := r.Cookie("session")
	if err != nil {
		return false
	}
	p := strings.Split(c.Value, ".")
	if len(p) != 3 {
		return false
	}
	head, err := base64.RawURLEncoding.DecodeString(p[0])
	if err != nil {
		return false
	}
	var h struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(head, &h) != nil || h.Alg != "HS256" {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(p[2])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(s.config.JWTSecret))
	mac.Write([]byte(p[0] + "." + p[1]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return false
	}
	b, err := base64.RawURLEncoding.DecodeString(p[1])
	if err != nil {
		return false
	}
	var claims struct {
		Username string `json:"username"`
		Exp      int64  `json:"exp"`
	}
	return json.Unmarshal(b, &claims) == nil && claims.Username == s.config.Username && claims.Exp > time.Now().Unix()
}
func (s *Server) cookie(w http.ResponseWriter, value string, max int) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: value, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(s.config.PublicOrigin, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: max})
}
func (s *Server) private(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		if !s.authenticated(r) {
			respond(w, 401, map[string]string{"error": "Sign in to access your content."})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) mutation(w http.ResponseWriter, r *http.Request, mediaTypes ...string) bool {
	if r.Header.Get("Origin") != s.config.PublicOrigin {
		respond(w, 403, map[string]string{"error": "Invalid origin"})
		return false
	}
	expected := "application/json"
	if len(mediaTypes) > 0 {
		expected = mediaTypes[0]
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), expected) {
		respond(w, 415, map[string]string{"error": expected + " required"})
		return false
	}
	return true
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	s.mu.Lock()
	now := time.Now()
	for k, a := range s.attempts {
		if now.After(a.until) {
			delete(s.attempts, k)
		}
	}
	a := s.attempts[host]
	if a.count >= 10 || len(s.attempts) >= 10000 {
		s.mu.Unlock()
		respond(w, 429, map[string]string{"error": "Too many login attempts; retry later."})
		return
	}
	if a.count == 0 {
		a.until = now.Add(15 * time.Minute)
	}
	a.count++
	s.attempts[host] = a
	s.mu.Unlock()
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, 4096, &input) {
		return
	}
	if s.config.PasswordHash == "" {
		respond(w, 503, map[string]string{"error": "Set AUTH_PASSWORD_HASH to enable login."})
		return
	}
	validPassword := bcrypt.CompareHashAndPassword([]byte(s.config.PasswordHash), []byte(input.Password)) == nil
	if subtle.ConstantTimeCompare([]byte(input.Username), []byte(s.config.Username)) != 1 || !validPassword {
		respond(w, 401, map[string]string{"error": "Invalid credentials"})
		return
	}
	s.mu.Lock()
	delete(s.attempts, host)
	s.mu.Unlock()
	s.cookie(w, s.token(), 86400)
	respond(w, 200, map[string]any{"authenticated": true, "username": s.config.Username})
}
