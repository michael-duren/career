package analysis

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func testStore(t *testing.T) *database.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	base, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { base.Close() })
	schema := "leetgrinder_analysis_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = base.DB.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { base.DB.Exec("DROP SCHEMA " + schema + " CASCADE") })
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := database.Open(dsn + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = db.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db
}

// clock is a settable test clock.
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newClock() *clock                   { return &clock{time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)} }
func (c *clock) worker(db *database.Store, api *fakeAPI) *Worker {
	return &Worker{Store: db, Key: testKey, Model: "claude-sonnet-5", DailyLimit: 50, BaseURL: api.URL, Now: c.now}
}

func saveAttempt(t *testing.T, db *database.Store, a leetgrinder.Attempt) leetgrinder.Attempt {
	t.Helper()
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.Minutes == 0 {
		a.Minutes = 20
	}
	if a.Outcome == "" {
		a.Outcome = "solved"
	}
	saved, err := db.SaveLeetgrinderAttempt(context.Background(), a, "")
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

func analysisOf(t *testing.T, db *database.Store, id string) (leetgrinder.Analysis, bool) {
	t.Helper()
	state, err := db.LeetgrinderState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return state.AnalysisFor(id)
}

func step(t *testing.T, w *Worker) {
	t.Helper()
	if err := w.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerQueueAndInvalidation(t *testing.T) {
	db := testStore(t)
	ctx := context.Background()
	api := newFakeAPI(t)
	c := newClock()
	w := c.worker(db, api)

	withCode := saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "def f(): pass", CodeLanguage: "python3", Source: "extension"})
	saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)"})                                    // no code
	saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", Outcome: "unfinished", Code: "x = 1", CodeLanguage: "python3", Source: "extension"}) // no complexity
	step(t, w)
	reqs := api.take()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want only the attempt with code and complexity", len(reqs))
	}
	got, ok := analysisOf(t, db, withCode.ID)
	if !ok || !got.Done() || got.ActualTime != "O(n)" || got.ActualSpace != "O(n)" || got.TimeMatches == nil || !*got.TimeMatches || got.SpaceMatches == nil || *got.SpaceMatches || got.Optimal == nil || !*got.Optimal || got.Model != "claude-sonnet-5" || got.Tries != 1 || !got.StatedWrong() {
		t.Fatalf("analysis %+v", got)
	}
	// Done analyses are not requested again.
	step(t, w)
	if n := len(api.take()); n != 0 {
		t.Fatalf("re-requested a done analysis: %d", n)
	}

	// Correcting a stated complexity changes the inputs and queues it again.
	correction := withCode
	correction.SpaceComplexity = "O(n)"
	corrected, err := db.SaveLeetgrinderAttempt(ctx, correction, withCode.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ = analysisOf(t, db, withCode.ID); got.Current || got.Done() || got.StatedWrong() {
		t.Fatalf("stale analysis still current: %+v", got)
	}
	api.answer(`{"actualTime":"O(n)","actualSpace":"O(n)","timeMatches":true,"spaceMatches":true,"optimal":true,"explanation":"Right."}`, "end_turn")
	step(t, w)
	if reqs = api.take(); len(reqs) != 1 {
		t.Fatalf("correction: %d requests", len(reqs))
	}
	if got, _ = analysisOf(t, db, corrected.ID); !got.Done() || got.StatedWrong() || got.Tries != 1 {
		t.Fatalf("after correction %+v", got)
	}
	// Corrected code (not possible from the app, which keeps captured code) is queued too.
	if _, err = db.DB.Exec("UPDATE leetgrinder_attempts SET code=code||'\n# changed' WHERE id=$1", withCode.ID); err != nil {
		t.Fatal(err)
	}
	step(t, w)
	if n := len(api.take()); n != 1 {
		t.Fatalf("changed code: %d requests", n)
	}

	// Re-analyse resets a done analysis to pending with no tries.
	if err = db.RequeueLeetgrinderAnalysis(ctx, "two-sum", withCode.ID, c.now()); err != nil {
		t.Fatal(err)
	}
	if got, _ = analysisOf(t, db, withCode.ID); got.Status != leetgrinder.AnalysisPending || got.Tries != 0 || got.ActualTime != "" || got.Optimal != nil {
		t.Fatalf("requeued %+v", got)
	}
	step(t, w)
	if got, _ = analysisOf(t, db, withCode.ID); !got.Done() || len(api.take()) != 1 {
		t.Fatalf("re-analysed %+v", got)
	}
	if err = db.RequeueLeetgrinderAnalysis(ctx, "valid-anagram", withCode.ID, c.now()); err != database.ErrNotFound {
		t.Fatalf("requeue under another slug: %v", err)
	}

	// Deleting the attempt deletes its analysis.
	if _, err = db.DB.Exec("DELETE FROM leetgrinder_attempts WHERE id=$1", withCode.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = db.DB.QueryRow("SELECT count(*) FROM leetgrinder_analyses").Scan(&n); err != nil || n != 0 {
		t.Fatalf("analyses left: %d %v", n, err)
	}
}

func TestWorkerNewestFirst(t *testing.T) {
	db := testStore(t)
	api := newFakeAPI(t)
	c := newClock()
	w := c.worker(db, api)
	w.DailyLimit = 1
	old := saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "old", CodeLanguage: "python3", Source: "extension"})
	if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=created_at-interval '1 day' WHERE id=$1", old.ID); err != nil {
		t.Fatal(err)
	}
	newer := saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "new", CodeLanguage: "python3", Source: "extension"})
	step(t, w)
	if _, ok := analysisOf(t, db, newer.ID); !ok {
		t.Fatal("newest attempt was not analysed first")
	}
	if _, ok := analysisOf(t, db, old.ID); ok {
		t.Fatal("daily limit of 1 allowed a second request")
	}
}

