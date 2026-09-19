package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"golang.org/x/crypto/bcrypt"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunningAPI(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	base, err := database.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	schema := "running_api_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = base.DB.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer base.DB.Exec("DROP SCHEMA " + schema + " CASCADE")
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	db, err := database.Open(url + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	s := Server{db: db, config: config.Config{Username: "admin", PasswordHash: string(hash), JWTSecret: "test-secret", PublicOrigin: "https://example.com"}, attempts: map[string]attempt{}}
	handler := s.RegisterRoutes()
	login := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"password"}`))
	login.Header.Set("Origin", "https://example.com")
	login.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, login)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	cookie := w.Result().Cookies()[0]
	id := uuid.NewString()
	upload := func(auth bool, origin string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		m := multipart.NewWriter(&body)
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="audio"; filename="memo.webm"`)
		h.Set("Content-Type", "audio/webm")
		f, _ := m.CreatePart(h)
		f.Write([]byte("testaudio"))
		m.WriteField("clientId", id)
		m.WriteField("durationMs", "1000")
		m.WriteField("recordedAt", time.Now().Format(time.RFC3339))
		m.Close()
		r := httptest.NewRequest("POST", "/api/running/clips", &body)
		r.Header.Set("Content-Type", m.FormDataContentType())
		r.Header.Set("Origin", origin)
		if auth {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w = upload(false, "https://example.com"); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w = upload(true, "https://evil.com"); w.Code != 403 {
		t.Fatal(w.Code)
	}
	w = upload(true, "https://example.com")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var clip database.RunningClip
	json.Unmarshal(w.Body.Bytes(), &clip)
	w = upload(true, "https://example.com")
	var duplicate database.RunningClip
	json.Unmarshal(w.Body.Bytes(), &duplicate)
	if duplicate.ID != clip.ID || duplicate.NoteID != clip.NoteID {
		t.Fatal(w.Body.String())
	}
	r := httptest.NewRequest("GET", "/api/running/clips/"+clip.NoteID+"/"+clip.ID+"/audio", nil)
	r.AddCookie(cookie)
	r.Header.Set("Range", "bytes=0-3")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 206 || w.Body.String() != "test" || w.Header().Get("Content-Type") != "audio/webm" {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, explicit := range []bool{false, true} {
		path := "/api/agent-context"
		if explicit {
			path += "?journal=running"
		}
		r = httptest.NewRequest("GET", path, nil)
		r.AddCookie(cookie)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		contains := strings.Contains(w.Body.String(), clip.NoteID)
		if contains != explicit {
			t.Fatal("running context privacy", w.Body.String())
		}
	}
}
