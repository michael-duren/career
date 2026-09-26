package database

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/linkedin"
)

// ConnectionCompany is one employer from the connections list, grouped by
// normalized company name so "Acme" and "Acme, Inc." count together.
type ConnectionCompany struct {
	Name            string             `json:"name"`
	Connections     int                `json:"connections"`
	LastContactedOn string             `json:"lastContactedOn,omitempty"`
	TrackedSlug     string             `json:"trackedSlug,omitempty"`
	TrackedTitle    string             `json:"trackedTitle,omitempty"`
	People          []ConnectionSample `json:"people"`
}

type ConnectionSample struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Role            string `json:"role"`
	LastContactedOn string `json:"lastContactedOn,omitempty"`
}

type ConnectionCompanyFilter struct {
	Query          string // company name substring, case-insensitive
	Role           string // only count people whose role contains this, case-insensitive
	MinConnections int
	Untracked      bool // hide companies already on the companies board
	Sort           string
	Offset, Limit  int
	People         int // people listed per company
}

type ConnectionCompanyPage struct {
	Companies  []ConnectionCompany `json:"companies"`
	Total      int                 `json:"total"`
	NextOffset *int                `json:"nextOffset"`
}

// companyKey is the case-insensitive identity used to group and deduplicate
// company names; punctuation-only names fall back to their lowercase text.
func companyKey(name string) string {
	if key := linkedin.NormalizeCompany(name); key != "" {
		return key
	}
	return strings.ToLower(strings.TrimSpace(name))
}

func trackedCompanies(ctx context.Context, q queryer) ([]linkedin.Company, error) {
	rows, err := q.QueryContext(ctx, "SELECT slug,title FROM companies ORDER BY position")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	companies := []linkedin.Company{}
	for rows.Next() {
		var c linkedin.Company
		if err = rows.Scan(&c.Slug, &c.Title); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}
	return companies, rows.Err()
}

type companyGroup struct {
	ConnectionCompany
	spellings map[string]int
	slugs     map[string]int
	people    []ConnectionSample
}

// ConnectionCompanies aggregates connections by employer, marking employers
// that already match a tracked company (by link or by the import matcher).
func (s *Store) ConnectionCompanies(ctx context.Context, f ConnectionCompanyFilter) (ConnectionCompanyPage, error) {
	page := ConnectionCompanyPage{Companies: []ConnectionCompany{}}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.People < 0 || f.People > 20 || !allowed(cmp.Or(f.Sort, "connections"), "connections|name|recent") {
		return page, ErrInvalid
	}
	companies, err := trackedCompanies(ctx, s.DB)
	if err != nil {
		return page, err
	}
	titles := map[string]string{}
	for _, c := range companies {
		titles[c.Slug] = c.Title
	}
	matcher := linkedin.NewMatcher(companies)
	rows, err := s.DB.QueryContext(ctx, "SELECT id::text,name,role,company_name,company_slug,to_char(last_contacted_on,'YYYY-MM-DD') FROM connections WHERE company_name<>''")
	if err != nil {
		return page, err
	}
	defer rows.Close()
	role := strings.ToLower(strings.TrimSpace(f.Role))
	groups := map[string]*companyGroup{}
	for rows.Next() {
		var p ConnectionSample
		var company string
		var slug, last sql.NullString
		if err = rows.Scan(&p.ID, &p.Name, &p.Role, &company, &slug, &last); err != nil {
			return page, err
		}
		if role != "" && !strings.Contains(strings.ToLower(p.Role), role) {
			continue
		}
		p.LastContactedOn = last.String
		key := companyKey(company)
		g := groups[key]
		if g == nil {
			g = &companyGroup{spellings: map[string]int{}, slugs: map[string]int{}}
			groups[key] = g
		}
		g.Connections++
		g.spellings[company]++
		if slug.Valid {
			g.slugs[slug.String]++
		}
		g.LastContactedOn = max(g.LastContactedOn, p.LastContactedOn)
		g.people = append(g.people, p)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	query := strings.ToLower(strings.TrimSpace(f.Query))
	out := []ConnectionCompany{}
	for key, g := range groups {
		g.Name = mostCommon(g.spellings)
		if query != "" && !strings.Contains(strings.ToLower(g.Name), query) && !strings.Contains(key, query) {
			continue
		}
		if g.Connections < f.MinConnections {
			continue
		}
		g.TrackedSlug = mostCommon(g.slugs)
		if g.TrackedSlug == "" {
			g.TrackedSlug = matcher.Match(g.Name)
		}
		g.TrackedTitle = titles[g.TrackedSlug]
		if f.Untracked && g.TrackedSlug != "" {
			continue
		}
		// Recently contacted people first, since they are the warmest leads.
		slices.SortFunc(g.people, func(a, b ConnectionSample) int {
			return cmp.Or(strings.Compare(b.LastContactedOn, a.LastContactedOn), strings.Compare(a.Name, b.Name), strings.Compare(a.ID, b.ID))
		})
		g.People = g.people[:min(len(g.people), f.People)]
		out = append(out, g.ConnectionCompany)
	}
	slices.SortFunc(out, func(a, b ConnectionCompany) int {
		byName := cmp.Or(strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), strings.Compare(a.Name, b.Name))
		switch f.Sort {
		case "name":
			return byName
		case "recent":
			return cmp.Or(strings.Compare(b.LastContactedOn, a.LastContactedOn), b.Connections-a.Connections, byName)
		}
		return cmp.Or(b.Connections-a.Connections, byName)
	})
	page.Total = len(out)
	start := min(f.Offset, len(out))
	end := min(start+f.Limit, len(out))
	page.Companies = out[start:end]
	if end < len(out) {
		page.NextOffset = &end
	}
	return page, nil
}

