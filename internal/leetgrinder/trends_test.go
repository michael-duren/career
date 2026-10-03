package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestWeeklyTrends(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	settings := DefaultSettings()
	settings.Timezone = "America/Chicago"
	// Wednesday 14 Oct 2026, 12:00 local; its week starts Monday 12 Oct.
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, loc)
	yes, no := true, false
	at := func(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, loc) }
	attempts := []Attempt{
		// This week: two solves of one problem (one problem solved), one
		// with help; a Hard struggle.
		{ID: "a1", ProblemSlug: "two-sum", Outcome: "solved", Minutes: 10, CreatedAt: at(2026, 10, 13, 9)},
		{ID: "a2", ProblemSlug: "two-sum", Outcome: "solved", Minutes: 21, Assisted: true, CreatedAt: at(2026, 10, 12, 9)},
		{ID: "a3", ProblemSlug: "median-of-two-sorted-arrays", Outcome: "struggled", Minutes: 60, CreatedAt: at(2026, 10, 12, 10)},
		// Sunday 11 Oct, 23:30 local, is already Monday in UTC: it stays in
		// the previous week.
		// Stored as UTC, as loaded from the database.
		{ID: "a4", ProblemSlug: "valid-anagram", Outcome: "solved", Minutes: 15, CreatedAt: time.Date(2026, 10, 12, 4, 30, 0, 0, time.UTC)},
		// The best week is before the last: two problems solved.
		{ID: "a6", ProblemSlug: "climbing-stairs", Outcome: "solved", Minutes: 9, CreatedAt: at(2026, 9, 29, 9)},
		{ID: "a7", ProblemSlug: "house-robber", Outcome: "solved", Minutes: 11, CreatedAt: at(2026, 9, 30, 9)},
		// The first attempt, three weeks back.
		{ID: "a5", ProblemSlug: "lru-cache", Outcome: "unfinished", Minutes: 30, CreatedAt: at(2026, 9, 23, 9)},
	}
	state := State{
		Attempts: attempts,
		Problems: map[string]Problem{
			"two-sum":                     {Slug: "two-sum", Difficulty: "Easy"},
			"median-of-two-sorted-arrays": {Slug: "median-of-two-sorted-arrays", Difficulty: "Hard"},
			"valid-anagram":               {Slug: "valid-anagram", Difficulty: "Easy"},
			"lru-cache":                   {Slug: "lru-cache", Difficulty: "Medium"},
		},
		Analyses: map[string]Analysis{
			"a1": {AttemptID: "a1", Status: AnalysisDone, Current: true, TimeMatches: &yes, SpaceMatches: &no},
			// Pending, stale, or unstated values do not count.
			"a2": {AttemptID: "a2", Status: AnalysisPending, Current: true, TimeMatches: &no},
			"a3": {AttemptID: "a3", Status: AnalysisDone, Current: false, TimeMatches: &no},
			"a4": {AttemptID: "a4", Status: AnalysisDone, Current: true, TimeMatches: &yes},
		},
	}
	trends := WeeklyTrends(NewToday(settings, state, now), 12)
	// Weeks start with the first attempt's week, not 12 weeks back.
	var starts []string
	for _, w := range trends.Weeks {
		starts = append(starts, w.Start.Format(time.DateOnly))
	}
	if want := "2026-09-21 2026-09-28 2026-10-05 2026-10-12"; strings.Join(starts, " ") != want {
		t.Fatalf("weeks %v, want %s", starts, want)
	}
	this, last, first := trends.Weeks[3], trends.Weeks[2], trends.Weeks[0]
	if this.Solved != 1 || this.Solves != 2 || this.Independent != 1 || last.Solved != 1 || first.Solved != 0 || trends.Weeks[1].Solved != 2 || trends.MaxSolved != 2 || trends.Limit != 12 {
		t.Fatalf("solves: this %+v last %+v first %+v max %d", this, last, first, trends.MaxSolved)
	}
	if rate, ok := this.IndependentRate(); !ok || rate != 0.5 {
		t.Fatalf("independent rate %v %v", rate, ok)
	}
	if _, ok := first.IndependentRate(); ok {
		t.Fatal("rate for a week without solves")
	}
	if m, ok := this.AverageMinutes("Easy"); !ok || m != 16 {
		t.Fatalf("easy average %d %v", m, ok)
	}
	if m, ok := this.AverageMinutes("Hard"); !ok || m != 60 {
		t.Fatalf("hard average %d %v", m, ok)
	}
	if _, ok := this.AverageMinutes("Medium"); ok {
		t.Fatal("medium average without medium attempts")
	}
	if m, ok := first.AverageMinutes("Medium"); !ok || m != 30 {
		t.Fatalf("first week medium %d %v", m, ok)
	}
	if acc, ok := this.Accuracy(); !ok || acc != 0.5 || this.Checked != 2 {
		t.Fatalf("accuracy %v %v checked %d", acc, ok, this.Checked)
	}
	if acc, ok := last.Accuracy(); !ok || acc != 1 {
		t.Fatalf("last week accuracy %v %v", acc, ok)
	}
	// An account older than the window shows exactly the window.
	old := state
	old.Attempts = append(append([]Attempt{}, attempts...), Attempt{ID: "a0", ProblemSlug: "lru-cache", Outcome: "unfinished", Minutes: 5, CreatedAt: at(2025, 1, 6, 9)})
	got := WeeklyTrends(NewToday(settings, old, now), 12)
	if len(got.Weeks) != 12 || got.Weeks[11].Start.Format(time.DateOnly) != "2026-10-12" {
		t.Fatalf("old account: %d weeks", len(got.Weeks))
	}
	// The attempt before the window is skipped, not counted in a week shown.
	for _, w := range got.Weeks {
		if m, ok := w.AverageMinutes("Medium"); ok && m != 30 {
			t.Fatalf("week of %s counts the attempt before the window: %d", w.Start.Format(time.DateOnly), m)
		}
	}
	if got := WeeklyTrends(NewToday(settings, State{}, now), 12); len(got.Weeks) != 0 {
		t.Fatalf("no attempts: %d weeks", len(got.Weeks))
	}
}

