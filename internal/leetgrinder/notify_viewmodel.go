package leetgrinder

import (
	"strconv"
	"time"
)

// NtfyForm is the ntfy server section. The token itself is write-only: it is
// never rendered, not even from a rejected draft.
type NtfyForm struct {
	URL, Topic, Revision string
	TokenSet, ClearToken bool
}

// NotificationDraft keeps one kind's submitted text for a retry.
type NotificationDraft struct {
	Kind                      NotificationKind
	Enabled                   bool
	Time, Threshold, Priority string
}

type NotificationsForm struct {
	Revision string
	Items    []NotificationDraft
}

// NotifyPanel is everything the notification sections of the settings page show.
type NotifyPanel struct {
	Ntfy          NtfyForm
	Notifications NotificationsForm
	Log           []NotificationLogEntry
	LogError      bool
	Location      *time.Location
	// SecretMissing warns that a token cannot be stored or read.
	SecretMissing bool
	TestSent      bool
	// TopicMissing reports that no ntfy topic is saved, so nothing can send.
	TopicMissing bool
}

func NewNtfyForm(s Settings) NtfyForm {
	return NtfyForm{URL: s.NtfyURL, Topic: s.NtfyTopic, Revision: s.Revision, TokenSet: s.TokenSet()}
}

func NewNotificationsForm(s Settings) NotificationsForm {
	f := NotificationsForm{Revision: s.Revision}
	for _, kind := range NotificationKinds {
		p := s.NotificationPref(kind.Key)
		d := NotificationDraft{Kind: kind, Enabled: p.Enabled, Time: p.Time, Priority: p.Priority}
		if kind.HasThreshold() {
			d.Threshold = strconv.Itoa(p.Threshold)
		}
		f.Items = append(f.Items, d)
	}
	return f
}

func NotificationStatusLabel(status string) string {
	switch status {
	case NotifySent:
		return "Sent"
	case NotifyFailed:
		return "Failed"
	case NotifySending:
		return "Sending"
	case NotifySkipped:
		return "Nothing to send"
	}
	return status
}

func NotificationTimeLabel(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("Mon 2 Jan 15:04")
}

func PriorityLabel(p string) string {
	if p == "" {
		return ""
	}
	return string(p[0]-'a'+'A') + p[1:]
}
