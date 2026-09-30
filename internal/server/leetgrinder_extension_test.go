package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderAPIRequiresToken(t *testing.T) {
	// No database: every request must be rejected before storage is touched.
	s := &Server{config: config.Config{Username: "admin", JWTSecret: "test", PublicOrigin: "https://example.com"}}
	handler := s.RegisterRoutes()
	for _, test := range []struct {
		method, path, auth string
	}{
		{"GET", "/api/leetgrinder/problem/two-sum", ""},
		{"POST", "/api/leetgrinder/attempts", ""},
		{"POST", "/api/leetgrinder/attempts", "Basic abc"},
		{"POST", "/api/leetgrinder/attempts", "Bearer not-a-token"},
		{"GET", "/api/leetgrinder/problem/two-sum", "Bearer"},
	} {
		r := httptest.NewRequest(test.method, test.path, strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		// A valid session cookie must not substitute for a token.
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		r.Header.Set("Origin", "https://example.com")
		if test.auth != "" {
			r.Header.Set("Authorization", test.auth)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 401 || !strings.HasPrefix(w.Header().Get("WWW-Authenticate"), "Bearer") || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
			t.Errorf("%s %s %q: %d %s", test.method, test.path, test.auth, w.Code, w.Body.String())
		}
	}
	// Token management needs the session and a same-origin form.
	for _, path := range []string{"/leetgrinder/settings/tokens", "/leetgrinder/settings/tokens/" + uuid.NewString() + "/revoke"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader("name=x")))
		if w.Code != 303 {
			t.Errorf("%s without session: %d", path, w.Code)
		}
		r := httptest.NewRequest("POST", path, strings.NewReader("name=x"))
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		r.Header.Set("Origin", "https://evil.com")
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Errorf("%s cross-origin: %d", path, w.Code)
		}
	}
}

var createdToken = regexp.MustCompile(`value="(lg_[A-Za-z0-9_-]{43})"`)

