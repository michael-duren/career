package database

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderAnalysesMigration(t *testing.T) {
	s := emptyStore(t)
	ctx := context.Background()
	migrateTo(t, s, 13)
	const id = "11111111-1111-4111-8111-111111111111"
	if _, err := s.DB.Exec(`UPDATE leetgrinder_settings SET ntfy_topic='keep-me'; INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source,time_complexity,space_complexity,code,code_language) VALUES('` + id + `','two-sum','solved',23,false,'','22222222-2222-4222-8222-222222222222','extension','O(n)','O(1)','pass','python3')`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.Migrate(ctx); err != nil {
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
	migrateTo(t, s, 25)
	const analysed, unicode, web = "11111111-1111-4111-8111-111111111111", "11111111-1111-4111-8111-111111111112", "11111111-1111-4111-8111-111111111113"
	const code = "def f(s):\n    return {c: \"é🙂\" for c in s}\n"
	if _, err := s.DB.Exec(`INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source,time_complexity,space_complexity,code,code_language,created_at) VALUES
($1,'two-sum','solved',23,false,'','21111111-1111-4111-8111-111111111111','extension','O(n)','O(1)','pass','python3',now() - interval '1 day'),
($2,'valid-anagram','solved',12,false,'','21111111-1111-4111-8111-111111111112','extension','O(n)','',$4,'python3',now() - interval '2 days'),
($3,'lru-cache','unfinished',30,false,'','21111111-1111-4111-8111-111111111113','web','','','','',now() - interval '3 days')`, analysed, unicode, web, code); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`INSERT INTO leetgrinder_analyses(attempt_id,code_sha256,status,actual_time,actual_space,time_matches,space_matches,optimal)
SELECT id,`+leetgrinderOldInputHash+`,'done','O(n)','O(1)',true,true,true FROM leetgrinder_attempts WHERE id=$1`, analysed); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	stored := func(id string) []byte {
		t.Helper()
		var h []byte
		if err := s.DB.QueryRow("SELECT input_sha256 FROM leetgrinder_attempts WHERE id=$1", id).Scan(&h); err != nil {
			t.Fatal(err)
		}
		return h
	}
	checkHashes := func(n int) {
		t.Helper()
		var rows, matching int
		if err := s.DB.QueryRow(`SELECT count(*), count(*) FILTER (WHERE input_sha256=`+leetgrinderOldInputHash+`) FROM leetgrinder_attempts`).Scan(&rows, &matching); err != nil {
			t.Fatal(err)
		}
		if rows != n || matching != n {
			t.Fatalf("%d of %d attempts store the old input hash, want %d", matching, rows, n)
		}
	}
	checkHashes(3)
	// The stored bytes are the SHA-256 of the UTF-8 inputs joined by newlines.
	for id, inputs := range map[string]string{
		analysed: "python3\nO(n)\nO(1)\npass",
		unicode:  "python3\nO(n)\n\n" + code,
		web:      "\n\n\n",
	} {
		if got, want := stored(id), sha256.Sum256([]byte(inputs)); string(got) != string(want[:]) {
			t.Errorf("%s: input_sha256 %x, want %x", id, got, want)
		}
	}
	current := func(id string) bool {
		t.Helper()
		state, err := s.LeetgrinderState(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return state.Analyses[id].Current
	}
	if !current(analysed) {
		t.Fatal("backfilled analysis is not current")
	}
	q, err := s.LeetgrinderAnalysisQueueCounts(ctx)
	if err != nil || q.Queued != 1 || q.Done != 1 || q.Failed != 0 {
		t.Fatalf("queue %+v %v", q, err)
	}

	// Updates that leave the inputs alone keep the hash and the analysis.
	before := stored(analysed)
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET notes='more notes', marked_at=clock_timestamp() WHERE id=$1", analysed); err != nil {
		t.Fatal(err)
	}
	if got := stored(analysed); string(got) != string(before) || !current(analysed) {
		t.Fatalf("notes update changed the hash to %x or made the analysis stale", got)
	}
	// A value written directly is replaced on insert and on update.
	bogus := make([]byte, 32)
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET input_sha256=$2 WHERE id=$1", analysed, bogus); err != nil {
		t.Fatal(err)
	}
	if got := stored(analysed); string(got) != string(before) {
		t.Fatalf("direct update stored %x", got)
	}
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET input_sha256=$2, space_complexity='O(n)' WHERE id=$1", web, bogus); err != nil {
		t.Fatal(err)
	}
	const direct = "11111111-1111-4111-8111-111111111114"
	if _, err = s.DB.Exec(`INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source,code,code_language,input_sha256,created_at)
VALUES($1,'two-sum','solved',5,false,'','21111111-1111-4111-8111-111111111114','extension','x','go',$2,now() - interval '4 days')`, direct, bogus); err != nil {
		t.Fatal(err)
	}
	checkHashes(4)
	if got, want := stored(direct), sha256.Sum256([]byte("go\n\n\nx")); string(got) != string(want[:]) {
		t.Fatalf("direct insert stored %x, want %x", got, want)
	}

	saved, err := s.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 9, Source: "extension", TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: "pass\n", CodeLanguage: "python3"}, "")
	if err != nil {
		t.Fatal(err)
	}
	checkHashes(5)
	// Finish the newest attempt so the correction below is next in the queue.
	job, ok, err := s.NextLeetgrinderAnalysis(ctx, time.Now())
	if err != nil || !ok || job.Attempt.ID != saved.ID || string(job.Hash) != string(stored(saved.ID)) {
		t.Fatalf("next job %+v %v %v", job, ok, err)
	}
	if err = s.FinishLeetgrinderAnalysis(ctx, job, leetgrinder.AnalysisDone, 1, leetgrinder.AnalysisResult{ActualTime: "O(n)", ActualSpace: "O(n)"}, "test", "", time.Now()); err != nil {
		t.Fatal(err)
	}

	// Correcting a stated complexity changes the hash and requeues the attempt.
	state, err := s.LeetgrinderState(ctx)
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
	checkHashes(5)
	if string(stored(analysed)) == string(before) || current(analysed) || !current(saved.ID) {
		t.Fatal("correction kept the hash or did not requeue the attempt")
	}
	q, err = s.LeetgrinderAnalysisQueueCounts(ctx)
	if err != nil || q.Queued != 2 || q.Done != 1 {
		t.Fatalf("queue after correction %+v %v", q, err)
	}
	job, ok, err = s.NextLeetgrinderAnalysis(ctx, time.Now())
	if err != nil || !ok || job.Attempt.ID != analysed || job.Tries != 0 || string(job.Hash) != string(stored(analysed)) {
		t.Fatalf("corrected attempt not queued: %+v %v %v", job, ok, err)
	}

	// Re-analyse stores the attempt's stored hash, so the reset analysis is current.
	if err = s.RequeueLeetgrinderAnalysis(ctx, "valid-anagram", unicode, time.Now()); err != nil {
		t.Fatal(err)
	}
	var requeued []byte
	if err = s.DB.QueryRow("SELECT code_sha256 FROM leetgrinder_analyses WHERE attempt_id=$1", unicode).Scan(&requeued); err != nil {
		t.Fatal(err)
	}
	if string(requeued) != string(stored(unicode)) || !current(unicode) {
		t.Fatalf("requeue stored %x, want %x", requeued, stored(unicode))
	}
}