func TestWorkerRetryCapAndRedaction(t *testing.T) {
	db := testStore(t)
	api := newFakeAPI(t)
	c := newClock()
	w := c.worker(db, api)
	a := saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "def f(): pass", CodeLanguage: "python3", Source: "extension"})
	// The fake echoes the key and the request in its error body.
	api.reply(500, `{"type":"error","error":{"type":"api_error","message":"boom `+testKey+` def f(): pass"}}`)
	for try := 1; try <= leetgrinder.AnalysisMaxTries; try++ {
		step(t, w)
		if n := len(api.take()); n != 1 {
			t.Fatalf("try %d: %d requests", try, n)
		}
		got, _ := analysisOf(t, db, a.ID)
		want := leetgrinder.AnalysisPending
		if try == leetgrinder.AnalysisMaxTries {
			want = leetgrinder.AnalysisFailed
		}
		if got.Status != want || got.Tries != try || !strings.Contains(got.Error, "500 api_error") || strings.Contains(got.Error, testKey) || strings.Contains(got.Error, "def f") {
			t.Fatalf("try %d: %+v", try, got)
		}
		// Nothing is sent again before the backoff passes.
		step(t, w)
		if n := len(api.take()); n != 0 {
			t.Fatalf("try %d: retried before the backoff", try)
		}
		c.advance(database.LeetgrinderAnalysisBackoff(try))
	}
	// A failed analysis waits for a re-analyse request.
	c.advance(time.Hour)
	step(t, w)
	if n := len(api.take()); n != 0 {
		t.Fatalf("failed analysis retried: %d", n)
	}
	var stored string
	if err := db.DB.QueryRow("SELECT error FROM leetgrinder_analyses WHERE attempt_id=$1", a.ID).Scan(&stored); err != nil || strings.Contains(stored, testKey) {
		t.Fatalf("stored error %q %v", stored, err)
	}

	// A refusal fails at once.
	b := saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "other", CodeLanguage: "python3", Source: "extension"})
	api.answer(`{}`, "refusal")
	step(t, w)
	if got, _ := analysisOf(t, db, b.ID); got.Status != leetgrinder.AnalysisFailed || got.Tries != 1 || !strings.Contains(got.Error, "declined") {
		t.Fatalf("refusal %+v", got)
	}
	// A 400 counts a try but pauses every worker, since it usually affects every attempt.
	d := saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "third", CodeLanguage: "python3", Source: "extension"})
	api.reply(400, `{"type":"error","error":{"type":"invalid_request_error","message":"bad"}}`)
	step(t, w)
	if got, _ := analysisOf(t, db, d.ID); got.Status != leetgrinder.AnalysisPending || got.Tries != 1 || !strings.Contains(got.Error, "400 invalid_request_error") {
		t.Fatalf("bad request %+v", got)
	}
	pause, err := db.LeetgrinderAnalysisPause(context.Background())
	if err != nil || !pause.Until.Equal(c.now().Add(ConfigPause)) || !strings.Contains(pause.Reason, "400") {
		t.Fatalf("pause %+v %v", pause, err)
	}
	// A second worker (another replica or a restart) honours the stored pause.
	other := c.worker(db, api)
	c.advance(5 * time.Minute)
	step(t, other)
	if n := len(api.take()); n != 2 {
		t.Fatalf("requests: %d, want the refusal and the 400 only", n)
	}
}

