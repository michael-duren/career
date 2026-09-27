package leetgrinder

import (
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
