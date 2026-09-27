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
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	settings.StartDate = &start

	fresh := OverviewPage{Today: NewToday(settings, State{}, nil, now)}
	if fresh.Started() || fresh.Streak() != 0 || fresh.IndependentRate() != "—" {
		t.Fatal("fresh learner has progress")
	}
	var out bytes.Buffer
	if err := Overview(fresh).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"Start Day 1", `href="/leetgrinder/day/1"`, "Two Sum", `<a href="/leetgrinder" aria-current="page">`, "Coming up", "Your logged attempts will show up here.", "<span>Dashboard</span>", "<span>Settings</span>"} {
		if !strings.Contains(html, want) {
			t.Errorf("fresh dashboard missing %q", want)
		}
	}

	state := State{
		Attempts: []Attempt{
			attempt("ransom-note", "solved", 20, true, now.Add(-time.Hour)),
			attempt("two-sum", "solved", 15, false, now.AddDate(0, 0, -1)),
			attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -3)),
		},
		CompletedDays: []int{1},
	}
	page := OverviewPage{Today: NewToday(settings, state, nil, now)}
	if page.Streak() != 2 || page.IndependentRate() != "50%" || len(page.Recent(2)) != 2 {
		t.Fatalf("streak %d, rate %s", page.Streak(), page.IndependentRate())
	}
	upcoming := page.Upcoming(3)
	if len(upcoming) != 3 || upcoming[0].Day.Number != 3 || !upcoming[0].Date.Equal(time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("upcoming %+v", upcoming)
	}
	out.Reset()
	if err := Overview(page).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html = out.String()
	for _, want := range []string{"Resume Day 2", "Welcome back", "Solved with help", "Overdue · was Sat 3 Oct 2026"} {
		if !strings.Contains(html, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}

	out.Reset()
	if err := About(page.Today.State).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html = out.String()
	for _, want := range []string{`id="curriculum"`, "How to use your two hours", `<a href="/leetgrinder/about" aria-current="page">`, "Finished"} {
		if !strings.Contains(html, want) {
			t.Errorf("about missing %q", want)
		}
	}
}

func TestDashboardSkipsFinishedAndUsesLocalDates(t *testing.T) {
	settings := DefaultSettings()
	settings.Timezone = "America/Los_Angeles"
	loc := settings.Location()
	// 10:00 local on 20 Oct is 17:00 UTC; 23:30 local on 19 Oct is already 20 Oct in UTC.
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, loc)
	state := State{
		Attempts: []Attempt{
			attempt("two-sum", "solved", 15, false, time.Date(2026, 10, 19, 23, 30, 0, 0, loc)),
			attempt("ransom-note", "solved", 15, false, time.Date(2026, 10, 18, 9, 0, 0, 0, loc)),
		},
		CompletedDays: []int{1, 3},
	}
	page := OverviewPage{Today: NewToday(settings, state, nil, now)}
	if DayUnit(1) != "day" || DayUnit(0) != "days" || DayUnit(2) != "days" {
		t.Fatal("day unit")
	}
	if got := page.Streak(); got != 2 {
		t.Fatalf("streak %d, want 2 local days ending yesterday", got)
	}
	upcoming := page.Upcoming(2)
	if len(upcoming) != 2 || upcoming[0].Day.Number != 4 || upcoming[1].Day.Number != 5 {
		t.Fatalf("upcoming should skip finished day 3: %+v", upcoming)
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

func TestUpcomingOverdue(t *testing.T) {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	today := Date(now, time.UTC)
	if (UpcomingSession{Date: today}).Overdue(today) || !(UpcomingSession{Date: today.AddDate(0, 0, -1)}).Overdue(today) || (UpcomingSession{}).Overdue(today) {
		t.Fatal("overdue boundary")
	}
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	start := today
	settings.StartDate = &start
	var out bytes.Buffer
	if err := Overview(OverviewPage{Today: NewToday(settings, State{}, nil, now)}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if html := out.String(); strings.Contains(html, "Overdue") || !strings.Contains(html, "Wed 21 Oct 2026") {
		t.Fatal("on-schedule sessions should show their date without an overdue label")
	}
}
