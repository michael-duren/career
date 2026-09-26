package leetgrinder

import (
	_ "embed"
	"fmt"
	"strconv"
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
	Error     string
}

type DayPage struct {
	Day   Day
	State State
	IDs   map[string]string
	Error string
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
func EditForm(a Attempt) AttemptForm {
	return AttemptForm{ID: a.ID, Revision: a.Revision, Outcome: a.Outcome, Minutes: Count(a.Minutes), Assisted: a.Assisted, Notes: a.Notes}
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
