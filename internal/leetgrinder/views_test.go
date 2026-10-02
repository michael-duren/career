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
	if err := ProblemHistory(problem, State{}, form, AnalysisAvailability{}, ProblemReview{}).Render(context.Background(), &out); err != nil {
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
		if err := ProblemHistory(p, state, NewForm(uuid.NewString()), AnalysisAvailability{}, ProblemReview{}).Render(context.Background(), &out); err != nil {
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
	if strings.Contains(html, ">Latest</span>") || strings.Contains(html, "Not attempted") {
		t.Error("unattempted problem shows a result")
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

func TestProblemHistoryReviewState(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	solve := attempt("two-sum", "solved", 10, false, now.AddDate(0, 0, -3))
	state := State{Problems: testProblems, Attempts: []Attempt{solve}, Analyses: map[string]Analysis{
		solve.ID: {Status: AnalysisDone, Current: true, SpaceMatches: boolPtr(false), UpdatedAt: now.AddDate(0, 0, -3)},
	}, Plans: map[time.Time][]string{Date(now, time.UTC): {"two-sum"}}}
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	review := NewProblemReview(NewToday(settings, state, now), "two-sum")
	if !review.Due || !review.Pick || review.FlagReason != "Space complexity judged wrong 3 days ago" {
		t.Fatalf("review %+v", review)
	}
	var out bytes.Buffer
	if err := ProblemHistory(testProblems["two-sum"], state, NewForm(uuid.NewString()), AnalysisAvailability{}, review).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Today's review", "Recall now", "Next due", "Sun 18 Oct 2026", "Space complexity judged wrong 3 days ago"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("history missing %q", want)
		}
	}
	rows := ProblemRows(state, BuildCards(state, time.UTC), now, time.UTC)
	if got := FilterProblems(rows, ParseProblemFilter(map[string][]string{"status": {"flagged"}})); len(got) != 1 {
		t.Fatalf("flagged filter: %d", len(got))
	}
}

func TestAttemptHistoryUsesSettingsZoneAndDayKind(t *testing.T) {
	settings := DefaultSettings()
	settings.Timezone = "America/Chicago"
	now := time.Date(2026, 10, 11, 4, 0, 0, 0, time.UTC)
	first := attempt("valid-anagram", "solved", 5, false, time.Date(2026, 10, 9, 15, 0, 0, 0, time.UTC))
	// 22:30 on 10 Oct in Chicago is already 11 Oct in UTC. The stored
	// is_review flag is stale: the problem was not due, so the day counts it
	// as practice.
	second := attempt("valid-anagram", "solved", 5, false, time.Date(2026, 10, 11, 3, 30, 0, 0, time.UTC))
	second.IsReview = true
	state := State{Problems: testProblems, Attempts: []Attempt{second, first}}
	review := NewProblemReview(NewToday(settings, state, now), "valid-anagram")
	if got := review.AttemptKind(second); got != "Practice" {
		t.Fatalf("kind %q", got)
	}
	if got := review.AttemptKind(first); got != "" {
		t.Fatalf("a new problem has no label, got %q", got)
	}
	var out bytes.Buffer
	if err := ProblemHistory(testProblems["valid-anagram"], state, NewForm(uuid.NewString()), AnalysisAvailability{}, review).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"10 Oct 2026 · 22:30 CDT", "· Practice", `datetime="2026-10-11T03:30:00Z"`} {
		if !strings.Contains(html, want) {
			t.Errorf("history missing %q", want)
		}
	}
	if strings.Contains(html, "· Review") || strings.Contains(html, "UTC</time>") {
		t.Error("history still uses stale review flag or UTC times")
	}
	if got := (ProblemReview{}).AttemptTime(second).Format("15:04 MST"); got != "03:30 UTC" {
		t.Fatalf("fallback %q", got)
	}
}

func TestProblemHistoryShowsLatestAndBest(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	render := func(state State) string {
		var out bytes.Buffer
		if err := ProblemHistory(testProblems["two-sum"], state, NewForm(uuid.NewString()), AnalysisAvailability{}, ProblemReview{}).Render(context.Background(), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	html := render(State{Attempts: []Attempt{attempt("two-sum", "unfinished", 25, false, now), attempt("two-sum", "solved", 10, false, now.AddDate(0, -7, 0))}})
	for _, want := range []string{`>Latest</span>`, `status-unfinished">Unfinished</span>`, `>Best</span>`, `status-solved">Solved independently</span>`} {
		if !strings.Contains(html, want) {
			t.Errorf("solved-then-unfinished page missing %q", want)
		}
	}
	html = render(State{Attempts: []Attempt{attempt("two-sum", "solved", 10, false, now)}})
	if !strings.Contains(html, `>Latest</span>`) || strings.Contains(html, `>Best</span>`) {
		t.Error("equal best and latest should show only the latest")
	}
}
