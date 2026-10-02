package leetgrinder

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func dfsProblem() Problem {
	return Problem{Slug: "max-depth", Number: 104, Title: "Maximum Depth", Difficulty: "Easy", Topics: []string{"depth-first-search", "tree"},
		TopicNames: map[string]string{"depth-first-search": "Depth-First Search"}}
}

func TestStoredTopicNamesDriveLabels(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	state := State{Problems: map[string]Problem{"max-depth": dfsProblem()}, Attempts: []Attempt{attempt("max-depth", "solved", 10, false, now)}}
	rows := ProblemRows(state, BuildCards(state, time.UTC), now, time.UTC)

	// Filter options use the stored name, and fall back for unnamed tags.
	labels := map[string]string{}
	for _, o := range ProblemTopics(rows) {
		labels[o.Key] = o.Label
	}
	if labels["depth-first-search"] != "Depth-First Search" || labels["tree"] != "Tree" {
		t.Fatalf("options: %v", labels)
	}

	// Search matches the stored name, including its hyphens.
	for query, want := range map[string]int{"depth-first": 1, "Depth-First Search": 1, "depth first": 0, "tree": 1} {
		q, _ := url.ParseQuery("q=" + url.QueryEscape(query))
		if got := FilterProblems(rows, ParseProblemFilter(q)); len(got) != want {
			t.Errorf("search %q: %d rows, want %d", query, len(got), want)
		}
	}

	// The problems page renders it.
	var out bytes.Buffer
	filter := ParseProblemFilter(url.Values{})
	page := ProblemsPage{Rows: FilterProblems(rows, filter), Total: len(rows), Topics: ProblemTopics(rows), Filter: filter, Location: time.UTC}
	if err := Problems(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if html := out.String(); !strings.Contains(html, "Depth-First Search") || strings.Contains(html, "Depth First Search") {
		t.Error("problems page does not use the stored topic name")
	}

	// Stats label topics by name.
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	stats := NewStatsPage(NewToday(settings, state, now), ParseStatsFilter(url.Values{"all": {"1"}}), 12)
	found := false
	for _, s := range stats.Topics {
		if s.Key == "depth-first-search" {
			found = s.Label == "Depth-First Search"
		}
	}
	if !found {
		t.Fatalf("stats labels: %+v", stats.Topics)
	}

	// Todo set summaries use it too.
	sum := TodoSet{Items: []TodoItem{{Problem: dfsProblem()}}}.Summary()
	if len(sum.Topics) != 2 || sum.Topics[0].Label != "Depth-First Search" && sum.Topics[1].Label != "Depth-First Search" {
		t.Fatalf("todo summary: %+v", sum.Topics)
	}
}
