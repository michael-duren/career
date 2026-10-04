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
		DueToday:     1,
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
	// Far-apart values sit level with their points: 0% at the baseline,
	// 50% halfway up.
	if independent.LabelY != base || accuracy.LabelY != mid {
		t.Fatalf("label heights %s %s", independent.LabelY, accuracy.LabelY)
	}
	if !strings.Contains(c.Hits[2].Title, "Independent solves: none") || !strings.Contains(c.Hits[3].Title, "Complexity checks right: 50% (1 of 2 checks)") {
		t.Fatalf("hover titles %q %q", c.Hits[2].Title, c.Hits[3].Title)
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
	for i := 7; i < 12; i++ {
		weeks = append(weeks, TrendWeek{Start: start.AddDate(0, 0, 7*i)})
	}
	// Twelve weeks label every other week back from the latest, so the
	// oldest has none.
	got = nil
	for _, l := range (Trends{Weeks: weeks}).SolvedChart().Labels {
		got = append(got, l.Label)
	}
	if want := "14 Sep 28 Sep 12 Oct 26 Oct 9 Nov 23 Nov"; strings.Join(got, " ") != want {
		t.Fatalf("twelve-week labels %v, want %s", got, want)
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
	if html := out.String(); !strings.Contains(html, "Rates appear after your first solve") || !strings.Contains(html, "Columns appear after your first solve.") || strings.Contains(html, `class="chart-line"`) || strings.Contains(html, `class="chart-bar"`) {
		t.Fatal("unfinished-only week should explain the missing rates")
	}
}

func TestSolvedChartGeometry(t *testing.T) {
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	tr := Trends{Weeks: []TrendWeek{{Start: start, Solved: 4}, {Start: start.AddDate(0, 0, 7), Solved: 2}}, MaxSolved: 4}
	c := tr.SolvedChart()
	// Two weeks share the plot: bands of 294 units, bars capped at 36.
	b := num(band(2))
	if c.Bars[0].HitX != ChartLeft()+".0" || c.Bars[0].HitW != b || c.Bars[1].HitX != num(chartLeft+band(2)) {
		t.Fatalf("hover bands %+v", c.Bars)
	}
	if end := chartLeft + 2*band(2); num(end) != ChartRight()+".0" {
		t.Fatalf("last band ends at %s, want %s", num(end), ChartRight())
	}
	// The tallest column reaches the top of the plot with a 4-unit rounded
	// top, and its label sits 5 units above it.
	x := centre(0, 2) - chartMaxBarW/2.0
	top := float64(chartTop)
	want := "M" + num(x) + "," + ChartBaseY() + ".0V" + num(top+4) + "Q" + num(x) + "," + num(top) + " " + num(x+4) + "," + num(top) + "H" + num(x+chartMaxBarW-4) + "Q" + num(x+chartMaxBarW) + "," + num(top) + " " + num(x+chartMaxBarW) + "," + num(top+4) + "V" + ChartBaseY() + ".0Z"
	if c.Bars[0].Path != want || c.Bars[0].LabelY != num(top-5) {
		t.Fatalf("tallest column %q label %s, want %q", c.Bars[0].Path, c.Bars[0].LabelY, want)
	}
	// A column at the middle tick's value tops out on that tick.
	if !strings.Contains(c.Bars[1].Path, "H") || !strings.Contains(c.Bars[1].Path, ","+c.Ticks[1].Y+" ") {
		t.Fatalf("middle column %q does not reach tick %s", c.Bars[1].Path, c.Ticks[1].Y)
	}
}

func TestRateChartSeparatesCloseEndLabels(t *testing.T) {
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	label := func(w TrendWeek) (string, string) {
		t.Helper()
		w.Start = start
		c := (Trends{Weeks: []TrendWeek{w}}).RateChart()
		return c.Series[0].LabelY, c.Series[1].LabelY
	}
	// 5% and 0%: pushed endLabelGap apart, then held at the baseline.
	if a, b := label(TrendWeek{Solves: 20, Independent: 1, Checked: 4}); a != "150.0" || b != "172.0" {
		t.Fatalf("5%% and 0%%: %s %s", a, b)
	}
	// Both at 100%: pushed apart, then held at the top of the plot.
	if a, b := label(TrendWeek{Solves: 1, Independent: 1, Checked: 1, Matched: 1}); a != "16.0" || b != "38.0" {
		t.Fatalf("both 100%%: %s %s", a, b)
	}
	// 60% and 50% in the middle spread evenly around their midpoint.
	if a, b := label(TrendWeek{Solves: 10, Independent: 6, Checked: 2, Matched: 1}); a != "75.2" || b != "97.2" {
		t.Fatalf("60%% and 50%%: %s %s", a, b)
	}
	// Only the latest week is labelled: a line with no value there has none.
	c := (Trends{Weeks: []TrendWeek{{Start: start, Checked: 1}, {Start: start.AddDate(0, 0, 7), Solves: 1}}}).RateChart()
	if c.Series[0].EndLabel != "0%" || c.Series[1].EndLabel != "" || len(c.Series[1].Points) != 1 {
		t.Fatalf("latest-week labels %+v", c.Series)
	}
}

func TestChartGeometryLiterals(t *testing.T) {
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	two := (Trends{Weeks: []TrendWeek{{Start: start, Solved: 1}, {Start: start.AddDate(0, 0, 7), Solved: 1}}, MaxSolved: 1}).SolvedChart()
	if two.Bars[0].HitW != "272.0" || two.Bars[1].HitX != "308.0" || !strings.HasPrefix(two.Bars[0].Path, "M154.0,172.0V20.0") {
		t.Fatalf("two weeks %+v", two.Bars)
	}
	one := (Trends{Weeks: []TrendWeek{{Start: start, Solves: 1, Independent: 1}}}).RateChart()
	if p := one.Series[0].Points[0]; p.X != "308.0" || p.Y != "16.0" {
		t.Fatalf("one week point %+v", p)
	}
}

func TestDifficultyBarsNeverPassFullWidth(t *testing.T) {
	bars := (StatsPage{Difficulties: []DifficultyStat{{"Medium", 400, 399}}}).DifficultyBars()
	if bars[0].SolvedWidth != "99.8" || bars[0].OpenWidth != "0.2" {
		t.Fatalf("bars %+v", bars[0])
	}
}

func TestStatsPageRendersChartValues(t *testing.T) {
	start := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	page := StatsPage{
		Progress:     Progress{Attempted: 5, Solved: 4, Independent: 3},
		Streaks:      Streaks{Current: 2, Longest: 6},
		DueToday:     1,
		Calendar:     [][]CalendarDay{{{Met: true, Active: true}, {Active: true}, {}, {Met: true}}},
		Difficulties: []DifficultyStat{{"Easy", 4, 3}, {"Medium", 2, 0}, {"", 1, 0}},
		Topics: []TopicStat{
			{Key: "array", Label: "Array", Problems: 4, Solved: 3, Attempts: 5, Struggles: 1, RecallSum: 2, Due: 2},
			{Key: "other", Label: "Other", Problems: 3, Solved: 1, Attempts: 3},
		},
		Trends: Trends{Weeks: []TrendWeek{
			{Start: start, Solved: 2, Solves: 2, Independent: 1, Checked: 2, Matched: 2},
			{Start: start.AddDate(0, 0, 7), Solved: 1, Solves: 1, Independent: 1, Checked: 1, Matched: 1},
		}, MaxSolved: 2, Limit: 12},
	}
	var out bytes.Buffer
	if err := Stats(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if strings.Contains(html, `topic=other`) {
		t.Error("Other links to a topic filter")
	}
	solved, rates, bars := page.Trends.SolvedChart(), page.Trends.RateChart(), page.DifficultyBars()
	for _, want := range []string{
		"<strong>4</strong> <span class=\"muted\">of 5 attempted", "<strong>75%</strong>", "longest 6", "<strong>2</strong> <span class=\"muted\">days of 3 practised", "<strong>1</strong> <span class=\"muted\">review ·",
		`d="` + solved.Bars[0].Path + `"`, `d="` + solved.Bars[1].Path + `"`,
		`d="` + rates.Series[0].Path + `"`,
		`y="` + rates.Series[0].LabelY + `" dominant-baseline="middle">100%</text>`,
		`y="` + rates.Series[1].LabelY + `" dominant-baseline="middle">100%</text>`,
		`width="` + bars[0].SolvedWidth + `"`, `x="` + bars[0].SolvedWidth + `" width="` + bars[0].OpenWidth + `"`,
		"<title>Array: average recall 50%</title>", "<title>Array: struggle rate 20% of 5 attempts</title>",
		`<span class="topic-due">2 due</span>`, `href="/leetgrinder/problems?topic=array"`,
		"Unknown: 0 solved of 1 attempted", `<span class="muted">Unknown</span>`,
		"<title>Week of 7 Sep: 2 problems solved</title>",
		"<title>Week of 7 Sep. Independent solves: 50% (1 of 2 solves)</title>",
		"<title>Week of 14 Sep. Independent solves: 100% (1 of 1 solve). Complexity checks right: 100% (1 of 1 check)</title>",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("stats page missing %q", want)
		}
	}
}
