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
	if len(state.CompletedDays) != 0 {
		t.Fatal("attempt completed a day")
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

func TestLeetgrinderDayCompletion(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	for _, day := range []int{84, 2, 2, 1} {
		if err := s.SetLeetgrinderDay(ctx, day, true); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := s.SetLeetgrinderDay(ctx, 2, false); err != nil {
			t.Fatal(err)
		}
	}
	state, err := s.LeetgrinderState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.CompletedDays, []int{1, 84}) || len(state.Attempts) != 0 {
		t.Fatalf("day state: %+v", state)
	}
	for _, day := range []int{0, 85} {
		if err := s.SetLeetgrinderDay(ctx, day, true); !errors.Is(err, ErrInvalid) {
			t.Fatalf("day %d: %v", day, err)
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
