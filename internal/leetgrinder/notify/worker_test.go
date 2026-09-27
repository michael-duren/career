package notify

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

var testKey = bytes.Repeat([]byte{9}, 32)

const testToken = "tk_worker_secret"

type sent struct {
	Path   string
	Header http.Header
	Body   string
}

// ntfyServer records requests and answers with status.
type ntfyServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []sent
	status   int
	// echo returns the Authorization header in the error body, to prove it is redacted.
	echo bool
}

func newNtfyServer(t *testing.T) *ntfyServer {
	n := &ntfyServer{status: 200}
	n.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		n.mu.Lock()
		n.requests = append(n.requests, sent{r.URL.Path, r.Header.Clone(), string(body)})
		status, echo := n.status, n.echo
		n.mu.Unlock()
		w.WriteHeader(status)
		if echo {
			_, _ = w.Write([]byte(`{"error":"denied for ` + r.Header.Get("Authorization") + `"}`))
		}
	}))
	t.Cleanup(n.Close)
	return n
}

func (n *ntfyServer) respond(status int, echo bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.status, n.echo = status, echo
}

func (n *ntfyServer) take() []sent {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := n.requests
	n.requests = nil
	return out
}

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
	schema := "leetgrinder_notify_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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

var chicago, _ = time.LoadLocation("America/Chicago")

// at is a wall-clock time on 2026-09-10 in Chicago, which is Day 10 of a
// schedule starting 2026-09-01.
func at(day int, clock string) time.Time {
	c, _ := time.Parse("15:04", clock)
	return time.Date(2026, 9, day, c.Hour(), c.Minute(), 0, 0, chicago)
}

type fixture struct {
	db     *database.Store
	ntfy   *ntfyServer
	now    time.Time
	worker *Worker
}

func (f *fixture) newWorker() *Worker {
	return &Worker{Store: f.db, SecretKey: testKey, Origin: "https://app.example", HTTP: NewHTTPClient(), Now: func() time.Time { return f.now }}
}

func (f *fixture) step(t *testing.T, now time.Time) []sent {
	t.Helper()
	f.now = now
	if err := f.worker.Step(context.Background()); err != nil {
		t.Fatalf("step at %v: %v", now, err)
	}
	return f.ntfy.take()
}

func (f *fixture) configure(t *testing.T, change func(*leetgrinder.Settings)) {
	t.Helper()
	ctx := context.Background()
	current, err := f.db.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.db.UpdateLeetgrinderSettings(ctx, current.Revision, func(s *leetgrinder.Settings) error { change(s); return nil }); err != nil {
		t.Fatal(err)
	}
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{db: testStore(t), ntfy: newNtfyServer(t)}
	f.worker = f.newWorker()
	box, _ := leetgrinder.NewSecretBox(testKey)
	sealed, _ := box.Seal([]byte(testToken))
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	f.configure(t, func(s *leetgrinder.Settings) {
		s.StartDate, s.NtfyURL, s.NtfyTopic, s.NtfyTokenCiphertext, s.DailyHours = &start, f.ntfy.URL, "grind", sealed, 2.5
	})
	return f
}

func (f *fixture) logEntry(t *testing.T, kind string, date time.Time) leetgrinder.NotificationLogEntry {
	t.Helper()
	e, err := f.db.LeetgrinderNotification(context.Background(), kind, leetgrinder.Date(date, chicago))
	if err != nil {
		t.Fatalf("%s log: %v", kind, err)
	}
	return e
}

func titles(requests []sent) map[string]sent {
	out := map[string]sent{}
	for _, r := range requests {
		out[r.Header.Get("Title")] = r
	}
	return out
}

