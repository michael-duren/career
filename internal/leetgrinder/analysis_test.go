package leetgrinder

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCleanAnalysisText(t *testing.T) {
	for in, want := range map[string]string{
		"  plain  ":            "plain",
		"a\tb\x00c\r\nd":       "a b c \nd",
		"line sep":             "line sep",
		"bad \xff utf8":        "bad � utf8",
		strings.Repeat("x", 5): "xxxxx",
	} {
		if got := CleanAnalysisText(in, 10); got != want {
			t.Errorf("CleanAnalysisText(%q) = %q, want %q", in, got, want)
		}
	}
	if got := CleanAnalysisText(strings.Repeat("é", 30), 10); utf8.RuneCountInString(got) != 10 || !strings.HasSuffix(got, "…") {
		t.Errorf("cap: %q", got)
	}
}

func TestAnalysisStatedWrong(t *testing.T) {
	yes, no := true, false
	done := Analysis{Status: AnalysisDone, Current: true, TimeMatches: &yes, SpaceMatches: &no}
	state := State{Analyses: map[string]Analysis{"a": done}}
	if !state.StatedWrong("a") || state.StatedWrong("missing") {
		t.Fatal("StatedWrong on state")
	}
	for name, a := range map[string]Analysis{
		"stale":     {Status: AnalysisDone, Current: false, TimeMatches: &no},
		"pending":   {Status: AnalysisPending, Current: true, TimeMatches: &no},
		"unstated":  {Status: AnalysisDone, Current: true},
		"all match": {Status: AnalysisDone, Current: true, TimeMatches: &yes, SpaceMatches: &yes},
	} {
		if a.StatedWrong() {
			t.Errorf("%s judged wrong", name)
		}
	}
	if (Attempt{Code: "x"}).Analysable() || !(Attempt{Code: "x", SpaceComplexity: "O(1)"}).Analysable() || (Attempt{TimeComplexity: "O(1)"}).Analysable() {
		t.Error("Analysable")
	}
}

func TestAnalysisCardUnknownStatus(t *testing.T) {
	problem, _ := FindProblem("two-sum")
	state := State{Attempts: []Attempt{{ID: "11111111-1111-4111-8111-111111111111", ProblemSlug: "two-sum", Outcome: "solved", Minutes: 5, TimeComplexity: "O(n)", Code: "pass", CodeLanguage: "python3"}}}
	var out strings.Builder
	if err := ProblemHistory(problem, state, AttemptForm{}, AnalysisAvailability{KeyConfigured: true, Unknown: true}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "analysis status could not be loaded") || strings.Contains(out.String(), "turned off") {
		t.Fatal("unknown status not explained")
	}
	// A daily limit of 0 never runs, so nothing promises tomorrow.
	out.Reset()
	if err := ProblemHistory(problem, state, AttemptForm{}, AnalysisAvailability{KeyConfigured: true, Enabled: true, ZeroLimit: true, LimitReached: true}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "daily request limit is 0") || strings.Contains(out.String(), "tomorrow") {
		t.Fatal("zero limit not explained")
	}
	if p := (AnalysisPanel{KeyConfigured: true, Enabled: true}); p.Active() || p.LimitReached() {
		t.Fatal("zero-limit panel reported active or reached")
	}
}
