package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/scheduler"
)

func (s *Server) registerSchedulerGoogle(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(s.private)
		r.Get("/api/scheduler/google/status", s.schedulerGoogleStatus)
		r.Post("/api/scheduler/google/connect", s.schedulerGoogleConnect)
		r.Get("/api/scheduler/google/callback", s.schedulerGoogleCallback)
		r.Get("/api/scheduler/google/calendars", s.schedulerGoogleCalendars)
		r.Post("/api/scheduler/google/calendars", s.schedulerGoogleSelect)
		r.Post("/api/scheduler/google/refresh", s.schedulerGoogleRefresh)
		r.Post("/api/scheduler/google/disconnect", s.schedulerGoogleDisconnect)
	})
}
func (s *Server) googleConfigured() bool {
	return s.config.SchedulerGoogleClientID != "" && s.config.SchedulerGoogleClientSecret.Reveal() != "" && len(s.config.SchedulerSecretKey) == 32
}
func (s *Server) googleClient() scheduler.GoogleClient {
	if s.schedulerGoogleClient != nil {
		return *s.schedulerGoogleClient
	}
	return scheduler.GoogleClient{}
}
func (s *Server) schedulerGoogleStatus(w http.ResponseWriter, r *http.Request) {
	c, err := s.db.SchedulerGoogle(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, map[string]any{"configured": s.googleConfigured(), "connected": len(c.Credentials) > 0, "reconnectRequired": c.ReconnectRequired, "lastRefresh": c.LastRefresh, "error": c.Error, "revision": c.Revision, "calendarId": c.CalendarID})
}
func randomGoogleValue() string {
	var b [32]byte
	rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}
