package leetgrinder

import "time"

// ReviewItem is one planned review for a date.
type ReviewItem struct {
	Slot    int
	Problem Problem
	Card    Card
	Reason  string
	Done    bool
}

// Today is the day-level view shared by pages, the API and notifications.
type Today struct {
	Settings Settings
	State    State
	Now      time.Time
	Date     time.Time
	// Goal is today's frozen goal.
	Goal DailyGoal
	// Reviews are today's frozen review picks.
	Reviews  []ReviewItem
	Progress DayProgress
	Streaks  Streaks
	// Cards are every problem's review card as of now.
	Cards []Card
	// cardBySlug indexes Cards; nil when Today was not built by NewTodayFrom.
	cardBySlug map[string]Card
	history    map[time.Time]*DayProgress
}

// NewToday combines saved state, including today's frozen goal and review
// plan, into today's view for now.
func NewToday(settings Settings, state State, now time.Time) Today {
	return NewTodayFrom(settings, state, now, ReplayAttempts(state, settings.Location()))
}

// NewTodayFrom is NewToday over r, the replay of state in the settings' time
// zone, so a caller that already replayed state to plan reviews does not
// replay it again. A replay in another zone, or none, is replaced by a
// fresh one. r must come from this state, not an earlier load.
func NewTodayFrom(settings Settings, state State, now time.Time, r Replay) Today {
	loc := settings.Location()
	if r.zone != settings.zone() {
		r = ReplayAttempts(state, loc)
	}
	t := Today{Settings: settings, State: state, Now: now, Date: Date(now, loc)}
	t.Goal = state.GoalFor(t.Date)
	t.Cards = r.cards
	t.cardBySlug = indexCards(t.Cards)
	t.history = history(r.replays, state, loc, t.Date)
	t.Progress = *t.history[t.Date]
	t.Streaks = ComputeStreaks(t.history, t.Progress)
	for i, slug := range state.Plans[t.Date] {
		item := ReviewItem{Slot: i + 1, Problem: state.Problem(slug), Done: t.AttemptedOn(slug)}
		// Reasons describe each card as it stood before today, so they stay
		// stable after today's review is logged.
		if replay, ok := r.replays[slug]; ok {
			if before := replay.before(t.Date, loc); before.Reviews > 0 {
				item.Card = withFlag(before, state.Analyses, loc)
				item.Reason = ReviewReason(item.Card, now, loc)
			}
		}
		t.Reviews = append(t.Reviews, item)
	}
	return t
}

// AttemptedOn reports whether slug has an attempt on today's local date.
func (t Today) AttemptedOn(slug string) bool {
	_, ok := t.Progress.Kinds[slug]
	return ok
}

// KindOf is how an attempt's local day counts its problem.
func (t Today) KindOf(a Attempt) string {
	if d, ok := t.history[Date(a.CreatedAt, t.Settings.Location())]; ok {
		return d.Kinds[a.ProblemSlug]
	}
	return ""
}

// Kind is how today counts the attempts on slug, or "" when there are none.
func (t Today) Kind(slug string) string { return t.Progress.Kinds[slug] }

// Card returns slug's review card, if it has one.
func (t Today) Card(slug string) (Card, bool) {
	if t.cardBySlug != nil {
		c, ok := t.cardBySlug[slug]
		return c, ok
	}
	for _, c := range t.Cards {
		if c.Problem.Slug == slug {
			return c, true
		}
	}
	return Card{}, false
}

// indexCards keys cards by problem slug, keeping the first of any repeats as
// the linear search does.
func indexCards(cards []Card) map[string]Card {
	m := make(map[string]Card, len(cards))
	for _, c := range cards {
		if _, ok := m[c.Problem.Slug]; !ok {
			m[c.Problem.Slug] = c
		}
	}
	return m
}

// Picked reports whether slug is one of today's review picks.
func (t Today) Picked(slug string) bool {
	for _, r := range t.Reviews {
		if r.Problem.Slug == slug {
			return true
		}
	}
	return false
}

// MissingReviews lists today's review picks without an attempt today.
func (t Today) MissingReviews() []ReviewItem {
	var m []ReviewItem
	for _, r := range t.Reviews {
		if !r.Done {
			m = append(m, r)
		}
	}
	return m
}

// DueOptional lists cards due by the end of today that are neither in
// today's plan nor attempted today, in review order.
func (t Today) DueOptional() []Card {
	skip := map[string]bool{}
	for _, r := range t.Reviews {
		skip[r.Problem.Slug] = true
	}
	for slug := range t.Progress.Kinds {
		skip[slug] = true
	}
	return DueCards(t.Cards, t.Date, t.Settings.Location(), skip)
}

// Backlog counts due cards, flagged ones included, that are not in today's
// plan and not attempted today.
func (t Today) Backlog() int { return len(t.DueOptional()) }

// Calendar is the last n days of goal results, oldest first.
func (t Today) Calendar(n int) []CalendarDay { return GoalMetDays(t.history, t.Progress, n) }

// Due reports whether a card is due by the end of today.
func (t Today) Due(c Card) bool { return c.Due.Before(EndOfDate(t.Date, t.Settings.Location())) }
