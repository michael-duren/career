package leetgrinder

import (
	"strings"
	"testing"
	"time"
)

// quickState is two-sum struggled 20 days before now, then whatever extra
// attempts follow.
func quickState(now time.Time, extra ...Attempt) State {
	attempts := []Attempt{{ID: "full", ProblemSlug: "two-sum", Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)", CreatedAt: now.AddDate(0, 0, -20), Mode: ModeFull}}
	attempts = append(attempts, extra...)
	// State keeps attempts newest first.
	for i, j := 0, len(attempts)-1; i < j; i, j = i+1, j-1 {
		attempts[i], attempts[j] = attempts[j], attempts[i]
	}
	return State{Attempts: attempts, Problems: testProblems, Plans: map[time.Time][]string{}, Goals: map[time.Time]DailyGoal{}}
}

func quickAttempt(id, outcome string, at time.Time) Attempt {
	return Attempt{ID: id, ProblemSlug: "two-sum", Mode: ModeQuick, Outcome: outcome, Minutes: 3, TimeComplexity: "O(n)", SpaceComplexity: "O(n)", CreatedAt: at}
}

func quickToday(t *testing.T, state State, now time.Time) (Today, Card) {
	t.Helper()
	today := NewToday(Settings{Timezone: "UTC", Goal: DefaultGoal, SpotCheckRate: DefaultSpotCheckRate}, state, now)
	c, ok := today.Card("two-sum")
	if !ok {
		t.Fatal("no card")
	}
	return today, c
}

func TestQuickRecalledClearsAndSchedulesLikeASolve(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	before, c := quickToday(t, quickState(now), now)
	if !before.Due(c) || !c.Flagged() || !before.QuickEligible(c) || len(before.FlaggedCards()) != 1 {
		t.Fatalf("struggled problem not due and flagged: %+v", c)
	}
	today, c := quickToday(t, quickState(now, quickAttempt("q", QuickRecalled, now.Add(-time.Minute))), now)
	if today.Due(c) || c.Flagged() || len(today.FlaggedCards()) != 0 || len(DueCards(today.Cards, today.Date, time.UTC, nil)) != 0 {
		t.Fatalf("recalled problem still due or flagged: %+v", c)
	}
	// A full solve rated Good (over the 25-minute slot) at the same time
	// schedules the same date.
	solve := Attempt{ID: "s", ProblemSlug: "two-sum", Mode: ModeFull, Outcome: "solved", Minutes: ReviewSlotMinutes + 5, TimeComplexity: "O(n)", SpaceComplexity: "O(n)", CreatedAt: now.Add(-time.Minute)}
	_, solved := quickToday(t, quickState(now, solve), now)
	if !c.Due.Equal(solved.Due) || !c.FSRSDue.Equal(solved.FSRSDue) {
		t.Fatalf("recalled due %v, solved due %v", c.Due, solved.Due)
	}
	if ReviewRating(quickAttempt("x", QuickRecalled, now)) != ReviewRating(solve) {
		t.Fatal("recalled must rate like a normal solve")
	}
	// It still counts as a past struggle on the spaced schedule.
	if !c.Reviewable() || c.SuggestedMode() != ModeFull {
		t.Fatalf("reviewable %v, suggested %q", c.Reviewable(), c.SuggestedMode())
	}
}

func TestQuickPartialStaysFlaggedWithReason(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	partial := quickAttempt("q", QuickPartial, now.Add(-time.Minute))
	partial.Missed = MissedSpace
	today, c := quickToday(t, quickState(now, partial), now)
	if !c.Flagged() || !c.Reviewable() || len(today.FlaggedCards()) != 1 {
		t.Fatalf("partial not flagged: %+v", c)
	}
	if got := c.Flag.Reason(now, time.UTC); got != "Space complexity missed in quick review today" {
		t.Fatalf("reason %q", got)
	}
	if got := ReviewReason(c, now, time.UTC); got != "Space complexity missed in quick review today" {
		t.Fatalf("review reason %q", got)
	}
	if want := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC); !c.Due.Equal(want) {
		t.Fatalf("due %v, want %v", c.Due, want)
	}
	// It may be quick-reviewed again; only a forgotten one blocks.
	if c.RequiredMode() != "" || !today.QuickEligible(c) {
		t.Fatal("partial must allow another quick review")
	}
}