func googleHash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (s *Server) schedulerGoogleConnect(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	if !s.googleConfigured() {
		respond(w, 503, map[string]string{"error": "Google Calendar is not configured on this server"})
		return
	}
	state, verifier := randomGoogleValue(), randomGoogleValue()
	session, _ := r.Cookie("session")
	if err := s.db.StoreSchedulerOAuth(r.Context(), googleHash(state), googleHash(session.Value), verifier); err != nil {
		failure(w, err)
		return
	}
	h := sha256.Sum256([]byte(verifier))
	q := url.Values{"client_id": {s.config.SchedulerGoogleClientID}, "redirect_uri": {s.config.PublicOrigin + "/api/scheduler/google/callback"}, "response_type": {"code"}, "scope": {scheduler.GoogleScopes}, "access_type": {"offline"}, "prompt": {"consent"}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(h[:])}, "code_challenge_method": {"S256"}}
	respond(w, 200, map[string]string{"url": "https://accounts.google.com/o/oauth2/v2/auth?" + q.Encode()})
}
func (s *Server) schedulerGoogleCallback(w http.ResponseWriter, r *http.Request) {
	unlock, lockErr := s.lockGoogleConnection(r.Context())
	if lockErr != nil {
		failure(w, lockErr)
		return
	}
	defer unlock()
	if !s.googleConfigured() {
		respond(w, 503, map[string]string{"error": "Google Calendar is not configured"})
		return
	}
	session, _ := r.Cookie("session")
	verifier, err := s.db.ConsumeSchedulerOAuth(r.Context(), googleHash(r.URL.Query().Get("state")), googleHash(session.Value))
	if err != nil {
		respond(w, 400, map[string]string{"error": "Google connection expired or belongs to another session. Try connecting again."})
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		http.Redirect(w, r, "/weekly-scheduler?google=declined", 303)
		return
	}
	c, err := s.db.SchedulerGoogle(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	client := s.googleClient()
	token, err := client.Token(r.Context(), url.Values{"client_id": {s.config.SchedulerGoogleClientID}, "client_secret": {s.config.SchedulerGoogleClientSecret.Reveal()}, "redirect_uri": {s.config.PublicOrigin + "/api/scheduler/google/callback"}, "code": {r.URL.Query().Get("code")}, "code_verifier": {verifier}, "grant_type": {"authorization_code"}})
	if err != nil {
		respond(w, 502, map[string]string{"error": err.Error()})
		return
	}
	account, err := client.Account(r.Context(), token.AccessToken)
	if err != nil {
		respond(w, 502, map[string]string{"error": err.Error()})
		return
	}
	if token.RefreshToken == "" {
		respond(w, 400, map[string]string{"error": "Google did not grant offline access. Reconnect and allow the requested access."})
		return
	}
	if c.AccountID != account || c.CalendarID == "" {
		calendar, err := s.db.SchedulerGoogleDestination(r.Context(), account)
		if err != nil {
			failure(w, err)
			return
		}
		if calendar == "" {
			calendar, err = client.CreateCalendar(r.Context(), token.AccessToken)
		}
		if err != nil {
			respond(w, 502, map[string]string{"error": err.Error()})
			return
		}
		c.CalendarID = calendar
		c.SelectedCalendars = []string{"primary"}
	}
	c.AccountID = account
	raw, err := json.Marshal(token)
	if err != nil {
		failure(w, err)
		return
	}
	c.Credentials, err = scheduler.SealGoogleSecret(s.config.SchedulerSecretKey, raw)
	if err != nil {
		failure(w, err)
		return
	}
	c.ReconnectRequired = false
	c.Error = ""
	c.Busy = nil
	c.LastRefresh = nil
	c.BusyFrom = nil
	c.BusyTo = nil
	if _, err = s.db.SaveSchedulerGoogle(r.Context(), c, c.Revision); err != nil {
		failure(w, err)
		return
	}
	http.Redirect(w, r, "/weekly-scheduler?google=connected", 303)
}
func (s *Server) googleAccess(ctx context.Context, c *database.SchedulerGoogleConnection) (string, error) {
	if len(c.Credentials) == 0 {
		return "", errors.New("Google Calendar is disconnected")
	}
	if !s.googleConfigured() {
		return "", errors.New("Google Calendar configuration is missing")
	}
	if c.ReconnectRequired {
		return "", scheduler.ErrGoogleReconnect
	}
	raw, err := scheduler.OpenGoogleSecret(s.config.SchedulerSecretKey, c.Credentials)
	if err != nil {
		wrapped := fmt.Errorf("%w: Google credentials cannot be decrypted; restore the scheduler key or reconnect", scheduler.ErrGoogleReconnect)
		s.recordGoogleError(ctx, c, wrapped)
		return "", wrapped
	}
	var token scheduler.GoogleToken
	if err = json.Unmarshal(raw, &token); err != nil {
		s.recordGoogleError(ctx, c, err)
		return "", err
	}
	if token.Expiry.After(s.schedulerNow().Add(time.Minute)) {
		return token.AccessToken, nil
	}
	fresh, err := s.googleClient().Token(ctx, url.Values{"client_id": {s.config.SchedulerGoogleClientID}, "client_secret": {s.config.SchedulerGoogleClientSecret.Reveal()}, "refresh_token": {token.RefreshToken}, "grant_type": {"refresh_token"}})
	if err != nil {
		s.recordGoogleError(ctx, c, err)
		return "", err
	}
	if fresh.RefreshToken == "" {
		fresh.RefreshToken = token.RefreshToken
	}
	raw, err = json.Marshal(fresh)
	if err != nil {
		return "", err
	}
	c.Credentials, err = scheduler.SealGoogleSecret(s.config.SchedulerSecretKey, raw)
	if err != nil {
		return "", err
	}
	if err = s.db.UpdateSchedulerGoogleHealth(ctx, c); err != nil {
		return "", err
	}
	return fresh.AccessToken, nil
}
func (s *Server) recordGoogleError(ctx context.Context, c *database.SchedulerGoogleConnection, err error) {
	c.Error = err.Error()
	if errors.Is(err, scheduler.ErrGoogleReconnect) {
		c.ReconnectRequired = true
	}
	if e := s.db.RecordSchedulerGoogleError(ctx, c); e != nil && !errors.Is(e, database.ErrConflict) && ctx.Err() == nil {
		log.Printf("scheduler Google: failed to persist connection error state: %v", e)
	}
}
func (s *Server) schedulerGoogleCalendars(w http.ResponseWriter, r *http.Request) {
	c, err := s.db.SchedulerGoogle(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	token, err := s.googleAccess(r.Context(), &c)
	if err != nil {
		respond(w, 503, map[string]string{"error": err.Error()})
		return
	}
	calendars, err := s.googleClient().Calendars(r.Context(), token)
	if err != nil {
		s.recordGoogleError(r.Context(), &c, err)
		respond(w, 502, map[string]string{"error": err.Error()})
		return
	}
	selected := map[string]bool{}
	for _, id := range c.SelectedCalendars {
		selected[id] = true
	}
	result := []scheduler.GoogleCalendar{}
	for _, calendar := range calendars {
		if calendar.ID == c.CalendarID {
			continue
		}
		calendar.Selected = selected[calendar.ID] || (calendar.Primary && selected["primary"])
		result = append(result, calendar)
	}
	respond(w, 200, map[string]any{"calendars": result, "revision": c.Revision})
}
func (s *Server) schedulerGoogleSelect(w http.ResponseWriter, r *http.Request) {
	unlock, lockErr := s.lockGoogleConnection(r.Context())
	if lockErr != nil {
		failure(w, lockErr)
		return
	}
	defer unlock()
	if !s.mutation(w, r) {
		return
	}
	var input struct {
		CalendarIDs []string `json:"calendarIds"`
		Revision    string   `json:"revision"`
	}
	if !decode(w, r, 16384, &input) {
		return
	}
	c, err := s.db.SchedulerGoogle(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	if len(input.CalendarIDs) > 50 {
		respond(w, 400, map[string]string{"error": "Select at most 50 busy calendars"})
		return
	}
	token, err := s.googleAccess(r.Context(), &c)
	if err != nil {
		respond(w, 503, map[string]string{"error": err.Error()})
		return
	}
	calendars, err := s.googleClient().Calendars(r.Context(), token)
	if err != nil {
		respond(w, 502, map[string]string{"error": err.Error()})
		return
	}
	valid := map[string]bool{}
	for _, v := range calendars {
		if v.ID != c.CalendarID {
			valid[v.ID] = true
		}
	}
	seen := map[string]bool{}
	for _, id := range input.CalendarIDs {
		if !valid[id] || seen[id] {
			respond(w, 400, map[string]string{"error": "Select unique calendars from the available list"})
			return
		}
		seen[id] = true
	}
	c.SelectedCalendars = input.CalendarIDs
	c.LastRefresh = nil
	c.BusyFrom = nil
	c.BusyTo = nil
	c.Busy = nil
	if _, err = s.db.SaveSchedulerGoogle(r.Context(), c, input.Revision); err != nil {
		failure(w, err)
		return
	}
	s.schedulerGoogleStatus(w, r)
}
func (s *Server) schedulerGoogleDisconnect(w http.ResponseWriter, r *http.Request) {
	unlock, lockErr := s.lockGoogleConnection(r.Context())
	if lockErr != nil {
		failure(w, lockErr)
		return
	}
	defer unlock()
	if !s.mutation(w, r) {
		return
	}
	var input struct {
		Revision string `json:"revision"`
	}
	if !decode(w, r, 2048, &input) {
		return
	}
	c, err := s.db.SchedulerGoogle(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	c.Credentials = nil
	c.Busy = nil
	c.LastRefresh = nil
	c.BusyFrom = nil
	c.BusyTo = nil
	c.ReconnectRequired = false
	c.Error = ""
	if _, err = s.db.SaveSchedulerGoogle(r.Context(), c, input.Revision); err != nil {
		failure(w, err)
		return
	}
	s.schedulerGoogleStatus(w, r)
}
func googleWeekRange(week string, now time.Time) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", week)
	from, to := start.Add(-24*time.Hour), start.Add(65*24*time.Hour)
	current, _ := time.Parse("2006-01-02", scheduler.Monday(now.UTC().Format("2006-01-02")))
	if current.Add(-24 * time.Hour).Before(from) {
		from = current.Add(-24 * time.Hour)
	}
	if current.Add(65 * 24 * time.Hour).After(to) {
		to = current.Add(65 * 24 * time.Hour)
	}
	return from, to, err
}
func googleBusy(c database.SchedulerGoogleConnection) []scheduler.Busy {
	result := []scheduler.Busy{}
	for i, b := range c.Busy {
		result = append(result, scheduler.Busy{ID: fmt.Sprintf("google:%s:%d", b.CalendarID, i), Title: "Google busy", Start: b.Start, End: b.End})
	}
	return result
}

const schedulerGoogleCacheTTL = 5 * time.Minute

func googleCacheCovers(c database.SchedulerGoogleConnection, from, to, now time.Time) bool {
	return c.Error == "" && !c.ReconnectRequired && c.LastRefresh != nil && !c.LastRefresh.After(now) && now.Sub(*c.LastRefresh) < schedulerGoogleCacheTTL && c.BusyFrom != nil && c.BusyTo != nil && !c.BusyFrom.After(from) && !c.BusyTo.Before(to)
}
func (s *Server) schedulerPlanningAvailability(ctx context.Context, week string) ([]scheduler.Busy, error) {
	return s.fetchSchedulerAvailability(ctx, week, true, false)
}
func (s *Server) schedulerAvailability(ctx context.Context, week string) ([]scheduler.Busy, error) {
	return s.fetchSchedulerAvailability(ctx, week, false, true)
}
func (s *Server) fetchSchedulerAvailability(ctx context.Context, week string, force, coalesce bool) ([]scheduler.Busy, error) {
	observed, err := s.db.SchedulerGoogle(ctx)
	if err != nil {
		return nil, err
	}
	if len(observed.Credentials) == 0 {
		return []scheduler.Busy{}, nil
	}
	from, to, err := googleWeekRange(week, s.schedulerNow())
	if err != nil {
		return nil, err
	}
	doc, err := s.db.SchedulerDocument(ctx)
	if err != nil {
		return nil, err
	}
	for _, session := range doc.Sessions {
		if session.Plan != nil && session.Plan.End.After(s.schedulerNow()) {
			if session.Plan.Start.Before(from) {
				from = session.Plan.Start.Add(-24 * time.Hour)
			}
			if session.Plan.End.After(to) {
				to = session.Plan.End.Add(24 * time.Hour)
			}
		}
	}
	if !force && googleCacheCovers(observed, from, to, s.schedulerNow()) {
		return googleBusy(observed), nil
	}
	// Serialize fetches across replicas without holding a reservation transaction.
	unlock, err := s.lockGoogleAvailability(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	c, err := s.db.SchedulerGoogle(ctx)
	if err != nil {
		return nil, err
	}
	if c.Revision != observed.Revision || len(c.Credentials) == 0 {
		return nil, errors.New("Google connection changed during availability refresh; retry")
	}
	advanced := c.AvailabilityRevision > observed.AvailabilityRevision
	if googleCacheCovers(c, from, to, s.schedulerNow()) && (!force || (coalesce && advanced)) {
		return googleBusy(c), nil
	}
	token, err := s.googleAccess(ctx, &c)
	if err != nil {
		return googleBusy(c), err
	}
	busy, err := s.googleClient().FreeBusy(ctx, token, c.SelectedCalendars, c.CalendarID, from, to)
	if err != nil {
		s.recordGoogleError(ctx, &c, err)
		return googleBusy(c), err
	}
	// A disconnect/selection change must invalidate an in-flight response.
	current, err := s.db.SchedulerGoogle(ctx)
	if err != nil {
		return nil, err
	}
	if current.Revision != c.Revision || len(current.Credentials) == 0 {
		return nil, errors.New("Google connection changed during availability refresh; retry")
	}
	now := s.schedulerNow().UTC()
	current.Busy = busy
	current.BusyFrom = &from
	current.BusyTo = &to
	current.LastRefresh = &now
	current.Error = ""
	if err = s.db.UpdateSchedulerGoogleAvailability(ctx, &current); err != nil {
		return googleBusy(current), err
	}
	return googleBusy(current), nil
}
func (s *Server) schedulerGoogleRefresh(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	var input struct {
		Week string `json:"week"`
	}
	if !decode(w, r, 2048, &input) {
		return
	}
	busy, err := s.fetchSchedulerAvailability(r.Context(), input.Week, true, true)
	if err != nil {
		respond(w, 503, map[string]string{"error": err.Error()})
		return
	}
	if _, err = s.db.SchedulerWeek(r.Context(), input.Week, s.schedulerNow(), busy); err != nil {
		failure(w, err)
		return
	}
	s.schedulerGoogleStatus(w, r)
}

// Connection changes wait for a running export batch before changing its authority.
func (s *Server) lockGoogleConnection(ctx context.Context) (func(), error) {
	conn, err := s.db.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock(724193621)`); err != nil {
		conn.Close()
		return nil, err
	}
	return func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = conn.ExecContext(cleanup, `SELECT pg_advisory_unlock(724193621)`)
		conn.Close()
	}, nil
}

func (s *Server) lockGoogleAvailability(ctx context.Context) (func(), error) {
	conn, err := s.db.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock(724193622)`); err != nil {
		conn.Close()
		return nil, err
	}
	return func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = conn.ExecContext(cleanup, `SELECT pg_advisory_unlock(724193622)`)
		conn.Close()
	}, nil
}
