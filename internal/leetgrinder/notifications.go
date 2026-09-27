package leetgrinder

import (
	"errors"
	"fmt"
	"time"
)

// Notification kinds. Each is configured separately in Settings.Notifications.
const (
	NotifyMissingWork    = "missing_work"
	NotifyBehindSchedule = "behind_schedule"
	NotifyMorningPlan    = "morning_plan"
	NotifyLateEscalation = "late_escalation"
	NotifyReviewBacklog  = "review_backlog"
)

// NotificationKind describes one kind for the settings page and the worker.
type NotificationKind struct {
	Key         string
	Label       string
	Description string
	// ThresholdLabel is empty for kinds without a threshold.
	ThresholdLabel string
	MaxThreshold   int
	Default        NotificationPref
}

// NotificationKinds lists every kind in display order.
var NotificationKinds = []NotificationKind{
	{Key: NotifyMorningPlan, Label: "Morning plan", Description: "Today's session, required reading, and review problems.",
		Default: NotificationPref{Enabled: false, Time: "08:00", Priority: "default"}},
	{Key: NotifyMissingWork, Label: "Missing work", Description: "Today's session or planned reviews are still unfinished.",
		Default: NotificationPref{Enabled: true, Time: "17:00", Priority: "default"}},
	{Key: NotifyBehindSchedule, Label: "Behind schedule", Description: "You are at least the threshold number of sessions behind.",
		ThresholdLabel: "Sessions behind", MaxThreshold: SessionCount,
		Default: NotificationPref{Enabled: true, Time: "17:00", Threshold: 3, Priority: "default"}},
	{Key: NotifyReviewBacklog, Label: "Review backlog", Description: "At least the threshold number of due reviews did not fit in today's plan.",
		ThresholdLabel: "Due reviews", MaxThreshold: 300,
		Default: NotificationPref{Enabled: false, Time: "17:00", Threshold: 10, Priority: "default"}},
	{Key: NotifyLateEscalation, Label: "Late escalation", Description: "A second, louder reminder when work is still missing late in the day.",
		Default: NotificationPref{Enabled: false, Time: "21:00", Priority: "high"}},
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
		if kind.MaxThreshold > 0 && (p.Threshold < 1 || p.Threshold > kind.MaxThreshold) {
			return fmt.Errorf("%s: threshold must be 1 to %d", kind.Label, kind.MaxThreshold)
		}
		if kind.MaxThreshold == 0 && p.Threshold != 0 {
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
	return key
}
