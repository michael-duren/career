package database

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderTokens(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	token, plain, err := s.CreateLeetgrinderToken(ctx, "  Firefox laptop  ")
	if err != nil {
		t.Fatal(err)
	}
	if token.Name != "Firefox laptop" || !strings.HasPrefix(plain, "lg_") || !leetgrinder.WellFormedAPIToken(plain) {
		t.Fatalf("bad token %+v %q", token, plain)
	}
	var stored []byte
	if err = s.DB.QueryRowContext(ctx, "SELECT token_hash FROM leetgrinder_api_tokens WHERE id=$1", token.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, leetgrinder.HashAPIToken(plain)) || bytes.Contains(stored, []byte(plain)) {
		t.Fatal("token stored in the clear or with the wrong hash")
	}

	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	got, err := s.AuthenticateLeetgrinderToken(ctx, plain, now)
	if err != nil || got.ID != token.ID || got.LastUsedAt == nil || !got.LastUsedAt.Equal(now) {
		t.Fatalf("authenticate: %+v %v", got, err)
	}
	lastUsed := func() time.Time {
		var used time.Time
		if err := s.DB.QueryRowContext(ctx, "SELECT last_used_at FROM leetgrinder_api_tokens WHERE id=$1", token.ID).Scan(&used); err != nil {
			t.Fatal(err)
		}
		return used
	}
	// Within a minute: no write.
	if _, err = s.AuthenticateLeetgrinderToken(ctx, plain, now.Add(59*time.Second)); err != nil {
		t.Fatal(err)
	}
	if !lastUsed().Equal(now) {
		t.Fatalf("last_used_at written within a minute: %v", lastUsed())
	}
	later := now.Add(61 * time.Second)
	if _, err = s.AuthenticateLeetgrinderToken(ctx, plain, later); err != nil {
		t.Fatal(err)
	}
	if !lastUsed().Equal(later) {
		t.Fatalf("last_used_at not refreshed after a minute: %v", lastUsed())
	}

	for _, bad := range []string{"", "lg_", "lg_short", strings.Repeat("x", 46), "lg_" + strings.Repeat("A", 43), plain + "x", strings.ToUpper(plain[:3]) + plain[3:]} {
		if _, err := s.AuthenticateLeetgrinderToken(ctx, bad, now); !errors.Is(err, ErrUnauthorized) {
			t.Errorf("%q: %v", bad, err)
		}
	}

	other, otherPlain, err := s.CreateLeetgrinderToken(ctx, "Chrome")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RevokeLeetgrinderToken(ctx, token.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.RevokeLeetgrinderToken(ctx, token.ID); err != nil {
		t.Fatalf("second revoke: %v", err)
	}
	if _, err = s.AuthenticateLeetgrinderToken(ctx, plain, later.Add(time.Hour)); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("revoked token accepted: %v", err)
	}
	if _, err = s.AuthenticateLeetgrinderToken(ctx, otherPlain, now); err != nil {
		t.Fatalf("other token rejected: %v", err)
	}
	if err = s.RevokeLeetgrinderToken(ctx, uuid.NewString()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing token: %v", err)
	}
	if err = s.RevokeLeetgrinderToken(ctx, "nope"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad id: %v", err)
	}

	tokens, err := s.LeetgrinderTokens(ctx)
	if err != nil || len(tokens) != 2 || tokens[0].ID != other.ID || tokens[1].ID != token.ID || !tokens[1].Revoked() || tokens[0].Revoked() {
		t.Fatalf("list: %+v %v", tokens, err)
	}

	for _, name := range []string{"", "   ", strings.Repeat("n", 65), "tab\tname"} {
		if _, _, err := s.CreateLeetgrinderToken(ctx, name); !errors.Is(err, ErrInvalid) {
			t.Errorf("name %q: %v", name, err)
		}
	}
	for i := 1; i < maxActiveLeetgrinderTokens; i++ {
		if _, _, err := s.CreateLeetgrinderToken(ctx, "bulk"); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := s.CreateLeetgrinderToken(ctx, "one too many"); !errors.Is(err, ErrTokenLimit) {
		t.Fatalf("limit: %v", err)
	}
}
