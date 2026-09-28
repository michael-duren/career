package scheduler

import (
	"fmt"
	"github.com/google/uuid"
	"sort"
	"strconv"
	"time"
)

const dateLayout = "2006-01-02"

func DateAdd(d string, n int) string {
	t, e := time.Parse(dateLayout, d)
	if e != nil {
		return ""
	}
	return t.AddDate(0, 0, n).Format(dateLayout)
}
func ValidDate(d string) bool {
	t, e := time.Parse(dateLayout, d)
	return e == nil && t.Format(dateLayout) == d
}
func Monday(d string) string {
	t, e := time.Parse(dateLayout, d)
	if e != nil {
		return ""
	}
	return t.AddDate(0, 0, -(int(t.Weekday())+6)%7).Format(dateLayout)
}
func New() Document {
	return Document{Revision: uuid.NewString(), Settings: Settings{TimeZone: "UTC", DefaultDay: DayInterval{Start: "05:00", End: "20:30"}, Weekdays: map[string]DayInterval{}, Dates: map[string]DayInterval{}}, Goals: map[string]Goal{}, Sessions: map[string]Session{}, Rules: map[string]Rule{}, ClosedWeeks: map[string][]Goal{}}
}
func minutes(s string) (int, error) {
	t, e := time.Parse("15:04", s)
	if e != nil {
		return 0, fmt.Errorf("invalid local time %q", s)
	}
	return t.Hour()*60 + t.Minute(), nil
}

