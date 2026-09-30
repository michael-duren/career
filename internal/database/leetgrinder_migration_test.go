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
