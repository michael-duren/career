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
	m := Message{Priority: pref.Priority, Click: origin + "/leetgrinder"}
	switch kind {
	case leetgrinder.NotifyMissingWork, leetgrinder.NotifyLateEscalation:
		missing := today.MissingReviews()
		if len(missing) == 0 {
			return m, false
		}
		m.Title, m.Tags = "Leetgrinder: work left today", []string{"hourglass"}
		if kind == leetgrinder.NotifyLateEscalation {
			m.Title, m.Tags = "Leetgrinder: today is still unfinished", []string{"rotating_light"}
		}
		var lines []string
		for _, r := range missing {
			lines = append(lines, "Review: "+r.Problem.DisplayTitle())
		}
		m.Body = strings.Join(lines, "\n")
	case leetgrinder.NotifyMorningPlan:
		m.Title, m.Tags = "Leetgrinder: today's plan", []string{"sunrise"}
		var lines []string
		for _, r := range today.Reviews {
			lines = append(lines, "Review: "+r.Problem.DisplayTitle())
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
