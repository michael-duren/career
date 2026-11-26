package leetgrinder

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestDashboard(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Timezone = "UTC"

	fresh := OverviewPage{Today: NewToday(settings, State{}, now)}
	if fresh.Started() || fresh.Today.Streaks != (Streaks{}) || fresh.IndependentRate() != "—" {
		t.Fatal("fresh learner has progress")
	}
	var out bytes.Buffer
	if err := Overview(fresh).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`<span class="lg-streak-num">00</span>`, `<a href="/leetgrinder" aria-current="page">Today</a>`, `>Reviews</a>`, `>Sets</a>`, `<a href="/leetgrinder/log">Log</a>`, `>Settings</a>`, `href="/leetgrinder/export"`,
		"Weekly columns appear after your first attempt.", "Nothing else is due today.", "Nothing is due for review today.", `id="theme-init"`, `href="/ui/fonts.css"`} {
		if !strings.Contains(html, want) {
			t.Errorf("fresh dashboard missing %q", want)
		}
	}
	for _, gone := range []string{"/leetgrinder/about", "/leetgrinder/day/", "lesson-player", `class="lg-build"`, `class="lg-weeks"`} {
		if strings.Contains(html, gone) {
			t.Errorf("fresh dashboard still has %q", gone)
		}
	}

	state := State{Problems: testProblems, Attempts: []Attempt{
		attempt("ransom-note", "solved", 20, true, now.Add(-time.Hour)),
		attempt("two-sum", "solved", 15, false, now.AddDate(0, 0, -1)),
		attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -3)),
		attempt("brand-new-slug", "unfinished", 30, false, now.AddDate(0, 0, -20)),
	}}
	today := NewToday(settings, state, now)
	page := OverviewPage{Today: today}
	if _, due := DueSoon(today, 10); today.Streaks.Active != 2 || page.IndependentRate() != "50%" || len(page.Recent(2)) != 2 || due != 1 {
		t.Fatalf("streak %d, rate %s, due %d", today.Streaks.Active, page.IndependentRate(), due)
	}
	out.Reset()
	if err := Overview(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html = out.String()
	for _, want := range []string{"brand-new-slug", "Optional: 1 more due by the end of today.", "Solved per week", `class="lg-week-bar lg-week-bar--current"`, "Week of Oct 19: 2 problems solved", "Independent", "Difficulty", "2/2 · 100%"} {
		if !strings.Contains(html, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}

	out.Reset()
	if err := LogView(LogPage{OverviewPage: page, Ref: "nope", Error: "Enter a LeetCode problem link"}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html = out.String()
	for _, want := range []string{`<a href="/leetgrinder/log" aria-current="page">Log</a>`, "Recent attempts", "Ransom Note", `action="/leetgrinder/log"`, `value="nope"`, "Enter a LeetCode problem link", "All time", "Oct 20"} {
		if !strings.Contains(html, want) {
			t.Errorf("log page missing %q", want)
		}
	}
}

func TestStreakTicks(t *testing.T) {
	for _, tc := range []struct{ current, record, top, n, on int }{
		{0, 0, 7, 8, 0},
		{2, 29, 29, 30, 2},
		{12, 12, 12, 13, 12},
		{60, 120, 120, 61, 30},
	} {
		top, ticks := streakTicks(tc.current, tc.record)
		on := 0
		for _, tick := range ticks {
			if tick.On {
				on++
			}
		}
		if top != tc.top || len(ticks) != tc.n || on != tc.on || !ticks[0].Major || ticks[1].Major || !ticks[5].Major {
			t.Errorf("streakTicks(%d, %d) = top %d, %d ticks, %d on", tc.current, tc.record, top, len(ticks), on)
		}
	}
}

func TestChartScale(t *testing.T) {
	for most, want := range map[int]int{0: 4, 3: 4, 5: 6, 9: 10, 10: 10, 11: 20, 20: 20, 41: 50, 51: 100} {
		if got := chartScale(most); got != want {
			t.Errorf("chartScale(%d) = %d, want %d", most, got, want)
		}
	}
}

func TestPaceNote(t *testing.T) {
	weeks := func(solved ...int) Trends {
		var tr Trends
		for _, n := range solved {
			tr.Weeks = append(tr.Weeks, TrendWeek{Solved: n})
		}
		return tr
	}
	for _, tc := range []struct {
		trends Trends
		want   string
	}{
		{weeks(5, 5, 5, 5, 5, 5, 5, 5), ""}, // the account is too young for two full windows
		{weeks(0, 0, 0, 0, 0, 0, 0, 0, 9), ""},
		// The current (last) week is partial and left out.
		{weeks(5, 5, 5, 5, 3, 3, 3, 3, 0), "pace slipping · 12 vs 20"},
		{weeks(9, 2, 2, 2, 2, 5, 5, 5, 5, 99), "pace picking up · 20 vs 8"},
		{weeks(5, 5, 5, 5, 5, 4, 5, 5, 1), "pace steady · 19 vs 20"},
		// The cutoffs: below 80% slips, above 120% picks up, both ends inclusive of steady.
		{weeks(5, 5, 5, 5, 4, 4, 4, 4, 0), "pace steady · 16 vs 20"},
		{weeks(5, 5, 5, 5, 4, 4, 4, 3, 0), "pace slipping · 15 vs 20"},
		{weeks(5, 5, 5, 5, 6, 6, 6, 6, 0), "pace steady · 24 vs 20"},
		{weeks(5, 5, 5, 5, 7, 6, 6, 6, 0), "pace picking up · 25 vs 20"},
		// One empty window still compares.
		{weeks(0, 0, 0, 0, 1, 0, 0, 0, 0), "pace picking up · 1 vs 0"},
		{weeks(1, 0, 0, 0, 0, 0, 0, 0, 0), "pace slipping · 0 vs 1"},
	} {
		note, counts, title := paceNote(tc.trends)
		got := note
		if counts != "" {
			got += " · " + counts
		}
		if got != tc.want || (got != "" && !strings.Contains(title, "full weeks")) {
			t.Errorf("paceNote = %q (%q), want %q", got, title, tc.want)
		}
	}
}

func TestDashboardRatesAndWeakestTopics(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	problems := map[string]Problem{}
	var attempts []Attempt
	// Four problems in "graph" and three in "tree"; graph struggles more.
	for i, slug := range []string{"g1", "g2", "g3", "g4"} {
		problems[slug] = Problem{Slug: slug, Title: slug, Difficulty: "Medium", Topics: []string{"graph"}}
		attempts = append(attempts, attempt(slug, "struggled", 30, false, now.AddDate(0, 0, -10-i)))
	}
	for i, slug := range []string{"t1", "t2", "t3"} {
		problems[slug] = Problem{Slug: slug, Title: slug, Difficulty: "Easy", Topics: []string{"tree"}}
		attempts = append(attempts, attempt(slug, "solved", 10, false, now.AddDate(0, 0, -2-i)))
	}
	// Newest first.
	slices.SortFunc(attempts, func(a, b Attempt) int { return b.CreatedAt.Compare(a.CreatedAt) })
	d := OverviewPage{Today: NewToday(settings, State{Problems: problems, Attempts: attempts}, now)}.Dashboard()

	if d.Independent != "100%" || d.AvgMedium != "30 min" || d.Complexity != "—" || d.Solved != 3 {
		t.Errorf("rates %q %q %q, solved %d", d.Independent, d.AvgMedium, d.Complexity, d.Solved)
	}
	if len(d.Topics) != 2 || d.Topics[0].Key != "graph" || !d.Topics[0].WeakStruggle || d.Topics[1].WeakStruggle {
		t.Fatalf("topics %+v", d.Topics)
	}
	if len(d.BuildProblems) != 2 || !strings.HasPrefix(d.BuildProblems[0], "g") || !strings.HasPrefix(d.BuildProblems[1], "t") || d.BuildTitle != "Weakest topics · Oct 20" {
		t.Errorf("build set %q %v", d.BuildTitle, d.BuildProblems)
	}
	if len(d.Difficulties) != 2 || d.Difficulties[0].Label != "3/3 · 100%" || d.Difficulties[1].Label != "0/4 · 0%" {
		t.Errorf("difficulties %+v", d.Difficulties)
	}
	if d.Scale != 4 || d.Guides[0].Value != 2 || len(d.Weeks) != 3 || d.Weeks[1].Height != "75.0%" || !d.Weeks[2].Current {
		t.Errorf("chart scale %d guides %+v weeks %+v", d.Scale, d.Guides, d.Weeks)
	}
}

func TestActiveStreakUsesLocalDates(t *testing.T) {
	settings := DefaultSettings()
	settings.Timezone = "America/Los_Angeles"
	loc := settings.Location()
	// 10:00 local on 20 Oct is 17:00 UTC; 23:30 local on 19 Oct is already 20 Oct in UTC.
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, loc)
	state := State{Attempts: []Attempt{
		attempt("two-sum", "solved", 15, false, time.Date(2026, 10, 19, 23, 30, 0, 0, loc)),
		attempt("ransom-note", "solved", 15, false, time.Date(2026, 10, 18, 9, 0, 0, 0, loc)),
	}}
	today := NewToday(settings, state, now)
	if DayUnit(1) != "day" || DayUnit(0) != "days" || DayUnit(2) != "days" {
		t.Fatal("day unit")
	}
	if got := today.Streaks.Active; got != 2 {
		t.Fatalf("streak %d, want 2 local days ending yesterday", got)
	}
}

func TestNtfyTopicWarning(t *testing.T) {
	render := func(missing bool) string {
		var out bytes.Buffer
		page := SettingsPage{Settings: DefaultSettings(), Notify: NotifyPanel{TopicMissing: missing}}
		if err := SettingsView(page).Render(context.Background(), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	html := render(true)
	if strings.Count(html, "No topic set") != 2 || !strings.Contains(html, "No reminders will be sent") || !strings.Contains(html, `popovertarget="ntfy-help"`) {
		t.Fatal("missing topic warnings")
	}
	html = render(false)
	if strings.Contains(html, "No topic set") || !strings.Contains(html, `id="notifications-help"`) {
		t.Fatal("warning shown with a topic, or help missing")
	}
}

func TestWeakFlags(t *testing.T) {
	for _, tc := range []struct {
		recall, struggle float64
		low, high        bool
	}{
		{0.8, 0.55, false, false},
		{0.7999, 0.5501, true, true},
		{0.81, 0.54, false, false},
		{0, 1, true, true},
	} {
		if low, high := weakFlags(tc.recall, tc.struggle); low != tc.low || high != tc.high {
			t.Errorf("weakFlags(%v, %v) = %v, %v", tc.recall, tc.struggle, low, high)
		}
	}
}

func dashboardSettings() Settings {
	s := DefaultSettings()
	s.Timezone = "UTC"
	return s
}

// TestDashboardWeakestTopicsOrderAndPicks: topics rank by recall (not by
// struggle or name), Other and Untagged are left out, and the build set takes
// each topic's lowest-recall problem once.
func TestDashboardWeakestTopicsOrderAndPicks(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	problems := map[string]Problem{}
	var attempts []Attempt
	add := func(slug string, topics []string, outcome string, daysAgo int) {
		problems[slug] = Problem{Slug: slug, Title: slug, Difficulty: "Easy", Topics: topics}
		attempts = append(attempts, attempt(slug, outcome, 20, false, now.AddDate(0, 0, -daysAgo)))
	}
	// Recall, struggle and name orders all differ: "alpha" is recent (highest
	// recall) and always struggled; "zeta" never struggled after "shared".
	add("shared", []string{"zeta", "mid"}, "struggled", 90)
	add("z2", []string{"zeta"}, "solved", 80)
	add("z3", []string{"zeta"}, "solved", 70)
	add("m2", []string{"mid"}, "struggled", 40)
	add("m3", []string{"mid"}, "solved", 30)
	add("a1", []string{"alpha"}, "struggled", 3)
	add("a2", []string{"alpha"}, "struggled", 2)
	add("a3", []string{"alpha"}, "struggled", 1)
	add("rare", []string{"rare-topic"}, "solved", 100) // under 3 problems: folded into Other
	add("bare", nil, "solved", 110)                    // untagged
	slices.SortFunc(attempts, func(a, b Attempt) int { return b.CreatedAt.Compare(a.CreatedAt) })
	page := OverviewPage{Today: NewToday(dashboardSettings(), State{Problems: problems, Attempts: attempts}, now)}
	d := page.Dashboard()

	var keys []string
	for i, topic := range d.Topics {
		keys = append(keys, topic.Key)
		if i > 0 && topic.Recall < d.Topics[i-1].Recall {
			t.Errorf("topics not in ascending recall: %+v", d.Topics)
		}
	}
	if !slices.Equal(keys, []string{"mid", "zeta", "alpha"}) {
		t.Fatalf("topic order %v, want by recall (not by name or struggle)", keys)
	}
	if !d.Topics[2].WeakStruggle || d.Topics[2].WeakRecall || d.Topics[1].WeakStruggle || !d.Topics[1].WeakRecall {
		t.Errorf("struggle flags %+v", d.Topics)
	}
	// "shared" is the weakest problem in mid and in zeta: mid takes it, and
	// zeta falls back to its next-weakest problem.
	if !slices.Equal(d.BuildProblems, []string{"shared", "z2", "a1"}) {
		t.Errorf("build problems %v", d.BuildProblems)
	}
	var out bytes.Buffer
	if err := Overview(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`<td class="lg-weak">`, ", above 55%", ", below 80%", `name="problems" value="shared`, "Build todo set from top 3"} {
		if !strings.Contains(html, want) {
			t.Errorf("weakest topics missing %q", want)
		}
	}
	for _, gone := range []string{`topic=other`, `topic=untagged`, `topic=rare-topic`} {
		if strings.Contains(html, gone) {
			t.Errorf("weakest topics lists %q", gone)
		}
	}
}

func TestDashboardTopicCaps(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	problems := map[string]Problem{}
	var attempts []Attempt
	for i := range 12 {
		topic := fmt.Sprintf("topic-%02d", i)
		for j := range 3 {
			slug := fmt.Sprintf("p-%02d-%d", i, j)
			problems[slug] = Problem{Slug: slug, Title: slug, Topics: []string{topic}}
			attempts = append(attempts, attempt(slug, "solved", 20, false, now.AddDate(0, 0, -(i*3+j+1))))
		}
	}
	slices.SortFunc(attempts, func(a, b Attempt) int { return b.CreatedAt.Compare(a.CreatedAt) })
	d := OverviewPage{Today: NewToday(dashboardSettings(), State{Problems: problems, Attempts: attempts}, now)}.Dashboard()
	if len(d.Topics) != DashboardTopics || len(d.BuildProblems) != BuildSetTopics {
		t.Fatalf("%d topics, %d build problems", len(d.Topics), len(d.BuildProblems))
	}
}

// TestDashboardPaceFromHistory runs the pace note end to end: nine weeks of
// solves, the last eight charted, the current week last.
func TestDashboardPaceFromHistory(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) // a Tuesday
	monday := time.Date(2026, 8, 24, 15, 0, 0, 0, time.UTC)
	problems := map[string]Problem{}
	var attempts []Attempt
	// Weeks of Aug 24 to Sep 14: 5 solves each; Sep 21 to Oct 12: 3 each; this week: 1.
	for week, n := range []int{5, 5, 5, 5, 3, 3, 3, 3, 1} {
		for i := range n {
			slug := fmt.Sprintf("w%d-%d", week, i)
			problems[slug] = Problem{Slug: slug, Title: slug, Difficulty: "Medium"}
			attempts = append(attempts, attempt(slug, "solved", 20, false, monday.AddDate(0, 0, 7*week+1)))
		}
	}
	slices.SortFunc(attempts, func(a, b Attempt) int { return b.CreatedAt.Compare(a.CreatedAt) })
	page := OverviewPage{Today: NewToday(dashboardSettings(), State{Problems: problems, Attempts: attempts}, now)}
	d := page.Dashboard()
	if d.Pace != "pace slipping" || d.PaceCounts != "12 vs 20" {
		t.Fatalf("pace %q %q", d.Pace, d.PaceCounts)
	}
	if len(d.Weeks) != DashboardWeeks || d.Weeks[0].Label != "Aug 31" || !d.Weeks[7].Current || d.Weeks[6].Current || d.Solved != 28 {
		t.Fatalf("weeks %+v, solved %d", d.Weeks, d.Solved)
	}
	var out bytes.Buffer
	if err := Overview(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `class="ui-note lg-pace"`) || !strings.Contains(out.String(), "12 solved in the last 4 full weeks, 20 in the 4 before") {
		t.Error("pace note not rendered")
	}
}
