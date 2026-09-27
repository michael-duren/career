package leetgrinder

import (
	"testing"
	"time"
)

func TestNotificationDefaults(t *testing.T) {
	s := DefaultSettings()
	want := map[string]NotificationPref{
		NotifyMissingWork:    {Enabled: true, Time: "17:00", Priority: "default"},
		NotifyBehindSchedule: {Enabled: true, Time: "17:00", Threshold: 3, Priority: "default"},
		NotifyMorningPlan:    {Enabled: false, Time: "08:00", Priority: "default"},
		NotifyLateEscalation: {Enabled: false, Time: "21:00", Priority: "high"},
		NotifyReviewBacklog:  {Enabled: false, Time: "17:00", Threshold: 10, Priority: "default"},
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
	s.Notifications = map[string]NotificationPref{NotifyMissingWork: {Enabled: false, Time: "18:30", Priority: "low"}}
	if s.NotificationPref(NotifyMissingWork).Time != "18:30" || !s.NotificationPref(NotifyBehindSchedule).Enabled {
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
}

func TestNotificationValidation(t *testing.T) {
	for name, p := range map[string]map[string]NotificationPref{
		"unknown kind":      {"weekly": {Time: "08:00", Priority: "default"}},
		"bad time":          {NotifyMissingWork: {Time: "25:00", Priority: "default"}},
		"seconds":           {NotifyMissingWork: {Time: "17:00:00", Priority: "default"}},
		"unpadded":          {NotifyMissingWork: {Time: "7:00", Priority: "default"}},
		"bad priority":      {NotifyMissingWork: {Time: "17:00", Priority: "loud"}},
		"missing threshold": {NotifyBehindSchedule: {Time: "17:00", Priority: "default"}},
		"huge threshold":    {NotifyBehindSchedule: {Time: "17:00", Threshold: 85, Priority: "default"}},
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
