package notify

import (
	"context"
	"fmt"
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
	date := leetgrinder.Date(now, loc)
	settings := leetgrinder.DefaultSettings()
	at := func(daysAgo int) time.Time { return now.AddDate(0, 0, -daysAgo) }
	a := func(slug, outcome string, when time.Time) leetgrinder.Attempt {
		return leetgrinder.Attempt{ID: slug + when.String(), ProblemSlug: slug, Outcome: outcome, Minutes: 20, CreatedAt: when}
	}
	problems := map[string]leetgrinder.Problem{"two-sum": {Slug: "two-sum", Title: "Two Sum"}}
	// Goal 1 new + 2 reviews was met yesterday and the day before; today two
	// picks are open and one more card is due.
	goals := map[time.Time]leetgrinder.DailyGoal{}
	var attempts []leetgrinder.Attempt
	for d := 1; d <= 2; d++ {
		goals[date.AddDate(0, 0, -d)] = leetgrinder.DailyGoal{New: 1}
		attempts = append(attempts, a(fmt.Sprintf("warmup-%d", d), "solved", at(d)))
	}
	goals[date] = leetgrinder.DailyGoal{New: 1, Review: 2}
	attempts = append(attempts, a("two-sum", "unfinished", at(20)), a("some-problem", "unfinished", at(20)), a("extra-due", "unfinished", at(20)))
	state := leetgrinder.State{Problems: problems, Attempts: attempts, Goals: goals, Plans: map[time.Time][]string{date: {"two-sum", "some-problem"}}}
	today := leetgrinder.NewToday(settings, state, now)
	pref := func(kind string, threshold int) leetgrinder.NotificationPref {
		p := settings.NotificationPref(kind)
		p.Threshold = threshold
		return p
	}
	origin := "https://app.example"

	m, ok := Compose(leetgrinder.NotifyMorningPlan, pref(leetgrinder.NotifyMorningPlan, 0), today, origin)
	if !ok || m.Body != "Goal: 1 new + 2 reviews\nReview: Two Sum\nReview: some-problem\nAlso due: 1\nStreak: 2 days" || m.Click != origin+"/leetgrinder" {
		t.Fatalf("morning: %q", m.Body)
	}
	m, ok = Compose(leetgrinder.NotifyGoalIncomplete, pref(leetgrinder.NotifyGoalIncomplete, 0), today, origin)
	if !ok || m.Body != "Left: 1 new, review: Two Sum, some-problem" || m.Priority != "default" {
		t.Fatalf("goal incomplete: %+v", m)
	}
	if m, ok = Compose(leetgrinder.NotifyStreakAtRisk, pref(leetgrinder.NotifyStreakAtRisk, 2), today, origin); !ok || m.Title != "Leetgrinder: 2-day streak at risk" || m.Priority != "high" {
		t.Fatalf("streak at risk: %+v", m)
	}
	if _, ok = Compose(leetgrinder.NotifyStreakAtRisk, pref(leetgrinder.NotifyStreakAtRisk, 3), today, origin); ok {
		t.Fatal("streak below threshold")
	}
	if m, ok = Compose(leetgrinder.NotifyReviewBacklog, pref(leetgrinder.NotifyReviewBacklog, 1), today, origin); !ok || m.Title != "Leetgrinder: 1 reviews waiting" || m.Click != origin+"/leetgrinder/reviews" {
		t.Fatalf("backlog: %+v", m)
	}
	if _, ok = Compose(leetgrinder.NotifyReviewBacklog, pref(leetgrinder.NotifyReviewBacklog, 2), today, origin); ok {
		t.Fatal("backlog below threshold")
	}
	// More reviews left than open picks are counted.
	state.Goals[date] = leetgrinder.DailyGoal{New: 0, Review: 3}
	if m, _ = Compose(leetgrinder.NotifyGoalIncomplete, pref(leetgrinder.NotifyGoalIncomplete, 0), leetgrinder.NewToday(settings, state, now), origin); m.Body != "Left: review: Two Sum, some-problem, 1 more review" {
		t.Fatalf("extra reviews: %q", m.Body)
	}
	// With a zero threshold, an unmet goal sends even without a streak.
	state.Goals = map[time.Time]leetgrinder.DailyGoal{date: {New: 1, Review: 2}}
	if m, ok = Compose(leetgrinder.NotifyStreakAtRisk, pref(leetgrinder.NotifyStreakAtRisk, 0), leetgrinder.NewToday(settings, state, now), origin); !ok || m.Title != "Leetgrinder: today's goal is still open" {
		t.Fatalf("zero threshold: %+v", m)
	}

	// Met goal: incomplete and streak reminders have nothing to say.
	state.Goals[date] = leetgrinder.DailyGoal{New: 1}
	state.Attempts = append(state.Attempts, a("brand-new", "solved", now.Add(-time.Hour)))
	met := leetgrinder.NewToday(settings, state, now)
	for _, kind := range []string{leetgrinder.NotifyGoalIncomplete, leetgrinder.NotifyStreakAtRisk} {
		if _, ok = Compose(kind, pref(kind, 0), met, origin); ok {
			t.Fatalf("%s sent with the goal met", kind)
		}
	}
	if _, ok = Compose("missing_work", leetgrinder.NotificationPref{}, today, origin); ok {
		t.Fatal("retired kind composed")
	}
}

func TestComposeMorningNewPicks(t *testing.T) {
	loc, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 9, 10, 7, 0, 0, 0, loc)
	date := leetgrinder.Date(now, loc)
	settings := leetgrinder.DefaultSettings()
	settings.NewFromTodos = true
	state := leetgrinder.State{
		Problems: map[string]leetgrinder.Problem{"two-sum": {Slug: "two-sum", Title: "Two Sum"}},
		Goals:    map[time.Time]leetgrinder.DailyGoal{date: {New: 2}},
		NewPlans: map[time.Time][]leetgrinder.NewPick{date: {{Slug: "two-sum", SetTitle: "Blind 75"}, {Slug: "binary-search"}}},
	}
	pref := settings.NotificationPref(leetgrinder.NotifyMorningPlan)
	m, ok := Compose(leetgrinder.NotifyMorningPlan, pref, leetgrinder.NewToday(settings, state, now), "https://app.example")
	if want := "Goal: 2 new + 0 reviews\nNew: Two Sum (Blind 75)\nNew: binary-search\nStreak: 0 days"; !ok || m.Body != want {
		t.Fatalf("morning: %q, want %q", m.Body, want)
	}
	// Turned off, frozen picks are left out.
	settings.NewFromTodos = false
	if m, _ = Compose(leetgrinder.NotifyMorningPlan, pref, leetgrinder.NewToday(settings, state, now), "https://app.example"); strings.Contains(m.Body, "New:") {
		t.Fatalf("off: %q", m.Body)
	}
}
