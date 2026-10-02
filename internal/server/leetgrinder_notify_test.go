package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderNtfySettings(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	s.config.LeetgrinderSecretKey = bytes.Repeat([]byte{5}, 32)
	const token = "tk_never_render_me"
	revision := func() string {
		settings, err := db.LeetgrinderSettings(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return settings.Revision
	}

	w := request("GET", "/leetgrinder/settings", nil)
	body := w.Body.String()
	for _, want := range []string{"No topic set", "No reminders will be sent", `id="ntfy"`, `id="notifications"`, `id="notification-log"`, "No token set.", "No notifications yet.", `name="goal_incomplete_enabled" value="true" checked`, `name="streak_at_risk_threshold" min="0" max="365" required value="1"`, `name="streak_at_risk_time" required value="21:00"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("settings page lacks %q", want)
		}
	}

	save := func(values url.Values) *httptest.ResponseRecorder {
		if values.Get("revision") == "" {
			values.Set("revision", revision())
		}
		return request("POST", "/leetgrinder/settings/ntfy", values)
	}
	if w = save(url.Values{"url": {"https://ntfy.example/"}, "topic": {"grind"}, "token": {token}}); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/settings?saved=ntfy#ntfy" {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	settings, _ := db.LeetgrinderSettings(ctx)
	box, _ := leetgrinder.NewSecretBox(s.config.LeetgrinderSecretKey)
	if plain, err := box.Open(settings.NtfyTokenCiphertext); err != nil || string(plain) != token || settings.NtfyURL != "https://ntfy.example" || settings.NtfyTopic != "grind" {
		t.Fatalf("stored: %q %v %+v", plain, err, settings)
	}
	w = request("GET", "/leetgrinder/settings", nil)
	if body = w.Body.String(); strings.Contains(body, token) || strings.Contains(body, "No topic set") || !strings.Contains(body, "Token set.") || !strings.Contains(body, "Clear the saved token") {
		t.Fatal("token rendered or status missing")
	}

	// Blank token keeps the saved one; rejected drafts keep the URL and topic but never the token.
	if w = save(url.Values{"url": {"https://ntfy.example"}, "topic": {"other"}}); w.Code != 303 {
		t.Fatalf("keep token: %d", w.Code)
	}
	if settings, _ = db.LeetgrinderSettings(ctx); !settings.TokenSet() {
		t.Fatal("blank token cleared the saved token")
	}
	for _, test := range []struct {
		values url.Values
		status int
		want   string
	}{
		{url.Values{"url": {"ftp://ntfy.example"}, "topic": {"draft-topic"}, "token": {token}}, 400, "ntfy server must be an absolute http(s) URL"},
		{url.Values{"url": {"https://ntfy.example"}, "topic": {"has space"}}, 400, "ntfy topic may use"},
		{url.Values{"url": {"https://ntfy.example"}, "topic": {"draft-topic"}, "token": {token}, "clear_token": {"true"}}, 400, "not both"},
		{url.Values{"url": {"https://ntfy.example"}, "topic": {"draft-topic"}, "token": {"has space"}}, 400, "without spaces"},
		{url.Values{"url": {"https://ntfy.example"}, "topic": {"draft-topic"}, "token": {token}, "revision": {"00000000-0000-4000-8000-000000000000"}}, 409, "Settings changed"},
	} {
		w = save(test.values)
		body = w.Body.String()
		if w.Code != test.status || !strings.Contains(body, test.want) || strings.Contains(body, token) || !strings.Contains(body, `value="`+test.values.Get("topic")+`"`) {
			t.Errorf("%v: %d %s", test.values, w.Code, body)
		}
	}

	// A saved token never follows a changed server unless re-entered.
	if w = save(url.Values{"url": {"https://attacker.example"}, "topic": {"grind"}}); w.Code != 400 || !strings.Contains(w.Body.String(), "Re-enter the access token") {
		t.Fatalf("host change kept token: %d", w.Code)
	}
	if w = save(url.Values{"url": {"HTTPS://NTFY.example/"}, "topic": {"grind"}}); w.Code != 303 {
		t.Fatalf("same host rejected: %d %s", w.Code, w.Body.String())
	}
	if w = save(url.Values{"url": {"https://ntfy2.example"}, "topic": {"grind"}, "token": {token}}); w.Code != 303 {
		t.Fatalf("host change with token: %d", w.Code)
	}

	// Clearing removes the ciphertext.
	if w = save(url.Values{"url": {"https://ntfy.example"}, "topic": {"grind"}, "clear_token": {"true"}}); w.Code != 303 {
		t.Fatalf("clear: %d", w.Code)
	}
	if settings, _ = db.LeetgrinderSettings(ctx); settings.TokenSet() {
		t.Fatal("token not cleared")
	}

	// Without a secret key, a token cannot be saved but other fields can.
	s.config.LeetgrinderSecretKey = nil
	if w = save(url.Values{"url": {"https://ntfy.example"}, "topic": {"grind"}, "token": {token}}); w.Code != 400 || !strings.Contains(w.Body.String(), "LEETGRINDER_SECRET_KEY is not configured") || strings.Contains(w.Body.String(), token) {
		t.Fatalf("no key: %d %s", w.Code, w.Body.String())
	}
	if w = save(url.Values{"url": {"https://ntfy.example"}, "topic": {"grind2"}}); w.Code != 303 {
		t.Fatalf("no key, no token: %d", w.Code)
	}

	// Cross-origin writes are refused.
	r := httptest.NewRequest("POST", "/leetgrinder/settings/ntfy/test", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
	r.Header.Set("Origin", "https://evil.com")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.RegisterRoutes().ServeHTTP(rec, r)
	if rec.Code != 403 {
		t.Fatalf("cross-origin test: %d", rec.Code)
	}
}

func TestLeetgrinderNotificationSettings(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	values := func() url.Values {
		settings, _ := db.LeetgrinderSettings(ctx)
		v := url.Values{"revision": {settings.Revision}}
		for _, k := range leetgrinder.NotificationKinds {
			v.Set(k.Key+"_time", k.Default.Time)
			v.Set(k.Key+"_priority", k.Default.Priority)
			if k.HasThreshold() {
				v.Set(k.Key+"_threshold", "5")
			}
		}
		return v
	}
	v := values()
	v.Set("morning_plan_enabled", "true")
	v.Set("morning_plan_time", "07:30")
	v.Set("streak_at_risk_priority", "urgent")
	v.Set("streak_at_risk_threshold", "0")
	if w := request("POST", "/leetgrinder/settings/notifications", v); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/settings?saved=notifications#notifications" {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	settings, _ := db.LeetgrinderSettings(ctx)
	if p := settings.NotificationPref(leetgrinder.NotifyMorningPlan); !p.Enabled || p.Time != "07:30" {
		t.Fatalf("morning: %+v", p)
	}
	if p := settings.NotificationPref(leetgrinder.NotifyGoalIncomplete); p.Enabled {
		t.Fatal("unchecked kind stayed enabled")
	}
	if p := settings.NotificationPref(leetgrinder.NotifyReviewBacklog); p.Threshold != 5 {
		t.Fatalf("backlog: %+v", p)
	}
	if p := settings.NotificationPref(leetgrinder.NotifyStreakAtRisk); p.Priority != "urgent" || p.Threshold != 0 {
		t.Fatalf("streak: %+v", p)
	}

	for name, change := range map[string]func(url.Values){
		"threshold":        func(v url.Values) { v.Set("review_backlog_threshold", "0") },
		"streak threshold": func(v url.Values) { v.Set("streak_at_risk_threshold", "-1") },
		"time":             func(v url.Values) { v.Set("goal_incomplete_time", "5pm") },
		"priority":         func(v url.Values) { v.Set("goal_incomplete_priority", "loud") },
	} {
		v := values()
		v.Set("streak_at_risk_threshold", "42")
		change(v)
		w := request("POST", "/leetgrinder/settings/notifications", v)
		if w.Code != 400 || !strings.Contains(w.Body.String(), `name="streak_at_risk_threshold" min="0" max="365" required value="`+v.Get("streak_at_risk_threshold")+`"`) {
			t.Errorf("%s: %d draft not kept", name, w.Code)
		}
	}
	if after, _ := db.LeetgrinderSettings(ctx); after.Revision != settings.Revision {
		t.Fatal("rejected save changed settings")
	}
}

func TestLeetgrinderTestNotification(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	s.config.LeetgrinderSecretKey = bytes.Repeat([]byte{5}, 32)
	s.now = func() time.Time { return time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC) }
	const token = "tk_test_button_secret"
	var mu sync.Mutex
	var got []http.Header
	status := 200
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, r.Header.Clone())
		w.WriteHeader(status)
		_, _ = w.Write([]byte(r.Header.Get("Authorization")))
	}))
	defer ntfy.Close()

	// With no topic, the test fails with a clear message and sends nothing.
	w := request("POST", "/leetgrinder/settings/ntfy/test", url.Values{})
	if w.Code != 502 || !strings.Contains(w.Body.String(), "set an ntfy topic first") || len(got) != 0 {
		t.Fatalf("no topic: %d %s", w.Code, w.Body.String())
	}

	settings, _ := db.LeetgrinderSettings(ctx)
	if w = request("POST", "/leetgrinder/settings/ntfy", url.Values{"revision": {settings.Revision}, "url": {ntfy.URL}, "topic": {"grind"}, "token": {token}}); w.Code != 303 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	if w = request("POST", "/leetgrinder/settings/ntfy/test", url.Values{}); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/settings?tested=1#ntfy" {
		t.Fatalf("test: %d %s", w.Code, w.Body.String())
	}
	if len(got) != 1 || got[0].Get("Authorization") != "Bearer "+token || got[0].Get("Title") != "Leetgrinder test notification" || got[0].Get("Click") != "https://example.com/leetgrinder/settings" {
		t.Fatalf("sent: %v", got)
	}
	w = request("GET", "/leetgrinder/settings?tested=1", nil)
	body := w.Body.String()
	if !strings.Contains(body, "Test notification sent.") || !strings.Contains(body, "<td>Test</td><td>Sent</td>") || strings.Contains(body, token) {
		t.Fatalf("after test: %s", body)
	}

	mu.Lock()
	status = 403
	mu.Unlock()
	w = request("POST", "/leetgrinder/settings/ntfy/test", url.Values{})
	body = w.Body.String()
	if w.Code != 502 || !strings.Contains(body, "The test notification failed: ntfy returned 403") || strings.Contains(body, token) || !strings.Contains(body, "<td>Test</td><td>Failed</td><td>3</td>") {
		t.Fatalf("failed test: %d %s", w.Code, body)
	}
	log, err := db.RecentLeetgrinderNotifications(ctx, 14)
	if err != nil || len(log) != 1 || strings.Contains(log[0].Detail, token) {
		t.Fatalf("log: %+v %v", log, err)
	}
}

func TestLeetgrinderExportCompleteness(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	s.config.LeetgrinderSecretKey = bytes.Repeat([]byte{7}, 32)
	const token = "tk_export_secret_value"
	settings, _ := db.LeetgrinderSettings(ctx)
	if w := request("POST", "/leetgrinder/settings/ntfy", url.Values{"url": {"https://ntfy.example"}, "topic": {"grind"}, "token": {token}, "revision": {settings.Revision}}); w.Code != 303 {
		t.Fatalf("ntfy: %d %s", w.Code, w.Body.String())
	}
	settings, _ = db.LeetgrinderSettings(ctx)
	if len(settings.NtfyTokenCiphertext) == 0 {
		t.Fatal("token not stored")
	}
	set, err := db.CreateLeetgrinderTodoSetDetailed(ctx, leetgrinder.TodoSet{Title: "Warmup", Description: "easy ones", Metadata: map[string]any{"source": "test"}}, []leetgrinder.TodoProblemInput{
		{Slug: "two-sum", Number: 1, Title: "Two Sum", Difficulty: "Easy", Topics: []string{"array"}},
		{Slug: "valid-anagram", Reference: "https://leetcode.com/problems/valid-anagram/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.AddLeetgrinderTodoProblem(ctx, "", leetgrinder.TodoProblemInput{Slug: "lonely-problem"}); err != nil {
		t.Fatal(err)
	}
	// Picks are stored in slot order, which is not alphabetical.
	for slot, slug := range []string{"valid-anagram", "two-sum"} {
		if _, err = db.DB.Exec("INSERT INTO leetgrinder_review_plan(plan_date,problem_slug,slot) VALUES('2026-10-02',$1,$2)", slug, slot+1); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.DB.Exec("INSERT INTO leetgrinder_review_plan(plan_date,problem_slug,slot) VALUES('2026-09-30','two-sum',1)"); err != nil {
		t.Fatal(err)
	}

	w := request("GET", "/leetgrinder/export", nil)
	if w.Code != 200 {
		t.Fatalf("export: %d", w.Code)
	}
	raw := w.Body.String()
	for _, secret := range []string{token, "ntfyToken", "ntfy_token", "ciphertext", "Ciphertext", base64.StdEncoding.EncodeToString(settings.NtfyTokenCiphertext)} {
		if strings.Contains(raw, secret) {
			t.Fatalf("export leaks %q", secret)
		}
	}
	var exported struct {
		Version     int                     `json:"version"`
		Attempts    []json.RawMessage       `json:"attempts"`
		Problems    []struct{ Slug string } `json:"problems"`
		DailyGoals  []json.RawMessage       `json:"dailyGoals"`
		ReviewPlans map[string][]string     `json:"reviewPlans"`
		Todos       struct {
			Sets []struct {
				ID, Title, Description string
				Metadata               map[string]any
				CreatedAt              time.Time
				Items                  []struct {
					Slug       string
					SourceData map[string]any
					CreatedAt  time.Time
				}
			}
			Items []struct{ Slug string }
		} `json:"todos"`
		Settings struct {
			Timezone        string
			Goal            struct{ New, Review int }
			NtfyURL         string `json:"ntfyUrl"`
			NtfyTopic       string
			Notifications   map[string]leetgrinder.NotificationPref
			AnalysisEnabled bool
		} `json:"settings"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &exported); err != nil {
		t.Fatal(err)
	}
	if exported.Version != 2 || exported.Attempts == nil || exported.DailyGoals == nil {
		t.Fatalf("legacy fields or version: %s", raw)
	}
	if !slices.Equal(exported.ReviewPlans["2026-10-02"], []string{"valid-anagram", "two-sum"}) || !slices.Equal(exported.ReviewPlans["2026-09-30"], []string{"two-sum"}) || len(exported.ReviewPlans) != 2 {
		t.Fatalf("reviewPlans: %v", exported.ReviewPlans)
	}
	if strings.Index(raw, `"2026-09-30"`) > strings.Index(raw, `"2026-10-02"`) {
		t.Fatal("review plan dates not in order")
	}
	if len(exported.Todos.Sets) != 1 || exported.Todos.Sets[0].ID != set.ID || exported.Todos.Sets[0].Title != "Warmup" || exported.Todos.Sets[0].Description != "easy ones" || exported.Todos.Sets[0].Metadata["source"] != "test" || exported.Todos.Sets[0].CreatedAt.IsZero() {
		t.Fatalf("todo sets: %+v", exported.Todos.Sets)
	}
	items := exported.Todos.Sets[0].Items
	if len(items) != 2 || items[0].Slug != "two-sum" || items[1].Slug != "valid-anagram" || items[0].CreatedAt.IsZero() || items[0].SourceData["title"] != "Two Sum" || items[1].SourceData["reference"] != "https://leetcode.com/problems/valid-anagram/" {
		t.Fatalf("todo items: %+v", items)
	}
	if len(exported.Todos.Items) != 1 || exported.Todos.Items[0].Slug != "lonely-problem" {
		t.Fatalf("standalone todos: %+v", exported.Todos.Items)
	}
	var slugs []string
	for _, p := range exported.Problems {
		slugs = append(slugs, p.Slug)
	}
	if !slices.Equal(slugs, []string{"lonely-problem", "two-sum", "valid-anagram"}) {
		t.Fatalf("problems: %v", slugs)
	}
	got := exported.Settings
	if got.Timezone != settings.Timezone || got.Goal.New != settings.Goal.New || got.Goal.Review != settings.Goal.Review || got.NtfyURL != "https://ntfy.example" || got.NtfyTopic != "grind" || !got.AnalysisEnabled || got.Notifications == nil {
		t.Fatalf("settings: %+v", got)
	}
}
