package leetgrinder

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultTimezone = "America/Chicago"
	DefaultNtfyURL  = "https://ntfy.sh"
	// ReviewSlotMinutes is the time an unassisted solve may take to rate Easy.
	ReviewSlotMinutes = 25
	// MaxGoal caps each daily target.
	MaxGoal = 10
)

// NotificationPref configures one notification kind. Unused fields stay zero.
type NotificationPref struct {
	Enabled   bool   `json:"enabled"`
	Time      string `json:"time,omitempty"`
	Threshold int    `json:"threshold,omitempty"`
	Priority  string `json:"priority,omitempty"`
}

// Settings is the singleton Leetgrinder settings row. The ntfy token is only
// ever held encrypted here; see SecretBox.
type Settings struct {
	Timezone string
	// Goal is the daily target: new problems and reviews.
	Goal                DailyGoal
	NtfyURL             string
	NtfyTopic           string
	NtfyTokenCiphertext []byte
	Notifications       map[string]NotificationPref
	// AnalysisEnabled turns LLM complexity analysis on or off. It defaults
	// to on; analysis also needs ANTHROPIC_API_KEY.
	AnalysisEnabled bool
	// NewFromTodos picks each day's new problems from the oldest todos still
	// to do. It defaults to off; todos and the goal work without it.
	NewFromTodos bool
	Revision     string
}

func DefaultSettings() Settings {
	return Settings{Timezone: DefaultTimezone, Goal: DefaultGoal, NtfyURL: DefaultNtfyURL, Notifications: map[string]NotificationPref{}, AnalysisEnabled: true}
}

// LoadTimezone accepts IANA names only; "Local" would depend on the host.
func LoadTimezone(name string) (*time.Location, error) {
	if name == "" || name == "Local" || strings.EqualFold(name, "local") {
		return nil, errors.New("choose an IANA time zone such as America/Chicago")
	}
	return time.LoadLocation(name)
}

// Location falls back to the default zone if the stored name stopped loading.
func (s Settings) Location() *time.Location {
	if loc, err := LoadTimezone(s.Timezone); err == nil {
		return loc
	}
	loc, _ := time.LoadLocation(DefaultTimezone)
	return loc
}

// zone is the name of the zone Location loads.
func (s Settings) zone() string {
	if _, err := LoadTimezone(s.Timezone); err == nil {
		return s.Timezone
	}
	return DefaultTimezone
}

func (s Settings) TokenSet() bool { return len(s.NtfyTokenCiphertext) > 0 }

var ntfyTopic = regexp.MustCompile(`^[A-Za-z0-9_-]{0,64}$`)

func (s Settings) Validate() error {
	if _, err := LoadTimezone(s.Timezone); err != nil {
		return fmt.Errorf("time zone: %w", err)
	}
	if err := s.Goal.Validate(); err != nil {
		return err
	}
	if u, err := url.Parse(s.NtfyURL); err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("ntfy server must be an absolute http(s) URL")
	}
	if !ntfyTopic.MatchString(s.NtfyTopic) {
		return errors.New("ntfy topic may use up to 64 letters, digits, - and _")
	}
	return validNotificationPrefs(s.Notifications)
}
