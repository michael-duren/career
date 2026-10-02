package leetgrinder

import (
	"errors"
	"fmt"
	"time"
)

// Notification kinds. Each is configured separately in Settings.Notifications.
const (
	NotifyMorningPlan    = "morning_plan"
	NotifyGoalIncomplete = "goal_incomplete"
	NotifyStreakAtRisk   = "streak_at_risk"
	NotifyReviewBacklog  = "review_backlog"
)

// NotificationKind describes one kind for the settings page and the worker.
type NotificationKind struct {
	Key         string
	Label       string
	Description string
	// ThresholdLabel is empty for kinds without a threshold, which run from
	// MinThreshold to MaxThreshold.
	ThresholdLabel             string
	MinThreshold, MaxThreshold int
	Default                    NotificationPref
	// Window is how long after its time the kind may still send. Later than
	// that the reminder is stale and is logged as skipped.
	Window time.Duration
}

// HasThreshold reports whether the kind takes a threshold.
func (k NotificationKind) HasThreshold() bool { return k.ThresholdLabel != "" }

// NotificationKinds lists every kind in display order.
var NotificationKinds = []NotificationKind{
	{Key: NotifyMorningPlan, Label: "Morning plan", Description: "Today's targets, new problems picked from todos, review picks, due count, and current streak.",
		Default: NotificationPref{Enabled: true, Time: "08:00", Priority: "default"}, Window: 4 * time.Hour},
	{Key: NotifyGoalIncomplete, Label: "Goal incomplete", Description: "Today's goal is not met yet; lists what is left.",
		Default: NotificationPref{Enabled: true, Time: "18:00", Priority: "default"}, Window: 2 * time.Hour},
	{Key: NotifyStreakAtRisk, Label: "Streak at risk", Description: "Today's goal is not met and your streak is at least the threshold (0 sends whenever the goal is not met).",
		ThresholdLabel: "Streak days", MinThreshold: 0, MaxThreshold: 365,
		Default: NotificationPref{Enabled: true, Time: "21:00", Threshold: 1, Priority: "high"}, Window: 2 * time.Hour},
	{Key: NotifyReviewBacklog, Label: "Review backlog", Description: "At least the threshold number of due reviews are outside today's picks.",
		ThresholdLabel: "Due reviews", MinThreshold: 1, MaxThreshold: 500,
		Default: NotificationPref{Enabled: false, Time: "18:00", Threshold: 10, Priority: "default"}, Window: 2 * time.Hour},
}

// legacyKindLabels name kinds retired with the curriculum, which old log
// rows still carry.
var legacyKindLabels = map[string]string{
	"missing_work":    "Missing work",
	"behind_schedule": "Behind schedule",
	"late_escalation": "Late escalation",
}

// NotificationPriorities are the ntfy priority names, lowest first.
var NotificationPriorities = []string{"min", "low", "default", "high", "urgent"}

func FindNotificationKind(key string) (NotificationKind, bool) {
	for _, k := range NotificationKinds {
		if k.Key == key {
			return k, true
		}
	}
	return NotificationKind{}, false
}

// NotificationPref returns the stored preference for kind, or its default when
// none is stored.
func (s Settings) NotificationPref(kind string) NotificationPref {
	if p, ok := s.Notifications[kind]; ok {
		return p
	}
	k, _ := FindNotificationKind(kind)
	return k.Default
}

// TriggerAt is the instant on date (a Date value) when pref fires in loc.
// It returns false when the time is malformed.
func (p NotificationPref) TriggerAt(date time.Time, loc *time.Location) (time.Time, bool) {
	clock, err := time.Parse("15:04", p.Time)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(date.Year(), date.Month(), date.Day(), clock.Hour(), clock.Minute(), 0, 0, loc), true
}

func validNotificationPrefs(prefs map[string]NotificationPref) error {
	for key, p := range prefs {
		kind, ok := FindNotificationKind(key)
		if !ok {
			return fmt.Errorf("unknown notification %q", key)
		}
		if clock, err := time.Parse("15:04", p.Time); err != nil || clock.Format("15:04") != p.Time {
			return fmt.Errorf("%s: time must be HH:MM", kind.Label)
		}
		if kind.HasThreshold() && (p.Threshold < kind.MinThreshold || p.Threshold > kind.MaxThreshold) {
			return fmt.Errorf("%s: threshold must be %d to %d", kind.Label, kind.MinThreshold, kind.MaxThreshold)
		}
		if !kind.HasThreshold() && p.Threshold != 0 {
			return fmt.Errorf("%s: takes no threshold", kind.Label)
		}
		valid := false
		for _, name := range NotificationPriorities {
			valid = valid || p.Priority == name
		}
		if !valid {
			return errors.New(kind.Label + ": choose a priority from min to urgent")
		}
	}
	return nil
}

// NotificationLogEntry is one row of leetgrinder_notification_log.
type NotificationLogEntry struct {
	Kind      string
	LocalDate time.Time
	SentAt    time.Time
	Status    string
	Detail    string
	Attempts  int
}

// Notification log statuses. A kind whose condition was not met when
// evaluated is logged as skipped so it is not re-evaluated that day.
const (
	NotifySent    = "sent"
	NotifyFailed  = "failed"
	NotifySending = "sending"
	NotifySkipped = "skipped"
	// NotifyTestKind logs results of the settings page's test button.
	NotifyTestKind = "test"
	// NotifyMaxAttempts caps sends per kind and local date: the first send
	// plus up to three retries on later ticks.
	NotifyMaxAttempts = 4
)

func NotificationKindLabel(key string) string {
	if key == NotifyTestKind {
		return "Test"
	}
	if k, ok := FindNotificationKind(key); ok {
		return k.Label
	}
	if label, ok := legacyKindLabels[key]; ok {
		return label
	}
	return key
}
