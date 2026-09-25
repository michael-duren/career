package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"strings"
)

type Source struct{ Store, Key, Revision, Checksum, Archive string }

// Import streams one entity at a time. The caller enforces a separate archive byte limit.
// Dry runs exercise SQL constraints and projections and always roll back.
func (s *Store) Import(ctx context.Context, r io.Reader, source Source, dry bool) (map[string]int, error) {
	counts := map[string]int{}
	pending := map[string][]any{}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return counts, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(724193602)"); err != nil {
		return counts, err
	}
	if err = lockGoals(ctx, tx); err != nil {
		return counts, err
	}
	// Exclude writers during the empty-workspace check and all inserts.
	if _, err = tx.ExecContext(ctx, "LOCK TABLE workspace_metadata,notes,documents,personal_journal_entries,journal_weeks,books,companies,goals,running_notes,connections IN EXCLUSIVE MODE"); err != nil {
		return counts, err
	}
	for _, table := range []string{"workspace_metadata", "notes", "documents", "personal_journal_entries", "journal_weeks", "books", "companies", "goals", "running_notes", "connections"} {
		var n int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			return counts, err
		}
		if n != 0 {
			return counts, fmt.Errorf("import requires an empty workspace (%s is populated)", table)
		}
	}
	d := json.NewDecoder(r)
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return counts, fmt.Errorf("workspace object required")
	}
	seen := map[string]bool{}
	legacy := legacyContacts{urls: map[string]bool{}, ids: map[string]bool{}}
	markers := map[string]any{}
	byCollection := map[string]string{}
	for k, m := range models {
		byCollection[m.Collection] = k
	}
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return counts, err
		}
		key := token.(string)
		if seen[key] {
			return counts, fmt.Errorf("duplicate collection/marker %s", key)
		}
		seen[key] = true
		if kind, ok := byCollection[key]; ok {
			token, err = d.Token()
			if err != nil || token != json.Delim('[') {
				return counts, fmt.Errorf("%s must be an array", key)
			}
			for d.More() {
				var e Entity
				if err = d.Decode(&e); err != nil {
					return counts, err
				}
				goalDefaults(kind, e)

				if err = Validate(kind, e); err != nil {
					return counts, fmt.Errorf("%s[%d]: %w", key, counts[key], err)
				}
				if kind == "goal" {
					pending[e["id"].(string)] = e["dependsOn"].([]any)
				}
				contacts, _ := e["contacts"].([]any)
				delete(e, "contacts")
				if _, err = saveTx(ctx, tx, kind, e, nil, true); err != nil {
					return counts, fmt.Errorf("%s[%d]: %w", key, counts[key], err)
				}
				for _, v := range contacts {
					if err = legacy.save(ctx, tx, e, v.(map[string]any)); err != nil {
						return counts, fmt.Errorf("%s[%d] contact: %w", key, counts[key], err)
					}
					counts["connections"]++
				}
				counts[key]++
			}
			if _, err = d.Token(); err != nil {
				return counts, err
			}
		} else if allowed(key, "version|catalogVersion|journalsVersion") {
			var v any
			if err = d.Decode(&v); err != nil {
				return counts, err
			}
			expected := map[string]float64{"version": 2, "catalogVersion": 1, "journalsVersion": 2}[key]
			if v != expected {
				return counts, fmt.Errorf("unsupported %s: normalize an archived copy using legacy migration rules", key)
			}
			markers[key] = v
		} else {
			return counts, fmt.Errorf("unsupported workspace field %s", key)
		}
	}
	if _, err = d.Token(); err != nil {
		return counts, err
	}
	var trailing any
	if err = d.Decode(&trailing); err != io.EOF {
		return counts, fmt.Errorf("unexpected trailing JSON")
	}
	for _, key := range []string{"version", "notes", "weeks", "books", "companies", "documents"} {
		if !seen[key] {
			return counts, fmt.Errorf("missing %s", key)
		}
	}
	for id, deps := range pending {
		if err = writeDependencies(ctx, tx, id, deps); err != nil {
			return counts, err
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO workspace_metadata(id,version,catalog_version,journals_version,personal_journal_present,goals_present,change_sequence) VALUES(1,2,$1,$2,$3,$4,1)", markers["catalogVersion"], markers["journalsVersion"], seen["personalJournal"], seen["goals"]); err != nil {
		return counts, err
	}
	var revision any
	if source.Revision != "" {
		revision = source.Revision
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO migration_imports(id,source_store,source_key,source_revision,source_checksum,archive_path,version,catalog_version,journals_version) VALUES($1,$2,$3,$4,$5,$6,2,$7,$8)", uuid.NewString(), source.Store, source.Key, revision, source.Checksum, source.Archive, markers["catalogVersion"], markers["journalsVersion"]); err != nil {
		return counts, err
	}
	if dry {
		return counts, nil
	}
	return counts, tx.Commit()
}
func (s *Store) Export(ctx context.Context, w io.Writer) error {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	var catalog, journals sql.NullInt64
	var personal, goals bool
	var sequence int64
	err = tx.QueryRowContext(ctx, "SELECT version,catalog_version,journals_version,personal_journal_present,goals_present,change_sequence FROM workspace_metadata WHERE id=1").Scan(&version, &catalog, &journals, &personal, &goals, &sequence)
	if err != nil {
		return err
	}
	write := func(s string) error { _, e := io.WriteString(w, s); return e }
	if err = write(fmt.Sprintf(`{"version":%d`, version)); err != nil {
		return err
	}
	if catalog.Valid {
		if err = write(fmt.Sprintf(`,"catalogVersion":%d`, catalog.Int64)); err != nil {
			return err
		}
	}
	if journals.Valid {
		if err = write(fmt.Sprintf(`,"journalsVersion":%d`, journals.Int64)); err != nil {
			return err
		}
	}
	for _, kind := range []string{"note", "week", "book", "company", "document", "personal", "goal", "run", "connection"} {
		if kind == "personal" && !personal || kind == "goal" && !goals {
			continue
		}
		if kind == "run" || kind == "connection" {
			var count int
			if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM "+models[kind].Table).Scan(&count); err != nil {
				return err
			}
			if count == 0 {
				continue
			}
		}
		m := models[kind]
		if err = write(`,"` + m.Collection + `":[`); err != nil {
			return err
		}
		first := true
		offset := 0
		for {
			rows, err := tx.QueryContext(ctx, "SELECT "+m.Key+" FROM "+m.Table+" ORDER BY position,"+m.Key+" LIMIT 100 OFFSET $1", offset)
			if err != nil {
				return err
			}
			ids := []string{}
			for rows.Next() {
				var id string
				if err = rows.Scan(&id); err != nil {
					rows.Close()
					return err
				}
				ids = append(ids, id)
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				break
			}
			for _, id := range ids {
				r, err := readOne(ctx, tx, kind, id)
				if err != nil {
					return err
				}
				// Photo bytes are not exported, so a revision would dangle on import.
				delete(r.Entry, "photo")
				if !first {
					if err = write(","); err != nil {
						return err
					}
				}
				first = false
				if err = json.NewEncoder(w).Encode(r.Entry); err != nil {
					return err
				}
			}
			offset += len(ids)
		}
		if err = write("]"); err != nil {
			return err
		}
	}
	if err = write("}\n"); err != nil {
		return err
	}
	return tx.Commit()
}

// legacyContacts converts company contacts from old archives into linked
// connections. Contacts were already URL- and email-validated, so only the
// migration 004 duplicate repairs apply: a repeated profile URL stays on its
// first row, and a UUID reused across companies gets a new id.
type legacyContacts struct{ urls, ids map[string]bool }

func (l legacyContacts) save(ctx context.Context, tx *sql.Tx, company Entity, contact map[string]any) error {
	id, url := strings.ToLower(contact["id"].(string)), strings.ToLower(contact["url"].(string))
	e := Entity{"id": contact["id"], "name": contact["name"], "role": contact["role"], "companyName": company["title"], "companySlug": company["slug"], "email": contact["email"], "queued": false, "notes": contact["notes"], "tags": []any{}}
	if l.ids[id] {
		e["id"] = uuid.NewString()
	}
	l.ids[id] = true
	if url != "" && !l.urls[url] {
		l.urls[url] = true
		e["url"] = contact["url"]
	}
	if err := Validate("connection", e); err != nil {
		return err
	}
	_, err := saveTx(ctx, tx, "connection", e, nil, true)
	return err
}