func TestQuickForgotRequiresFullAttempt(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	today, c := quickToday(t, quickState(now, quickAttempt("q", QuickForgot, now.Add(-time.Minute))), now)
	if !c.RequiresFull() || c.RequiredMode() != ModeFull || today.QuickEligible(c) || QuickBlocked(today, c, true) == "" {
		t.Fatalf("forgot must require a full attempt: %+v", c)
	}
	if want := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC); !c.Due.Equal(want) {
		t.Fatalf("due %v, want %v", c.Due, want)
	}
	if !strings.Contains(c.Flag.Reason(now, time.UTC), "full re-solve required") {
		t.Fatal(c.Flag.Reason(now, time.UTC))
	}
	// Tomorrow it is due, but still only for a full attempt.
	tomorrow := now.AddDate(0, 0, 1)
	today, c = quickToday(t, quickState(tomorrow, quickAttempt("q", QuickForgot, now.Add(-time.Minute))), tomorrow)
	if !today.Due(c) || today.QuickEligible(c) || len(today.QuickCards(nil)) != 0 {
		t.Fatal("forgot problem must be due but not quick-eligible")
	}
	// A full attempt lifts the block.
	full := Attempt{ID: "f", ProblemSlug: "two-sum", Mode: ModeFull, Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)", CreatedAt: tomorrow.Add(-time.Hour)}
	_, c = quickToday(t, quickState(tomorrow, quickAttempt("q", QuickForgot, now.Add(-time.Minute)), full), tomorrow)
	if c.RequiresFull() || c.RequiredMode() != "" {
		t.Fatal("full attempt must clear the block")
	}
}

func TestSpotCheckHiddenUntilDue(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	// A problem flagged only for a complexity mistake: never struggled.
	solve := Attempt{ID: "full", ProblemSlug: "binary-search", Mode: ModeFull, Outcome: "solved", Minutes: 10, TimeComplexity: "O(n)", SpaceComplexity: "O(1)", CreatedAt: now.AddDate(0, 0, -3)}
	spot := Attempt{ID: "q", ProblemSlug: "binary-search", Mode: ModeQuick, Outcome: QuickRecalled, SpotCheck: true, Minutes: 3, TimeComplexity: "O(log n)", SpaceComplexity: "O(1)", CreatedAt: now.Add(-time.Minute)}
	state := State{Attempts: []Attempt{spot, solve}, Problems: testProblems}
	settings := Settings{Timezone: "UTC", Goal: DefaultGoal}
	today := NewToday(settings, state, now)
	c, _ := today.Card("binary-search")
	if !c.SpotCheck() || c.Flagged() || today.Due(c) {
		t.Fatalf("spot check must leave the due and flagged lists: %+v", c)
	}
	if len(today.ReviewCards()) != 0 || len(UpcomingReviews(today, 30)) != 0 || c.Reviewable() {
		t.Fatal("pending spot check must stay hidden")
	}
	// Once due it comes back as a full re-solve, labelled.
	later := c.Due.Add(time.Hour)
	today = NewToday(settings, state, later)
	c, _ = today.Card("binary-search")
	if !today.Due(c) || c.RequiredMode() != ModeFull || today.QuickEligible(c) || len(today.ReviewCards()) != 1 {
		t.Fatalf("due spot check: due %v required %q", today.Due(c), c.RequiredMode())
	}
	if got := ReviewReason(c, later, time.UTC); got != "Spot check of quick review from Tue 24 Nov 2026" {
		t.Fatalf("reason %q", got)
	}
	if !strings.Contains(QuickBlocked(today, c, true), "spot check") {
		t.Fatal("due spot check must explain the block")
	}
}

