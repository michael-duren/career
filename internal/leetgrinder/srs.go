package leetgrinder

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"
)

// Default FSRS parameters, minus the sub-day learning steps: only one attempt
// per local day counts, so minute-scale steps would only mean "due tomorrow".
var scheduler = func() *fsrs.FSRS {
	p := fsrs.DefaultParam()
	p.EnableShortTerm = false
	return fsrs.NewFSRS(p)
}()

// ReviewRating maps an attempt onto an FSRS rating. A recalled quick review
// rates Good, a normal solve's interval; partial rates Hard and forgot Again.
func ReviewRating(a Attempt) fsrs.Rating {
	switch {
	case a.Outcome == QuickRecalled:
		return fsrs.Good
	case a.Outcome == QuickPartial:
		return fsrs.Hard
	case a.Outcome == QuickForgot:
		return fsrs.Again
	case a.Outcome == "unfinished":
		return fsrs.Again
	case a.Outcome == "struggled" || a.Assisted:
		return fsrs.Hard
	case a.Minutes > ReviewSlotMinutes:
		return fsrs.Good
	default:
		return fsrs.Easy
	}
}

// Card is the spaced-repetition state of one attempted problem. It is derived
// by replaying attempts, never stored, so corrections flow through.
type Card struct {
	Problem Problem
	// Last is the attempt that counted most recently: the newest attempt.
	Last Attempt
	// LastFull is the newest attempt that was not a quick review.
	LastFull Attempt
	Reviews  int
	// Due is the effective due time: the FSRS due time, or the flag's due
	// time when that is earlier.
	Due time.Time
	// FSRSDue is when FSRS alone schedules the next review.
	FSRSDue time.Time
	// Flag is set when Last was a struggle, was marked for review, or its
	// analysis found a mistake.
	Flag *Flag
	// Struggled reports that some attempt so far was a struggle (see
	// Struggle), even one a later attempt the same day replaced.
	Struggled bool
	fsrs      fsrs.Card
	loc       *time.Location
}

// Flag marks a problem for review the day after its latest attempt said so:
// the attempt was a struggle (see Struggle), its analysis judged a stated
// complexity wrong or the solution not optimal, or the learner asked to
// review it or took a simpler approach for time. It clears with the next
// attempt, or a re-analysis without those verdicts when the analysis alone
// raised it.
type Flag struct {
	// Date is the earliest local date that raised the flag, of MarkedDate
	// and AnalysisDate; the other two are zero when that source is absent.
	// AttemptDate is the attempt's own day, which a struggle's reason names.
	Date, MarkedDate, AnalysisDate, AttemptDate time.Time
	TimeWrong, SpaceWrong                       bool
	NotOptimal                                  bool
	// Actual and optimal complexities, for the not-optimal reason.
	ActualTime, ActualSpace   string
	OptimalTime, OptimalSpace string
	// WantsReview and Suboptimal are the learner's own marks, and Struggle
	// the attempt's struggle label, or "".
	WantsReview, Suboptimal bool
	Struggle                string
	// Quick is a partial or forgotten quick review's grade, or "". A partial
	// one says what it Missed; a forgotten one requires a full re-solve.
	Quick, Missed string
}

// RequiresFull reports that the flag came from a forgotten quick review, so
// only a full attempt clears it.
func (f Flag) RequiresFull() bool { return f.Quick == QuickForgot }

// Flagged reports whether the card carries a flag.
func (c Card) Flagged() bool { return c.Flag != nil }

// Reviewable reports whether the card is in the review queue: only problems
// you struggled with, or that carry a flag, are scheduled for review. A
// pending spot check is left out until it is due (see DueBefore), so it is
// not seen before then.
func (c Card) Reviewable() bool { return c.Reviews > 0 && (c.Struggled || c.Flagged()) }

// DueBefore reports whether the card is in the review queue, or a spot
// check, and due before t.
func (c Card) DueBefore(t time.Time) bool {
	return (c.Reviewable() || c.SpotCheck()) && c.Due.Before(t)
}

