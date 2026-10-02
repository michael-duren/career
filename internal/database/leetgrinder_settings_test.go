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
	if err != nil || settings.Timezone != "America/Chicago" || settings.Goal != leetgrinder.DefaultGoal || settings.NtfyURL != "https://ntfy.sh" || settings.TokenSet() || len(settings.Notifications) != 0 {
		t.Fatalf("default settings: %+v %v", settings, err)
	}
	for _, table := range []string{"leetgrinder_api_tokens", "leetgrinder_review_plan", "leetgrinder_notification_log"} {
		if _, err = s.DB.Exec("SELECT count(*) FROM " + table); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
	}
	for _, bad := range []string{
		"INSERT INTO leetgrinder_settings(id,revision) VALUES(2,gen_random_uuid())",
		"UPDATE leetgrinder_settings SET goal_new=11",
		"UPDATE leetgrinder_settings SET goal_review=-1",
		"INSERT INTO leetgrinder_daily_goal VALUES('2026-10-01',2,1),('2026-10-01',3,1)",
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
	saved, err := s.UpdateLeetgrinderSettings(ctx, original.Revision, func(v *leetgrinder.Settings) error {
		v.Timezone, v.Goal = "Asia/Tokyo", leetgrinder.DailyGoal{New: 3, Review: 0}
		v.NtfyTokenCiphertext = []byte{1, 2, 3}
		v.Notifications = map[string]leetgrinder.NotificationPref{"goal_incomplete": {Enabled: true, Time: "17:00", Priority: "default"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision == original.Revision || saved.Timezone != "Asia/Tokyo" || saved.Goal != (leetgrinder.DailyGoal{New: 3}) || !saved.TokenSet() || !saved.Notifications["goal_incomplete"].Enabled {
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
	if _, err = s.UpdateLeetgrinderSettings(ctx, saved.Revision, func(v *leetgrinder.Settings) error { v.Goal = leetgrinder.DailyGoal{}; return nil }); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty goal: %v", err)
	}
	sentinel := errors.New("rejected by caller")
	if _, err = s.UpdateLeetgrinderSettings(ctx, saved.Revision, func(*leetgrinder.Settings) error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("caller error: %v", err)
	}
	cleared, err := s.UpdateLeetgrinderSettings(ctx, saved.Revision, func(v *leetgrinder.Settings) error {
		v.NtfyTokenCiphertext = nil
		return nil
	})
	if err != nil || cleared.TokenSet() {
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
	// Two hours a day gives one review slot.
	if settings, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error {
		v.Timezone = "UTC"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -20)
	for _, slug := range []string{"isomorphic-strings", "binary-search", "two-sum"} {
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
		if !reflect.DeepEqual(plan, []string{"binary-search"}) {
			t.Fatalf("concurrent plans disagree: %v", plan)
		}
	}
	// Today's goal is frozen: raising the review target changes only later days.
	if _, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Goal.Review = 2; return nil }); err != nil {
		t.Fatal(err)
	}
	today, err := s.LeetgrinderToday(ctx, now)
	if err != nil || today.Goal != leetgrinder.DefaultGoal || len(today.Reviews) != 1 || today.Reviews[0].Problem.Title != "Binary Search" {
		t.Fatalf("frozen goal: %+v %+v %v", today.Goal, today.Reviews, err)
	}
	if g, ok, err := s.LeetgrinderDailyGoal(ctx, today.Date); err != nil || !ok || g != leetgrinder.DefaultGoal {
		t.Fatalf("stored goal %+v %v %v", g, ok, err)
	}
	if _, ok, err := s.LeetgrinderDailyGoal(ctx, today.Date.AddDate(0, 0, 1)); err != nil || ok {
		t.Fatalf("tomorrow frozen early: %v %v", ok, err)
	}
	// Logging the review marks it done, keeps the plan, and counts toward the goal.
	review := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "binary-search", Outcome: "solved", Minutes: 20, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}
	if _, err = s.SaveLeetgrinderAttempt(ctx, review, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", now.Add(-time.Hour), review.ID); err != nil {
		t.Fatal(err)
	}
	today, err = s.LeetgrinderToday(ctx, now)
	if err != nil || len(today.Reviews) != 1 || !today.Reviews[0].Done || len(today.MissingReviews()) != 0 || today.Progress.Reviews() != 1 || today.Kind("binary-search") != leetgrinder.KindReview {
		t.Fatalf("plan changed or done state wrong: %+v %v", today.Reviews, err)
	}
	var isReview bool
	if err = s.DB.QueryRow("SELECT is_review FROM leetgrinder_attempts WHERE id=$1", review.ID).Scan(&isReview); err != nil || !isReview {
		t.Fatalf("is_review not decided by the server: %v", err)
	}
	// The next day freezes the new goal and picks two reviews.
	tomorrow, err := s.LeetgrinderToday(ctx, now.AddDate(0, 0, 1))
	if err != nil || tomorrow.Goal.Review != 2 || len(tomorrow.Reviews) != 2 {
		t.Fatalf("next day: %+v %+v %v", tomorrow.Goal, tomorrow.Reviews, err)
	}
	var planned int
	if err = s.DB.QueryRow("SELECT count(*) FROM leetgrinder_review_plan WHERE plan_date='2026-10-02'").Scan(&planned); err != nil || planned != 1 {
		t.Fatalf("persisted %d picks: %v", planned, err)
	}
}

// Accesses that would not change today's goal or plan never take the planner
// lock; the rest still do and still agree on one plan.
func TestLeetgrinderTodayReadsWithoutLock(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error {
		v.Timezone = "UTC"
		v.Goal.Review = 2
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -20)
	created := map[string]time.Time{"binary-search": old, "two-sum": now.Add(-time.Hour), "isomorphic-strings": now.Add(-time.Hour)}
	for slug, at := range created {
		a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "unfinished", Minutes: 25}
		if _, err = s.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", at, a.ID); err != nil {
			t.Fatal(err)
		}
	}
	// holdLock takes the planner lock on its own connection until released.
	holdLock := func() func() {
		t.Helper()
		conn, err := s.DB.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock(724193611)"); err != nil {
			t.Fatal(err)
		}
		return func() {
			if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock(724193611)"); err != nil {
				t.Error(err)
			}
			conn.Close()
		}
	}
	// todayWhileLocked reports whether LeetgrinderToday finished while the
	// lock was held.
	todayWhileLocked := func() (leetgrinder.Today, bool) {
		t.Helper()
		release := holdLock()
		defer release()
		short, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		today, err := s.LeetgrinderToday(short, now)
		return today, err == nil
	}
	picks := func(today leetgrinder.Today) []string {
		var slugs []string
		for _, r := range today.Reviews {
			slugs = append(slugs, r.Problem.Slug)
		}
		return slugs
	}
	counts := func() (goals, planned int) {
		t.Helper()
		if err := s.DB.QueryRow("SELECT (SELECT count(*) FROM leetgrinder_daily_goal),(SELECT count(*) FROM leetgrinder_review_plan)").Scan(&goals, &planned); err != nil {
			t.Fatal(err)
		}
		return goals, planned
	}
	concurrent := func(want []string) {
		t.Helper()
		var wg sync.WaitGroup
		plans := make(chan []string, 8)
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				today, err := s.LeetgrinderToday(ctx, now)
				if err != nil {
					t.Error(err)
					return
				}
				plans <- picks(today)
			}()
		}
		wg.Wait()
		close(plans)
		for plan := range plans {
			if !reflect.DeepEqual(plan, want) {
				t.Fatalf("concurrent plans disagree: %v, want %v", plan, want)
			}
		}
	}

	// The first visit freezes the goal, so it needs the lock.
	if _, ok := todayWhileLocked(); ok {
		t.Fatal("first visit did not wait for the planner lock")
	}
	if goals, planned := counts(); goals != 0 || planned != 0 {
		t.Fatalf("blocked first visit wrote %d goals %d picks", goals, planned)
	}
	concurrent([]string{"binary-search"})
	// Only one card is due, so the short plan is read without the lock.
	today, ok := todayWhileLocked()
	if !ok || today.Goal.Review != 2 || !reflect.DeepEqual(picks(today), []string{"binary-search"}) {
		t.Fatalf("short plan with nothing due took the lock: %v %+v %v", ok, today.Goal, picks(today))
	}
	if goals, planned := counts(); goals != 1 || planned != 1 {
		t.Fatalf("persisted %d goals %d picks", goals, planned)
	}
	// Another card falling due fills the open slot, under the lock.
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE problem_slug='two-sum'", old); err != nil {
		t.Fatal(err)
	}
	if _, ok = todayWhileLocked(); ok {
		t.Fatal("extending the plan did not wait for the planner lock")
	}
	concurrent([]string{"binary-search", "two-sum"})
	// A full plan is read without the lock and writes nothing.
	today, ok = todayWhileLocked()
	if !ok || !reflect.DeepEqual(picks(today), []string{"binary-search", "two-sum"}) {
		t.Fatalf("full plan took the lock: %v %v", ok, picks(today))
	}
	if goals, planned := counts(); goals != 1 || planned != 2 {
		t.Fatalf("persisted %d goals %d picks", goals, planned)
	}
	// A higher frozen target with a due card adds a pick.
	if _, err = s.DB.Exec("UPDATE leetgrinder_daily_goal SET goal_review=3"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE problem_slug='isomorphic-strings'", old); err != nil {
		t.Fatal(err)
	}
	if today, err = s.LeetgrinderToday(ctx, now); err != nil || len(today.Reviews) != 3 || today.Reviews[2].Problem.Slug != "isomorphic-strings" {
		t.Fatalf("raised target: %v %v", picks(today), err)
	}
	if goals, planned := counts(); goals != 1 || planned != 3 {
		t.Fatalf("persisted %d goals %d picks", goals, planned)
	}
}

