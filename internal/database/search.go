package database

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/michael-duren/career-strategy/internal/scheduler"
	"github.com/michael-duren/career-strategy/internal/timefmt"
)

type SearchResult struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Search performs one bounded database query. It never exports or materializes the workspace.
func (s *Store) Search(ctx context.Context, query string, limit int, kinds ...string) ([]SearchResult, error) {
	kind := ""
	if len(kinds) > 0 {
		kind = kinds[0]
	}
	return s.SearchExcluding(ctx, query, limit, kind, "")
}

// SearchExcluding is Search limited to kind (when set) and never returning
// the excluded kinds.
func (s *Store) SearchExcluding(ctx context.Context, query string, limit int, kind string, exclude ...string) ([]SearchResult, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	excluded := []string{}
	for _, k := range append([]string{kind}, exclude...) {
		if _, ok := models[k]; k != "" && !ok {
			return nil, ErrInvalid
		}
	}
	for _, k := range exclude {
		if k != "" {
			excluded = append(excluded, k)
		}
	}
	excludedJSON, _ := json.Marshal(excluded)
	rows, err := s.DB.QueryContext(ctx, `
		SELECT kind,id,title,body FROM (
			SELECT 'note' kind,id,title,body,updated_at FROM notes
			UNION ALL SELECT 'run',id,title,body,updated_at FROM running_notes
			UNION ALL SELECT 'document',id,title,body,updated_at FROM documents
			UNION ALL SELECT 'personal',id,title,body,updated_at FROM personal_journal_entries
			UNION ALL SELECT 'week',slug,'Week ' || week || ' · ' || dates::text,body,updated_at FROM journal_weeks
			UNION ALL SELECT 'book',slug,title,body,updated_at FROM books
			UNION ALL SELECT 'company',slug,title,body,updated_at FROM companies
			UNION ALL SELECT 'connection',id::text,name,concat_ws(E'\n',role,company_name,notes),updated_at FROM connections
		) entries
		WHERE ($3='' OR kind=$3) AND NOT ($4::jsonb ? kind) AND (title ILIKE '%' || $1 || '%' OR body ILIKE '%' || $1 || '%')
		ORDER BY updated_at DESC NULLS LAST,kind,id LIMIT $2`, query, limit, kind, string(excludedJSON))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SearchResult{}
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.Kind, &item.ID, &item.Title, &item.Body); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// SiteResult is one match for the app's global search: an entry, a goal, the
// next upcoming session of a scheduled title, or a Leetgrinder problem.
// Context is a short line of metadata (status, dates, author, role); it may be empty.
type SiteResult struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Body    string `json:"-"`
	Context string `json:"context"`
	// TitleMatch says the query is in the title, so Context describes the match better than an excerpt.
	TitleMatch bool `json:"titleMatch"`
	// Type is a book's item type (book, paper, course) so search can draw its icon; empty for other kinds.
	Type string `json:"type,omitempty"`
}

// SiteSearchKinds are the kinds SiteSearch can return besides the entry
// kinds: upcoming schedule sessions and Leetgrinder problems.
var SiteSearchKinds = []string{"session", "problem"}

