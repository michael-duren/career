package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/modules"
)

type searchResult struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Module     string `json:"module"`
	Context    string `json:"context"`
	Excerpt    string `json:"excerpt"`
	TitleMatch bool   `json:"titleMatch"`
	Type       string `json:"type"`
}

type searchView struct {
	Results []searchResult `json:"results"`
	Totals  map[string]int `json:"totals"`
}

// GET /api/search finds goals, the next session of each scheduled title and
// Leetgrinder problems besides entries, caps each kind while counting every
// match, and leaves out turned-off modules.
func TestGlobalSearch(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	// Thursday Nov 26, 12:00 PM in Los Angeles.
	now := time.Date(2026, 11, 26, 20, 0, 0, 0, time.UTC)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.DB.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	goal := "6f1d2c3b-1a2b-4c3d-8e4f-5a6b7c8d9e0f"
	exec(`INSERT INTO goals (id,title,start_date,end_date,color,created_at,revision,position,updated_at,status) VALUES ($1,'Zephyr launch','2026-09-14','2026-12-31','#336699',now(),'r1',900,now(),'active')`, goal)
	exec(`INSERT INTO goal_steps (goal_id,id,position,title,done) VALUES ($1,'7f1d2c3b-1a2b-4c3d-8e4f-5a6b7c8d9e0f',0,'Draft the plan',false)`, goal)
	exec(`INSERT INTO leetgrinder_problems (slug,number,title,difficulty) VALUES ('zephyr-tower',9999,'Zephyr Tower','Hard')`)
	session := func(id, start, end, state, title string) string {
		plan := "null"
		if start != "" {
			plan = fmt.Sprintf(`{"start":%q,"end":%q}`, start, end)
		}
		return fmt.Sprintf(`%q:{"id":%q,"date":"2026-11-26","plan":%s,"state":%q,"assignment":{"title":%q,"goalId":%q}}`, id, id, plan, state, title, goal)
	}
	schedule := func(zone string, sessions ...string) {
		t.Helper()
		exec(`INSERT INTO scheduler_state (id,document) VALUES (1,$1::jsonb) ON CONFLICT (id) DO UPDATE SET document=EXCLUDED.document`, `{"settings":{"timeZone":"`+zone+`"},"sessions":{`+strings.Join(sessions, ",")+`}}`)
	}
	sessions := []string{
		session("ended", "2026-11-26T17:00:00Z", "2026-11-26T18:00:00Z", "accepted", "Zephyr launch"),    // 9 AM today, over
		session("canceled", "2026-11-26T22:00:00Z", "2026-11-26T23:00:00Z", "canceled", "Zephyr launch"), // 2 PM today, canceled
		session("unplanned", "", "", "accepted", "Zephyr launch"),                                        // logged work with no plan
		session("later", "2026-11-30T03:00:00Z", "2026-11-30T04:00:00Z", "accepted", "Zephyr launch"),
		session("next", "2026-11-27T03:00:00Z", "2026-11-27T04:00:00Z", "accepted", "Zephyr launch"),   // 7 PM today
		session("review", "2026-11-27T17:00:00Z", "2026-11-27T18:00:00Z", "accepted", "Zephyr review"), // 9 AM tomorrow
		session("retro", "2026-11-30T16:00:00Z", "2026-11-30T17:00:00Z", "accepted", "Zephyr retro"),   // 8 AM Monday
	}
	schedule("America/Los_Angeles", sessions...)
	// Seven documents mention zephyr in the body only, and one in its title.
	for i := range 7 {
		exec(`INSERT INTO documents (id,title,description,tags,body,revision,position,updated_at,is_core) VALUES ($1,$2,'','{}',$3,'r1',$4::bigint,now() - make_interval(days => $4::int),false)`,
			fmt.Sprintf("zephyr-doc-%d", i), fmt.Sprintf("Page %d", i), "# Heading\n\nSome words before the zephyr notes.", 1000+i)
	}
	exec(`INSERT INTO documents (id,title,description,tags,body,revision,position,updated_at,is_core) VALUES ('zephyr-title','Zephyr overview','','{}','Body','r1',2000,now() - interval '30 days',false)`)
	exec(`INSERT INTO books (slug,title,category,type,status,featured,priority,tags,body,revision,position,authors) VALUES ('zephyr-paper','Zephyr consensus','Systems','paper','reading',false,'medium','{}','','r1',3000,'{}')`)

	s := &Server{db: db, now: func() time.Time { return now }, config: config.Config{StaticDir: t.TempDir(), Username: "admin", JWTSecret: "test-secret", PublicOrigin: testOrigin}, attempts: map[string]attempt{}}
	handler := s.RegisterRoutes()
	search := func(query string) (int, searchView) {
		t.Helper()
		r := httptest.NewRequest("GET", "/api/search?"+query, nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var out searchView
		if w.Code == 200 {
			if err := decodeJSON(w, &out); err != nil {
				t.Fatal(err)
			}
		}
		return w.Code, out
	}

	code, got := search("q=zephyr")
	if code != 200 {
		t.Fatal(code)
	}
	byKind := map[string][]int{}
	for i, r := range got.Results {
		byKind[r.Kind] = append(byKind[r.Kind], i)
	}
	if len(byKind["goal"]) != 1 || len(byKind["problem"]) != 1 || got.Totals["goal"] != 1 || got.Totals["problem"] != 1 {
		t.Fatalf("goal and problem once: %+v", got.Results)
	}
	g := got.Results[byKind["goal"][0]]
	if g.ID != goal || g.Module != "timeline" || g.Context != "Active · Sep 14 → Dec 31" || !g.TitleMatch {
		t.Errorf("goal: %+v", g)
	}
	// The next planned, not canceled, not yet ended session of each title, soonest first, in the
	// scheduler's zone.
	contexts := []string{}
	for _, i := range byKind["session"] {
		r := got.Results[i]
		if r.Module != "schedule" {
			t.Errorf("session module: %+v", r)
		}
		contexts = append(contexts, r.ID+" "+r.Context)
	}
	want := []string{"next Today · 7:00 PM – 8:00 PM", "review Tomorrow · 9:00 AM – 10:00 AM", "retro Mon, Nov 30 · 8:00 AM – 9:00 AM"}
	if !slices.Equal(contexts, want) || got.Totals["session"] != 3 {
		t.Errorf("sessions: %q totals %v", contexts, got.Totals)
	}
	p := got.Results[byKind["problem"][0]]
	if p.ID != "zephyr-tower" || p.Module != "leetgrinder" || p.Context != "#9999 · Hard" {
		t.Errorf("problem: %+v", p)
	}
	// Books carry their item type so search can draw the type icon; other kinds have none.
	if b := byKind["book"]; len(b) != 1 || got.Results[b[0]].Type != "paper" || got.Results[b[0]].Module != "bookshelf" {
		t.Errorf("book: %+v", got.Results)
	}
	for _, r := range got.Results {
		if r.Kind != "book" && r.Type != "" {
			t.Errorf("type on a %s: %+v", r.Kind, r)
		}
	}
	// Documents: at most five, the title match first, all eight counted; excerpts are short.
	docs := byKind["document"]
	if len(docs) != 5 || got.Totals["document"] != 8 || got.Results[docs[0]].ID != "zephyr-title" || got.Results[docs[0]].Module != "" {
		t.Fatalf("documents: %v %+v", got.Totals, got.Results)
	}
	if e := got.Results[docs[1]].Excerpt; !strings.Contains(e, "zephyr") || got.Results[docs[1]].TitleMatch {
		t.Errorf("excerpt: %+v", got.Results[docs[1]])
	}
	// A match in a goal's steps finds the goal, not as a title match.
	if _, got = search("q=draft+the+plan&kind=goal"); len(got.Results) != 1 || got.Results[0].TitleMatch {
		t.Errorf("goal step match: %+v", got.Results)
	}
	// The query is literal: % and _ are not wildcards.
	if _, got = search("q=z%25r"); len(got.Results) != 0 {
		t.Errorf("wildcard matched: %+v", got.Results)
	}
	// An unknown zone falls back to UTC (7 PM Los Angeles is 3 AM tomorrow there).
	schedule("Mars/Olympus", sessions...)
	if _, got = search("q=zephyr+launch&kind=session"); len(got.Results) != 1 || got.Results[0].Context != "Tomorrow · 3:00 AM – 4:00 AM" {
		t.Errorf("UTC fallback: %+v", got.Results)
	}
	// A schedule that cannot be read drops sessions only.
	schedule("UTC", `"bad":{"id":"bad","date":"2026-11-26","plan":{"start":"soon","end":"later"},"state":"accepted","assignment":{"title":"Zephyr launch"}}`)
	if code, got = search("q=zephyr"); code != 200 || len(got.Results) == 0 || slices.ContainsFunc(got.Results, func(r searchResult) bool { return r.Kind == "session" }) {
		t.Errorf("unreadable schedule: %d %+v", code, got)
	}
	schedule("America/Los_Angeles", sessions...)

	if code, got = search("q=zephyr&kind=goal"); code != 200 || len(got.Results) != 1 || got.Results[0].Kind != "goal" {
		t.Errorf("kind filter: %d %+v", code, got)
	}
	if code, _ = search("q=zephyr&kind=nope"); code != 400 {
		t.Errorf("unknown kind: %d", code)
	}

	// Timeline off takes Schedule with it; Leetgrinder off hides problems.
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"timeline": false, "leetgrinder": false}}); err != nil {
		t.Fatal(err)
	}
	_, got = search("q=zephyr")
	for _, r := range got.Results {
		if r.Kind == "goal" || r.Kind == "session" || r.Kind == "problem" {
			t.Errorf("%s while its module is off: %+v", r.Kind, r)
		}
	}
	if _, ok := got.Totals["goal"]; ok || len(got.Results) == 0 {
		t.Errorf("totals while off: %v", got.Totals)
	}
	if code, got = search("q=zephyr&kind=session"); code != 200 || len(got.Results) != 0 {
		t.Errorf("session kind while off: %d %+v", code, got)
	}
}

func TestSearchExcerpt(t *testing.T) {
	long := strings.Repeat("alpha beta ", 20) + "the zephyr match" + strings.Repeat(" gamma delta", 20)
	got := searchExcerpt(long, "zephyr")
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "…") || !strings.Contains(got, "zephyr") || utf8.RuneCountInString(got) > 162 {
		t.Fatalf("%q", got)
	}
	// Whole words only at the start: every word is from the body's vocabulary.
	for _, word := range strings.Fields(strings.Trim(got, "…")) {
		if !slices.Contains([]string{"alpha", "beta", "the", "zephyr", "match", "gamma", "delta"}, word) {
			t.Errorf("partial word %q in %q", word, got)
		}
	}
	if got := searchExcerpt("zephyr at the start", "zephyr"); got != "zephyr at the start" {
		t.Errorf("%q", got)
	}
}
