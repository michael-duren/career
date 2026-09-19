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