func TestSpotChecksRaiseRateUntilThreePass(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// Each spot check is a recalled quick review of its own problem,
	// resolved by the next full attempt.
	build := func(passes ...bool) State {
		var attempts []Attempt
		for i, pass := range passes {
			slug := "p" + string(rune('a'+i))
			at := base.AddDate(0, 0, 2*i)
			attempts = append(attempts, Attempt{ID: slug + "1", ProblemSlug: slug, Mode: ModeQuick, Outcome: QuickRecalled, SpotCheck: true, CreatedAt: at})
			outcome := "solved"
			if !pass {
				outcome = "struggled"
			}
			attempts = append(attempts, Attempt{ID: slug + "2", ProblemSlug: slug, Mode: ModeFull, Outcome: outcome, CreatedAt: at.AddDate(0, 0, 1)})
		}
		return State{Attempts: attempts}
	}
	for _, test := range []struct {
		name           string
		passes         []bool
		elevated       bool
		passed, failed int
	}{
		{"none", nil, false, 0, 0},
		{"one failure", []bool{true, false, true}, false, 2, 1},
		{"two of five fail", []bool{false, true, true, false}, true, 2, 2},
		{"two passes do not restore", []bool{false, false, true, true}, true, 2, 2},
		{"three in a row restore", []bool{false, false, true, true, true}, false, 3, 2},
		{"failure resets the run", []bool{false, false, true, true, false, true, true}, true, 4, 3},
		{"old failures fall out", []bool{false, true, true, true, true, false}, false, 4, 2},
	} {
		s := SpotChecks(build(test.passes...), 10)
		if s.Elevated != test.elevated || s.Passed != test.passed || s.Failed != test.failed {
			t.Errorf("%s: %+v", test.name, s)
		}
		want := 10
		if test.elevated {
			want = ElevatedSpotCheckRate
		}
		if s.Rate != want {
			t.Errorf("%s: rate %d", test.name, s.Rate)
		}
	}
	// 0% turns spot checks off, even after failures.
	if s := SpotChecks(build(false, true, true, false), 0); s.Elevated || s.Rate != 0 || s.Failed != 2 {
		t.Fatalf("rate 0 after failures: %+v", s)
	}
	// Help on the re-solve fails a spot check; an unresolved one counts
	// for nothing.
	state := State{Attempts: []Attempt{
		{ID: "a1", ProblemSlug: "a", Mode: ModeQuick, Outcome: QuickRecalled, SpotCheck: true, CreatedAt: base},
		{ID: "a2", ProblemSlug: "a", Mode: ModeFull, Outcome: "solved", Assisted: true, CreatedAt: base.AddDate(0, 0, 1)},
		{ID: "b1", ProblemSlug: "b", Mode: ModeQuick, Outcome: QuickRecalled, SpotCheck: true, CreatedAt: base},
	}}
	if s := SpotChecks(state, 0); s.Failed != 1 || s.Passed != 0 || s.Rate != 0 {
		t.Fatalf("assisted or pending: %+v", s)
	}
}

func TestPickSpotCheckRate(t *testing.T) {
	picked := 0
	for i := range 1000 {
		if PickSpotCheck(DefaultSpotCheckRate, float64(i)/1000) {
			picked++
		}
	}
	if picked != 100 {
		t.Fatalf("10%% of draws picked %d of 1000", picked)
	}
	if PickSpotCheck(0, 0) || !PickSpotCheck(25, 0.2499) || PickSpotCheck(25, 0.25) {
		t.Fatal("rate bounds")
	}
}

