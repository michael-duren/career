package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestStatsHeadlineAndDifficultyBars(t *testing.T) {
	page := StatsPage{
		Progress:     Progress{Attempted: 5, Solved: 4, Independent: 3},
		Streaks:      Streaks{Current: 2, Longest: 6},
		DueNow:       1,
		Calendar:     [][]CalendarDay{{{Met: true, Active: true}, {Active: true}, {}, {Met: true}}},
		Difficulties: []DifficultyStat{{"Easy", 4, 3}, {"Medium", 2, 0}, {"Hard", 0, 0}},
	}
	h := page.Headline()
	if h.Solved != 4 || h.Attempted != 5 || h.Independent != "75%" || h.GoalMet != 2 || h.Practised != 3 || h.Due != 1 || h.Streak.Longest != 6 {
		t.Fatalf("headline %+v", h)
	}
	if (StatsPage{}).Headline().Independent != "—" {
		t.Fatal("independent rate before any solve")
	}
	bars := page.DifficultyBars()
	// Bars scale to the most attempted: Easy is full width, 3 solved of 4.
	if bars[0].SolvedWidth != "75.0" || bars[0].OpenWidth != "25.0" || bars[1].SolvedWidth != "0" || bars[1].OpenWidth != "50.0" || bars[2].OpenWidth != "0" {
		t.Fatalf("bars %+v", bars)
	}
}

func TestSolvedChart(t *testing.T) {
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	week := func(i, solved int) TrendWeek { return TrendWeek{Start: start.AddDate(0, 0, 7*i), Solved: solved} }
	tr := Trends{Weeks: []TrendWeek{week(0, 3), week(1, 0), week(2, 3), week(3, 1)}, MaxSolved: 3}
	c := tr.SolvedChart()
	if len(c.Bars) != 4 || c.Bars[1].Path != "" || c.Bars[0].Path == "" {
		t.Fatalf("bars %+v", c.Bars)
	}
	// The axis rounds 3 up to 4, with a middle line at 2.
	if len(c.Ticks) != 3 || c.Ticks[1].Label != "2" || c.Ticks[2].Label != "4" || c.Ticks[0].Y != ChartBaseY()+".0" {
		t.Fatalf("ticks %+v", c.Ticks)
	}
	// Only the first tallest week and the latest are labelled.
	for i, want := range []bool{true, false, false, true} {
		if c.Bars[i].Labelled != want {
			t.Errorf("bar %d labelled %v", i, c.Bars[i].Labelled)
		}
	}
	if c.Bars[3].Title != "Week of 28 Sep: 1 problem solved" || c.Bars[1].Title != "Week of 14 Sep: 0 problems solved" {
		t.Fatalf("titles %q %q", c.Bars[3].Title, c.Bars[1].Title)
	}
	if one := (Trends{Weeks: []TrendWeek{week(0, 0)}}).SolvedChart(); len(one.Ticks) != 2 || one.Ticks[1].Label != "1" || one.Bars[0].Labelled {
		t.Fatalf("empty week chart %+v", one)
	}
}

