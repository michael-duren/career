package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func (s *Store) CreateLeetgrinderTodoSet(ctx context.Context, title string, slugs []string) (leetgrinder.TodoSet, error) {
	problems := make([]leetgrinder.TodoProblemInput, 0, len(slugs))
	for _, slug := range slugs {
		problems = append(problems, leetgrinder.TodoProblemInput{Slug: slug})
	}
	return s.CreateLeetgrinderTodoSetDetailed(ctx, leetgrinder.TodoSet{Title: title}, problems)
}

// CreateLeetgrinderTodoSetDetailed copies a source set and its problem metadata
// atomically. Catalog metadata remains after the todo set is deleted.
func (s *Store) CreateLeetgrinderTodoSetDetailed(ctx context.Context, set leetgrinder.TodoSet, problems []leetgrinder.TodoProblemInput) (leetgrinder.TodoSet, error) {
	set.Title = strings.TrimSpace(set.Title)
	set.Description = strings.TrimSpace(set.Description)
	if set.Title == "" || utf8.RuneCountInString(set.Title) > 120 || !utf8.ValidString(set.Title) || strings.IndexFunc(set.Title, unicode.IsControl) >= 0 ||
		!utf8.ValidString(set.Description) || strings.ContainsRune(set.Description, 0) || utf8.RuneCountInString(set.Description) > 2000 || len(problems) > 200 {
		return leetgrinder.TodoSet{}, ErrInvalid
	}
	setMetadata, err := todoMetadataJSON(set.Metadata)
	if err != nil {
		return leetgrinder.TodoSet{}, err
	}
	seen := map[string]bool{}
	for _, problem := range problems {
		if !leetgrinder.ValidSlug(problem.Slug) || seen[problem.Slug] || validateTodoProblem(problem) != nil {
			return leetgrinder.TodoSet{}, ErrInvalid
		}
		seen[problem.Slug] = true
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return leetgrinder.TodoSet{}, err
	}
	defer tx.Rollback()
	set.ID = uuid.NewString()
	if _, err = tx.ExecContext(ctx, "INSERT INTO leetgrinder_todo_sets(id,title,description,metadata) VALUES($1,$2,$3,$4::jsonb)", set.ID, set.Title, set.Description, setMetadata); err != nil {
		return leetgrinder.TodoSet{}, err
	}
	for _, problem := range problems {
		if err = saveTodoProblem(ctx, tx, problem); err != nil {
			return leetgrinder.TodoSet{}, err
		}
		itemID := uuid.NewString()
		source, _ := todoSourceJSON(problem)
		if _, err = tx.ExecContext(ctx, "INSERT INTO leetgrinder_todo_items(id,set_id,problem_slug,source_data) VALUES($1,$2,$3,$4::jsonb)", itemID, set.ID, problem.Slug, source); err != nil {
			return leetgrinder.TodoSet{}, err
		}
		if err = recordTodoSource(ctx, tx, problem.Slug, itemID, source); err != nil {
			return leetgrinder.TodoSet{}, err
		}
	}
	return set, tx.Commit()
}

func todoMetadataJSON(metadata map[string]any) ([]byte, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	data, err := json.Marshal(metadata)
	if err != nil || len(data) > 8192 {
		return nil, ErrInvalid
	}
	return data, nil
}

func validateTodoProblem(problem leetgrinder.TodoProblemInput) error {
	if !leetgrinder.ValidSlug(problem.Slug) {
		return ErrInvalid
	}
	topics := make([]leetgrinder.TopicTag, 0, len(problem.Topics))
	for _, slug := range problem.Topics {
		topics = append(topics, leetgrinder.TopicTag{Slug: slug})
	}
	meta := leetgrinder.ProblemMetadata{Number: problem.Number, Title: problem.Title, Difficulty: problem.Difficulty, Topics: topics}
	if meta.Normalize() != nil {
		return ErrInvalid
	}
	_, err := todoMetadataJSON(problem.ImportMetadata)
	if err != nil {
		return err
	}
	_, err = todoSourceJSON(problem)
	return err
}

