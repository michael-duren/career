package leetgrinder

import (
	"strconv"
	"strings"
)

// Chart geometry for the stats page. Charts are inline SVG drawn on the
// server, so they need no JavaScript; each mark carries a <title> for its
// hover tooltip. The topic and weekly charts keep their tables behind a
// toggle; the difficulty bars carry their numbers as text.

// StatsHeadline is the stats page's row of headline numbers.
type StatsHeadline struct {
	Attempted, Solved int
	// Independent is "—" before the first solve.
	Independent string
	Streak      Streaks
	// GoalMet and Practised count days in the calendar window.
	GoalMet, Practised int
	Due                int
}

// Headline summarises the page's state for its tiles.
func (p StatsPage) Headline() StatsHeadline {
	h := StatsHeadline{Attempted: p.Progress.Attempted, Solved: p.Progress.Solved, Independent: "—", Streak: p.Streaks, Due: p.DueToday}
	if p.Progress.Solved > 0 {
		h.Independent = Percent(float64(p.Progress.Independent) / float64(p.Progress.Solved))
	}
	for _, week := range p.Calendar {
		for _, d := range week {
			if d.Met {
				h.GoalMet++
			}
			if d.Active || d.Met {
				h.Practised++
			}
		}
	}
	return h
}

// DifficultyBar is one difficulty's stacked bar: solved, then attempted but
// not solved, as widths out of 100 of the largest attempted count.
type DifficultyBar struct {
	DifficultyStat
	SolvedWidth, OpenWidth string
}

// DifficultyBars scales the difficulty mix to the largest attempted count.
func (p StatsPage) DifficultyBars() []DifficultyBar {
	most := 0
	for _, d := range p.Difficulties {
		most = max(most, d.Attempted)
	}
	bars := make([]DifficultyBar, 0, len(p.Difficulties))
	for _, d := range p.Difficulties {
		bars = append(bars, DifficultyBar{
			DifficultyStat: d,
			SolvedWidth:    BarWidth(float64(d.Solved), float64(most)),
			OpenWidth:      openWidth(d, most),
		})
	}
	return bars
}

// openWidth is the attempted-but-unsolved segment's width, trimmed so the
// two segments never pass 100 after BarWidth's rounding and minimum.
func openWidth(d DifficultyStat, most int) string {
	open := BarWidth(float64(d.Attempted-d.Solved), float64(most))
	solved, _ := strconv.ParseFloat(BarWidth(float64(d.Solved), float64(most)), 64)
	if w, _ := strconv.ParseFloat(open, 64); w > 0 && solved+w > 100 {
		return num(100 - solved)
	}
	return open
}

// Weekly chart frame, in SVG user units.
const (
	chartWidth  = 640
	chartHeight = 200
	chartLeft   = 36
	// chartRight leaves a gutter right of the plot for the rate chart's
	// latest values; both charts share it so their weeks line up.
	chartRight   = 60
	chartTop     = 16
	chartBottom  = 28
	chartPlotW   = chartWidth - chartLeft - chartRight
	chartPlotH   = chartHeight - chartTop - chartBottom
	chartBaseY   = chartTop + chartPlotH
	chartMaxBarW = 36
	// End labels sit in the gutter, centred on their point's height and
	// at least endLabelGap apart: the largest chart value text is 20 units
	// (phones; keep in sync with .chart-value in style.css).
	endLabelGap = 22
)

// ChartTick is a horizontal grid line with its axis label.
type ChartTick struct {
	Y     string
	Label string
}

// ChartLabel is an x-axis label centred under a week.
type ChartLabel struct {
	X     string
	Label string
}

// ChartBar is one week's column, or in a RateChart only its hover band.
type ChartBar struct {
	// Path draws the column with a rounded top; empty for a zero week.
	Path string
	// HitX and HitW span the week's band, a full-height hover target.
	HitX, HitW string
	// LabelX and LabelY place the value above the column when it is labelled.
	LabelX, LabelY string
	Value          int
	Labelled       bool
	Title          string
}

// SolvedChart is the problems-solved-per-week column chart.
type SolvedChart struct {
	Ticks  []ChartTick
	Bars   []ChartBar
	Labels []ChartLabel
}

