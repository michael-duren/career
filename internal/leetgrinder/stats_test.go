package leetgrinder

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestStatsPage(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) // a Wednesday
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	problems := map[string]Problem{}
	for slug, p := range testProblems {
		problems[slug] = p
	}
	problems["three-sum"] = Problem{Slug: "three-sum", Title: "3Sum", Difficulty: "Medium", Topics: []string{"array", "two-pointers"}}
	state := State{Problems: problems, Attempts: []Attempt{
		attempt("two-sum", "solved", 10, false, now.AddDate(0, 0, -1)),
		attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -20)),
		attempt("binary-search", "unfinished", 25, false, now.AddDate(0, 0, -20)),
		attempt("three-sum", "solved", 10, true, now.AddDate(0, 0, -2)),
		attempt("mystery", "unfinished", 25, false, now.AddDate(0, 0, -3)),
	}}
	today := NewToday(settings, state, now)
	page := NewStatsPage(today, ParseStatsFilter(url.Values{}), 12)
	byKey := map[string]TopicStat{}
	for _, s := range page.Topics {
		byKey[s.Key] = s
	}
	array := byKey["array"]
	if array.Problems != 3 || array.Solved != 2 || array.Attempts != 4 || array.Struggles != 3 || array.StruggleRate() != 0.75 || array.Due != 2 {
		t.Fatalf("array: %+v", array)
	}
	// hash-table, binary-search and two-pointers have fewer than 3 problems.
	if _, ok := byKey["hash-table"]; ok || byKey["other"].Problems != 3 || byKey["untagged"].Problems != 1 {
		t.Fatalf("grouping: %+v", page.Topics)
	}
	// Weakest recall first.
	for i := 1; i < len(page.Topics); i++ {
		if page.Topics[i-1].Recall() > page.Topics[i].Recall() {
			t.Fatalf("not sorted by recall: %+v", page.Topics)
		}
	}
	all := NewStatsPage(today, ParseStatsFilter(url.Values{"all": {"1"}, "sort": {"problems"}}), 12)
	if all.Topics[0].Key != "array" || len(all.Topics) != 5 {
		t.Fatalf("all topics: %+v", all.Topics)
	}
	if d := page.Difficulties; len(d) != 4 || d[0].Attempted != 2 || d[0].Solved != 1 || d[1].Attempted != 1 || d[1].Solved != 1 || d[2].Attempted != 0 || d[3].Difficulty != "" || d[3].Attempted != 1 {
		t.Fatalf("difficulties: %+v", d)
	}
	// Three problems are due: two array problems and the untagged one.
	if page.DueToday != 3 || page.Streaks != today.Streaks || page.Progress != Summarize(state) {
		t.Fatalf("tiles: due %d streaks %+v progress %+v", page.DueToday, page.Streaks, page.Progress)
	}
	cal := page.Calendar
	if len(cal) != 12 || cal[11][2].Date != Date(now, time.UTC) || !cal[11][3].Date.IsZero() || cal[11][0].Date.Weekday() != time.Monday {
		t.Fatalf("calendar last week: %+v", cal[11])
	}
	if !cal[11][1].Active || CalendarClass(cal[11][3]) != "cal-day cal-future" {
		t.Fatalf("calendar cells: %+v", cal[11])
	}
	var out bytes.Buffer
	if err := Stats(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<a href="/leetgrinder/stats" aria-current="page">`, "Array", "75%", "Other", "Untagged", "Unknown", `class="goal-calendar"`, `href="/leetgrinder/problems?topic=array"`, "The last 12 weeks."} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("stats page missing %q", want)
		}
	}
	out.Reset()
	if err := Stats(NewStatsPage(NewToday(settings, State{}, now), StatsFilter{Sort: "recall"}, 12)).Render(context.Background(), &out); err != nil || !strings.Contains(out.String(), "Log attempts to see") {
		t.Fatalf("empty stats: %v", err)
	}
}

func TestStatsCalendarUsesLocalDate(t *testing.T) {
	settings := DefaultSettings()
	settings.Timezone = "America/Los_Angeles"
	loc := settings.Location()
	// 23:30 Sunday 18 Oct in Los Angeles is already Monday in UTC.
	now := time.Date(2026, 10, 18, 23, 30, 0, 0, loc)
	state := State{Attempts: []Attempt{attempt("two-sum", "solved", 10, false, now.Add(-time.Hour))}}
	cal := NewStatsPage(NewToday(settings, state, now), StatsFilter{Sort: "recall"}, 2).Calendar
	last := cal[1]
	if last[0].Date.Weekday() != time.Monday || !last[6].Date.Equal(time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)) || !last[6].Active {
		t.Fatalf("last week %+v", last)
	}
}

func TestStatsPageShowsTrendsBetweenMixAndCalendar(t *testing.T) {
	now := time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	state := State{Attempts: []Attempt{{ID: "a", ProblemSlug: "two-sum", Outcome: "solved", Minutes: 10, CreatedAt: now.Add(-time.Hour)}}, Problems: map[string]Problem{"two-sum": {Slug: "two-sum", Difficulty: "Easy"}}}
	var out bytes.Buffer
	if err := Stats(NewStatsPage(NewToday(settings, state, now), StatsFilter{}, 12)).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	mix, trends, calendar := strings.Index(html, "Difficulty mix"), strings.Index(html, "Weekly trends"), strings.Index(html, "Goal calendar")
	if mix < 0 || trends < mix || calendar < trends || !strings.Contains(html, "100% of 1") {
		t.Fatalf("order mix %d trends %d calendar %d", mix, trends, calendar)
	}
}
