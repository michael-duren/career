package database

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderAttempts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	input := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "struggled", Minutes: 25, Assisted: true, Notes: "Try a map", TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}
	original, err := s.SaveLeetgrinderAttempt(ctx, input, "")
	if err != nil {
		t.Fatal(err)
	}
	if original.CreatedAt.IsZero() || original.Revision == "" {
		t.Fatal("missing timestamp or revision")
	}
	retry, err := s.SaveLeetgrinderAttempt(ctx, input, "")
	if err != nil || !reflect.DeepEqual(retry, original) {
		t.Fatalf("retry = %+v, %v; want %+v", retry, err, original)
	}
	conflicting := input
	conflicting.Minutes++
	if _, err := s.SaveLeetgrinderAttempt(ctx, conflicting, ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting retry: %v", err)
	}
	correction := original
	correction.Outcome, correction.Assisted = "solved", false
	correction.CreatedAt = correction.CreatedAt.AddDate(-1, 0, 0)
	corrected, err := s.SaveLeetgrinderAttempt(ctx, correction, original.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if corrected.Revision == original.Revision || !corrected.CreatedAt.Equal(original.CreatedAt) || corrected.Outcome != "solved" || corrected.Assisted {
		t.Fatalf("incorrect correction: %+v", corrected)
	}
	if _, err := s.SaveLeetgrinderAttempt(ctx, correction, original.Revision); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale correction: %v", err)
	}
	correction.ProblemSlug = "three-sum"
	if _, err := s.SaveLeetgrinderAttempt(ctx, correction, corrected.Revision); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed problem: %v", err)
	}
	input.ID = uuid.NewString()
	second, err := s.SaveLeetgrinderAttempt(ctx, input, "")
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.LeetgrinderState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.Attempts, []leetgrinder.Attempt{second, corrected}) {
		t.Fatalf("persisted history: %+v", state.Attempts)
	}
	if p := state.Problem("two-sum"); p.Title != "Two Sum" || p.OptimalSource != "curated" {
		t.Fatalf("catalog not loaded: %+v", p)
	}
}

func TestLeetgrinderConcurrentCreate(t *testing.T) {
	s := testStore(t)
	input := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 1}
	var wg sync.WaitGroup
	results := make(chan leetgrinder.Attempt, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, err := s.SaveLeetgrinderAttempt(context.Background(), input, "")
			if err != nil {
				t.Error(err)
			}
			results <- a
		}()
	}
	wg.Wait()
	close(results)
	var first leetgrinder.Attempt
	for a := range results {
		if first.ID == "" {
			first = a
		}
		if !reflect.DeepEqual(a, first) {
			t.Fatal("concurrent retry changed attempt")
		}
	}
}

func TestLeetgrinderInvalidAttempts(t *testing.T) {
	s := testStore(t)
	valid := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 240, Notes: strings.Repeat("界", 2000), TimeComplexity: "O(" + strings.Repeat("m", 37) + ")", SpaceComplexity: "O(1)"}
	for name, change := range map[string]func(*leetgrinder.Attempt){
		"id":           func(a *leetgrinder.Attempt) { a.ID = "bad" },
		"slug":         func(a *leetgrinder.Attempt) { a.ProblemSlug = "../two-sum" },
		"empty slug":   func(a *leetgrinder.Attempt) { a.ProblemSlug = "" },
		"long slug":    func(a *leetgrinder.Attempt) { a.ProblemSlug = strings.Repeat("a", 101) },
		"outcome":      func(a *leetgrinder.Attempt) { a.Outcome = "failed" },
		"zero minutes": func(a *leetgrinder.Attempt) { a.Minutes = 0 },
		"many minutes": func(a *leetgrinder.Attempt) { a.Minutes = 241 },
		"notes":        func(a *leetgrinder.Attempt) { a.Notes += "x" },
	} {
		t.Run(name, func(t *testing.T) {
			a := valid
			change(&a)
			if _, err := s.SaveLeetgrinderAttempt(context.Background(), a, ""); !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid input: %v", err)
			}
		})
	}
	if _, err := s.SaveLeetgrinderAttempt(context.Background(), valid, "bad revision"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid revision: %v", err)
	}
	if _, err := s.SaveLeetgrinderAttempt(context.Background(), valid, ""); err != nil {
		t.Fatalf("boundary values: %v", err)
	}
}

