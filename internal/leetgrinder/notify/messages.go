package notify

import (
	"fmt"
	"strings"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// Compose builds the notification for kind, or returns false when its
// condition is not met today. origin is the app's public origin, used for
// the Click deep link. cards is only read for the review backlog.
func Compose(kind string, pref leetgrinder.NotificationPref, today leetgrinder.Today, cards []leetgrinder.Card, origin string) (Message, bool) {
	m := Message{Priority: pref.Priority, Click: origin + dayLink(today)}
	switch kind {
	case leetgrinder.NotifyMissingWork, leetgrinder.NotifyLateEscalation:
		missing := today.Missing()
		if missing.Empty() {
			return m, false
		}
		m.Title, m.Tags = "Leetgrinder: work left today", []string{"hourglass"}
		if kind == leetgrinder.NotifyLateEscalation {
			m.Title, m.Tags = "Leetgrinder: today is still unfinished", []string{"rotating_light"}
		}
		var lines []string
		if missing.Session > 0 {
			lines = append(lines, sessionLine(missing.Session))
		}
		for _, r := range missing.Reviews {
			lines = append(lines, "Review: "+r.Problem.Title)
		}
		m.Body = strings.Join(lines, "\n")
	case leetgrinder.NotifyBehindSchedule:
		schedule, ok := today.Schedule()
		if !ok {
			return m, false
		}
		completed := leetgrinder.Summarize(today.State).Completed
		delta := schedule.Delta(completed, today.Now)
		if pref.Threshold < 1 || delta > -pref.Threshold {
			return m, false
		}
		m.Title, m.Tags = fmt.Sprintf("Leetgrinder: %s behind", sessions(-delta)), []string{"chart_with_downwards_trend"}
		m.Body = fmt.Sprintf("%d of %d expected sessions are finished. Pick up with the earliest unfinished session.", completed, schedule.ExpectedSessions(today.Now))
		m.Click = origin + "/leetgrinder"
	case leetgrinder.NotifyMorningPlan:
		m.Title, m.Tags = "Leetgrinder: today's plan", []string{"sunrise"}
		var lines []string
		if day, ok := leetgrinder.FindDay(today.Session); ok {
			lines = append(lines, sessionLine(day.Number))
			for _, r := range day.Readings {
				if !r.Optional {
					lines = append(lines, fmt.Sprintf("Reading: %s (%d min)", r.Title, r.Minutes))
				}
			}
		}
		for _, r := range today.Reviews {
			lines = append(lines, "Review: "+r.Problem.Title)
		}
		if len(lines) == 0 {
			lines = append(lines, "Nothing is scheduled today.")
		}
		m.Body = strings.Join(lines, "\n")
	case leetgrinder.NotifyReviewBacklog:
		n := today.Backlog(cards)
		if pref.Threshold < 1 || n < pref.Threshold {
			return m, false
		}
		m.Title, m.Tags = fmt.Sprintf("Leetgrinder: %d reviews waiting", n), []string{"books"}
		m.Body = fmt.Sprintf("%d due reviews did not fit in today's plan. Add daily time or catch up from the review queue.", n)
		m.Click = origin + "/leetgrinder/reviews"
	default:
		return m, false
	}
	return m, true
}

// TestMessage is sent by the settings page's test button.
func TestMessage(origin string) Message {
	return Message{Title: "Leetgrinder test notification", Body: "ntfy is set up. Leetgrinder reminders will arrive here.", Priority: "default", Tags: []string{"white_check_mark"}, Click: origin + "/leetgrinder/settings"}
}

func dayLink(today leetgrinder.Today) string {
	if today.Session > 0 {
		return leetgrinder.DayURL(today.Session)
	}
	return "/leetgrinder"
}

func sessionLine(n int) string {
	day, _ := leetgrinder.FindDay(n)
	return fmt.Sprintf("Session %d: %s", n, day.Title)
}

func sessions(n int) string {
	if n == 1 {
		return "1 session"
	}
	return fmt.Sprintf("%d sessions", n)
}
