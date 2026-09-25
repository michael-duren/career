package database

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/linkedin"
)

func TestLegacyContactsImportAsConnections(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	var workspace Entity
	json.Unmarshal(fixture(t), &workspace)
	connections := workspace["connections"].([]any)
	delete(workspace, "connections")
	company := workspace["companies"].([]any)[0].(map[string]any)
	company["contacts"] = []any{map[string]any{"id": "11111111-1111-4111-8111-111111111111", "name": "Zoë", "role": "", "email": "", "url": "", "notes": "Markdown **ok**"}}
	raw, _ := json.Marshal(workspace)
	counts, err := s.Import(ctx, bytes.NewReader(raw), Source{Store: "fixture", Key: "legacy", Checksum: uuid.NewString(), Archive: "test"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if counts["connections"] != 1 {
		t.Fatalf("counts %v", counts)
	}
	var out bytes.Buffer
	if err = s.Export(ctx, &out); err != nil {
		t.Fatal(err)
	}
	var got Entity
	json.Unmarshal(out.Bytes(), &got)
	if !reflect.DeepEqual(got["connections"], connections) {
		t.Fatalf("legacy contacts exported as %v", got["connections"])
	}
	c, _ := s.Detail(ctx, "company", "company/nested")
	if _, err = s.Save(ctx, "company", Entity(company), &c.Revision); err == nil {
		t.Fatal("interactive save accepted legacy contacts")
	}
}

func TestConnectionSaveAndPhoto(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	id := uuid.NewString()
	e := Entity{"id": id, "name": " Ada ", "role": "SRE", "companyName": "Example", "companySlug": "company/nested", "email": "", "url": "", "connectedOn": "", "lastContactedOn": "2026-09-01", "cadenceDays": float64(30), "queued": true, "notes": "", "tags": []any{}, "photo": "stale"}
	saved, err := s.Save(ctx, "connection", e, nil)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Entry["name"] != "Ada" || saved.Entry["url"] != nil || saved.Entry["photo"] != nil {
		t.Fatalf("saved %v", saved.Entry)
	}
	page, err := s.List(ctx, "connection", Filter{Limit: 100, Company: "company/nested"})
	if err != nil || len(page.Entries) != 2 {
		t.Fatalf("company filter %v %v", page, err)
	}
	if _, err = s.List(ctx, "book", Filter{Limit: 10, Linked: true}); !errors.Is(err, ErrInvalid) {
		t.Fatal("company filter accepted for books")
	}
	bad := Entity{}
	for k, v := range saved.Entry {
		bad[k] = v
	}
	bad["companySlug"] = "missing"
	if _, err = s.Save(ctx, "connection", bad, &saved.Revision); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing company: %v", err)
	}
	bad["cadenceDays"] = float64(0)
	if _, err = PrepareSave("connection", bad); err == nil {
		t.Fatal("zero cadence accepted")
	}

	rev, err := s.SetConnectionPhoto(ctx, id, "image/png", []byte("png"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SetConnectionPhoto(ctx, uuid.NewString(), "image/png", []byte("png")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("photo for missing connection: %v", err)
	}
	detail, _ := s.Detail(ctx, "connection", id)
	if detail.Entry["photo"] != rev {
		t.Fatalf("photo revision %v", detail.Entry["photo"])
	}
	// The projection is read-only: saving the entity back keeps the photo.
	if _, err = s.Save(ctx, "connection", detail.Entry, &detail.Revision); err != nil {
		t.Fatal(err)
	}
	if p, err := s.ConnectionPhoto(ctx, id); err != nil || string(p.Data) != "png" {
		t.Fatalf("photo %v %v", p, err)
	}
	detail, _ = s.Detail(ctx, "connection", id)
	if err = s.Delete(ctx, "connection", id, &detail.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConnectionPhoto(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatal("photo survived connection delete")
	}
}

func TestImportConnections(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	manual := uuid.NewString()
	if _, err := s.Save(ctx, "connection", Entity{"id": manual, "name": "Manual", "role": "", "companyName": "", "email": "", "url": "https://linkedin.com/in/Manual/", "queued": false, "notes": "keep", "tags": []any{}}, nil); err != nil {
		t.Fatal(err)
	}
	rows := []linkedin.Connection{
		{Name: "Ada", Role: "SRE", Company: "Example, Inc.", URL: "https://www.linkedin.com/in/ada", ConnectedOn: "2026-01-02"},
		{Name: "Bob", Company: "Elsewhere", Email: "not an email", URL: "https://www.linkedin.com/in/bob"},
		{Name: "Dup", URL: "https://www.linkedin.com/in/ADA/"},
		{Name: "No URL"},
		{Name: "Manual Person", URL: "https://www.linkedin.com/in/manual"},
	}
	messages := map[string]string{"https://www.linkedin.com/in/ada": "2026-05-01", "https://www.linkedin.com/in/manual": "2026-04-01"}
	got, err := s.ImportConnections(ctx, rows, messages)
	if err != nil {
		t.Fatal(err)
	}
	want := ConnectionImport{Parsed: 5, Created: 2, Updated: 1, Skipped: 2, Linked: 1, MessagesMatched: 2}
	if got != want {
		t.Fatalf("summary %+v", got)
	}
	byName := func() map[string]Entity {
		page, err := s.List(ctx, "connection", Filter{Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]Entity{}
		for _, r := range page.Entries {
			out[r.Entry["name"].(string)] = r.Entry
		}
		return out
	}
	all := byName()
	if all["Ada"]["companySlug"] != "company/nested" || all["Ada"]["lastContactedOn"] != "2026-05-01" || all["Ada"]["connectedOn"] != "2026-01-02" {
		t.Fatalf("ada %v", all["Ada"])
	}
	if all["Bob"]["email"] != "" || all["Bob"]["companySlug"] != nil {
		t.Fatalf("bob %v", all["Bob"])
	}
	if all["Manual Person"]["id"] != manual || all["Manual Person"]["notes"] != "keep" || all["Manual Person"]["lastContactedOn"] != "2026-04-01" {
		t.Fatalf("manual %v", all["Manual Person"])
	}

	// Manual link on Bob survives a re-import with the same LinkedIn company;
	// a changed company re-matches Ada.
	bob, _ := s.Detail(ctx, "connection", all["Bob"]["id"].(string))
	bob.Entry["companySlug"] = "company/nested"
	bob.Entry["notes"] = "mine"
	if _, err = s.Save(ctx, "connection", bob.Entry, &bob.Revision); err != nil {
		t.Fatal(err)
	}
	rows[0].Company = "Other Co"
	got, err = s.ImportConnections(ctx, rows, map[string]string{"https://www.linkedin.com/in/ada": "2026-03-01"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Updated != 1 || got.Unchanged != 2 || got.Created != 0 {
		t.Fatalf("reimport %+v", got)
	}
	all = byName()
	if all["Bob"]["companySlug"] != "company/nested" || all["Bob"]["notes"] != "mine" {
		t.Fatalf("bob after reimport %v", all["Bob"])
	}
	if all["Ada"]["companySlug"] != nil || all["Ada"]["lastContactedOn"] != "2026-05-01" {
		t.Fatalf("ada after reimport %v", all["Ada"])
	}
}

func TestMigration004RepairsLegacyContacts(t *testing.T) {
	s := emptyStore(t)
	ctx := context.Background()
	// Apply 001-003 by hand so 004 runs against populated company_contacts.
	if _, err := s.DB.Exec("CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())"); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"001_workspace.sql", "002_running_notes.sql", "003_goal_dependencies.sql"} {
		version := i + 1
		b, _ := migrations.ReadFile("migrations/" + name)
		if _, err := s.DB.Exec(string(b)); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.Exec("INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", version, fmt.Sprintf("%x", sha256.Sum256(b))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.DB.Exec(`INSERT INTO companies(slug,title,category,type,url,status,featured,priority,tags,body,revision,position,contacts_present) VALUES
		('a','Acme','X','company','https://a.co','applied',false,'high','{}','','r1',1,true),('b','Beta','X','company','https://b.co','applied',false,'high','{}','','r2',2,true)`); err != nil {
		t.Fatal(err)
	}
	shared := "11111111-1111-4111-8111-111111111111"
	if _, err := s.DB.Exec(`INSERT INTO company_contacts VALUES
		('a',$1,0,'Sam','Eng','sam@a.co','https://www.linkedin.com/in/sam','hi'),
		('b',$1,0,'Sam again','','','https://WWW.linkedin.com/in/sam','second job'),
		('a','21111111-1111-4111-8111-111111111111',1,'Bad','','not an email','linkedin.com/in/bad','')`, shared); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	page, err := s.List(ctx, "connection", Filter{Limit: 100})
	if err != nil || len(page.Entries) != 3 {
		t.Fatalf("migrated %v %v", page, err)
	}
	byName := map[string]Result{}
	for _, r := range page.Entries {
		byName[r.Entry["name"].(string)] = r
	}
	if byName["Sam"].Entry["id"] != shared || byName["Sam"].Entry["url"] != "https://www.linkedin.com/in/sam" {
		t.Fatalf("first Sam %v", byName["Sam"].Entry)
	}
	again := byName["Sam again"].Entry
	if again["id"] == shared || again["url"] != nil || again["notes"] != "second job" || again["companySlug"] != "b" {
		t.Fatalf("second Sam %v", again)
	}
	bad := byName["Bad"]
	if bad.Entry["url"] != nil || bad.Entry["email"] != "" || bad.Entry["notes"] != "Profile: linkedin.com/in/bad\nEmail: not an email" {
		t.Fatalf("bad %v", bad.Entry)
	}
	// Repaired rows stay editable through the normal save path.
	bad.Entry["queued"] = true
	if _, err = s.Save(ctx, "connection", bad.Entry, &bad.Revision); err != nil {
		t.Fatal(err)
	}
}

// Contacts were URL- and email-validated on save, so archives only need the
// duplicate URL and reused-id repairs.
func TestLegacyArchiveRepairsContacts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	var workspace Entity
	json.Unmarshal(fixture(t), &workspace)
	delete(workspace, "connections")
	company := workspace["companies"].([]any)[0].(map[string]any)
	other := map[string]any{}
	for k, v := range company {
		other[k] = v
	}
	other["slug"] = "company/other"
	shared := "11111111-1111-4111-8111-111111111111"
	contact := func(name, email, url string) map[string]any {
		return map[string]any{"id": shared, "name": name, "role": "", "email": email, "url": url, "notes": ""}
	}
	company["contacts"] = []any{contact("Zoë", "", "https://www.linkedin.com/in/zoe")}
	other["contacts"] = []any{contact("Zoë elsewhere", "", "https://www.linkedin.com/in/ZOE")}
	workspace["companies"] = []any{company, other}
	raw, _ := json.Marshal(workspace)
	counts, err := s.Import(ctx, bytes.NewReader(raw), Source{Store: "fixture", Key: "legacy-dirty", Checksum: uuid.NewString(), Archive: "test"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if counts["connections"] != 2 {
		t.Fatalf("counts %v", counts)
	}
	page, _ := s.List(ctx, "connection", Filter{Limit: 100})
	for _, r := range page.Entries {
		e := r.Entry
		if e["name"] == "Zoë elsewhere" && (e["id"] == shared || e["url"] != nil) {
			t.Fatalf("duplicate not repaired %v", e)
		}
	}
}

func TestImportRespectsUnlinkAndAdoptsURLlessRows(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	// The fixture's migrated contact "Zoë" at Example has no URL.
	rows := []linkedin.Connection{{Name: "Zoë", Company: "Example", URL: "https://www.linkedin.com/in/zoe"}, {Name: "Ada", Company: "Example", URL: "https://www.linkedin.com/in/ada"}}
	got, err := s.ImportConnections(ctx, rows, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Created != 1 || got.Updated != 1 {
		t.Fatalf("summary %+v", got)
	}
	zoe, _ := s.Detail(ctx, "connection", "11111111-1111-4111-8111-111111111111")
	if zoe.Entry["url"] != "https://www.linkedin.com/in/zoe" || zoe.Entry["notes"] != "Markdown **ok**" {
		t.Fatalf("zoe not adopted %v", zoe.Entry)
	}
	delete(zoe.Entry, "companySlug")
	if _, err = s.Save(ctx, "connection", zoe.Entry, &zoe.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ImportConnections(ctx, rows, nil); err != nil {
		t.Fatal(err)
	}
	zoe, _ = s.Detail(ctx, "connection", "11111111-1111-4111-8111-111111111111")
	if zoe.Entry["companySlug"] != nil {
		t.Fatal("deliberate unlink was re-linked")
	}
}

func TestClipIsLinear(t *testing.T) {
	if got := clip(" "+strings.Repeat("é", 300)+"😀", 201); size(got) > 201 || !strings.HasPrefix(got, "é") {
		t.Fatalf("clip %q", got)
	}
	if got := clip("ab😀", 3); got != "ab" {
		t.Fatalf("clip splits surrogate pair: %q", got)
	}
	start := time.Now()
	clip(strings.Repeat("x", 5<<20), 200)
	if time.Since(start) > time.Second {
		t.Fatal("clip is not linear")
	}
}
