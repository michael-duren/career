package database

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderManualOptimal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	estimate := leetgrinder.AnalysisResult{OptimalTime: "O(n²)", OptimalSpace: "O(1)", OptimalNote: "guess"}

	// A model estimate fills a problem with none, then a manual value
	// replaces it and later estimates never overwrite it.
	if err := saveLeetgrinderModelOptimal(ctx, s.DB, "some-new-problem", estimate); err != nil {
		t.Fatal(err)
	}
	p, _ := s.LeetgrinderProblem(ctx, "some-new-problem")
	if p.OptimalSource != "model" || p.OptimalTime != "O(n²)" {
		t.Fatalf("model estimate: %+v", p)
	}
	if err := s.SaveLeetgrinderManualOptimal(ctx, "some-new-problem", "O(n)", "O(n)", "hash map", p.OptimalRevision()); err != nil {
		t.Fatal(err)
	}
	p, _ = s.LeetgrinderProblem(ctx, "some-new-problem")
	if p.OptimalSource != "manual" || p.OptimalTime != "O(n)" || p.OptimalSpace != "O(n)" || p.OptimalNote != "hash map" {
		t.Fatalf("manual value: %+v", p)
	}
	if err := saveLeetgrinderModelOptimal(ctx, s.DB, "some-new-problem", estimate); err != nil {
		t.Fatal(err)
	}
	if q, _ := s.LeetgrinderProblem(ctx, "some-new-problem"); q.OptimalSource != p.OptimalSource || q.OptimalTime != p.OptimalTime || q.OptimalNote != p.OptimalNote {
		t.Fatalf("model estimate overwrote a manual value: %+v", q)
	}

	// Curated values are never overwritten by estimates, and the seed keeps
	// a manual value across restarts.
	if err := saveLeetgrinderModelOptimal(ctx, s.DB, "two-sum", estimate); err != nil {
		t.Fatal(err)
	}
	two, _ := s.LeetgrinderProblem(ctx, "two-sum")
	if two.OptimalSource != "curated" || two.OptimalTime != "O(n)" {
		t.Fatalf("curated overwritten: %+v", two)
	}
	if err := s.SaveLeetgrinderManualOptimal(ctx, "two-sum", "O(n log n)", "O(1)", "", two.OptimalRevision()); err != nil {
		t.Fatal(err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = seedLeetgrinderProblems(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if two, _ = s.LeetgrinderProblem(ctx, "two-sum"); two.OptimalSource != "manual" || two.OptimalTime != "O(n log n)" {
		t.Fatalf("seed replaced a manual value: %+v", two)
	}

	// Validation and stale revisions.
	rev := two.OptimalRevision()
	for name, args := range map[string][3]string{
		"empty time":  {"", "O(1)", ""},
		"empty space": {"O(1)", "", ""},
		"not big-O":   {"fast", "O(1)", ""},
		"not canon":   {"O(n^2)", "O(1)", ""},
		"long note":   {"O(1)", "O(1)", strings.Repeat("x", leetgrinder.MaxOptimalNote+1)},
	} {
		if err := s.SaveLeetgrinderManualOptimal(ctx, "two-sum", args[0], args[1], args[2], rev); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if err := s.SaveLeetgrinderManualOptimal(ctx, "Bad Slug", "O(1)", "O(1)", "", rev); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad slug: %v", err)
	}
	if err := s.SaveLeetgrinderManualOptimal(ctx, "two-sum", "O(1)", "O(1)", "", "stale"); !errors.Is(err, ErrConflict) {
		t.Errorf("stale revision: %v", err)
	}
	if two, _ = s.LeetgrinderProblem(ctx, "two-sum"); two.OptimalTime != "O(n log n)" {
		t.Fatalf("rejected save changed the value: %+v", two)
	}
}

func TestLeetgrinderClearModelOptimal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	estimate := leetgrinder.AnalysisResult{OptimalTime: "O(n²)", OptimalSpace: "O(1)"}
	if err := saveLeetgrinderModelOptimal(ctx, s.DB, "estimated-problem", estimate); err != nil {
		t.Fatal(err)
	}
	p, _ := s.LeetgrinderProblem(ctx, "estimated-problem")
	if err := s.ClearLeetgrinderModelOptimal(ctx, "estimated-problem", "stale"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale revision: %v", err)
	}
	if err := s.ClearLeetgrinderModelOptimal(ctx, "estimated-problem", p.OptimalRevision()); err != nil {
		t.Fatal(err)
	}
	if p, _ = s.LeetgrinderProblem(ctx, "estimated-problem"); p.OptimalSource != "" || p.HasOptimal() || p.OptimalNote != "" {
		t.Fatalf("not cleared: %+v", p)
	}
	// The next analysis estimates again.
	if err := saveLeetgrinderModelOptimal(ctx, s.DB, "estimated-problem", leetgrinder.AnalysisResult{OptimalTime: "O(n)", OptimalSpace: "O(n)"}); err != nil {
		t.Fatal(err)
	}
	if p, _ = s.LeetgrinderProblem(ctx, "estimated-problem"); p.OptimalSource != "model" || p.OptimalTime != "O(n)" {
		t.Fatalf("re-estimate: %+v", p)
	}

	// Curated and manual values are never cleared, even with a fresh revision.
	two, _ := s.LeetgrinderProblem(ctx, "two-sum")
	if err := s.ClearLeetgrinderModelOptimal(ctx, "two-sum", two.OptimalRevision()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("curated: %v", err)
	}
	if err := s.SaveLeetgrinderManualOptimal(ctx, "estimated-problem", "O(1)", "O(1)", "", p.OptimalRevision()); err != nil {
		t.Fatal(err)
	}
	p, _ = s.LeetgrinderProblem(ctx, "estimated-problem")
	if err := s.ClearLeetgrinderModelOptimal(ctx, "estimated-problem", p.OptimalRevision()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("manual: %v", err)
	}
	two, _ = s.LeetgrinderProblem(ctx, "two-sum")
	p, _ = s.LeetgrinderProblem(ctx, "estimated-problem")
	if two.OptimalSource != "curated" || p.OptimalSource != "manual" || p.OptimalTime != "O(1)" {
		t.Fatalf("values changed: %+v %+v", two, p)
	}
}
