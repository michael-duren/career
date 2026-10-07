package database

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderCorrectnessPersistenceAndAnalysis(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	answers := leetgrinder.Correctness{Claim: "Returns the pair.\nEven with duplicates.", Invariant: "Earlier values are stored.", Initially: "The map is empty.", AfterStep: "Store the current value.", Therefore: "The invariant is preserved.", Termination: "All indices are checked."}
	input := leetgrinder.Attempt{Correctness: answers, ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 10, Source: "extension", TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: "return []", CodeLanguage: "python3"}
	saved, err := s.SaveLeetgrinderAttempt(ctx, input, "")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Correctness != answers {
		t.Fatalf("saved answers: %+v", saved.Correctness)
	}
	retry, err := s.SaveLeetgrinderAttempt(ctx, input, "")
	if err != nil || !reflect.DeepEqual(saved, retry) {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	input.Claim = "Different guarantee"
	if _, err := s.SaveLeetgrinderAttempt(ctx, input, ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting retry: %v", err)
	}
	job, found, err := s.NextLeetgrinderAnalysis(ctx, time.Now())
	if err != nil || !found || job.Attempt.Correctness != answers {
		t.Fatalf("job: %+v %v %v", job, found, err)
	}
	result := leetgrinder.AnalysisResult{ActualTime: "O(1)", ActualSpace: "O(1)", CorrectnessFeedback: "Claim: The code returns no pair.\nTermination: The empty result violates the claim."}
	if err := s.FinishLeetgrinderAnalysis(ctx, job, leetgrinder.AnalysisDone, 1, result, "test", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	state, err := s.LeetgrinderState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := state.Analyses[saved.ID]; !got.Done() || got.CorrectnessFeedback != result.CorrectnessFeedback {
		t.Fatalf("analysis: %+v", got)
	}
	// Editing each answer invalidates the previous assessment, including removing it.
	for _, field := range saved.Correctness.Fields() {
		correction := saved
		switch field.Name {
		case "claim":
			correction.Claim = ""
		case "invariant":
			correction.Invariant = "changed"
		case "initially":
			correction.Initially = "changed"
		case "afterStep":
			correction.AfterStep = "changed"
		case "therefore":
			correction.Therefore = "changed"
		case "termination":
			correction.Termination = "changed"
		}
		saved, err = s.SaveLeetgrinderAttempt(ctx, correction, saved.Revision)
		if err != nil {
			t.Fatal(err)
		}
		job, found, err = s.NextLeetgrinderAnalysis(ctx, time.Now())
		if err != nil || !found || job.Attempt.Correctness != saved.Correctness || job.Tries != 0 {
			t.Fatalf("edited %s job: %+v %v %v", field.Name, job, found, err)
		}
		if err := s.FinishLeetgrinderAnalysis(ctx, job, leetgrinder.AnalysisDone, 1, result, "test", "", time.Now()); err != nil {
			t.Fatal(err)
		}
	}
}
