package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/michael-duren/career-strategy/internal/scheduler"
	"sync"
	"testing"
	"time"
)

func TestSchedulerConcurrentReservationsAndStaleDraft(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	w, err := s.SchedulerWeek(ctx, "2030-01-07", now, nil)
	if err != nil {
		t.Fatal(err)
	}
	session := scheduler.Session{Date: "2030-01-07", Assignment: scheduler.Assignment{Title: "Work"}, Plan: &scheduler.Plan{Start: now.Add(9 * time.Hour), End: now.Add(10 * time.Hour)}}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: w.Week, Action: "session", Session: &session}, now, nil)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
	fresh, err := s.SchedulerWeek(ctx, w.Week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh.Sessions) != 1 {
		t.Fatalf("sessions=%d", len(fresh.Sessions))
	}
	_, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: fresh.Revision, Week: w.Week, Action: "session", Session: &session}, now, nil)
	var reservation *scheduler.Conflict
	if !errors.As(err, &reservation) {
		t.Fatalf("overlapping fresh save: %v", err)
	}
}
func TestSchedulerIdleTickDoesNotRotateRevisionOrDirtyOutbox(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	w, err := s.SchedulerWeek(ctx, "2030-01-07", now, nil)
	if err != nil {
		t.Fatal(err)
	}
	var generationBefore int64
	if err = s.DB.QueryRowContext(ctx, "SELECT generation FROM scheduler_google_outbox WHERE id=1").Scan(&generationBefore); err != nil {
		t.Fatal(err)
	}
	// An idle worker tick a minute later, with nothing else having changed,
	// must not rotate the revision (open scheduler tabs would spuriously
	// conflict) or dirty the Google export outbox (a full resync every tick).
	if err = s.SchedulerTick(ctx, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var generationAfter int64
	if err = s.DB.QueryRowContext(ctx, "SELECT generation FROM scheduler_google_outbox WHERE id=1").Scan(&generationAfter); err != nil {
		t.Fatal(err)
	}
	if generationAfter != generationBefore {
		t.Fatalf("idle tick dirtied the Google outbox: %d -> %d", generationBefore, generationAfter)
	}
	session := scheduler.Session{Date: "2030-01-07", Assignment: scheduler.Assignment{Title: "Work"}, Plan: &scheduler.Plan{Start: now.Add(9 * time.Hour), End: now.Add(10 * time.Hour)}}
	if _, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: w.Week, Action: "session", Session: &session}, now.Add(time.Minute), nil); err != nil {
		t.Fatalf("idle tick stale-revisioned an open draft: %v", err)
	}
}
func TestSchedulerBackupRoundtripAndInitialTimeZone(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	if err := s.SchedulerInitializeTimeZone(ctx, "America/Chicago"); err != nil {
		t.Fatal(err)
	}
	if err := s.SchedulerInitializeTimeZone(ctx, "Asia/Tokyo"); err != nil {
		t.Fatal(err)
	}
	doc, err := s.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Settings.TimeZone != "America/Chicago" {
		t.Fatal("browser changed persisted zone")
	}
	var archive bytes.Buffer
	if err = s.Export(ctx, &archive); err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err = json.Unmarshal(archive.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if raw["scheduler"] == nil {
		t.Fatal("schedule missing from export")
	}
	dest := testStore(t)
	if _, err = dest.Import(ctx, bytes.NewReader(archive.Bytes()), Source{}, false); err != nil {
		t.Fatal(err)
	}
	restored, err := dest.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Settings.TimeZone != "America/Chicago" {
		t.Fatal("schedule not restored")
	}
}
