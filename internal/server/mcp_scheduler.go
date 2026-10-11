package server

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/scheduler"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type scheduleWeekInput struct {
	Date string `json:"date,omitempty" jsonschema:"any YYYY-MM-DD date in the week to show; default is the current week in the schedule time zone"`
}

type scheduleAssignmentInput struct {
	GoalID       string `json:"goalId,omitempty" jsonschema:"goal ID from get_weekly_schedule goals; omit for a fixed commitment such as lunch"`
	StepID       string `json:"stepId,omitempty" jsonschema:"optional subgoal (step) ID of the goal"`
	Title        string `json:"title,omitempty" jsonschema:"title of a fixed commitment without a goal; goal titles are filled in automatically"`
	NewStepTitle string `json:"newStepTitle,omitempty" jsonschema:"optional title of a new subgoal to add to the goal and assign"`
}

type saveScheduleSessionInput struct {
	scheduleAssignmentInput
	Revision string `json:"revision" jsonschema:"schedule revision from get_weekly_schedule"`
	ID       string `json:"id,omitempty" jsonschema:"session ID to edit; omit to create a one-off session"`
	Date     string `json:"date" jsonschema:"scheduling date YYYY-MM-DD; overnight work belongs to the date it started on"`
	Start    string `json:"start" jsonschema:"start as RFC 3339 (2026-10-12T09:00:00-05:00) or local YYYY-MM-DDTHH:MM in the schedule time zone"`
	End      string `json:"end" jsonschema:"end as RFC 3339 or local YYYY-MM-DDTHH:MM in the schedule time zone"`
}

type removeScheduleSessionInput struct {
	Revision string `json:"revision" jsonschema:"schedule revision from get_weekly_schedule"`
	ID       string `json:"id" jsonschema:"session ID from get_weekly_schedule"`
	Scope    string `json:"scope,omitempty" jsonschema:"occurrence (default) removes this session only; future also ends its recurring rule from this occurrence"`
}

type saveScheduleRuleInput struct {
	scheduleAssignmentInput
	Revision        string `json:"revision" jsonschema:"schedule revision from get_weekly_schedule"`
	ID              string `json:"id,omitempty" jsonschema:"rule ID to change; omit to create a weekly rule. A change ends the old rule before effectiveFrom and starts a new rule ID"`
	Weekday         int    `json:"weekday" jsonschema:"day of week, 1=Monday through 7=Sunday"`
	LocalStart      string `json:"localStart" jsonschema:"start time HH:MM in the schedule time zone"`
	DurationMinutes int    `json:"durationMinutes" jsonschema:"duration in minutes, 1-1440"`
	EffectiveFrom   string `json:"effectiveFrom" jsonschema:"first date YYYY-MM-DD the rule (or the change) applies, today or later"`
	EffectiveTo     string `json:"effectiveTo,omitempty" jsonschema:"optional last date YYYY-MM-DD of the rule"`
}

type endScheduleRuleInput struct {
	Revision string `json:"revision" jsonschema:"schedule revision from get_weekly_schedule"`
	ID       string `json:"id" jsonschema:"rule ID from get_weekly_schedule rules"`
	From     string `json:"from" jsonschema:"first date YYYY-MM-DD without the rule, today or later; earlier history is kept"`
}

type recordScheduleActualInput struct {
	Revision  string `json:"revision" jsonschema:"schedule revision from get_weekly_schedule"`
	SessionID string `json:"sessionId,omitempty" jsonschema:"planned session ID; omit to log unplanned work, which needs goalId"`
	Status    string `json:"status" jsonschema:"explicit (worked start to end), assumed (worked as planned, after the plan ends) or skipped (planned session not done)"`
	Start     string `json:"start,omitempty" jsonschema:"explicit work start as RFC 3339 or local YYYY-MM-DDTHH:MM"`
	End       string `json:"end,omitempty" jsonschema:"explicit work end as RFC 3339 or local YYYY-MM-DDTHH:MM, not in the future"`
	GoalID    string `json:"goalId,omitempty" jsonschema:"goal for unplanned work"`
	StepID    string `json:"stepId,omitempty" jsonschema:"optional subgoal for unplanned work"`
}

type scheduleDayInput struct {
	Start   string `json:"start" jsonschema:"window start HH:MM"`
	End     string `json:"end" jsonschema:"window end HH:MM"`
	NextDay bool   `json:"nextDay,omitempty" jsonschema:"true when end is after midnight on the next day"`
}

