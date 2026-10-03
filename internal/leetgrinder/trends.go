package leetgrinder

import (
	"strconv"
	"time"
)

// TrendWeek is one Monday-to-Sunday week of practice, by local date in the
// settings time zone.
type TrendWeek struct {
	// Start is the week's Monday, as a Date value.
	Start time.Time
	// Solved counts distinct problems with a solved attempt that week.
	Solved int
	// Solves counts solved attempts and Independent those without help.
	Solves, Independent int
	// Minutes and Timed sum the minutes and count the attempts of each
	// difficulty ("Easy", "Medium", "Hard").
	Minutes, Timed map[string]int
	// Checked counts stated complexities a completed analysis judged, and
	// Matched those it judged right.
	Checked, Matched int
}

// IndependentRate is the share of the week's solves made without help, and
// false when the week has no solves.
func (w TrendWeek) IndependentRate() (float64, bool) {
	if w.Solves == 0 {
		return 0, false
	}
	return float64(w.Independent) / float64(w.Solves), true
}

// AverageMinutes is the week's average minutes per attempt at difficulty
// d, and false when it has none.
func (w TrendWeek) AverageMinutes(d string) (int, bool) {
	if w.Timed[d] == 0 {
		return 0, false
	}
	return (w.Minutes[d] + w.Timed[d]/2) / w.Timed[d], true
}

// Accuracy is the share of judged stated complexities that were right, and
// false when none were judged.
func (w TrendWeek) Accuracy() (float64, bool) {
	if w.Checked == 0 {
		return 0, false
	}
	return float64(w.Matched) / float64(w.Checked), true
}

// TrendDifficulties are the difficulties average minutes are shown for.
var TrendDifficulties = []string{"Easy", "Medium", "Hard"}

// Trends are weekly series over the last weeks, oldest first, ending with
// the week that holds today. Weeks before the week of the first attempt are
// left out, so a new account shows only the weeks it has; with no attempts
// it is empty.
type Trends struct {
	Weeks []TrendWeek
	// MaxSolved is the largest weekly Solved, for scaling its bars.
	MaxSolved int
	// Limit is the most weeks shown.
	Limit int
}

// WeeklyTrends builds the last weeks of trends from today's state.
func WeeklyTrends(today Today, weeks int) Trends {
	loc := today.Settings.Location()
	state := today.State
	if len(state.Attempts) == 0 || weeks < 1 {
		return Trends{}
	}
	monday := func(d time.Time) time.Time { return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7)) }
	last := monday(today.Date)
	first := last.AddDate(0, 0, -7*(weeks-1))
	// Attempts are newest first, so the last one is the first attempt.
	if start := monday(Date(state.Attempts[len(state.Attempts)-1].CreatedAt, loc)); start.After(first) {
		first = start
	}
	t := Trends{Limit: weeks}
	index := map[time.Time]int{}
	for d := first; !d.After(last); d = d.AddDate(0, 0, 7) {
		index[d] = len(t.Weeks)
		t.Weeks = append(t.Weeks, TrendWeek{Start: d, Minutes: map[string]int{}, Timed: map[string]int{}})
	}
	solved := map[time.Time]map[string]bool{}
	for _, a := range state.Attempts {
		i, ok := index[monday(Date(a.CreatedAt, loc))]
		if !ok {
			continue
		}
		w := &t.Weeks[i]
		if a.Outcome == "solved" {
			w.Solves++
			if !a.Assisted {
				w.Independent++
			}
			if solved[w.Start] == nil {
				solved[w.Start] = map[string]bool{}
			}
			solved[w.Start][a.ProblemSlug] = true
		}
		if d := state.Problem(a.ProblemSlug).Difficulty; d != "" {
			w.Minutes[d] += a.Minutes
			w.Timed[d]++
		}
		if an, ok := state.Analyses[a.ID]; ok && an.Done() {
			for _, m := range []*bool{an.TimeMatches, an.SpaceMatches} {
				if m != nil {
					w.Checked++
					if *m {
						w.Matched++
					}
				}
			}
		}
	}
	for i := range t.Weeks {
		t.Weeks[i].Solved = len(solved[t.Weeks[i].Start])
		t.MaxSolved = max(t.MaxSolved, t.Weeks[i].Solved)
	}
	return t
}

// BarWidth is a bar's width out of 100 for value v of most, at least 1 for
// a positive value so it stays visible.
func BarWidth(v, most float64) string {
	if most <= 0 || v <= 0 {
		return "0"
	}
	return strconv.FormatFloat(min(max(100*v/most, 1), 100), 'f', 1, 64)
}

// RateLabel shows a share as a percentage, or a dash when there is none.
func RateLabel(f float64, ok bool) string {
	if !ok {
		return "—"
	}
	return Percent(f)
}

// MinutesLabel shows an average in minutes, or a dash when there is none.
func MinutesLabel(n int, ok bool) string {
	if !ok {
		return "—"
	}
	return strconv.Itoa(n) + " min"
}
