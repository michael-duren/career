package leetgrinder

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func problemFixture(now time.Time) (State, []ProblemRow) {
	state := State{Problems: testProblems, Attempts: []Attempt{
		attempt("ransom-note", "solved", 20, true, now),
		attempt("two-sum", "solved", 15, false, now.AddDate(0, 0, -1)),
		attempt("mystery-problem", "unfinished", 25, false, now.AddDate(0, 0, -2)),
		attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -3)),
		attempt("binary-search", "struggled", 30, false, now.AddDate(0, 0, -40)),
	}}
	return state, ProblemRows(state, BuildCards(state, time.UTC), now, time.UTC)
}

func TestProblemRowsAndFilters(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	_, rows := problemFixture(now)
	if len(rows) != 4 || rows[0].Problem.Slug != "ransom-note" || rows[1].Problem.Slug != "two-sum" || rows[1].Attempts != 2 || !rows[1].Last.CreatedAt.Equal(now.AddDate(0, 0, -1)) || rows[1].Card.Reviews != 2 {
		t.Fatalf("rows: %d %+v", len(rows), rows)
	}
	filter := func(query string) []ProblemRow {
		q, _ := url.ParseQuery(query)
		return FilterProblems(rows, ParseProblemFilter(q))
	}
	for query, want := range map[string]int{
		"":                  4,
		"difficulty=Easy":   3,
		"difficulty=Hard":   0,
		"topic=hash-table":  1,
		"topic=untagged":    2,
		"topic=Bad Topic":   4,
		"status=solved":     2,
		"status=struggled":  1,
		"status=unfinished": 1,
		"status=due":        2,
		// Struggles flag: with help, unfinished, struggled; not the re-solve.
		"status=flagged":            3,
		"q=ransom+NOTE":             1,
		"q=hash":                    1,
		"q=mystery":                 1,
		"difficulty=bogus&sort=zzz": 4,
	} {
		if got := len(filter(query)); got != want {
			t.Errorf("%q: %d, want %d", query, got, want)
		}
	}
	if got := filter("q=1"); len(got) != 1 || got[0].Problem.Number != 1 {
		t.Fatalf("number search should match exactly: %d rows", len(got))
	}
	if got := filter("sort=number"); got[0].Problem.Number != 1 || got[len(got)-1].Problem.Slug != "mystery-problem" {
		t.Fatal("number sort should put unknown numbers last")
	}
	if got := filter("sort=title"); got[0].Problem.Slug != "binary-search" {
		t.Fatalf("title sort: %s", got[0].Problem.Slug)
	}
	if got := filter("sort=recall"); got[0].Recall > got[1].Recall {
		t.Fatal("recall sort")
	}
	if got := filter("sort=due"); got[0].Card.Due.After(got[1].Card.Due) {
		t.Fatal("due sort")
	}
	long, _ := url.ParseQuery("q=" + strings.Repeat("é", 150))
	if f := ParseProblemFilter(long); len([]rune(f.Query)) != 100 || !strings.HasSuffix(f.Query, "é") {
		t.Fatal("query truncation split a character")
	}
	topics := ProblemTopics(rows)
	if len(topics) != 4 || topics[0].Label != "Array" || topics[len(topics)-1].Key != "untagged" {
		t.Fatalf("topics: %+v", topics)
	}
}