func TestWorkerOutageAndConfigErrors(t *testing.T) {
	db := testStore(t)
	api := newFakeAPI(t)
	c := newClock()
	w := c.worker(db, api)
	var ids []string
	for i := range 3 {
		ids = append(ids, saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "code " + string(rune('a'+i)), CodeLanguage: "python3", Source: "extension"}).ID)
	}
	// An outage costs one request per tick, not one per queued attempt.
	api.reply(529, `{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`)
	step(t, w)
	if n := len(api.take()); n != 1 {
		t.Fatalf("outage: %d requests in one tick", n)
	}

	// A bad key leaves attempts pending without using a try, and pauses the worker.
	c.advance(time.Hour)
	api.reply(401, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	step(t, w)
	if n := len(api.take()); n != 1 {
		t.Fatalf("bad key: %d requests", n)
	}
	var failed int
	for _, id := range ids {
		got, ok := analysisOf(t, db, id)
		if ok && got.Status == leetgrinder.AnalysisFailed {
			failed++
		}
		if ok && strings.Contains(got.Error, "check ANTHROPIC_API_KEY") && (got.Status != leetgrinder.AnalysisPending || got.Tries != 1) { // the one try from the outage
			t.Fatalf("config error counted a try: %+v", got)
		}
	}
	if failed != 0 {
		t.Fatalf("%d attempts failed on a configuration error", failed)
	}
	c.advance(ConfigPause - time.Minute)
	step(t, w)
	if n := len(api.take()); n != 0 {
		t.Fatalf("sent during the configuration pause: %d", n)
	}
	// Once the key works, everything is analysed with no re-analyse needed.
	api.answer(`{"actualTime":"O(n)","actualSpace":"O(1)","timeMatches":true,"spaceMatches":true,"optimal":true,"explanation":"Fine."}`, "end_turn")
	c.advance(2 * time.Minute)
	step(t, w)
	if n := len(api.take()); n != 3 {
		t.Fatalf("after the pause: %d requests", n)
	}
	for _, id := range ids {
		if got, _ := analysisOf(t, db, id); !got.Done() {
			t.Fatalf("not done: %+v", got)
		}
	}
}

