package database

import (
	"context"
	"fmt"
	"reflect"
	"strings"
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
	// Concurrent first visits return the same plan. (The planner lock itself
	// is proven by waitsForPlannerLock in the top-up and delete tests.)
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
	if today, err = s.LeetgrinderToday(ctx, now); err != nil || today.Progress.New() != 2 {
		t.Fatalf("frozen plan: %v", err)
	}
	want[1].Done = true
	if !reflect.DeepEqual(picks(today), want) {
		t.Fatalf("frozen plan: %+v", picks(today))
	}
	if _, ok := today.NextNewPick(); ok {
		t.Fatal("next pick after all were done")
	}
	// A later day plans from the todos left: problems attempted before it
	// are skipped.
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
	// A todo queued later that day fills the empty plan.
	if _, err = s.AddLeetgrinderTodoItem(ctx, "", "house-robber"); err != nil {
		t.Fatal(err)
	}
	if today, err = s.LeetgrinderToday(ctx, later); err != nil || !reflect.DeepEqual(picks(today), []leetgrinder.NewPickItem{{Slot: 1, Problem: leetgrinder.Problem{Slug: "house-robber"}}}) {
		t.Fatalf("queued later: %+v %v", picks(today), err)
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

// newPicksStore is a store in zone with goal new problems a day picked from
// todos, and a helper that logs an unfinished attempt at a time.
func newPicksStore(t *testing.T, zone string, goal int) (*Store, func(slug string, at time.Time)) {
	t.Helper()
	s := testStore(t)
	ctx := context.Background()
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error {
		v.Timezone, v.Goal, v.NewFromTodos = zone, leetgrinder.DailyGoal{New: goal}, true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return s, func(slug string, at time.Time) {
		t.Helper()
		a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "unfinished", Minutes: 25}
		if _, err := s.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", at, a.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func newPickSlots(today leetgrinder.Today) []string {
	var out []string
	for _, p := range today.NewPicks {
		out = append(out, fmt.Sprintf("%d:%s:%s", p.Slot, p.Problem.Slug, p.SetTitle))
	}
	return out
}

// A problem queued in two sets is picked once, from its oldest entry, so the
// next pick is the following todo rather than a duplicate.
func TestLeetgrinderNewPicksQueuedTwice(t *testing.T) {
	s, _ := newPicksStore(t, "UTC", 2)
	ctx := context.Background()
	if _, err := s.CreateLeetgrinderTodoSet(ctx, "A", []string{"two-sum"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateLeetgrinderTodoSet(ctx, "B", []string{"two-sum", "valid-anagram"}); err != nil {
		t.Fatal(err)
	}
	today, err := s.LeetgrinderToday(ctx, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC))
	if want := []string{"1:two-sum:A", "2:valid-anagram:B"}; err != nil || !reflect.DeepEqual(newPickSlots(today), want) {
		t.Fatalf("picks %v, want %v: %v", newPickSlots(today), want, err)
	}
}

// A day short of its target reads without planning while no todo is
// queued, and tops up after slot 1, under the planner lock, when one is
// added.
func TestLeetgrinderNewPicksTopUp(t *testing.T) {
	s, _ := newPicksStore(t, "UTC", 2)
	ctx := context.Background()
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	if _, err := s.AddLeetgrinderTodoItem(ctx, "", "two-sum"); err != nil {
		t.Fatal(err)
	}
	today, err := s.LeetgrinderToday(ctx, now)
	if want := []string{"1:two-sum:"}; err != nil || !reflect.DeepEqual(newPickSlots(today), want) || !today.NewPicksShort() {
		t.Fatalf("first plan %v: %v", newPickSlots(today), err)
	}
	if _, ok, err := s.leetgrinderTodayPlanned(ctx, now); err != nil || !ok {
		t.Fatalf("short day with nothing queued plans again: %v %v", ok, err)
	}
	for _, slug := range []string{"valid-anagram", "group-anagrams"} {
		if _, err = s.AddLeetgrinderTodoItem(ctx, "", slug); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok, err := s.leetgrinderTodayPlanned(ctx, now); err != nil || ok {
		t.Fatalf("queued todo not planned: %v %v", ok, err)
	}
	// The top-up takes the planner lock, so concurrent visits agree on it.
	done := make(chan []string, 1)
	waitsForPlannerLock(t, s, func() {
		today, err := s.LeetgrinderToday(ctx, now)
		if err != nil {
			t.Error(err)
		}
		done <- newPickSlots(today)
	})
	if slots, want := <-done, []string{"1:two-sum:", "2:valid-anagram:"}; !reflect.DeepEqual(slots, want) {
		t.Fatalf("top-up %v, want %v", slots, want)
	}
	// Later visits read the same plan.
	if today, err = s.LeetgrinderToday(ctx, now); err != nil || !reflect.DeepEqual(newPickSlots(today), []string{"1:two-sum:", "2:valid-anagram:"}) {
		t.Fatalf("reread %v: %v", newPickSlots(today), err)
	}
}

// "Attempted before the day" uses the settings time zone: an attempt on the
// previous local evening, already the plan date in UTC, rules a todo out.
func TestLeetgrinderNewPicksLocalDayStart(t *testing.T) {
	s, attempt := newPicksStore(t, "America/Chicago", 1)
	ctx := context.Background()
	// 01:00 UTC on Oct 2 is 20:00 on Oct 1 in Chicago.
	attempt("two-sum", time.Date(2026, 10, 2, 1, 0, 0, 0, time.UTC))
	if _, err := s.CreateLeetgrinderTodoSet(ctx, "A", []string{"two-sum", "valid-anagram"}); err != nil {
		t.Fatal(err)
	}
	today, err := s.LeetgrinderToday(ctx, time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC))
	if want := []string{"1:valid-anagram:A"}; err != nil || !reflect.DeepEqual(newPickSlots(today), want) {
		t.Fatalf("picks %v, want %v: %v", newPickSlots(today), want, err)
	}
}

// waitsForPlannerLock holds the planner lock, runs call in the background,
// and fails unless call queues for the lock; it then releases the lock.
func waitsForPlannerLock(t *testing.T, s *Store, call func()) {
	t.Helper()
	ctx := context.Background()
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", leetgrinderPlannerLock); err != nil {
		t.Fatal(err)
	}
	go call()
	for deadline := time.Now().Add(5 * time.Second); ; {
		var waiting bool
		if err = s.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype='advisory' AND classid=0 AND objid=$1 AND objsubid=1 AND NOT granted)", leetgrinderPlannerLock).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("call did not wait for the planner lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err = conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", leetgrinderPlannerLock); err != nil {
		t.Fatal(err)
	}
}

// Deleting a set takes the planner lock, so a pick is never saved with the
// ID of a set deleted after the planner read it.
func TestLeetgrinderDeleteTodoSetWaitsForPlanner(t *testing.T) {
	s, _ := newPicksStore(t, "UTC", 1)
	ctx := context.Background()
	set, err := s.CreateLeetgrinderTodoSet(ctx, "A", []string{"two-sum"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	waitsForPlannerLock(t, s, func() { done <- s.DeleteLeetgrinderTodoSet(ctx, set.ID) })
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}

// Today's frozen goal decides how many picks it takes: raising the target
// adds picks from the next day, not today. A pick left unattempted is still
// to do, so the next day picks it again first.
func TestLeetgrinderNewPicksUseFrozenGoal(t *testing.T) {
	s, _ := newPicksStore(t, "UTC", 1)
	ctx := context.Background()
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	if _, err := s.CreateLeetgrinderTodoSet(ctx, "A", []string{"two-sum", "valid-anagram", "group-anagrams"}); err != nil {
		t.Fatal(err)
	}
	if today, err := s.LeetgrinderToday(ctx, now); err != nil || len(today.NewPicks) != 1 {
		t.Fatalf("first plan %v: %v", newPickSlots(today), err)
	}
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Goal.New = 3; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.leetgrinderTodayPlanned(ctx, now); err != nil || !ok {
		t.Fatalf("raised target replans today: %v %v", ok, err)
	}
	if today, err := s.LeetgrinderToday(ctx, now); err != nil || len(today.NewPicks) != 1 {
		t.Fatalf("today after raise %v: %v", newPickSlots(today), err)
	}
	if today, err := s.LeetgrinderToday(ctx, now.AddDate(0, 0, 1)); err != nil || !reflect.DeepEqual(newPickSlots(today), []string{"1:two-sum:A", "2:valid-anagram:A", "3:group-anagrams:A"}) {
		t.Fatalf("next day %v: %v", newPickSlots(today), err)
	}
}

// Attempts earlier today do not rule a todo out, since an attempt today still
// counts as new; a todo already completed today is no longer to do.
func TestLeetgrinderNewPicksAttemptsToday(t *testing.T) {
	s, attempt := newPicksStore(t, "UTC", 1)
	ctx := context.Background()
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	attempt("two-sum", now.Add(-4*time.Hour))
	if _, err := s.AddLeetgrinderTodoItem(ctx, "", "two-sum"); err != nil {
		t.Fatal(err)
	}
	today, err := s.LeetgrinderToday(ctx, now)
	if err != nil || !reflect.DeepEqual(newPickSlots(today), []string{"1:two-sum:"}) || !today.NewPicks[0].Done {
		t.Fatalf("attempted this morning: %+v %v", today.NewPicks, err)
	}
	// Next day: a set problem solved that morning is done, so the next todo
	// is picked.
	tomorrow := now.AddDate(0, 0, 1)
	solved := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "valid-anagram", Outcome: "solved", Minutes: 20, TimeComplexity: "O(n)", SpaceComplexity: "O(1)"}
	if _, err = s.SaveLeetgrinderAttempt(ctx, solved, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", tomorrow.Add(-4*time.Hour), solved.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateLeetgrinderTodoSet(ctx, "A", []string{"valid-anagram", "group-anagrams"}); err != nil {
		t.Fatal(err)
	}
	if today, err = s.LeetgrinderToday(ctx, tomorrow); err != nil || !reflect.DeepEqual(newPickSlots(today), []string{"1:group-anagrams:A"}) {
		t.Fatalf("solved this morning: %v %v", newPickSlots(today), err)
	}
}

// The read-only check also starts the day in the settings zone: east of
// UTC, an attempt after local midnight but before UTC midnight is today's,
// so a todo queued for it still tops up a short day.
func TestLeetgrinderNewPicksPlannedLocalDayStart(t *testing.T) {
	s, attempt := newPicksStore(t, "Asia/Tokyo", 2)
	ctx := context.Background()
	now := time.Date(2026, 10, 2, 3, 0, 0, 0, time.UTC)
	if _, err := s.AddLeetgrinderTodoItem(ctx, "", "two-sum"); err != nil {
		t.Fatal(err)
	}
	if today, err := s.LeetgrinderToday(ctx, now); err != nil || len(today.NewPicks) != 1 {
		t.Fatalf("first plan %v: %v", newPickSlots(today), err)
	}
	// 20:00 UTC on Oct 1 is 05:00 on Oct 2 in Tokyo.
	attempt("valid-anagram", time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC))
	if _, err := s.AddLeetgrinderTodoItem(ctx, "", "valid-anagram"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.leetgrinderTodayPlanned(ctx, now); err != nil || ok {
		t.Fatalf("today's attempt read as an earlier day's: %v %v", ok, err)
	}
	if today, err := s.LeetgrinderToday(ctx, now); err != nil || !reflect.DeepEqual(newPickSlots(today), []string{"1:two-sum:", "2:valid-anagram:"}) {
		t.Fatalf("top-up %v: %v", newPickSlots(today), err)
	}
}

// A failure saving new picks is reported but keeps the day's goal and
// review picks, and a later access plans the new picks.
func TestLeetgrinderNewPicksFailureKeepsReviewPlan(t *testing.T) {
	s, attempt := newPicksStore(t, "UTC", 1)
	ctx := context.Background()
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Goal.Review = 1; return nil }); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	attempt("binary-search", now.AddDate(0, 0, -20))
	if _, err = s.AddLeetgrinderTodoItem(ctx, "", "two-sum"); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		"CREATE FUNCTION fail_new_pick() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'injected'; END$$",
		"CREATE TRIGGER fail_new_pick BEFORE INSERT ON leetgrinder_new_plan FOR EACH ROW EXECUTE FUNCTION fail_new_pick()",
	} {
		if _, err = s.DB.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.PlanLeetgrinderToday(ctx, now); err == nil || !strings.Contains(err.Error(), "injected") {
		t.Fatalf("new-pick failure not reported: %v", err)
	}
	var goals, reviews, picks int
	if err = s.DB.QueryRow("SELECT (SELECT count(*) FROM leetgrinder_daily_goal), (SELECT count(*) FROM leetgrinder_review_plan), (SELECT count(*) FROM leetgrinder_new_plan)").Scan(&goals, &reviews, &picks); err != nil || goals != 1 || reviews != 1 || picks != 0 {
		t.Fatalf("after failure: goals %d reviews %d picks %d: %v", goals, reviews, picks, err)
	}
	// Today's view is still served, with the review pick and the failure.
	today, err := s.LeetgrinderToday(ctx, now)
	if err != nil || !today.NewPicksFailed || len(today.NewPicks) != 0 || len(today.Reviews) != 1 {
		t.Fatalf("view after failure: %+v %v", today.Reviews, err)
	}
	if _, err = s.DB.Exec("DROP TRIGGER fail_new_pick ON leetgrinder_new_plan"); err != nil {
		t.Fatal(err)
	}
	// A failing check on the read-only path also serves the planned day.
	if _, err = s.DB.Exec("ALTER TABLE leetgrinder_todo_items RENAME TO todo_items_away"); err != nil {
		t.Fatal(err)
	}
	if today, err = s.LeetgrinderToday(ctx, now); err != nil || !today.NewPicksFailed || len(today.Reviews) != 1 {
		t.Fatalf("read-only failure: %+v %v", today.Reviews, err)
	}
	if _, err = s.DB.Exec("ALTER TABLE todo_items_away RENAME TO leetgrinder_todo_items"); err != nil {
		t.Fatal(err)
	}
	today, err = s.LeetgrinderToday(ctx, now)
	if err != nil || today.NewPicksFailed || !reflect.DeepEqual(newPickSlots(today), []string{"1:two-sum:"}) || len(today.Reviews) != 1 || today.Reviews[0].Problem.Slug != "binary-search" {
		t.Fatalf("retry: %v %+v %v", newPickSlots(today), today.Reviews, err)
	}
}

// Turning the option on after today's goal is frozen picks today.
func TestLeetgrinderNewPicksTurnedOnSameDay(t *testing.T) {
	s, _ := newPicksStore(t, "UTC", 1)
	ctx := context.Background()
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if settings, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.NewFromTodos = false; return nil }); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	if _, err = s.AddLeetgrinderTodoItem(ctx, "", "two-sum"); err != nil {
		t.Fatal(err)
	}
	if today, err := s.LeetgrinderToday(ctx, now); err != nil || len(today.NewPicks) != 0 {
		t.Fatalf("off: %v %v", newPickSlots(today), err)
	}
	if _, err = s.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.NewFromTodos = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if today, err := s.LeetgrinderToday(ctx, now); err != nil || !reflect.DeepEqual(newPickSlots(today), []string{"1:two-sum:"}) {
		t.Fatalf("turned on: %v %v", newPickSlots(today), err)
	}
}
