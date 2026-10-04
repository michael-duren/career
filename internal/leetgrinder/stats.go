package leetgrinder

import (
	"cmp"
	"net/url"
	"slices"
	"strings"
	"time"
)

// otherTopic groups topics with fewer than MinTopicProblems problems.
const (
	otherTopic       = "other"
	MinTopicProblems = 3
)

// TopicStat is the weakness summary of one topic tag.
type TopicStat struct {
	Key, Label string
	// Problems attempted and solved (at least one solved attempt).
	Problems, Solved int
	// Attempts, and those that struggled, stayed unfinished, or used help.
	Attempts, Struggles int
	// RecallSum adds each problem's current recall estimate.
	RecallSum float64
	Due       int
}

// StruggleRate is the share of attempts that struggled, stayed unfinished,
// or used help.
func (t TopicStat) StruggleRate() float64 {
	if t.Attempts == 0 {
		return 0
	}
	return float64(t.Struggles) / float64(t.Attempts)
}

// Recall is the average current recall estimate of the topic's problems.
func (t TopicStat) Recall() float64 {
	if t.Problems == 0 {
		return 0
	}
	return t.RecallSum / float64(t.Problems)
}

// DifficultyStat counts problems attempted and solved at one difficulty.
type DifficultyStat struct {
	Difficulty        string
	Attempted, Solved int
}

var statSorts = []FilterOption{
	{"recall", "Weakest recall"},
	{"struggle", "Struggle rate"},
	{"due", "Due now"},
	{"problems", "Problems"},
	{"topic", "Topic"},
}

// StatsFilter is the stats page's query: a sort, and whether small topics
// are listed on their own instead of under "Other".
type StatsFilter struct {
	Sort string
	All  bool
}

func ParseStatsFilter(q url.Values) StatsFilter {
	f := StatsFilter{Sort: "recall", All: q.Get("all") == "1"}
	for _, s := range statSorts {
		if q.Get("sort") == s.Key {
			f.Sort = s.Key
		}
	}
	return f
}

func StatSortOptions() []FilterOption { return statSorts }

// StatsPage is the topic weakness view, difficulty mix and goal calendar.
type StatsPage struct {
	Topics       []TopicStat
	Difficulties []DifficultyStat
	Calendar     [][]CalendarDay
	Trends       Trends
	Filter       StatsFilter
	Today        time.Time
	// Progress, Streaks and DueToday (reviews due by the end of today) feed
	// the headline tiles.
	Progress Progress
	Streaks  Streaks
	DueToday int
}