func TestLeetgrinderExtensionAPI(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	chicago, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 10, 10, 20, 0, 0, 0, chicago)
	s.now = func() time.Time { return now }
	handler := s.RegisterRoutes()

	// Create a token through the settings page; the plaintext appears once.
	w := request("POST", "/leetgrinder/settings/tokens", url.Values{"name": {"Firefox <laptop>"}})
	match := createdToken.FindStringSubmatch(w.Body.String())
	if w.Code != 200 || match == nil || !strings.Contains(w.Body.String(), "Firefox &lt;laptop&gt;") || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("create token: %d %s", w.Code, w.Body.String())
	}
	plain := match[1]
	w = request("GET", "/leetgrinder/settings", nil)
	if w.Code != 200 || strings.Contains(w.Body.String(), plain) || !strings.Contains(w.Body.String(), "Firefox &lt;laptop&gt;") || !strings.Contains(w.Body.String(), "Never") {
		t.Fatalf("settings after create: %d", w.Code)
	}
	if w = request("POST", "/leetgrinder/settings/tokens", url.Values{"name": {"  "}}); w.Code != 400 || createdToken.MatchString(w.Body.String()) {
		t.Fatalf("blank name: %d", w.Code)
	}

	api := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var payload string
		if body != nil {
			b, _ := json.Marshal(body)
			payload = string(b)
		}
		r := httptest.NewRequest(method, path, strings.NewReader(payload))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}

	// Any slug is accepted; an unknown one is reported as new and not known.
	w = api("GET", "/api/leetgrinder/problem/not-a-problem", plain, nil)
	var info leetgrinderAPIProblem
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &info) != nil || info.Known || info.Status != "new" || info.HistoryURL != "/leetgrinder/problem/not-a-problem" {
		t.Fatalf("unknown problem: %d %s", w.Code, w.Body.String())
	}
	if w = api("GET", "/api/leetgrinder/problem/Not_A_Slug", plain, nil); w.Code != 400 {
		t.Fatalf("bad slug: %d", w.Code)
	}
	w = api("GET", "/api/leetgrinder/problem/two-sum", plain, nil)
	info = leetgrinderAPIProblem{}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &info) != nil || !info.Known || info.Title != "Two Sum" || info.Number != 1 || info.Difficulty != "Easy" || info.Status != "new" || info.Latest != nil || info.TodaysPick {
		t.Fatalf("two-sum: %d %s", w.Code, w.Body.String())
	}
	for _, gone := range []string{"inCurriculum", "session", "week", "todaysReview", "reviewDone"} {
		if strings.Contains(w.Body.String(), `"`+gone+`"`) {
			t.Errorf("response still has %q", gone)
		}
	}

	// Metadata from the extension fills in an unknown problem.
	meta := map[string]any{"number": 146, "title": "LRU Cache", "difficulty": "Medium", "topics": []map[string]string{{"slug": "design", "name": "Design"}, {"slug": "hash-table", "name": "Hash Table"}}}
	if w = api("PUT", "/api/leetgrinder/problem/design-lru", plain, meta); w.Code != 204 {
		t.Fatalf("put metadata: %d %s", w.Code, w.Body.String())
	}
	w = api("GET", "/api/leetgrinder/problem/design-lru", plain, nil)
	info = leetgrinderAPIProblem{}
	if json.Unmarshal(w.Body.Bytes(), &info) != nil || !info.Known || info.Title != "LRU Cache" || info.Number != 146 || strings.Join(info.Topics, ",") != "design,hash-table" {
		t.Fatalf("metadata not stored: %s", w.Body.String())
	}
	for _, bad := range []map[string]any{
		{"difficulty": "Trivial"},
		{"number": -3},
		{"title": strings.Repeat("x", 201)},
		{"topics": []map[string]string{{"slug": "Not A Slug", "name": "x"}}},
		{"unknown": true},
	} {
		if w = api("PUT", "/api/leetgrinder/problem/design-lru", plain, bad); w.Code != 400 {
			t.Errorf("bad metadata %v: %d", bad, w.Code)
		}
	}
	if w = api("PUT", "/api/leetgrinder/problem/Bad_Slug", plain, meta); w.Code != 400 {
		t.Errorf("bad slug metadata: %d", w.Code)
	}

	id := uuid.NewString()
	attempt := map[string]any{"id": id, "problemSlug": "two-sum", "outcome": "solved", "minutes": 18, "assisted": false, "notes": "hash map", "timeComplexity": "O(n)", "spaceComplexity": "O(n)"}
	for i := 0; i < 2; i++ {
		if w = api("POST", "/api/leetgrinder/attempts", plain, attempt); w.Code != 200 {
			t.Fatalf("save %d: %d %s", i, w.Code, w.Body.String())
		}
	}
	state, err := db.LeetgrinderState(ctx)
	if err != nil || len(state.Attempts) != 1 || state.Attempts[0].Source != "extension" || state.Attempts[0].IsReview || state.Attempts[0].Minutes != 18 {
		t.Fatalf("idempotent save: %+v %v", state, err)
	}
	w = api("GET", "/api/leetgrinder/problem/two-sum", plain, nil)
	info = leetgrinderAPIProblem{}
	if json.Unmarshal(w.Body.Bytes(), &info) != nil || info.Latest == nil || info.Latest.ID != id {
		t.Fatalf("latest attempt: %s", w.Body.String())
	}

	conflict := map[string]any{"id": id, "problemSlug": "two-sum", "outcome": "struggled", "minutes": 30, "assisted": false, "notes": "hash map", "timeComplexity": "O(n)", "spaceComplexity": "O(n)"}
	if w = api("POST", "/api/leetgrinder/attempts", plain, conflict); w.Code != 409 {
		t.Fatalf("reused id: %d %s", w.Code, w.Body.String())
	}
	review := map[string]any{"id": uuid.NewString(), "problemSlug": "valid-anagram", "outcome": "unfinished", "minutes": 25, "assisted": true, "notes": "", "isReview": true}
	if w = api("POST", "/api/leetgrinder/attempts", plain, review); w.Code != 200 {
		t.Fatalf("review save: %d %s", w.Code, w.Body.String())
	}
	state, _ = db.LeetgrinderState(ctx)
	if len(state.Attempts) != 2 || !state.Attempts[0].IsReview || state.Attempts[0].Source != "extension" || !state.Attempts[0].Assisted {
		t.Fatalf("review flag: %+v", state.Attempts)
	}
	// An attempt on any problem is saved, with its metadata.
	other := map[string]any{"id": uuid.NewString(), "problemSlug": "min-cost-to-connect-all-points-ii", "outcome": "unfinished", "minutes": 40, "assisted": false, "notes": "",
		"problem": map[string]any{"number": 9999, "title": "Min Cost II", "difficulty": "Hard", "topics": []map[string]string{{"slug": "graph", "name": "Graph"}}}}
	if w = api("POST", "/api/leetgrinder/attempts", plain, other); w.Code != 200 {
		t.Fatalf("any-problem save: %d %s", w.Code, w.Body.String())
	}
	if p, _ := db.LeetgrinderProblem(ctx, "min-cost-to-connect-all-points-ii"); p.Title != "Min Cost II" || p.Number != 9999 || p.Difficulty != "Hard" {
		t.Fatalf("attempt metadata not stored: %+v", p)
	}

	for _, test := range []struct {
		body any
		raw  string
		want int
	}{
		{body: map[string]any{"id": uuid.NewString(), "problemSlug": "Not A Slug", "outcome": "unfinished", "minutes": 10}, want: 400},
		{body: map[string]any{"id": uuid.NewString(), "problemSlug": "two-sum", "outcome": "unfinished", "minutes": 10, "problem": map[string]any{"difficulty": "Trivial"}}, want: 400},
		{body: map[string]any{"id": "nope", "problemSlug": "two-sum", "outcome": "solved", "minutes": 10}, want: 400},
		{body: map[string]any{"id": uuid.NewString(), "problemSlug": "two-sum", "outcome": "great", "minutes": 10}, want: 400},
		{body: map[string]any{"id": uuid.NewString(), "problemSlug": "two-sum", "outcome": "solved", "minutes": 0}, want: 400},
		{body: map[string]any{"id": uuid.NewString(), "problemSlug": "two-sum", "outcome": "solved", "minutes": 241}, want: 400},
		{body: map[string]any{"id": uuid.NewString(), "problemSlug": "two-sum", "outcome": "solved", "minutes": 10, "notes": strings.Repeat("a", 2001)}, want: 400},
		{body: map[string]any{"id": uuid.NewString(), "problemSlug": "two-sum", "outcome": "solved", "minutes": 10, "source": "web"}, want: 400},
		{raw: `{"id":`, want: 400},
		{raw: `{"notes":"` + strings.Repeat("a", leetgrinderAPIBodyLimit) + `"}`, want: 413},
	} {
		payload := test.raw
		if test.body != nil {
			b, _ := json.Marshal(test.body)
			payload = string(b)
		}
		r := httptest.NewRequest("POST", "/api/leetgrinder/attempts", strings.NewReader(payload))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+plain)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != test.want {
			t.Errorf("%.80s: %d %s", payload, w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("POST", "/api/leetgrinder/attempts", strings.NewReader("id=x"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Authorization", "Bearer "+plain)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatalf("form body: %d", w.Code)
	}

	// Revoke through the settings page; the token stops working at once.
	tokens, err := db.LeetgrinderTokens(ctx)
	if err != nil || len(tokens) != 1 || tokens[0].LastUsedAt == nil {
		t.Fatalf("tokens: %+v %v", tokens, err)
	}
	if w = request("POST", "/leetgrinder/settings/tokens/"+tokens[0].ID+"/revoke", url.Values{}); w.Code != 303 {
		t.Fatalf("revoke: %d %s", w.Code, w.Body.String())
	}
	if w = api("GET", "/api/leetgrinder/problem/two-sum", plain, nil); w.Code != 401 {
		t.Fatalf("revoked token: %d", w.Code)
	}
	if w = api("POST", "/api/leetgrinder/attempts", plain, map[string]any{"id": uuid.NewString(), "problemSlug": "two-sum", "outcome": "solved", "minutes": 10}); w.Code != 401 {
		t.Fatalf("revoked token save: %d", w.Code)
	}
	if w = request("GET", "/leetgrinder/settings", nil); !strings.Contains(w.Body.String(), "Revoked") || strings.Contains(w.Body.String(), plain) {
		t.Fatal("revoked token not listed")
	}
	if w = request("POST", "/leetgrinder/settings/tokens/"+uuid.NewString()+"/revoke", url.Values{}); w.Code != 404 {
		t.Fatalf("missing token revoke: %d", w.Code)
	}
}

func TestLeetgrinderTokenShownWhenSettingsFail(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	if _, err := db.DB.Exec("DELETE FROM leetgrinder_settings"); err != nil {
		t.Fatal(err)
	}
	w := request("POST", "/leetgrinder/settings/tokens", url.Values{"name": {"Chrome"}})
	if w.Code != 200 || !createdToken.MatchString(w.Body.String()) || !strings.Contains(w.Body.String(), "could not be loaded") {
		t.Fatalf("created token hidden by settings failure: %d %s", w.Code, w.Body.String())
	}
}

func TestLeetgrinderAPIReviewStatus(t *testing.T) {
	s, db, _ := leetgrinderTestServer(t)
	ctx := context.Background()
	chicago, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 10, 10, 20, 0, 0, 0, chicago)
	s.now = func() time.Time { return now }
	handler := s.RegisterRoutes()
	// An unfinished attempt long ago makes two-sum due for review today.
	past := now.AddDate(0, 0, -30)
	if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 25}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.ExecContext(ctx, "UPDATE leetgrinder_attempts SET created_at=$1", past); err != nil {
		t.Fatal(err)
	}
	_, plain, err := db.CreateLeetgrinderToken(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	get := func() leetgrinderAPIProblem {
		r := httptest.NewRequest("GET", "/api/leetgrinder/problem/two-sum", nil)
		r.Header.Set("Authorization", "Bearer "+plain)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var info leetgrinderAPIProblem
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &info) != nil {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
		return info
	}
	if info := get(); info.Status != "due" || !info.TodaysPick || info.AttemptedToday || info.Latest == nil || info.Recall == nil || *info.Recall <= 0 || info.DueDate == "" || info.LastAttemptedAt == nil {
		t.Fatalf("due review not reported: %+v", info)
	}
	review := uuid.NewString()
	if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: review, ProblemSlug: "two-sum", Outcome: "solved", Minutes: 12, Source: "extension", IsReview: true, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.ExecContext(ctx, "UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", now.Add(-time.Hour), review); err != nil {
		t.Fatal(err)
	}
	// Still today's pick, now attempted today.
	if info := get(); !info.AttemptedToday || !info.TodaysPick {
		t.Fatalf("done review not reported: %+v", info)
	}
	// The next day it is no longer due.
	now = now.AddDate(0, 0, 1)
	if info := get(); info.Status != "notDue" || info.TodaysPick || info.NextDue == "" || info.DueDate != "" {
		t.Fatalf("not-due status: %+v", info)
	}
}
