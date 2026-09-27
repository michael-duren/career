package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
	"github.com/michael-duren/career-strategy/internal/leetgrinder/notify"
)

// notificationLogRows is how many log rows the settings page shows.
const notificationLogRows = 14

func (s *Server) registerLeetgrinderNotify(r chi.Router) {
	r.Post("/leetgrinder/settings/ntfy", s.leetgrinderSaveNtfy)
	r.Post("/leetgrinder/settings/ntfy/test", s.leetgrinderTestNtfy)
	r.Post("/leetgrinder/settings/notifications", s.leetgrinderSaveNotifications)
}

// withNotify fills the notification sections from saved settings, keeping
// any draft forms page already carries.
func (s *Server) withNotify(ctx context.Context, page leetgrinder.SettingsPage) leetgrinder.SettingsPage {
	n := &page.Notify
	if n.Ntfy.Revision == "" {
		n.Ntfy = leetgrinder.NewNtfyForm(page.Settings)
	}
	if n.Notifications.Revision == "" {
		n.Notifications = leetgrinder.NewNotificationsForm(page.Settings)
	}
	n.Ntfy.TokenSet = page.Settings.TokenSet()
	n.Location = page.Settings.Location()
	n.SecretMissing = len(s.config.LeetgrinderSecretKey) == 0
	n.TopicMissing = strings.TrimSpace(page.Settings.NtfyTopic) == ""
	var err error
	n.Log, err = s.db.RecentLeetgrinderNotifications(ctx, notificationLogRows)
	n.LogError = err != nil
	return page
}

// rejectNotify re-renders settings with the submitted drafts and a message.
func (s *Server) rejectNotify(w http.ResponseWriter, r *http.Request, status int, message string, draft leetgrinder.NotifyPanel) {
	settings, err := s.db.LeetgrinderSettings(r.Context())
	if err != nil {
		message += " Your settings could not be reloaded; your draft is retained below."
	}
	page := leetgrinder.SettingsPage{Settings: settings, Now: s.clock(), Schedule: leetgrinder.NewScheduleForm(settings), Error: message, Notify: draft}
	renderLeetgrinder(w, r, status, leetgrinder.SettingsView(s.withAPITokens(r, s.withNotify(r.Context(), page))))
}

// saveNotifySettings applies change and maps the outcome onto a response.
func (s *Server) saveNotifySettings(w http.ResponseWriter, r *http.Request, revision, anchor string, draft leetgrinder.NotifyPanel, change func(*leetgrinder.Settings) error) {
	_, err := s.db.UpdateLeetgrinderSettings(r.Context(), revision, change)
	var invalid settingsError
	switch {
	case err == nil:
		http.Redirect(w, r, "/leetgrinder/settings?saved="+anchor+"#"+anchor, http.StatusSeeOther)
	case errors.As(err, &invalid):
		s.rejectNotify(w, r, 400, invalid.Error(), draft)
	case errors.Is(err, database.ErrConflict):
		s.rejectNotify(w, r, 409, "Settings changed since you opened this page. Reload settings to see the current values, then reapply your draft shown below.", draft)
	case errors.Is(err, database.ErrInvalid):
		s.rejectNotify(w, r, 400, "Reload the settings page and try again.", draft)
	default:
		s.rejectNotify(w, r, 503, "Your settings could not be saved. Your draft is retained; please retry.", draft)
	}
}