func TestWorkerSendsEachKindOnceAtItsTime(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	// Six early problems struggled 20 days ago are all due, more than today's slots.
	for _, slug := range []string{"two-sum", "contains-duplicate", "valid-anagram", "ransom-note", "majority-element", "group-anagrams"} {
		if _, err := f.db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1", at(10, "12:00").AddDate(0, 0, -20)); err != nil {
		t.Fatal(err)
	}
	f.configure(t, func(s *leetgrinder.Settings) {
		s.Notifications = map[string]leetgrinder.NotificationPref{
			leetgrinder.NotifyMorningPlan:    {Enabled: true, Time: "08:00", Priority: "low"},
			leetgrinder.NotifyReviewBacklog:  {Enabled: true, Time: "17:00", Threshold: 1, Priority: "default"},
			leetgrinder.NotifyLateEscalation: {Enabled: true, Time: "21:00", Priority: "high"},
			// missing_work and behind_schedule use their defaults: on at 17:00.
		}
	})

	if got := f.step(t, at(10, "07:59")); len(got) != 0 {
		t.Fatalf("sent before 08:00: %v", got)
	}
	got := f.step(t, at(10, "08:00"))
	if len(got) != 1 || got[0].Header.Get("Title") != "Leetgrinder: today's plan" {
		t.Fatalf("morning: %+v", got)
	}
	morning := got[0]
	if morning.Path != "/grind" || morning.Header.Get("Priority") != "low" || morning.Header.Get("Tags") != "sunrise" ||
		morning.Header.Get("Click") != "https://app.example/leetgrinder/day/10" || morning.Header.Get("Authorization") != "Bearer "+testToken {
		t.Fatalf("morning headers: %v %v", morning.Path, morning.Header)
	}
	if !strings.Contains(morning.Body, "Session 10: Lower and upper boundaries") || !strings.Contains(morning.Body, "Review: ") {
		t.Fatalf("morning body: %q", morning.Body)
	}
	if got := f.step(t, at(10, "16:59")); len(got) != 0 {
		t.Fatalf("sent early: %v", got)
	}

	byTitle := titles(f.step(t, at(10, "17:00")))
	missing, ok := byTitle["Leetgrinder: work left today"]
	if !ok || len(byTitle) != 3 {
		t.Fatalf("17:00 sends: %v", byTitle)
	}
	if !strings.Contains(missing.Body, "Session 10: ") || !strings.Contains(missing.Body, "Review: ") || missing.Header.Get("Tags") != "hourglass" {
		t.Fatalf("missing body: %q", missing.Body)
	}
	behind, ok := byTitle["Leetgrinder: 10 sessions behind"]
	if !ok || behind.Header.Get("Click") != "https://app.example/leetgrinder" || !strings.Contains(behind.Body, "0 of 10 expected sessions") {
		t.Fatalf("behind: %v", byTitle)
	}
	var backlog sent
	for title, r := range byTitle {
		if strings.HasSuffix(title, "reviews waiting") {
			backlog = r
		}
	}
	if backlog.Header.Get("Click") != "https://app.example/leetgrinder/reviews" {
		t.Fatalf("backlog: %v", byTitle)
	}

	// A restarted worker reads the log and sends nothing twice.
	f.worker = f.newWorker()
	if got := f.step(t, at(10, "20:59")); len(got) != 0 {
		t.Fatalf("resent after restart: %v", got)
	}
	got = f.step(t, at(10, "21:00"))
	if len(got) != 1 || got[0].Header.Get("Priority") != "high" || got[0].Header.Get("Title") != "Leetgrinder: today is still unfinished" {
		t.Fatalf("late: %+v", got)
	}
	if got := f.step(t, at(10, "23:59")); len(got) != 0 {
		t.Fatalf("resent: %v", got)
	}
	e := f.logEntry(t, leetgrinder.NotifyMissingWork, at(10, "17:00"))
	if e.Status != leetgrinder.NotifySent || e.Attempts != 1 || !strings.Contains(e.Detail, "Session 10") {
		t.Fatalf("log: %+v", e)
	}
	// The next local day starts fresh.
	if got := f.step(t, at(11, "08:00")); len(got) != 1 {
		t.Fatalf("next day morning: %v", got)
	}
}

