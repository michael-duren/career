package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestLeetgrinderMigrationPreservesProgress(t *testing.T) {
	s := emptyStore(t)
	ctx := context.Background()
	if _, err := s.DB.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		version, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil {
			t.Fatal(err)
		}
		if version > 9 {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec(string(sql)); err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec("INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", version, fmt.Sprintf("%x", sha256.Sum256(sql))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.DB.Exec(`INSERT INTO bootcamp_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision) VALUES('11111111-1111-4111-8111-111111111111','two-sum','solved',23,false,'Keep this history','22222222-2222-4222-8222-222222222222'); INSERT INTO bootcamp_completed_days(day) VALUES(1)`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var notes string
	var minutes int
	if err = s.DB.QueryRow("SELECT notes,minutes FROM leetgrinder_attempts WHERE problem_slug='two-sum'").Scan(&notes, &minutes); err != nil {
		t.Fatal(err)
	}
	if notes != "Keep this history" || minutes != 23 {
		t.Fatal("migration changed saved progress")
	}
	var oldNames int
	if err = s.DB.QueryRow(`SELECT count(*) FROM pg_class WHERE relnamespace=current_schema()::regnamespace AND relname LIKE 'bootcamp_%'`).Scan(&oldNames); err != nil {
		t.Fatal(err)
	}
	if oldNames != 0 {
		t.Fatalf("%d old table/index names remain", oldNames)
	}
}

func TestMigrateExistingNoteCreationSchema(t *testing.T) {
	s := emptyStore(t)
	ctx := context.Background()
	if _, err := s.DB.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		version, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil {
			t.Fatal(err)
		}
		if version > 7 {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec(string(sql)); err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec("INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", version, fmt.Sprintf("%x", sha256.Sum256(sql))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.DB.Exec(`ALTER TABLE notes ADD COLUMN created_at TIMESTAMPTZ; UPDATE notes SET created_at=updated_at;
 INSERT INTO schema_migrations(version,checksum) VALUES(8,'b9deff5f6bbab8fd20f638c59885fc335ed32e206e24d686e302399549eb5ea7');
 INSERT INTO notes(id,title,description,tags,body,revision,position,topic,created_at) VALUES('keep-me','Existing note','','{}','Preserve this body','revision-1',1,'work','2026-09-01T12:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var body, created string
	if err = s.DB.QueryRow(`SELECT body,to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI:SS') FROM notes WHERE id='keep-me'`).Scan(&body, &created); err != nil {
		t.Fatal(err)
	}
	if body != "Preserve this body" || created != "2026-09-01 12:00:00" {
		t.Fatal("existing note changed")
	}
	if _, err = s.LeetgrinderState(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal("repeat migration:", err)
	}
	if _, err = s.DB.Exec("UPDATE schema_migrations SET checksum='unexpected' WHERE version=8"); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err == nil {
		t.Fatal("unknown checksum was accepted")
	}
}

// Migration 023 dates existing struggles' flags from the migration, so past
// days are not recounted; a mark keeps its own date, and clean solves stay.
func TestLeetgrinderStruggleFlagMigration(t *testing.T) {
	s := emptyStore(t)
	ctx := context.Background()
	if _, err := s.DB.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		version, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil {
			t.Fatal(err)
		}
		if version > 22 {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec(string(sql)); err != nil {
			t.Fatalf("%s: %v", file.Name(), err)
		}
		if _, err = s.DB.Exec("INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", version, fmt.Sprintf("%x", sha256.Sum256(sql))); err != nil {
			t.Fatal(err)
		}
	}
	old := "now() - interval '30 days'"
	if _, err = s.DB.Exec(`INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,created_at,marked_at,wants_review) VALUES
('11111111-1111-4111-8111-111111111111','two-sum','struggled',30,false,'','21111111-1111-4111-8111-111111111111',` + old + `,` + old + `,false),
('11111111-1111-4111-8111-111111111112','valid-anagram','solved',10,true,'','21111111-1111-4111-8111-111111111112',` + old + `,` + old + `,false),
('11111111-1111-4111-8111-111111111113','lru-cache','unfinished',30,false,'','21111111-1111-4111-8111-111111111113',` + old + `,` + old + `,true),
('11111111-1111-4111-8111-111111111114','binary-search','solved',10,false,'','21111111-1111-4111-8111-111111111114',` + old + `,` + old + `,false)`); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	rows, err := s.DB.Query(`SELECT problem_slug, marked_at > now() - interval '1 hour' FROM leetgrinder_attempts`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]bool{}
	for rows.Next() {
		var slug string
		var recent bool
		if err = rows.Scan(&slug, &recent); err != nil {
			t.Fatal(err)
		}
		got[slug] = recent
	}
	want := map[string]bool{"two-sum": true, "valid-anagram": true, "lru-cache": false, "binary-search": false}
	for slug, recent := range want {
		if got[slug] != recent {
			t.Errorf("%s: marked_at moved to now = %v, want %v", slug, got[slug], recent)
		}
	}
}

func TestLeetgrinderAttemptSlugIndex(t *testing.T) {
	s := emptyStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var def string
	if err := s.DB.QueryRow(`SELECT indexdef FROM pg_indexes WHERE schemaname=current_schema() AND tablename='leetgrinder_attempts' AND indexname='leetgrinder_attempts_problem_slug_created_at_idx'`).Scan(&def); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(def, "(problem_slug, created_at) INCLUDE (outcome)") {
		t.Fatalf("unexpected index definition: %s", def)
	}
	// Seqscans are only cheaper on an empty table, so forbid them to prove the
	// planner can use the index for the todo done_at lookup.
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SET LOCAL enable_seqscan=off"); err != nil {
		t.Fatal(err)
	}
	rows, err := tx.QueryContext(ctx, `EXPLAIN SELECT max(a.created_at) FROM leetgrinder_attempts a WHERE a.problem_slug='two-sum' AND a.outcome IN ('solved','struggled')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err = rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		plan.WriteString(line + "\n")
	}
	if !strings.Contains(plan.String(), "leetgrinder_attempts_problem_slug_created_at_idx") {
		t.Fatalf("done_at lookup does not use the index:\n%s", plan.String())
	}
}