func TestProblemsPageRenders(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	_, rows := problemFixture(now)
	q, _ := url.ParseQuery("difficulty=Easy&topic=hash-table&status=solved")
	filter := ParseProblemFilter(q)
	var out bytes.Buffer
	page := ProblemsPage{Rows: FilterProblems(rows, filter), Total: len(rows), Topics: ProblemTopics(rows), Filter: filter, Location: time.UTC}
	if err := Problems(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"Showing 1 of 4 problems", `href="/leetgrinder/problem/two-sum"`, `href="https://leetcode.com/problems/two-sum/"`, `value="Easy" selected`, `value="hash-table" selected`, `value="solved" selected`, "Clear filters", "status-solved", "19 Oct 2026", `<a href="/leetgrinder/problems" aria-current="page">`, "Hash Table"} {
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
	out.Reset()
	if err := Problems(ProblemsPage{Location: time.UTC, Filter: ParseProblemFilter(url.Values{})}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No attempts yet.") {
		t.Fatal("no-attempts state missing")
	}
}

// ProblemRows groups attempts once; each row must still agree with the
// per-slug State lookups it replaced.
func TestProblemRowsMatchStateLookups(t *testing.T) {
	settings, state, now := benchmarkState()
	loc := settings.Location()
	rows := ProblemRows(state, BuildCards(state, loc), now, loc)
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	seen := map[string]bool{}
	for i, r := range rows {
		slug := r.Problem.Slug
		if seen[slug] {
			t.Fatalf("%s listed twice", slug)
		}
		seen[slug] = true
		attempts := state.ProblemAttempts(slug)
		if r.Best != state.ProblemBest(slug) || r.Latest != state.ProblemLatest(slug) || r.Attempts != len(attempts) || r.Last.ID != attempts[0].ID {
			t.Fatalf("%s: row %q/%q/%d/%s, state %q/%q/%d/%s", slug, r.Best, r.Latest, r.Attempts, r.Last.ID, state.ProblemBest(slug), state.ProblemLatest(slug), len(attempts), attempts[0].ID)
		}
		if i > 0 && r.Last.CreatedAt.After(rows[i-1].Last.CreatedAt) {
			t.Fatalf("%s: rows not newest first", slug)
		}
	}
}

func TestTodayCardLookup(t *testing.T) {
	settings, state, now := benchmarkState()
	today := NewToday(settings, state, now)
	if len(today.Cards) == 0 {
		t.Fatal("no cards")
	}
	for _, c := range today.Cards {
		got, ok := today.Card(c.Problem.Slug)
		if !ok || got.Problem.Slug != c.Problem.Slug || got.Reviews != c.Reviews {
			t.Fatalf("%s: lookup %v %+v", c.Problem.Slug, ok, got)
		}
	}
	if _, ok := today.Card("no-such-problem"); ok {
		t.Fatal("unknown slug has a card")
	}
	// A Today built by hand has no index and still finds its cards.
	manual := Today{Cards: today.Cards}
	if c, ok := manual.Card(today.Cards[0].Problem.Slug); !ok || c.Reviews != today.Cards[0].Reviews {
		t.Fatal("unindexed lookup failed")
	}
}

func BenchmarkProblemRows(b *testing.B) {
	settings, state, now := benchmarkState()
	loc := settings.Location()
	cards := BuildCards(state, loc)
	b.ResetTimer()
	for range b.N {
		ProblemRows(state, cards, now, loc)
	}
}

// A problem solved long ago and failed since is still ever solved, but its
// latest result is what the struggle filters and the page show.
func TestBestAndLatestResults(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	for name, test := range map[string]struct {
		attempts     []Attempt // newest first
		best, latest string
	}{
		"none":                   {nil, ResultNone, ResultNone},
		"independent":            {[]Attempt{attempt("a", "solved", 10, false, now)}, ResultIndependent, ResultIndependent},
		"help only":              {[]Attempt{attempt("a", "solved", 10, true, now)}, ResultWithHelp, ResultWithHelp},
		"unfinished only":        {[]Attempt{attempt("a", "unfinished", 10, false, now)}, ResultNotSolved, ResultUnfinished},
		"struggled only":         {[]Attempt{attempt("a", "struggled", 10, false, now)}, ResultNotSolved, ResultStruggled},
		"solved then unfinished": {[]Attempt{attempt("a", "unfinished", 10, false, now), attempt("a", "solved", 10, false, now.AddDate(0, -7, 0))}, ResultIndependent, ResultUnfinished},
		"solved then struggled":  {[]Attempt{attempt("a", "struggled", 10, false, now), attempt("a", "solved", 10, false, now.AddDate(0, -7, 0))}, ResultIndependent, ResultStruggled},
		"help then independent":  {[]Attempt{attempt("a", "solved", 10, false, now), attempt("a", "solved", 10, true, now.AddDate(0, 0, -9))}, ResultIndependent, ResultIndependent},
		"independent then help":  {[]Attempt{attempt("a", "solved", 10, true, now), attempt("a", "solved", 10, false, now.AddDate(0, 0, -9))}, ResultIndependent, ResultWithHelp},
		"help then unfinished":   {[]Attempt{attempt("a", "unfinished", 10, false, now), attempt("a", "solved", 10, true, now.AddDate(0, 0, -9))}, ResultWithHelp, ResultUnfinished},
	} {
		if best, latest := bestOf(test.attempts), latestOf(test.attempts); best != test.best || latest != test.latest {
			t.Errorf("%s: best %q latest %q, want %q and %q", name, best, latest, test.best, test.latest)
		}
	}
}

func TestStatusFiltersUseLatestResult(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	state := State{Problems: testProblems, Attempts: []Attempt{
		attempt("two-sum", "unfinished", 25, false, now.AddDate(0, 0, -1)),
		attempt("two-sum", "solved", 10, false, now.AddDate(0, -7, 0)),
		attempt("ransom-note", "solved", 10, false, now.AddDate(0, 0, -2)),
		attempt("ransom-note", "struggled", 40, false, now.AddDate(0, -3, 0)),
	}}
	rows := ProblemRows(state, BuildCards(state, time.UTC), now, time.UTC)
	slugs := func(query string) []string {
		q, _ := url.ParseQuery(query)
		var out []string
		for _, r := range FilterProblems(rows, ParseProblemFilter(q)) {
			out = append(out, r.Problem.Slug)
		}
		return out
	}
	for query, want := range map[string]string{
		"status=unfinished":  "two-sum",
		"status=struggled":   "",
		"status=solved":      "two-sum,ransom-note",
		"status=solved-last": "ransom-note",
	} {
		if got := strings.Join(slugs(query), ","); got != want {
			t.Errorf("%q: %q, want %q", query, got, want)
		}
	}
	var twoSum ProblemRow
	for _, r := range rows {
		if r.Problem.Slug == "two-sum" {
			twoSum = r
		}
	}
	if twoSum.Best != ResultIndependent || twoSum.Latest != ResultUnfinished {
		t.Fatalf("two-sum row: best %q latest %q", twoSum.Best, twoSum.Latest)
	}

	var out bytes.Buffer
	page := ProblemsPage{Rows: rows, Total: len(rows), Location: time.UTC, Filter: ParseProblemFilter(url.Values{})}
	if err := Problems(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"Ever solved", "Solved last time", "Unfinished last time", `title="Latest result"`, "Best: <span class=\"status-pill status-solved\">Solved independently</span>"} {
		if !strings.Contains(html, want) {
			t.Errorf("problems page missing %q", want)
		}
	}
	// ransom-note best and latest agree, so only two-sum shows a best.
	if n := strings.Count(html, "Best: "); n != 1 {
		t.Errorf("%d Best labels, want 1", n)
	}
}
