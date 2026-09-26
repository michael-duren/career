package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s := emptyStore(t)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}

// emptyStore returns a disposable schema with no migrations applied.
func emptyStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	s, err := Open(url)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = s.DB.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	scoped, err := Open(url + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { scoped.Close(); s.DB.Exec("DROP SCHEMA " + schema + " CASCADE"); s.Close() })
	return scoped
}
func fixture(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("../../tests/fixtures/migration/workspace-v2.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func imported(t *testing.T) *Store {
	t.Helper()
	s := testStore(t)
	if _, err := s.Import(context.Background(), bytes.NewReader(fixture(t)), Source{Store: "fixture", Key: "content", Checksum: uuid.NewString(), Archive: "test"}, false); err != nil {
		t.Fatal(err)
	}
	return s
}

// withNoteCreatedAt is what an archive exports after import: notes that
// predate createdAt take their updatedAt.
func withNoteCreatedAt(workspace any) any {
	for _, v := range workspace.(map[string]any)["notes"].([]any) {
		n := v.(map[string]any)
		if _, ok := n["createdAt"]; !ok && n["updatedAt"] != nil {
			n["createdAt"] = n["updatedAt"]
		}
	}
	return workspace
}
func TestRoundTrip(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	var b bytes.Buffer
	if err := s.Export(ctx, &b); err != nil {
		t.Fatal(err)
	}
	var want, got any
	json.Unmarshal(fixture(t), &want)
	json.Unmarshal(b.Bytes(), &got)
	if !reflect.DeepEqual(withNoteCreatedAt(want), got) {
		t.Fatalf("round trip differs\n%s", b.String())
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(ctx, bytes.NewReader(fixture(t)), Source{}, false); err == nil {
		t.Fatal("repeat import accepted")
	}
	p, err := s.List(ctx, "book", Filter{Limit: 1})
	if err != nil || len(p.Entries) != 1 || p.NextOffset == nil {
		t.Fatalf("pagination: %+v %v", p, err)
	}
	if _, ok := p.Entries[0].Entry["body"]; ok {
		t.Fatal("list leaked body")
	}
	var total, done int
	if err = s.DB.QueryRow("SELECT chapter_count,completed_count FROM book_summaries WHERE book_slug='books/test'").Scan(&total, &done); err != nil || total != 2 || done != 1 {
		t.Fatalf("projection %d %d %v", total, done, err)
	}
}
func TestConcurrentRevisionsAndChildren(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	n, err := s.Detail(ctx, "note", "systems/nested-note")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := s.Save(ctx, "note", n.Entry, &n.Revision); results <- e }()
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflicts)
	}
	n, _ = s.Detail(ctx, "note", "systems/nested-note")
	g, _ := s.Detail(ctx, "goal", "22222222-2222-4222-8222-222222222222")
	results = make(chan error, 2)
	for kind, r := range map[string]Result{"note": n, "goal": g} {
		wg.Add(1)
		go func(k string, r Result) { defer wg.Done(); _, e := s.Save(ctx, k, r.Entry, &r.Revision); results <- e }(kind, r)
	}
	wg.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatal(e)
		}
	}
	c, _ := s.Detail(ctx, "company", "company/nested")
	if err = s.Delete(ctx, "company", "company/nested", &c.Revision); err != nil {
		t.Fatal(err)
	}
	var count int
	s.DB.QueryRow("SELECT count(*) FROM connections WHERE company_slug IS NULL AND company_name='Example'").Scan(&count)
	if count != 1 {
		t.Fatal("company delete should unlink, not remove, its connections")
	}
	doc, _ := s.Detail(ctx, "document", "index")
	if !errors.Is(s.Delete(ctx, "document", "index", &doc.Revision), ErrCore) {
		t.Fatal("core document deleted")
	}
	b, _ := s.Detail(ctx, "book", "books/test")
	if _, err = s.Toggle(ctx, "book", "books/test", b.Revision, 3, true); !errors.Is(err, ErrConflict) {
		t.Fatalf("non-task toggle: %v", err)
	}
	if _, err = s.Toggle(ctx, "book", "books/test", b.Revision, 5, true); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Toggle(ctx, "book", "books/test", b.Revision, 5, false); !errors.Is(err, ErrConflict) {
		t.Fatal("stale toggle accepted")
	}
}
func TestImportRollbackAndOptionalPresence(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	source := Source{Store: "fixture", Checksum: "test", Archive: "test"}
	if _, e := s.Import(ctx, bytes.NewReader(fixture(t)), source, true); e != nil {
		t.Fatal(e)
	}
	var n int
	s.DB.QueryRow("SELECT count(*) FROM notes").Scan(&n)
	if n != 0 {
		t.Fatal("dry run wrote data")
	}
	bad := bytes.Replace(fixture(t), []byte(`"endDate": "2026-09-14"`), []byte(`"endDate": "2026-09-13"`), 1)
	if _, err := s.Import(ctx, bytes.NewReader(bad), source, false); err == nil {
		t.Fatal("invalid import accepted")
	}
	s.DB.QueryRow("SELECT count(*) FROM notes").Scan(&n)
	if n != 0 {
		t.Fatal("failed import wrote data")
	}
	raw := []byte(`{"version":2,"notes":[],"weeks":[],"books":[],"companies":[],"documents":[]}`)
	if _, err := s.Import(ctx, bytes.NewReader(raw), source, false); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := s.Export(ctx, &out); err != nil {
		t.Fatal(err)
	}
	var a, b any
	json.Unmarshal(raw, &a)
	json.Unmarshal(out.Bytes(), &b)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("presence mismatch %s", out.String())
	}
}
func TestValidation(t *testing.T) {
	var w map[string]any
	json.Unmarshal(fixture(t), &w)
	for kind, m := range models {
		entries, _ := w[m.Collection].([]any)
		for _, e := range entries {
			if err := Validate(kind, e.(map[string]any)); err != nil {
				t.Fatalf("%s: %v", kind, err)
			}
		}
	}
	n := w["notes"].([]any)[0].(map[string]any)
	n["unknown"] = true
	if Validate("note", n) == nil {
		t.Fatal("unknown field accepted")
	}
}
func TestMissingDatabaseDoesNotExit(t *testing.T) {
	s, err := Open("postgres://invalid:invalid@127.0.0.1:1/missing?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Health()["status"] != "down" {
		t.Fatal("expected down")
	}
}
func TestParentChildRollback(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	g, _ := s.Detail(ctx, "goal", "22222222-2222-4222-8222-222222222222")
	_, err := s.DB.Exec(`CREATE FUNCTION reject_steps() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test child failure'; END $$; CREATE TRIGGER reject_step BEFORE INSERT ON goal_steps FOR EACH ROW EXECUTE FUNCTION reject_steps()`)
	if err != nil {
		t.Fatal(err)
	}
	g.Entry["title"] = "Should roll back"
	if _, err = s.Save(ctx, "goal", g.Entry, &g.Revision); err == nil {
		t.Fatal("child failure accepted")
	}
	after, _ := s.Detail(ctx, "goal", g.Entry["id"].(string))
	if after.Revision != g.Revision || after.Entry["title"] == g.Entry["title"] {
		t.Fatal("partial parent write")
	}
}
func ExampleFilter() {
	fmt.Println(Filter{Limit: 50}.Limit) // Output: 50
}