type updateScheduleSettingsInput struct {
	Revision   string                      `json:"revision" jsonschema:"schedule revision from get_weekly_schedule"`
	TimeZone   string                      `json:"timeZone" jsonschema:"IANA time zone such as America/Chicago"`
	DefaultDay scheduleDayInput            `json:"defaultDay" jsonschema:"default daily planning window"`
	Weekdays   map[string]scheduleDayInput `json:"weekdays,omitempty" jsonschema:"window overrides keyed by weekday 1=Monday through 7=Sunday"`
	Dates      map[string]scheduleDayInput `json:"dates,omitempty" jsonschema:"window overrides keyed by YYYY-MM-DD date"`
}

func (in scheduleDayInput) interval() scheduler.DayInterval {
	return scheduler.DayInterval{Start: in.Start, End: in.End, NextDay: in.NextDay}
}

func scheduleIntervals(in map[string]scheduleDayInput) map[string]scheduler.DayInterval {
	out := make(map[string]scheduler.DayInterval, len(in))
	for key, day := range in {
		out[key] = day.interval()
	}
	return out
}

var scheduleWrite = &mcp.ToolAnnotations{DestructiveHint: new(bool), OpenWorldHint: new(bool)}

func (s *Server) addSchedulerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_weekly_schedule", Description: "Read one week (Monday to Sunday) of the weekly schedule: settings and time zone, day windows, planned and recorded sessions, recurring weekly rules, goal hour summaries, Google busy time, warnings, and the revision every schedule write needs. Times are RFC 3339 instants; weekday 1 is Monday.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in scheduleWeekInput) (*mcp.CallToolResult, any, error) {
		week, err := s.scheduleWeek(ctx, in.Date)
		if err != nil {
			return nil, nil, err
		}
		busy, availabilityErr := s.schedulerAvailability(ctx, week)
		out, err := s.db.SchedulerWeek(ctx, week, s.schedulerNow(), busy)
		if err != nil {
			return nil, nil, scheduleToolError(err)
		}
		if availabilityErr != nil {
			out.Warnings = append(out.Warnings, "Google availability is stale or unavailable; refresh before planning")
		}
		return nil, out, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "save_schedule_session", Description: "Create or edit a one-off planned session that starts in the future. Assign it to a goal (optionally a subgoal, or a new subgoal via newStepTitle) or give a title for a fixed commitment. Editing an occurrence of a recurring rule changes that date only; use save_schedule_rule for the rule. Rejected when it overlaps other plans or busy time, or falls outside the day window. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Save schedule session", DestructiveHint: new(bool), OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in saveScheduleSessionInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		if !scheduler.ValidDate(in.Date) {
			return nil, nil, invalidInput("date must be YYYY-MM-DD")
		}
		loc, err := s.scheduleLocation(ctx)
		if err != nil {
			return nil, nil, err
		}
		start, err := scheduleTime(in.Start, loc, "start")
		if err != nil {
			return nil, nil, err
		}
		end, err := scheduleTime(in.End, loc, "end")
		if err != nil {
			return nil, nil, err
		}
		m := scheduler.Mutation{Action: "session", Revision: in.Revision, Week: scheduler.Monday(in.Date), ID: in.ID,
			Session: &scheduler.Session{Date: in.Date, Assignment: in.assignment(), Plan: &scheduler.Plan{Start: start, End: end}}, NewStep: in.newStep()}
		return s.scheduleMutate(ctx, m)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_schedule_session", Description: "Cancel a session that has not started; it stays listed with state=canceled so the date keeps no plan. scope=future also ends its recurring rule from this occurrence on. A recorded unplanned log is deleted; other recorded work is kept (use record_schedule_actual with status=skipped instead). Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Remove schedule session", DestructiveHint: &deleteIsDestructive, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in removeScheduleSessionInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		scope := ""
		switch in.Scope {
		case "", "occurrence":
		case "future":
			scope = "future"
		default:
			return nil, nil, invalidInput("scope must be occurrence or future")
		}
		week, err := s.scheduleSessionWeek(ctx, in.ID)
		if err != nil {
			return nil, nil, err
		}
		return s.scheduleMutate(ctx, scheduler.Mutation{Action: "cancel", Revision: in.Revision, Week: week, ID: in.ID, Scope: scope})
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "save_schedule_rule", Description: "Create or change a recurring weekly rule (for example every Monday 09:00 for 90 minutes). Changing a rule ends it the day before effectiveFrom and starts a replacement with a new ID, so earlier history is kept. New occurrences are generated automatically; the save is rejected and names the dates when an occurrence would overlap other plans. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Save schedule rule", DestructiveHint: new(bool), OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in saveScheduleRuleInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		if !scheduler.ValidDate(in.EffectiveFrom) {
			return nil, nil, invalidInput("effectiveFrom must be YYYY-MM-DD")
		}
		m := scheduler.Mutation{Action: "rule", Revision: in.Revision, Week: scheduler.Monday(in.EffectiveFrom), ID: in.ID, NewStep: in.newStep(),
			Rule: &scheduler.Rule{Weekday: in.Weekday, LocalStart: in.LocalStart, DurationMinutes: in.DurationMinutes, EffectiveFrom: in.EffectiveFrom, EffectiveTo: in.EffectiveTo, Assignment: in.assignment()}}
		if in.ID != "" {
			m.EffectiveFrom = in.EffectiveFrom
		}
		return s.scheduleMutate(ctx, m)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "end_schedule_rule", Description: "Delete a recurring weekly rule from a date on: the rule stops the day before from, and its later sessions that have not started or recorded work are removed. Earlier sessions and recorded work are kept. A rule that already ended before from is rejected; a rule ended on or before its first date is removed. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "End schedule rule", DestructiveHint: &deleteIsDestructive, IdempotentHint: true, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in endScheduleRuleInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		if !scheduler.ValidDate(in.From) {
			return nil, nil, invalidInput("from must be YYYY-MM-DD")
		}
		return s.scheduleMutate(ctx, scheduler.Mutation{Action: "endRule", Revision: in.Revision, Week: scheduler.Monday(in.From), ID: in.ID, EffectiveFrom: in.From})
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "record_schedule_actual", Description: "Record work against the schedule. With sessionId: explicit saves the actual start and end, assumed restores the plan as worked once it has ended, skipped marks it not done. Without sessionId: logs unplanned explicit work for goalId. Actual work must be on a goal and cannot end in the future. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Record schedule actual work", DestructiveHint: new(bool), OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in recordScheduleActualInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		m := scheduler.Mutation{Action: "actual", Revision: in.Revision, ID: in.SessionID, Actual: &scheduler.Actual{Status: in.Status}}
		switch in.Status {
		case "explicit":
			loc, err := s.scheduleLocation(ctx)
			if err != nil {
				return nil, nil, err
			}
			if m.Actual.Start, err = scheduleTime(in.Start, loc, "start"); err != nil {
				return nil, nil, err
			}
			if m.Actual.End, err = scheduleTime(in.End, loc, "end"); err != nil {
				return nil, nil, err
			}
			// The store replaces this with the scheduling date the start belongs to.
			m.Actual.Date = m.Actual.Start.In(loc).Format(time.DateOnly)
			m.Week = scheduler.Monday(m.Actual.Date)
		case "assumed", "skipped":
			if in.SessionID == "" {
				return nil, nil, invalidInput("assumed and skipped need a sessionId")
			}
		default:
			return nil, nil, invalidInput("status must be explicit, assumed or skipped")
		}
		if in.SessionID == "" {
			if in.GoalID == "" {
				return nil, nil, invalidInput("unplanned work needs a goalId")
			}
			m.Session = &scheduler.Session{Assignment: scheduler.Assignment{GoalID: in.GoalID, StepID: in.StepID}}
		} else if m.Week == "" {
			week, err := s.scheduleSessionWeek(ctx, in.SessionID)
			if err != nil {
				return nil, nil, err
			}
			m.Week = week
		}
		return s.scheduleMutate(ctx, m)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "update_schedule_settings", Description: "Replace schedule settings: IANA time zone, the default daily planning window, and optional windows per weekday or date. Times are HH:MM; set nextDay when a window ends after midnight. Read the current settings with get_weekly_schedule first. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Update schedule settings", DestructiveHint: new(bool), IdempotentHint: true, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in updateScheduleSettingsInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		week, err := s.scheduleWeek(ctx, "")
		if err != nil {
			return nil, nil, err
		}
		settings := scheduler.Settings{TimeZone: in.TimeZone, DefaultDay: in.DefaultDay.interval(), Weekdays: scheduleIntervals(in.Weekdays), Dates: scheduleIntervals(in.Dates)}
		return s.scheduleMutate(ctx, scheduler.Mutation{Action: "settings", Revision: in.Revision, Week: week, Settings: &settings})
	})
}

