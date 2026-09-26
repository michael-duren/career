package leetgrinder

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSecretBoxRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	box, err := NewSecretBox(key)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal([]byte("tk_secret"))
	if err != nil {
		t.Fatal(err)
	}
	again, _ := box.Seal([]byte("tk_secret"))
	if bytes.Contains(sealed, []byte("tk_secret")) || bytes.Equal(sealed, again) {
		t.Fatal("ciphertext leaks plaintext or reuses a nonce")
	}
	if plain, err := box.Open(sealed); err != nil || string(plain) != "tk_secret" {
		t.Fatalf("open: %q %v", plain, err)
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := box.Open(sealed); err == nil {
		t.Fatal("tampered ciphertext opened")
	}
	other, _ := NewSecretBox(bytes.Repeat([]byte{8}, 32))
	if _, err := other.Open(again); err == nil {
		t.Fatal("wrong key opened ciphertext")
	}
	if _, err := box.Open([]byte("short")); err == nil {
		t.Fatal("short ciphertext opened")
	}
	if _, err := NewSecretBox(nil); !errors.Is(err, ErrNoSecretKey) {
		t.Fatalf("missing key: %v", err)
	}
	if _, err := NewSecretBox(key[:16]); err == nil {
		t.Fatal("accepted a 16-byte key")
	}
}

func TestSettingsValidation(t *testing.T) {
	valid := DefaultSettings()
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, h := range []float64{2, 2.5, 3, 3.5, 4} {
		if !ValidDailyHours(h) {
			t.Errorf("rejected %.1f hours", h)
		}
	}
	far := time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC)
	for name, change := range map[string]func(*Settings){
		"local zone":    func(s *Settings) { s.Timezone = "Local" },
		"unknown zone":  func(s *Settings) { s.Timezone = "Mars/Olympus" },
		"few hours":     func(s *Settings) { s.DailyHours = 1.5 },
		"many hours":    func(s *Settings) { s.DailyHours = 4.5 },
		"odd step":      func(s *Settings) { s.DailyHours = 2.25 },
		"far start":     func(s *Settings) { s.StartDate = &far },
		"ntfy scheme":   func(s *Settings) { s.NtfyURL = "javascript:alert(1)" },
		"ntfy userinfo": func(s *Settings) { s.NtfyURL = "https://user:pass@ntfy.sh" },
		"ntfy topic":    func(s *Settings) { s.NtfyTopic = "has spaces" },
	} {
		s := DefaultSettings()
		change(&s)
		if s.Validate() == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if got := ExtraReviewSlots(3.5); got != 3 {
		t.Fatalf("extra slots at 3.5h: %d", got)
	}
}

func TestSettingsAndReviewPagesRender(t *testing.T) {
	start := day("2026-10-01")
	settings := DefaultSettings()
	settings.StartDate, settings.Revision = &start, "11111111-1111-4111-8111-111111111111"
	settings.NtfyTokenCiphertext = []byte("sealed-token-bytes")
	var out bytes.Buffer
	if err := SettingsView(SettingsPage{Settings: settings, Schedule: NewScheduleForm(settings)}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`name="start" value="2026-10-01"`, `name="end" value="2026-12-23"`, `value="America/Chicago"`, `value="2.0" selected`, settings.Revision, `action="/leetgrinder/settings/schedule"`} {
		if !strings.Contains(html, want) {
			t.Errorf("settings missing %q", want)
		}
	}
	if strings.Contains(html, "sealed-token-bytes") {
		t.Fatal("settings page rendered the token ciphertext")
	}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	state := State{Attempts: []Attempt{attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -9))}}
	today := NewToday(settings, state, []string{"two-sum"}, now)
	ids := map[string]string{ReviewKey("two-sum"): "22222222-2222-4222-8222-222222222222"}
	out.Reset()
	if err := Overview(OverviewPage{Today: today, IDs: ids}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html = out.String()
	for _, want := range []string{"20 sessions behind", "Today is Day 20 of 84.", `id="reviews-title"`, "Struggled 9 days ago", `name="review" value="true"`, `name="return" value="overview"`, ids[ReviewKey("two-sum")], `id="review-two-sum"`} {
		if !strings.Contains(html, want) {
			t.Errorf("overview missing %q", want)
		}
	}
	out.Reset()
	if err := Reviews(ReviewsPage{Today: today, Cards: BuildCards(state.Attempts, time.UTC), IDs: ids}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if html = out.String(); !strings.Contains(html, "All cards") || !strings.Contains(html, "Two Sum") || !strings.Contains(html, "Due but not planned") {
		t.Fatal("review queue missing cards")
	}
	unset := NewToday(DefaultSettings(), State{}, nil, now)
	out.Reset()
	if err := Overview(OverviewPage{Today: unset}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if html = out.String(); !strings.Contains(html, "Set your schedule") || !strings.Contains(html, "Nothing is due for review today.") {
		t.Fatal("unset schedule prompt missing")
	}
}
