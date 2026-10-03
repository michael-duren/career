package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/scheduler"
)

const schedulerGoogleBatchSize = 20

// RunScheduler finalizes plans even with no browser open and retries durable exports.
func RunScheduler(ctx context.Context, c config.Config, db *database.Store) {
	s := &Server{db: db, config: c}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	var lastReconcile time.Time
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("scheduler: panic recovered: %v", r)
				}
			}()
			cycle, stop := context.WithTimeout(ctx, 55*time.Second)
			defer stop()
			now := time.Now()
			if err := db.SchedulerTick(cycle, now); err != nil {
				log.Printf("scheduler catch-up: %v", err)
			} else if s.googleConfigured() {
				periodic := now.Sub(lastReconcile) >= 5*time.Minute
				if err := s.syncSchedulerGoogle(cycle, now, periodic); err != nil {
					log.Printf("scheduler Google sync: %v", err)
				} else if periodic {
					lastReconcile = now
				}
			}
		}()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Server) syncSchedulerGoogle(ctx context.Context, now time.Time, periodic bool) error {
	// One worker owns remote writes across service replicas. Local writes stay available.
	conn, err := s.db.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked bool
	if err = conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(724193621)`).Scan(&locked); err != nil || !locked {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = conn.ExecContext(cleanup, `SELECT pg_advisory_unlock(724193621)`)
	}()
	generation, err := s.db.SchedulerGoogleOutbox(ctx)
	if err != nil {
		return err
	}
	if generation == 0 && !periodic {
		return nil
	}
	c, err := s.db.SchedulerGoogle(ctx)
	if err != nil {
		return err
	}
	if len(c.Credentials) == 0 || c.ReconnectRequired {
		return nil
	}
	if c.AccountID == "" || c.CalendarID == "" {
		return errors.New("Google account or destination identity is missing; reconnect")
	}
	doc, err := s.db.SchedulerDocument(ctx)
	if err != nil {
		return err
	}
	var availabilityErr error
	if periodic {
		loc, err := time.LoadLocation(doc.Settings.TimeZone)
		if err != nil {
			return err
		}
		week := scheduler.Monday(now.In(loc).Format("2006-01-02"))
		busy, refreshErr := s.fetchSchedulerAvailability(ctx, week, true, true)
		availabilityErr = refreshErr
		if refreshErr == nil {
			if _, err = s.db.SchedulerWeek(ctx, week, now, busy); err != nil {
				return err
			}
		}
		generation, err = s.db.SchedulerGoogleOutbox(ctx)
		if err != nil {
			return err
		}
		doc, err = s.db.SchedulerDocument(ctx)
		if err != nil {
			return err
		}
		c, err = s.db.SchedulerGoogle(ctx)
		if err != nil {
			return err
		}
	}
	token, err := s.googleAccess(ctx, &c)
	if err != nil {
		return err
	}
	mappings, err := s.db.SchedulerGoogleMappings(ctx, c.AccountID)
	if err != nil {
		return err
	}
	existing := map[string]database.SchedulerGoogleMapping{}
	for _, m := range mappings {
		existing[m.SessionID] = m
	}
	type work struct {
		mapping database.SchedulerGoogleMapping
		event   scheduler.GoogleEvent
		remove  bool
		pending bool
	}
	jobs := map[string]work{}
	active := []string{}
	for id, session := range doc.Sessions {
		if session.Plan == nil || session.State != "accepted" || !session.Plan.Start.After(now) {
			continue
		}
		event := scheduler.GoogleEvent{ID: scheduler.GoogleEventID(c.AccountID, id), Summary: session.Assignment.Title, Description: "Managed by Career Weekly Scheduler. Changes in Google are overwritten. " + s.config.PublicOrigin + "/weekly-scheduler", Start: scheduler.GoogleEventTime{DateTime: session.Plan.Start.Format(time.RFC3339)}, End: scheduler.GoogleEventTime{DateTime: session.Plan.End.Format(time.RFC3339)}}
		if session.Assignment.StepID != "" {
			event.Summary = session.Assignment.GoalTitle + ": " + session.Assignment.Title
		}
		event.ExtendedProperties.Private = map[string]string{"careerSchedulerSession": id}
		previous := existing[id]
		if previous.EventID != "" {
			event.ID = previous.EventID
		}
		if previous.CalendarID != "" && previous.CalendarID != c.CalendarID {
			return fmt.Errorf("mapped Google destination changed; reconnect to the original calendar")
		}
		raw, err := json.Marshal(event)
		if err != nil {
			return err
		}
		m := database.SchedulerGoogleMapping{AccountID: c.AccountID, SessionID: id, CalendarID: c.CalendarID, EventID: event.ID, PlannedStart: session.Plan.Start, DesiredFingerprint: googleHash(string(raw)), DesiredGeneration: generation}
		if err = s.db.PrepareSchedulerGoogleMapping(ctx, m); err != nil {
			return err
		}
		jobs[id] = work{mapping: m, event: event, pending: previous.SyncedFingerprint != m.DesiredFingerprint}
		active = append(active, id)
		delete(existing, id)
	}
	for id, m := range existing {
		session, exists := doc.Sessions[id]
		if !m.PlannedStart.After(now) || (exists && session.Plan != nil && !session.Plan.Start.After(now) && session.State == "accepted") {
			continue
		}
		jobs[id] = work{mapping: m, remove: true, pending: true}
	}
	sort.Strings(active)
	pending := []string{}
	for id, job := range jobs {
		if job.pending {
			pending = append(pending, id)
		}
	}
	sort.Strings(pending)
	cursor := ""
	reconcile := []string{}
	if periodic {
		cursor, err = s.db.SchedulerGoogleReconciliationCursor(ctx, c.AccountID, c.CalendarID)
		if err != nil {
			return err
		}
		for _, id := range active {
			if id > cursor {
				reconcile = append(reconcile, id)
			}
		}
	}
	// Pending work always advances first. Reconciliation fills the remainder and
	// persists a separate cursor even when an ordinary outbox retry has no work.
	selected := append([]string(nil), pending...)
	included := map[string]bool{}
	for _, id := range selected {
		included[id] = true
	}
	for _, id := range reconcile {
		if !included[id] {
			selected = append(selected, id)
		}
	}
	if len(selected) > schedulerGoogleBatchSize {
		selected = selected[:schedulerGoogleBatchSize]
	}
	succeeded := map[string]bool{}
	client := s.googleClient()
	for _, id := range selected {
		job := jobs[id]
		m := job.mapping
		if err = ctx.Err(); err != nil {
			return err
		}
		if job.remove {
			err = client.DeleteEvent(ctx, token, m.CalendarID, m.EventID)
			if err == nil {
				err = s.db.DeleteSchedulerGoogleMapping(ctx, c.AccountID, id)
			}
		} else {
			err = client.UpsertEvent(ctx, token, c.CalendarID, job.event)
			if errors.Is(err, scheduler.ErrGoogleEventDeleted) {
				job.event.ID = scheduler.GoogleEventID(c.AccountID, job.event.ID)
				m.EventID = job.event.ID
				raw, _ := json.Marshal(job.event)
				m.DesiredFingerprint = googleHash(string(raw))
				if err = s.db.PrepareSchedulerGoogleMapping(ctx, m); err != nil {
					return err
				}
				err = client.UpsertEvent(ctx, token, c.CalendarID, job.event)
			}
			if err == nil {
				err = s.db.CompleteSchedulerGoogleMapping(ctx, m)
			}
		}
		if err != nil {
			s.recordGoogleError(ctx, &c, err)
			return err
		}
		succeeded[id] = true
		if periodic {
			// Only advance across a contiguous completed prefix of the reconciliation set.
			for len(reconcile) > 0 && succeeded[reconcile[0]] {
				cursor = reconcile[0]
				reconcile = reconcile[1:]
			}
			if err = s.db.SaveSchedulerGoogleReconciliationCursor(ctx, c.AccountID, c.CalendarID, cursor); err != nil {
				return err
			}
		}
	}
	if periodic && len(reconcile) == 0 {
		if err = s.db.SaveSchedulerGoogleReconciliationCursor(ctx, c.AccountID, c.CalendarID, ""); err != nil {
			return err
		}
	}
	remaining := false
	for _, id := range pending {
		if !succeeded[id] {
			remaining = true
			break
		}
	}
	c.Error = ""
	if availabilityErr != nil {
		c.Error = availabilityErr.Error()
	}
	if err = s.db.UpdateSchedulerGoogleHealth(ctx, &c); err != nil {
		return err
	}
	if !remaining {
		if err = s.db.CompleteSchedulerGoogleOutbox(ctx, generation); err != nil {
			return err
		}
	}
	return availabilityErr
}
