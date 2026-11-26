package leetgrinder

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/michael-duren/career-strategy/internal/timefmt"
)

// Started reports whether the learner has logged anything yet.
func (p OverviewPage) Started() bool { return len(p.Today.State.Attempts) > 0 }

// Recent returns the newest n attempts; State keeps attempts newest first.
func (p OverviewPage) Recent(n int) []Attempt {
	if len(p.Today.State.Attempts) < n {
		return p.Today.State.Attempts
	}
	return p.Today.State.Attempts[:n]
}

// DueSoon is the start of today's optional due list, at most n cards, and
// the list's full length.
func DueSoon(today Today, n int) ([]Card, int) {
	due := today.DueOptional()
	return due[:min(n, len(due))], len(due)
}

// DayUnit is the unit label for a count of days.
func DayUnit(n int) string {
	if n == 1 {
		return "day"
	}
	return "days"
}

// IndependentRate is the share of solved problems solved without help.
func (p OverviewPage) IndependentRate() string {
	progress := Summarize(p.Today.State)
	if progress.Solved == 0 {
		return "—"
	}
	return Percent(float64(progress.Independent) / float64(progress.Solved))
}

// DifficultyClass maps a LeetCode difficulty to its badge class.
func DifficultyClass(difficulty string) string { return "diff diff-" + strings.ToLower(difficulty) }

// KindLabel names an attempt kind for display.
func KindLabel(kind string) string {
	switch kind {
	case KindNew:
		return "New"
	case KindReview:
		return "Review"
	case KindPractice:
		return "Practice"
	}
	return ""
}

// DashboardWeeks is how many weeks the Today page's SOLVED PER WEEK chart shows.
const DashboardWeeks = 8

// paceWindow is the number of full weeks the trend note compares with the
// same number of weeks before them.
const paceWindow = 4

// Weak-topic thresholds: recall below or struggle above is drawn in accent.
const (
	weakRecall   = 0.8
	weakStruggle = 0.55
)

// DashboardTopics and BuildSetTopics are how many weakest topics the Today
// page lists and how many seed the "build todo set" button.
const (
	DashboardTopics = 10
	BuildSetTopics  = 6
)

// Dashboard is the Today page's derived data: the streak scale, weekly
// solves, recent rates, difficulty mix and weakest topics.
type Dashboard struct {
	// Ticks is the streak scale from 0 to TickMax, filled up to the current streak.
	Ticks   []Tick
	TickMax int
	Weeks   []DashWeek
	// Scale is the value at the top of the weekly chart; Guides are the
	// dashed lines drawn at Scale/2 and Scale.
	Scale  int
	Guides []DashGuide
	// Solved adds the shown weeks' solved problems.
	Solved int
	// Pace is a short note comparing the last full weeks with the weeks
	// before them ("pace slipping"), PaceCounts the two totals ("12 vs 20")
	// and PaceTitle spells it out; all empty without enough history.
	Pace, PaceCounts, PaceTitle        string
	Independent, Complexity, AvgMedium string
	Difficulties                       []DashDifficulty
	Topics                             []DashTopic
	// BuildTitle and BuildProblems prefill a new todo set from the weakest topics.
	BuildTitle    string
	BuildProblems []string
}

// Tick is one mark of a tick scale; every fifth is major.
type Tick struct{ On, Major bool }

// DashWeek is one column of the weekly chart.
type DashWeek struct {
	Label, Title string
	Solved       int
	// Height is the column height in percent of the chart's scale.
	Height  string
	Current bool
}

// DashGuide is a dashed guide line at Value, Bottom percent up the chart.
type DashGuide struct {
	Value  int
	Bottom string
}

// DashDifficulty is a difficulty's solved share, drawn on a measurement line.
type DashDifficulty struct {
	Name, Label, Width string
}

// DashTopic is one row of the weakest topics table.
type DashTopic struct {
	Key, Label               string
	Problems                 int
	Recall, Struggle         float64
	WeakRecall, WeakStruggle bool
}