func TestRateChartBreaksAtMissingWeeks(t *testing.T) {
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	week := func(i int) time.Time { return start.AddDate(0, 0, 7*i) }
	tr := Trends{Weeks: []TrendWeek{
		{Start: week(0), Solves: 2, Independent: 2},
		{Start: week(1), Solves: 2, Independent: 1},
		{Start: week(2)},
		{Start: week(3), Solves: 1, Checked: 2, Matched: 1},
	}}
	c := tr.RateChart()
	independent, accuracy := c.Series[0], c.Series[1]
	// Consecutive weeks join; the empty week breaks the line. 100% is the
	// top of the plot and 0% the baseline.
	x := func(i int) string { return num(centre(i, 4)) }
	top, mid, base := ChartTop()+".0", num(chartTop+chartPlotH/2.0), ChartBaseY()+".0"
	if want := "M" + x(0) + "," + top + "L" + x(1) + "," + mid + "M" + x(3) + "," + base; independent.Path != want {
		t.Fatalf("independent path %q, want %q", independent.Path, want)
	}
	if len(accuracy.Points) != 1 || accuracy.EndLabel != "50%" || independent.EndLabel != "0%" {
		t.Fatalf("series %+v %+v", independent, accuracy)
	}
	// The lower line ends on the axis: its label stays above the point,
	// clear of the week labels.
	if independent.EndDy != "-9" || accuracy.EndDy != "-9" {
		t.Fatalf("end labels %s %s", independent.EndDy, accuracy.EndDy)
	}
	if !strings.Contains(c.Hits[2].Title, "Independent solves: none") || !strings.Contains(c.Hits[3].Title, "Complexity checks right: 50% (1 of 2 checks)") {
		t.Fatalf("hover titles %q %q", c.Hits[2].Title, c.Hits[3].Title)
	}
	// With room below, the lower line's label moves under its point.
	tr.Weeks[3].Independent = 1
	tr.Weeks[3].Solves = 4
	if c = tr.RateChart(); c.Series[0].EndDy != Count(endLabelBelow) || c.Series[1].EndDy != "-9" {
		t.Fatalf("lower line label %s, upper %s", c.Series[0].EndDy, c.Series[1].EndDy)
	}
	if c.Empty() || !(Trends{Weeks: []TrendWeek{{Start: start}}}).RateChart().Empty() {
		t.Fatal("Empty")
	}
}

func TestWeekLabelsThinAndKeepTheLatest(t *testing.T) {
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	var weeks []TrendWeek
	for i := 0; i < 7; i++ {
		weeks = append(weeks, TrendWeek{Start: start.AddDate(0, 0, 7*i)})
	}
	var got []string
	for _, l := range (Trends{Weeks: weeks}).SolvedChart().Labels {
		got = append(got, l.Label)
	}
	if want := "7 Sep 21 Sep 5 Oct 19 Oct"; strings.Join(got, " ") != want {
		t.Fatalf("labels %v, want %s", got, want)
	}
	if n := len((Trends{Weeks: weeks[:6]}).RateChart().Labels); n != 6 {
		t.Fatalf("six weeks show %d labels, want all", n)
	}
}

func TestStatsPageCharts(t *testing.T) {
	now := time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	state := State{Attempts: []Attempt{{ID: "a", ProblemSlug: "two-sum", Outcome: "solved", Minutes: 10, CreatedAt: now.Add(-time.Hour)}}, Problems: map[string]Problem{"two-sum": {Slug: "two-sum", Difficulty: "Easy", Topics: []string{"array"}}}}
	var out bytes.Buffer
	if err := Stats(NewStatsPage(NewToday(settings, state, now), StatsFilter{All: true}, 12)).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{
		`class="stat-tiles"`, "Problems solved", "of 1 attempted",
		`aria-label="Average recall and struggle rate by topic"`, "Array: struggle rate 0% of 1 attempt",
		"Show topics as a table", "Easy: 1 solved of 1 attempted",
		"Problems solved per week", "Week of 12 Oct: 1 problem solved", `class="chart-bar"`,
		"Independent solves: 100% (1 of 1 solve)", "Show weekly trends as a table",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("stats page missing %q", want)
		}
	}
}

func TestStatsPageWithoutRates(t *testing.T) {
	now := time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	state := State{Attempts: []Attempt{{ID: "a", ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 10, CreatedAt: now.Add(-time.Hour)}}}
	var out bytes.Buffer
	if err := Stats(NewStatsPage(NewToday(settings, state, now), StatsFilter{}, 12)).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if html := out.String(); !strings.Contains(html, "Rates appear after your first solve") || strings.Contains(html, `class="chart-line"`) {
		t.Fatal("unfinished-only week should explain the missing rates")
	}
}
