package database

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/linkedin"
)

func newCompany(title string) Entity {
	return Entity{"slug": uuid.NewString(), "title": title, "category": "Test", "type": "company", "url": "https://example.com", "status": "not_started", "featured": false, "priority": "medium", "tags": []any{}, "body": "## Log\n"}
}

func TestConnectionCompaniesAndAddCompanies(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if _, err := s.Save(ctx, "company", Entity{"slug": "tracked/globex", "title": "Globex", "category": "Test", "type": "company", "url": "https://globex.example", "status": "applied", "featured": false, "priority": "high", "tags": []any{}, "body": ""}, nil); err != nil {
		t.Fatal(err)
	}
	rows := []linkedin.Connection{
		{Name: "Ada", Role: "Staff Engineer", Company: "Acme, Inc.", URL: "https://linkedin.com/in/ada"},
		{Name: "Bob", Role: "Recruiter", Company: "ACME", URL: "https://linkedin.com/in/bob"},
		{Name: "Cy", Role: "Engineering Manager", Company: "Acme Inc", URL: "https://linkedin.com/in/cy"},
		{Name: "Dee", Role: "SRE", Company: "Globex Corporation", URL: "https://linkedin.com/in/dee"},
		{Name: "Eve", Role: "Engineer", Company: "Initech", URL: "https://linkedin.com/in/eve"},
		{Name: "Fay", Role: "Founder", Company: "", URL: "https://linkedin.com/in/fay"},
	}
	if _, err := s.ImportConnections(ctx, rows, map[string]string{"https://www.linkedin.com/in/bob": "2026-08-01", "https://www.linkedin.com/in/eve": "2026-09-01"}); err != nil {
		t.Fatal(err)
	}

	page, err := s.ConnectionCompanies(ctx, ConnectionCompanyFilter{Limit: 2, People: 2})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Companies) != 2 || page.NextOffset == nil || *page.NextOffset != 2 {
		t.Fatalf("page %+v", page)
	}
	acme := page.Companies[0]
	if acme.Name != "ACME" || acme.Connections != 3 || acme.TrackedSlug != "" || acme.LastContactedOn != "2026-08-01" {
		t.Fatalf("acme %+v", acme)
	}
	if len(acme.People) != 2 || acme.People[0].Name != "Bob" {
		t.Fatalf("recently contacted people should come first: %+v", acme.People)
	}
	if globex := page.Companies[1]; globex.TrackedSlug != "tracked/globex" || globex.TrackedTitle != "Globex" {
		t.Fatalf("globex should match the tracked company: %+v", globex)
	}
	page, _ = s.ConnectionCompanies(ctx, ConnectionCompanyFilter{Limit: 50, Untracked: true, Role: "engineer"})
	if page.Total != 2 || page.Companies[0].Connections != 2 || page.Companies[1].Name != "Initech" {
		t.Fatalf("role filter %+v", page)
	}
	page, _ = s.ConnectionCompanies(ctx, ConnectionCompanyFilter{Limit: 50, Sort: "recent", MinConnections: 1, Query: "cme"})
	if page.Total != 1 || page.Companies[0].Connections != 3 {
		t.Fatalf("query/recent %+v", page)
	}
	page, _ = s.ConnectionCompanies(ctx, ConnectionCompanyFilter{Limit: 50, Sort: "recent"})
	if page.Companies[0].Name != "Initech" {
		t.Fatalf("recent sort %+v", page)
	}
	if _, err = s.ConnectionCompanies(ctx, ConnectionCompanyFilter{Limit: 50, Sort: "size"}); !errors.Is(err, ErrInvalid) {
		t.Fatal("bad sort accepted", err)
	}

	results, err := s.AddCompanies(ctx, []Entity{newCompany("Acme"), newCompany("GLOBEX, Inc."), newCompany("acme inc"), newCompany("Initech")})
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Created || results[0].Linked != 3 || results[1].Created || results[1].Slug != "tracked/globex" || results[1].Title != "Globex" || results[2].Created || results[2].Slug != results[0].Slug || !results[3].Created || results[3].Linked != 1 {
		t.Fatalf("results %+v", results)
	}
	bob, _ := s.ConnectionCompanies(ctx, ConnectionCompanyFilter{Limit: 50, Query: "acme", People: 20})
	if bob.Companies[0].TrackedSlug != results[0].Slug || bob.Companies[0].LastContactedOn != "2026-08-01" {
		t.Fatalf("linking changed contact dates or missed the link: %+v", bob)
	}
	c, err := s.Detail(ctx, "company", results[0].Slug)
	if err != nil || c.Revision != results[0].Revision || c.Entry["status"] != "not_started" {
		t.Fatal(c, err)
	}
	// Retrying is a no-op.
	again, err := s.AddCompanies(ctx, []Entity{newCompany("Initech")})
	if err != nil || again[0].Created || again[0].Slug != results[3].Slug {
		t.Fatal(again, err)
	}
	// Invalid input saves nothing.
	bad := newCompany("Hooli")
	bad["url"] = "not a url"
	if _, err = s.AddCompanies(ctx, []Entity{newCompany("Pied Piper"), bad}); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid company accepted", err)
	}
	if page, _ := s.List(ctx, "company", Filter{Limit: 100}); len(page.Entries) != 3 {
		t.Fatalf("partial batch saved: %d companies", len(page.Entries))
	}
}

