package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestLeetgrinderAnalysesMigration(t *testing.T) {
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
		if version > 13 {
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
	const id = "11111111-1111-4111-8111-111111111111"
	if _, err = s.DB.Exec(`UPDATE leetgrinder_settings SET ntfy_topic='keep-me'; INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source,time_complexity,space_complexity,code,code_language) VALUES('`+id+`','two-sum','solved',23,false,'','22222222-2222-4222-8222-222222222222','extension','O(n)','O(1)','pass','python3')`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err = s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	settings, err := s.LeetgrinderSettings(ctx)
	if err != nil || !settings.AnalysisEnabled || settings.NtfyTopic != "keep-me" {
		t.Fatalf("settings %+v %v", settings, err)
	}
	insert := "INSERT INTO leetgrinder_analyses(attempt_id,code_sha256,status,explanation,error) VALUES($1,$2,$3,$4,$5)"
	hash := make([]byte, 32)
	for name, args := range map[string][]any{
		"short hash":       {id, []byte{1}, "pending", "", ""},
		"unknown status":   {id, hash, "running", "", ""},
		"long explanation": {id, hash, "done", strings.Repeat("x", 2001), ""},
		"long error":       {id, hash, "failed", "", strings.Repeat("x", 301)},
		"unknown attempt":  {"33333333-3333-4333-8333-333333333333", hash, "pending", "", ""},
	} {
		if _, err = s.DB.Exec(insert, args...); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err = s.DB.Exec(insert, id, hash, "done", strings.Repeat("é", 2000), ""); err != nil {
		t.Fatal(err)
	}
	// The stored hash is not the attempt's current input hash.
	state, err := s.LeetgrinderState(ctx)
	if err != nil || len(state.Analyses) != 1 || state.Analyses[id].Current {
		t.Fatalf("analyses %+v %v", state.Analyses, err)
	}
	if _, err = s.DB.Exec("DELETE FROM leetgrinder_attempts WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = s.DB.QueryRow("SELECT count(*) FROM leetgrinder_analyses").Scan(&n); err != nil || n != 0 {
		t.Fatalf("cascade left %d rows: %v", n, err)
	}
}