// Local explicitly finds matching instants, including both sides of a DST fold.
func Local(d, s string, loc *time.Location) (time.Time, error) {
	wall, e := time.Parse(dateLayout+" 15:04", d+" "+s)
	if e != nil {
		return time.Time{}, e
	}
	offsets := map[int]bool{}
	for h := -36; h <= 36; h++ {
		_, o := wall.Add(time.Duration(h) * time.Hour).In(loc).Zone()
		offsets[o] = true
	}
	var first time.Time
	for o := range offsets {
		candidate := wall.Add(-time.Duration(o) * time.Second)
		if candidate.In(loc).Format(dateLayout+" 15:04") == d+" "+s && (first.IsZero() || candidate.Before(first)) {
			first = candidate
		}
	}
	if first.IsZero() {
		return first, fmt.Errorf("%s %s does not exist in %s (daylight-saving gap)", d, s, loc)
	}
	return first, nil
}
func (s Settings) Interval(d string) DayInterval {
	if x, ok := s.Dates[d]; ok {
		return x
	}
	t, _ := time.Parse(dateLayout, d)
	if x, ok := s.Weekdays[strconv.Itoa((int(t.Weekday())+6)%7+1)]; ok {
		return x
	}
	return s.DefaultDay
}
func (s Settings) Day(d string) (Day, error) {
	loc, e := time.LoadLocation(s.TimeZone)
	if e != nil {
		return Day{}, fmt.Errorf("invalid IANA time zone")
	}
	i := s.Interval(d)
	a, e := Local(d, i.Start, loc)
	if e != nil {
		return Day{}, e
	}
	endDate := d
	if i.NextDay {
		endDate = DateAdd(d, 1)
	}
	b, e := Local(endDate, i.End, loc)
	if e != nil {
		return Day{}, e
	}
	return Day{Date: d, Interval: i, Start: a, End: b}, nil
}
func (s Settings) Validate() error {
	if _, e := time.LoadLocation(s.TimeZone); e != nil {
		return fmt.Errorf("invalid IANA time zone")
	}
	check := func(i DayInterval) error {
		a, e := minutes(i.Start)
		if e != nil {
			return e
		}
		b, e := minutes(i.End)
		if e != nil {
			return e
		}
		if i.NextDay {
			b += 1440
		}
		if b <= a || b-a > 1440 {
			return fmt.Errorf("day must be greater than zero and at most 24 wall-clock hours; choose next day explicitly")
		}
		return nil
	}
	if e := check(s.DefaultDay); e != nil {
		return e
	}
	for k, i := range s.Weekdays {
		n, e := strconv.Atoi(k)
		if e != nil || n < 1 || n > 7 {
			return fmt.Errorf("invalid weekday")
		}
		if e := check(i); e != nil {
			return e
		}
	}
	dates := map[string]bool{}
	for n := 0; n < 8; n++ {
		dates[DateAdd("2026-01-05", n)] = true
	}
	for d, i := range s.Dates {
		if !ValidDate(d) {
			return fmt.Errorf("invalid date override")
		}
		if e := check(i); e != nil {
			return e
		}
		dates[d] = true
		dates[DateAdd(d, -1)] = true
	}
	for d := range dates {
		a := s.Interval(d)
		b := s.Interval(DateAdd(d, 1))
		end, _ := minutes(a.End)
		start, _ := minutes(b.Start)
		if a.NextDay && end > start {
			return fmt.Errorf("day intervals overlap after %s", d)
		}
	}
	return nil
}
func Eligible(g Goal, d string) bool {
	for _, p := range g.Pauses {
		if d >= p.From && d <= p.To {
			return false
		}
	}
	return d >= g.StartDate && d <= g.EndDate && (g.EligibleFrom == "" || d >= g.EligibleFrom) && (g.StoppedDate == "" || d <= g.StoppedDate)
}
func Required(g Goal, w string) *float64 {
	if g.DailyHours == nil {
		return nil
	}
	n := 0
	for i := 0; i < 7; i++ {
		d := DateAdd(w, i)
		if !Eligible(g, d) {
			continue
		}
		selected := g.SelectedWeekdays == nil
		for _, v := range g.SelectedWeekdays {
			if v == i+1 {
				selected = true
			}
		}
		if selected {
			n++
		}
	}
	v := *g.DailyHours * float64(n)
	return &v
}
func overlap(a, b Plan) bool { return a.Start.Before(b.End) && b.Start.Before(a.End) }
func (d *Document) assignment(a Assignment, date string) (Assignment, error) {
	if a.GoalID == "" {
		if a.StepID != "" {
			return a, fmt.Errorf("subgoal requires a parent goal")
		}
		if a.Title == "" {
			return a, fmt.Errorf("fixed commitment needs a title")
		}
		return a, nil
	}
	g, ok := d.Goals[a.GoalID]
	if !ok || !Eligible(g, date) || g.Status == "done" || g.Status == "dropped" {
		return a, fmt.Errorf("goal is not eligible on %s", date)
	}
	a.GoalTitle = g.Title
	a.Color = g.Color
	a.Title = g.Title
	if a.StepID != "" {
		found := false
		for _, s := range g.Steps {
			if s.ID == a.StepID && !s.Completed {
				a.Title = s.Title
				found = true
			}
		}
		if !found {
			return a, fmt.Errorf("subgoal is completed or deleted")
		}
	}
	return a, nil
}
func (d *Document) validateSession(s Session, busy []Busy) error {
	if !ValidDate(s.Date) || s.Plan == nil || !s.Plan.End.After(s.Plan.Start) {
		return fmt.Errorf("valid date and positive planned interval required")
	}
	day, e := d.Settings.Day(s.Date)
	if e != nil {
		return e
	}
	if s.Plan.Start.Before(day.Start) || s.Plan.End.After(day.End) {
		return fmt.Errorf("session must fit the configured day")
	}
	ids := []string{}
	for id, x := range d.Sessions {
		if id != s.ID && x.State == "accepted" && x.Plan != nil && overlap(*s.Plan, *x.Plan) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, x := range busy {
		if overlap(*s.Plan, Plan{Start: x.Start, End: x.End}) {
			ids = append(ids, x.ID)
		}
	}
	if len(ids) > 0 {
		return &Conflict{Message: "Reservation conflicts with another block", IDs: ids}
	}
	return nil
}
func (d *Document) Finalize(now time.Time) {
	ids := d.sessionIDs()
	for _, id := range ids {
		s := d.Sessions[id]
		if s.Assignment.GoalID == "" || s.Plan == nil || s.Actual != nil || s.State != "accepted" || s.Plan.End.After(now) {
			continue
		}
		a := Actual{Status: "assumed", Date: s.Date, Start: s.Plan.Start, End: s.Plan.End}
		if e := d.validateActual(id, a); e != nil {
			s.State = "attention"
			s.Attention = "Actual time overlaps recorded work"
		} else {
			s.Actual = &a
		}
		d.Sessions[id] = s
	}
}
func (d *Document) sessionIDs() []string {
	ids := make([]string, 0, len(d.Sessions))
	for id := range d.Sessions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := d.Sessions[ids[i]], d.Sessions[ids[j]]
		if a.Plan != nil && b.Plan != nil && !a.Plan.Start.Equal(b.Plan.Start) {
			return a.Plan.Start.Before(b.Plan.Start)
		}
		return ids[i] < ids[j]
	})
	return ids
}
func (d *Document) validateActual(id string, a Actual) error {
	if a.Status == "skipped" {
		return nil
	}
	if a.Status != "explicit" && a.Status != "assumed" {
		return fmt.Errorf("actual status must be explicit, assumed or skipped")
	}
	if !ValidDate(a.Date) || !a.End.After(a.Start) {
		return fmt.Errorf("valid actual date and positive interval required")
	}
	for key, s := range d.Sessions {
		if key != id && s.Actual != nil && s.Actual.Status != "skipped" && overlap(Plan{Start: a.Start, End: a.End}, Plan{Start: s.Actual.Start, End: s.Actual.End}) {
			return &Conflict{Message: "Actual work overlaps another actual session", IDs: []string{key}}
		}
	}
	return nil
}
func (d *Document) Reconcile(goals []Goal, now time.Time) {
	loc, _ := time.LoadLocation(d.Settings.TimeZone)
	today := now.In(loc).Format(dateLayout)
	current := Monday(today)
	if d.LastDate != "" {
		for w := Monday(d.LastDate); w < current; w = DateAdd(w, 7) {
			if _, ok := d.ClosedWeeks[w]; !ok {
				snapshot := []Goal{}
				for _, g := range d.Goals {
					snapshot = append(snapshot, g)
				}
				d.ClosedWeeks[w] = snapshot
			}
		}
	}
	d.LastDate = today
	d.Finalize(now)
	next := map[string]Goal{}
	stepParents := map[string]string{}
	for _, g := range goals {
		old, ok := d.Goals[g.ID]
		if ok {
			g.EligibleFrom = old.EligibleFrom
			g.StoppedDate = old.StoppedDate
			g.Pauses = old.Pauses
		}
		stopped := g.Status == "done" || g.Status == "dropped"
		wasStopped := ok && (old.Status == "done" || old.Status == "dropped")
		if stopped && !wasStopped {
			g.StoppedDate = today
		}
		if !stopped && wasStopped {
			if old.StoppedDate != "" && DateAdd(old.StoppedDate, 1) < today {
				g.Pauses = append(g.Pauses, Pause{From: DateAdd(old.StoppedDate, 1), To: DateAdd(today, -1)})
			}
			g.StoppedDate = ""
		}
		next[g.ID] = g
		for _, s := range g.Steps {
			stepParents[s.ID] = g.ID
		}
	}
	for id, g := range d.Goals {
		if _, ok := next[id]; !ok {
			if g.StoppedDate == "" {
				g.StoppedDate = today
			}
			g.Status = "dropped"
			next[id] = g
		}
	}
	d.Goals = next
	for id, r := range d.Rules {
		if p, ok := stepParents[r.Assignment.StepID]; ok {
			r.Assignment.GoalID = p
			d.Rules[id] = r
		}
	}
	for id, s := range d.Sessions {
		if s.Plan == nil || !s.Plan.Start.After(now) || s.State == "canceled" {
			continue
		}
		if p, ok := stepParents[s.Assignment.StepID]; ok {
			s.Assignment.GoalID = p
		}
		a, e := d.assignment(s.Assignment, s.Date)
		if e != nil {
			s.State = "canceled"
			s.Attention = e.Error()
		} else {
			s.Assignment = a
		}
		d.Sessions[id] = s
	}
}
// Generate creates sessions for each active rule occurrence between from and
// to. sinceDate is the date this document was last reconciled through; a rule
// occurrence with no existing session is skipped when its date is earlier
// than sinceDate, so a goal or subgoal becoming eligible again (an extended
// end date, an unfinished subgoal) never fabricates sessions, and actuals,
// in weeks already closed out. An empty sinceDate (never reconciled) leaves
// the full range unrestricted.
func (d *Document) Generate(from, to string, now time.Time, busy []Busy, sinceDate string) {
	loc, _ := time.LoadLocation(d.Settings.TimeZone)
	rules := []string{}
	for id := range d.Rules {
		rules = append(rules, id)
	}
	sort.Strings(rules)
	for date := from; date <= to; date = DateAdd(date, 1) {
		t, _ := time.Parse(dateLayout, date)
		weekday := (int(t.Weekday())+6)%7 + 1
		for _, id := range rules {
			r := d.Rules[id]
			if date < r.EffectiveFrom || (r.EffectiveTo != "" && date > r.EffectiveTo) || weekday != r.Weekday {
				continue
			}
			excepted := false
			for _, existing := range d.Sessions {
				if existing.RuleID == r.ID && existing.Exception && (existing.OccurrenceDate == date || existing.OccurrenceDate == "" && existing.Date == date) {
					excepted = true
					break
				}
			}
			if excepted {
				continue
			}
			sid := r.ID + ":" + date
			old, exists := d.Sessions[sid]
			if exists && (old.Exception || old.Plan != nil && !old.Plan.Start.After(now)) {
				continue
			}
			if !exists && sinceDate != "" && date < sinceDate {
				continue
			}
			a, e := d.assignment(r.Assignment, date)
			if e != nil {
				continue
			}
			s := Session{ID: sid, RuleID: r.ID, OccurrenceDate: date, Date: date, Assignment: a, State: "accepted", ConflictIDs: []string{}}
			start, e := Local(date, r.LocalStart, loc)
			if e == nil {
				day := d.Settings.Interval(date)
				m, _ := minutes(r.LocalStart)
				dayStart, _ := minutes(day.Start)
				if day.NextDay && m < dayStart {
					start, e = Local(DateAdd(date, 1), r.LocalStart, loc)
				}
			}
			if e != nil {
				s.State = "attention"
				s.Attention = e.Error()
			} else {
				s.Plan = &Plan{Start: start, End: start.Add(time.Duration(r.DurationMinutes) * time.Minute)}
			}
			if s.Plan != nil {
				if err := d.validateSession(s, busy); err != nil {
					s.State = "attention"
					s.Attention = err.Error()
					if c, ok := err.(*Conflict); ok {
						s.ConflictIDs = c.IDs
					}
				}
			}
			d.Sessions[sid] = s
		}
	}
	d.Revalidate(now, busy)
	d.Finalize(now)
}
func (d *Document) Revalidate(now time.Time, busy []Busy) {
	ids := d.sessionIDs()
	sort.SliceStable(ids, func(i, j int) bool {
		return d.Sessions[ids[i]].State == "accepted" && d.Sessions[ids[j]].State != "accepted"
	})
	for _, id := range ids {
		s := d.Sessions[id]
		if s.State == "canceled" || s.Plan == nil || !s.Plan.Start.After(now) {
			continue
		}
		s.State = "attention"
		d.Sessions[id] = s
	}
	for _, id := range ids {
		s := d.Sessions[id]
		if s.State == "canceled" || s.Plan == nil || !s.Plan.Start.After(now) {
			continue
		}
		s.State = "accepted"
		s.Attention = ""
		s.ConflictIDs = []string{}
		if e := d.validateSession(s, busy); e != nil {
			s.State = "attention"
			s.Attention = e.Error()
			if c, ok := e.(*Conflict); ok {
				s.ConflictIDs = c.IDs
			}
		}
		d.Sessions[id] = s
	}
}
func (d *Document) Apply(m Mutation, now time.Time, busy []Busy) error {
	if m.Week != "" {
		date := ""
		if m.Action == "session" && m.Session != nil {
			date = m.Session.Date
		}
		if m.Action == "rule" && m.Rule != nil {
			date = m.Rule.EffectiveFrom
			if m.ID != "" {
				date = m.EffectiveFrom
			}
		}
		if date != "" && Monday(date) != m.Week {
			return fmt.Errorf("planned date must belong to the requested week")
		}
	}
	switch m.Action {
	case "settings":
		if m.Settings == nil {
			return fmt.Errorf("settings required")
		}
		if e := m.Settings.Validate(); e != nil {
			return e
		}
		d.Settings = *m.Settings
		d.Settings.Initialized = true
	case "rule":
		if m.Rule == nil {
			return fmt.Errorf("rule required")
		}
		r := *m.Rule
		if r.Weekday < 1 || r.Weekday > 7 || r.DurationMinutes < 1 || r.DurationMinutes > 1440 || !ValidDate(r.EffectiveFrom) || (r.EffectiveTo != "" && (!ValidDate(r.EffectiveTo) || r.EffectiveTo < r.EffectiveFrom)) {
			return fmt.Errorf("valid weekday, duration and effective dates required")
		}
		if _, e := minutes(r.LocalStart); e != nil {
			return e
		}
		if m.ID != "" {
			old, ok := d.Rules[m.ID]
			if !ok {
				return fmt.Errorf("rule not found")
			}
			if !ValidDate(m.EffectiveFrom) {
				return fmt.Errorf("effectiveFrom required for rule changes")
			}
			old.EffectiveTo = DateAdd(m.EffectiveFrom, -1)
			d.Rules[m.ID] = old
			r.EffectiveFrom = m.EffectiveFrom
			for id, s := range d.Sessions {
				if s.RuleID == m.ID && s.Date >= m.EffectiveFrom && s.Plan != nil && s.Plan.Start.After(now) && !s.Exception {
					delete(d.Sessions, id)
				}
			}
		}
		loc, _ := time.LoadLocation(d.Settings.TimeZone)
		if r.EffectiveFrom < now.In(loc).Format(dateLayout) {
			return fmt.Errorf("recurrence cannot invent earlier occurrences")
		}
		if firstDate := r.EffectiveFrom; firstDate == now.In(loc).Format(dateLayout) {
			day, _ := time.Parse(dateLayout, firstDate)
			if (int(day.Weekday())+6)%7+1 == r.Weekday {
				start, e := Local(firstDate, r.LocalStart, loc)
				interval := d.Settings.Interval(firstDate)
				m, _ := minutes(r.LocalStart)
				dayStart, _ := minutes(interval.Start)
				if interval.NextDay && m < dayStart {
					start, e = Local(DateAdd(firstDate, 1), r.LocalStart, loc)
				}
				if e == nil && !start.After(now) {
					return fmt.Errorf("new recurrence cannot start in the past")
				}
			}
		}
		r.ID = uuid.NewString()
		if m.ID != "" {
			for id, s := range d.Sessions {
				occurrence := s.OccurrenceDate
				if occurrence == "" {
					occurrence = s.Date
				}
				if s.RuleID == m.ID && s.Exception && occurrence >= r.EffectiveFrom {
					s.RuleID = r.ID
					d.Sessions[id] = s
				}
			}
		}
		d.Rules[r.ID] = r
	case "session":
		if m.Session == nil {
			return fmt.Errorf("session required")
		}
		s := *m.Session
		s.ID = m.ID
		if s.ID == "" {
			s.ID = uuid.NewString()
		}
		if old, ok := d.Sessions[s.ID]; ok {
			if old.Plan != nil && !old.Plan.Start.After(now) {
				return fmt.Errorf("started plans are preserved; edit actual time")
			}
			s.RuleID = old.RuleID
			s.OccurrenceDate = old.OccurrenceDate
			s.Exception = old.RuleID != ""
		}
		if s.Plan == nil || !s.Plan.Start.After(now) {
			return fmt.Errorf("plans must start in the future; record actual work for the past")
		}
		a, e := d.assignment(s.Assignment, s.Date)
		if e != nil {
			return e
		}
		s.Assignment = a
		s.Actual = nil
		s.State = "accepted"
		s.Attention = ""
		s.ConflictIDs = []string{}
		if e = d.validateSession(s, busy); e != nil {
			return e
		}
		d.Sessions[s.ID] = s
	case "cancel":
		s, ok := d.Sessions[m.ID]
		if !ok {
			return fmt.Errorf("session not found")
		}
		if s.Plan != nil && !s.Plan.Start.After(now) {
			return fmt.Errorf("started plans are preserved; skip actual instead")
		}
		if s.Actual != nil {
			if s.RuleID != "" {
				return fmt.Errorf("recorded actual work is preserved; use skip instead")
			}
			// An unplanned, non-recurring actual is a manual log entry with no
			// recurring identity to preserve; canceling it removes the mistake.
			delete(d.Sessions, m.ID)
			return nil
		}
		s.State = "canceled"
		s.Exception = true
		d.Sessions[m.ID] = s
	case "actual":
		if m.Actual == nil {
			return fmt.Errorf("actual required")
		}
		s, ok := d.Sessions[m.ID]
		if !ok {
			if m.ID != "" || m.Session == nil {
				return fmt.Errorf("session required for unplanned actual")
			}
			s = *m.Session
			s.ID = uuid.NewString()
			s.Plan = nil
			s.State = "accepted"
			g, exists := d.Goals[s.Assignment.GoalID]
			if !exists {
				return fmt.Errorf("goal required")
			}
			s.Assignment.GoalTitle = g.Title
			s.Assignment.Color = g.Color
			s.Assignment.Title = g.Title
			stepFound := s.Assignment.StepID == ""
			for _, step := range g.Steps {
				if step.ID == s.Assignment.StepID {
					s.Assignment.Title = step.Title
					stepFound = true
				}
			}
			if !stepFound {
				return fmt.Errorf("subgoal not found for unplanned actual")
			}
		}
		if s.Assignment.GoalID == "" {
			return fmt.Errorf("actual work requires a goal")
		}
		a := *m.Actual
		if a.Status == "assumed" {
			if s.Plan == nil {
				return fmt.Errorf("no original plan")
			}
			if s.Plan.End.After(now) {
				return fmt.Errorf("actual work cannot be assumed before the planned session ends")
			}
			a.Start = s.Plan.Start
			a.End = s.Plan.End
			a.Date = s.Date
		}
		if a.Status == "skipped" && a.Date == "" {
			a.Date = s.Date
		}
		if a.Status == "explicit" {
			expected := d.ActualDate(a.Start)
			if a.Date != expected {
				return fmt.Errorf("actual scheduling date must be %s for this start time", expected)
			}
			if a.End.After(now) {
				return fmt.Errorf("actual work cannot end in the future")
			}
		}
		if e := d.validateActual(s.ID, a); e != nil {
			return e
		}
		g := d.Goals[s.Assignment.GoalID]
		a.OutsideTimeline = !Eligible(g, a.Date)
		s.Actual = &a
		d.Sessions[s.ID] = s
	default:
		return fmt.Errorf("unknown scheduler action")
	}
	return nil
}
func (d *Document) Week(w string, now time.Time, busy []Busy) Week {
	out := Week{Revision: d.Revision, Week: w, Settings: d.Settings, Days: []Day{}, Goals: []Summary{}, Sessions: []Session{}, Rules: []Rule{}, Busy: busy, Warnings: []string{}}
	if out.Busy == nil {
		out.Busy = []Busy{}
	}
	end := DateAdd(w, 6)
	for i := 0; i < 7; i++ {
		day, e := d.Settings.Day(DateAdd(w, i))
		if e != nil {
			out.Warnings = append(out.Warnings, e.Error())
		} else {
			out.Days = append(out.Days, day)
		}
	}
	for _, r := range d.Rules {
		out.Rules = append(out.Rules, r)
	}
	for _, id := range d.sessionIDs() {
		s := d.Sessions[id]
		if s.Date >= w && s.Date <= end || s.Actual != nil && s.Actual.Date >= w && s.Actual.Date <= end {
			out.Sessions = append(out.Sessions, s)
			if s.State == "attention" {
				out.Warnings = append(out.Warnings, s.Date+": "+s.Attention)
			}
		}
	}
	goals := []Goal{}
	if closed, ok := d.ClosedWeeks[w]; ok {
		goals = closed
	} else {
		for _, g := range d.Goals {
			goals = append(goals, g)
		}
	}
	historicalExtras := map[string]bool{}
	if _, closed := d.ClosedWeeks[w]; closed {
		seen := map[string]bool{}
		for _, g := range goals {
			seen[g.ID] = true
		}
		for _, s := range out.Sessions {
			if s.Assignment.GoalID != "" && !seen[s.Assignment.GoalID] {
				if g, ok := d.Goals[s.Assignment.GoalID]; ok {
					goals = append(goals, g)
					seen[g.ID] = true
					historicalExtras[g.ID] = true
				}
			}
		}
	}
	sort.Slice(goals, func(i, j int) bool { return goals[i].Title < goals[j].Title })
	uncovered := 0.
	for _, g := range goals {
		sum := Summary{Goal: g, RequiredHours: Required(g, w), UnscheduledStepIDs: []string{}}
		if historicalExtras[g.ID] {
			zero := 0.
			sum.RequiredHours = &zero
		}
		for _, id := range g.DependsOn {
			if prerequisite, ok := d.Goals[id]; ok && prerequisite.Status != "done" {
				out.Warnings = append(out.Warnings, g.Title+": unfinished dependency "+prerequisite.Title)
			}
		}
		steps := map[string]bool{}
		hasWork := false
		for _, s := range out.Sessions {
			if s.Assignment.GoalID != g.ID {
				continue
			}
			hasWork = true
			if s.Actual != nil {
				if s.Actual.Status != "skipped" && s.Actual.Date >= w && s.Actual.Date <= end {
					sum.ActualHours += s.Actual.End.Sub(s.Actual.Start).Hours()
					steps[s.Assignment.StepID] = true
				}
			} else if s.State == "accepted" && s.Plan != nil && s.Date >= w && s.Date <= end {
				sum.RemainingScheduledHours += s.Plan.End.Sub(s.Plan.Start).Hours()
				steps[s.Assignment.StepID] = true
			}
		}
		if !hasWork && (g.StartDate > end || g.EndDate < w || g.EligibleFrom > end && g.EligibleFrom != "" || g.StoppedDate < w && g.StoppedDate != "") {
			continue
		}
		for _, step := range g.Steps {
			if !step.Completed && !steps[step.ID] {
				sum.UnscheduledStepIDs = append(sum.UnscheduledStepIDs, step.ID)
			}
		}
		if sum.RequiredHours == nil {
			out.Warnings = append(out.Warnings, g.Title+": Hours not set")
		} else {
			covered := sum.ActualHours + sum.RemainingScheduledHours
			sum.UncoveredHours = max(0, *sum.RequiredHours-covered)
			sum.ExcessHours = max(0, covered-*sum.RequiredHours)
			uncovered += sum.UncoveredHours
			if sum.UncoveredHours > 0 {
				out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %.2f hours uncovered", g.Title, sum.UncoveredHours))
			}
		}
		out.Goals = append(out.Goals, sum)
	}
	for _, day := range out.Days {
		start := day.Start
		if now.After(start) {
			start = now
		}
		if !day.End.After(start) {
			continue
		}
		blocked := []Plan{}
		for _, s := range d.Sessions {
			if s.State == "accepted" && s.Plan != nil {
				blocked = append(blocked, *s.Plan)
			}
		}
		for _, b := range busy {
			blocked = append(blocked, Plan{Start: b.Start, End: b.End})
		}
		sort.Slice(blocked, func(i, j int) bool { return blocked[i].Start.Before(blocked[j].Start) })
		cursor := start
		free := 0.
		for _, p := range blocked {
			if !p.End.After(cursor) || !p.Start.Before(day.End) {
				continue
			}
			if p.Start.After(cursor) {
				free += p.Start.Sub(cursor).Hours()
			}
			if p.End.After(cursor) {
				cursor = p.End
			}
			if cursor.After(day.End) {
				cursor = day.End
			}
		}
		free += day.End.Sub(cursor).Hours()
		out.RemainingCapacityHours += free
	}
	if uncovered > out.RemainingCapacityHours {
		out.Warnings = append(out.Warnings, "Uncovered goal hours exceed remaining unreserved time")
	}
	return out
}

