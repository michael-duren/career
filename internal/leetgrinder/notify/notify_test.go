package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func settingsFor(url string) leetgrinder.Settings {
	s := leetgrinder.DefaultSettings()
	s.NtfyURL, s.NtfyTopic = url+"/", "my_topic"
	return s
}

func TestSendHeadersAndErrors(t *testing.T) {
	var got *http.Request
	var body string
	status := 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(context.Background())
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.WriteHeader(status)
		_, _ = w.Write([]byte("bad token " + r.Header.Get("Authorization") + "\nline two"))
	}))
	defer server.Close()
	settings := settingsFor(server.URL)
	box, _ := leetgrinder.NewSecretBox(testKey)
	settings.NtfyTokenCiphertext, _ = box.Seal([]byte(testToken))
	m := Message{Title: "Säule: plan", Body: "line one\nline two", Priority: "high", Tags: []string{"a", "b"}, Click: "https://app.example/leetgrinder"}
	if err := Send(context.Background(), nil, settings, testKey, m); err != nil {
		t.Fatal(err)
	}
	if got.Method != "POST" || got.URL.Path != "/my_topic" || got.Header.Get("Authorization") != "Bearer "+testToken || got.Header.Get("Priority") != "high" ||
		got.Header.Get("Tags") != "a,b" || got.Header.Get("Click") != m.Click || !strings.HasPrefix(got.Header.Get("Title"), "=?utf-8?b?") || body != m.Body {
		t.Fatalf("request: %s %s %v %q", got.Method, got.URL.Path, got.Header, body)
	}

	status = 401
	err := Send(context.Background(), nil, settings, testKey, m)
	if err == nil || !strings.Contains(err.Error(), "ntfy returned 401") || strings.Contains(err.Error(), testToken) || strings.Contains(err.Error(), "\n") {
		t.Fatalf("error: %v", err)
	}

	// No token configured: no Authorization header.
	status = 200
	settings.NtfyTokenCiphertext = nil
	if err = Send(context.Background(), nil, settings, nil, m); err != nil || got.Header.Get("Authorization") != "" {
		t.Fatalf("tokenless: %v %v", err, got.Header)
	}

	// A token without a key is an error that names the key, not the token.
	settings.NtfyTokenCiphertext, _ = box.Seal([]byte(testToken))
	if err = Send(context.Background(), nil, settings, nil, m); err == nil || !strings.Contains(err.Error(), "LEETGRINDER_SECRET_KEY") {
		t.Fatalf("missing key: %v", err)
	}
	settings.NtfyTopic = ""
	if err = Send(context.Background(), nil, settings, testKey, m); err == nil {
		t.Fatal("sent without a topic")
	}
}

func TestSendRedactsTokenAcrossCuts(t *testing.T) {
	for _, pad := range []int{200, 220, 1000, 1010, 1020} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(strings.Repeat("a", pad) + strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") + strings.Repeat("b", 50)))
		}))
		settings := settingsFor(server.URL)
		box, _ := leetgrinder.NewSecretBox(testKey)
		settings.NtfyTokenCiphertext, _ = box.Seal([]byte(testToken))
		err := Send(context.Background(), nil, settings, testKey, Message{Title: "x"})
		server.Close()
		if err == nil || strings.Contains(err.Error(), "tk_") || len([]rune(err.Error())) > 240 {
			t.Fatalf("pad %d: %v", pad, err)
		}
	}
}

func TestSendRedactsLongTokenInMultibyteBody(t *testing.T) {
	token := "tk_" + strings.Repeat("0123456789", 20)
	for _, pad := range []int{150, 160, 170, 200, 255} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(strings.Repeat("😀", pad) + strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") + strings.Repeat("😀", 100)))
		}))
		settings := settingsFor(server.URL)
		box, _ := leetgrinder.NewSecretBox(testKey)
		settings.NtfyTokenCiphertext, _ = box.Seal([]byte(token))
		err := Send(context.Background(), nil, settings, testKey, Message{Title: "x"})
		server.Close()
		if err == nil {
			t.Fatalf("pad %d: no error", pad)
		}
		for i := 0; i+8 <= len(token); i++ {
			if strings.Contains(err.Error(), token[i:i+8]) {
				t.Fatalf("pad %d: token fragment %q leaked: %v", pad, token[i:i+8], err)
			}
		}
	}
}