// NewStatsPage summarises today's state by topic and difficulty, with a
// weeks-long goal calendar.
func NewStatsPage(today Today, f StatsFilter, weeks int) StatsPage {
	state := today.State
	cards := map[string]Card{}
	for _, c := range today.Cards {
		cards[c.Problem.Slug] = c
	}
	type problemStat struct {
		problem             Problem
		attempts, struggles int
		solved              bool
	}
	var problems []*problemStat
	bySlug := map[string]*problemStat{}
	for _, a := range state.Attempts {
		p := bySlug[a.ProblemSlug]
		if p == nil {
			p = &problemStat{problem: state.Problem(a.ProblemSlug)}
			bySlug[a.ProblemSlug] = p
			problems = append(problems, p)
		}
		p.attempts++
		if a.Outcome != "solved" || a.Assisted {
			p.struggles++
		}
		if a.Outcome == "solved" {
			p.solved = true
		}
	}
	// Count problems per topic first, to fold small topics into "Other".
	counts := map[string]int{}
	for _, p := range problems {
		for _, t := range topicKeys(p.problem) {
			counts[t]++
		}
	}
	topics := map[string]*TopicStat{}
	for _, p := range problems {
		keys := map[string]bool{}
		for _, t := range topicKeys(p.problem) {
			if !f.All && t != untaggedTopic && counts[t] < MinTopicProblems {
				t = otherTopic
			}
			keys[t] = true
		}
		card, carded := cards[p.problem.Slug]
		for key := range keys {
			s := topics[key]
			if s == nil {
				s = &TopicStat{Key: key, Label: topicStatLabel(key, p.problem)}
				topics[key] = s
			}
			s.Problems++
			s.Attempts += p.attempts
			s.Struggles += p.struggles
			if p.solved {
				s.Solved++
			}
			if carded {
				s.RecallSum += card.Retrievability(today.Now)
				if today.Due(card) {
					s.Due++
				}
			}
		}
	}
	page := StatsPage{Filter: f, Today: today.Date, Progress: Summarize(state), Streaks: today.Streaks}
	for _, c := range today.Cards {
		if today.Due(c) {
			page.DueToday++
		}
	}
	for _, s := range topics {
		page.Topics = append(page.Topics, *s)
	}
	slices.SortFunc(page.Topics, func(a, b TopicStat) int {
		var c int
		switch f.Sort {
		case "struggle":
			c = cmp.Compare(b.StruggleRate(), a.StruggleRate())
		case "due":
			c = cmp.Compare(b.Due, a.Due)
		case "problems":
			c = cmp.Compare(b.Problems, a.Problems)
		case "recall":
			c = cmp.Compare(a.Recall(), b.Recall())
		}
		return cmp.Or(c, cmp.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label)))
	})
	for _, d := range []string{"Easy", "Medium", "Hard", ""} {
		stat := DifficultyStat{Difficulty: d}
		for _, p := range problems {
			if p.problem.Difficulty == d {
				stat.Attempted++
				if p.solved {
					stat.Solved++
				}
			}
		}
		if d != "" || stat.Attempted > 0 {
			page.Difficulties = append(page.Difficulties, stat)
		}
	}
	page.Calendar = calendarWeeks(today, weeks)
	page.Trends = WeeklyTrends(today, weeks)
	return page
}

func topicKeys(p Problem) []string {
	if len(p.Topics) == 0 {
		return []string{untaggedTopic}
	}
	return p.Topics
}

func topicStatLabel(key string, p Problem) string {
	switch key {
	case untaggedTopic:
		return "Untagged"
	case otherTopic:
		return "Other"
	}
	return p.TopicLabel(key)
}

// calendarWeeks lays the last weeks of goal results out as rows of Monday
// to Sunday, ending with the week that holds today. Days after today are
// left zero.
func calendarWeeks(today Today, weeks int) [][]CalendarDay {
	offset := (int(today.Date.Weekday()) + 6) % 7 // days since Monday
	days := weeks*7 - (6 - offset)
	cells := today.Calendar(days)
	var out [][]CalendarDay
	for i := 0; i < weeks; i++ {
		row := make([]CalendarDay, 7)
		for j := range row {
			if k := i*7 + j; k < len(cells) {
				row[j] = cells[k]
			}
		}
		out = append(out, row)
	}
	return out
}

// DifficultyLabel names a difficulty, or "Unknown" before metadata arrives.
func DifficultyLabel(d string) string {
	if d == "" {
		return "Unknown"
	}
	return d
}

// CalendarClass is the calendar cell class for a day.
func CalendarClass(d CalendarDay) string {
	switch {
	case d.Date.IsZero():
		return "cal-day cal-future"
	case d.Met:
		return "cal-day cal-met"
	case d.Active:
		return "cal-day cal-active"
	}
	return "cal-day"
}

// CalendarTitle describes a calendar cell for its tooltip and screen readers.
func CalendarTitle(d CalendarDay) string {
	if d.Date.IsZero() {
		return ""
	}
	switch {
	case d.Met:
		return DateLabel(d.Date) + ": goal met"
	case d.Active:
		return DateLabel(d.Date) + ": practised, goal not met"
	}
	return DateLabel(d.Date) + ": no attempts"
}
