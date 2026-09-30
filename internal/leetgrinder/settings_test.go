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
	for _, g := range []DailyGoal{{0, 1}, {1, 0}, {10, 10}, {2, 1}} {
		if err := g.Validate(); err != nil {
			t.Errorf("rejected %+v: %v", g, err)
		}
	}
	for name, change := range map[string]func(*Settings){
		"local zone":    func(s *Settings) { s.Timezone = "Local" },
		"unknown zone":  func(s *Settings) { s.Timezone = "Mars/Olympus" },
		"no goal":       func(s *Settings) { s.Goal = DailyGoal{} },
		"many new":      func(s *Settings) { s.Goal.New = 11 },
		"negative":      func(s *Settings) { s.Goal.Review = -1 },
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
}

func TestSettingsAndReviewPagesRender(t *testing.T) {
	settings := DefaultSettings()
	settings.Revision = "11111111-1111-4111-8111-111111111111"
	settings.NtfyTokenCiphertext = []byte("sealed-token-bytes")
	var out bytes.Buffer
	if err := SettingsView(SettingsPage{Settings: settings, General: NewGeneralForm(settings)}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`value="America/Chicago"`, `name="goalNew"`, `<option value="2" selected>2</option>`, `<option value="1" selected>1</option>`, "Changes apply from today.", settings.Revision, `action="/leetgrinder/settings/general"`} {
		if !strings.Contains(html, want) {
			t.Errorf("settings missing %q", want)
		}
	}
	if strings.Contains(html, "sealed-token-bytes") || strings.Contains(html, "/leetgrinder/settings/schedule") {
		t.Fatal("settings page rendered the token ciphertext or the retired schedule form")
	}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	state := State{Problems: testProblems, Attempts: []Attempt{attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -9))}}
	state.Plans = map[time.Time][]string{Date(now, time.UTC): {"two-sum"}}
	settings.Timezone = "UTC"
	today := NewToday(settings, state, now)
	ids := map[string]string{ReviewKey("two-sum"): "22222222-2222-4222-8222-222222222222"}
	out.Reset()
	if err := Overview(OverviewPage{Today: today, IDs: ids}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html = out.String()
	for _, want := range []string{`id="reviews-title"`, "Struggled 9 days ago", `name="review" value="true"`, `name="return" value="overview"`, ids[ReviewKey("two-sum")], `id="review-two-sum"`, "LEETCODE 1"} {
		if !strings.Contains(html, want) {
			t.Errorf("overview missing %q", want)
		}
	}
	out.Reset()
	if err := Reviews(ReviewsPage{Today: today, IDs: ids}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if html = out.String(); !strings.Contains(html, "All cards") || !strings.Contains(html, "Two Sum") || !strings.Contains(html, "Backlog") {
		t.Fatal("review queue missing cards")
	}
}
