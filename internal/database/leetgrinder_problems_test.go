package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// migrateTo applies the migrations up to and including version, without
// the Go seeding that Migrate runs afterwards.
func migrateTo(t *testing.T, s *Store, version int) {
	t.Helper()
	if _, err := s.DB.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		v, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil {
			t.Fatal(err)
		}
		if v > version {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec(string(sql)); err != nil {
			t.Fatalf("%s: %v", file.Name(), err)
		}
		if _, err = s.DB.Exec("INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", v, fmt.Sprintf("%x", sha256.Sum256(sql))); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLeetgrinderCatalogMigrationSeeds(t *testing.T) {
	s := emptyStore(t)
	ctx := context.Background()
	migrateTo(t, s, 18)
	if _, err := s.DB.Exec(`INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision) VALUES
('11111111-1111-4111-8111-111111111111','two-sum','solved',23,false,'Keep this history','22222222-2222-4222-8222-222222222222'),
('33333333-3333-4333-8333-333333333333','design-twitter','unfinished',40,false,'','44444444-4444-4444-8444-444444444444')`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var total, curated int
	if err := s.DB.QueryRow("SELECT count(*), count(*) FILTER (WHERE optimal_source='curated' AND metadata_source='seed') FROM leetgrinder_problems").Scan(&total, &curated); err != nil {
		t.Fatal(err)
	}
	if total != 301 || curated != 300 {
		t.Fatalf("catalog has %d rows, %d curated", total, curated)
	}
	state, err := s.LeetgrinderState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Attempts) != 2 || state.Attempts[0].Notes != "Keep this history" && state.Attempts[1].Notes != "Keep this history" {
		t.Fatalf("attempts changed: %+v", state.Attempts)
	}
	two, bare := state.Problem("two-sum"), state.Problem("design-twitter")
	if two.Number != 1 || two.Title != "Two Sum" || two.Difficulty != "Easy" || two.OptimalTime != "O(n)" || two.OptimalSource != "curated" || len(two.Topics) != 0 {
		t.Fatalf("seeded two-sum: %+v", two)
	}
	if bare.Known() || bare.Number != 0 || bare.MetadataSource != "" || !bare.Fetching() {
		t.Fatalf("bare row: %+v", bare)
	}
	for _, bad := range []string{
		"INSERT INTO leetgrinder_problems(slug) VALUES('Not A Slug')",
		"INSERT INTO leetgrinder_problems(slug,difficulty) VALUES('x','easy')",
		"INSERT INTO leetgrinder_problems(slug,optimal_source) VALUES('x','guess')",
		"INSERT INTO leetgrinder_problems(slug,number) VALUES('x',0)",
		"INSERT INTO leetgrinder_problems(slug) VALUES('two-sum')",
	} {
		if _, err = s.DB.Exec(bad); err == nil {
			t.Errorf("constraint accepted: %s", bad)
		}
	}
}

func TestLeetgrinderMetadataPrecedence(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	meta := leetgrinder.ProblemMetadata{Number: 1, Title: "Two Sum", Difficulty: "Easy", Topics: []leetgrinder.TopicTag{{Slug: "array", Name: "Array"}}}
	if err := s.SaveLeetgrinderMetadata(ctx, "two-sum", meta, now); err != nil {
		t.Fatal(err)
	}
	p, _ := s.LeetgrinderProblem(ctx, "two-sum")
	if p.MetadataSource != "extension" || p.OptimalSource != "curated" || p.OptimalTime != "O(n)" || strings.Join(p.Topics, ",") != "array" {
		t.Fatalf("curated row after metadata: %+v", p)
	}
	// Empty values never erase known ones.
	if err := s.SaveLeetgrinderMetadata(ctx, "two-sum", leetgrinder.ProblemMetadata{Title: "Two Sum"}, now); err != nil {
		t.Fatal(err)
	}
	if p, _ = s.LeetgrinderProblem(ctx, "two-sum"); p.Title != "Two Sum" || p.Number != 1 || len(p.Topics) != 1 {
		t.Fatalf("empty metadata erased values: %+v", p)
	}
	for name, bad := range map[string]leetgrinder.ProblemMetadata{
		"difficulty": {Title: "X", Difficulty: "Impossible"},
		"topics":     {Title: "X", Topics: []leetgrinder.TopicTag{{Slug: "Bad Tag"}}},
		"no title":   {Number: 1, Difficulty: "Easy"},
	} {
		if err := s.SaveLeetgrinderMetadata(ctx, "two-sum", bad, now); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if err := s.SaveLeetgrinderMetadata(ctx, "Bad Slug", meta, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad slug: %v", err)
	}
	// An attempt adds its problem, with metadata in the same transaction.
	a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "lru-cache", Outcome: "unfinished", Minutes: 30, Source: "extension"}
	lru := &leetgrinder.ProblemMetadata{Number: 146, Title: "LRU Cache", Difficulty: "Medium", Topics: []leetgrinder.TopicTag{{Slug: "design", Name: "Design"}}}
	if _, err := s.SaveLeetgrinderAttemptWithProblem(ctx, a, "", lru, now); err != nil {
		t.Fatal(err)
	}
	if p, _ = s.LeetgrinderProblem(ctx, "lru-cache"); p.Title != "LRU Cache" || p.Number != 146 || p.MetadataSource != "extension" || p.Fetching() {
		t.Fatalf("attempt metadata: %+v", p)
	}
	// Invalid metadata rejects the whole attempt.
	b := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "lfu-cache", Outcome: "unfinished", Minutes: 30, Source: "extension"}
	if _, err := s.SaveLeetgrinderAttemptWithProblem(ctx, b, "", &leetgrinder.ProblemMetadata{Number: -1}, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid metadata: %v", err)
	}
	if state, _ := s.LeetgrinderState(ctx); len(state.ProblemAttempts("lfu-cache")) != 0 {
		t.Fatal("attempt saved despite invalid metadata")
	}
	// Web attempts add a bare row.
	c := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "design-hit-counter", Outcome: "unfinished", Minutes: 30}
	if _, err := s.SaveLeetgrinderAttempt(ctx, c, ""); err != nil {
		t.Fatal(err)
	}
	if p, _ = s.LeetgrinderProblem(ctx, "design-hit-counter"); p.Slug != "design-hit-counter" || p.Known() || !p.Fetching() {
		t.Fatalf("bare row: %+v", p)
	}
	if p, err := s.LeetgrinderProblem(ctx, "never-seen"); err != nil || p.Slug != "never-seen" || p.Known() {
		t.Fatalf("missing row: %+v %v", p, err)
	}
	if _, err := s.EnsureLeetgrinderProblem(ctx, "Bad Slug"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ensure bad slug: %v", err)
	}
}
