package server

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/scheduler"
)

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
		busy, refreshErr := s.schedulerPlanningAvailability(ctx, week)
		availabilityErr = refreshErr
		if refreshErr == nil {
			if _, err = s.db.SchedulerWeek(ctx, week, now, busy); err != nil {
				return err
			}
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
	client := s.googleClient()
	for id, session := range doc.Sessions {
		if session.Plan == nil || session.State != "accepted" || !session.Plan.Start.After(now) {
			continue
		}
		event := scheduler.GoogleEvent{ID: scheduler.GoogleEventID(c.AccountID, id), Summary: session.Assignment.Title, Description: "Managed by Career Weekly Scheduler. Changes in Google are overwritten. " + s.config.PublicOrigin + "/weekly-scheduler", Start: scheduler.GoogleEventTime{DateTime: session.Plan.Start.Format(time.RFC3339)}, End: scheduler.GoogleEventTime{DateTime: session.Plan.End.Format(time.RFC3339)}}
		if session.Assignment.StepID != "" {
			event.Summary = session.Assignment.GoalTitle + ": " + session.Assignment.Title
		}
		event.ExtendedProperties.Private = map[string]string{"careerSchedulerSession": id}
		if mapping, ok := existing[id]; ok {
			event.ID = mapping.EventID
		}
		mapping := database.SchedulerGoogleMapping{AccountID: c.AccountID, SessionID: id, CalendarID: c.CalendarID, EventID: event.ID, PlannedStart: session.Plan.Start}
		// Persist identity before sending, so a crash after Google's response retries it.
		if err = s.db.SaveSchedulerGoogleMapping(ctx, mapping); err != nil {
			return err
		}
		err = client.UpsertEvent(ctx, token, c.CalendarID, event)
		if errors.Is(err, scheduler.ErrGoogleEventDeleted) {
			event.ID = scheduler.GoogleEventID(c.AccountID, event.ID)
			mapping.EventID = event.ID
			if err = s.db.SaveSchedulerGoogleMapping(ctx, mapping); err != nil {
				return err
			}
			err = client.UpsertEvent(ctx, token, c.CalendarID, event)
		}
		if err != nil {
			s.recordGoogleError(ctx, &c, err)
			return err
		}
		delete(existing, id)
	}
	for id, m := range existing {
		session, exists := doc.Sessions[id]
		if !m.PlannedStart.After(now) || (exists && session.Plan != nil && !session.Plan.Start.After(now) && session.State == "accepted") {
			continue
		}
		if err = client.DeleteEvent(ctx, token, m.CalendarID, m.EventID); err != nil {
			s.recordGoogleError(ctx, &c, err)
			return err
		}
		if err = s.db.DeleteSchedulerGoogleMapping(ctx, c.AccountID, id); err != nil {
			return err
		}
	}
	c.Error = ""
	if availabilityErr != nil {
		c.Error = availabilityErr.Error()
	}
	if err = s.db.UpdateSchedulerGoogleHealth(ctx, &c); err != nil {
		return err
	}
	if err = s.db.CompleteSchedulerGoogleOutbox(ctx, generation); err != nil {
		return err
	}
	return availabilityErr
}