func (in scheduleAssignmentInput) assignment() scheduler.Assignment {
	return scheduler.Assignment{GoalID: in.GoalID, StepID: in.StepID, Title: strings.TrimSpace(in.Title)}
}

func (in scheduleAssignmentInput) newStep() *scheduler.NewStep {
	if strings.TrimSpace(in.NewStepTitle) == "" {
		return nil
	}
	return &scheduler.NewStep{Title: in.NewStepTitle}
}

// scheduleMutate applies m like POST /api/scheduler/mutate, including the
// fresh Google availability check for planning changes.
func (s *Server) scheduleMutate(ctx context.Context, m scheduler.Mutation) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(m.Revision) == "" {
		return nil, nil, invalidInput("revision is required; read it from get_weekly_schedule")
	}
	var busy []scheduler.Busy
	if m.Action != "actual" && m.Action != "cancel" && m.Action != "endRule" {
		var err error
		if busy, err = s.schedulerPlanningAvailability(ctx, m.Week); err != nil {
			log.Printf("mcp: scheduler availability: %v", err)
			return nil, nil, errors.New("Google availability check failed; nothing was saved. Retry shortly")
		}
	}
	out, err := s.db.SchedulerMutate(ctx, m, s.schedulerNow(), busy)
	if err != nil {
		return nil, nil, scheduleToolError(err)
	}
	return nil, out, nil
}

