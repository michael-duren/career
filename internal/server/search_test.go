package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/modules"
)

type searchView struct {
	Results []struct {
		Kind       string `json:"kind"`
		ID         string `json:"id"`
		Title      string `json:"title"`
		Module     string `json:"module"`
		Context    string `json:"context"`
		Excerpt    string `json:"excerpt"`
		TitleMatch bool   `json:"titleMatch"`
	} `json:"results"`
	Totals map[string]int `json:"totals"`
}

// GET /api/search finds goals, the next session of each scheduled title and
// Leetgrinder problems besides entries, caps each kind while counting every
// match, and leaves out turned-off modules.
func TestGlobalSearch(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	day := func(offset int) string { return now.AddDate(0, 0, offset).Format("2006-01-02") }
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
	session := func(id, date, start, end, state, title string) string {
		return fmt.Sprintf(`%q:{"id":%q,"date":%q,"plan":{"start":"%sT%s:00Z","end":"%sT%s:00Z"},"state":%q,"assignment":{"title":%q,"goalId":%q}}`, id, id, date, date, start, date, end, state, title, goal)
	}
	sessions := strings.Join([]string{
		session("past", day(-2), "09:00", "10:00", "accepted", "Zephyr launch"),
		session("canceled", day(1), "08:00", "09:00", "canceled", "Zephyr launch"),
		session("later", day(3), "19:00", "20:00", "accepted", "Zephyr launch"),
		session("next", day(1), "19:00", "20:00", "accepted", "Zephyr launch"),
	}, ",")
	exec(`INSERT INTO scheduler_state (id,document) VALUES (1,$1::jsonb) ON CONFLICT (id) DO UPDATE SET document=EXCLUDED.document`, `{"settings":{"timeZone":"UTC"},"sessions":{`+sessions+`}}`)
	// Seven documents mention zephyr in the body only, and one in its title.
	for i := range 7 {
		exec(`INSERT INTO documents (id,title,description,tags,body,revision,position,updated_at,is_core) VALUES ($1,$2,'','{}',$3,'r1',$4::bigint,now() - make_interval(days => $4::int),false)`,
			fmt.Sprintf("zephyr-doc-%d", i), fmt.Sprintf("Page %d", i), "# Heading\n\nSome words before the zephyr notes.", 1000+i)
	}
	exec(`INSERT INTO documents (id,title,description,tags,body,revision,position,updated_at,is_core) VALUES ('zephyr-title','Zephyr overview','','{}','Body','r1',2000,now() - interval '30 days',false)`)

	s := &Server{db: db, config: config.Config{StaticDir: t.TempDir(), Username: "admin", JWTSecret: "test-secret", PublicOrigin: testOrigin}, attempts: map[string]attempt{}}
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
	if len(byKind["goal"]) != 1 || len(byKind["session"]) != 1 || len(byKind["problem"]) != 1 {
		t.Fatalf("goal, session and problem each once: %+v", got.Results)
	}
	g := got.Results[byKind["goal"][0]]
	if g.ID != goal || g.Module != "timeline" || g.Context != "Active · Sep 14 → Dec 31" || !g.TitleMatch {
		t.Errorf("goal: %+v", g)
	}
	// The soonest upcoming, not canceled, session of the title.
	sess := got.Results[byKind["session"][0]]
	if sess.ID != "next" || sess.Module != "schedule" || sess.Context != "Tomorrow · 7:00 PM – 8:00 PM" {
		t.Errorf("session: %+v", sess)
	}
	p := got.Results[byKind["problem"][0]]
	if p.ID != "zephyr-tower" || p.Module != "leetgrinder" || p.Context != "#9999 · Hard" {
		t.Errorf("problem: %+v", p)
	}
	// Documents: at most five, the title match first, all eight counted; excerpts are short.
	docs := byKind["document"]
	if len(docs) != 5 || got.Totals["document"] != 8 || got.Results[docs[0]].ID != "zephyr-title" || got.Results[docs[0]].Module != "" {
		t.Fatalf("documents: %v %+v", got.Totals, got.Results)
	}
	if e := got.Results[docs[1]].Excerpt; !strings.Contains(e, "zephyr") || got.Results[docs[1]].TitleMatch {
		t.Errorf("excerpt: %+v", got.Results[docs[1]])
	}
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