func todoSourceJSON(problem leetgrinder.TodoProblemInput) ([]byte, error) {
	source := map[string]any{}
	if problem.Reference != "" && problem.Reference != problem.Slug {
		source["reference"] = problem.Reference
	}
	if problem.Number != 0 {
		source["number"] = problem.Number
	}
	if problem.Title != "" {
		source["title"] = problem.Title
	}
	if problem.Difficulty != "" {
		source["difficulty"] = problem.Difficulty
	}
	if len(problem.Topics) != 0 {
		source["topics"] = problem.Topics
	}
	if len(problem.ImportMetadata) != 0 {
		source["metadata"] = problem.ImportMetadata
	}
	data, err := json.Marshal(source)
	if err != nil || len(data) > 8192 {
		return nil, ErrInvalid
	}
	return data, nil
}

func saveTodoProblem(ctx context.Context, tx *sql.Tx, problem leetgrinder.TodoProblemInput) error {
	if err := validateTodoProblem(problem); err != nil {
		return err
	}
	problem.Title = strings.TrimSpace(problem.Title)
	if err := ensureLeetgrinderProblem(ctx, tx, problem.Slug); err != nil {
		return err
	}
	var number any
	if problem.Number > 0 {
		number = problem.Number
	}
	source := ""
	if number != nil || problem.Title != "" || problem.Difficulty != "" || len(problem.Topics) > 0 {
		source = "mcp"
	}
	_, err := tx.ExecContext(ctx, `UPDATE leetgrinder_problems SET
number=COALESCE(number,$2),
title=COALESCE(NULLIF(title,''),$3),
difficulty=COALESCE(NULLIF(difficulty,''),$4),
topics=CASE WHEN cardinality(topics)=0 AND cardinality($5::text[])>0 THEN $5::text[] ELSE topics END,
metadata_source=CASE WHEN metadata_source='' AND $6='mcp' THEN 'mcp' ELSE metadata_source END
WHERE slug=$1`, problem.Slug, number, problem.Title, problem.Difficulty, problem.Topics, source)
	return err
}

func recordTodoSource(ctx context.Context, tx *sql.Tx, slug, itemID string, source []byte) error {
	if string(source) == "{}" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE leetgrinder_problems SET import_metadata=jsonb_set(import_metadata,ARRAY[$2::text],
COALESCE(import_metadata->$2,'[]'::jsonb) || jsonb_build_array($3::jsonb),true)
WHERE slug=$1 AND NOT EXISTS (
  SELECT 1 FROM jsonb_array_elements(COALESCE(import_metadata->$2,'[]'::jsonb)) AS prior(value)
  WHERE prior.value=$3::jsonb
)`, slug, itemID, source)
	return err
}

func (s *Store) LeetgrinderTodoSets(ctx context.Context) ([]leetgrinder.TodoSet, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,title,description,metadata::text FROM leetgrinder_todo_sets ORDER BY created_at,id")
	if err != nil {
		return nil, err
	}
	sets := []leetgrinder.TodoSet{}
	for rows.Next() {
		var set leetgrinder.TodoSet
		var metadata string
		if err = rows.Scan(&set.ID, &set.Title, &set.Description, &metadata); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal([]byte(metadata), &set.Metadata); err != nil {
			rows.Close()
			return nil, err
		}
		set.Items = []leetgrinder.TodoItem{}
		sets = append(sets, set)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range sets {
		sets[i].Items, err = s.LeetgrinderTodoItems(ctx, sets[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return sets, nil
}

func (s *Store) LeetgrinderTodoItems(ctx context.Context, setID string) ([]leetgrinder.TodoItem, error) {
	if setID != "" {
		if _, err := uuid.Parse(setID); err != nil {
			return nil, ErrInvalid
		}
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id,COALESCE(i.set_id::text,''),p.slug,COALESCE(p.number,0),p.title,p.difficulty,array_to_json(p.topics)::text,i.source_data::text
FROM leetgrinder_todo_items i JOIN leetgrinder_problems p ON p.slug=i.problem_slug
WHERE i.set_id IS NOT DISTINCT FROM $1::uuid ORDER BY i.created_at,i.id`, nullableUUID(setID))
	if err != nil {
		return nil, err
	}
	return scanLeetgrinderTodoItems(rows)
}

