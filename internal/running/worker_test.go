package running

import (
	"context"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWorkerFakeWhisper(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to disposable PostgreSQL")
	}
	base, err := database.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	schema := "running_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = base.DB.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer base.DB.Exec("DROP SCHEMA " + schema + " CASCADE")
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	s, err := database.Open(url + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	status := 503
	calls := 0
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/inference" {
			t.Error(r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
		}
		defer r.MultipartForm.RemoveAll()
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
		} else {
			f.Close()
		}
		w.WriteHeader(status)
		if status == 200 {
			w.Write([]byte(`{"text":"hello from local whisper"}`))
		}
	}))
	defer fake.Close()
	c, err := s.AddRunningClip(ctx, database.RunningClip{ID: uuid.NewString(), RecordedAt: time.Now(), DurationMs: 60000, MIME: "audio/mp4", Audio: []byte("fake audio")})
	if err != nil {
		t.Fatal(err)
	}
	worker := Worker{Store: s, URL: fake.URL, Model: "ggml-base.en"}
	if err = worker.Step(ctx); err != nil {
		t.Fatal(err)
	}
	clips, _ := s.RunningClips(ctx, c.NoteID)
	if clips[0].Status != "pending" || clips[0].Attempts != 0 {
		t.Fatal(clips)
	}
	status = 200
	s.DB.Exec("UPDATE running_note_clips SET next_attempt_at=now()")
	if err = worker.Step(ctx); err != nil {
		t.Fatal(err)
	}
	clips, _ = s.RunningClips(ctx, c.NoteID)
	if clips[0].Status != "done" || clips[0].Model != "ggml-base.en" || calls != 2 {
		t.Fatal(clips, calls)
	}
}
