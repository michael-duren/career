package leetgrinder

import (
	"cmp"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Spot-check rates, in percent of recalled quick reviews.
const (
	DefaultSpotCheckRate = 10
	MaxSpotCheckRate     = 25
	// ElevatedSpotCheckRate applies while spot checks keep failing; see
	// SpotChecks.
	ElevatedSpotCheckRate = 25
	// QuickReviewMinutes is the default time a quick review takes.
	QuickReviewMinutes = 3
)

// QuickEligible reports whether c can be quick-reviewed today: it is due or
// flagged, and its next review is not required to be a full attempt (a
// forgotten quick review or a spot check).
func (t Today) QuickEligible(c Card) bool {
	return c.Reviews > 0 && (t.Due(c) || c.Reviewable() && c.Flagged()) && c.RequiredMode() == ""
}

// QuickCards lists the quick-eligible cards, flagged first, then lowest
// recall now, then the most overdue. Cards in skip are left out.
func (t Today) QuickCards(skip map[string]bool) []Card {
	type candidate struct {
		card Card
		r    float64
	}
	var candidates []candidate
	for _, c := range t.Cards {
		if !skip[c.Problem.Slug] && t.QuickEligible(c) {
			candidates = append(candidates, candidate{c, c.Retrievability(t.Now)})
		}
	}
	flagRank := func(c Card) int {
		if c.Flagged() {
			return 0
		}
		return 1
	}
	slices.SortFunc(candidates, func(a, b candidate) int {
		return cmp.Or(cmp.Compare(flagRank(a.card), flagRank(b.card)), cmp.Compare(a.r, b.r), a.card.Due.Compare(b.card.Due), cmp.Compare(a.card.Problem.Slug, b.card.Problem.Slug))
	})
	out := make([]Card, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.card)
	}
	return out
}

// DueCount counts the cards due by the end of today, flagged ones and due
// spot checks included.
func (t Today) DueCount() int {
	n := 0
	for _, c := range t.Cards {
		if t.Due(c) {
			n++
		}
	}
	return n
}

// QuickReviewsOn lists the newest quick review of each problem
// quick-reviewed on date, newest first.
func QuickReviewsOn(state State, date time.Time, loc *time.Location) []Attempt {
	return QuickReviewsSince(state, StartOfDate(date, loc), EndOfDate(date, loc))
}

// QuickReviewsSince lists the newest quick review of each problem
// quick-reviewed at or after from and before to, newest first.
func QuickReviewsSince(state State, from, to time.Time) []Attempt {
	seen := map[string]bool{}
	var out []Attempt
	// Attempts are newest first, so the first of each problem is its newest.
	for _, a := range state.Attempts {
		if a.Quick() && !a.CreatedAt.Before(from) && a.CreatedAt.Before(to) && !seen[a.ProblemSlug] {
			seen[a.ProblemSlug] = true
			out = append(out, a)
		}
	}
	return out
}

// ComplexityCheck compares a stated complexity with the problem's optimum.
type ComplexityCheck struct {
	Stated, Optimal string
}

// Known reports whether both values are known, so the check has a verdict.
func (c ComplexityCheck) Known() bool { return c.Stated != "" && c.Optimal != "" }

// Matches reports whether the stated complexity is the optimum.
func (c ComplexityCheck) Matches() bool { return c.Known() && c.Stated == c.Optimal }

// QuickCheck is the complexity check of a quick review: the stated time and
// space against the problem's optimal values.
type QuickCheck struct {
	Time, Space ComplexityCheck
}

// CheckQuick compares stated time and space complexities (normalised) with
// p's optimum.
func CheckQuick(p Problem, timeComplexity, spaceComplexity string) QuickCheck {
	return QuickCheck{Time: ComplexityCheck{timeComplexity, p.OptimalTime}, Space: ComplexityCheck{spaceComplexity, p.OptimalSpace}}
}

// Missed is what the check found wrong, as a Missed value: MissedTime,
// MissedSpace, MissedBoth, or MissedDetail when neither complexity was
// judged wrong.
func (q QuickCheck) Missed() string {
	timeWrong, spaceWrong := q.Time.Known() && !q.Time.Matches(), q.Space.Known() && !q.Space.Matches()
	switch {
	case timeWrong && spaceWrong:
		return MissedBoth
	case timeWrong:
		return MissedTime
	case spaceWrong:
		return MissedSpace
	}
	return MissedDetail
}

