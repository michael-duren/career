package leetgrinder

import (
	_ "embed"
	"fmt"
	"strconv"
	"time"
)

//go:embed style.css
var Styles string

type AttemptForm struct {
	ID        string
	Revision  string
	Outcome   string
	Minutes   string
	Assisted  bool
	Notes     string
	ReturnDay int
	// Review marks attempts logged from a review card; Return is "",
	// "overview", or "reviews" and picks the page to go back to.
	Review bool
	Return string
	Error  string
	// Time and Space are the complexity controls as submitted.
	Time, Space ComplexityInput
}

// ComplexityInput is one complexity control: Choice is "", a canonical
// value from Complexities, or "other", in which case Other holds the text.
type ComplexityInput struct {
	Choice, Other string
}

// NewComplexityInput selects a saved value in the control.
func NewComplexityInput(value string) ComplexityInput {
	switch {
	case value == "" || IsCanonicalComplexity(value):
		return ComplexityInput{Choice: value}
	default:
		return ComplexityInput{Choice: "other", Other: value}
	}
}

// Value is the stated complexity before normalisation. Without JavaScript
// the text field is always submitted, so it also counts when nothing is
// selected; the page script disables it unless "Other" is chosen.
func (c ComplexityInput) Value() string {
	if c.Choice == "" || c.Choice == "other" {
		return c.Other
	}
	return c.Choice
}

type DayPage struct {
	Day   Day
	State State
	IDs   map[string]string
	Error string
	// Today is nil when today's plan could not be loaded.
	Today *Today
}

// ShowReviews puts today's reviews on the scheduled session and on the
// session the learner is actually working through.
func (p DayPage) ShowReviews() bool {
	return p.Today != nil && (p.Day.Number == p.Today.Session || p.Day.Number == Summarize(p.State).NextDay)
}

type OverviewPage struct {
	Today Today
	IDs   map[string]string
}

type ReviewsPage struct {
	Today Today
	Cards []Card
	IDs   map[string]string
}

// ScheduleForm keeps the submitted text so a rejected save can be retried.
type ScheduleForm struct {
	Start, End, Timezone, Hours, Revision string
}

func NewScheduleForm(s Settings) ScheduleForm {
	f := ScheduleForm{Timezone: s.Timezone, Hours: HoursValue(s.DailyHours), Revision: s.Revision}
	if schedule, ok := s.Schedule(); ok {
		f.Start, f.End = schedule.Start.Format(time.DateOnly), schedule.End().Format(time.DateOnly)
	}
	return f
}

var timezoneSuggestions = []string{"America/Chicago", "America/New_York", "America/Denver", "America/Phoenix", "America/Los_Angeles", "America/Anchorage", "Pacific/Honolulu", "Europe/London", "Europe/Berlin", "Asia/Kolkata", "Asia/Tokyo", "Australia/Sydney", "UTC"}

type SettingsPage struct {
	Settings Settings
	Now      time.Time
	Schedule ScheduleForm
	Error    string
	Saved    bool
	// Notify holds the ntfy and notification sections.
	Notify NotifyPanel
	// APITokens is the extension token section; see apitokens.templ.
	APITokens APITokensSection
	// Analysis is the complexity analysis section; see analysis.templ.
	Analysis AnalysisPanel
}

func ReviewKey(slug string) string { return "review-" + slug }
func DateLabel(t time.Time) string { return t.Format("Mon 2 Jan 2006") }
func Percent(f float64) string     { return strconv.Itoa(int(f*100+0.5)) + "%" }
func HoursValue(h float64) string  { return strconv.FormatFloat(h, 'f', 1, 64) }
func HoursLabel(h float64) string  { return strconv.FormatFloat(h, 'f', -1, 64) + " hours" }
func ReviewSlotsLabel(n int) string {
	if n == 1 {
		return "1 review slot"
	}
	return fmt.Sprintf("%d review slots", n)
}

func DayURL(n int) string           { return "/leetgrinder/day/" + strconv.Itoa(n) }
func ProblemURL(slug string) string { return "/leetgrinder/problem/" + slug }
func Count(n int) string            { return strconv.Itoa(n) }
func WeekNumber(day int) int        { return (day-1)/7 + 1 }
func DayLabel(day int) string       { return fmt.Sprintf("Day %d", day) }
func WeekLabel(week int) string     { return fmt.Sprintf("Week %02d", week) }
func NewForm(id string, day int) AttemptForm {
	return AttemptForm{ID: id, Outcome: "unfinished", Minutes: "25", ReturnDay: day}
}
func NewReviewForm(id string, day int, ret string) AttemptForm {
	f := NewForm(id, day)
	f.Review, f.Return = true, ret
	return f
}
func EditForm(a Attempt) AttemptForm {
	return AttemptForm{ID: a.ID, Revision: a.Revision, Outcome: a.Outcome, Minutes: Count(a.Minutes), Assisted: a.Assisted, Notes: a.Notes, Time: NewComplexityInput(a.TimeComplexity), Space: NewComplexityInput(a.SpaceComplexity)}
}
func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// ByteSize formats a size such as captured code for display.
func ByteSize(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f KB", float64(n)/1024)
}
func OutcomeLabel(s string) string {
	switch s {
	case "solved":
		return "Solved"
	case "struggled":
		return "Struggled"
	default:
		return "Unfinished"
	}
}