func (d *Document) ActualDate(start time.Time) string {
	loc, _ := time.LoadLocation(d.Settings.TimeZone)
	date := start.In(loc).Format(dateLayout)
	previous, err := d.Settings.Day(DateAdd(date, -1))
	if err == nil && previous.Interval.NextDay && start.Before(previous.End) && !start.Before(previous.Start) {
		return previous.Date
	}
	return date
}

// Validate checks imported history without requiring the original live goals;
// title snapshots and attribution are intentionally independent of those rows.
func (d *Document) Validate() error {
	if err := d.Settings.Validate(); err != nil {
		return err
	}
	if d.Settings.Weekdays == nil {
		d.Settings.Weekdays = map[string]DayInterval{}
	}
	if d.Settings.Dates == nil {
		d.Settings.Dates = map[string]DayInterval{}
	}
	if d.Goals == nil {
		d.Goals = map[string]Goal{}
	}
	if d.Sessions == nil {
		d.Sessions = map[string]Session{}
	}
	if d.Rules == nil {
		d.Rules = map[string]Rule{}
	}
	if d.ClosedWeeks == nil {
		d.ClosedWeeks = map[string][]Goal{}
	}
	if d.LastDate != "" && !ValidDate(d.LastDate) {
		return fmt.Errorf("invalid scheduler last date")
	}
	goal := func(g Goal) error {
		if g.ID == "" || !ValidDate(g.StartDate) || !ValidDate(g.EndDate) || g.StartDate > g.EndDate {
			return fmt.Errorf("invalid saved goal policy")
		}
		if g.DailyHours != nil && (*g.DailyHours < 0 || *g.DailyHours > 24) {
			return fmt.Errorf("invalid saved estimate")
		}
		seen := map[int]bool{}
		for _, v := range g.SelectedWeekdays {
			if v < 1 || v > 7 || seen[v] {
				return fmt.Errorf("invalid saved weekdays")
			}
			seen[v] = true
		}
		if g.StoppedDate != "" && !ValidDate(g.StoppedDate) || g.EligibleFrom != "" && !ValidDate(g.EligibleFrom) {
			return fmt.Errorf("invalid saved eligibility")
		}
		for _, p := range g.Pauses {
			if !ValidDate(p.From) || !ValidDate(p.To) || p.From > p.To {
				return fmt.Errorf("invalid saved eligibility pause")
			}
		}
		return nil
	}
	for id, g := range d.Goals {
		if id != g.ID {
			return fmt.Errorf("goal policy ID mismatch")
		}
		if e := goal(g); e != nil {
			return e
		}
	}
	for w, goals := range d.ClosedWeeks {
		if !ValidDate(w) || Monday(w) != w {
			return fmt.Errorf("invalid saved week")
		}
		for _, g := range goals {
			if e := goal(g); e != nil {
				return e
			}
		}
	}
	for id, r := range d.Rules {
		if id == "" || id != r.ID || r.Weekday < 1 || r.Weekday > 7 || r.DurationMinutes < 1 || r.DurationMinutes > 1440 || !ValidDate(r.EffectiveFrom) || r.EffectiveTo != "" && !ValidDate(r.EffectiveTo) {
			return fmt.Errorf("invalid saved recurring rule")
		}
		if _, e := minutes(r.LocalStart); e != nil {
			return e
		}
		if r.Assignment.GoalID == "" && r.Assignment.Title == "" {
			return fmt.Errorf("rule assignment required")
		}
	}
	for id, s := range d.Sessions {
		if id == "" || id != s.ID || !ValidDate(s.Date) || s.OccurrenceDate != "" && !ValidDate(s.OccurrenceDate) {
			return fmt.Errorf("invalid saved session identity")
		}
		if s.State != "accepted" && s.State != "attention" && s.State != "canceled" {
			return fmt.Errorf("invalid saved session state")
		}
		if s.RuleID != "" {
			if _, ok := d.Rules[s.RuleID]; !ok {
				return fmt.Errorf("saved session rule missing")
			}
		}
		if s.Plan != nil && (s.Plan.Start.IsZero() || !s.Plan.End.After(s.Plan.Start)) {
			return fmt.Errorf("invalid saved plan")
		}
		if s.Plan == nil && s.Actual == nil && s.State != "attention" && s.State != "canceled" {
			return fmt.Errorf("saved session has neither plan nor actual")
		}
		if s.Actual != nil {
			if s.Assignment.GoalID == "" || !ValidDate(s.Actual.Date) {
				return fmt.Errorf("actual assignment and date required")
			}
			if e := d.validateActual(id, *s.Actual); e != nil {
				return e
			}
		}
	}
	d.Busy = nil
	return nil
}