// SpotCheck reports that the last attempt was a recalled quick review picked
// for a spot check: the next review must be a full re-solve. Callers show
// it only once the card is due.
func (c Card) SpotCheck() bool {
	return c.Reviews > 0 && c.Last.Quick() && c.Last.Outcome == QuickRecalled && c.Last.SpotCheck
}

// RequiresFull reports that the last attempt was a forgotten quick review,
// so only a full attempt can review the problem next.
func (c Card) RequiresFull() bool { return c.Flag != nil && c.Flag.RequiresFull() }

// RequiredMode is ModeFull when the next review must be a full attempt (a
// forgotten quick review, or a spot check), or "" when either mode will do.
func (c Card) RequiredMode() string {
	if c.RequiresFull() || c.SpotCheck() {
		return ModeFull
	}
	return ""
}

// SuggestedMode hints at the review mode: ModeFull when a full attempt is
// required or the last full attempt was struggled or unfinished, otherwise
// ModeQuick. A quick review is still allowed unless RequiredMode says no.
func (c Card) SuggestedMode() string {
	if c.RequiredMode() != "" || c.LastFull.Outcome == "struggled" || c.LastFull.Outcome == "unfinished" {
		return ModeFull
	}
	return ModeQuick
}

// go-fsrs counts elapsed days between UTC calendar dates. Feeding it local
// wall-clock times labelled as UTC makes those days the learner's local days.
func wallClock(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), l.Hour(), l.Minute(), l.Second(), l.Nanosecond(), time.UTC)
}