func TestLeetgrinderStateCodeScopes(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	for _, a := range []leetgrinder.Attempt{
		{ProblemSlug: "two-sum", Code: "def two(): pass", CodeLanguage: "python3", TimeComplexity: "O(n)", SpaceComplexity: "O(1)"},
		{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)"},
		{ProblemSlug: "valid-anagram", Code: "def anagram(): pass", CodeLanguage: "python3", TimeComplexity: "O(n)", SpaceComplexity: "O(1)"},
	} {
		a.ID, a.Outcome, a.Minutes = uuid.NewString(), "solved", 20
		if _, err := s.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
	}
	full, err := s.LeetgrinderState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bare, err := s.LeetgrinderStateWithoutCode(ctx)
	if err != nil {
		t.Fatal(err)
	}
	scoped, err := s.LeetgrinderProblemState(ctx, "two-sum")
	if err != nil {
		t.Fatal(err)
	}
	if len(bare.Attempts) != 3 || len(scoped.Attempts) != 3 {
		t.Fatalf("attempts: %d, %d", len(bare.Attempts), len(scoped.Attempts))
	}
	wantBare, wantScoped := full, full
	wantBare.Attempts, wantScoped.Attempts = append([]leetgrinder.Attempt{}, full.Attempts...), append([]leetgrinder.Attempt{}, full.Attempts...)
	for i, a := range full.Attempts {
		wantBare.Attempts[i].Code = ""
		if a.ProblemSlug != "two-sum" {
			wantScoped.Attempts[i].Code = ""
		}
	}
	if !reflect.DeepEqual(bare, wantBare) {
		t.Fatalf("state without code differs beyond code: %+v", bare)
	}
	if !reflect.DeepEqual(scoped, wantScoped) {
		t.Fatalf("problem state differs beyond other problems' code: %+v", scoped)
	}
	codes := 0
	for _, a := range scoped.Attempts {
		if a.Code != "" {
			codes++
			if a.ProblemSlug != "two-sum" {
				t.Fatalf("loaded code of %s", a.ProblemSlug)
			}
		}
	}
	if codes != 1 {
		t.Fatalf("two-sum code attempts = %d, want 1", codes)
	}
}

func TestLeetgrinderTodoSetsMatchesPerSetItems(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	first, err := s.CreateLeetgrinderTodoSet(ctx, "First", []string{"two-sum", "valid-anagram", "group-anagrams"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateLeetgrinderTodoSet(ctx, "Second", []string{"group-anagrams", "two-sum"})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := s.CreateLeetgrinderTodoSet(ctx, "Empty", nil)
	if err != nil {
		t.Fatal(err)
	}
	third, err := s.CreateLeetgrinderTodoSet(ctx, "Third", []string{"top-k-frequent-elements"})
	if err != nil {
		t.Fatal(err)
	}
	// two-sum is shared by two sets and the standalone list.
	if _, err = s.AddLeetgrinderTodoItem(ctx, "", "two-sum"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AddLeetgrinderTodoItem(ctx, "", "contains-duplicate"); err != nil {
		t.Fatal(err)
	}
	// Attempts logged after everything was added.
	for _, a := range []leetgrinder.Attempt{
		{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 5, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"},
		{ID: uuid.NewString(), ProblemSlug: "valid-anagram", Outcome: "struggled", Minutes: 5, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"},
		{ID: uuid.NewString(), ProblemSlug: "top-k-frequent-elements", Outcome: "unfinished", Minutes: 5, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"},
	} {
		if _, err = s.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
	}
	sets, err := s.LeetgrinderTodoSets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []string{first.ID, second.ID, empty.ID, third.ID}
	if len(sets) != len(wantIDs) {
		t.Fatalf("got %d sets, want %d", len(sets), len(wantIDs))
	}
	doneCount := 0
	for i, set := range sets {
		if set.ID != wantIDs[i] {
			t.Fatalf("set %d = %s, want %s", i, set.ID, wantIDs[i])
		}
		want, err := s.LeetgrinderTodoItems(ctx, set.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(set.Items, want) {
			t.Fatalf("set %q items = %+v, want %+v", set.Title, set.Items, want)
		}
		for _, item := range set.Items {
			if !item.DoneAt.IsZero() {
				doneCount++
			}
		}
	}
	if len(sets[2].Items) != 0 || sets[2].Items == nil {
		t.Fatalf("empty set items = %#v, want empty non-nil", sets[2].Items)
	}
	if len(sets[0].Items) != 3 || len(sets[1].Items) != 2 || len(sets[3].Items) != 1 {
		t.Fatalf("item counts = %d,%d,%d", len(sets[0].Items), len(sets[1].Items), len(sets[3].Items))
	}
	// two-sum (in two sets) and valid-anagram are done; the rest are not.
	if doneCount != 3 {
		t.Fatalf("done items = %d, want 3", doneCount)
	}
	// Standalone semantics are unchanged: two-sum left the list once solved.
	standalone, err := s.LeetgrinderTodoItems(ctx, "")
	if err != nil || len(standalone) != 1 || standalone[0].Problem.Slug != "contains-duplicate" {
		t.Fatalf("standalone = %+v, %v", standalone, err)
	}
}
