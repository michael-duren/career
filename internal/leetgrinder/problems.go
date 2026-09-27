package leetgrinder

import (
	"cmp"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ProblemRow is one curriculum problem with the learner's progress on it.
type ProblemRow struct {
	Problem  Problem
	Week     Week
	Day      Day
	Optional bool
	// Order is the problem's position in the curriculum, starting at 1.
	Order    int
	Status   string
	Attempts int
	// Last is the newest attempt; it is zero when Attempts is 0.
	Last Attempt
}

// Status filter keys, in the order the filter lists them.
var problemStatuses = []struct{ Key, Label string }{
	{"not-attempted", "Not attempted"},
	{"unfinished", "Unfinished"},
	{"struggled", "Struggled"},
	{"solved-help", "Solved with help"},
	{"solved", "Solved independently"},
}

var problemSorts = []struct{ Key, Label string }{
	{"curriculum", "Curriculum order"},
	{"difficulty", "Difficulty"},
	{"number", "LeetCode number"},
	{"title", "Title"},
	{"recent", "Recently attempted"},
}

var difficultyRank = map[string]int{"Easy": 1, "Medium": 2, "Hard": 3}

// ProblemFilter holds the search form; zero values match everything.
type ProblemFilter struct {
	Query      string
	Difficulty string
	Week       int
	Status     string
	Kind       string // "", "core", or "optional"
	Sort       string
}

// ParseProblemFilter reads the search form, ignoring values it does not know.
func ParseProblemFilter(q url.Values) ProblemFilter {
	f := ProblemFilter{Query: strings.TrimSpace(q.Get("q")), Sort: "curriculum"}
	if r := []rune(f.Query); len(r) > 100 {
		f.Query = string(r[:100])
	}
	if _, ok := difficultyRank[q.Get("difficulty")]; ok {
		f.Difficulty = q.Get("difficulty")
	}
	if week, err := strconv.Atoi(q.Get("week")); err == nil && week >= 1 && week <= 12 {
		f.Week = week
	}
	for _, s := range problemStatuses {
		if q.Get("status") == s.Key {
			f.Status = s.Key
		}
	}
	if kind := q.Get("kind"); kind == "core" || kind == "optional" {
		f.Kind = kind
	}
	for _, s := range problemSorts {
		if q.Get("sort") == s.Key {
			f.Sort = s.Key
		}
	}
	return f
}

// Active reports whether any filter narrows the list.
func (f ProblemFilter) Active() bool {
	return f.Query != "" || f.Difficulty != "" || f.Week != 0 || f.Status != "" || f.Kind != ""
}

func (f ProblemFilter) match(r ProblemRow) bool {
	if f.Query != "" {
		q := strings.ToLower(f.Query)
		text := strings.ToLower(r.Problem.Title + " " + r.Problem.Slug + " " + r.Week.Title + " " + r.Day.Title + " " + strconv.Itoa(r.Problem.ID))
		if !strings.Contains(text, q) {
			return false
		}
	}
	switch {
	case f.Difficulty != "" && r.Problem.Difficulty != f.Difficulty,
		f.Week != 0 && r.Week.Number != f.Week,
		f.Status != "" && statusKey(r.Status) != f.Status,
		f.Kind == "core" && r.Optional,
		f.Kind == "optional" && !r.Optional:
		return false
	}
	return true
}

func statusKey(label string) string {
	for _, s := range problemStatuses {
		if s.Label == label {
			return s.Key
		}
	}
	return "not-attempted"
}

// ProblemRows lists every curriculum problem in curriculum order.
func ProblemRows(state State) []ProblemRow {
	var rows []ProblemRow
	for _, week := range Curriculum() {
		for _, day := range week.Days {
			add := func(p Problem, optional bool) {
				attempts := state.ProblemAttempts(p.Slug)
				row := ProblemRow{Problem: p, Week: week, Day: day, Optional: optional, Order: len(rows) + 1, Status: state.ProblemStatus(p.Slug), Attempts: len(attempts)}
				if len(attempts) > 0 {
					row.Last = attempts[0]
				}
				rows = append(rows, row)
			}
			for _, p := range day.Core {
				add(p, false)
			}
			for _, p := range day.Optional {
				add(p, true)
			}
		}
	}
	return rows
}

// FilterProblems applies the filter and sort to rows without modifying them.
func FilterProblems(rows []ProblemRow, f ProblemFilter) []ProblemRow {
	var result []ProblemRow
	for _, r := range rows {
		if f.match(r) {
			result = append(result, r)
		}
	}
	slices.SortStableFunc(result, func(a, b ProblemRow) int {
		switch f.Sort {
		case "difficulty":
			return cmp.Compare(difficultyRank[a.Problem.Difficulty], difficultyRank[b.Problem.Difficulty])
		case "number":
			return cmp.Compare(a.Problem.ID, b.Problem.ID)
		case "title":
			return cmp.Compare(strings.ToLower(a.Problem.Title), strings.ToLower(b.Problem.Title))
		case "recent":
			// Newest attempt first; never-attempted problems keep curriculum order at the end.
			return b.Last.CreatedAt.Compare(a.Last.CreatedAt)
		}
		return cmp.Compare(a.Order, b.Order)
	})
	return result
}

// ProblemsPage is the searchable list of every curriculum problem.
type ProblemsPage struct {
	Rows     []ProblemRow
	Total    int
	Filter   ProblemFilter
	Location *time.Location
}

// FilterOption is one choice in a filter select.
type FilterOption struct{ Key, Label string }

func ProblemStatusOptions() []FilterOption {
	var o []FilterOption
	for _, s := range problemStatuses {
		o = append(o, FilterOption(s))
	}
	return o
}

func ProblemSortOptions() []FilterOption {
	var o []FilterOption
	for _, s := range problemSorts {
		o = append(o, FilterOption(s))
	}
	return o
}

// StatusClass maps a progress label to its pill class.
func StatusClass(label string) string { return "status-pill status-" + statusKey(label) }