func TestQuickReviewsStayOutOfGoalAndSolveRates(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	solved := Attempt{ID: "s", ProblemSlug: "binary-search", Mode: ModeFull, Outcome: "solved", Minutes: 10, TimeComplexity: "O(log n)", SpaceComplexity: "O(1)", CreatedAt: now.AddDate(0, 0, -20)}
	without := quickState(now, solved)
	partial := quickAttempt("q2", QuickPartial, now.Add(-time.Hour))
	partial.Missed, partial.SpaceComplexity = MissedSpace, "O(1)"
	with := quickState(now, solved, quickAttempt("q1", QuickRecalled, now.AddDate(0, 0, -1)), partial)
	settings := Settings{Timezone: "UTC", Goal: DefaultGoal}
	a, b := NewToday(settings, without, now), NewToday(settings, with, now)
	if len(b.Progress.Kinds) != 0 || b.Progress.Reviews() != 0 || b.QuickReviews != 1 {
		t.Fatalf("quick review counted toward the goal: %+v, quick %d", b.Progress.Kinds, b.QuickReviews)
	}
	if a.KindOf(with.Attempts[0]) != "" || b.KindOf(with.Attempts[0]) != "" {
		t.Fatal("quick reviews have no kind")
	}
	if Summarize(without) != Summarize(with) {
		t.Fatal("quick reviews changed solve counts")
	}
	ta, tb := WeeklyTrends(a, 12), WeeklyTrends(b, 12)
	last := func(tr Trends) TrendWeek { return tr.Weeks[len(tr.Weeks)-1] }
	ra, oka := last(ta).IndependentRate()
	rb, okb := last(tb).IndependentRate()
	if ra != rb || oka != okb {
		t.Fatal("independent-solve rate changed")
	}
	for _, d := range TrendDifficulties {
		ma, _ := last(ta).AverageMinutes(d)
		mb, _ := last(tb).AverageMinutes(d)
		if ma != mb {
			t.Fatalf("%s minutes changed", d)
		}
	}
	// Two-sum's optimum is O(n)/O(n): the partial's O(n) time matches and
	// O(1) space does not, so this week has two quick-review checks.
	if week := last(tb); week.QuickReviews != 2 || week.Checked-last(ta).Checked != 4 || week.Matched-last(ta).Matched != 3 {
		t.Fatalf("quick checks: %+v", week)
	}
	stats := NewStatsPage(b, StatsFilter{Sort: "recall"}, 12)
	if stats.Quick != (QuickStats{Count: 2, Recalled: 1, Partial: 1}) {
		t.Fatalf("quick stats %+v", stats.Quick)
	}
	if r, ok := stats.Quick.RecallRate(); !ok || r != 0.5 {
		t.Fatal("recall rate", r)
	}
	if b.State.ProblemLatest("two-sum") != ResultStruggled {
		t.Fatal("latest result must ignore quick reviews")
	}
}

