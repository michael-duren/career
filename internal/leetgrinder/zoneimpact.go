package leetgrinder

import "time"

// ZoneWeeks is how far back a time zone preview counts goal-met days.
const ZoneWeeks = 8

// ZoneStats is how history reads in one time zone.
type ZoneStats struct {
	Zone     string
	Current  int
	Longest  int
	MetDays  int
}

// ZoneImpact compares history in the saved time zone with history in the
// zone being chosen, so the settings page can say what changing it recounts.
type ZoneImpact struct {
	Before, After ZoneStats
	// Days is the window MetDays counts over.
	Days int
}

// NewZoneImpact replays state in the saved zone and in zone without saving
// or freezing anything: it only reads the frozen goals and plans in state.
func NewZoneImpact(settings Settings, state State, now time.Time, zone string) ZoneImpact {
	days := ZoneWeeks * 7
	stats := func(s Settings) ZoneStats {
		today := NewToday(s, state, now)
		out := ZoneStats{Zone: s.zone(), Current: today.Streaks.Current, Longest: today.Streaks.Longest}
		for _, d := range today.Calendar(days) {
			if d.Met {
				out.MetDays++
			}
		}
		return out
	}
	after := settings
	after.Timezone = zone
	return ZoneImpact{Before: stats(settings), After: stats(after), Days: days}
}

// Changed reports whether any figure differs between the zones.
func (z ZoneImpact) Changed() bool {
	b, a := z.Before, z.After
	return b.Current != a.Current || b.Longest != a.Longest || b.MetDays != a.MetDays
}