func TestAddCompaniesAgreesWithTracked(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	datadog := newCompany("Datadog (DDOG)")
	datadog["slug"] = "datadog"
	globex := newCompany("Globex")
	for _, c := range []Entity{datadog, newCompany("Amazon Web Services"), globex} {
		if _, err := s.Save(ctx, "company", c, nil); err != nil {
			t.Fatal(err)
		}
	}
	rows := []linkedin.Connection{
		{Name: "Ada", Role: "SRE", Company: "Datadog", URL: "https://linkedin.com/in/ada"},
		{Name: "Bob", Role: "SRE", Company: "Umbrella", URL: "https://linkedin.com/in/bob"},
		{Name: "Cy", Role: "SRE", Company: "Amazon Web Services", URL: "https://linkedin.com/in/cy"},
		{Name: "Dee", Role: "SRE", Company: "Amazon", URL: "https://linkedin.com/in/dee"},
		{Name: "Eve", Role: "SRE", Company: "---", URL: "https://linkedin.com/in/eve"},
	}
	if _, err := s.ImportConnections(ctx, rows, nil); err != nil {
		t.Fatal(err)
	}
	// Bob was linked by hand to another company; Cy was deliberately unlinked.
	if _, err := s.DB.ExecContext(ctx, "UPDATE connections SET company_slug=CASE name WHEN 'Bob' THEN $1 END WHERE name IN ('Bob','Cy')", globex["slug"]); err != nil {
		t.Fatal(err)
	}
	page, err := s.ConnectionCompanies(ctx, ConnectionCompanyFilter{Limit: 50, Sort: "name"})
	if err != nil {
		t.Fatal(err)
	}
	tracked := map[string]string{}
	for _, c := range page.Companies {
		tracked[c.Name] = c.TrackedSlug
	}
	if len(page.Companies) != 4 || tracked["Datadog"] != "datadog" || tracked["Umbrella"] != globex["slug"] || tracked["Amazon"] != "" {
		t.Fatalf("tracked %v", tracked)
	}
	results, err := s.AddCompanies(ctx, []Entity{newCompany("Datadog"), newCompany("umbrella"), newCompany("Amazon")})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Created || results[0].Slug != "datadog" || results[0].Title != "Datadog (DDOG)" || results[1].Created || results[1].Title != "Globex" {
		t.Fatalf("names listed as tracked were added: %+v", results)
	}
	// Dee joins the new company; Cy still best matches Amazon Web Services and stays unlinked.
	if !results[2].Created || results[2].Linked != 1 {
		t.Fatalf("amazon %+v", results[2])
	}
	var cy *string
	if err = s.DB.QueryRowContext(ctx, "SELECT company_slug FROM connections WHERE name='Cy'").Scan(&cy); err != nil || cy != nil {
		t.Fatalf("new shorter name took a connection from an existing company: %v %v", cy, err)
	}
	if _, err = s.AddCompanies(ctx, []Entity{newCompany("!!!")}); !errors.Is(err, ErrInvalid) {
		t.Fatal("punctuation-only title accepted", err)
	}
}
