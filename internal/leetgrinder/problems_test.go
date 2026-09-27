package leetgrinder

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProblemRowsAndFilters(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	state := State{Attempts: []Attempt{
		attempt("ransom-note", "solved", 20, true, now),
		attempt("two-sum", "solved", 15, false, now.AddDate(0, 0, -1)),
		attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -3)),
	}}
	rows := ProblemRows(state)
	if len(rows) != 300 || rows[0].Problem.Slug != "two-sum" || rows[0].Order != 1 || rows[0].Attempts != 2 || !rows[0].Last.CreatedAt.Equal(now.AddDate(0, 0, -1)) {
		t.Fatalf("rows: %d %+v", len(rows), rows[0])
	}
	optional := 0
	for _, r := range rows {
		if r.Optional {
			optional++
		}
	}
	if optional != 48 {
		t.Fatalf("optional %d", optional)
	}

	filter := func(query string) []ProblemRow {
		q, _ := url.ParseQuery(query)
		return FilterProblems(rows, ParseProblemFilter(q))
	}
	for query, want := range map[string]int{
		"":                         300,
		"difficulty=Hard":          20,
		"difficulty=Easy":          78,
		"kind=optional":            48,
		"kind=core":                252,
		"week=1&kind=core":         21,
		"status=solved":            1,
		"status=solved-help":       1,
		"status=not-attempted":     298,
		"q=ransom+NOTE":            1,
		"difficulty=bogus&week=99": 300,
	} {
		if got := len(filter(query)); got != want {
			t.Errorf("%q: %d, want %d", query, got, want)
		}
	}
	if got := filter("q=TWO+SUM"); len(got) < 2 || got[0].Problem.Slug != "two-sum" {
		t.Fatal("case-insensitive title search")
	}
	if got := filter("q=1"); len(got) == 0 {
		t.Fatal("number search")
	}
	if got := filter("q=binary+search&week=2"); len(got) == 0 || got[0].Week.Number != 2 {
		t.Fatal("topic search")
	}
	if got := filter("sort=recent"); got[0].Problem.Slug != "ransom-note" || got[1].Problem.Slug != "two-sum" || got[2].Order != 2 {
		t.Fatalf("recent sort: %s %s", got[0].Problem.Slug, got[1].Problem.Slug)
	}
	if got := filter("sort=difficulty"); got[0].Problem.Difficulty != "Easy" || got[299].Problem.Difficulty != "Hard" {
		t.Fatal("difficulty sort")
	}
	if got := filter("sort=number"); got[0].Problem.ID > got[1].Problem.ID {
		t.Fatal("number sort")
	}
	long, _ := url.ParseQuery("q=" + strings.Repeat("é", 150))
	if f := ParseProblemFilter(long); len([]rune(f.Query)) != 100 || !strings.HasSuffix(f.Query, "é") {
		t.Fatal("query truncation split a character")
	}
}

func TestProblemsPageRenders(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	rows := ProblemRows(State{Attempts: []Attempt{attempt("two-sum", "struggled", 30, false, now)}})
	q, _ := url.ParseQuery("difficulty=Easy&week=1&status=struggled")
	filter := ParseProblemFilter(q)
	var out bytes.Buffer
	page := ProblemsPage{Rows: FilterProblems(rows, filter), Total: len(rows), Filter: filter, Location: time.UTC}
	if err := Problems(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"Showing 1 of 300 problems", `href="/leetgrinder/problem/two-sum"`, `href="https://leetcode.com/problems/two-sum/"`, `value="Easy" selected`, `value="1" selected`, `value="struggled" selected`, "Clear filters", "status-struggled", "Tue 20 Oct 2026", `<a href="/leetgrinder/problems" aria-current="page">`} {
		if !strings.Contains(html, want) {
			t.Errorf("problems page missing %q", want)
		}
	}
	out.Reset()
	none := ParseProblemFilter(url.Values{"q": {"no such problem"}})
	if err := Problems(ProblemsPage{Rows: FilterProblems(rows, none), Total: len(rows), Filter: none, Location: time.UTC}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No problems match these filters.") {
		t.Fatal("empty state missing")
	}
}