// SpotCheckStatus summarises resolved spot checks and the rate that applies
// to the next recalled quick review.
type SpotCheckStatus struct {
	Passed, Failed int
	// Elevated reports that 2 of the last 5 spot checks failed and 3 in a
	// row have not passed since, so ElevatedSpotCheckRate applies.
	Elevated bool
	// Rate is the current rate in percent.
	Rate int
}

// SpotChecks resolves every spot check in state and works out the current
// rate from the settings' rate base. A spot check resolves with the next
// full attempt on its problem: it passes when that attempt is a solve
// without help, and fails otherwise. A quick review logged first leaves it
// unresolved. When 2 of the last 5 resolved spot checks fail, the rate goes
// up to ElevatedSpotCheckRate until 3 in a row pass.
func SpotChecks(state State, base int) SpotCheckStatus {
	type result struct {
		at   time.Time
		id   string
		pass bool
	}
	var results []result
	for _, list := range attemptsBySlug(state.Attempts) {
		pending := false
		for _, a := range list {
			if pending && !a.Quick() {
				results = append(results, result{a.CreatedAt, a.ID, a.Outcome == "solved" && !a.Assisted})
			}
			pending = a.Quick() && a.Outcome == QuickRecalled && a.SpotCheck
		}
	}
	slices.SortFunc(results, func(a, b result) int { return cmp.Or(a.at.Compare(b.at), cmp.Compare(a.id, b.id)) })
	s := SpotCheckStatus{}
	streak := 0
	for i, r := range results {
		if r.pass {
			s.Passed++
		} else {
			s.Failed++
		}
		if s.Elevated {
			if r.pass {
				streak++
			} else {
				streak = 0
			}
			if streak >= 3 {
				s.Elevated = false
			}
			continue
		}
		failed := 0
		for _, w := range results[max(0, i-4) : i+1] {
			if !w.pass {
				failed++
			}
		}
		if failed >= 2 {
			s.Elevated, streak = true, 0
		}
	}
	s.Rate = base
	if s.Elevated {
		s.Rate = max(base, ElevatedSpotCheckRate)
	}
	return s
}

// PickSpotCheck decides whether a recalled quick review becomes a spot
// check at rate percent, given r drawn uniformly from [0, 1).
func PickSpotCheck(rate int, r float64) bool {
	return r*100 < float64(rate)
}

// QuickStats counts quick reviews by grade.
type QuickStats struct {
	Count, Recalled, Partial, Forgot int
}

// RecallRate is the share of quick reviews graded recalled, and false when
// there are none.
func (q QuickStats) RecallRate() (float64, bool) {
	if q.Count == 0 {
		return 0, false
	}
	return float64(q.Recalled) / float64(q.Count), true
}

// CountQuickReviews counts every quick review in state by grade.
func CountQuickReviews(state State) QuickStats {
	var q QuickStats
	for _, a := range state.Attempts {
		if !a.Quick() {
			continue
		}
		q.Count++
		switch a.Outcome {
		case QuickRecalled:
			q.Recalled++
		case QuickPartial:
			q.Partial++
		case QuickForgot:
			q.Forgot++
		}
	}
	return q
}

// QuickForm is a quick review as entered: the answer before Reveal, and the
// grade after it.
type QuickForm struct {
	ID       string
	Approach string
	// Time and Space are the complexity controls as submitted.
	Time, Space ComplexityInput
	Grade       string
	Missed      string
	Minutes     string
	// Batch is the Clear backlog batch the review belongs to, or "".
	Batch string
	Error string
}

// NewQuickForm is an empty quick review with a new attempt id.
func NewQuickForm(id, batch string) QuickForm {
	return QuickForm{ID: id, Minutes: strconv.Itoa(QuickReviewMinutes), Batch: batch}
}

// QuickReviewPage is the quick review of one problem: the answer form, or
// after Reveal the answer beside the last solution and the grade form.
type QuickReviewPage struct {
	Problem Problem
	Form    QuickForm
	// Revealed is true after Reveal, with the complexity check, the newest
	// attempt with code, and the newest notes of a full attempt.
	Revealed bool
	Check    QuickCheck
	LastCode *Attempt
	Notes    string
	// Blocked explains why the problem cannot be quick-reviewed now, or "".
	Blocked string
	// Suggested is the card's SuggestedMode.
	Suggested string
	Location  *time.Location
}

// NewQuickReviewPage builds the quick review page of slug from today.
// Blocked is set when the problem cannot be quick-reviewed now.
func NewQuickReviewPage(today Today, problem Problem, form QuickForm) QuickReviewPage {
	page := QuickReviewPage{Problem: problem, Form: form, Location: today.Settings.Location()}
	c, ok := today.Card(problem.Slug)
	page.Blocked = QuickBlocked(today, c, ok)
	page.Suggested = c.SuggestedMode()
	return page
}

