package leetgrinder

import (
	"testing"
	"time"
)

func TestNotificationDefaults(t *testing.T) {
	s := DefaultSettings()
	want := map[string]NotificationPref{
		NotifyMorningPlan:    {Enabled: true, Time: "08:00", Priority: "default"},
		NotifyGoalIncomplete: {Enabled: true, Time: "18:00", Priority: "default"},
		NotifyStreakAtRisk:   {Enabled: true, Time: "21:00", Threshold: 1, Priority: "high"},
		NotifyReviewBacklog:  {Enabled: false, Time: "18:00", Threshold: 10, Priority: "default"},
	}
	if len(NotificationKinds) != len(want) {
		t.Fatalf("kinds: %d", len(NotificationKinds))
	}
	for kind, p := range want {
		if got := s.NotificationPref(kind); got != p {
			t.Errorf("%s default: %+v", kind, got)
		}
	}
	// Stored preferences replace the default for that kind only.
	s.Notifications = map[string]NotificationPref{NotifyGoalIncomplete: {Enabled: false, Time: "18:30", Priority: "low"}}
	if s.NotificationPref(NotifyGoalIncomplete).Time != "18:30" || !s.NotificationPref(NotifyStreakAtRisk).Enabled {
		t.Fatal("stored preference not applied")
	}
	defaults := map[string]NotificationPref{}
	for _, k := range NotificationKinds {
		defaults[k.Key] = k.Default
	}
	s.Notifications = defaults
	if err := s.Validate(); err != nil {
		t.Fatalf("defaults invalid: %v", err)
	}
	// A streak threshold of 0 means every day the goal is not met.
	s.Notifications = map[string]NotificationPref{NotifyStreakAtRisk: {Enabled: true, Time: "21:00", Priority: "high"}}
	if err := s.Validate(); err != nil {
		t.Fatalf("zero streak threshold: %v", err)
	}
	for key, want := range map[string]string{"missing_work": "Missing work", "late_escalation": "Late escalation", "behind_schedule": "Behind schedule", NotifyStreakAtRisk: "Streak at risk", NotifyTestKind: "Test", "other": "other"} {
		if got := NotificationKindLabel(key); got != want {
			t.Errorf("label %s: %q", key, got)
		}
	}
}

func TestNotificationValidation(t *testing.T) {
	for name, p := range map[string]map[string]NotificationPref{
		"unknown kind":      {"weekly": {Time: "08:00", Priority: "default"}},
		"retired kind":      {"missing_work": {Time: "17:00", Priority: "default"}},
		"bad time":          {NotifyGoalIncomplete: {Time: "25:00", Priority: "default"}},
		"seconds":           {NotifyGoalIncomplete: {Time: "17:00:00", Priority: "default"}},
		"unpadded":          {NotifyGoalIncomplete: {Time: "7:00", Priority: "default"}},
		"bad priority":      {NotifyGoalIncomplete: {Time: "17:00", Priority: "loud"}},
		"missing threshold": {NotifyReviewBacklog: {Time: "17:00", Priority: "default"}},
		"huge threshold":    {NotifyReviewBacklog: {Time: "17:00", Threshold: 501, Priority: "default"}},
		"negative streak":   {NotifyStreakAtRisk: {Time: "21:00", Threshold: -1, Priority: "high"}},
		"stray threshold":   {NotifyMorningPlan: {Time: "08:00", Threshold: 2, Priority: "default"}},
	} {
		s := DefaultSettings()
		s.Notifications = p
		if s.Validate() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestTriggerAt(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	date := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC) // DST starts in Chicago
	at, ok := NotificationPref{Time: "17:00"}.TriggerAt(date, loc)
	if !ok || at.In(loc).Hour() != 17 || at.In(loc).Day() != 8 {
		t.Fatalf("trigger: %v %v", at, ok)
	}
	if _, ok = (NotificationPref{Time: "soon"}).TriggerAt(date, loc); ok {
		t.Fatal("malformed time accepted")
	}
}
