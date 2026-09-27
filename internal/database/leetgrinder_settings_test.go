package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderScheduleMigrationKeepsAttempts(t *testing.T) {
	s := emptyStore(t)
	ctx := context.Background()
	if _, err := s.DB.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		version, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil {
			t.Fatal(err)
		}
		if version > 10 {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec(string(sql)); err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec("INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", version, fmt.Sprintf("%x", sha256.Sum256(sql))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.DB.Exec(`INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision) VALUES('11111111-1111-4111-8111-111111111111','two-sum','solved',23,false,'Keep this','22222222-2222-4222-8222-222222222222')`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err = s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.CheckSchema(ctx); err != nil {
		t.Fatal(err)
	}
	state, err := s.LeetgrinderState(ctx)
	if err != nil || len(state.Attempts) != 1 || state.Attempts[0].Source != "web" || state.Attempts[0].IsReview || state.Attempts[0].Notes != "Keep this" {
		t.Fatalf("existing attempt changed: %+v %v", state, err)
	}
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil || settings.StartDate != nil || settings.Timezone != "America/Chicago" || settings.DailyHours != 2 || settings.NtfyURL != "https://ntfy.sh" || settings.TokenSet() || len(settings.Notifications) != 0 {
		t.Fatalf("default settings: %+v %v", settings, err)
	}
	for _, table := range []string{"leetgrinder_api_tokens", "leetgrinder_review_plan", "leetgrinder_notification_log"} {
		if _, err = s.DB.Exec("SELECT count(*) FROM " + table); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
	}
	for _, bad := range []string{
		"INSERT INTO leetgrinder_settings(id,revision) VALUES(2,gen_random_uuid())",
		"UPDATE leetgrinder_settings SET daily_hours=4.5",
		"UPDATE leetgrinder_attempts SET source='mobile'",
		"INSERT INTO leetgrinder_review_plan VALUES('2026-10-01','two-sum',1),('2026-10-01','two-sum',2)",
		"INSERT INTO leetgrinder_notification_log(kind,local_date,status) VALUES('missing_work','2026-10-01','sent'),('missing_work','2026-10-01','sent')",
	} {
		if _, err = s.DB.Exec(bad); err == nil {
			t.Errorf("constraint accepted: %s", bad)
		}
	}
}

func TestLeetgrinderSettingsRevisions(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	original, err := s.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	saved, err := s.UpdateLeetgrinderSettings(ctx, original.Revision, func(v *leetgrinder.Settings) error {
		v.StartDate, v.Timezone, v.DailyHours = &start, "Asia/Tokyo", 3.5
		v.NtfyTokenCiphertext = []byte{1, 2, 3}
		v.Notifications = map[string]leetgrinder.NotificationPref{"missing_work": {Enabled: true, Time: "17:00", Priority: "default"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision == original.Revision || !saved.StartDate.Equal(start) || saved.Timezone != "Asia/Tokyo" || saved.DailyHours != 3.5 || !saved.TokenSet() || !saved.Notifications["missing_work"].Enabled {
		t.Fatalf("saved: %+v", saved)
	}
	loaded, err := s.LeetgrinderSettings(ctx)
	if err != nil || !reflect.DeepEqual(loaded, saved) {
		t.Fatalf("reloaded %+v, want %+v (%v)", loaded, saved, err)
	}
	if stored, err := s.LeetgrinderNtfyTokenStored(ctx); err != nil || !stored {
		t.Fatalf("token stored: %v %v", stored, err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, original.Revision, func(*leetgrinder.Settings) error { return nil }); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale revision: %v", err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, "not-a-uuid", func(*leetgrinder.Settings) error { return nil }); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad revision: %v", err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, saved.Revision, func(v *leetgrinder.Settings) error { v.DailyHours = 5; return nil }); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid hours: %v", err)
	}
	sentinel := errors.New("rejected by caller")
	if _, err = s.UpdateLeetgrinderSettings(ctx, saved.Revision, func(*leetgrinder.Settings) error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("caller error: %v", err)
	}
	cleared, err := s.UpdateLeetgrinderSettings(ctx, saved.Revision, func(v *leetgrinder.Settings) error {
		v.StartDate, v.NtfyTokenCiphertext = nil, nil
		return nil
	})
	if err != nil || cleared.StartDate != nil || cleared.TokenSet() {
		t.Fatalf("clear: %+v %v", cleared, err)
	}
	// Concurrent writers with one revision: exactly one wins.
	var wins, conflicts int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.UpdateLeetgrinderSettings(ctx, cleared.Revision, func(v *leetgrinder.Settings) error { v.NtfyTopic = fmt.Sprintf("topic-%d", i); return nil })
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				wins++
			case errors.Is(err, ErrConflict):
				conflicts++
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if wins != 1 || conflicts != 5 {
		t.Fatalf("wins %d conflicts %d", wins, conflicts)
	}
}

func TestLeetgrinderTodayFreezesReviewPlan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	// Day 2 has only optional reading, so it has one base review slot.
	if settings, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error {
		v.StartDate, v.Timezone = &start, "UTC"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -20)
	for _, slug := range []string{"contains-duplicate", "binary-search", "two-sum"} {
		if _, err = s.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "unfinished", Minutes: 25}, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1", old); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	plans := make(chan []string, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			today, err := s.LeetgrinderToday(ctx, now)
			if err != nil {
				t.Error(err)
				return
			}
			var slugs []string
			for _, r := range today.Reviews {
				slugs = append(slugs, r.Problem.Slug)
			}
			plans <- slugs
		}()
	}
	wg.Wait()
	close(plans)
	for plan := range plans {
		if !reflect.DeepEqual(plan, []string{"contains-duplicate"}) {
			t.Fatalf("concurrent plans disagree: %v", plan)
		}
	}
	// More hours add picks after the frozen one.
	if _, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.DailyHours = 2.5; return nil }); err != nil {
		t.Fatal(err)
	}
	today, err := s.LeetgrinderToday(ctx, now)
	if err != nil || len(today.Reviews) != 2 || today.Reviews[0].Problem.Slug != "contains-duplicate" || today.Reviews[1].Problem.Slug != "two-sum" || today.Session != 2 {
		t.Fatalf("topped-up plan: %+v %v", today.Reviews, err)
	}
	// Logging the review marks it done and keeps the plan.
	review := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "contains-duplicate", Outcome: "solved", Minutes: 20, IsReview: true, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}
	if _, err = s.SaveLeetgrinderAttempt(ctx, review, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", now.Add(-time.Hour), review.ID); err != nil {
		t.Fatal(err)
	}
	today, err = s.LeetgrinderToday(ctx, now)
	if err != nil || len(today.Reviews) != 2 || !today.Reviews[0].Done || today.Reviews[1].Done || len(today.Missing().Reviews) != 1 {
		t.Fatalf("plan changed or done state wrong: %+v %v", today.Reviews, err)
	}
	var planned int
	if err = s.DB.QueryRow("SELECT count(*) FROM leetgrinder_review_plan WHERE plan_date='2026-10-02'").Scan(&planned); err != nil || planned != 2 {
		t.Fatalf("persisted %d picks: %v", planned, err)
	}
	var isReview bool
	if err = s.DB.QueryRow("SELECT is_review FROM leetgrinder_attempts WHERE id=$1", review.ID).Scan(&isReview); err != nil || !isReview {
		t.Fatalf("is_review not saved: %v", err)
	}
}

func TestLeetgrinderAttemptSource(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 12, Source: "extension", IsReview: true, TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "class Solution: pass", CodeLanguage: "python3"}
	saved, err := s.SaveLeetgrinderAttempt(ctx, a, "")
	if err != nil || saved.Source != "extension" || !saved.IsReview {
		t.Fatalf("saved %+v %v", saved, err)
	}
	retry := a
	retry.IsReview = false
	if _, err = s.SaveLeetgrinderAttempt(ctx, retry, ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed retry: %v", err)
	}
	correction := saved
	correction.Minutes, correction.Source, correction.IsReview = 13, "web", false
	corrected, err := s.SaveLeetgrinderAttempt(ctx, correction, saved.Revision)
	if err != nil || corrected.Source != "extension" || !corrected.IsReview || corrected.Minutes != 13 {
		t.Fatalf("correction changed provenance: %+v %v", corrected, err)
	}
	a.ID, a.Source = uuid.NewString(), "mobile"
	if _, err = s.SaveLeetgrinderAttempt(ctx, a, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown source: %v", err)
	}
}
