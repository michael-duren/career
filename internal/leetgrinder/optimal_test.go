package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestEveryProblemHasOptimalComplexity(t *testing.T) {
	count := 0
	for _, week := range Curriculum() {
		for _, day := range week.Days {
			for _, p := range append(append([]Problem{}, day.Core...), day.Optional...) {
				count++
				// Stored values must already be in the normalised form attempts use.
				for _, v := range []string{p.OptimalTime, p.OptimalSpace} {
					if got, err := NormalizeComplexity(v); err != nil || got == "" || got != v {
						t.Errorf("%s: optimal complexity %q is not normalised (%q, %v)", p.Slug, v, got, err)
					}
				}
				if p.OptimalNote != strings.TrimSpace(p.OptimalNote) || len([]rune(p.OptimalNote)) > 160 {
					t.Errorf("%s: note must be trimmed and at most 160 characters", p.Slug)
				}
			}
		}
	}
	if count != 300 {
		t.Fatalf("checked %d problems, want 300", count)
	}
}

func TestOptimalComplexityDisplay(t *testing.T) {
	problem, _ := FindProblem("group-anagrams")
	render := func(state State) string {
		var out bytes.Buffer
		if err := ProblemHistory(problem, state, AttemptForm{}, false).Render(context.Background(), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	before := render(State{})
	if strings.Contains(before, "O(n·k)") || !strings.Contains(before, "appears after your first attempt") {
		t.Fatal("history page must hide the optimal complexity before the first attempt")
	}
	after := render(State{Attempts: []Attempt{attempt("group-anagrams", "unfinished", 10, false, time.Now())}})
	for _, want := range []string{"Optimal time", "Optimal space", "O(n·k)", "n strings of length up to k"} {
		if !strings.Contains(after, want) {
			t.Errorf("history page after an attempt missing %q", want)
		}
	}
	if strings.Contains(after, "appears after your first attempt") {
		t.Error("history page still shows the pre-attempt note")
	}

	rows := ProblemRows(State{Attempts: []Attempt{attempt("group-anagrams", "solved", 20, false, time.Now())}})
	var out bytes.Buffer
	if err := Problems(ProblemsPage{Rows: rows, Total: len(rows), Location: time.UTC}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`<th scope="col">Optimal time / space</th>`, "O(n·k)", "Shown after your first attempt"} {
		if !strings.Contains(html, want) {
			t.Errorf("problems page missing %q", want)
		}
	}
	// Only group-anagrams is attempted, so no other problem's values appear.
	if strings.Count(html, `<abbr title="Time"`) != 1 {
		t.Errorf("problems page shows optimal values for %d problems, want 1", strings.Count(html, `<abbr title="Time"`))
	}
}
