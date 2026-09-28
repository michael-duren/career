package scheduler

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const GoogleScopes = "openid https://www.googleapis.com/auth/calendar.calendarlist.readonly https://www.googleapis.com/auth/calendar.events.freebusy https://www.googleapis.com/auth/calendar.app.created"

var ErrGoogleEventDeleted = errors.New("Google event was deleted and needs a new identity")

var ErrGoogleReconnect = errors.New("Google authorization expired; reconnect your account")

type GoogleToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	Expiry       time.Time `json:"expiry"`
}
type GoogleClient struct {
	HTTP                           *http.Client
	APIBase, TokenURL, UserInfoURL string
}
type GoogleCalendar struct {
	ID       string `json:"id"`
	Summary  string `json:"summary"`
	Primary  bool   `json:"primary,omitempty"`
	Selected bool   `json:"selected"`
}
type GoogleInterval struct {
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	CalendarID string    `json:"calendarId"`
}
type GoogleEventTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone,omitempty"`
}
type GoogleEvent struct {
	ID                 string          `json:"id,omitempty"`
	Summary            string          `json:"summary"`
	Description        string          `json:"description"`
	Start              GoogleEventTime `json:"start"`
	End                GoogleEventTime `json:"end"`
	ExtendedProperties struct {
		Private map[string]string `json:"private"`
	} `json:"extendedProperties"`
}
type googleError struct{ Status int }