func TestWorkerSkipsWhenNothingIsMissing(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	for day := 1; day <= 10; day++ {
		if err := f.db.SetLeetgrinderDay(ctx, day, true); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.step(t, at(10, "17:00")); len(got) != 0 {
		t.Fatalf("sent with nothing missing: %v", got)
	}
	for _, kind := range []string{leetgrinder.NotifyMissingWork, leetgrinder.NotifyBehindSchedule} {
		if e := f.logEntry(t, kind, at(10, "17:00")); e.Status != leetgrinder.NotifySkipped {
			t.Fatalf("%s: %+v", kind, e)
		}
	}
	// It was evaluated at 17:00, so reopening the day later sends nothing.
	if err := f.db.SetLeetgrinderDay(ctx, 10, false); err != nil {
		t.Fatal(err)
	}
	if got := f.step(t, at(10, "18:00")); len(got) != 0 {
		t.Fatalf("re-evaluated: %v", got)
	}
}

func TestWorkerRetriesFailuresUpToTheCap(t *testing.T) {
	f := newFixture(t)
	f.configure(t, func(s *leetgrinder.Settings) {
		s.Notifications = map[string]leetgrinder.NotificationPref{leetgrinder.NotifyBehindSchedule: {Enabled: false, Time: "17:00", Threshold: 3, Priority: "default"}}
	})
	f.ntfy.respond(500, true)
	sends := 0
	for minute := 0; minute <= 40; minute++ {
		sends += len(f.step(t, at(10, "17:00").Add(time.Duration(minute)*time.Minute)))
	}
	if sends != leetgrinder.NotifyMaxAttempts {
		t.Fatalf("sends: %d", sends)
	}
	e := f.logEntry(t, leetgrinder.NotifyMissingWork, at(10, "17:00"))
	if e.Status != leetgrinder.NotifyFailed || e.Attempts != leetgrinder.NotifyMaxAttempts || !strings.Contains(e.Detail, "ntfy returned 500") || strings.Contains(e.Detail, testToken) {
		t.Fatalf("log: %+v", e)
	}

	// A retry that succeeds is recorded as sent.
	f.ntfy.respond(200, false)
	if _, err := f.db.DB.Exec("UPDATE leetgrinder_notification_log SET attempts=2"); err != nil {
		t.Fatal(err)
	}
	if got := f.step(t, at(10, "18:00")); len(got) != 1 {
		t.Fatalf("retry: %v", got)
	}
	if e = f.logEntry(t, leetgrinder.NotifyMissingWork, at(10, "17:00")); e.Status != leetgrinder.NotifySent || e.Attempts != 3 {
		t.Fatalf("after retry: %+v", e)
	}
}

func TestWorkerReclaimsAbandonedSends(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	date := leetgrinder.Date(at(10, "17:00"), chicago)
	// A crash after claiming leaves a "sending" row behind.
	if _, ok, err := f.db.ClaimLeetgrinderNotification(ctx, leetgrinder.NotifyMissingWork, date, at(10, "17:00")); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if got := titles(f.step(t, at(10, "17:01"))); len(got) != 1 {
		t.Fatalf("only behind_schedule should send while the claim is fresh: %v", got)
	}
	if got := f.step(t, at(10, "17:05")); len(got) != 1 || got[0].Header.Get("Title") != "Leetgrinder: work left today" {
		t.Fatalf("abandoned send not retried: %v", got)
	}
}

func TestWorkerSkipsAbandonedSendWhoseConditionCleared(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	date := leetgrinder.Date(at(10, "17:00"), chicago)
	if _, ok, err := f.db.ClaimLeetgrinderNotification(ctx, leetgrinder.NotifyMissingWork, date, at(10, "17:00")); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if err := f.db.SetLeetgrinderDay(ctx, 10, true); err != nil {
		t.Fatal(err)
	}
	f.step(t, at(10, "17:01"))
	if e := f.logEntry(t, leetgrinder.NotifyMissingWork, at(10, "17:00")); e.Status != leetgrinder.NotifySending {
		t.Fatalf("fresh claim overwritten: %+v", e)
	}
	f.step(t, at(10, "17:05"))
	if e := f.logEntry(t, leetgrinder.NotifyMissingWork, at(10, "17:00")); e.Status != leetgrinder.NotifySkipped {
		t.Fatalf("abandoned send not skipped: %+v", e)
	}
	// Reopening the day later sends nothing.
	if err := f.db.SetLeetgrinderDay(ctx, 10, false); err != nil {
		t.Fatal(err)
	}
	for _, r := range f.step(t, at(10, "22:00")) {
		if r.Header.Get("Title") == "Leetgrinder: work left today" {
			t.Fatal("sent after the day was skipped")
		}
	}
}

func TestWorkerRespectsScheduleBounds(t *testing.T) {
	f := newFixture(t)
	// Before the start date.
	if got := f.step(t, time.Date(2026, 8, 31, 18, 0, 0, 0, chicago)); len(got) != 0 {
		t.Fatalf("sent before start: %v", got)
	}
	// The day after the end still reports being behind.
	end := time.Date(2026, 11, 23, 18, 0, 0, 0, chicago) // start + 83 days
	if got := titles(f.step(t, end.AddDate(0, 0, 1))); len(got) != 1 {
		t.Fatalf("end + 1 day: %v", got)
	}
	if got := f.step(t, end.AddDate(0, 0, 2)); len(got) != 0 {
		t.Fatalf("sent after end + 1 day: %v", got)
	}
	// No topic, or no schedule, means nothing is sent.
	f.configure(t, func(s *leetgrinder.Settings) { s.NtfyTopic = "" })
	if got := f.step(t, at(12, "18:00")); len(got) != 0 {
		t.Fatalf("sent without topic: %v", got)
	}
	f.configure(t, func(s *leetgrinder.Settings) { s.NtfyTopic, s.StartDate = "grind", nil })
	if got := f.step(t, at(13, "18:00")); len(got) != 0 {
		t.Fatalf("sent without schedule: %v", got)
	}
}

func TestWorkerLogsUnreadableToken(t *testing.T) {
	f := newFixture(t)
	f.worker.SecretKey = bytes.Repeat([]byte{1}, 32)
	if got := f.step(t, at(10, "17:00")); len(got) != 0 {
		t.Fatalf("sent without a readable token: %v", got)
	}
	if e := f.logEntry(t, leetgrinder.NotifyMissingWork, at(10, "17:00")); e.Status != leetgrinder.NotifyFailed || !strings.Contains(e.Detail, "re-enter it") {
		t.Fatalf("log: %+v", e)
	}
}