func TestExportSnapshotDuringWrite(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	before, _ := s.Detail(ctx, "note", "systems/nested-note")
	w := &onFirstWrite{hook: func() {
		edited := Entity{}
		for k, v := range before.Entry {
			edited[k] = v
		}
		edited["title"] = "Changed while exporting"
		if _, err := s.Save(ctx, "note", edited, &before.Revision); err != nil {
			t.Fatal(err)
		}
	}}
	if err := s.Export(ctx, w); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(w.Bytes(), []byte("Changed while exporting")) {
		t.Fatal("export mixed snapshots")
	}
	after, _ := s.Detail(ctx, "note", "systems/nested-note")
	if after.Entry["title"] != "Changed while exporting" {
		t.Fatal("concurrent write missing")
	}
	var schema string
	if err := s.DB.QueryRow("SHOW search_path").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	s.Close()
	url := os.Getenv("TEST_DATABASE_URL")
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	reopened, err := Open(url + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	persisted, err := reopened.Detail(ctx, "note", "systems/nested-note")
	if err != nil || persisted.Revision != after.Revision {
		t.Fatalf("restart lost data: %v", err)
	}
}

type onFirstWrite struct {
	bytes.Buffer
	hook func()
}

func (w *onFirstWrite) Write(p []byte) (int, error) {
	if w.hook != nil {
		f := w.hook
		w.hook = nil
		f()
	}
	return w.Buffer.Write(p)
}
func (w *onFirstWrite) WriteString(p string) (int, error) { return w.Write([]byte(p)) }

func TestQueryPlans(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, err := s.DB.Exec(`INSERT INTO notes(id,title,topic,description,tags,body,revision,position) SELECT 'note-'||n,'Title '||n,CASE WHEN n%10=0 THEN 'Systems' ELSE 'General' END,'',ARRAY[]::text[],repeat('body ',2000),'revision-'||n,n FROM generate_series(1,1000) n; ANALYZE notes`)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{`SELECT id,title,topic FROM notes WHERE topic='Systems' ORDER BY topic,title,id LIMIT 50`, `SELECT id,title,body,revision FROM notes WHERE id='note-500'`, `UPDATE notes SET title='Edited',revision='new' WHERE id='note-500' AND revision='revision-500'`} {
		rows, err := s.DB.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+query)
		if err != nil {
			t.Fatal(err)
		}
		var plan []string
		for rows.Next() {
			var line string
			rows.Scan(&line)
			plan = append(plan, line)
		}
		rows.Close()
		t.Log(query + "\n" + strings.Join(plan, "\n"))
		if !strings.Contains(strings.Join(plan, "\n"), "Index") {
			t.Fatal("expected indexed query path")
		}
	}
}