func (e googleError) Error() string {
	return fmt.Sprintf("Google Calendar request failed (HTTP %d)", e.Status)
}
func (c GoogleClient) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}
func (c GoogleClient) api() string {
	if c.APIBase != "" {
		return c.APIBase
	}
	return "https://www.googleapis.com/calendar/v3"
}
func (c GoogleClient) request(ctx context.Context, method, target, token string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return errors.New("Google Calendar is unavailable; retry your saved draft")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return ErrGoogleReconnect
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return googleError{resp.StatusCode}
	}
	if out == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(resp.Body, 2<<20))
		return err
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out)
}
func (c GoogleClient) Token(ctx context.Context, values url.Values) (GoogleToken, error) {
	endpoint := c.TokenURL
	if endpoint == "" {
		endpoint = "https://oauth2.googleapis.com/token"
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return GoogleToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.client().Do(req)
	if err != nil {
		return GoogleToken{}, errors.New("Google authorization is unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 || resp.StatusCode == 401 {
		return GoogleToken{}, ErrGoogleReconnect
	}
	if resp.StatusCode != 200 {
		return GoogleToken{}, googleError{resp.StatusCode}
	}
	var token GoogleToken
	if err = json.NewDecoder(io.LimitReader(resp.Body, 65536)).Decode(&token); err != nil {
		return token, err
	}
	if token.AccessToken == "" || token.ExpiresIn <= 0 {
		return token, errors.New("Google returned an invalid authorization")
	}
	token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return token, nil
}
func (c GoogleClient) Account(ctx context.Context, token string) (string, error) {
	endpoint := c.UserInfoURL
	if endpoint == "" {
		endpoint = "https://openidconnect.googleapis.com/v1/userinfo"
	}
	var profile struct {
		Sub string `json:"sub"`
	}
	err := c.request(ctx, "GET", endpoint, token, nil, &profile)
	if err == nil && profile.Sub == "" {
		err = errors.New("Google account identity is missing")
	}
	return profile.Sub, err
}
func (c GoogleClient) Calendars(ctx context.Context, token string) ([]GoogleCalendar, error) {
	result := []GoogleCalendar{}
	page := ""
	for {
		var response struct {
			Items []GoogleCalendar `json:"items"`
			Next  string           `json:"nextPageToken"`
		}
		err := c.request(ctx, "GET", c.api()+"/users/me/calendarList?maxResults=250&pageToken="+url.QueryEscape(page), token, nil, &response)
		if err != nil {
			return nil, err
		}
		result = append(result, response.Items...)
		page = response.Next
		if page == "" {
			return result, nil
		}
		if len(result) > 10000 {
			return nil, errors.New("Google calendar list exceeds supported size")
		}
	}
}
func (c GoogleClient) CreateCalendar(ctx context.Context, token string) (string, error) {
	var response GoogleCalendar
	err := c.request(ctx, "POST", c.api()+"/calendars", token, map[string]string{"summary": "Career Weekly Scheduler", "description": "Managed by Career. Edit plans in the Weekly Scheduler; changes here are overwritten."}, &response)
	if err == nil && response.ID == "" {
		err = errors.New("Google did not return a destination calendar")
	}
	return response.ID, err
}
func (c GoogleClient) FreeBusy(ctx context.Context, token string, inputs []string, output string, from, to time.Time) ([]GoogleInterval, error) {
	items := []map[string]string{}
	seen := map[string]bool{}
	for _, id := range inputs {
		if id != "" && id != output && !seen[id] {
			items = append(items, map[string]string{"id": id})
			seen[id] = true
		}
	}
	if len(items) > 50 {
		return nil, errors.New("select at most 50 busy calendars")
	}
	result := []GoogleInterval{}
	if len(items) == 0 {
		return result, nil
	}
	var response struct {
		Calendars map[string]struct {
			Errors []json.RawMessage `json:"errors"`
			Busy   []struct {
				Start time.Time `json:"start"`
				End   time.Time `json:"end"`
			} `json:"busy"`
		} `json:"calendars"`
	}
	err := c.request(ctx, "POST", c.api()+"/freeBusy", token, map[string]any{"timeMin": from.Format(time.RFC3339), "timeMax": to.Format(time.RFC3339), "items": items}, &response)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		id := item["id"]
		entry, ok := response.Calendars[id]
		if !ok || len(entry.Errors) > 0 {
			return nil, errors.New("Google availability is incomplete; check selected calendars and retry")
		}
		for _, b := range entry.Busy {
			if b.Start.IsZero() || !b.End.After(b.Start) {
				return nil, errors.New("Google returned invalid busy times")
			}
			result = append(result, GoogleInterval{b.Start, b.End, id})
		}
	}
	return result, nil
}
func GoogleEventID(account, session string) string {
	hash := sha256.Sum256([]byte(account + "\x00" + session))
	return "career" + hex.EncodeToString(hash[:])
}
func (c GoogleClient) UpsertEvent(ctx context.Context, token, calendar string, event GoogleEvent) error {
	endpoint := c.api() + "/calendars/" + url.PathEscape(calendar) + "/events"
	err := c.request(ctx, "PUT", endpoint+"/"+url.PathEscape(event.ID), token, event, nil)
	var apierr googleError
	if errors.As(err, &apierr) && (apierr.Status == 404 || apierr.Status == 410) {
		err = c.request(ctx, "POST", endpoint, token, event, nil)
		if errors.As(err, &apierr) && apierr.Status == 409 {
			err = c.request(ctx, "PUT", endpoint+"/"+url.PathEscape(event.ID), token, event, nil)
			if errors.As(err, &apierr) && (apierr.Status == 404 || apierr.Status == 410) {
				return ErrGoogleEventDeleted
			}
		}
	}
	return err
}
func (c GoogleClient) DeleteEvent(ctx context.Context, token, calendar, id string) error {
	err := c.request(ctx, "DELETE", c.api()+"/calendars/"+url.PathEscape(calendar)+"/events/"+url.PathEscape(id), token, nil, nil)
	var apierr googleError
	if errors.As(err, &apierr) && (apierr.Status == 404 || apierr.Status == 410) {
		return nil
	}
	return err
}
func googleCipher(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("SCHEDULER_SECRET_KEY must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func SealGoogleSecret(key, plain []byte) ([]byte, error) {
	a, err := googleCipher(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	return a.Seal(nonce, nonce, plain, []byte("career-scheduler-google-v1")), nil
}
func OpenGoogleSecret(key, sealed []byte) ([]byte, error) {
	a, err := googleCipher(key)
	if err != nil {
		return nil, err
	}
	if len(sealed) < a.NonceSize()+a.Overhead() {
		return nil, errors.New("invalid encrypted Google credentials")
	}
	return a.Open(nil, sealed[:a.NonceSize()], sealed[a.NonceSize():], []byte("career-scheduler-google-v1"))
}
