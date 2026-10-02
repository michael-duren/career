package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
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
	if _, err = s.DB.Exec(`UPDATE leetgrinder_settings SET ntfy_topic='keep-me'; INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source,time_complexity,space_complexity,code,code_language) VALUES('` + id + `','two-sum','solved',23,false,'','22222222-2222-4222-8222-222222222222','extension','O(n)','O(1)','pass','python3')`); err != nil {
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

// leetgrinderOldInputHash is the expression analyses were stored against
// before migration 026 stored it on the attempt.
const leetgrinderOldInputHash = `sha256(convert_to(code_language || chr(10) || time_complexity || chr(10) || space_complexity || chr(10) || code, 'UTF8'))`

// Migration 026 backfills input_sha256 with the hash existing analyses were
// stored against, so they stay current, and the trigger keeps it current for
// new and corrected attempts.
func TestLeetgrinderAttemptInputHash(t *testing.T) {
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
		if version > 25 {
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
	const analysed, unicode, web = "11111111-1111-4111-8111-111111111111", "11111111-1111-4111-8111-111111111112", "11111111-1111-4111-8111-111111111113"
	const code = "def f(s):\n    return {c: \"é🙂\" for c in s}\n"
	if _, err = s.DB.Exec(`INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source,time_complexity,space_complexity,code,code_language,created_at) VALUES
($1,'two-sum','solved',23,false,'','21111111-1111-4111-8111-111111111111','extension','O(n)','O(1)','pass','python3',now() - interval '1 day'),
($2,'valid-anagram','solved',12,false,'','21111111-1111-4111-8111-111111111112','extension','O(n)','',$4,'python3',now() - interval '2 days'),
($3,'lru-cache','unfinished',30,false,'','21111111-1111-4111-8111-111111111113','web','','','','',now() - interval '3 days')`, analysed, unicode, web, code); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(`INSERT INTO leetgrinder_analyses(attempt_id,code_sha256,status,actual_time,actual_space,time_matches,space_matches,optimal)
SELECT id,`+leetgrinderOldInputHash+`,'done','O(n)','O(1)',true,true,true FROM leetgrinder_attempts WHERE id=$1`, analysed); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err = s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	checkHashes := func(n int) {
		t.Helper()
		var rows, matching int
		if err := s.DB.QueryRow(`SELECT count(*), count(*) FILTER (WHERE input_sha256=` + leetgrinderOldInputHash + `) FROM leetgrinder_attempts`).Scan(&rows, &matching); err != nil {
			t.Fatal(err)
		}
		if rows != n || matching != n {
			t.Fatalf("%d of %d attempts store the old input hash, want %d", matching, rows, n)
		}
	}
	checkHashes(3)
	// The stored bytes are the SHA-256 of the UTF-8 inputs joined by newlines.
	var stored []byte
	if err = s.DB.QueryRow("SELECT input_sha256 FROM leetgrinder_attempts WHERE id=$1", unicode).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if want := sha256.Sum256([]byte("python3\nO(n)\n\n" + code)); string(stored) != string(want[:]) {
		t.Fatalf("input_sha256 %x, want %x", stored, want)
	}
	state, err := s.LeetgrinderState(ctx)
	if err != nil || !state.Analyses[analysed].Current {
		t.Fatalf("backfilled analysis is not current: %+v %v", state.Analyses, err)
	}
	q, err := s.LeetgrinderAnalysisQueueCounts(ctx)
	if err != nil || q.Queued != 1 || q.Done != 1 || q.Failed != 0 {
		t.Fatalf("queue %+v %v", q, err)
	}

	saved, err := s.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 9, Source: "extension", TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: "pass\n", CodeLanguage: "python3"}, "")
	if err != nil {
		t.Fatal(err)
	}
	checkHashes(4)
	// Finish the newest attempt so the correction below is next in the queue.
	job, ok, err := s.NextLeetgrinderAnalysis(ctx, time.Now())
	if err != nil || !ok || job.Attempt.ID != saved.ID {
		t.Fatalf("next job %+v %v %v", job, ok, err)
	}
	if err = s.FinishLeetgrinderAnalysis(ctx, job, leetgrinder.AnalysisDone, 1, leetgrinder.AnalysisResult{ActualTime: "O(n)", ActualSpace: "O(n)"}, "test", "", time.Now()); err != nil {
		t.Fatal(err)
	}

	// Correcting a stated complexity changes the hash and requeues the attempt.
	state, err = s.LeetgrinderState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var old leetgrinder.Attempt
	for _, a := range state.Attempts {
		if a.ID == analysed {
			old = a
		}
	}
	old.TimeComplexity = "O(n log n)"
	if _, err = s.SaveLeetgrinderAttempt(ctx, old, old.Revision); err != nil {
		t.Fatal(err)
	}
	checkHashes(4)
	state, err = s.LeetgrinderState(ctx)
	if err != nil || state.Analyses[analysed].Current || !state.Analyses[saved.ID].Current {
		t.Fatalf("analyses after correction %+v %v", state.Analyses, err)
	}
	q, err = s.LeetgrinderAnalysisQueueCounts(ctx)
	if err != nil || q.Queued != 2 || q.Done != 1 {
		t.Fatalf("queue after correction %+v %v", q, err)
	}
	job, ok, err = s.NextLeetgrinderAnalysis(ctx, time.Now())
	if err != nil || !ok || job.Attempt.ID != analysed || job.Tries != 0 {
		t.Fatalf("corrected attempt not queued: %+v %v %v", job, ok, err)
	}
	var want []byte
	if err = s.DB.QueryRow("SELECT "+leetgrinderOldInputHash+" FROM leetgrinder_attempts WHERE id=$1", analysed).Scan(&want); err != nil || string(job.Hash) != string(want) {
		t.Fatalf("job hash %x, want %x (%v)", job.Hash, want, err)
	}
}
