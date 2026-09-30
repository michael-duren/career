package notify

import (
	"fmt"
	"strings"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// Compose builds the notification for kind, or returns false when its
// condition is not met today. origin is the app's public origin, used for
// the Click deep link.
func Compose(kind string, pref leetgrinder.NotificationPref, today leetgrinder.Today, origin string) (Message, bool) {
	m := Message{Priority: pref.Priority, Click: origin + "/leetgrinder"}
	switch kind {
	case leetgrinder.NotifyMorningPlan:
		m.Title, m.Tags = "Leetgrinder: today's plan", []string{"sunrise"}
		lines := []string{"Goal: " + leetgrinder.GoalLabel(today.Goal)}
		for _, r := range today.Reviews {
			lines = append(lines, "Review: "+r.Problem.DisplayTitle())
		}
		if n := today.Backlog(); n > 0 {
			lines = append(lines, fmt.Sprintf("Also due: %d", n))
		}
		lines = append(lines, fmt.Sprintf("Streak: %d %s", today.Streaks.Current, leetgrinder.DayUnit(today.Streaks.Current)))
		m.Body = strings.Join(lines, "\n")
	case leetgrinder.NotifyGoalIncomplete:
		if today.Progress.Met() {
			return m, false
		}
		m.Title, m.Tags = "Leetgrinder: goal not met yet", []string{"hourglass"}
		m.Body = "Left: " + leftToDo(today)
	case leetgrinder.NotifyStreakAtRisk:
		streak := today.Streaks.Current
		if today.Progress.Met() || streak < pref.Threshold {
			return m, false
		}
		m.Title, m.Tags = fmt.Sprintf("Leetgrinder: %d-day streak at risk", streak), []string{"rotating_light"}
		if streak == 0 {
			m.Title = "Leetgrinder: today's goal is still open"
		}
		m.Body = "Left: " + leftToDo(today)
	case leetgrinder.NotifyReviewBacklog:
		n := today.Backlog()
		if pref.Threshold < 1 || n < pref.Threshold {
			return m, false
		}
		m.Title, m.Tags = fmt.Sprintf("Leetgrinder: %d reviews waiting", n), []string{"books"}
		m.Body = fmt.Sprintf("%d due reviews are outside today's picks. Catch up from the review queue; they count as bonus.", n)
		m.Click = origin + "/leetgrinder/reviews"
	default:
		return m, false
	}
	return m, true
}

// leftToDo lists what today's goal still needs, e.g. "1 new, review: Two Sum".
func leftToDo(today leetgrinder.Today) string {
	var parts []string
	if n := today.Progress.NewLeft(); n > 0 {
		parts = append(parts, fmt.Sprintf("%d new", n))
	}
	if n := today.Progress.ReviewsLeft(); n > 0 {
		var titles []string
		for _, r := range today.MissingReviews() {
			if len(titles) == n {
				break
			}
			titles = append(titles, r.Problem.DisplayTitle())
		}
		if len(titles) > 0 {
			parts = append(parts, "review: "+strings.Join(titles, ", "))
		}
		if extra := n - len(titles); extra == 1 {
			parts = append(parts, "1 more review")
		} else if extra > 1 {
			parts = append(parts, fmt.Sprintf("%d more reviews", extra))
		}
	}
	return strings.Join(parts, ", ")
}

// TestMessage is sent by the settings page's test button.
func TestMessage(origin string) Message {
	return Message{Title: "Leetgrinder test notification", Body: "ntfy is set up. Leetgrinder reminders will arrive here.", Priority: "default", Tags: []string{"white_check_mark"}, Click: origin + "/leetgrinder/settings"}
}
