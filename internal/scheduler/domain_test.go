package scheduler

import (
	"testing"
	"time"
)

func TestRequirements(t *testing.T) {
	h := 1.
	g := Goal{StartDate: "2026-09-28", EndDate: "2026-10-04", DailyHours: &h}
	if v := Required(g, "2026-09-28"); v == nil || *v != 7 {
		t.Fatalf("all weekdays: %v", v)
	}
	g.SelectedWeekdays = []int{1, 3, 5}
	if *Required(g, "2026-09-28") != 3 {
		t.Fatal("selected weekdays")
	}
	g.SelectedWeekdays = []int{}
	if *Required(g, "2026-09-28") != 0 {
		t.Fatal("empty weekdays")
	}
	g.SelectedWeekdays = nil
	g.StartDate = "2026-10-01"
	if *Required(g, "2026-09-28") != 4 {
		t.Fatal("partial week")
	}
}
func TestLocalDST(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	if _, err := Local("2026-03-08", "02:30", loc); err == nil {
		t.Fatal("gap accepted")
	}
	got, err := Local("2026-11-01", "01:30", loc)
	if err != nil || got.UTC().Format(time.RFC3339) != "2026-11-01T06:30:00Z" {
		t.Fatalf("earlier fold: %v %v", got, err)
	}
}
func instant(s string) time.Time { v, _ := time.Parse(time.RFC3339, s); return v }
func testDocument() Document {
	d := New()
	h := 1.
	d.Goals["goal"] = Goal{ID: "goal", Title: "Goal", StartDate: "2026-09-28", EndDate: "2026-12-31", DailyHours: &h}
	return d
}
func TestAccountingActualReplacement(t *testing.T) {
	d := testDocument()
	now := instant("2026-09-30T12:00:00Z")
	d.Sessions["a"] = Session{ID: "a", Date: "2026-09-28", Assignment: Assignment{GoalID: "goal"}, State: "accepted", Plan: &Plan{Start: instant("2026-09-28T06:00:00Z"), End: instant("2026-09-28T08:00:00Z")}}
	d.Sessions["b"] = Session{ID: "b", Date: "2026-10-03", Assignment: Assignment{GoalID: "goal"}, State: "accepted", Plan: &Plan{Start: instant("2026-10-03T06:00:00Z"), End: instant("2026-10-03T10:00:00Z")}}
	d.Finalize(now)
	sum := d.Week("2026-09-28", now, nil).Goals[0]
	if sum.ActualHours != 2 || sum.RemainingScheduledHours != 4 || sum.UncoveredHours != 1 {
		t.Fatalf("accounting: %+v", sum)
	}
	a := Actual{Status: "explicit", Date: "2026-09-28", Start: instant("2026-09-28T06:00:00Z"), End: instant("2026-09-28T09:00:00Z")}
	if e := d.Apply(Mutation{Action: "actual", ID: "a", Actual: &a}, now, nil); e != nil {
		t.Fatal(e)
	}
	sum = d.Week("2026-09-28", now, nil).Goals[0]
	if sum.ActualHours != 3 || sum.UncoveredHours != 0 {
		t.Fatalf("actual replacement %+v", sum)
	}
	a.Status = "skipped"
	_ = d.Apply(Mutation{Action: "actual", ID: "a", Actual: &a}, now, nil)
	sum = d.Week("2026-09-28", now, nil).Goals[0]
	if sum.ActualHours != 0 || sum.UncoveredHours != 3 {
		t.Fatalf("skip %+v", sum)
	}
	if d.Sessions["a"].Plan.End != instant("2026-09-28T08:00:00Z") {
		t.Fatal("original plan changed")
	}
}
func TestRecurrenceIdentityExceptionAndCatchup(t *testing.T) {
	d := testDocument()
	now := instant("2026-09-28T00:00:00Z")
	d.Rules["r"] = Rule{ID: "r", Weekday: 1, LocalStart: "06:00", DurationMinutes: 60, EffectiveFrom: "2026-09-28", Assignment: Assignment{GoalID: "goal"}}
	d.Generate("2026-09-28", "2026-10-05", now, nil, "")
	d.Generate("2026-09-28", "2026-10-05", now, nil, "")
	if len(d.Sessions) != 2 {
		t.Fatal("duplicate occurrences")
	}
	s := d.Sessions["r:2026-10-05"]
	s.Plan.End = s.Plan.End.Add(time.Hour)
	if e := d.Apply(Mutation{Action: "session", ID: s.ID, Session: &s}, now, nil); e != nil {
		t.Fatal(e)
	}
	d.Generate("2026-09-28", "2026-10-05", now, nil, "")
	if d.Sessions[s.ID].Plan.End.Sub(d.Sessions[s.ID].Plan.Start) != 2*time.Hour {
		t.Fatal("exception overwritten")
	}
	d.Generate("2026-09-28", "2026-10-05", instant("2026-10-06T00:00:00Z"), nil, "")
	if d.Sessions[s.ID].Actual == nil {
		t.Fatal("catchup did not assume actual")
	}
}
func TestGenerateDoesNotReviveSessionsBeforeLastReconcile(t *testing.T) {
	d := testDocument()
	g := d.Goals["goal"]
	g.StartDate = "2026-10-01"
	d.Goals["goal"] = g
	d.Rules["r"] = Rule{ID: "r", Weekday: 1, LocalStart: "06:00", DurationMinutes: 60, EffectiveFrom: "2026-09-28", Assignment: Assignment{GoalID: "goal"}}
	now := instant("2026-10-06T00:00:00Z")
	sinceDate := "2026-10-01"
	d.Generate("2026-09-28", "2026-10-05", now, nil, sinceDate)
	if _, ok := d.Sessions["r:2026-10-05"]; !ok {
		t.Fatal("occurrence on or after the watermark was not generated")
	}
	if _, ok := d.Sessions["r:2026-09-28"]; ok {
		t.Fatal("occurrence before the goal's start date should not exist yet")
	}
	g.StartDate = "2026-09-01"
	d.Goals["goal"] = g
	d.Generate("2026-09-28", "2026-10-05", now, nil, sinceDate)
	if _, ok := d.Sessions["r:2026-09-28"]; ok {
		t.Fatal("retroactive eligibility fabricated a session before the reconciled watermark")
	}
}
func TestDayOvernightAndConflicts(t *testing.T) {
	d := testDocument()
	d.Settings.DefaultDay = DayInterval{Start: "09:00", End: "00:00", NextDay: true}
	if e := d.Settings.Validate(); e != nil {
		t.Fatal(e)
	}
	d.Settings.Weekdays["1"] = DayInterval{Start: "09:00", End: "10:00", NextDay: true}
	if e := d.Settings.Validate(); e == nil {
		t.Fatal("overlapping days allowed")
	}
	d.Settings.Weekdays = map[string]DayInterval{}
	now := instant("2026-09-28T00:00:00Z")
	s := Session{Date: "2026-09-28", Assignment: Assignment{GoalID: "goal"}, Plan: &Plan{Start: instant("2026-09-28T10:00:00Z"), End: instant("2026-09-28T11:00:00Z")}}
	if e := d.Apply(Mutation{Action: "session", Session: &s}, now, nil); e != nil {
		t.Fatal(e)
	}
	if e := d.Apply(Mutation{Action: "session", Session: &s}, now, nil); e == nil {
		t.Fatal("overlap allowed")
	}
}
func TestClosedRequirementAndHistoricalAttribution(t *testing.T) {
	d := testDocument()
	d.LastDate = "2026-09-28"
	g := d.Goals["goal"]
	h := 2.
	g.DailyHours = &h
	d.Reconcile([]Goal{g}, instant("2026-10-05T00:00:00Z"))
	if *d.Week("2026-09-28", instant("2026-10-05T00:00:00Z"), nil).Goals[0].RequiredHours != 7 {
		t.Fatal("closed requirement changed")
	}
	if *d.Week("2026-10-05", instant("2026-10-05T00:00:00Z"), nil).Goals[0].RequiredHours != 14 {
		t.Fatal("current requirement stale")
	}
}
func TestEffectiveRuleEditPreservesException(t *testing.T) {
	d := testDocument()
	now := instant("2026-09-28T00:00:00Z")
	r := Rule{ID: "r", Weekday: 1, LocalStart: "06:00", DurationMinutes: 60, EffectiveFrom: "2026-09-28", Assignment: Assignment{GoalID: "goal"}}
	d.Rules[r.ID] = r
	d.Generate("2026-09-28", "2026-10-12", now, nil, "")
	s := d.Sessions["r:2026-10-05"]
	s.Plan.End = s.Plan.End.Add(time.Hour)
	if e := d.Apply(Mutation{Action: "session", ID: s.ID, Session: &s}, now, nil); e != nil {
		t.Fatal(e)
	}
	r.DurationMinutes = 90
	if e := d.Apply(Mutation{Action: "rule", ID: "r", EffectiveFrom: "2026-10-05", Rule: &r}, now, nil); e != nil {
		t.Fatal(e)
	}
	d.Generate("2026-09-28", "2026-10-12", now, nil, "")
	count := 0
	for _, session := range d.Sessions {
		if session.Date == "2026-10-05" {
			count++
			if session.ID != s.ID || session.Plan.End.Sub(session.Plan.Start) != 2*time.Hour {
				t.Fatalf("exception changed: %+v", session)
			}
		}
	}
	if count != 1 {
		t.Fatalf("date exception duplicated: %d", count)
	}
}
func TestStopReopenRetainsEarlierRequirement(t *testing.T) {
	d := testDocument()
	g := d.Goals["goal"]
	g.Status = "done"
	d.Reconcile([]Goal{g}, instant("2026-09-30T12:00:00Z"))
	if *Required(d.Goals["goal"], "2026-09-28") != 3 {
		t.Fatal("stop transition lost elapsed requirement")
	}
	g.Status = "active"
	d.Reconcile([]Goal{g}, instant("2026-10-02T12:00:00Z"))
	if got := *Required(d.Goals["goal"], "2026-09-28"); got != 6 {
		t.Fatalf("reopening should omit Thursday but retain Monday-Wednesday: %v", got)
	}
}
func TestOvernightActualAttributionAndCapacityUnion(t *testing.T) {
	d := testDocument()
	d.Settings.DefaultDay = DayInterval{Start: "09:00", End: "02:00", NextDay: true}
	now := instant("2026-10-06T12:00:00Z")
	a := Actual{Status: "explicit", Date: "2026-10-05", Start: instant("2026-10-05T01:00:00Z"), End: instant("2026-10-05T01:30:00Z")}
	s := Session{Date: a.Date, Assignment: Assignment{GoalID: "goal"}}
	if e := d.Apply(Mutation{Action: "actual", Session: &s, Actual: &a}, now, nil); e == nil {
		t.Fatal("post-midnight actual assigned wrong week")
	}
	a.Date = "2026-10-04"
	s.Date = a.Date
	if e := d.Apply(Mutation{Action: "actual", Session: &s, Actual: &a}, now, nil); e != nil {
		t.Fatal(e)
	}
	if got := d.Week("2026-09-28", now, nil).Goals[0].ActualHours; got != .5 {
		t.Fatalf("overnight actual contribution %v", got)
	}
	d.Settings.DefaultDay = DayInterval{Start: "09:00", End: "10:00"}
	busy := []Busy{{Start: instant("2026-09-28T09:00:00Z"), End: instant("2026-09-28T09:45:00Z")}, {Start: instant("2026-09-28T09:15:00Z"), End: instant("2026-09-28T10:00:00Z")}}
	if got := d.Week("2026-09-28", instant("2026-09-28T00:00:00Z"), busy).RemainingCapacityHours; got != 6 {
		t.Fatalf("overlapping busy counted twice: %v", got)
	}
}
func TestSubgoalMoveRetainsHistoricalParent(t *testing.T) {
	d := testDocument()
	old := d.Goals["goal"]
	old.Steps = []Step{{ID: "step", Title: "Step"}}
	d.Goals[old.ID] = old
	next := old
	next.ID = "next"
	next.Title = "Next"
	d.Goals[next.ID] = next
	d.Rules["r"] = Rule{ID: "r", Weekday: 1, LocalStart: "06:00", DurationMinutes: 60, EffectiveFrom: "2026-09-28", Assignment: Assignment{GoalID: old.ID, StepID: "step"}}
	d.Generate("2026-09-28", "2026-10-05", instant("2026-09-28T00:00:00Z"), nil, "")
	old.Steps = nil
	d.Reconcile([]Goal{old, next}, instant("2026-09-29T00:00:00Z"))
	if d.Sessions["r:2026-09-28"].Assignment.GoalID != old.ID || d.Sessions["r:2026-10-05"].Assignment.GoalID != next.ID || d.Rules["r"].Assignment.GoalID != next.ID {
		t.Fatal("subgoal history or future attribution incorrect")
	}
}
func TestImportValidationNormalizesAndRejectsBadHistory(t *testing.T) {
	d := New()
	d.Sessions = nil
	d.Goals = nil
	if e := d.Validate(); e != nil {
		t.Fatal(e)
	}
	if d.Sessions == nil || d.Goals == nil {
		t.Fatal("nil maps remain")
	}
	d.Sessions["bad"] = Session{ID: "bad", Date: "invalid", State: "accepted"}
	if e := d.Validate(); e == nil {
		t.Fatal("invalid historical date accepted")
	}
}
func TestCancelingPlanlessOccurrenceRoundTripsThroughImport(t *testing.T) {
	d := testDocument()
	now := instant("2026-09-28T00:00:00Z")
	d.Sessions["s"] = Session{ID: "s", RuleID: "r", OccurrenceDate: "2026-09-28", Date: "2026-09-28", Assignment: Assignment{GoalID: "goal"}, State: "attention", Attention: "spring-forward gap"}
	d.Rules["r"] = Rule{ID: "r", Weekday: 1, LocalStart: "02:30", DurationMinutes: 60, EffectiveFrom: "2026-09-28", Assignment: Assignment{GoalID: "goal"}}
	if e := d.Apply(Mutation{Action: "cancel", ID: "s"}, now, nil); e != nil {
		t.Fatal(e)
	}
	if d.Sessions["s"].State != "canceled" {
		t.Fatal("session not canceled")
	}
	if e := d.Validate(); e != nil {
		t.Fatalf("canceled plan-less session should import: %v", e)
	}
}
func TestCancelRemovesUnplannedActualButPreservesRecurringHistory(t *testing.T) {
	d := testDocument()
	now := instant("2026-09-29T00:00:00Z")
	d.Sessions["manual"] = Session{ID: "manual", Date: "2026-09-28", Assignment: Assignment{GoalID: "goal"}, State: "accepted", Actual: &Actual{Status: "explicit", Date: "2026-09-28", Start: instant("2026-09-28T06:00:00Z"), End: instant("2026-09-28T07:00:00Z")}}
	if got := d.Week("2026-09-28", now, nil).Goals[0].ActualHours; got != 1 {
		t.Fatalf("actual not counted before cancel: %v", got)
	}
	if e := d.Apply(Mutation{Action: "cancel", ID: "manual"}, now, nil); e != nil {
		t.Fatal(e)
	}
	if _, ok := d.Sessions["manual"]; ok {
		t.Fatal("unplanned actual should be removed outright")
	}
	if got := d.Week("2026-09-28", now, nil).Goals[0].ActualHours; got != 0 {
		t.Fatalf("canceled unplanned actual still counted: %v", got)
	}
	d.Sessions["r:2026-09-28"] = Session{ID: "r:2026-09-28", RuleID: "r", Date: "2026-09-28", Assignment: Assignment{GoalID: "goal"}, State: "accepted", Actual: &Actual{Status: "explicit", Date: "2026-09-28", Start: instant("2026-09-28T06:00:00Z"), End: instant("2026-09-28T07:00:00Z")}}
	if e := d.Apply(Mutation{Action: "cancel", ID: "r:2026-09-28"}, now, nil); e == nil {
		t.Fatal("recorded recurring actual work should not be cancelable")
	}
}
func TestAssumedActualRejectedBeforePlannedEnd(t *testing.T) {
	d := testDocument()
	now := instant("2026-09-28T06:30:00Z")
	d.Sessions["s"] = Session{ID: "s", Date: "2026-09-28", Assignment: Assignment{GoalID: "goal"}, State: "accepted", Plan: &Plan{Start: instant("2026-09-28T06:00:00Z"), End: instant("2026-09-28T08:00:00Z")}}
	a := Actual{Status: "assumed"}
	if e := d.Apply(Mutation{Action: "actual", ID: "s", Actual: &a}, now, nil); e == nil {
		t.Fatal("assumed actual accepted before the planned session ended")
	}
	if e := d.Apply(Mutation{Action: "actual", ID: "s", Actual: &a}, instant("2026-09-28T08:00:01Z"), nil); e != nil {
		t.Fatalf("assumed actual rejected after the planned session ended: %v", e)
	}
}
