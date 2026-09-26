package database

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"strings"
	"sync"
	"testing"
	"time"
)

func clipFixture() RunningClip {
	return RunningClip{ID: uuid.NewString(), RecordedAt: time.Now().UTC(), DurationMs: 60000, MIME: "audio/webm", Audio: []byte("audio")}
}
func TestRunningValidation(t *testing.T) {
	e := Entity{"id": "run", "title": "Run", "runDate": "2026-09-19", "startedAt": "2026-09-19T10:00:00Z", "body": "", "tags": []any{}, "distanceKm": 500.0, "durationMin": 1440.0}
	if err := Validate("run", e); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"distanceKm", "durationMin"} {
		copy := Entity{}
		for k, v := range e {
			copy[k] = v
		}
		copy[key] = 2000.0
		if Validate("run", copy) == nil {
			t.Fatal("accepted out of bounds", key)
		}
	}
}
func TestRunningIdempotencyGroupingEditsAndClaims(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	input := clipFixture()
	var wg sync.WaitGroup
	results := make(chan RunningClip, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := s.AddRunningClip(ctx, input)
			if err != nil {
				t.Error(err)
			} else {
				results <- c
			}
		}()
	}
	wg.Wait()
	close(results)
	var note string
	for c := range results {
		if note != "" && note != c.NoteID {
			t.Fatal("duplicate run")
		}
		note = c.NoteID
	}
	second := clipFixture()
	second.RecordedAt = input.RecordedAt.Add(5 * time.Minute)
	c2, err := s.AddRunningClip(ctx, second)
	if err != nil || c2.NoteID != note {
		t.Fatal(c2, err)
	}
	clips, err := s.RunningClips(ctx, note)
	if err != nil || len(clips) != 2 {
		t.Fatal(clips, err)
	}
	claim, err := s.ClaimRunningClip(ctx)
	if err != nil {
		t.Fatal(err)
	}
	claim2, err := s.ClaimRunningClip(ctx)
	if err != nil || claim2.ID == claim.ID {
		t.Fatal(claim2, err)
	}
	if _, err = s.ClaimRunningClip(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	before, err := s.Detail(ctx, "run", note)
	if err != nil {
		t.Fatal(err)
	}
	before.Entry["body"] = "My edited text"
	saved, err := s.Save(ctx, "run", before.Entry, &before.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinishRunningClip(ctx, claim, "first words", "base"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, "run", saved.Entry, &saved.Revision); !errors.Is(err, ErrConflict) {
		t.Fatal("stale revision accepted", err)
	}
	if err = s.FinishRunningClip(ctx, claim2, "second words", "base"); err != nil {
		t.Fatal(err)
	}
	if err = s.Retranscribe(ctx, note, claim.ID); err != nil {
		t.Fatal(err)
	}
	again, err := s.ClaimRunningClip(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinishRunningClip(ctx, again, "replacement", "small"); err != nil {
		t.Fatal(err)
	}
	final, _ := s.Detail(ctx, "run", note)
	body := final.Entry["body"].(string)
	if !strings.Contains(body, "My edited text") || !strings.Contains(body, "first words") || !strings.Contains(body, "second words") || strings.Contains(body, "replacement") {
		t.Fatal(body)
	}
	if body != "My edited text\n\nfirst words\n\nsecond words" {
		t.Fatalf("transcripts must append without timestamp headings: %q", body)
	}
	if final.Entry["title"] != "My edited text first words" {
		t.Fatal("placeholder title not replaced by opening words", final.Entry["title"])
	}
	listed, _ := s.List(ctx, "run", Filter{})
	if len(listed.Entries) != 1 || listed.Entries[0].Entry["excerpt"] != "My edited text first words second words" || listed.Entries[0].Entry["body"] != nil {
		t.Fatal("list must carry a whitespace-collapsed excerpt instead of the body", listed.Entries)
	}
	notes, _ := s.List(ctx, "note", Filter{})
	personal, _ := s.List(ctx, "personal", Filter{})
	if len(notes.Entries) != 0 || len(personal.Entries) != 0 {
		t.Fatal("running note leaked")
	}
	var exported bytes.Buffer
	if err = s.Export(ctx, &exported); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(exported.String(), `"runningNotes"`) || strings.Contains(exported.String(), `"audio":`) {
		t.Fatal("export must retain running text without audio")
	}
	found, err := s.Search(ctx, "first words", 50, "run")
	if err != nil || len(found) != 1 {
		t.Fatal(found, err)
	}
}
func TestRunningRetryAndStuckReset(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.AddRunningClip(ctx, clipFixture())
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.ClaimRunningClip(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RetryRunningClip(ctx, claimed, "unreachable", true); err != nil {
		t.Fatal(err)
	}
	clips, _ := s.RunningClips(ctx, c.NoteID)
	if clips[0].Attempts != 0 || clips[0].Status != "pending" {
		t.Fatal(clips)
	}
	for i := 1; i <= 3; i++ {
		s.DB.Exec("UPDATE running_note_clips SET next_attempt_at=now()")
		claimed, err = s.ClaimRunningClip(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.RetryRunningClip(ctx, claimed, "bad audio", false); err != nil {
			t.Fatal(err)
		}
	}
	clips, _ = s.RunningClips(ctx, c.NoteID)
	if clips[0].Status != "failed" || clips[0].Attempts != 3 {
		t.Fatal(clips)
	}
	if err = s.Retranscribe(ctx, c.NoteID, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ClaimRunningClip(ctx); err != nil {
		t.Fatal(err)
	}
	s.DB.Exec("UPDATE running_note_clips SET claimed_at=now()-interval '11 minutes'")
	if err = s.ResetRunningClaims(ctx); err != nil {
		t.Fatal(err)
	}
	clips, _ = s.RunningClips(ctx, c.NoteID)
	if clips[0].Status != "pending" {
		t.Fatal(clips)
	}
}

func TestThoughtTitle(t *testing.T) {
	for body, want := range map[string]string{
		"":                                   "",
		"  short idea  ":                     "short idea",
		"one two three four five six":        "one two three four five six",
		"one two three four five six, seven": "one two three four five six...",
		"\n\nso I was thinking about\nthe queue design today": "so I was thinking about the...",
		strings.Repeat("x", 80):                               strings.Repeat("x", 60) + "...",
	} {
		if got := ThoughtTitle(body); got != want {
			t.Errorf("ThoughtTitle(%q) = %q, want %q", body, got, want)
		}
	}
}
func TestTranscriptKeepsUserTitle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.AddRunningClip(ctx, clipFixture())
	if err != nil {
		t.Fatal(err)
	}
	detail, _ := s.Detail(ctx, "run", c.NoteID)
	if detail.Entry["title"] != ThoughtPlaceholder {
		t.Fatal("new thoughts start with the placeholder title", detail.Entry["title"])
	}
	detail.Entry["title"] = "Queue design"
	if _, err = s.Save(ctx, "run", detail.Entry, &detail.Revision); err != nil {
		t.Fatal(err)
	}
	claimed, err := s.ClaimRunningClip(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinishRunningClip(ctx, claimed, "a long transcript about many things", "base"); err != nil {
		t.Fatal(err)
	}
	final, _ := s.Detail(ctx, "run", c.NoteID)
	if final.Entry["title"] != "Queue design" {
		t.Fatal("user title overwritten", final.Entry["title"])
	}
}
func TestDeleteThoughtWhileTranscribing(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.AddRunningClip(ctx, clipFixture())
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.ClaimRunningClip(ctx)
	if err != nil || claimed.ID != c.ID {
		t.Fatal(claimed, err)
	}
	detail, err := s.Detail(ctx, "run", c.NoteID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Delete(ctx, "run", c.NoteID, &detail.Revision); err != nil {
		t.Fatal(err)
	}
	// The worker finishing or retrying afterwards must not recreate the thought.
	if err = s.FinishRunningClip(ctx, claimed, "late transcript", "base"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("finish after delete", err)
	}
	if err = s.RetryRunningClip(ctx, claimed, "late failure", false); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Detail(ctx, "run", c.NoteID); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted thought recreated", err)
	}
	if clips, err := s.RunningClips(ctx, c.NoteID); err != nil || len(clips) != 0 {
		t.Fatal("deleted thought kept clips", clips, err)
	}
	if _, err = s.ClaimRunningClip(ctx); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("deleted clip still claimable", err)
	}
}
func TestAudioThoughtMigrationCleansLegacyNotes(t *testing.T) {
	s := testStore(t)
	_, err := s.DB.Exec(`INSERT INTO running_notes(id,title,run_date,started_at,tags,body,revision,position) VALUES
		('legacy','Run 2026-09-20 07:15','2026-09-20',now(),'{}',E'\n\n### 07:15\nfirst take words here and some more after\n\n### 07:40\nsecond take','r1',1),
		('empty','Run 2026-09-21 07:15','2026-09-21',now(),'{}','','r2',2),
		('named','Tempo notes','2026-09-22',now(),'{}',E'### 08:00\nkept title','r3',3),
		('dated','Audio thought 2026-09-23','2026-09-23',now(),'{}','short one','r4',4),
		('blank','Run 2026-09-24 07:15','2026-09-24',now(),'{}',E'\n','r5',5),
		('padded','Run 2026-09-25 07:15','2026-09-25',now(),'{}',E'\n\thello world','r6',6),
		('crlf','Run 2026-09-26 07:15','2026-09-26',now(),'{}',E'### 07:15\r\nfrom windows\r\n\n\n\nkept gap','r7',7)`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := migrations.ReadFile("migrations/006_audio_thought_titles.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{
		"legacy": {"first take words here and some...", "first take words here and some more after\n\nsecond take"},
		"empty":  {ThoughtPlaceholder, ""},
		"named":  {"Tempo notes", "kept title"},
		"dated":  {"short one", "short one"},
		"blank":  {ThoughtPlaceholder, "\n"},
		"padded": {"hello world", "\n\thello world"},
		"crlf":   {"from windows kept gap", "from windows\r\n\n\n\nkept gap"},
	}
	for id, w := range want {
		var title, body, revision string
		if err = s.DB.QueryRow("SELECT title,body,revision FROM running_notes WHERE id=$1", id).Scan(&title, &body, &revision); err != nil {
			t.Fatal(err)
		}
		if title != w[0] || body != w[1] {
			t.Errorf("%s: got %q / %q", id, title, body)
		}
		if strings.HasPrefix(revision, "r") && len(revision) == 2 {
			t.Errorf("%s: revision not bumped", id)
		}
	}
}