func TestLeetgrinderAttemptSource(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	// The client's review flag is ignored: a first attempt is never a review.
	a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 12, Source: "extension", IsReview: true, TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "class Solution: pass", CodeLanguage: "python3"}
	saved, err := s.SaveLeetgrinderAttempt(ctx, a, "")
	if err != nil || saved.Source != "extension" || saved.IsReview {
		t.Fatalf("saved %+v %v", saved, err)
	}
	retry := a
	retry.IsReview = false
	if again, err := s.SaveLeetgrinderAttempt(ctx, retry, ""); err != nil || again.Revision != saved.Revision {
		t.Fatalf("retry with another review flag: %v", err)
	}
	correction := saved
	correction.Minutes, correction.Source, correction.IsReview = 13, "web", true
	corrected, err := s.SaveLeetgrinderAttempt(ctx, correction, saved.Revision)
	if err != nil || corrected.Source != "extension" || corrected.IsReview || corrected.Minutes != 13 {
		t.Fatalf("correction changed provenance: %+v %v", corrected, err)
	}
	// An attempt after one on an earlier local day is a review.
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=now()-interval '3 days'"); err != nil {
		t.Fatal(err)
	}
	later, err := s.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 5}, "")
	if err != nil || !later.IsReview {
		t.Fatalf("later attempt: %+v %v", later, err)
	}
	a.ID, a.Source = uuid.NewString(), "mobile"
	if _, err = s.SaveLeetgrinderAttempt(ctx, a, ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown source: %v", err)
	}
}
