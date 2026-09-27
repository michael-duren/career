package leetgrinder

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultTimezone   = "America/Chicago"
	DefaultNtfyURL    = "https://ntfy.sh"
	MinDailyHours     = 2.0
	MaxDailyHours     = 4.0
	DailyHoursStep    = 0.5
	ReviewSlotMinutes = 25
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
	StartDate           *time.Time
	Timezone            string
	DailyHours          float64
	NtfyURL             string
	NtfyTopic           string
	NtfyTokenCiphertext []byte
	Notifications       map[string]NotificationPref
	// AnalysisEnabled turns LLM complexity analysis on or off. It defaults
	// to on; analysis also needs ANTHROPIC_API_KEY.
	AnalysisEnabled bool
	Revision        string
}

func DefaultSettings() Settings {
	return Settings{Timezone: DefaultTimezone, DailyHours: MinDailyHours, NtfyURL: DefaultNtfyURL, Notifications: map[string]NotificationPref{}, AnalysisEnabled: true}
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

func (s Settings) Schedule() (Schedule, bool) {
	if s.StartDate == nil {
		return Schedule{}, false
	}
	return NewSchedule(*s.StartDate, s.Location()), true
}

func (s Settings) TokenSet() bool { return len(s.NtfyTokenCiphertext) > 0 }

var ntfyTopic = regexp.MustCompile(`^[A-Za-z0-9_-]{0,64}$`)

func ValidDailyHours(h float64) bool {
	steps := (h - MinDailyHours) / DailyHoursStep
	return h >= MinDailyHours && h <= MaxDailyHours && steps == math.Trunc(steps)
}

func (s Settings) Validate() error {
	if _, err := LoadTimezone(s.Timezone); err != nil {
		return fmt.Errorf("time zone: %w", err)
	}
	if !ValidDailyHours(s.DailyHours) {
		return errors.New("daily hours must be 2 to 4 in half-hour steps")
	}
	if s.StartDate != nil && (s.StartDate.Year() < 2000 || s.StartDate.Year() > 2100) {
		return errors.New("start date must be between 2000 and 2100")
	}
	if u, err := url.Parse(s.NtfyURL); err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("ntfy server must be an absolute http(s) URL")
	}
	if !ntfyTopic.MatchString(s.NtfyTopic) {
		return errors.New("ntfy topic may use up to 64 letters, digits, - and _")
	}
	return validNotificationPrefs(s.Notifications)
}

// ExtraReviewSlots converts time beyond the base two hours into 25-minute review slots.
func ExtraReviewSlots(hours float64) int {
	return max(0, int(math.Floor((hours-MinDailyHours)*60/ReviewSlotMinutes+1e-9)))
}

func HoursOptions() []float64 {
	var out []float64
	for h := MinDailyHours; h <= MaxDailyHours; h += DailyHoursStep {
		out = append(out, h)
	}
	return out
}