// likeEscaper makes the query literal in ILIKE ... ESCAPE '\'.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// SiteSearch is the global search behind GET /api/search. Unlike
// SearchExcluding it also finds goals (title, steps and notes), the next
// upcoming session of each scheduled title (see upcomingSessions) and
// Leetgrinder problems, treats the query literally, returns at most perKind
// matches of each kind (title matches first, then newest; sessions soonest
// first) and counts every match per kind in totals.
func (s *Store) SiteSearch(ctx context.Context, query string, perKind int, kind string, now time.Time, exclude ...string) ([]SiteResult, map[string]int, error) {
	if perKind < 1 || perKind > 50 {
		return nil, nil, ErrInvalid
	}
	excluded := []string{}
	for i, k := range append([]string{kind}, exclude...) {
		if _, ok := models[k]; k != "" && !ok && !slices.Contains(SiteSearchKinds, k) {
			return nil, nil, ErrInvalid
		}
		if i > 0 && k != "" {
			excluded = append(excluded, k)
		}
	}
	out, totals := []SiteResult{}, map[string]int{}
	if kind != "session" {
		rows, err := s.DB.QueryContext(ctx, `
			WITH entries AS (
				SELECT 'note' kind,id,title,body,topic || ' · ' || to_char(cardinality(regexp_split_to_array(btrim(body), '\s+')) - (btrim(body) = '')::int, 'FM999,999,990') || ' words' context,updated_at sort_at,'' item_type FROM notes
				UNION ALL SELECT 'run',id,title,body,to_char(run_date,'Mon FMDD, YYYY'),updated_at,'' FROM running_notes
				UNION ALL SELECT 'document',id,title,body,'',updated_at,'' FROM documents
				UNION ALL SELECT 'personal',id,title,body,COALESCE(to_char(entry_date,'Mon FMDD, YYYY'),''),updated_at,'' FROM personal_journal_entries
				UNION ALL SELECT 'week',slug,'Week ' || week || ' · ' || to_char(start_date,'Mon FMDD'),body,'',updated_at,'' FROM journal_weeks
				UNION ALL SELECT 'book',slug,title,body,concat_ws(' · ',upper(left(status,1)) || substr(status,2),NULLIF(array_to_string(authors,', '),'')),updated_at,type FROM books
				UNION ALL SELECT 'company',slug,title,body,upper(left(replace(status,'_',' '),1)) || substr(replace(status,'_',' '),2),updated_at,'' FROM companies
				UNION ALL SELECT 'connection',id::text,name,concat_ws(E'\n',role,company_name,notes),concat_ws(' · ',NULLIF(role,''),NULLIF(company_name,'')),updated_at,'' FROM connections
				UNION ALL SELECT 'goal',g.id::text,g.title,
					concat_ws(E'\n',(SELECT string_agg(title,E'\n' ORDER BY position) FROM goal_steps WHERE goal_id=g.id),(SELECT string_agg(body,E'\n' ORDER BY position) FROM goal_notes WHERE goal_id=g.id)),
					upper(left(g.status,1)) || substr(g.status,2) || ' · ' || to_char(g.start_date,'Mon FMDD') || ' → ' || to_char(g.end_date,'Mon FMDD'),g.updated_at,'' FROM goals g
				UNION ALL SELECT 'problem',slug,title,'',concat_ws(' · ','#' || number,NULLIF(difficulty,'')),created_at,'' FROM leetgrinder_problems WHERE title <> ''
			), matches AS (
				SELECT *, title ILIKE '%' || $1 || '%' ESCAPE '\' title_match FROM entries
				WHERE ($3='' OR kind=$3) AND NOT (kind = ANY($4::text[])) AND (title ILIKE '%' || $1 || '%' ESCAPE '\' OR body ILIKE '%' || $1 || '%' ESCAPE '\')
			), ranked AS (
				SELECT *, row_number() OVER (PARTITION BY kind ORDER BY title_match DESC, sort_at DESC NULLS LAST, id) n, count(*) OVER (PARTITION BY kind) total FROM matches
			)
			SELECT kind,id,title,body,COALESCE(context,''),title_match,item_type,total FROM ranked WHERE n <= $2 ORDER BY kind,n`, likeEscaper.Replace(query), perKind, kind, excluded)
		if err != nil {
			return nil, nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var item SiteResult
			var total int
			if err := rows.Scan(&item.Kind, &item.ID, &item.Title, &item.Body, &item.Context, &item.TitleMatch, &item.Type, &total); err != nil {
				return nil, nil, err
			}
			totals[item.Kind] = total
			out = append(out, item)
		}
		if err := rows.Err(); err != nil {
			return nil, nil, err
		}
	}
	if (kind == "" || kind == "session") && !slices.Contains(excluded, "session") {
		doc, err := schedulerLoad(ctx, s.DB)
		if err != nil {
			// A schedule that cannot be read only drops sessions from the results.
			log.Printf("search: schedule unreadable, sessions left out: %v", err)
		} else if sessions := upcomingSessions(doc, query, now); len(sessions) > 0 {
			totals["session"] = len(sessions)
			out = append(out, sessions[:min(perKind, len(sessions))]...)
		}
	}
	return out, totals, nil
}

// upcomingSessions returns, soonest first, the next planned session of each
// title containing query (ignoring case) that has not ended by now and is not
// canceled, with "Today"/"Tomorrow"/"Mon, Jan 2" and a 12-hour time range in
// the scheduler's time zone (UTC when it is unset or unknown) as context.
func upcomingSessions(doc scheduler.Document, query string, now time.Time) []SiteResult {
	zone, err := time.LoadLocation(doc.Settings.TimeZone)
	if err != nil || doc.Settings.TimeZone == "" {
		zone = time.UTC
	}
	needle := strings.ToLower(query)
	next := map[string]scheduler.Session{}
	for _, session := range doc.Sessions {
		title := session.Assignment.Title
		if session.Plan == nil || session.State == "canceled" || title == "" || !session.Plan.End.After(now) || !strings.Contains(strings.ToLower(title), needle) {
			continue
		}
		if prev, ok := next[title]; !ok || session.Plan.Start.Before(prev.Plan.Start) || (session.Plan.Start.Equal(prev.Plan.Start) && session.ID < prev.ID) {
			next[title] = session
		}
	}
	sessions := slices.Collect(maps.Values(next))
	slices.SortFunc(sessions, func(a, b scheduler.Session) int {
		if c := a.Plan.Start.Compare(b.Plan.Start); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	today := now.In(zone).Format("2006-01-02")
	tomorrow := now.In(zone).AddDate(0, 0, 1).Format("2006-01-02")
	out := []SiteResult{}
	for _, session := range sessions {
		start, end := session.Plan.Start.In(zone), session.Plan.End.In(zone)
		day := start.Format("Mon, Jan 2")
		switch start.Format("2006-01-02") {
		case today:
			day = "Today"
		case tomorrow:
			day = "Tomorrow"
		}
		out = append(out, SiteResult{Kind: "session", ID: session.ID, Title: session.Assignment.Title, Context: day + " · " + timefmt.Range(start, end), TitleMatch: true})
	}
	return out
}
