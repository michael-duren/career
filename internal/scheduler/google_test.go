package scheduler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGoogleBusyExcludesOutputAndRejectsPartialErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Items) != 1 || body.Items[0].ID != "input" {
			t.Errorf("items = %+v", body.Items)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"calendars":{"input":{"errors":[{"reason":"notFound"}]}}}`))
	}))
	defer server.Close()
	c := GoogleClient{HTTP: server.Client(), APIBase: server.URL}
	_, err := c.FreeBusy(context.Background(), "token", []string{"input", "output"}, "output", time.Now(), time.Now().Add(time.Hour))
	if err == nil {
		t.Fatal("partial Google availability must fail closed")
	}
}
func TestGoogleEventRetryUsesStableIdentity(t *testing.T) {
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method)
		if !strings.HasSuffix(r.URL.Path, GoogleEventID("account", "session")) && r.Method != "POST" {
			t.Errorf("unstable path: %s", r.URL.Path)
		}
		if r.Method == "PUT" {
			w.WriteHeader(404)
			w.Write([]byte(`{}`))
			return
		}
		var event GoogleEvent
		json.NewDecoder(r.Body).Decode(&event)
		if event.ID != GoogleEventID("account", "session") {
			t.Errorf("event ID = %s", event.ID)
		}
		w.WriteHeader(409)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	c := GoogleClient{HTTP: server.Client(), APIBase: server.URL}
	err := c.UpsertEvent(context.Background(), "token", "calendar", GoogleEvent{ID: GoogleEventID("account", "session")})
	if err == nil {
		t.Fatal("unresolved conflict should remain retryable")
	}
	if len(calls) != 3 {
		t.Fatalf("want PUT POST PUT, got %v", calls)
	}
}
func TestGoogleSecretEncryption(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	raw := []byte(`{"refresh_token":"private"}`)
	sealed, err := SealGoogleSecret(key, raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sealed), "private") {
		t.Fatal("plaintext credential")
	}
	opened, err := OpenGoogleSecret(key, sealed)
	if err != nil || string(opened) != string(raw) {
		t.Fatalf("round trip: %s %v", opened, err)
	}
	sealed[len(sealed)-1] ^= 1
	if _, err = OpenGoogleSecret(key, sealed); err == nil {
		t.Fatal("tampered credential accepted")
	}
}

func TestGoogleBusyReturnsIntervalsAndSkipsEmptySelection(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"calendars":{"input":{"busy":[{"start":"2026-09-28T10:00:00Z","end":"2026-09-28T11:00:00Z"}]}}}`))
	}))
	defer server.Close()
	c := GoogleClient{HTTP: server.Client(), APIBase: server.URL}
	from := instant("2026-09-28T00:00:00Z")
	busy, err := c.FreeBusy(context.Background(), "token", []string{"input", "input", "output"}, "output", from, from.Add(24*time.Hour))
	if err != nil || len(busy) != 1 || busy[0].CalendarID != "input" || busy[0].End.Sub(busy[0].Start) != time.Hour {
		t.Fatalf("busy=%+v err=%v", busy, err)
	}
	busy, err = c.FreeBusy(context.Background(), "token", []string{"output"}, "output", from, from.Add(time.Hour))
	if err != nil || len(busy) != 0 || calls != 1 {
		t.Fatalf("output exclusion: %+v %v calls=%d", busy, err, calls)
	}
}

func TestGoogleAuthorizationRevokedRequiresReconnect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer server.Close()
	c := GoogleClient{HTTP: server.Client(), TokenURL: server.URL, APIBase: server.URL}
	if _, err := c.Token(context.Background(), nil); err != ErrGoogleReconnect {
		t.Fatalf("refresh error=%v", err)
	}
	if _, err := c.Calendars(context.Background(), "expired"); err != ErrGoogleReconnect {
		t.Fatalf("calendar error=%v", err)
	}
}

func TestGoogleDeletedEventSignalsNewIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(409)
		} else {
			w.WriteHeader(410)
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	c := GoogleClient{HTTP: server.Client(), APIBase: server.URL}
	if err := c.UpsertEvent(context.Background(), "token", "calendar", GoogleEvent{ID: "deleted"}); err != ErrGoogleEventDeleted {
		t.Fatalf("deleted event=%v", err)
	}
}