func TestBarWidth(t *testing.T) {
	for _, c := range []struct {
		v, most float64
		want    string
	}{{0, 5, "0"}, {3, 0, "0"}, {5, 5, "100.0"}, {1, 4, "25.0"}, {0.001, 1, "1.0"}, {9, 5, "100.0"}} {
		if got := BarWidth(c.v, c.most); got != c.want {
			t.Errorf("BarWidth(%v, %v) = %s, want %s", c.v, c.most, got, c.want)
		}
	}
}

func TestTrendsSectionRenders(t *testing.T) {
	var out bytes.Buffer
	// Bars scale to the best week, not each week; rates differ per column.
	small := TrendWeek{Start: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Solved: 1, Solves: 1, Independent: 1, Minutes: map[string]int{}, Timed: map[string]int{}}
	w := TrendWeek{Start: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), Solved: 2, Solves: 4, Independent: 3, Minutes: map[string]int{"Easy": 20}, Timed: map[string]int{"Easy": 2}, Checked: 5, Matched: 2}
	if err := trendsSection(Trends{Weeks: []TrendWeek{small, w}, MaxSolved: 2, Limit: 12}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"Weekly trends", "at most the last 12 weeks", "12 Oct", "75% of 4", "40% of 5", "10 min", `width="50.0"`, `width="100.0"`, `width="75.0"`, `width="40.0"`, "—"} {
		if !strings.Contains(html, want) {
			t.Errorf("trends missing %q", want)
		}
	}
	out.Reset()
	if err := trendsSection(Trends{}).Render(context.Background(), &out); err != nil || !strings.Contains(out.String(), "Trends appear after your first attempt.") {
		t.Fatalf("empty trends: %s %v", out.String(), err)
	}
}

func TestUpcomingReviews(t *testing.T) {
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	date := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	card := func(slug string, due time.Time, flagged bool) Card {
		c := Card{Problem: Problem{Slug: slug}, Due: due, Reviews: 1}
		if flagged {
			c.Flag = &Flag{}
		}
		return c
	}
	today := Today{
		Settings: settings,
		Date:     date,
		Now:      date.Add(12 * time.Hour),
		Cards: []Card{
			card("later-5", date.AddDate(0, 0, 5), false),
			card("flagged", date.Add(-time.Hour), true),
			card("later-2", date.AddDate(0, 0, 2).Add(time.Hour), false),
			// Today's pick ranks below the flagged card in DueCards order,
			// and a second pick was rescheduled by today's attempt.
			card("pick", date.Add(-time.Hour), false),
			card("pick-done", date.AddDate(0, 0, 4), false),
			card("later-1", date.AddDate(0, 0, 1).Add(time.Hour), false),
		},
		Reviews: []ReviewItem{{Slot: 1, Problem: Problem{Slug: "pick"}}, {Slot: 2, Problem: Problem{Slug: "pick-done"}, Done: true}},
	}
	slugs := func(cards []Card) string {
		var out []string
		for _, c := range cards {
			out = append(out, c.Problem.Slug)
		}
		return strings.Join(out, " ")
	}
	for days, want := range map[int]string{
		0: "pick pick-done flagged",
		3: "pick pick-done flagged later-1 later-2",
		5: "pick pick-done flagged later-1 later-2 later-5",
	} {
		if got := slugs(UpcomingReviews(today, days)); got != want {
			t.Errorf("days %d: %s, want %s", days, got, want)
		}
	}
}
