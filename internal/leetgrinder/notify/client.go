// Package notify sends Leetgrinder reminders through ntfy.
package notify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// Timeout bounds each ntfy request.
const Timeout = 10 * time.Second

// Message is one ntfy notification. Click is an absolute URL or empty.
type Message struct {
	Title    string
	Body     string
	Priority string
	Tags     []string
	Click    string
}

// NewHTTPClient returns a client that never follows redirects, so the bearer
// token only ever reaches the configured server.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// Token decrypts the stored ntfy token, or returns "" when none is set. Its
// errors never contain the token.
func Token(settings leetgrinder.Settings, key []byte) (string, error) {
	if !settings.TokenSet() {
		return "", nil
	}
	box, err := leetgrinder.NewSecretBox(key)
	if err != nil {
		if errors.Is(err, leetgrinder.ErrNoSecretKey) {
			return "", errors.New("the ntfy token cannot be read because LEETGRINDER_SECRET_KEY is not configured")
		}
		return "", errors.New("the ntfy token cannot be read: invalid LEETGRINDER_SECRET_KEY")
	}
	plain, err := box.Open(settings.NtfyTokenCiphertext)
	if err != nil {
		return "", errors.New("the stored ntfy token cannot be decrypted; re-enter it in settings")
	}
	return string(plain), nil
}

// Send posts m to the configured server and topic. Errors are safe to store
// and show: they never contain the token.
func Send(ctx context.Context, client *http.Client, settings leetgrinder.Settings, key []byte, m Message) error {
	if settings.NtfyTopic == "" {
		return errors.New("set an ntfy topic first")
	}
	token, err := Token(settings, key)
	if err != nil {
		return err
	}
	if client == nil {
		client = NewHTTPClient()
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	target := strings.TrimRight(settings.NtfyURL, "/") + "/" + url.PathEscape(settings.NtfyTopic)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(m.Body))
	if err != nil {
		return errors.New("the ntfy server URL is invalid")
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Title", mime.BEncoding.Encode("utf-8", headerSafe(m.Title)))
	if m.Priority != "" {
		req.Header.Set("Priority", m.Priority)
	}
	if len(m.Tags) > 0 {
		req.Header.Set("Tags", strings.Join(m.Tags, ","))
	}
	if m.Click != "" {
		req.Header.Set("Click", m.Click)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		if errors.Is(err, context.DeadlineExceeded) || (ue != nil && ue.Timeout()) {
			return errors.New("ntfy request timed out")
		}
		return errors.New(redact("ntfy request failed: "+err.Error(), token))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		detail := fmt.Sprintf("ntfy returned %d", resp.StatusCode)
		if text := strings.TrimSpace(headerSafe(string(body))); text != "" {
			detail += ": " + text
		}
		return errors.New(redact(Truncate(detail, 240), token))
	}
	return nil
}

// headerSafe flattens control characters, including newlines, to spaces.
func headerSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}

func redact(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "[redacted]")
}

// Truncate shortens s to at most n runes.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