func TestQuickCheckAndBatch(t *testing.T) {
	p := testProblems["two-sum"]
	for _, test := range []struct{ time, space, missed string }{
		{"O(n)", "O(n)", MissedDetail},
		{"O(n²)", "O(n)", MissedTime},
		{"O(n)", "O(1)", MissedSpace},
		{"O(1)", "O(1)", MissedBoth},
	} {
		if got := CheckQuick(p, test.time, test.space).Missed(); got != test.missed {
			t.Errorf("%s %s: %q", test.time, test.space, got)
		}
	}
	if c := CheckQuick(testProblems["valid-anagram"], "O(n)", "O(1)"); c.Time.Known() || c.Missed() != MissedDetail {
		t.Fatal("no optimum must not judge")
	}
	b := Batch{Since: time.Unix(1764000000, 0), Flagged: 48, Due: 64}
	if got, ok := ParseBatch(b.String()); !ok || got != b {
		t.Fatalf("batch round trip: %v %v", got, ok)
	}
	for _, bad := range []string{"", "1.2", "x.1.2", "0.1.2", "1.-1.2", "1.1.1.1"} {
		if _, ok := ParseBatch(bad); ok {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestNormalizeQuickDetails(t *testing.T) {
	ok := Attempt{Mode: ModeQuick, Outcome: QuickPartial, Missed: MissedTime, TimeComplexity: "O(n)", SpaceComplexity: "O(1)"}
	if err := ok.NormalizeDetails(); err != nil {
		t.Fatal(err)
	}
	full := Attempt{Outcome: "solved", TimeComplexity: "O(n)", SpaceComplexity: "O(1)"}
	if err := full.NormalizeDetails(); err != nil || full.Mode != ModeFull {
		t.Fatal("empty mode must read as full", err)
	}
	for name, a := range map[string]Attempt{
		"full outcome":         {Mode: ModeQuick, Outcome: "solved", TimeComplexity: "O(n)", SpaceComplexity: "O(1)"},
		"quick outcome":        {Mode: ModeFull, Outcome: QuickRecalled, TimeComplexity: "O(n)", SpaceComplexity: "O(1)"},
		"missed on recalled":   {Mode: ModeQuick, Outcome: QuickRecalled, Missed: MissedTime, TimeComplexity: "O(n)", SpaceComplexity: "O(1)"},
		"code":                 {Mode: ModeQuick, Outcome: QuickForgot, Code: "x", CodeLanguage: "python3"},
		"assisted":             {Mode: ModeQuick, Outcome: QuickForgot, Assisted: true},
		"spot check on forgot": {Mode: ModeQuick, Outcome: QuickForgot, SpotCheck: true},
		"no complexity":        {Mode: ModeQuick, Outcome: QuickRecalled},
	} {
		if a.NormalizeDetails() == nil {
			t.Errorf("%s accepted", name)
		}
	}
	forgot := Attempt{Mode: ModeQuick, Outcome: QuickForgot}
	if err := forgot.NormalizeDetails(); err != nil {
		t.Fatal("forgot may leave complexity blank", err)
	}
}

func TestSpotCheckOnScheduledProblemStaysUnnamed(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	// Struggled, then solved cleanly: on the spaced schedule either way.
	solve := Attempt{ID: "s", ProblemSlug: "two-sum", Mode: ModeFull, Outcome: "solved", Minutes: 10, TimeComplexity: "O(n)", SpaceComplexity: "O(n)", CreatedAt: now.AddDate(0, 0, -10)}
	spot := quickAttempt("q", QuickRecalled, now.Add(-time.Minute))
	spot.SpotCheck = true
	state := quickState(now, solve, spot)
	today, c := quickToday(t, state, now)
	if !c.Reviewable() || today.Due(c) {
		t.Fatal("expected a scheduled, not yet due card")
	}
	listed := false
	for _, u := range UpcomingReviews(today, 365) {
		if u.Problem.Slug == "two-sum" {
			listed = true
		}
	}
	if !listed {
		t.Fatal("scheduled card should be listed as upcoming")
	}
	if r := ReviewReason(c, now, time.UTC); strings.Contains(r, "Spot check") || c.SuggestedMode() != ModeQuick {
		t.Fatalf("pending spot check named: reason %q, suggested %q", r, c.SuggestedMode())
	}
	later := c.Due.Add(time.Hour)
	_, c = quickToday(t, state, later)
	if r := ReviewReason(c, later, time.UTC); !strings.HasPrefix(r, "Spot check of quick review from ") || c.RequiredMode() != ModeFull {
		t.Fatalf("due spot check: %q", r)
	}
}

func TestQuickReviewBlockedAfterFullAttemptToday(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	// Flagged by a struggle this morning: not quick-reviewable until tomorrow.
	struggle := Attempt{ID: "s", ProblemSlug: "two-sum", Mode: ModeFull, Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)", CreatedAt: now.Add(-6 * time.Hour)}
	today, c := quickToday(t, quickState(now, struggle), now)
	if !c.Flagged() || today.QuickEligible(c) || !strings.Contains(QuickBlocked(today, c, true), "full attempt on this problem today") || len(today.QuickCards(nil)) != 0 {
		t.Fatal("quick review must not replace today's full attempt")
	}
	tomorrow := now.AddDate(0, 0, 1)
	today, c = quickToday(t, quickState(tomorrow, struggle), tomorrow)
	if !today.QuickEligible(c) {
		t.Fatal("quick review allowed the next day")
	}
}

func TestBacklogCountsProblemsNeedingFullAttempt(t *testing.T) {
	now := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	tomorrow := now.AddDate(0, 0, 1)
	state := quickState(tomorrow, quickAttempt("q", QuickForgot, now))
	today, _ := quickToday(t, state, tomorrow)
	page := NewBacklogPage(today, StartBatch(today))
	if len(page.Remaining) != 0 || page.NeedFull != 1 || page.Due != 1 || page.Flagged != 1 {
		t.Fatalf("backlog: remaining %d, need full %d, due %d, flagged %d", len(page.Remaining), page.NeedFull, page.Due, page.Flagged)
	}
}
