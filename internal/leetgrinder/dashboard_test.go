package leetgrinder

import (
	"bytes"
	"context"
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
