package database

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/linkedin"
)

type ConnectionImport struct {
	Parsed          int `json:"parsed"`
	Created         int `json:"created"`
	Updated         int `json:"updated"`
	Unchanged       int `json:"unchanged"`
	Skipped         int `json:"skipped"`
	Linked          int `json:"linked"`
	MessagesMatched int `json:"messagesMatched"`
}

// clip truncates s to max UTF-16 code units, matching Validate's length rules.
func clip(s string, max int) string {
	s = strings.TrimSpace(s)
	for size(s) > max {
		r := []rune(s)
		s = string(r[:len(r)-1])
	}
	return s
}

type existingConnection struct {
	id, name, role, companyName, email      string
	companySlug, connectedOn, lastContacted sql.NullString
}

// ImportConnections upserts LinkedIn connections by profile URL. LinkedIn owns
// name, role, company, email and connection date; notes, tags, cadence and the
// queue flag stay as the user left them. A company link is kept while the
// LinkedIn company name is unchanged, so manual links survive re-imports.
func (s *Store) ImportConnections(ctx context.Context, rows []linkedin.Connection, lastMessages map[string]string) (ConnectionImport, error) {
	summary := ConnectionImport{Parsed: len(rows)}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return summary, err
	}
	defer tx.Rollback()
	// Serialize imports so concurrent uploads cannot race on the URL index.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(724193603)"); err != nil {
		return summary, err
	}
	companies := []linkedin.Company{}
	companyRows, err := tx.QueryContext(ctx, "SELECT slug,title FROM companies")
	if err != nil {
		return summary, err
	}
	for companyRows.Next() {
		var c linkedin.Company
		if err = companyRows.Scan(&c.Slug, &c.Title); err != nil {
			companyRows.Close()
			return summary, err
		}
		companies = append(companies, c)
	}
	companyRows.Close()
	if err = companyRows.Err(); err != nil {
		return summary, err
	}
	matcher := linkedin.NewMatcher(companies)

	existing := map[string]*existingConnection{}
	existingRows, err := tx.QueryContext(ctx, "SELECT id::text,url,name,role,company_name,email,company_slug,to_char(connected_on,'YYYY-MM-DD'),to_char(last_contacted_on,'YYYY-MM-DD') FROM connections WHERE url IS NOT NULL")
	if err != nil {
		return summary, err
	}
	for existingRows.Next() {
		var url string
		c := &existingConnection{}
		if err = existingRows.Scan(&c.id, &url, &c.name, &c.role, &c.companyName, &c.email, &c.companySlug, &c.connectedOn, &c.lastContacted); err != nil {
			existingRows.Close()
			return summary, err
		}
		// Manually entered URLs may use another LinkedIn spelling.
		existing[strings.ToLower(linkedin.CanonicalURL(url))] = c
	}
	existingRows.Close()
	if err = existingRows.Err(); err != nil {
		return summary, err
	}

	seen := map[string]bool{}
	for _, row := range rows {
		url := linkedin.CanonicalURL(row.URL)
		key := strings.ToLower(url)
		name, role, company := clip(row.Name, 200), clip(row.Role, 200), clip(row.Company, 200)
		if url == "" || name == "" || seen[key] || size(url) > 2000 {
			summary.Skipped++
			continue
		}
		seen[key] = true
		email := clip(row.Email, 254)
		if a, err := mail.ParseAddress(email); email != "" && (err != nil || a.Address != email) {
			email = ""
		}
		connectedOn := sql.NullString{String: row.ConnectedOn, Valid: validDate(row.ConnectedOn)}
		message, hasMessage := lastMessages[key]
		if hasMessage {
			summary.MessagesMatched++
		}
		lastContacted := sql.NullString{String: message, Valid: hasMessage && validDate(message)}
		old := existing[key]
		if old == nil {
			slug := sql.NullString{String: matcher.Match(company)}
			slug.Valid = slug.String != ""
			if slug.Valid {
				summary.Linked++
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO connections(id,name,role,company_name,company_slug,email,url,connected_on,last_contacted_on,queued,notes,tags,revision,position,updated_at)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,false,'','{}',$10,nextval('entity_position'),now())`,
				uuid.NewString(), name, role, company, slug, email, url, connectedOn, lastContacted, uuid.NewString()); err != nil {
				return summary, dbError(err)
			}
			summary.Created++
			continue
		}
		next := *old
		next.name, next.role = name, role
		if company != old.companyName {
			next.companyName = company
			next.companySlug = sql.NullString{String: matcher.Match(company)}
			next.companySlug.Valid = next.companySlug.String != ""
		} else if !old.companySlug.Valid {
			next.companySlug = sql.NullString{String: matcher.Match(company)}
			next.companySlug.Valid = next.companySlug.String != ""
		}
		if email != "" {
			next.email = email
		}
		if connectedOn.Valid {
			next.connectedOn = connectedOn
		}
		if lastContacted.Valid && lastContacted.String > old.lastContacted.String {
			next.lastContacted = lastContacted
		}
		if next.companySlug.Valid {
			summary.Linked++
		}
		if next == *old {
			summary.Unchanged++
			continue
		}
		if err = updateImported(ctx, tx, &next); err != nil {
			return summary, err
		}
		summary.Updated++
	}
	// Message history also refreshes people who were not in this Connections.csv.
	for key, old := range existing {
		if seen[key] {
			continue
		}
		if message, ok := lastMessages[key]; ok && validDate(message) && message > old.lastContacted.String {
			summary.MessagesMatched++
			next := *old
			next.lastContacted = sql.NullString{String: message, Valid: true}
			if err = updateImported(ctx, tx, &next); err != nil {
				return summary, err
			}
			summary.Updated++
		}
	}
	if summary.Created+summary.Updated > 0 {
		if err = bump(ctx, tx, "connection"); err != nil {
			return summary, err
		}
	}
	return summary, tx.Commit()
}

func updateImported(ctx context.Context, tx *sql.Tx, c *existingConnection) error {
	_, err := tx.ExecContext(ctx, `UPDATE connections SET name=$2,role=$3,company_name=$4,company_slug=$5,email=$6,connected_on=$7,last_contacted_on=$8,revision=$9,updated_at=now() WHERE id=$1`,
		c.id, c.name, c.role, c.companyName, c.companySlug, c.email, c.connectedOn, c.lastContacted, uuid.NewString())
	return err
}

type ConnectionPhoto struct {
	MIME, Revision string
	Data           []byte
}

// SetConnectionPhoto stores or replaces a photo and returns its revision.
func (s *Store) SetConnectionPhoto(ctx context.Context, id, mime string, data []byte) (string, error) {
	if !ValidID("connection", id) {
		return "", ErrInvalid
	}
	rev := uuid.NewString()
	r, err := s.DB.ExecContext(ctx, `INSERT INTO connection_photos(connection_id,mime_type,photo,revision) SELECT id,$2,$3,$4 FROM connections WHERE id=$1
		ON CONFLICT(connection_id) DO UPDATE SET mime_type=EXCLUDED.mime_type,photo=EXCLUDED.photo,revision=EXCLUDED.revision`, id, mime, data, rev)
	if err != nil {
		return "", err
	}
	if n, _ := r.RowsAffected(); n != 1 {
		return "", ErrNotFound
	}
	return rev, nil
}

func (s *Store) ConnectionPhoto(ctx context.Context, id string) (ConnectionPhoto, error) {
	var p ConnectionPhoto
	if !ValidID("connection", id) {
		return p, ErrInvalid
	}
	err := s.DB.QueryRowContext(ctx, "SELECT mime_type,photo,revision FROM connection_photos WHERE connection_id=$1", id).Scan(&p.MIME, &p.Data, &p.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *Store) DeleteConnectionPhoto(ctx context.Context, id string) error {
	if !ValidID("connection", id) {
		return ErrInvalid
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM connection_photos WHERE connection_id=$1", id)
	return err
}
