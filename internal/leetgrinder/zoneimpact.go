package leetgrinder

import (
	"maps"
	"slices"
	"time"
)

// ZoneWeeks is how far back a time zone preview counts goal-met days.
const ZoneWeeks = 8

// ZoneStats is how history reads in one time zone.
type ZoneStats struct {
	Zone    string
	Current int
	Longest int
	MetDays int
}

// ZoneImpact compares history in the saved time zone with history in the
// zone being chosen, so the settings page can say what changing it recounts.
type ZoneImpact struct {
	Before, After ZoneStats
	// SavedGoal is the goal now; Goal is the one the save would apply.
	SavedGoal, Goal DailyGoal
}

// NewZoneImpact reads state as the saved settings and as the settings with
// zone and goal applied, without saving or freezing anything. Today is
// planned in memory on a copy of state the way the first visit after a save
// would plan it, so the figures match what the dashboard shows afterwards.
func NewZoneImpact(settings Settings, state State, now time.Time, zone string, goal DailyGoal) ZoneImpact {
	after := settings
	after.Timezone, after.Goal = zone, goal
	return ZoneImpact{Before: zoneStats(settings, state, now), After: zoneStats(after, state, now), SavedGoal: settings.Goal, Goal: goal}
}

func zoneStats(s Settings, state State, now time.Time) ZoneStats {
	loc := s.Location()
	date := Date(now, loc)
	state.Goals, state.Plans = maps.Clone(state.Goals), maps.Clone(state.Plans)
	if state.Goals == nil {
		state.Goals = map[time.Time]DailyGoal{}
	}
	if state.Plans == nil {
		state.Plans = map[time.Time][]string{}
	}
	if _, ok := state.Goals[date]; !ok {
		state.Goals[date] = s.Goal
	}
	replay := ReplayAttempts(state, loc)
	existing := state.Plans[date]
	if plan := PlanReviews(replay.Cards(), date, loc, state.Goals[date].Review, slices.Clone(existing)); len(plan) > len(existing) {
		state.Plans[date] = plan
	}
	today := NewTodayFrom(s, state, now, replay)
	out := ZoneStats{Zone: s.zone(), Current: today.Streaks.Current, Longest: today.Streaks.Longest}
	for _, d := range today.Calendar(ZoneWeeks * 7) {
		if d.Met {
			out.MetDays++
		}
	}
	return out
}

// Changed reports whether any figure differs between the zones.
func (z ZoneImpact) Changed() bool {
	b, a := z.Before, z.After
	return b.Current != a.Current || b.Longest != a.Longest || b.MetDays != a.MetDays
}
