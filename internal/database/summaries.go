package database

import (
	"context"
	"encoding/json"
	"strings"
)

// ProgressSummary returns metric-bearing journal metadata without private bodies.
func (s *Store) ProgressSummary(ctx context.Context) ([]Entity, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT json_build_object(
		'slug', w.slug, 'week', w.week, 'year', w.year, 'dates', w.dates, 'tags', w.tags,
		'hours', COALESCE((SELECT json_object_agg(track, hours::double precision) FROM journal_week_metrics WHERE week_slug=w.slug AND hours IS NOT NULL), '{}'::json),
		'targets', CASE WHEN w.targets_present THEN COALESCE((SELECT json_object_agg(track, target::double precision) FROM journal_week_metrics WHERE week_slug=w.slug AND target IS NOT NULL), '{}'::json) ELSE NULL END
	) FROM journal_weeks w ORDER BY start_date, slug LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Entity{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var entry Entity
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, err
		}
		if entry["targets"] == nil {
			delete(entry, "targets")
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

// Counts returns the number of saved entries for every entry kind.
func (s *Store) Counts(ctx context.Context) (map[string]int, error) {
	var parts []string
	for kind, m := range models {
		parts = append(parts, "SELECT '"+kind+"',count(*) FROM "+m.Table)
	}
	rows, err := s.DB.QueryContext(ctx, strings.Join(parts, " UNION ALL "))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var kind string
		var n int
		if err := rows.Scan(&kind, &n); err != nil {
			return nil, err
		}
		counts[kind] = n
	}
	return counts, rows.Err()
}
