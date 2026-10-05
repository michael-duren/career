package scheduler

import "time"

type DayInterval struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	NextDay bool   `json:"nextDay"`
}
type Settings struct {
	Initialized bool                   `json:"initialized,omitempty"`
	TimeZone    string                 `json:"timeZone"`
	DefaultDay  DayInterval            `json:"defaultDay"`
	Weekdays    map[string]DayInterval `json:"weekdays"`
	Dates       map[string]DayInterval `json:"dates"`
}
type Step struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}
type Pause struct {
	From string `json:"from"`
	To   string `json:"to"`
}
type Goal struct {
	Pauses           []Pause  `json:"pauses,omitempty"`
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Color            string   `json:"color"`
	StartDate        string   `json:"startDate"`
	EndDate          string   `json:"endDate"`
	Status           string   `json:"status"`
	DailyHours       *float64 `json:"dailyHours"`
	SelectedWeekdays []int    `json:"selectedWeekdays"`
	Steps            []Step   `json:"steps"`
	DependsOn        []string `json:"dependsOn"`
	EligibleFrom     string   `json:"eligibleFrom,omitempty"`
	StoppedDate      string   `json:"stoppedDate,omitempty"`
}
type Assignment struct {
	GoalID    string `json:"goalId,omitempty"`
	StepID    string `json:"stepId,omitempty"`
	Title     string `json:"title"`
	GoalTitle string `json:"goalTitle,omitempty"`
	Color     string `json:"color,omitempty"`
}
type Plan struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
type Actual struct {
	Status          string    `json:"status"`
	Date            string    `json:"date"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	OutsideTimeline bool      `json:"outsideTimeline,omitempty"`
}
type Session struct {
	OccurrenceDate string     `json:"occurrenceDate,omitempty"`
	ID             string     `json:"id"`
	RuleID         string     `json:"ruleId,omitempty"`
	Date           string     `json:"date"`
	Assignment     Assignment `json:"assignment"`
	Plan           *Plan      `json:"plan"`
	Actual         *Actual    `json:"actual"`
	State          string     `json:"state"`
	Attention      string     `json:"attention,omitempty"`
	ConflictIDs    []string   `json:"conflictIds,omitempty"`
	Exception      bool       `json:"exception"`
}
type Rule struct {
	ID              string     `json:"id"`
	Weekday         int        `json:"weekday"`
	LocalStart      string     `json:"localStart"`
	DurationMinutes int        `json:"durationMinutes"`
	EffectiveFrom   string     `json:"effectiveFrom"`
	EffectiveTo     string     `json:"effectiveTo,omitempty"`
	Assignment      Assignment `json:"assignment"`
}
type Busy struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
type Summary struct {
	Goal                    Goal     `json:"goal"`
	RequiredHours           *float64 `json:"requiredHours"`
	ActualHours             float64  `json:"actualHours"`
	RemainingScheduledHours float64  `json:"remainingScheduledHours"`
	UncoveredHours          float64  `json:"uncoveredHours"`
	ExcessHours             float64  `json:"excessHours"`
	UnscheduledStepIDs      []string `json:"unscheduledStepIds"`
}
type Day struct {
	Valid    bool        `json:"valid"`
	Reason   string      `json:"reason,omitempty"`
	Date     string      `json:"date"`
	Interval DayInterval `json:"interval"`
	Start    time.Time   `json:"start"`
	End      time.Time   `json:"end"`
}
type Warning struct {
	Message   string `json:"message"`
	Date      string `json:"date,omitempty"`
	GoalID    string `json:"goalId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}
type Week struct {
	WarningTargets         []Warning `json:"warningTargets"`
	Revision               string    `json:"revision"`
	Week                   string    `json:"week"`
	Settings               Settings  `json:"settings"`
	Days                   []Day     `json:"days"`
	Goals                  []Summary `json:"goals"`
	Sessions               []Session `json:"sessions"`
	Rules                  []Rule    `json:"rules"`
	Busy                   []Busy    `json:"busy"`
	Warnings               []string  `json:"warnings"`
	RemainingCapacityHours float64   `json:"remainingCapacityHours"`
}
type Document struct {
	Busy     []Busy             `json:"busy,omitempty"`
	Revision string             `json:"revision"`
	Settings Settings           `json:"settings"`
	Goals    map[string]Goal    `json:"goals"`
	Sessions map[string]Session `json:"sessions"`
	Rules    map[string]Rule    `json:"rules"`
	// ClosedWeeks and LastDate track reconciliation for closed-week snapshots.
	ClosedWeeks map[string][]Goal `json:"closedWeeks"`
	LastDate    string            `json:"lastDate"`
}

// All writes use POST /api/scheduler/mutate, including settings, rule, session,
// cancel and actual. Week identifies the response week. ID is optional for creates.
type Mutation struct {
	Revision      string    `json:"revision"`
	Week          string    `json:"week"`
	Action        string    `json:"action"`
	ID            string    `json:"id,omitempty"`
	Settings      *Settings `json:"settings,omitempty"`
	Rule          *Rule     `json:"rule,omitempty"`
	Session       *Session  `json:"session,omitempty"`
	Actual        *Actual   `json:"actual,omitempty"`
	EffectiveFrom string    `json:"effectiveFrom,omitempty"`
	// Scope applies to "cancel": "future" ends the occurrence's rule the day
	// before it and removes later, not-yet-started occurrences of that rule.
	// Anything else (including empty) cancels only this occurrence.
	Scope string `json:"scope,omitempty"`
	// NewStep creates a subgoal on the assignment's goal in the same
	// transaction as the save, and assigns the session or rule to it.
	NewStep *NewStep `json:"newStep,omitempty"`
}
type NewStep struct {
	Title string `json:"title"`
}
type Conflict struct {
	Message string   `json:"error"`
	IDs     []string `json:"conflictIds"`
}

func (e *Conflict) Error() string { return e.Message }
