package database

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderTodayPicksNewFromTodos(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if settings, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error {
		v.Timezone, v.Goal, v.NewFromTodos = "UTC", leetgrinder.DailyGoal{New: 2, Review: 0}, true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	attempt := func(slug string, at time.Time) {
		t.Helper()
		a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "unfinished", Minutes: 25}
		if _, err := s.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", at, a.ID); err != nil {
			t.Fatal(err)
		}
	}
	// Attempted before today, so an attempt on it would not be new.
	attempt("contains-duplicate", now.AddDate(0, 0, -20))
	if _, err = s.AddLeetgrinderTodoItem(ctx, "", "binary-search"); err != nil {
		t.Fatal(err)
	}
	set, err := s.CreateLeetgrinderTodoSet(ctx, "Blind 75", []string{"contains-duplicate", "two-sum", "valid-anagram", "group-anagrams"})
	if err != nil {
		t.Fatal(err)
	}
	// A problem queued twice is picked once, from its oldest entry.
	if _, err = s.CreateLeetgrinderTodoSet(ctx, "Later", []string{"two-sum"}); err != nil {
		t.Fatal(err)
	}
	picks := func(today leetgrinder.Today) []leetgrinder.NewPickItem {
		out := []leetgrinder.NewPickItem{}
		for _, p := range today.NewPicks {
			p.Problem = leetgrinder.Problem{Slug: p.Problem.Slug}
			out = append(out, p)
		}
		return out
	}
	want := []leetgrinder.NewPickItem{
		{Slot: 1, Problem: leetgrinder.Problem{Slug: "binary-search"}},
		{Slot: 2, Problem: leetgrinder.Problem{Slug: "two-sum"}, SetTitle: "Blind 75"},
	}
	// Concurrent first visits agree on one plan.
	var wg sync.WaitGroup
	got := make(chan []leetgrinder.NewPickItem, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			today, err := s.LeetgrinderToday(ctx, now)
			if err != nil {
				t.Error(err)
				return
			}
			got <- picks(today)
		}()
	}
	wg.Wait()
	close(got)
	for p := range got {
		if !reflect.DeepEqual(p, want) {
			t.Fatalf("picks %+v, want %+v", p, want)
		}
	}
	// An attempt marks its pick done; the next pick is the first left.
	attempt("binary-search", now.Add(-time.Hour))
	today, err := s.LeetgrinderToday(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	want[0].Done = true
	if next, ok := today.NextNewPick(); !reflect.DeepEqual(picks(today), want) || !ok || next.Problem.Slug != "two-sum" || next.SetTitle != "Blind 75" {
		t.Fatalf("after one attempt: %+v next %+v %v", picks(today), next, ok)
	}
	// Picks are frozen: a finished pick is not replaced the same day.
	attempt("two-sum", now.Add(-time.Hour))
	if today, err = s.LeetgrinderToday(ctx, now); err != nil || len(today.NewPicks) != 2 || today.Progress.New() != 2 {
		t.Fatalf("frozen plan: %+v %v", today.NewPicks, err)
	}
	if _, ok := today.NextNewPick(); ok {
		t.Fatal("next pick after all were done")
	}
	// Raising today's frozen new goal is not possible, but a later day plans
	// from the todos left: problems attempted before it are skipped.
	tomorrow := now.AddDate(0, 0, 1)
	if today, err = s.LeetgrinderToday(ctx, tomorrow); err != nil {
		t.Fatal(err)
	}
	if want := []leetgrinder.NewPickItem{
		{Slot: 1, Problem: leetgrinder.Problem{Slug: "valid-anagram"}, SetTitle: "Blind 75"},
		{Slot: 2, Problem: leetgrinder.Problem{Slug: "group-anagrams"}, SetTitle: "Blind 75"},
	}; !reflect.DeepEqual(picks(today), want) {
		t.Fatalf("tomorrow %+v", picks(today))
	}
	// Deleting the set keeps the frozen picks without a set name.
	if err = s.DeleteLeetgrinderTodoSet(ctx, set.ID); err != nil {
		t.Fatal(err)
	}
	if today, err = s.LeetgrinderToday(ctx, tomorrow); err != nil || len(today.NewPicks) != 2 || today.NewPicks[0].SetTitle != "" {
		t.Fatalf("after set delete: %+v %v", today.NewPicks, err)
	}
	// With no todo left to pick, a day plans no picks and the next access
	// reads without planning again.
	later := now.AddDate(0, 0, 2)
	if today, err = s.LeetgrinderToday(ctx, later); err != nil || len(today.NewPicks) != 0 {
		t.Fatalf("no todos left: %+v %v", today.NewPicks, err)
	}
	if _, ok, err := s.leetgrinderTodayPlanned(ctx, later); err != nil || !ok {
		t.Fatalf("empty todo queue needs planning: %v %v", ok, err)
	}
	// Turned off, a day picks nothing and frozen picks are hidden.
	if _, err = s.AddLeetgrinderTodoItem(ctx, "", "climbing-stairs"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.NewFromTodos = false; return nil }); err != nil {
		t.Fatal(err)
	}
	if today, err = s.LeetgrinderToday(ctx, now.AddDate(0, 0, 3)); err != nil || len(today.NewPicks) != 0 {
		t.Fatalf("off: %+v %v", today.NewPicks, err)
	}
	if today, err = s.LeetgrinderToday(ctx, tomorrow); err != nil || len(today.NewPicks) != 0 {
		t.Fatalf("off hides frozen picks: %+v %v", today.NewPicks, err)
	}
	var planned int
	if err = s.DB.QueryRow("SELECT count(*) FROM leetgrinder_new_plan WHERE plan_date='2026-10-05'").Scan(&planned); err != nil || planned != 0 {
		t.Fatalf("off planned %d picks: %v", planned, err)
	}
}
