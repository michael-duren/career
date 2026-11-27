package database

import (
	"context"
	"encoding/json"
	"slices"
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
// next upcoming session of a scheduled goal, or a Leetgrinder problem.
// Context is a short line of metadata (status, dates, author, role); it may be empty.
type SiteResult struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Body    string `json:"-"`
	Context string `json:"context"`
	// TitleMatch says the query is in the title, so Context describes the match better than an excerpt.
	TitleMatch bool `json:"titleMatch"`
}

// SiteSearchKinds are the kinds SiteSearch can return besides the entry
// kinds: upcoming schedule sessions and Leetgrinder problems.
var SiteSearchKinds = []string{"session", "problem"}

// SiteSearch is the global search behind GET /api/search. Unlike
// SearchExcluding it also finds goals (title, steps and notes), the next
// upcoming session of each scheduled title and Leetgrinder problems, returns
// at most perKind matches of each kind (title matches first, then newest) and
// counts every match per kind in totals.
func (s *Store) SiteSearch(ctx context.Context, query string, perKind int, kind string, exclude ...string) ([]SiteResult, map[string]int, error) {
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
	excludedJSON, _ := json.Marshal(excluded)
	rows, err := s.DB.QueryContext(ctx, `
		WITH zone AS (
			-- The scheduler's time zone, for "today" and session times; UTC if unset or unknown here.
			SELECT COALESCE((SELECT name FROM pg_timezone_names WHERE name = (SELECT document->'settings'->>'timeZone' FROM scheduler_state LIMIT 1)), 'UTC') tz
		), today AS (
			SELECT (now() AT TIME ZONE tz)::date d, tz FROM zone
		), entries AS (
			SELECT 'note' kind,id,title,body,topic || ' · ' || to_char(cardinality(regexp_split_to_array(btrim(body), '\s+')) - (btrim(body) = '')::int, 'FM999,999,990') || ' words' context,updated_at sort_at FROM notes
			UNION ALL SELECT 'run',id,title,body,to_char(run_date,'Mon FMDD, YYYY'),updated_at FROM running_notes
			UNION ALL SELECT 'document',id,title,body,'',updated_at FROM documents
			UNION ALL SELECT 'personal',id,title,body,COALESCE(to_char(entry_date,'Mon FMDD, YYYY'),''),updated_at FROM personal_journal_entries
			UNION ALL SELECT 'week',slug,'Week ' || week || ' · ' || to_char(start_date,'Mon FMDD'),body,'',updated_at FROM journal_weeks
			UNION ALL SELECT 'book',slug,title,body,concat_ws(' · ',upper(left(status,1)) || substr(status,2),NULLIF(array_to_string(authors,', '),'')),updated_at FROM books
			UNION ALL SELECT 'company',slug,title,body,upper(left(replace(status,'_',' '),1)) || substr(replace(status,'_',' '),2),updated_at FROM companies
			UNION ALL SELECT 'connection',id::text,name,concat_ws(E'\n',role,company_name,notes),concat_ws(' · ',NULLIF(role,''),NULLIF(company_name,'')),updated_at FROM connections
			UNION ALL SELECT 'goal',g.id::text,g.title,
				concat_ws(E'\n',(SELECT string_agg(title,E'\n' ORDER BY position) FROM goal_steps WHERE goal_id=g.id),(SELECT string_agg(body,E'\n' ORDER BY position) FROM goal_notes WHERE goal_id=g.id)),
				upper(left(g.status,1)) || substr(g.status,2) || ' · ' || to_char(g.start_date,'Mon FMDD') || ' → ' || to_char(g.end_date,'Mon FMDD'),g.updated_at FROM goals g
			UNION ALL (SELECT DISTINCT ON (s.value->'assignment'->>'title') 'session',s.key,s.value->'assignment'->>'title','',
				CASE (s.value->>'date')::date WHEN t.d THEN 'Today' WHEN t.d + 1 THEN 'Tomorrow' ELSE to_char((s.value->>'date')::date,'Dy, Mon FMDD') END
					|| ' · ' || to_char((s.value->'plan'->>'start')::timestamptz AT TIME ZONE t.tz,'FMHH12:MI AM') || ' – ' || to_char((s.value->'plan'->>'end')::timestamptz AT TIME ZONE t.tz,'FMHH12:MI AM'),
				-- The soonest session sorts first among the newest-first others.
				now() - ((s.value->'plan'->>'start')::timestamptz - now())
				FROM scheduler_state, jsonb_each(document->'sessions') s, today t
				WHERE s.value->>'state' <> 'canceled' AND (s.value->>'date')::date >= t.d AND COALESCE(s.value->'assignment'->>'title','') <> ''
				ORDER BY s.value->'assignment'->>'title', (s.value->'plan'->>'start')::timestamptz)
			UNION ALL SELECT 'problem',slug,title,'',concat_ws(' · ','#' || number,NULLIF(difficulty,'')),created_at FROM leetgrinder_problems WHERE title <> ''
		), matches AS (
			SELECT *, title ILIKE '%' || $1 || '%' title_match FROM entries
			WHERE ($3='' OR kind=$3) AND NOT ($4::jsonb ? kind) AND (title ILIKE '%' || $1 || '%' OR body ILIKE '%' || $1 || '%')
		), ranked AS (
			SELECT *, row_number() OVER (PARTITION BY kind ORDER BY title_match DESC, sort_at DESC NULLS LAST, id) n, count(*) OVER (PARTITION BY kind) total FROM matches
		)
		SELECT kind,id,title,body,context,title_match,total FROM ranked WHERE n <= $2 ORDER BY kind,n`, query, perKind, kind, string(excludedJSON))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out, totals := []SiteResult{}, map[string]int{}
	for rows.Next() {
		var item SiteResult
		var total int
		if err := rows.Scan(&item.Kind, &item.ID, &item.Title, &item.Body, &item.Context, &item.TitleMatch, &total); err != nil {
			return nil, nil, err
		}
		totals[item.Kind] = total
		out = append(out, item)
	}
	return out, totals, rows.Err()
}