func fromWallClock(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// Retrievability is the estimated recall probability at t.
func (c Card) Retrievability(t time.Time) float64 {
	loc := c.loc
	if loc == nil {
		loc = time.UTC
	}
	r, err := scheduler.Retrievability(c.fsrs, wallClock(t, loc))
	if err != nil {
		return 0
	}
	return r
}

// flagFor derives the flag of an attempt from its outcome, the learner's
// marks and its analysis, or nil. A quick review flags only when partial or
// forgotten; a recalled one clears every flag.
func flagFor(a Attempt, problem Problem, analyses map[string]Analysis, loc *time.Location) *Flag {
	if a.Quick() {
		if a.Outcome == QuickRecalled {
			return nil
		}
		date := Date(a.CreatedAt, loc)
		return &Flag{Date: date, MarkedDate: date, AttemptDate: date, Quick: a.Outcome, Missed: a.Missed}
	}
	f := &Flag{WantsReview: a.WantsReview, Suboptimal: a.Approach == ApproachSuboptimal, Struggle: Struggle(a)}
	if a.SelfFlagged() {
		marked := a.MarkedAt
		if marked.IsZero() {
			marked = a.CreatedAt
		}
		f.MarkedDate = Date(marked, loc)
		f.Date = f.MarkedDate
		f.AttemptDate = Date(a.CreatedAt, loc)
	}
	if an, ok := analyses[a.ID]; ok && an.Done() && (isFalse(an.TimeMatches) || isFalse(an.SpaceMatches) || isFalse(an.Optimal)) {
		f.TimeWrong, f.SpaceWrong, f.NotOptimal = isFalse(an.TimeMatches), isFalse(an.SpaceMatches), isFalse(an.Optimal)
		f.ActualTime, f.ActualSpace, f.OptimalTime, f.OptimalSpace = an.ActualTime, an.ActualSpace, problem.OptimalTime, problem.OptimalSpace
		f.AnalysisDate = Date(an.UpdatedAt, loc)
		if f.Date.IsZero() || f.AnalysisDate.Before(f.Date) {
			f.Date = f.AnalysisDate
		}
	}
	if f.Date.IsZero() {
		return nil
	}
	return f
}

// QuickPartialDays is how many days after a partial quick review the
// problem is due again; other flags are due the next day.
const QuickPartialDays = 2

// DueAt is the start of the local day after the flag was raised, or for a
// partial quick review QuickPartialDays days after.
func (f Flag) DueAt(loc *time.Location) time.Time {
	days := 1
	if f.Quick == QuickPartial {
		days = QuickPartialDays
	}
	return time.Date(f.Date.Year(), f.Date.Month(), f.Date.Day()+days, 0, 0, 0, 0, loc)
}

// Reason explains the flag, e.g. "Time complexity judged wrong 2 days ago",
// "Not optimal: O(n²) vs O(n)" or "Marked for review yesterday". The
// analysis's verdicts come first, being the most specific.
func (f Flag) Reason(now time.Time, loc *time.Location) string {
	today := Date(now, loc)
	when := daysAgo(f.AnalysisDate, today)
	switch {
	case f.Quick == QuickPartial:
		return MissedLabel(f.Missed) + " missed in quick review " + daysAgo(f.AttemptDate, today)
	case f.Quick == QuickForgot:
		return "Forgot in quick review " + daysAgo(f.AttemptDate, today) + " · full re-solve required"
	case f.TimeWrong && f.SpaceWrong:
		return "Time and space complexity judged wrong " + when
	case f.TimeWrong:
		return "Time complexity judged wrong " + when
	case f.SpaceWrong:
		return "Space complexity judged wrong " + when
	case f.NotOptimal && f.OptimalTime != "" && f.ActualTime != "" && f.ActualTime != f.OptimalTime:
		return fmt.Sprintf("Not optimal: %s vs %s", f.ActualTime, f.OptimalTime)
	case f.NotOptimal && f.OptimalSpace != "" && f.ActualSpace != "" && f.ActualSpace != f.OptimalSpace:
		return fmt.Sprintf("Not optimal: space %s vs %s", f.ActualSpace, f.OptimalSpace)
	case f.NotOptimal:
		return "Not optimal"
	case f.Struggle != "":
		return f.Struggle + " " + daysAgo(f.AttemptDate, today)
	case f.Suboptimal:
		return "Took a simpler approach " + daysAgo(f.MarkedDate, today)
	}
	return "Marked for review " + daysAgo(f.MarkedDate, today)
}

// Label is a short form for badges, e.g. "Flagged: time complexity judged wrong".
func (f Flag) Label() string {
	switch {
	case f.Quick == QuickPartial:
		return "Flagged: missed in quick review"
	case f.Quick == QuickForgot:
		return "Flagged: full re-solve required"
	case f.TimeWrong && f.SpaceWrong:
		return "Flagged: time and space complexity judged wrong"
	case f.TimeWrong:
		return "Flagged: time complexity judged wrong"
	case f.SpaceWrong:
		return "Flagged: space complexity judged wrong"
	case f.NotOptimal:
		return "Flagged: not optimal"
	case f.Struggle != "":
		return "Flagged: " + strings.ToLower(f.Struggle)
	case f.Suboptimal:
		return "Flagged: simpler approach taken"
	}
	return "Flagged: marked for review"
}

func daysAgo(from, to time.Time) string {
	switch days := DaysBetween(from, to); days {
	case 0:
		return "today"
	case 1:
		return "yesterday"
	default:
		return fmt.Sprintf("%d days ago", days)
	}
}

// replaySlug replays one problem's attempts, sorted oldest first, through
// FSRS. Only the last attempt on each local day counts. visit, when set,
// sees each local day with the card as it stood before that day, and
// whether the day had an attempt that was not a quick review.
func replaySlug(problem Problem, list []Attempt, loc *time.Location, visit func(date time.Time, before Card, full bool)) Card {
	card := Card{Problem: problem, loc: loc}
	start := 0
	for i, a := range list {
		date := Date(a.CreatedAt, loc)
		if i+1 < len(list) && Date(list[i+1].CreatedAt, loc).Equal(date) {
			continue
		}
		day := list[start : i+1]
		full := slices.IndexFunc(day, func(a Attempt) bool { return !a.Quick() })
		if visit != nil {
			visit(date, card, full >= 0)
		}
		if slices.ContainsFunc(day, func(a Attempt) bool { return Struggle(a) != "" }) {
			card.Struggled = true
		}
		for _, a := range day {
			if !a.Quick() {
				card.LastFull = a
			}
		}
		start = i + 1
		state := card.fsrs
		if card.Reviews == 0 {
			state = fsrs.NewCard(wallClock(a.CreatedAt, loc))
		}
		info, err := scheduler.Next(state, wallClock(a.CreatedAt, loc), ReviewRating(a))
		if err != nil {
			continue
		}
		card.fsrs, card.Last = info.Card, a
		card.FSRSDue = fromWallClock(info.Card.Due, loc)
		card.Due = card.FSRSDue
		card.Reviews++
	}
	return card
}

// attemptsBySlug groups attempts by problem, each list oldest first.
func attemptsBySlug(attempts []Attempt) map[string][]Attempt {
	bySlug := map[string][]Attempt{}
	for _, a := range attempts {
		bySlug[a.ProblemSlug] = append(bySlug[a.ProblemSlug], a)
	}
	for _, list := range bySlug {
		slices.SortFunc(list, func(a, b Attempt) int {
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
		})
	}
	return bySlug
}

// slugReplay is one problem's replay: each counted local day with the card
// as it stood before that day, and the card after the last day.
type slugReplay struct {
	problem Problem
	list    []Attempt
	dates   []time.Time
	befores []Card
	// full reports, for each date, whether it had an attempt that was not
	// a quick review.
	full  []bool
	final Card
	// ordered reports that the attempts' local dates never go back, as they
	// can when a clock change moves back across midnight. While it holds,
	// dates are strictly increasing, which before's binary search needs.
	ordered bool
}

// replayAll replays each problem's attempts in state through FSRS once.
func replayAll(state State, loc *time.Location) map[string]*slugReplay {
	replays := map[string]*slugReplay{}
	for slug, list := range attemptsBySlug(state.Attempts) {
		r := &slugReplay{problem: state.Problem(slug), list: list, ordered: true}
		for i := 1; i < len(list); i++ {
			if Date(list[i].CreatedAt, loc).Before(Date(list[i-1].CreatedAt, loc)) {
				r.ordered = false
			}
		}
		r.final = replaySlug(r.problem, list, loc, func(date time.Time, before Card, full bool) {
			r.dates, r.befores, r.full = append(r.dates, date), append(r.befores, before), append(r.full, full)
		})
		replays[slug] = r
	}
	return replays
}

// before is the card replayed from the attempts on local dates before date.
// While dates are ordered those attempts are whole days at the start of the
// replay, so the card is the one the replay held when it reached date.
func (r *slugReplay) before(date time.Time, loc *time.Location) Card {
	if !r.ordered {
		var earlier []Attempt
		for _, a := range r.list {
			if Date(a.CreatedAt, loc).Before(date) {
				earlier = append(earlier, a)
			}
		}
		return replaySlug(r.problem, earlier, loc, nil)
	}
	// Counted days are distinct, so ordered dates are strictly increasing.
	i, _ := slices.BinarySearchFunc(r.dates, date, time.Time.Compare)
	if i < len(r.befores) {
		return r.befores[i]
	}
	return r.final
}

// withFlag applies the flag of the card's last attempt, if any.
func withFlag(card Card, analyses map[string]Analysis, loc *time.Location) Card {
	if f := flagFor(card.Last, card.Problem, analyses, loc); f != nil {
		card.Flag = f
		if due := f.DueAt(loc); due.Before(card.Due) {
			card.Due = due
		}
	}
	return card
}

// cardsFrom turns replays into flagged cards ordered by due time.
func cardsFrom(replays map[string]*slugReplay, state State, loc *time.Location) []Card {
	cards := make([]Card, 0)
	for _, r := range replays {
		if r.final.Reviews == 0 {
			continue
		}
		cards = append(cards, withFlag(r.final, state.Analyses, loc))
	}
	slices.SortFunc(cards, func(a, b Card) int {
		return cmp.Or(a.Due.Compare(b.Due), cmp.Compare(a.Problem.Slug, b.Problem.Slug))
	})
	return cards
}

// BuildCards replays state's attempts through FSRS, one card per attempted
// problem, and applies flags. Cards come back ordered by due time.
func BuildCards(state State, loc *time.Location) []Card {
	return cardsFrom(replayAll(state, loc), state, loc)
}

// Replay is a state's attempts replayed through FSRS once, in one time
// zone. A request that plans reviews and then builds today's view shares it
// (see NewTodayFrom) instead of replaying the history for each. A replay
// must not outlive the state it was built from: once attempts, problems or
// analyses change, it is stale.
type Replay struct {
	// zone is the time zone's name; "" for the zero Replay.
	zone    string
	replays map[string]*slugReplay
	cards   []Card
}

// ReplayAttempts replays state's attempts in loc. The replay holds for any
// state with the same attempts, problems and analyses; goals and plans may
// change.
func ReplayAttempts(state State, loc *time.Location) Replay {
	replays := replayAll(state, loc)
	return Replay{zone: loc.String(), replays: replays, cards: cardsFrom(replays, state, loc)}
}

// Cards are the replay's cards, as BuildCards returns them. They are
// shared, so callers must not modify them.
func (r Replay) Cards() []Card { return r.cards }

// DueCards lists reviewable cards due by the end of date, flagged first, then lowest
// estimated recall, then the most overdue. Cards in skip are left out.
func DueCards(cards []Card, date time.Time, loc *time.Location, skip map[string]bool) []Card {
	end := EndOfDate(date, loc)
	type candidate struct {
		card Card
		r    float64
	}
	var candidates []candidate
	for _, c := range cards {
		if !skip[c.Problem.Slug] && c.DueBefore(end) {
			candidates = append(candidates, candidate{c, c.Retrievability(end)})
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

// PlanReviews extends a frozen plan for date to slots picks. Existing picks
// are kept in order; new picks come from DueCards, leaving out problems first
// attempted on date.
func PlanReviews(cards []Card, date time.Time, loc *time.Location, slots int, existing []string) []string {
	plan := slices.Clone(existing)
	if len(plan) >= slots {
		return plan
	}
	skip := map[string]bool{}
	for _, slug := range plan {
		skip[slug] = true
	}
	for _, c := range cards {
		if c.firstAttemptOn(date) {
			skip[c.Problem.Slug] = true
		}
	}
	for _, c := range DueCards(cards, date, loc, skip) {
		if len(plan) == slots {
			break
		}
		plan = append(plan, c.Problem.Slug)
	}
	return plan
}

// firstAttemptOn reports whether the card's first counted attempt was on date.
func (c Card) firstAttemptOn(date time.Time) bool {
	return c.Reviews == 1 && Date(c.Last.CreatedAt, c.loc).Equal(date)
}

// ReviewReason explains a card in plain text: its flag, or its last counted
// attempt and recall, e.g. "Struggled 9 days ago · recall estimate 62%".
func ReviewReason(c Card, now time.Time, loc *time.Location) string {
	if c.SpotCheck() {
		return SpotCheckReason(c, loc)
	}
	// A struggle reads with its recall, as unflagged cards do.
	if c.Flag != nil && (c.Flag.Struggle == "" || c.Flag.TimeWrong || c.Flag.SpaceWrong || c.Flag.NotOptimal) {
		return c.Flag.Reason(now, loc)
	}
	label := OutcomeLabel(c.Last.Outcome)
	if c.Last.Outcome == "solved" && c.Last.Assisted {
		label = "Solved with help"
	}
	if c.Last.Quick() {
		label += " in quick review"
	}
	return fmt.Sprintf("%s %s · recall estimate %d%%", label, daysAgo(Date(c.Last.CreatedAt, loc), Date(now, loc)), int(c.Retrievability(now)*100+0.5))
}

// SpotCheckReason is a spot check's review reason, e.g. "Spot check of quick
// review from Mon 2 Nov 2026".
func SpotCheckReason(c Card, loc *time.Location) string {
	return "Spot check of quick review from " + DateLabel(Date(c.Last.CreatedAt, loc))
}
