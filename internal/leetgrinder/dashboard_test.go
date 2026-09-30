package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestDashboard(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Timezone = "UTC"

	fresh := OverviewPage{Today: NewToday(settings, State{}, nil, now)}
	if fresh.Started() || fresh.ActiveStreak() != 0 || fresh.IndependentRate() != "—" {
		t.Fatal("fresh learner has progress")
	}
	var out bytes.Buffer
	if err := Overview(fresh).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"Ready when you are", `<a href="/leetgrinder" aria-current="page">`, "Your logged attempts will show up here.", "<span>Dashboard</span>", "<span>Reviews</span>", "<span>Settings</span>", `action="/leetgrinder/log"`, "Nothing else is due today.", "Nothing is due for review today."} {
		if !strings.Contains(html, want) {
			t.Errorf("fresh dashboard missing %q", want)
		}
	}
	for _, gone := range []string{"/leetgrinder/about", "/leetgrinder/day/", "lesson-player"} {
		if strings.Contains(html, gone) {
			t.Errorf("dashboard still links %q", gone)
		}
	}

	state := State{Problems: testProblems, Attempts: []Attempt{
		attempt("ransom-note", "solved", 20, true, now.Add(-time.Hour)),
		attempt("two-sum", "solved", 15, false, now.AddDate(0, 0, -1)),
		attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -3)),
		attempt("brand-new-slug", "unfinished", 30, false, now.AddDate(0, 0, -20)),
	}}
	today := NewToday(settings, state, nil, now)
	page := OverviewPage{Today: today, Due: today.DueOptional(today.Cards()), LogRef: "nope", LogError: "Enter a LeetCode problem link"}
	if page.ActiveStreak() != 2 || page.IndependentRate() != "50%" || len(page.Recent(2)) != 2 || len(page.DueSoon(10)) != 1 {
		t.Fatalf("streak %d, rate %s, due %d", page.ActiveStreak(), page.IndependentRate(), len(page.DueSoon(10)))
	}
	out.Reset()
	if err := Overview(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html = out.String()
	for _, want := range []string{"Welcome back", "Ransom Note", "brand-new-slug", "Optional: 1 more due by the end of today.", `value="nope"`, "Enter a LeetCode problem link"} {
		if !strings.Contains(html, want) {
			t.Errorf("dashboard missing %q", want)
		}
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
	page := OverviewPage{Today: NewToday(settings, state, nil, now)}
	if DayUnit(1) != "day" || DayUnit(0) != "days" || DayUnit(2) != "days" {
		t.Fatal("day unit")
	}
	if got := page.ActiveStreak(); got != 2 {
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
