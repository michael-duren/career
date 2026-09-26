package leetgrinder

import "testing"

func TestProgressSeparatesSessionAndSolveCounts(t *testing.T) {
	state := State{CompletedDays: []int{1, 3}, Attempts: []Attempt{
		{ProblemSlug: "two-sum", Outcome: "solved", Assisted: true},
		{ProblemSlug: "two-sum", Outcome: "solved"},
		{ProblemSlug: "valid-anagram", Outcome: "unfinished"},
		{ProblemSlug: "contains-duplicate", Outcome: "solved", Assisted: true},
	}}
	got := Summarize(state)
	if got.Solved != 2 || got.Independent != 1 || got.Completed != 2 || got.NextDay != 2 {
		t.Fatalf("unexpected progress: %+v", got)
	}
}
func TestFinishedCurriculumHasNoNextDay(t *testing.T) {
	state := State{}
	for i := 1; i <= 84; i++ {
		state.CompletedDays = append(state.CompletedDays, i)
	}
	if got := Summarize(state); got.NextDay != 0 || got.Completed != 84 {
		t.Fatalf("unexpected completed progress: %+v", got)
	}
}