// mostCommon returns the most frequent key, breaking ties alphabetically.
func mostCommon(counts map[string]int) string {
	best, bestN := "", 0
	for k, n := range counts {
		if n > bestN || (n == bestN && k < best) {
			best, bestN = k, n
		}
	}
	return best
}

// AddedCompany reports what AddCompanies did with one requested company.
type AddedCompany struct {
	Title    string `json:"title"`
	Slug     string `json:"slug"`
	Revision string `json:"revision,omitempty"`
	Created  bool   `json:"created"`
	// Linked counts unlinked connections now attached to the new company.
	Linked int `json:"linkedConnections"`
}

// AddCompanies creates companies unless one with the same normalized title
// (case-insensitive, ignoring legal suffixes) already exists, then links
// unlinked connections whose employer matches a new company. It never
// changes existing companies or connection contact dates. Invalid input
// saves nothing.
func (s *Store) AddCompanies(ctx context.Context, entries []Entity) ([]AddedCompany, error) {
	prepared := make([]Entity, len(entries))
	for i, e := range entries {
		p, err := PrepareSave("company", e)
		if err != nil {
			return nil, fmt.Errorf("%w: company %d: %v", ErrInvalid, i+1, err)
		}
		prepared[i] = p
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Block concurrent company inserts so duplicate checks stay true until commit.
	if _, err = tx.ExecContext(ctx, "LOCK TABLE companies IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return nil, err
	}
	existing, err := trackedCompanies(ctx, tx)
	if err != nil {
		return nil, err
	}
	slugByKey := map[string]string{}
	for _, c := range existing {
		slugByKey[companyKey(c.Title)] = c.Slug
	}
	results := make([]AddedCompany, len(prepared))
	created := []linkedin.Company{}
	index := map[string]int{}
	for i, e := range prepared {
		title, _ := e["title"].(string)
		key := companyKey(title)
		if slug, ok := slugByKey[key]; ok {
			results[i] = AddedCompany{Title: title, Slug: slug}
			continue
		}
		saved, err := saveTx(ctx, tx, "company", e, nil, false)
		if err != nil {
			return nil, dbError(err)
		}
		slug, _ := saved.Entry["slug"].(string)
		slugByKey[key] = slug
		index[slug] = i
		created = append(created, linkedin.Company{Slug: slug, Title: title})
		results[i] = AddedCompany{Title: title, Slug: slug, Revision: saved.Revision, Created: true}
	}
	if len(created) == 0 {
		return results, nil
	}
	linked, err := linkNewCompanies(ctx, tx, created)
	if err != nil {
		return nil, err
	}
	for slug, n := range linked {
		results[index[slug]].Linked = n
	}
	if err = bump(ctx, tx, "company"); err != nil {
		return nil, err
	}
	return results, tx.Commit()
}

// linkNewCompanies attaches unlinked connections to newly created companies
// using the import matcher, so the companies board shows who works there.
func linkNewCompanies(ctx context.Context, tx *sql.Tx, companies []linkedin.Company) (map[string]int, error) {
	matcher := linkedin.NewMatcher(companies)
	rows, err := tx.QueryContext(ctx, "SELECT id::text,company_name FROM connections WHERE company_slug IS NULL AND company_name<>''")
	if err != nil {
		return nil, err
	}
	matches := map[string]string{}
	for rows.Next() {
		var id, company string
		if err = rows.Scan(&id, &company); err != nil {
			rows.Close()
			return nil, err
		}
		if slug := matcher.Match(company); slug != "" {
			matches[id] = slug
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	linked := map[string]int{}
	for id, slug := range matches {
		// Only the link and revision change; last_contacted_on stays as it was.
		r, err := tx.ExecContext(ctx, "UPDATE connections SET company_slug=$2,revision=$3,updated_at=now() WHERE id=$1 AND company_slug IS NULL", id, slug, uuid.NewString())
		if err != nil {
			return nil, err
		}
		if n, _ := r.RowsAffected(); n == 1 {
			linked[slug]++
		}
	}
	return linked, nil
}