func TestWorkerDailyLimitToggleAndKey(t *testing.T) {
	db := testStore(t)
	ctx := context.Background()
	api := newFakeAPI(t)
	c := newClock()
	w := c.worker(db, api)
	w.DailyLimit = 2
	for i := range 4 {
		saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "code " + string(rune('a'+i)), CodeLanguage: "python3", Source: "extension"})
	}

	// Without a key, nothing touches the network.
	noKey := c.worker(db, api)
	noKey.Key = ""
	step(t, noKey)
	noKey.Run(ctx) // returns at once
	if n := len(api.take()); n != 0 {
		t.Fatalf("requests without a key: %d", n)
	}

	// Turned off in settings: nothing is sent.
	settings, _ := db.LeetgrinderSettings(ctx)
	if !settings.AnalysisEnabled {
		t.Fatal("analysis should default to on")
	}
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(s *leetgrinder.Settings) error { s.AnalysisEnabled = false; return nil }); err != nil {
		t.Fatal(err)
	}
	step(t, w)
	if n := len(api.take()); n != 0 {
		t.Fatalf("requests while off: %d", n)
	}
	settings, _ = db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(s *leetgrinder.Settings) error { s.AnalysisEnabled = true; return nil }); err != nil {
		t.Fatal(err)
	}

	// Turning analysis off stops the next request, even within a tick.
	api.onRequest(func() {
		s, _ := db.LeetgrinderSettings(ctx)
		if _, err := db.UpdateLeetgrinderSettings(ctx, s.Revision, func(s *leetgrinder.Settings) error { s.AnalysisEnabled = false; return nil }); err != nil {
			t.Error(err)
		}
	})
	step(t, w)
	if n := len(api.take()); n != 1 {
		t.Fatalf("requests after turning off mid-tick: %d", n)
	}
	api.onRequest(nil)
	settings, _ = db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(s *leetgrinder.Settings) error { s.AnalysisEnabled = true; return nil }); err != nil {
		t.Fatal(err)
	}
	w.DailyLimit = 3

	// The daily limit stops the queue until the next local day.
	step(t, w)
	if n := len(api.take()); n != 2 {
		t.Fatalf("requests up to a limit of 3: %d", n)
	}
	day := leetgrinder.Date(c.now(), settings.Location())
	if used, err := db.LeetgrinderAnalysisUsage(ctx, day); err != nil || used != 3 {
		t.Fatalf("usage %d %v", used, err)
	}
	c.advance(time.Hour)
	step(t, w)
	if n := len(api.take()); n != 0 {
		t.Fatalf("requests past the limit: %d", n)
	}
	q, err := db.LeetgrinderAnalysisQueueCounts(ctx)
	if err != nil || q.Queued != 1 || q.Done != 3 || q.Failed != 0 {
		t.Fatalf("queue %+v %v", q, err)
	}
	c.advance(24 * time.Hour)
	step(t, w)
	if n := len(api.take()); n != 1 {
		t.Fatalf("next day: %d requests", n)
	}

	// Another worker holding the lock blocks this one.
	saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(1)", Code: "locked", CodeLanguage: "python3", Source: "extension"})
	ran, err := db.WithLeetgrinderAnalysisLock(ctx, func(context.Context) error {
		step(t, w)
		return nil
	})
	if err != nil || !ran {
		t.Fatalf("lock: %v %v", ran, err)
	}
	if n := len(api.take()); n != 0 {
		t.Fatalf("requests while another worker held the lock: %d", n)
	}
}

func TestWorkerStoresEstimatedOptimumOnce(t *testing.T) {
	db := testStore(t)
	ctx := context.Background()
	api := newFakeAPI(t)
	c := newClock()
	w := c.worker(db, api)
	api.answer(`{"actualTime":"O(n)","actualSpace":"O(n)","timeMatches":true,"spaceMatches":true,"optimal":false,"explanation":"x","optimalTime":"O(n)","optimalSpace":"O(1)","optimalNote":"Two pointers."}`, "end_turn")
	saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "not-seeded-problem", TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: "a", CodeLanguage: "python3", Source: "extension"})
	step(t, w)
	p, err := db.LeetgrinderProblem(ctx, "not-seeded-problem")
	if err != nil || p.OptimalTime != "O(n)" || p.OptimalSpace != "O(1)" || p.OptimalNote != "Two pointers." || p.OptimalSource != "model" {
		t.Fatalf("estimate not stored: %+v %v", p, err)
	}
	// A known optimum is sent as the reference, and later estimates never replace it.
	api.answer(`{"actualTime":"O(n)","actualSpace":"O(n)","timeMatches":true,"spaceMatches":true,"optimal":false,"explanation":"x"}`, "end_turn")
	saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "not-seeded-problem", TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: "b", CodeLanguage: "python3", Source: "extension"})
	step(t, w)
	reqs := api.take()
	user, _ := json.Marshal(reqs[len(reqs)-1].Body["messages"])
	if !strings.Contains(string(user), "Reference optimal space: O(1)") {
		t.Fatalf("second request lacks the stored estimate: %s", user)
	}
	// Curated optima are never replaced.
	api.answer(`{"actualTime":"O(n)","actualSpace":"O(n)","timeMatches":true,"spaceMatches":true,"optimal":true,"explanation":"x"}`, "end_turn")
	saveAttempt(t, db, leetgrinder.Attempt{ProblemSlug: "two-sum", TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: "c", CodeLanguage: "python3", Source: "extension"})
	step(t, w)
	if p, _ = db.LeetgrinderProblem(ctx, "two-sum"); p.OptimalSource != "curated" || p.OptimalTime != "O(n)" {
		t.Fatalf("curated optimum changed: %+v", p)
	}
}
