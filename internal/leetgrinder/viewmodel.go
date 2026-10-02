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
	ID       string
	Revision string
	Outcome  string
	Minutes  string
	Assisted bool
	Notes    string
	// WantsReview and Approach are the learner's own assessment.
	WantsReview bool
	Approach    string
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

type OverviewPage struct {
	Today     Today
	IDs       map[string]string
	NextTodos []TodoItem
	// LogRef and LogError keep a rejected "Log an attempt" entry.
	LogRef, LogError string
}

type ReviewsPage struct {
	Today Today
	IDs   map[string]string
}

// GeneralForm keeps the submitted text so a rejected save can be retried.
type GeneralForm struct {
	Timezone, GoalNew, GoalReview, Revision string
}

func NewGeneralForm(s Settings) GeneralForm {
	return GeneralForm{Timezone: s.Timezone, GoalNew: strconv.Itoa(s.Goal.New), GoalReview: strconv.Itoa(s.Goal.Review), Revision: s.Revision}
}

// GoalOptions are the choices for each daily target.
func GoalOptions() []string {
	out := make([]string, 0, MaxGoal+1)
	for n := 0; n <= MaxGoal; n++ {
		out = append(out, strconv.Itoa(n))
	}
	return out
}

// GoalLabel reads "2 new + 1 review".
func GoalLabel(g DailyGoal) string {
	review := "reviews"
	if g.Review == 1 {
		review = "review"
	}
	return fmt.Sprintf("%d new + %d %s", g.New, g.Review, review)
}

// ProblemReview is the review state shown on a problem's page; Card.Reviews
// is 0 for a problem without attempts.
type ProblemReview struct {
	Card   Card
	Due    bool
	Recall float64
	// FlagReason explains a flag, or is "".
	FlagReason string
	Pick       bool
	Location   *time.Location
	// Kinds is how each local day counts an attempt; nil hides the labels.
	Kinds func(Attempt) string
}

// AttemptTime is when a was logged, in the settings time zone (UTC when the
// settings are unavailable).
func (r ProblemReview) AttemptTime(a Attempt) time.Time {
	if r.Location == nil {
		return a.CreatedAt.UTC()
	}
	return a.CreatedAt.In(r.Location)
}

// AttemptKind is "" or the label of how a's day counted its problem, left out
// for the first attempt (new).
func (r ProblemReview) AttemptKind(a Attempt) string {
	if r.Kinds == nil {
		return ""
	}
	if kind := r.Kinds(a); kind != KindNew && kind != "" {
		return KindLabel(kind)
	}
	return ""
}

// NewProblemReview reads slug's review state from today.
func NewProblemReview(today Today, slug string) ProblemReview {
	r := ProblemReview{Location: today.Settings.Location(), Pick: today.Picked(slug), Kinds: today.KindOf}
	if c, ok := today.Card(slug); ok {
		r.Card, r.Due, r.Recall = c, today.Due(c), c.Retrievability(today.Now)
		if c.Flag != nil {
			r.FlagReason = c.Flag.Reason(today.Now, r.Location)
		}
	}
	return r
}

var timezoneSuggestions = []string{"America/Chicago", "America/New_York", "America/Denver", "America/Phoenix", "America/Los_Angeles", "America/Anchorage", "Pacific/Honolulu", "Europe/London", "Europe/Berlin", "Asia/Kolkata", "Asia/Tokyo", "Australia/Sydney", "UTC"}

type SettingsPage struct {
	Settings Settings
	Now      time.Time
	General  GeneralForm
	// TodayGoal is today's frozen goal, or nil when today has not started,
	// so the page can say when goal changes apply.
	TodayGoal *DailyGoal
	Error     string
	Saved     bool
	// Notify holds the ntfy and notification sections.
	Notify NotifyPanel
	// APITokens is the extension token section; see apitokens.templ.
	APITokens APITokensSection
	// Analysis is the complexity analysis section; see analysis.templ.
	Analysis AnalysisPanel
}

func ReviewKey(slug string) string { return "review-" + slug }
func DateLabel(t time.Time) string { return t.Format("Mon 2 Jan 2006") }

// CompactDate drops the weekday, for dense tables.
func CompactDate(t time.Time) string { return t.Format("2 Jan 2006") }
func Percent(f float64) string       { return strconv.Itoa(int(f*100+0.5)) + "%" }

func ProblemURL(slug string) string { return "/leetgrinder/problem/" + slug }
func Count(n int) string            { return strconv.Itoa(n) }

// NumberLabel is "LEETCODE 1" style text, or "" while the number is unknown.
func NumberLabel(n int) string {
	if n == 0 {
		return ""
	}
	return "LEETCODE " + strconv.Itoa(n)
}
func NewForm(id string) AttemptForm {
	return AttemptForm{ID: id, Outcome: "unfinished", Minutes: "25"}
}
func NewReviewForm(id string, ret string) AttemptForm {
	f := NewForm(id)
	f.Review, f.Return = true, ret
	return f
}
func EditForm(a Attempt) AttemptForm {
	return AttemptForm{ID: a.ID, Revision: a.Revision, Outcome: a.Outcome, Minutes: Count(a.Minutes), Assisted: a.Assisted, Notes: a.Notes, WantsReview: a.WantsReview, Approach: a.Approach, Time: NewComplexityInput(a.TimeComplexity), Space: NewComplexityInput(a.SpaceComplexity)}
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