// Dashboard derives the Today page's charts and tables from today's state.
func (p OverviewPage) Dashboard() Dashboard {
	today := p.Today
	d := Dashboard{}
	d.TickMax, d.Ticks = streakTicks(today.Streaks.Current, today.Streaks.Longest)

	trends := WeeklyTrends(today, DashboardWeeks+paceWindow)
	d.Pace, d.PaceCounts, d.PaceTitle = paceNote(trends)
	shown := trends.Weeks[max(0, len(trends.Weeks)-DashboardWeeks):]
	most := 0
	var solves, independent, checked, matched, mediumMinutes, mediumTimed int
	for _, w := range shown {
		most = max(most, w.Solved)
		d.Solved += w.Solved
		solves += w.Solves
		independent += w.Independent
		checked += w.Checked
		matched += w.Matched
		mediumMinutes += w.Minutes["Medium"]
		mediumTimed += w.Timed["Medium"]
	}
	d.Scale = chartScale(most)
	d.Guides = []DashGuide{{d.Scale / 2, "50%"}, {d.Scale, "100%"}}
	for i, w := range shown {
		d.Weeks = append(d.Weeks, DashWeek{
			Label:   timefmt.Date(w.Start),
			Title:   "Week of " + timefmt.Date(w.Start) + ": " + Count(w.Solved) + " " + plural(w.Solved, "problem", "problems") + " solved",
			Solved:  w.Solved,
			Height:  strconv.FormatFloat(float64(w.Solved)*100/float64(d.Scale), 'f', 1, 64) + "%",
			Current: i == len(shown)-1,
		})
	}
	d.Independent = "—"
	if solves > 0 {
		d.Independent = Percent(float64(independent) / float64(solves))
	}
	d.Complexity = "—"
	if checked > 0 {
		d.Complexity = Count(matched) + "/" + Count(checked)
	}
	d.AvgMedium = "—"
	if mediumTimed > 0 {
		d.AvgMedium = Count((mediumMinutes+mediumTimed/2)/mediumTimed) + " min"
	}

	stats := NewStatsPage(today, StatsFilter{Sort: "recall"}, 1)
	for _, s := range stats.Difficulties {
		if s.Attempted == 0 {
			continue
		}
		share := float64(s.Solved) / float64(s.Attempted)
		d.Difficulties = append(d.Difficulties, DashDifficulty{
			Name:  DifficultyLabel(s.Difficulty),
			Label: Count(s.Solved) + "/" + Count(s.Attempted) + " · " + Percent(share),
			Width: BarWidth(share, 1) + "%",
		})
	}
	var weakest []TopicStat
	for _, t := range stats.Topics {
		if t.Key == otherTopic || t.Key == untaggedTopic {
			continue
		}
		weakest = append(weakest, t)
	}
	for _, t := range weakest[:min(DashboardTopics, len(weakest))] {
		row := DashTopic{Key: t.Key, Label: t.Label, Problems: t.Problems, Recall: t.Recall(), Struggle: t.StruggleRate()}
		row.WeakRecall, row.WeakStruggle = weakFlags(row.Recall, row.Struggle)
		d.Topics = append(d.Topics, row)
	}
	d.BuildProblems = weakTopicProblems(today, weakest[:min(BuildSetTopics, len(weakest))])
	d.BuildTitle = "Weakest topics · " + timefmt.Date(today.Date)
	return d
}

// weakFlags marks a topic's recall below 80% and struggle above 55%.
func weakFlags(recall, struggle float64) (lowRecall, highStruggle bool) {
	return recall < weakRecall, struggle > weakStruggle
}

// streakTicks lays out the streak scale from 0 to the record (at least a
// week), with at most 61 ticks: a longer record is drawn to scale.
func streakTicks(current, record int) (int, []Tick) {
	top := max(record, current, 7)
	n, filled := top+1, current
	if n > 61 {
		n, filled = 61, (current*60+top/2)/top
		if current > 0 {
			// A short streak against a long record still lights one tick.
			filled = max(1, filled)
		}
	}
	ticks := make([]Tick, n)
	for i := range ticks {
		ticks[i] = Tick{On: i < filled, Major: i%5 == 0}
	}
	return top, ticks
}

// chartScale rounds the largest weekly count up to an even top value with
// round guide lines: at least 4, even up to 10, tens up to 50, then fifties.
func chartScale(most int) int {
	switch {
	case most <= 4:
		return 4
	case most <= 10:
		return most + most%2
	case most <= 50:
		return (most + 9) / 10 * 10
	}
	return (most + 49) / 50 * 50
}

// paceNote compares solves in the last full weeks (the current week is
// left out, being partial) with the same number of weeks before them, as a
// note, the two totals and a full sentence. All are empty until there are
// enough weeks, or with no solves in either window.
func paceNote(t Trends) (note, counts, title string) {
	if len(t.Weeks) < 2*paceWindow+1 {
		return "", "", ""
	}
	full := t.Weeks[len(t.Weeks)-1-2*paceWindow : len(t.Weeks)-1]
	before, last := 0, 0
	for i, w := range full {
		if i < paceWindow {
			before += w.Solved
		} else {
			last += w.Solved
		}
	}
	if before == 0 && last == 0 {
		return "", "", ""
	}
	counts = Count(last) + " vs " + Count(before)
	title = Count(last) + " solved in the last " + Count(paceWindow) + " full weeks, " + Count(before) + " in the " + Count(paceWindow) + " before"
	switch {
	case last*10 < before*8:
		return "pace slipping", counts, title
	case last*10 > before*12:
		return "pace picking up", counts, title
	}
	return "pace steady", counts, title
}

// weakTopicProblems picks, for each topic in order, the attempted problem
// in it with the lowest recall that is not already picked.
func weakTopicProblems(today Today, topics []TopicStat) []string {
	picked := map[string]bool{}
	var slugs []string
	for _, t := range topics {
		best, found := Card{}, false
		for _, c := range today.Cards {
			if picked[c.Problem.Slug] || !slices.Contains(c.Problem.Topics, t.Key) {
				continue
			}
			if !found || c.Retrievability(today.Now) < best.Retrievability(today.Now) {
				best, found = c, true
			}
		}
		if found {
			picked[best.Problem.Slug] = true
			slugs = append(slugs, best.Problem.Slug)
		}
	}
	return slugs
}

// StreakLabel is the big two-digit streak number.
func StreakLabel(n int) string { return fmt.Sprintf("%02d", n) }

// ReviewMeta is a review card's detail line: difficulty, current recall and
// topics, as "Easy · recall 67% · Heap / Design".
func ReviewMeta(item ReviewItem, now time.Time) string {
	var parts []string
	if item.Problem.Difficulty != "" {
		parts = append(parts, item.Problem.Difficulty)
	}
	if item.Card.Reviews > 0 {
		parts = append(parts, "recall "+Percent(item.Card.Retrievability(now)))
	}
	var topics []string
	for _, t := range item.Problem.Topics[:min(3, len(item.Problem.Topics))] {
		topics = append(topics, item.Problem.TopicLabel(t))
	}
	if len(topics) > 0 {
		parts = append(parts, strings.Join(topics, " / "))
	}
	return strings.Join(parts, " · ")
}
