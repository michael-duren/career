package database

import (
	"context"
	"errors"
	"github.com/michael-duren/career-strategy/internal/scheduler"
	"testing"
	"time"
)

func TestSchedulerGoogleCredentialsRevisionsAndDisconnect(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.SchedulerGoogle(ctx)
	if err != nil || c.Revision != "0" {
		t.Fatalf("initial=%+v %v", c, err)
	}
	sealed, err := scheduler.SealGoogleSecret(make([]byte, 32), []byte(`{"refresh_token":"private"}`))
	if err != nil {
		t.Fatal(err)
	}
	c.AccountID = "account"
	c.CalendarID = "calendar"
	c.Credentials = sealed
	c, err = s.SaveSchedulerGoogle(ctx, c, "0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveSchedulerGoogle(ctx, c, "0"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale save=%v", err)
	}
	m := SchedulerGoogleMapping{AccountID: "account", SessionID: "session", CalendarID: "calendar", EventID: "event", PlannedStart: time.Now()}
	if err = s.SaveSchedulerGoogleMapping(ctx, m); err != nil {
		t.Fatal(err)
	}
	c.Credentials = nil
	c, err = s.SaveSchedulerGoogle(ctx, c, c.Revision)
	if err != nil {
		t.Fatal(err)
	}
	read, err := s.SchedulerGoogle(ctx)
	if err != nil || len(read.Credentials) != 0 || read.CalendarID != "calendar" {
		t.Fatalf("disconnect=%+v %v", read, err)
	}
	mappings, err := s.SchedulerGoogleMappings(ctx, "account")
	if err != nil || len(mappings) != 1 {
		t.Fatalf("mappings lost: %+v %v", mappings, err)
	}
}
func TestSchedulerOAuthSessionBindingAndSingleUse(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.StoreSchedulerOAuth(ctx, "state", "session", "verifier"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConsumeSchedulerOAuth(ctx, "state", "other-session"); err == nil {
		t.Fatal("wrong session accepted")
	}
	got, err := s.ConsumeSchedulerOAuth(ctx, "state", "session")
	if err != nil || got != "verifier" {
		t.Fatalf("consume=%s %v", got, err)
	}
	if _, err = s.ConsumeSchedulerOAuth(ctx, "state", "session"); err == nil {
		t.Fatal("state replay accepted")
	}
}
func TestSchedulerGoogleOutboxDoesNotLoseConcurrentWork(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c, err := s.SchedulerGoogle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c, err = s.SaveSchedulerGoogle(ctx, c, c.Revision)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.SchedulerGoogleOutbox(ctx)
	if err != nil || first == 0 {
		t.Fatal("missing durable job", err)
	}
	_, err = s.SaveSchedulerGoogle(ctx, c, c.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteSchedulerGoogleOutbox(ctx, first); err != nil {
		t.Fatal(err)
	}
	newer, err := s.SchedulerGoogleOutbox(ctx)
	if err != nil || newer <= first {
		t.Fatalf("lost concurrent work: %d %v", newer, err)
	}
}

func TestSchedulerGoogleProgressCannotCompleteNewDesiredWork(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	old := SchedulerGoogleMapping{AccountID: "account", CalendarID: "calendar", SessionID: "session", EventID: "stable", PlannedStart: time.Now(), DesiredFingerprint: "old", DesiredGeneration: 3}
	if err := s.PrepareSchedulerGoogleMapping(ctx, old); err != nil {
		t.Fatal(err)
	}
	newer := old
	newer.DesiredFingerprint = "new"
	newer.DesiredGeneration = 4
	if err := s.PrepareSchedulerGoogleMapping(ctx, newer); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteSchedulerGoogleMapping(ctx, old); err != nil {
		t.Fatal(err)
	}
	mappings, err := s.SchedulerGoogleMappings(ctx, "account")
	if err != nil || len(mappings) != 1 {
		t.Fatalf("read=%+v %v", mappings, err)
	}
	if got := mappings[0]; got.DesiredFingerprint != "new" || got.DesiredGeneration != 4 || got.SyncedFingerprint != "" || got.EventID != "stable" {
		t.Fatalf("stale completion overwrote desired progress: %+v", got)
	}
	if err := s.CompleteSchedulerGoogleMapping(ctx, newer); err != nil {
		t.Fatal(err)
	}
	mappings, err = s.SchedulerGoogleMappings(ctx, "account")
	if err != nil || mappings[0].SyncedFingerprint != "new" {
		t.Fatalf("completion=%+v %v", mappings, err)
	}
	replacement := newer
	replacement.EventID = "replacement"
	if err := s.PrepareSchedulerGoogleMapping(ctx, replacement); err != nil {
		t.Fatal(err)
	}
	mappings, err = s.SchedulerGoogleMappings(ctx, "account")
	if err != nil || mappings[0].SyncedFingerprint != "" {
		t.Fatalf("identity retained old success: %+v %v", mappings, err)
	}
}
func TestSchedulerGoogleReconciliationCursorScopedToAuthority(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.SaveSchedulerGoogleReconciliationCursor(ctx, "account", "calendar", "session-020"); err != nil {
		t.Fatal(err)
	}
	for _, authority := range [][2]string{{"account", "calendar"}, {"other", "calendar"}, {"account", "other"}} {
		got, err := s.SchedulerGoogleReconciliationCursor(ctx, authority[0], authority[1])
		if err != nil {
			t.Fatal(err)
		}
		want := ""
		if authority[0] == "account" && authority[1] == "calendar" {
			want = "session-020"
		}
		if got != want {
			t.Fatalf("authority %v cursor=%s", authority, got)
		}
	}
}
