package database

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

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

// ValidCompanyName reports whether a name has letters or digits left after
// normalization, so it can be grouped and deduplicated.
func ValidCompanyName(name string) bool { return linkedin.NormalizeCompany(name) != "" }

// companyTracker decides whether an employer is already on the companies
// board. Listing and adding share it so a company listed as tracked is never
// added again: an employer whose connections are linked to a company maps to
// that company, otherwise the same matcher that links connections compares
// titles and slugs.
type companyTracker struct {
	companies []linkedin.Company
	titles    map[string]string
	linked    map[string]string // normalized employer name -> most linked slug
	matcher   *linkedin.Matcher
}

func loadTracker(ctx context.Context, q queryer) (*companyTracker, error) {
	t := &companyTracker{titles: map[string]string{}, linked: map[string]string{}}
	companies, err := trackedCompanies(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, c := range companies {
		t.add(c)
	}
	links, err := q.QueryContext(ctx, "SELECT company_name,company_slug,count(*) FROM connections WHERE company_slug IS NOT NULL GROUP BY 1,2")
	if err != nil {
		return nil, err
	}
	defer links.Close()
	counts := map[string]map[string]int{}
	for links.Next() {
		var name, slug string
		var n int
		if err = links.Scan(&name, &slug, &n); err != nil {
			return nil, err
		}
		key := linkedin.NormalizeCompany(name)
		if key == "" {
			continue
		}
		if counts[key] == nil {
			counts[key] = map[string]int{}
		}
		counts[key][slug] += n
	}
	for key, slugs := range counts {
		t.linked[key] = mostCommon(slugs)
	}
	return t, links.Err()
}

// Match returns the tracked slug for an employer name, or "".
func (t *companyTracker) Match(name string) string {
	if slug := t.linked[linkedin.NormalizeCompany(name)]; slug != "" {
		return slug
	}
	return t.matcher.Match(name)
}

// add tracks a company created in this transaction.
func (t *companyTracker) add(c linkedin.Company) {
	t.companies = append(t.companies, c)
	t.titles[c.Slug] = c.Title
	t.matcher = linkedin.NewMatcher(t.companies)
}

type companyGroup struct {
	ConnectionCompany
	spellings map[string]int
	people    []ConnectionSample
}

// ConnectionCompanies aggregates connections by employer, marking employers
// that already match a tracked company (by link or by the import matcher).
func (s *Store) ConnectionCompanies(ctx context.Context, f ConnectionCompanyFilter) (ConnectionCompanyPage, error) {
	page := ConnectionCompanyPage{Companies: []ConnectionCompany{}}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.People < 0 || f.People > 20 || !allowed(cmp.Or(f.Sort, "connections"), "connections|name|recent") {
		return page, ErrInvalid
	}
	tracker, err := loadTracker(ctx, s.DB)
	if err != nil {
		return page, err
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT id::text,name,role,company_name,to_char(last_contacted_on,'YYYY-MM-DD') FROM connections WHERE company_name<>''")
	if err != nil {
		return page, err
	}
	defer rows.Close()
	role := strings.ToLower(strings.TrimSpace(f.Role))
	groups := map[string]*companyGroup{}
	for rows.Next() {
		var p ConnectionSample
		var company string
		var last sql.NullString
		if err = rows.Scan(&p.ID, &p.Name, &p.Role, &company, &last); err != nil {
			return page, err
		}
		if role != "" && !strings.Contains(strings.ToLower(p.Role), role) {
			continue
		}
		p.LastContactedOn = last.String
		key := linkedin.NormalizeCompany(company)
		if key == "" {
			continue // punctuation-only employer names cannot be matched
		}
		g := groups[key]
		if g == nil {
			g = &companyGroup{spellings: map[string]int{}}
			groups[key] = g
		}
		g.Connections++
		g.spellings[company]++
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
		g.TrackedSlug = tracker.Match(g.Name)
		g.TrackedTitle = tracker.titles[g.TrackedSlug]
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
}

type AddedCompanies struct {
	Companies []AddedCompany
	// Linked counts unlinked connections now attached to the new companies.
	Linked int64
}

// AddCompanies creates companies unless list_connection_companies would call
// the name tracked (see companyTracker), then links unlinked connections whose
// employer best matches a new company (see linkUnlinkedConnections). Skipped
// companies report the stored title and slug. It never
// changes existing companies or connection contact dates. Invalid input
// saves nothing.
func (s *Store) AddCompanies(ctx context.Context, entries []Entity) (AddedCompanies, error) {
	prepared := make([]Entity, len(entries))
	for i, e := range entries {
		p, err := PrepareSave("company", e)
		if err == nil && !ValidCompanyName(fmt.Sprint(p["title"])) {
			err = fmt.Errorf("title needs letters or digits")
		}
		if err != nil {
			return AddedCompanies{}, fmt.Errorf("%w: company %d: %v", ErrInvalid, i+1, err)
		}
		prepared[i] = p
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return AddedCompanies{}, err
	}
	defer tx.Rollback()
	// Block concurrent company inserts so duplicate checks stay true until commit.
	if _, err = tx.ExecContext(ctx, "LOCK TABLE companies IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return AddedCompanies{}, err
	}
	tracker, err := loadTracker(ctx, tx)
	if err != nil {
		return AddedCompanies{}, err
	}
	results := make([]AddedCompany, len(prepared))
	created := []string{}
	for i, e := range prepared {
		title, _ := e["title"].(string)
		if slug := tracker.Match(title); slug != "" {
			results[i] = AddedCompany{Title: tracker.titles[slug], Slug: slug}
			continue
		}
		saved, err := saveTx(ctx, tx, "company", e, nil, false)
		if err != nil {
			return AddedCompanies{}, dbError(err)
		}
		slug, _ := saved.Entry["slug"].(string)
		created = append(created, slug)
		tracker.add(linkedin.Company{Slug: slug, Title: title})
		results[i] = AddedCompany{Title: title, Slug: slug, Revision: saved.Revision, Created: true}
	}
	if len(created) == 0 {
		return AddedCompanies{Companies: results}, nil
	}
	if err = bump(ctx, tx, "company"); err != nil {
		return AddedCompanies{}, err
	}
	linked, err := linkUnlinkedConnections(ctx, tx, created)
	if err != nil {
		return AddedCompanies{}, err
	}
	if linked > 0 {
		if err = bump(ctx, tx, "connection"); err != nil {
			return AddedCompanies{}, err
		}
	}
	return AddedCompanies{Companies: results, Linked: linked}, tx.Commit()
}