// QuickBlocked explains why a card cannot be quick-reviewed today, or is ""
// when it can. ok is false when the problem has no card.
func QuickBlocked(today Today, c Card, ok bool) string {
	switch {
	case !ok || c.Reviews == 0:
		return "Quick review is for problems you have attempted before. Log a full attempt first."
	case c.RequiresFull():
		return "You forgot this problem in your last quick review, so it needs a full re-solve before it can be quick-reviewed again."
	case c.SpotCheck() && today.Due(c):
		return "This review is a spot check of an earlier quick review: re-solve the problem in full."
	case !today.QuickEligible(c):
		return "This problem is not due or flagged for review, so there is nothing to quick-review."
	}
	return ""
}

// Reveal fills the revealed part of the page from the problem's attempts,
// newest first: the newest code and the newest notes of a full attempt.
func (p *QuickReviewPage) Reveal(attempts []Attempt, timeComplexity, spaceComplexity string) {
	p.Revealed = true
	p.Check = CheckQuick(p.Problem, timeComplexity, spaceComplexity)
	if p.Form.Missed == "" {
		p.Form.Missed = p.Check.Missed()
	}
	for i := range attempts {
		a := attempts[i]
		if a.Quick() {
			continue
		}
		if p.LastCode == nil && a.Code != "" {
			p.LastCode = &a
		}
		if p.Notes == "" {
			p.Notes = a.Notes
		}
	}
}

// QuickURL is the quick review page of slug, in batch when set.
func QuickURL(slug, batch string) string {
	u := ProblemURL(slug) + "/quick"
	if batch != "" {
		u += "?batch=" + url.QueryEscape(batch)
	}
	return u
}

// Batch is a Clear backlog run: when it started and the flagged and due
// counts then. It travels in the URL, so the run needs no stored state.
type Batch struct {
	Since        time.Time
	Flagged, Due int
}

// String encodes the batch for a URL, as "since.flagged.due" with since in
// Unix seconds.
func (b Batch) String() string {
	return strconv.FormatInt(b.Since.Unix(), 10) + "." + strconv.Itoa(b.Flagged) + "." + strconv.Itoa(b.Due)
}

// ParseBatch reads a batch String. Counts must be 0 to 100000 and the start
// a positive Unix time.
func ParseBatch(s string) (Batch, bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Batch{}, false
	}
	since, err := strconv.ParseInt(parts[0], 10, 64)
	flagged, errF := strconv.Atoi(parts[1])
	due, errD := strconv.Atoi(parts[2])
	if err != nil || errF != nil || errD != nil || since <= 0 || since > 1<<40 || flagged < 0 || flagged > 100000 || due < 0 || due > 100000 {
		return Batch{}, false
	}
	return Batch{Since: time.Unix(since, 0), Flagged: flagged, Due: due}, true
}

// BacklogPage is the Clear backlog view of a batch: counts at its start and
// now, the quick-eligible problems not yet reviewed in it, and those its
// reviews left on the list.
type BacklogPage struct {
	Batch        Batch
	Flagged, Due int
	Remaining    []Card
	Reviewed     []Attempt
	StillOnList  []Attempt
	Today        Today
	BatchToken   string
}

// NewBacklogPage builds the backlog view of batch from today.
func NewBacklogPage(today Today, batch Batch) BacklogPage {
	page := BacklogPage{Batch: batch, Today: today, BatchToken: batch.String(), Flagged: len(today.FlaggedCards()), Due: today.DueCount()}
	page.Reviewed = QuickReviewsSince(today.State, batch.Since, today.Now.Add(time.Second))
	skip := map[string]bool{}
	for _, a := range page.Reviewed {
		skip[a.ProblemSlug] = true
		if a.Outcome != QuickRecalled {
			page.StillOnList = append(page.StillOnList, a)
		}
	}
	page.Remaining = today.QuickCards(skip)
	return page
}

// StartBatch starts a Clear backlog batch now, with today's counts.
func StartBatch(today Today) Batch {
	return Batch{Since: today.Now.Truncate(time.Second), Flagged: len(today.FlaggedCards()), Due: today.DueCount()}
}

// BacklogURL is the Clear backlog view of batch; "" starts a new batch.
func BacklogURL(batch string) string {
	if batch == "" {
		return "/leetgrinder/reviews/backlog"
	}
	return "/leetgrinder/reviews/backlog?batch=" + url.QueryEscape(batch)
}

func problemUnit(n int) string {
	if n == 1 {
		return "problem"
	}
	return "problems"
}