// LeetgrinderNextTodoItems returns the oldest queued entry for each problem.
// A problem present in several sets appears only once on the dashboard.
func (s *Store) LeetgrinderNextTodoItems(ctx context.Context, limit int) ([]leetgrinder.TodoItem, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,set_id,slug,number,title,difficulty,topics,source_data
FROM (
  SELECT DISTINCT ON (p.slug) i.id,COALESCE(i.set_id::text,'') AS set_id,p.slug,
    COALESCE(p.number,0) AS number,p.title,p.difficulty,
    array_to_json(p.topics)::text AS topics,i.source_data::text AS source_data,
    i.created_at
  FROM leetgrinder_todo_items i JOIN leetgrinder_problems p ON p.slug=i.problem_slug
  ORDER BY p.slug,i.created_at,i.id
) next
ORDER BY created_at,id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return scanLeetgrinderTodoItems(rows)
}

func scanLeetgrinderTodoItems(rows *sql.Rows) ([]leetgrinder.TodoItem, error) {
	defer rows.Close()
	items := []leetgrinder.TodoItem{}
	for rows.Next() {
		var item leetgrinder.TodoItem
		var topics, metadata string
		if err := rows.Scan(&item.ID, &item.SetID, &item.Problem.Slug, &item.Problem.Number, &item.Problem.Title, &item.Problem.Difficulty, &topics, &metadata); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(topics), &item.Problem.Topics); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(metadata), &item.SourceData); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func nullableUUID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

func (s *Store) AddLeetgrinderTodoItem(ctx context.Context, setID, slug string) (leetgrinder.TodoItem, error) {
	return s.AddLeetgrinderTodoProblem(ctx, setID, leetgrinder.TodoProblemInput{Slug: slug})
}

func (s *Store) AddLeetgrinderTodoProblem(ctx context.Context, setID string, problem leetgrinder.TodoProblemInput) (leetgrinder.TodoItem, error) {
	if validateTodoProblem(problem) != nil {
		return leetgrinder.TodoItem{}, ErrInvalid
	}
	if setID != "" {
		if _, err := uuid.Parse(setID); err != nil {
			return leetgrinder.TodoItem{}, ErrInvalid
		}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return leetgrinder.TodoItem{}, err
	}
	defer tx.Rollback()
	if setID != "" {
		var exists bool
		if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM leetgrinder_todo_sets WHERE id=$1)", setID).Scan(&exists); err != nil {
			return leetgrinder.TodoItem{}, err
		}
		if !exists {
			return leetgrinder.TodoItem{}, ErrNotFound
		}
	}
	if err = saveTodoProblem(ctx, tx, problem); err != nil {
		return leetgrinder.TodoItem{}, err
	}
	source, _ := todoSourceJSON(problem)
	item := leetgrinder.TodoItem{ID: uuid.NewString(), SetID: setID, Problem: leetgrinder.Problem{Slug: problem.Slug}}
	if setID == "" {
		err = tx.QueryRowContext(ctx, `INSERT INTO leetgrinder_todo_items(id,problem_slug,source_data) VALUES($1,$2,$3::jsonb)
ON CONFLICT (problem_slug) WHERE set_id IS NULL DO UPDATE SET source_data=CASE WHEN EXCLUDED.source_data='{}'::jsonb THEN leetgrinder_todo_items.source_data ELSE EXCLUDED.source_data END RETURNING id`, item.ID, problem.Slug, source).Scan(&item.ID)
	} else {
		err = tx.QueryRowContext(ctx, `INSERT INTO leetgrinder_todo_items(id,set_id,problem_slug,source_data) VALUES($1,$2,$3,$4::jsonb)
ON CONFLICT (set_id,problem_slug) DO UPDATE SET source_data=CASE WHEN EXCLUDED.source_data='{}'::jsonb THEN leetgrinder_todo_items.source_data ELSE EXCLUDED.source_data END RETURNING id`, item.ID, setID, problem.Slug, source).Scan(&item.ID)
	}
	if err != nil {
		return leetgrinder.TodoItem{}, err
	}
	if err = recordTodoSource(ctx, tx, problem.Slug, item.ID, source); err != nil {
		return leetgrinder.TodoItem{}, err
	}
	return item, tx.Commit()
}

func (s *Store) DeleteLeetgrinderTodoItem(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalid
	}
	result, err := s.DB.ExecContext(ctx, "DELETE FROM leetgrinder_todo_items WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteLeetgrinderTodoSet(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalid
	}
	result, err := s.DB.ExecContext(ctx, "DELETE FROM leetgrinder_todo_sets WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