func TestOptionalEntityPresence(t *testing.T) {
	s := testStore(t)
	var workspace Entity
	json.Unmarshal(fixture(t), &workspace)
	workspace["weeks"].([]any)[0].(map[string]any)["targets"] = map[string]any{"only-target": float64(0)}
	workspace["personalJournal"] = []any{}
	workspace["goals"] = []any{}
	raw, _ := json.Marshal(workspace)
	if _, err := s.Import(context.Background(), bytes.NewReader(raw), Source{Store: "fixture", Key: "content", Checksum: "presence", Archive: "test"}, false); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := s.Export(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	var got Entity
	json.Unmarshal(out.Bytes(), &got)
	if !reflect.DeepEqual(withNoteCreatedAt(any(workspace)), any(got)) {
		t.Fatal("optional presence round trip differs")
	}
}

func TestTypeScriptProjectionParity(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	raw, err := os.ReadFile("../../tests/fixtures/migration/expected-projections.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected Entity
	json.Unmarshal(raw, &expected)
	for _, v := range expected["books"].([]any) {
		b := v.(map[string]any)
		chapters, err := childJSON(ctx, s.DB, `SELECT json_build_object('index',source_line,'done',checked,'label',label) FROM book_checklist WHERE owner_slug=$1 AND section IN ('chapters','modules','sections') ORDER BY position`, b["slug"])
		if err != nil || !reflect.DeepEqual(chapters, b["chapters"]) {
			t.Fatalf("chapters differ: %v", err)
		}
		logs, err := childJSON(ctx, s.DB, `SELECT json_build_object('date',log_date,'text',text) FROM book_logs WHERE owner_slug=$1 ORDER BY position`, b["slug"])
		if err != nil || !reflect.DeepEqual(logs, b["log"]) {
			t.Fatalf("book logs differ: %v", err)
		}
	}
	for _, v := range expected["companies"].([]any) {
		c := v.(map[string]any)
		steps, err := childJSON(ctx, s.DB, `SELECT json_build_object('index',source_line,'completed',checked,'label',label) FROM company_checklist WHERE owner_slug=$1 AND section='steps' ORDER BY position`, c["slug"])
		if err != nil || !reflect.DeepEqual(steps, c["steps"]) {
			t.Fatalf("steps differ: %+v %v", steps, err)
		}
		var why string
		s.DB.QueryRow("SELECT why FROM company_summaries WHERE company_slug=$1", c["slug"]).Scan(&why)
		if why != c["why"] {
			t.Fatal("why differs")
		}
		rows, err := s.DB.Query("SELECT text FROM company_logs WHERE owner_slug=$1 ORDER BY position", c["slug"])
		if err != nil {
			t.Fatal(err)
		}
		logs := []any{}
		for rows.Next() {
			var text string
			rows.Scan(&text)
			logs = append(logs, text)
		}
		rows.Close()
		if !reflect.DeepEqual(logs, c["logEntries"]) {
			t.Fatalf("company logs differ %+v", logs)
		}
	}
	if err = s.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}
}
func TestNoteTodos(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	todo := func(title string, done bool) any {
		return map[string]any{"id": uuid.NewString(), "title": title, "done": done}
	}
	note := Entity{"id": "todo-note", "title": "Todos", "topic": "General", "description": "", "tags": []any{}, "body": "", "todos": []any{todo(" first ", false), todo("second", true)}}
	e, err := PrepareSave("note", note)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := s.Save(ctx, "note", e, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Detail(ctx, "note", "todo-note")
	if err != nil {
		t.Fatal(err)
	}
	todos, _ := got.Entry["todos"].([]any)
	if len(todos) != 2 || todos[0].(map[string]any)["title"] != "first" || todos[1].(map[string]any)["done"] != true {
		t.Fatalf("todos not persisted in order: %+v", got.Entry["todos"])
	}
	delete(e, "todos")
	if _, err = s.Save(ctx, "note", e, &saved.Revision); err != nil {
		t.Fatal(err)
	}
	if got, err = s.Detail(ctx, "note", "todo-note"); err != nil || got.Entry["todos"] != nil {
		t.Fatalf("absent todos should clear and be omitted: %+v %v", got.Entry, err)
	}
	for _, bad := range []any{
		[]any{map[string]any{"id": uuid.NewString(), "title": " ", "done": false}},
		[]any{map[string]any{"id": uuid.NewString(), "title": "x"}},
		[]any{map[string]any{"id": "nope", "title": "x", "done": false}},
	} {
		note["todos"] = bad
		if _, err := PrepareSave("note", note); err == nil {
			t.Fatalf("invalid todos accepted: %+v", bad)
		}
	}
}

func TestNoteCardSummary(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	todo := func(done bool) any { return map[string]any{"id": uuid.NewString(), "title": "step", "done": done} }
	note := Entity{"id": "card-note", "title": "Card", "topic": "General", "description": "", "tags": []any{"go"}, "body": "", "createdAt": "2000-01-01T00:00:00Z", "todos": []any{todo(true), todo(false), todo(true)}}
	e, err := PrepareSave("note", note)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := s.Save(ctx, "note", e, nil)
	if err != nil {
		t.Fatal(err)
	}
	created, _ := saved.Entry["createdAt"].(string)
	if created == "" || strings.HasPrefix(created, "2000") {
		t.Fatalf("createdAt should be set by the server: %q", created)
	}
	summary := func() map[string]any {
		t.Helper()
		page, err := s.List(ctx, "note", Filter{})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range page.Entries {
			if r.Entry["id"] == "card-note" {
				if r.Entry["body"] != nil {
					t.Fatalf("list result should omit the body: %+v", r.Entry)
				}
				out, _ := r.Entry["summary"].(map[string]any)
				return out
			}
		}
		t.Fatal("note missing from list")
		return nil
	}
	if got := summary(); got["wordCount"] != float64(0) || got["todoCount"] != float64(3) || got["todoDone"] != float64(2) {
		t.Fatalf("unexpected list summary: %+v", got)
	}
	e["body"] = "  one two\n\nthree  "
	delete(e, "createdAt")
	if saved, err = s.Save(ctx, "note", e, &saved.Revision); err != nil {
		t.Fatal(err)
	}
	if saved.Entry["createdAt"] != created {
		t.Fatalf("createdAt changed on edit: %v != %v", saved.Entry["createdAt"], created)
	}
	if got := summary(); got["wordCount"] != float64(3) {
		t.Fatalf("unexpected word count: %+v", got)
	}
}
func TestImportedNoteCreatedAt(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	n, err := s.Detail(ctx, "note", "systems/nested-note")
	if err != nil {
		t.Fatal(err)
	}
	if n.Entry["createdAt"] == nil || n.Entry["createdAt"] != n.Entry["updatedAt"] {
		t.Fatalf("imported note without createdAt should take updatedAt: %+v", n.Entry)
	}
	saved, err := s.Save(ctx, "note", n.Entry, &n.Revision)
	if err != nil {
		t.Fatal(err)
	}
	// Edits drop the archive's exact spelling, so compare instants.
	before, _ := time.Parse(time.RFC3339Nano, n.Entry["createdAt"].(string))
	after, _ := time.Parse(time.RFC3339Nano, saved.Entry["createdAt"].(string))
	if !after.Equal(before) {
		t.Fatalf("edit replaced imported createdAt: %v != %v", after, before)
	}
}
