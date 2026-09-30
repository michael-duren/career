package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAttemptDraftEscapesAndRetainsInput(t *testing.T) {
	problem := Problem{Number: 1, Slug: "two-sum", Title: "Two Sum"}
	form := AttemptForm{ID: uuid.NewString(), Revision: uuid.NewString(), Outcome: "struggled", Minutes: "26", Assisted: true, Notes: "<script>alert(1)</script>", Error: "Please retry"}
	var out bytes.Buffer
	if err := ProblemHistory(problem, State{}, form, AnalysisAvailability{}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`value="26"`, `value="struggled" selected`, `checked`, `&lt;script&gt;`, `Please retry`, form.ID, form.Revision, "Save correction"} {
		if !strings.Contains(html, want) {
			t.Errorf("draft missing %q", want)
		}
	}
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatal("unescaped draft")
	}
}

func TestProblemHistoryForAnyProblem(t *testing.T) {
	render := func(p Problem, state State) string {
		var out bytes.Buffer
		if err := ProblemHistory(p, state, NewForm(uuid.NewString()), AnalysisAvailability{}).Render(context.Background(), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	// An unknown slug shows its slug and waits for metadata.
	html := render(Problem{Slug: "some-new-problem"}, State{})
	for _, want := range []string{"<h1>some-new-problem</h1>", "Details load from LeetCode after you log an attempt.", `action="/leetgrinder/problem/some-new-problem/attempts"`, "No attempts recorded yet.", "appears after your first attempt"} {
		if !strings.Contains(html, want) {
			t.Errorf("unknown problem page missing %q", want)
		}
	}
	if html = render(Problem{Slug: "queued", InCatalog: true}, State{}); !strings.Contains(html, "Fetching details from LeetCode…") {
		t.Error("catalogued problem does not say it is fetching")
	}
	if html = render(Problem{Slug: "gone", InCatalog: true, NotFound: true}, State{}); !strings.Contains(html, "Not found on LeetCode") || strings.Contains(html, "Fetching details") {
		t.Error("not-found problem not marked")
	}
	if html = render(Problem{Slug: "given-up", InCatalog: true, FetchAttempts: MaxFetchAttempts}, State{}); strings.Contains(html, "Fetching details") {
		t.Error("gave-up fetch still says fetching")
	}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	tried := State{Attempts: []Attempt{attempt("ransom-note", "solved", 10, false, now)}}
	html = render(testProblems["ransom-note"], tried)
	for _, want := range []string{"LEETCODE 383", "Ransom Note", "O(m + n)", "Claude's estimate"} {
		if !strings.Contains(html, want) {
			t.Errorf("model optimum page missing %q", want)
		}
	}
	html = render(testProblems["two-sum"], State{Attempts: []Attempt{attempt("two-sum", "solved", 10, false, now)}})
	for _, want := range []string{"Best-known bounds", `href="/leetgrinder/problems?topic=hash-table"`, "Hash Table"} {
		if !strings.Contains(html, want) {
			t.Errorf("curated optimum page missing %q", want)
		}
	}
	if strings.Contains(html, "Claude's estimate") {
		t.Error("curated optimum labelled as an estimate")
	}
	if html = render(Problem{Slug: "x", Title: "X"}, State{Attempts: []Attempt{attempt("x", "solved", 10, false, now)}}); !strings.Contains(html, "Not known yet") {
		t.Error("missing optimum not explained")
	}
}

func TestRetiredPage(t *testing.T) {
	var out bytes.Buffer
	if err := Retired().Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `href="/leetgrinder"`) || !strings.Contains(out.String(), "retired") {
		t.Fatal("retired page lacks dashboard link")
	}
}