func (s *Server) leetgrinderSaveNtfy(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	form := leetgrinder.NtfyForm{URL: strings.TrimSpace(r.PostForm.Get("url")), Topic: strings.TrimSpace(r.PostForm.Get("topic")), Revision: r.PostForm.Get("revision"), ClearToken: r.PostForm.Get("clear_token") == "true"}
	token := strings.TrimSpace(r.PostForm.Get("token"))
	draft := leetgrinder.NotifyPanel{Ntfy: form}
	reject := func(status int, message string) {
		if token != "" {
			message += " Re-enter the token; it is never shown again."
		}
		s.rejectNotify(w, r, status, message, draft)
	}
	if token != "" && form.ClearToken {
		reject(400, "Enter a new token or clear the saved one, not both.")
		return
	}
	var sealed []byte
	if token != "" {
		if len(token) > 256 || strings.IndexFunc(token, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
			reject(400, "The token must be at most 256 characters without spaces.")
			return
		}
		box, err := leetgrinder.NewSecretBox(s.config.LeetgrinderSecretKey)
		if err != nil {
			reject(400, "The token cannot be saved because LEETGRINDER_SECRET_KEY is not configured on the server.")
			return
		}
		if sealed, err = box.Seal([]byte(token)); err != nil {
			reject(503, "The token could not be encrypted. Please retry.")
			return
		}
	}
	s.saveNotifySettings(w, r, form.Revision, "ntfy", draft, func(settings *leetgrinder.Settings) error {
		next := strings.TrimRight(form.URL, "/")
		// A saved token only ever goes to the server it was entered for.
		if settings.TokenSet() && sealed == nil && !form.ClearToken && ntfyOrigin(next) != ntfyOrigin(settings.NtfyURL) {
			return settingsError("The server changed. Re-enter the access token for the new server, or clear the saved token.")
		}
		settings.NtfyURL, settings.NtfyTopic = next, form.Topic
		switch {
		case sealed != nil:
			settings.NtfyTokenCiphertext = sealed
		case form.ClearToken:
			settings.NtfyTokenCiphertext = nil
		}
		if err := settings.Validate(); err != nil {
			return settingsError("Check the ntfy settings: " + err.Error() + ".")
		}
		return nil
	})
}

// ntfyOrigin is the scheme and host of an ntfy server URL, or "" if it is invalid.
func ntfyOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Scheme + "://" + u.Host)
}

func (s *Server) leetgrinderSaveNotifications(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	form := leetgrinder.NotificationsForm{Revision: r.PostForm.Get("revision")}
	prefs := map[string]leetgrinder.NotificationPref{}
	var problem string
	for _, kind := range leetgrinder.NotificationKinds {
		d := leetgrinder.NotificationDraft{Kind: kind, Enabled: r.PostForm.Get(kind.Key+"_enabled") == "true", Time: strings.TrimSpace(r.PostForm.Get(kind.Key + "_time")), Priority: r.PostForm.Get(kind.Key + "_priority")}
		pref := leetgrinder.NotificationPref{Enabled: d.Enabled, Time: d.Time, Priority: d.Priority}
		if kind.MaxThreshold > 0 {
			d.Threshold = strings.TrimSpace(r.PostForm.Get(kind.Key + "_threshold"))
			n, err := strconv.Atoi(d.Threshold)
			if (err != nil || n < 1 || n > kind.MaxThreshold) && problem == "" {
				problem = kind.Label + ": enter a threshold from 1 to " + strconv.Itoa(kind.MaxThreshold) + "."
			}
			pref.Threshold = n
		}
		form.Items = append(form.Items, d)
		prefs[kind.Key] = pref
	}
	draft := leetgrinder.NotifyPanel{Notifications: form}
	if problem != "" {
		s.rejectNotify(w, r, 400, problem, draft)
		return
	}
	s.saveNotifySettings(w, r, form.Revision, "notifications", draft, func(settings *leetgrinder.Settings) error {
		settings.Notifications = prefs
		if err := settings.Validate(); err != nil {
			return settingsError("Check the notifications: " + err.Error() + ".")
		}
		return nil
	})
}

func (s *Server) leetgrinderTestNtfy(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	settings, err := s.db.LeetgrinderSettings(r.Context())
	if err != nil {
		s.rejectNotify(w, r, 503, "Your settings are unavailable. Please retry.", leetgrinder.NotifyPanel{})
		return
	}
	now := s.clock()
	sendErr := notify.Send(r.Context(), s.ntfyClient, settings, s.config.LeetgrinderSecretKey, notify.TestMessage(s.config.PublicOrigin))
	status, detail := leetgrinder.NotifySent, "Test notification"
	if sendErr != nil {
		status, detail = leetgrinder.NotifyFailed, sendErr.Error()
	}
	// The log is informational; a failed write must not hide the send result.
	_ = s.db.RecordLeetgrinderTestNotification(r.Context(), leetgrinder.Date(now, settings.Location()), status, detail, now)
	if sendErr != nil {
		s.rejectNotify(w, r, 502, "The test notification failed: "+sendErr.Error()+".", leetgrinder.NotifyPanel{})
		return
	}
	http.Redirect(w, r, "/leetgrinder/settings?tested=1#ntfy", http.StatusSeeOther)
}
