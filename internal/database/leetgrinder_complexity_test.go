package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderComplexityMigrationKeepsAttempts(t *testing.T) {
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
		if version > 12 {
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
	if _, err = s.DB.Exec(`INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source) VALUES($1,'two-sum','solved',23,false,'Keep this history','22222222-2222-4222-8222-222222222222','extension')`, id); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
	}
	state, err := s.LeetgrinderState(ctx)
	if err != nil || len(state.Attempts) != 1 {
		t.Fatalf("state: %+v %v", state, err)
	}
	a := state.Attempts[0]
	if a.Notes != "Keep this history" || a.Minutes != 23 || a.Source != "extension" || a.TimeComplexity != "" || a.SpaceComplexity != "" || a.Code != "" || a.CodeLanguage != "" {
		t.Fatalf("migration changed the attempt: %+v", a)
	}
	for _, update := range []string{
		"time_complexity=repeat('x',41)",
		"space_complexity=repeat('x',41)",
		"code=repeat('x',65537)",
		"code_language=repeat('x',33)",
		// 21,846 three-byte characters are 65,538 bytes.
		"code=repeat('界',21846)",
	} {
		if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET "+update+" WHERE id=$1", id); err == nil {
			t.Errorf("constraint allowed %s", update)
		}
	}
	// Limits count characters for complexity and bytes for code.
	if _, err = s.DB.Exec("UPDATE leetgrinder_attempts SET time_complexity=repeat('界',40),code=repeat('x',65536),code_language=repeat('x',32) WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
}

func TestLeetgrinderAttemptDetails(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	input := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 12, Source: "extension", TimeComplexity: "O(nlogn)", SpaceComplexity: " O(n) ", Code: "class Solution:\n    pass\n", CodeLanguage: "python3"}
	saved, err := s.SaveLeetgrinderAttempt(ctx, input, "")
	if err != nil || saved.TimeComplexity != "O(n log n)" || saved.SpaceComplexity != "O(n)" || saved.Code != input.Code || saved.CodeLanguage != "python3" {
		t.Fatalf("saved %+v %v", saved, err)
	}
	// Retries compare normalised values, so the same raw input is idempotent.
	if retry, err := s.SaveLeetgrinderAttempt(ctx, input, ""); err != nil || !reflect.DeepEqual(retry, saved) {
		t.Fatalf("retry = %+v, %v", retry, err)
	}
	for name, change := range map[string]func(*leetgrinder.Attempt){
		"time":     func(a *leetgrinder.Attempt) { a.TimeComplexity = "O(n)" },
		"space":    func(a *leetgrinder.Attempt) { a.SpaceComplexity = "O(1)" },
		"code":     func(a *leetgrinder.Attempt) { a.Code += "#" },
		"language": func(a *leetgrinder.Attempt) { a.CodeLanguage = "python" },
	} {
		changed := input
		change(&changed)
		if _, err := s.SaveLeetgrinderAttempt(ctx, changed, ""); !errors.Is(err, ErrConflict) {
			t.Errorf("retry with different %s: %v", name, err)
		}
	}
	// A web correction carries no code; the captured code stays.
	correction := saved
	correction.Code, correction.CodeLanguage, correction.TimeComplexity = "", "", "O(n^2)"
	corrected, err := s.SaveLeetgrinderAttempt(ctx, correction, saved.Revision)
	if err != nil || corrected.TimeComplexity != "O(n²)" || corrected.Code != input.Code || corrected.CodeLanguage != "python3" {
		t.Fatalf("corrected %+v %v", corrected, err)
	}
	// Corrections of solved or struggled attempts need both complexities.
	correction = corrected
	correction.SpaceComplexity = ""
	if _, err := s.SaveLeetgrinderAttempt(ctx, correction, corrected.Revision); !errors.Is(err, ErrInvalid) {
		t.Fatalf("correction without space: %v", err)
	}
	correction.Outcome = "unfinished"
	if unfinished, err := s.SaveLeetgrinderAttempt(ctx, correction, corrected.Revision); err != nil || unfinished.SpaceComplexity != "" {
		t.Fatalf("unfinished correction: %+v %v", unfinished, err)
	}
	for name, change := range map[string]func(*leetgrinder.Attempt){
		"missing time":  func(a *leetgrinder.Attempt) { a.TimeComplexity = "" },
		"missing space": func(a *leetgrinder.Attempt) { a.Outcome, a.SpaceComplexity = "struggled", "" },
		"bad notation":  func(a *leetgrinder.Attempt) { a.TimeComplexity = "fast" },
		"long code":     func(a *leetgrinder.Attempt) { a.Code = strings.Repeat("x", leetgrinder.MaxCodeBytes+1) },
		"no language":   func(a *leetgrinder.Attempt) { a.CodeLanguage = "" },
	} {
		bad := input
		bad.ID = uuid.NewString()
		change(&bad)
		if _, err := s.SaveLeetgrinderAttempt(ctx, bad, ""); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	unfinished := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 25, Code: strings.Repeat("x", leetgrinder.MaxCodeBytes), CodeLanguage: "cpp"}
	if _, err := s.SaveLeetgrinderAttempt(ctx, unfinished, ""); err != nil {
		t.Fatalf("unfinished without complexity: %v", err)
	}
}