// ChartPoint is a marker on a line series.
type ChartPoint struct {
	X, Y  string
	Title string
}

// ChartSeries is one line: its path, broken where a week has no value, its
// markers, and a direct label at its point in the latest week, if any.
type ChartSeries struct {
	Key, Name string
	Path      string
	Points    []ChartPoint
	// EndLabel is the latest week's value, drawn in the gutter at LabelY.
	EndLabel string
	LabelY   string
	labelY   float64
}

// RateChart plots weekly percentages on one 0-100% axis.
type RateChart struct {
	Ticks  []ChartTick
	Series []ChartSeries
	Labels []ChartLabel
	// Hits are per-week hover bands that name every series' value.
	Hits []ChartBar
}

// Empty reports whether no series has a point.
func (c RateChart) Empty() bool {
	for _, s := range c.Series {
		if len(s.Points) > 0 {
			return false
		}
	}
	return true
}

func num(f float64) string { return strconv.FormatFloat(f, 'f', 1, 64) }

// band is the width of each week's slot and centre returns a week's middle.
func band(n int) float64 { return float64(chartPlotW) / float64(max(n, 1)) }

func centre(i, n int) float64 { return chartLeft + band(n)*(float64(i)+0.5) }

// weekLabels labels every week when they fit and every other week, always
// including the latest, when there are many.
func weekLabels(weeks []TrendWeek) []ChartLabel {
	step := 1
	if len(weeks) > 6 {
		step = 2
	}
	var labels []ChartLabel
	for i, w := range weeks {
		if (len(weeks)-1-i)%step == 0 {
			labels = append(labels, ChartLabel{X: num(centre(i, len(weeks))), Label: w.Start.Format("2 Jan")})
		}
	}
	return labels
}

// solvedTop rounds the largest week up to an even number so the middle grid
// line is a whole number of problems.
func solvedTop(most int) int {
	if most <= 1 {
		return 1
	}
	return (most + 1) / 2 * 2
}

// SolvedChart lays out problems solved per week. The tallest and latest
// columns are labelled; every column has a tooltip.
func (t Trends) SolvedChart() SolvedChart {
	top := solvedTop(t.MaxSolved)
	scale := func(v int) float64 { return chartBaseY - float64(chartPlotH)*float64(v)/float64(top) }
	c := SolvedChart{Labels: weekLabels(t.Weeks)}
	ticks := []int{0, top}
	if top > 1 {
		ticks = []int{0, top / 2, top}
	}
	for _, v := range ticks {
		c.Ticks = append(c.Ticks, ChartTick{Y: num(scale(v)), Label: Count(v)})
	}
	n := len(t.Weeks)
	w := min(band(n)*0.6, chartMaxBarW)
	// Label the first tallest column and the latest one.
	tallest := -1
	for i, week := range t.Weeks {
		if tallest < 0 && week.Solved == t.MaxSolved {
			tallest = i
		}
	}
	for i, week := range t.Weeks {
		x := centre(i, n) - w/2
		bar := ChartBar{
			HitX:   num(chartLeft + band(n)*float64(i)),
			HitW:   num(band(n)),
			Value:  week.Solved,
			LabelX: num(centre(i, n)),
			Title:  "Week of " + week.Start.Format("2 Jan") + ": " + Count(week.Solved) + " " + plural(week.Solved, "problem", "problems") + " solved",
		}
		if week.Solved > 0 {
			y := scale(week.Solved)
			r := min(4, w/2, chartBaseY-y)
			bar.Path = "M" + num(x) + "," + num(chartBaseY) +
				"V" + num(y+r) +
				"Q" + num(x) + "," + num(y) + " " + num(x+r) + "," + num(y) +
				"H" + num(x+w-r) +
				"Q" + num(x+w) + "," + num(y) + " " + num(x+w) + "," + num(y+r) +
				"V" + num(chartBaseY) + "Z"
			bar.LabelY = num(y - 5)
			bar.Labelled = i == tallest || i == n-1
		}
		c.Bars = append(c.Bars, bar)
	}
	return c
}

