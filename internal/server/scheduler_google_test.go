package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/scheduler"
)

type googleFixture struct {
	s                  *Server
	now                time.Time
	mu                 sync.Mutex
	events             map[string]scheduler.GoogleEvent
	writes             []string
	deletes            []string
	busyCalls          int
	writeFailureStatus int
	failWrite          int
	failBusy           bool
	hook               func(*http.Request)
	tombstones         map[string]bool
}

func newGoogleFixture(t *testing.T, count int) *googleFixture {
	t.Helper()
	f := &googleFixture{now: time.Date(2026, 10, 5, 4, 0, 0, 0, time.UTC), events: map[string]scheduler.GoogleEvent{}, tombstones: map[string]bool{}}
	db := testDB(t)
	f.s = &Server{db: db, config: config.Config{SchedulerGoogleClientID: "client", SchedulerSecretKey: make([]byte, 32), PublicOrigin: testOrigin, Username: "admin", JWTSecret: "test-secret"}}
	f.s.config.SchedulerGoogleClientSecret = config.Secret("secret")
	f.s.now = func() time.Time { return f.now }
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.hook != nil {
			f.hook(r)
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/freeBusy") {
			f.busyCalls++
			if f.failBusy {
				status := f.writeFailureStatus
				if status == 0 {
					status = 503
				}
				w.WriteHeader(status)
				return
			}
			fmt.Fprint(w, `{"calendars":{"primary":{"busy":[]}}}`)
			return
		}
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		switch r.Method {
		case "PUT", "POST":
			var event scheduler.GoogleEvent
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			f.writes = append(f.writes, event.ID)
			if f.failWrite > 0 && len(f.writes) == f.failWrite {
				status := f.writeFailureStatus
				if status == 0 {
					status = 503
				}
				w.WriteHeader(status)
				return
			}
			if r.Method == "PUT" {
				if _, ok := f.events[event.ID]; !ok {
					w.WriteHeader(404)
					return
				}
			}
			if f.tombstones[event.ID] {
				if r.Method == "POST" {
					w.WriteHeader(409)
				} else {
					w.WriteHeader(410)
				}
				return
			}
			f.events[event.ID] = event
		case "DELETE":
			f.deletes = append(f.deletes, id)
			delete(f.events, id)
		default:
			t.Errorf("unexpected Google request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(provider.Close)
	f.s.schedulerGoogleClient = &scheduler.GoogleClient{HTTP: provider.Client(), APIBase: provider.URL}
	token, _ := json.Marshal(scheduler.GoogleToken{AccessToken: "access", RefreshToken: "refresh", Expiry: time.Now().Add(365 * 24 * time.Hour)})
	sealed, err := scheduler.SealGoogleSecret(f.s.config.SchedulerSecretKey, token)
	if err != nil {
		t.Fatal(err)
	}
	c := database.SchedulerGoogleConnection{AccountID: "account", CalendarID: "destination", Credentials: sealed, SelectedCalendars: []string{"primary"}}
	if _, err = db.SaveSchedulerGoogle(context.Background(), c, "0"); err != nil {
		t.Fatal(err)
	}
	doc := scheduler.New()
	doc.Settings.TimeZone = "UTC"
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("session-%03d", i)
		start := f.now.Add(time.Duration(5+(i/8)*24+i%8) * time.Hour)
		doc.Sessions[id] = scheduler.Session{ID: id, Date: start.Format("2006-01-02"), Assignment: scheduler.Assignment{Title: id}, State: "accepted", Plan: &scheduler.Plan{Start: start, End: start.Add(30 * time.Minute)}}
	}
	f.save(t, doc)
	return f
}
func (f *googleFixture) save(t *testing.T, doc scheduler.Document) {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.db.DB.Exec(`INSERT INTO scheduler_state(id,document,reconciled_at) VALUES(1,$1::jsonb,$2) ON CONFLICT(id) DO UPDATE SET document=EXCLUDED.document,reconciled_at=EXCLUDED.reconciled_at`, raw, f.now)
	if err != nil {
		t.Fatal(err)
	}
}
func (f *googleFixture) sync(t *testing.T, periodic bool) {
	t.Helper()
	if err := f.s.syncSchedulerGoogle(context.Background(), f.now, periodic); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerGoogleBoundedRetryAdvances(t *testing.T) {
	f := newGoogleFixture(t, 47)
	// Pre-existing remote identities make each logical write one HTTP request.
	for i := 0; i < 47; i++ {
		id := scheduler.GoogleEventID("account", fmt.Sprintf("session-%03d", i))
		f.events[id] = scheduler.GoogleEvent{ID: id}
	}
	f.failWrite = 6
	if err := f.s.syncSchedulerGoogle(context.Background(), f.now, false); err == nil {
		t.Fatal("expected partial provider failure")
	}
	completed := append([]string(nil), f.writes[:5]...)
	for i, id := range completed {
		if id != scheduler.GoogleEventID("account", fmt.Sprintf("session-%03d", i)) {
			t.Fatalf("batch order was nondeterministic: %v", completed)
		}
	}
	f.failWrite = 0
	before := len(f.writes)
	f.sync(t, false)
	if got := len(f.writes) - before; got > 20 {
		t.Fatalf("unbounded retry: %d writes", got)
	}
	for _, id := range completed {
		for _, write := range f.writes[before:] {
			if write == id {
				t.Fatalf("retry repeated completed unchanged event %s", id)
			}
		}
	}
	for i := 0; i < 4; i++ {
		f.sync(t, false)
	}
	mappings, err := f.s.db.SchedulerGoogleMappings(context.Background(), "account")
	if err != nil || len(mappings) != 47 {
		t.Fatalf("mappings=%d err=%v", len(mappings), err)
	}
	for _, m := range mappings {
		if f.events[m.EventID].Summary != m.SessionID {
			t.Fatalf("event %s never synchronized", m.SessionID)
		}
	}
	gen, err := f.s.db.SchedulerGoogleOutbox(context.Background())
	if err != nil || gen != 0 {
		t.Fatalf("outbox=%d %v", gen, err)
	}
	before = len(f.writes)
	f.sync(t, false)
	if len(f.writes) != before {
		t.Fatal("empty outbox performed writes")
	}
}
func TestSchedulerGoogleReadCacheAndFreshPlanning(t *testing.T) {
	f := newGoogleFixture(t, 0)
	ctx := context.Background()
	if _, err := f.s.schedulerPlanningAvailability(ctx, "2026-10-05"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.schedulerAvailability(ctx, "2026-10-05"); err != nil {
		t.Fatal(err)
	}
	if f.busyCalls != 1 {
		t.Fatalf("refresh plus week read queried FreeBusy %d times; want 1", f.busyCalls)
	}
	if _, err := f.s.schedulerPlanningAvailability(ctx, "2026-10-05"); err != nil {
		t.Fatal(err)
	}
	if f.busyCalls != 2 {
		t.Fatalf("planning reused stale read cache: %d", f.busyCalls)
	}
}

func TestSchedulerGooglePeriodicCursorRestoresEntireRemoteSet(t *testing.T) {
	f := newGoogleFixture(t, 47)
	for i := 0; i < 4; i++ {
		f.sync(t, false)
	}
	// Ordinary retries avoid healthy writes. Periodic reconciliation must still
	// restore remote edits and remotely deleted events across bounded batches.
	for id, event := range f.events {
		event.Summary = "remote edit"
		f.events[id] = event
	}
	deletedID := scheduler.GoogleEventID("account", "session-030")
	delete(f.events, deletedID)
	for i := 0; i < 3; i++ {
		f.sync(t, true)
	}
	mappings, err := f.s.db.SchedulerGoogleMappings(context.Background(), "account")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mappings {
		if f.events[m.EventID].Summary != m.SessionID {
			t.Fatalf("periodic cursor omitted %s", m.SessionID)
		}
	}
	if len(f.events) != 47 {
		t.Fatalf("remote set=%d", len(f.events))
	}
	if f.busyCalls != 3 {
		t.Fatalf("periodic refresh calls=%d", f.busyCalls)
	}
}
func TestSchedulerGoogleChangedPlanDuringBatchKeepsNewGeneration(t *testing.T) {
	f := newGoogleFixture(t, 3)
	var changed bool
	f.hook = func(r *http.Request) {
		if r.Method != "PUT" || changed {
			return
		}
		changed = true
		doc, err := f.s.db.SchedulerDocument(context.Background())
		if err != nil {
			t.Error(err)
			return
		}
		session := doc.Sessions["session-001"]
		session.Plan.Start = session.Plan.Start.Add(10 * time.Minute)
		session.Plan.End = session.Plan.End.Add(10 * time.Minute)
		session.Assignment.Title = "New plan"
		doc.Sessions[session.ID] = session
		f.save(t, doc)
	}
	f.sync(t, false)
	gen, err := f.s.db.SchedulerGoogleOutbox(context.Background())
	if err != nil || gen == 0 {
		t.Fatalf("new generation lost: %d %v", gen, err)
	}
	f.hook = nil
	before := len(f.writes)
	f.sync(t, false)
	id := scheduler.GoogleEventID("account", "session-001")
	if f.events[id].Summary != "New plan" {
		t.Fatalf("new plan lost: %+v", f.events[id])
	}
	if len(f.writes)-before != 1 {
		t.Fatalf("changed retry performed %d writes", len(f.writes)-before)
	}
}
func TestSchedulerGoogleCancellationAndReconnectPreserveIdentities(t *testing.T) {
	f := newGoogleFixture(t, 4)
	ctx, cancel := context.WithCancel(context.Background())
	f.hook = func(r *http.Request) {
		if len(f.writes) >= 2 {
			cancel()
		}
	}
	if err := f.s.syncSchedulerGoogle(ctx, f.now, false); err == nil {
		t.Fatal("canceled export succeeded")
	}
	f.hook = nil
	f.sync(t, false)
	mappings, err := f.s.db.SchedulerGoogleMappings(context.Background(), "account")
	if err != nil || len(mappings) != 4 {
		t.Fatalf("mapping recovery %d %v", len(mappings), err)
	}
	c, err := f.s.db.SchedulerGoogle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	sealed := append([]byte(nil), c.Credentials...)
	c.Credentials = nil
	c, err = f.s.db.SaveSchedulerGoogle(context.Background(), c, c.Revision)
	if err != nil {
		t.Fatal(err)
	}
	before := len(f.writes)
	f.sync(t, true)
	if len(f.writes) != before {
		t.Fatal("disconnected worker contacted provider")
	}
	c.Credentials = sealed
	if _, err = f.s.db.SaveSchedulerGoogle(context.Background(), c, c.Revision); err != nil {
		t.Fatal(err)
	}
	f.sync(t, false)
	if len(f.writes) != before {
		t.Fatal("reconnect repeated unchanged writes")
	}
	after, err := f.s.db.SchedulerGoogleMappings(context.Background(), "account")
	if err != nil {
		t.Fatal(err)
	}
	for i, m := range mappings {
		if m.EventID != after[i].EventID {
			t.Fatal("reconnect changed mapping")
		}
	}
}
func TestSchedulerGoogleCanceledSessionDeletesOnlyMappedEvent(t *testing.T) {
	f := newGoogleFixture(t, 2)
	f.sync(t, false)
	f.events["unrelated"] = scheduler.GoogleEvent{ID: "unrelated", Summary: "private meeting"}
	doc, err := f.s.db.SchedulerDocument(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	session := doc.Sessions["session-000"]
	session.State = "canceled"
	doc.Sessions[session.ID] = session
	f.save(t, doc)
	f.sync(t, false)
	if len(f.deletes) != 1 || f.deletes[0] != scheduler.GoogleEventID("account", session.ID) {
		t.Fatalf("deleted unrelated events: %v", f.deletes)
	}
	if f.events["unrelated"].Summary != "private meeting" {
		t.Fatal("unrelated event changed")
	}
}
func TestSchedulerGoogleMissingAuthorityMakesNoProviderWrites(t *testing.T) {
	for _, missing := range []string{"account", "destination"} {
		t.Run(missing, func(t *testing.T) {
			f := newGoogleFixture(t, 2)
			c, err := f.s.db.SchedulerGoogle(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if missing == "account" {
				c.AccountID = ""
			} else {
				c.CalendarID = ""
			}
			if _, err = f.s.db.SaveSchedulerGoogle(context.Background(), c, c.Revision); err != nil {
				t.Fatal(err)
			}
			if err = f.s.syncSchedulerGoogle(context.Background(), f.now, false); err == nil {
				t.Fatal("missing authority exported")
			}
			if len(f.writes) != 0 || len(f.deletes) != 0 {
				t.Fatal("missing identity contacted provider")
			}
		})
	}
}
func TestSchedulerGoogleCacheExpiryRangeSelectionAndFailure(t *testing.T) {
	f := newGoogleFixture(t, 0)
	ctx := context.Background()
	read := func(week string) {
		t.Helper()
		if _, err := f.s.schedulerAvailability(ctx, week); err != nil {
			t.Fatal(err)
		}
	}
	read("2026-10-05")
	read("2026-10-05")
	if f.busyCalls != 1 {
		t.Fatal("fresh cache unused")
	}
	f.now = f.now.Add(5 * time.Minute)
	read("2026-10-05")
	if f.busyCalls != 2 {
		t.Fatal("expired cache reused")
	}
	read("2027-04-05")
	if f.busyCalls != 3 {
		t.Fatal("larger range reused incomplete cache")
	}
	c, err := f.s.db.SchedulerGoogle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c.Busy = nil
	c.LastRefresh = nil
	c.BusyFrom = nil
	c.BusyTo = nil
	if _, err = f.s.db.SaveSchedulerGoogle(ctx, c, c.Revision); err != nil {
		t.Fatal(err)
	}
	read("2026-10-05")
	if f.busyCalls != 4 {
		t.Fatal("selection invalidation ignored")
	}
	f.failBusy = true
	if _, err = f.s.schedulerPlanningAvailability(ctx, "2026-10-05"); err == nil {
		t.Fatal("provider failure allowed planning")
	}
	if _, err = f.s.schedulerAvailability(ctx, "2026-10-05"); err == nil {
		t.Fatal("failed provider reported fresh read")
	}
	f.failBusy = false
	read("2026-10-05")
	c, err = f.s.db.SchedulerGoogle(ctx)
	if err != nil || c.Error != "" {
		t.Fatalf("recovery health: %+v %v", c, err)
	}
}
func TestSchedulerGoogleDisconnectDuringAvailabilityFetch(t *testing.T) {
	f := newGoogleFixture(t, 0)
	f.hook = func(r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/freeBusy") {
			return
		}
		c, err := f.s.db.SchedulerGoogle(context.Background())
		if err != nil {
			t.Error(err)
			return
		}
		c.Credentials = nil
		if _, err = f.s.db.SaveSchedulerGoogle(context.Background(), c, c.Revision); err != nil {
			t.Error(err)
		}
	}
	if _, err := f.s.schedulerPlanningAvailability(context.Background(), "2026-10-05"); err == nil {
		t.Fatal("disconnected in-flight availability accepted")
	}
	c, err := f.s.db.SchedulerGoogle(context.Background())
	if err != nil || len(c.Credentials) != 0 || c.LastRefresh != nil {
		t.Fatalf("stale fetch overwrote disconnect %+v %v", c, err)
	}
}
func TestSchedulerGoogleOverlappingRefreshCoalescesAcrossServers(t *testing.T) {
	f := newGoogleFixture(t, 0)
	entered := make(chan struct{})
	release := make(chan struct{})
	f.hook = func(r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/freeBusy") {
			close(entered)
			<-release
		}
	}
	errors := make(chan error, 2)
	go func() {
		_, err := f.s.fetchSchedulerAvailability(context.Background(), "2026-10-05", true, true)
		errors <- err
	}()
	<-entered
	other := Server{db: f.s.db, config: f.s.config, now: f.s.now, schedulerGoogleClient: f.s.schedulerGoogleClient}
	// An independent server shares PostgreSQL, not an in-memory mutex.
	started := make(chan struct{})
	go func() {
		close(started)
		_, err := other.fetchSchedulerAvailability(context.Background(), "2026-10-05", true, true)
		errors <- err
	}()
	<-started
	// Wait until the second database fetch is blocked on the availability lock.
	deadline := time.Now().Add(3 * time.Second)
	for {
		var n int
		err := f.s.db.DB.QueryRow(`SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND objid=724193622 AND NOT granted`).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second refresh did not wait for provider fetch")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(release)
	for i := 0; i < 2; i++ {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
	if f.busyCalls != 1 {
		t.Fatalf("overlapping refreshes queried %d times", f.busyCalls)
	}
}

func TestSchedulerGoogleRemoteTombstoneReplacesIdentityOnce(t *testing.T) {
	f := newGoogleFixture(t, 1)
	f.sync(t, false)
	old := scheduler.GoogleEventID("account", "session-000")
	delete(f.events, old)
	f.tombstones[old] = true
	f.sync(t, true)
	mappings, err := f.s.db.SchedulerGoogleMappings(context.Background(), "account")
	if err != nil || len(mappings) != 1 {
		t.Fatalf("mappings=%+v %v", mappings, err)
	}
	replacement := mappings[0].EventID
	if replacement == old || f.events[replacement].Summary != "session-000" {
		t.Fatalf("tombstone not recreated: %+v", mappings)
	}
	before := len(f.writes)
	f.sync(t, false)
	if len(f.writes) != before {
		t.Fatal("recreated identity not marked complete")
	}
	f.sync(t, true)
	after, err := f.s.db.SchedulerGoogleMappings(context.Background(), "account")
	if err != nil || after[0].EventID != replacement {
		t.Fatalf("replacement unstable: %+v %v", after, err)
	}
}
func TestSchedulerGoogleManualRefreshOneQueryAndLocalSaveSurvivesExportFailure(t *testing.T) {
	f := newGoogleFixture(t, 0)
	h := newOAuthHarness(t, f.s.db)
	h.handler = f.s.RegisterRoutes()
	headers := map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}
	refresh := h.do("POST", "/api/scheduler/google/refresh", `{"week":"2026-10-05"}`, headers, true)
	if refresh.Code != 200 {
		t.Fatalf("refresh=%d %s", refresh.Code, refresh.Body.String())
	}
	read := h.do("GET", "/api/scheduler/week?week=2026-10-05", "", nil, true)
	if read.Code != 200 {
		t.Fatalf("read=%d %s", read.Code, read.Body.String())
	}
	if f.busyCalls != 1 {
		t.Fatalf("manual refresh plus read made %d queries", f.busyCalls)
	}
	var week scheduler.Week
	if err := json.Unmarshal(read.Body.Bytes(), &week); err != nil {
		t.Fatal(err)
	}
	plan := &scheduler.Plan{Start: f.now.Add(5 * time.Hour), End: f.now.Add(6 * time.Hour)}
	raw, _ := json.Marshal(scheduler.Mutation{Action: "session", Week: week.Week, Revision: week.Revision, Session: &scheduler.Session{Date: week.Week, Assignment: scheduler.Assignment{Title: "Locally saved"}, Plan: plan}})
	write := h.do("POST", "/api/scheduler/mutate", string(raw), headers, true)
	if write.Code != 200 {
		t.Fatalf("local save=%d %s", write.Code, write.Body.String())
	}
	if f.busyCalls != 2 {
		t.Fatalf("local save did not freshly check availability: %d", f.busyCalls)
	}
	var saved scheduler.Week
	if err := json.Unmarshal(write.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	f.failWrite = 1
	if err := f.s.syncSchedulerGoogle(context.Background(), f.now, false); err == nil {
		t.Fatal("expected export failure")
	}
	doc, err := f.s.db.SchedulerDocument(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sessions) != 1 || doc.Revision != saved.Revision {
		t.Fatalf("export failure undid local save %+v", doc)
	}
	// Fresh provider failure keeps the next proposal uncommitted for client retry.
	f.failBusy = true
	raw, _ = json.Marshal(scheduler.Mutation{Action: "session", Week: saved.Week, Revision: saved.Revision, Session: &scheduler.Session{Date: saved.Week, Assignment: scheduler.Assignment{Title: "Draft"}, Plan: &scheduler.Plan{Start: plan.End, End: plan.End.Add(time.Hour)}}})
	write = h.do("POST", "/api/scheduler/mutate", string(raw), headers, true)
	if write.Code != 503 || !strings.Contains(write.Body.String(), "draft is kept") {
		t.Fatalf("failed check accepted draft: %d %s", write.Code, write.Body.String())
	}
	after, err := f.s.db.SchedulerDocument(context.Background())
	if err != nil || after.Revision != doc.Revision || len(after.Sessions) != 1 {
		t.Fatalf("failed availability mutated plan %+v %v", after, err)
	}
}
func TestSchedulerGoogleWorkerAndDisconnectShareAuthorityLock(t *testing.T) {
	f := newGoogleFixture(t, 2)
	h := newOAuthHarness(t, f.s.db)
	h.handler = f.s.RegisterRoutes()
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	f.hook = func(r *http.Request) {
		if r.Method == "PUT" {
			once.Do(func() { close(entered); <-release })
		}
	}
	done := make(chan error, 1)
	go func() { done <- f.s.syncSchedulerGoogle(context.Background(), f.now, false) }()
	<-entered
	other := Server{db: f.s.db, config: f.s.config, now: f.s.now, schedulerGoogleClient: f.s.schedulerGoogleClient}
	if err := other.syncSchedulerGoogle(context.Background(), f.now, false); err != nil {
		t.Fatal(err)
	}
	c, err := f.s.db.SchedulerGoogle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"revision": c.Revision})
	disconnected := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		disconnected <- h.do("POST", "/api/scheduler/google/disconnect", string(raw), map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}, true)
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var n int
		err := f.s.db.DB.QueryRow(`SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND objid=724193621 AND NOT granted`).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("disconnect did not wait for export")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	result := <-disconnected
	if result.Code != 200 {
		t.Fatalf("disconnect=%d %s", result.Code, result.Body.String())
	}
	before := len(f.writes)
	f.sync(t, false)
	if len(f.writes) != before {
		t.Fatal("worker wrote after disconnect")
	}
	if len(f.events) != 2 {
		t.Fatalf("concurrent worker changed remote set: %d", len(f.events))
	}
}

func TestSchedulerGooglePeriodicFailureKeepsCompletedCursor(t *testing.T) {
	f := newGoogleFixture(t, 27)
	for i := 0; i < 3; i++ {
		f.sync(t, false)
	}
	before := len(f.writes)
	f.failWrite = before + 6
	if err := f.s.syncSchedulerGoogle(context.Background(), f.now, true); err == nil {
		t.Fatal("expected periodic partial failure")
	}
	cursor, err := f.s.db.SchedulerGoogleReconciliationCursor(context.Background(), "account", "destination")
	if err != nil || cursor != "session-004" {
		t.Fatalf("successful reconciliation prefix lost: %s %v", cursor, err)
	}
	f.failWrite = 0
	before = len(f.writes)
	f.sync(t, true)
	if f.writes[before] != scheduler.GoogleEventID("account", "session-005") {
		t.Fatal("periodic retry restarted completed prefix")
	}
	f.sync(t, true)
	cursor, err = f.s.db.SchedulerGoogleReconciliationCursor(context.Background(), "account", "destination")
	if err != nil || cursor != "" {
		t.Fatalf("full pass cursor did not reset: %s %v", cursor, err)
	}
}

func TestSchedulerGoogleRevocationPersistsAfterConcurrentHealthWrite(t *testing.T) {
	f := newGoogleFixture(t, 1)
	f.failWrite = 1
	f.writeFailureStatus = 401
	f.hook = func(r *http.Request) {
		if r.Method != "PUT" {
			return
		}
		c, err := f.s.db.SchedulerGoogle(context.Background())
		if err != nil {
			t.Error(err)
			return
		}
		if err = f.s.db.UpdateSchedulerGoogleHealth(context.Background(), &c); err != nil {
			t.Error(err)
		}
	}
	if err := f.s.syncSchedulerGoogle(context.Background(), f.now, false); err == nil {
		t.Fatal("revocation succeeded")
	}
	c, err := f.s.db.SchedulerGoogle(context.Background())
	if err != nil || !c.ReconnectRequired {
		t.Fatalf("worker revocation health lost: reconnect=%v error=%s err=%v", c.ReconnectRequired, c.Error, err)
	}
	before := len(f.writes)
	f.sync(t, true)
	if len(f.writes) != before {
		t.Fatal("revoked authorization retried")
	}
}