func TestSendTimeoutAndRedirects(t *testing.T) {
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-release }))
	defer slow.Close()
	defer close(release)
	client := NewHTTPClient()
	client.Timeout = 50 * time.Millisecond
	if err := Send(context.Background(), client, settingsFor(slow.URL), nil, Message{Title: "x"}); err == nil || err.Error() != "ntfy request timed out" {
		t.Fatalf("timeout: %v", err)
	}
	if NewHTTPClient().Timeout != 10*time.Second {
		t.Fatal("default timeout is not 10s")
	}

	hit := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer target.Close()
	redirect := httptest.NewServer(http.RedirectHandler(target.URL, http.StatusTemporaryRedirect))
	defer redirect.Close()
	if err := Send(context.Background(), nil, settingsFor(redirect.URL), nil, Message{Title: "x"}); err == nil || hit {
		t.Fatalf("redirect followed: %v %v", err, hit)
	}
}

func TestCompose(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 9, 10, 17, 0, 0, 0, loc)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	settings := leetgrinder.DefaultSettings()
	settings.StartDate = &start
	two, _ := leetgrinder.FindProblem("two-sum")
	today := leetgrinder.NewToday(settings, leetgrinder.State{CompletedDays: []int{1, 2, 3, 4, 5, 6, 7}}, nil, now)
	today.Reviews = []leetgrinder.ReviewItem{{Slot: 1, Problem: two}}
	pref := func(kind string, threshold int) leetgrinder.NotificationPref {
		p := settings.NotificationPref(kind)
		p.Threshold = threshold
		return p
	}
	origin := "https://app.example"

	m, ok := Compose(leetgrinder.NotifyMissingWork, pref(leetgrinder.NotifyMissingWork, 0), today, nil, origin)
	if !ok || m.Body != "Session 10: Lower and upper boundaries\nReview: Two Sum" || m.Click != origin+"/leetgrinder/day/10" || m.Priority != "default" {
		t.Fatalf("missing: %+v", m)
	}
	if m, ok = Compose(leetgrinder.NotifyLateEscalation, pref(leetgrinder.NotifyLateEscalation, 0), today, nil, origin); !ok || m.Priority != "high" {
		t.Fatalf("late: %+v", m)
	}
	// Three behind (7 of 10) meets a threshold of 3 but not 4.
	if m, ok = Compose(leetgrinder.NotifyBehindSchedule, pref(leetgrinder.NotifyBehindSchedule, 3), today, nil, origin); !ok || m.Title != "Leetgrinder: 3 sessions behind" {
		t.Fatalf("behind: %+v", m)
	}
	if _, ok = Compose(leetgrinder.NotifyBehindSchedule, pref(leetgrinder.NotifyBehindSchedule, 4), today, nil, origin); ok {
		t.Fatal("behind below threshold")
	}
	m, ok = Compose(leetgrinder.NotifyMorningPlan, pref(leetgrinder.NotifyMorningPlan, 0), today, nil, origin)
	if !ok || !strings.HasPrefix(m.Body, "Session 10: Lower and upper boundaries\n") || !strings.HasSuffix(m.Body, "\nReview: Two Sum") {
		t.Fatalf("morning: %q", m.Body)
	}
	day10, _ := leetgrinder.FindDay(10)
	for _, r := range day10.Readings {
		if !r.Optional && !strings.Contains(m.Body, "Reading: "+r.Title) {
			t.Fatalf("morning misses reading %q: %q", r.Title, m.Body)
		}
	}
	due := []leetgrinder.Card{{Problem: two, Due: now.Add(-time.Hour)}}
	for _, slug := range []string{"contains-duplicate", "valid-anagram"} {
		p, _ := leetgrinder.FindProblem(slug)
		due = append(due, leetgrinder.Card{Problem: p, Due: now.Add(-time.Hour)})
	}
	if m, ok = Compose(leetgrinder.NotifyReviewBacklog, pref(leetgrinder.NotifyReviewBacklog, 2), today, due, origin); !ok || m.Title != "Leetgrinder: 2 reviews waiting" || m.Click != origin+"/leetgrinder/reviews" {
		t.Fatalf("backlog: %+v", m)
	}
	if _, ok = Compose(leetgrinder.NotifyReviewBacklog, pref(leetgrinder.NotifyReviewBacklog, 3), today, due, origin); ok {
		t.Fatal("backlog below threshold")
	}

	// Finished work: missing and late have nothing to say.
	done := leetgrinder.NewToday(settings, leetgrinder.State{CompletedDays: []int{10}}, nil, now)
	for _, kind := range []string{leetgrinder.NotifyMissingWork, leetgrinder.NotifyLateEscalation} {
		if _, ok = Compose(kind, pref(kind, 0), done, nil, origin); ok {
			t.Fatalf("%s sent with nothing missing", kind)
		}
	}
	if _, ok = Compose("unknown", leetgrinder.NotificationPref{}, today, nil, origin); ok {
		t.Fatal("unknown kind composed")
	}
}