// scheduleWeek returns the Monday of date's week, or of the current week in
// the schedule time zone when date is empty.
func (s *Server) scheduleWeek(ctx context.Context, date string) (string, error) {
	if date != "" {
		if !scheduler.ValidDate(date) {
			return "", invalidInput("date must be YYYY-MM-DD")
		}
		return scheduler.Monday(date), nil
	}
	loc, err := s.scheduleLocation(ctx)
	if err != nil {
		return "", err
	}
	return scheduler.Monday(s.schedulerNow().In(loc).Format(time.DateOnly)), nil
}

// scheduleSessionWeek returns the week of a saved session so the response
// shows where the change happened.
func (s *Server) scheduleSessionWeek(ctx context.Context, id string) (string, error) {
	doc, err := s.db.SchedulerDocument(ctx)
	if err != nil {
		return "", toolError(err)
	}
	session, ok := doc.Sessions[id]
	if !ok || !scheduler.ValidDate(session.Date) {
		return "", invalidInput("session not found; refresh with get_weekly_schedule")
	}
	return scheduler.Monday(session.Date), nil
}

func (s *Server) scheduleLocation(ctx context.Context) (*time.Location, error) {
	doc, err := s.db.SchedulerDocument(ctx)
	if err != nil {
		return nil, toolError(err)
	}
	loc, err := time.LoadLocation(doc.Settings.TimeZone)
	if err != nil {
		return nil, toolError(err)
	}
	return loc, nil
}

// scheduleTime parses an RFC 3339 instant, or a local date-time without an
// offset in the schedule time zone.
func scheduleTime(value string, loc *time.Location, field string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	date, clock, ok := strings.Cut(value, "T")
	if ok && scheduler.ValidDate(date) && len(clock) == 5 {
		if t, err := scheduler.Local(date, clock, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, invalidInput("%s must be RFC 3339 or YYYY-MM-DDTHH:MM in %s", field, loc)
}

func scheduleToolError(err error) error {
	var conflict *scheduler.Conflict
	switch {
	case errors.Is(err, database.ErrConflict):
		return invalidInput("the schedule changed since that revision; call get_weekly_schedule and retry with the new revision")
	case errors.As(err, &conflict):
		return invalidInput("%s (conflicting session IDs: %s)", conflict.Message, strings.Join(conflict.IDs, ", "))
	case errors.Is(err, database.ErrInvalid):
		return invalidInput("%s", strings.TrimPrefix(err.Error(), database.ErrInvalid.Error()+": "))
	}
	return toolError(err)
}
