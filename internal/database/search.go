package database

import "context"

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

// SearchExcluding is Search limited to kind (when set) and never returning exclude.
func (s *Store) SearchExcluding(ctx context.Context, query string, limit int, kind, exclude string) ([]SearchResult, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	for _, k := range []string{kind, exclude} {
		if _, ok := models[k]; k != "" && !ok {
			return nil, ErrInvalid
		}
	}
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
		WHERE ($3='' OR kind=$3) AND ($4='' OR kind<>$4) AND (title ILIKE '%' || $1 || '%' OR body ILIKE '%' || $1 || '%')
		ORDER BY updated_at DESC NULLS LAST,kind,id LIMIT $2`, query, limit, kind, exclude)
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
