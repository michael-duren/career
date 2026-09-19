package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"time"
)

var ErrConflict = errors.New("content changed; reload before saving; your draft is kept")
var ErrNotFound = errors.New("entry not found")
var ErrInvalid = errors.New("invalid query")
var ErrCore = errors.New("edit this core page instead of deleting it")

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type Result struct {
	Entry    Entity `json:"entry"`
	Revision string `json:"revision"`
}
type Page struct {
	Entries    []Result `json:"entries"`
	NextOffset *int     `json:"nextOffset"`
}
type Filter struct {
	Limit, Offset                               int
	Topic, Status, Category, Priority, From, To string
}

func projection(m model, detail bool) string {
	var pairs []string
	for _, f := range m.Fields {
		if !detail && f.Name == "body" {
			continue
		}
		col := f.Column
		if f.Type == "date" || f.Type == "undated" {
			col = "to_char(" + col + ", 'YYYY-MM-DD')"
		}
		if f.Type == "timestamp" {
			col = "to_char(" + col + " AT TIME ZONE 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')"
		}
		pairs = append(pairs, "'"+f.Name+"',"+col)
	}
	if m.Table == "books" {
		pairs = append(pairs, "'progress', CASE WHEN progress_unit IS NULL THEN NULL ELSE json_build_object('unit',progress_unit,'total',progress_total,'completed',progress_completed) END")
	}
	if m.Table == "companies" {
		pairs = append(pairs, "'_contactsPresent',contacts_present")
	}
	if !detail && m.Table == "books" {
		pairs = append(pairs, "'summary',(SELECT json_build_object('chapterCount',chapter_count,'completedCount',completed_count,'excerpt',excerpt,'sourceRevision',source_revision,'parserVersion',parser_version) FROM book_summaries WHERE book_slug=books.slug)")
	}
	if !detail && m.Table == "companies" {
		pairs = append(pairs, "'summary',(SELECT json_build_object('why',why,'stepCount',step_count,'completedCount',completed_count,'sourceRevision',source_revision,'parserVersion',parser_version) FROM company_summaries WHERE company_slug=companies.slug)")
	}

	if m.Table == "journal_weeks" {
		pairs = append(pairs, "'_targetsPresent',targets_present")
	}
	return "json_build_object(" + strings.Join(pairs, ",") + ")"
}
func scanResult(raw []byte, rev string) (Result, error) {
	e, err := Decode(raw)
	if err != nil {
		return Result{}, err
	}
	for k, v := range e {
		if v == nil {
			if k == "date" {
				e[k] = ""
			} else {
				delete(e, k)
			}
		}
	}
	return Result{e, rev}, nil
}
func readOne(ctx context.Context, q queryer, kind, id string) (Result, error) {
	m, ok := models[kind]
	if !ok {
		return Result{}, ErrNotFound
	}
	if !ValidID(kind, id) {
		return Result{}, ErrInvalid
	}
	var raw []byte
	var rev string
	err := q.QueryRowContext(ctx, "SELECT "+projection(m, true)+",revision FROM "+m.Table+" WHERE "+m.Key+"=$1", id).Scan(&raw, &rev)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, err
	}
	r, err := scanResult(raw, rev)
	if err != nil {
		return r, err
	}
	err = readChildren(ctx, q, kind, r.Entry)
	if err != nil {
		return r, err
	}
	rows, err := q.QueryContext(ctx, "SELECT field,value FROM source_timestamps WHERE kind=$1 AND entity_id=$2", kind, id)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var field, value string
		if err = rows.Scan(&field, &value); err != nil {
			return r, err
		}
		parts := strings.Split(field, ".")
		if len(parts) == 3 && parts[0] == "notes" {
			for _, v := range r.Entry["notes"].([]any) {
				n := v.(map[string]any)
				if n["id"] == parts[1] {
					n["createdAt"] = value
				}
			}
		} else {
			r.Entry[field] = value
		}
	}
	return r, rows.Err()
}
func (s *Store) Detail(ctx context.Context, kind, id string) (Result, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()
	r, err := readOne(ctx, tx, kind, id)
	if err != nil {
		return r, err
	}
	return r, tx.Commit()
}
func (s *Store) List(ctx context.Context, kind string, f Filter) (Page, error) {
	m, ok := models[kind]
	if !ok {
		return Page{}, ErrNotFound
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 100 || f.Offset < 0 || f.Offset > 1000000 {
		return Page{}, fmt.Errorf("%w: invalid pagination", ErrInvalid)
	}
	where := []string{"TRUE"}
	args := []any{}
	add := func(col, op string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf("%s%s$%d", col, op, len(args)))
	}
	for _, x := range []struct{ col, v string }{{"topic", f.Topic}, {"status", f.Status}, {"category", f.Category}, {"priority", f.Priority}} {
		if x.v != "" {
			found := false
			for _, field := range m.Fields {
				if field.Column == x.col {
					found = true
				}
			}
			if !found {
				return Page{}, fmt.Errorf("%w: unsupported filter %s", ErrInvalid, x.col)
			}
			add(x.col, "=", x.v)
		}
	}
	if f.From != "" || f.To != "" {
		if kind != "goal" {
			return Page{}, fmt.Errorf("%w: date filters require goals", ErrInvalid)
		}
		if f.From != "" {
			if !validDate(f.From) {
				return Page{}, fmt.Errorf("%w: invalid from date", ErrInvalid)
			}
			add("end_date", ">=", f.From)
		}
		if f.To != "" {
			if !validDate(f.To) {
				return Page{}, fmt.Errorf("%w: invalid to date", ErrInvalid)
			}
			add("start_date", "<=", f.To)
		}
	}
	order := map[string]string{"run": "started_at DESC,id", "note": "topic,title,id", "document": "title,id", "personal": "entry_date DESC NULLS LAST,id", "week": "start_date DESC,slug", "book": "status,category,priority,slug", "company": "status,category,priority,slug", "goal": "start_date,end_date,id"}[kind]
	args = append(args, f.Limit+1, f.Offset)
	rows, err := s.DB.QueryContext(ctx, "SELECT "+projection(m, false)+",revision FROM "+m.Table+" WHERE "+strings.Join(where, " AND ")+" ORDER BY "+order+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	p := Page{Entries: []Result{}}
	for rows.Next() {
		var raw []byte
		var rev string
		if err := rows.Scan(&raw, &rev); err != nil {
			return p, err
		}
		r, err := scanResult(raw, rev)
		if err != nil {
			return p, err
		}
		delete(r.Entry, "_contactsPresent")
		delete(r.Entry, "_targetsPresent")
		p.Entries = append(p.Entries, r)
	}
	if err = rows.Err(); err != nil {
		return p, err
	}
	if len(p.Entries) > f.Limit {
		p.Entries = p.Entries[:f.Limit]
		n := f.Offset + f.Limit
		p.NextOffset = &n
	}
	return p, nil
}

// ListDetails returns a bounded page with bodies and children for views, such as
// the company board, that operate on every visible entity. The browser still
// makes one scoped request rather than one request per card.
func (s *Store) ListDetails(ctx context.Context, kind string, f Filter) (Page, error) {
	page, err := s.List(ctx, kind, f)
	if err != nil {
		return Page{}, err
	}
	for index, item := range page.Entries {
		id, _ := item.Entry[models[kind].Key].(string)
		detail, detailErr := s.Detail(ctx, kind, id)
		if detailErr != nil {
			return Page{}, detailErr
		}
		page.Entries[index] = detail
	}
	return page, nil
}
func dbError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrConflict
	}
	return err
}
func (s *Store) Save(ctx context.Context, kind string, e Entity, revision *string) (Result, error) {
	var validationErr error
	e, validationErr = PrepareSave(kind, e)
	if validationErr != nil {
		err := validationErr
		return Result{}, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()
	r, err := saveTx(ctx, tx, kind, e, revision, false)
	if err != nil {
		return r, dbError(err)
	}
	if err = bump(ctx, tx, kind); err != nil {
		return r, err
	}
	return r, tx.Commit()
}
func bump(ctx context.Context, tx *sql.Tx, kind string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO workspace_metadata(id,version,catalog_version,journals_version,personal_journal_present,goals_present,change_sequence) VALUES(1,2,1,2,$1,$2,1) ON CONFLICT(id) DO UPDATE SET change_sequence=workspace_metadata.change_sequence+1,updated_at=now(),personal_journal_present=workspace_metadata.personal_journal_present OR $1,goals_present=workspace_metadata.goals_present OR $2`, kind == "personal", kind == "goal")
	return err
}
func saveTx(ctx context.Context, tx *sql.Tx, kind string, e Entity, revision *string, importing bool) (Result, error) {
	m := models[kind]
	id := e[m.Key].(string)
	rev := uuid.NewString()
	cols := []string{}
	args := []any{}
	vals := []string{}
	add := func(col string, v any, cast string) {
		cols = append(cols, col)
		args = append(args, v)
		vals = append(vals, fmt.Sprintf("$%d%s", len(args), cast))
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, f := range m.Fields {
		v := e[f.Name]
		if !importing && f.Name == "updatedAt" {
			v = now
		}
		if !importing && kind == "goal" && f.Name == "createdAt" {
			v = now
		}
		cast := ""
		if f.Type == "array" {
			b, _ := json.Marshal(v)
			v = string(b)
			cast = "::jsonb"
			cols = append(cols, f.Column)
			args = append(args, v)
			vals = append(vals, fmt.Sprintf("ARRAY(SELECT jsonb_array_elements_text($%d::jsonb))", len(args)))
			continue
		}
		if f.Type == "undated" && v == "" {
			v = nil
		}
		add(f.Column, v, cast)
	}
	if kind == "document" {
		add("is_core", id == "index" || id == "2026/career-study-plan", "")
	}
	if kind == "week" {
		p := strings.Split(e["dates"].(string), " to ")
		add("start_date", p[0], "")
		add("end_date", p[1], "")
		_, present := e["targets"]
		add("targets_present", present, "")
	}
	if kind == "company" {
		_, present := e["contacts"]
		add("contacts_present", present, "")
	}
	if kind == "book" {
		p, _ := e["progress"].(map[string]any)
		add("progress_unit", p["unit"], "")
		add("progress_total", p["total"], "")
		add("progress_completed", p["completed"], "")
	}
	add("revision", rev, "")
	var result sql.Result
	var err error
	if revision == nil {
		cols = append(cols, "position")
		vals = append(vals, "nextval('entity_position')")
		result, err = tx.ExecContext(ctx, "INSERT INTO "+m.Table+" ("+strings.Join(cols, ",")+") VALUES ("+strings.Join(vals, ",")+")", args...)
	} else {
		sets := []string{"position=nextval('entity_position')"}
		for i, c := range cols {
			if kind == "goal" && c == "created_at" {
				sets = append(sets, c+"=COALESCE(created_at,"+vals[i]+"::timestamptz)")
				continue
			}
			sets = append(sets, c+"="+vals[i])
		}
		args = append(args, id, *revision)
		result, err = tx.ExecContext(ctx, "UPDATE "+m.Table+" SET "+strings.Join(sets, ",")+fmt.Sprintf(" WHERE %s=$%d AND revision=$%d", m.Key, len(args)-1, len(args)), args...)
	}
	if err != nil {
		return Result{}, err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return Result{}, ErrConflict
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM source_timestamps WHERE kind=$1 AND entity_id=$2", kind, id); err != nil {
		return Result{}, err
	}
	if importing {
		for _, f := range m.Fields {
			if f.Type == "timestamp" && e[f.Name] != nil {
				if _, err = tx.ExecContext(ctx, "INSERT INTO source_timestamps(kind,entity_id,field,value) VALUES($1,$2,$3,$4)", kind, id, f.Name, e[f.Name]); err != nil {
					return Result{}, err
				}
			}
		}
	}
	if importing && kind == "goal" {
		for _, v := range e["notes"].([]any) {
			n := v.(map[string]any)
			if _, err = tx.ExecContext(ctx, "INSERT INTO source_timestamps(kind,entity_id,field,value) VALUES($1,$2,$3,$4)", kind, id, "notes."+n["id"].(string)+".createdAt", n["createdAt"]); err != nil {
				return Result{}, err
			}
		}
	}
	if err = writeChildren(ctx, tx, kind, id, e); err != nil {
		return Result{}, err
	}
	if kind == "book" || kind == "company" {
		if err = project(ctx, tx, kind, id, e["body"].(string), rev); err != nil {
			return Result{}, err
		}
	}
	return readOne(ctx, tx, kind, id)
}
func (s *Store) Delete(ctx context.Context, kind, id string, revision *string) error {
	m, ok := models[kind]
	if !ok {
		return ErrNotFound
	}
	if kind == "document" && (id == "index" || id == "2026/career-study-plan") {
		return ErrCore
	}
	if revision == nil {
		return ErrConflict
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, "DELETE FROM "+m.Table+" WHERE "+m.Key+"=$1 AND revision=$2", id, *revision)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM source_timestamps WHERE kind=$1 AND entity_id=$2", kind, id); err != nil {
		return err
	}
	if err = bump(ctx, tx, kind); err != nil {
		return err
	}
	return tx.Commit()
}