// RateChart plots the weekly independent-solve rate and complexity-check
// accuracy, leaving gaps in weeks without solves or checks.
func (t Trends) RateChart() RateChart {
	scale := func(f float64) float64 { return chartBaseY - float64(chartPlotH)*f }
	c := RateChart{Labels: weekLabels(t.Weeks)}
	for _, f := range []float64{0, 0.5, 1} {
		c.Ticks = append(c.Ticks, ChartTick{Y: num(scale(f)), Label: Percent(f)})
	}
	n := len(t.Weeks)
	series := []struct {
		key, name string
		value     func(TrendWeek) (float64, bool)
		detail    func(TrendWeek) string
	}{
		{"independent", "Independent solves", TrendWeek.IndependentRate, func(w TrendWeek) string {
			return Count(w.Independent) + " of " + Count(w.Solves) + " " + plural(w.Solves, "solve", "solves")
		}},
		{"accuracy", "Complexity checks right", TrendWeek.Accuracy, func(w TrendWeek) string {
			return Count(w.Matched) + " of " + Count(w.Checked) + " " + plural(w.Checked, "check", "checks")
		}},
	}
	hits := make([][]string, n)
	for _, s := range series {
		cs := ChartSeries{Key: s.key, Name: s.name}
		var path strings.Builder
		open := false
		for i, week := range t.Weeks {
			v, ok := s.value(week)
			if !ok {
				open = false
				hits[i] = append(hits[i], s.name+": none")
				continue
			}
			x, y := num(centre(i, n)), num(scale(v))
			if open {
				path.WriteString("L" + x + "," + y)
			} else {
				path.WriteString("M" + x + "," + y)
			}
			open = true
			label := s.name + ": " + Percent(v) + " (" + s.detail(week) + ")"
			cs.Points = append(cs.Points, ChartPoint{X: x, Y: y, Title: "Week of " + week.Start.Format("2 Jan") + ". " + label})
			if i == n-1 {
				cs.EndLabel, cs.labelY = Percent(v), scale(v)
			}
			hits[i] = append(hits[i], label)
		}
		cs.Path = path.String()
		c.Series = append(c.Series, cs)
	}
	// Only the latest week is labelled, in the gutter beside its point.
	// Labels closer than endLabelGap are pushed apart around their middle,
	// then kept between the plot's top and baseline.
	if len(c.Series) == 2 {
		a, b := &c.Series[0], &c.Series[1]
		if a.EndLabel != "" && b.EndLabel != "" {
			upper, lower := a, b
			if a.labelY > b.labelY {
				upper, lower = b, a
			}
			if gap := lower.labelY - upper.labelY; gap < endLabelGap {
				mid := (upper.labelY + lower.labelY) / 2
				upper.labelY, lower.labelY = mid-endLabelGap/2.0, mid+endLabelGap/2.0
			}
			if shift := lower.labelY - chartBaseY; shift > 0 {
				upper.labelY, lower.labelY = upper.labelY-shift, lower.labelY-shift
			}
			if shift := chartTop - upper.labelY; shift > 0 {
				upper.labelY, lower.labelY = upper.labelY+shift, lower.labelY+shift
			}
		}
	}
	for i := range c.Series {
		c.Series[i].LabelY = num(c.Series[i].labelY)
	}
	for i, week := range t.Weeks {
		c.Hits = append(c.Hits, ChartBar{
			HitX:  num(chartLeft + band(n)*float64(i)),
			HitW:  num(band(n)),
			Title: "Week of " + week.Start.Format("2 Jan") + ". " + strings.Join(hits[i], ". "),
		})
	}
	return c
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// Chart frame values for templates.
func ChartViewBox() string    { return "0 0 " + Count(chartWidth) + " " + Count(chartHeight) }
func ChartLeft() string       { return Count(chartLeft) }
func ChartRight() string      { return Count(chartWidth - chartRight) }
func ChartGutterX() string    { return Count(chartWidth - chartRight + 10) }
func ChartBaseY() string      { return Count(chartBaseY) }
func ChartAxisLabelY() string { return Count(chartBaseY + 18) }
func ChartTop() string        { return Count(chartTop) }
func ChartPlotH() string      { return Count(chartPlotH) }
