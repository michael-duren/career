package database

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// maxActiveLeetgrinderTokens bounds how many unrevoked tokens can exist.
const maxActiveLeetgrinderTokens = 20

// leetgrinderTokenTouchInterval throttles last_used_at writes.
const leetgrinderTokenTouchInterval = time.Minute

// ErrUnauthorized means an API token is unknown, malformed, or revoked.
var ErrUnauthorized = errors.New("invalid or revoked API token")

// ErrTokenLimit means too many active API tokens exist to create another.
var ErrTokenLimit = errors.New("too many active API tokens")

const leetgrinderTokenColumns = "id,name,created_at,last_used_at,revoked_at"

func scanLeetgrinderToken(row interface{ Scan(...any) error }) (leetgrinder.APIToken, error) {
	var t leetgrinder.APIToken
	var used, revoked sql.NullTime
	if err := row.Scan(&t.ID, &t.Name, &t.CreatedAt, &used, &revoked); err != nil {
		return t, err
	}
	if used.Valid {
		t.LastUsedAt = &used.Time
	}
	if revoked.Valid {
		t.RevokedAt = &revoked.Time
	}
	return t, nil
}

// CreateLeetgrinderToken stores a new API token and returns its metadata and
// plaintext. The plaintext is not recoverable afterwards.
func (s *Store) CreateLeetgrinderToken(ctx context.Context, name string) (leetgrinder.APIToken, string, error) {
	name, err := leetgrinder.ValidateAPITokenName(name)
	if err != nil {
		return leetgrinder.APIToken{}, "", ErrInvalid
	}
	plain, hash, err := leetgrinder.NewAPIToken()
	if err != nil {
		return leetgrinder.APIToken{}, "", err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return leetgrinder.APIToken{}, "", err
	}
	defer tx.Rollback()
	// Serializes creators so the active-token cap holds under concurrency.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(724193612)"); err != nil {
		return leetgrinder.APIToken{}, "", err
	}
	var active int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM leetgrinder_api_tokens WHERE revoked_at IS NULL").Scan(&active); err != nil {
		return leetgrinder.APIToken{}, "", err
	}
	if active >= maxActiveLeetgrinderTokens {
		return leetgrinder.APIToken{}, "", ErrTokenLimit
	}
	token, err := scanLeetgrinderToken(tx.QueryRowContext(ctx, "INSERT INTO leetgrinder_api_tokens(id,name,token_hash) VALUES($1,$2,$3) RETURNING "+leetgrinderTokenColumns, uuid.NewString(), name, hash))
	if err != nil {
		return leetgrinder.APIToken{}, "", err
	}
	return token, plain, tx.Commit()
}

// LeetgrinderTokens lists tokens, active ones first, newest first.
func (s *Store) LeetgrinderTokens(ctx context.Context) ([]leetgrinder.APIToken, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+leetgrinderTokenColumns+" FROM leetgrinder_api_tokens ORDER BY revoked_at IS NOT NULL, created_at DESC, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tokens := []leetgrinder.APIToken{}
	for rows.Next() {
		t, err := scanLeetgrinderToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// RevokeLeetgrinderToken revokes a token. Revoking an already revoked token
// succeeds and keeps the original revocation time.
func (s *Store) RevokeLeetgrinderToken(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalid
	}
	res, err := s.DB.ExecContext(ctx, "UPDATE leetgrinder_api_tokens SET revoked_at=COALESCE(revoked_at, now()) WHERE id=$1", id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

// AuthenticateLeetgrinderToken resolves a plaintext token to its active
// record. Lookup is by SHA-256 hash, so timing reveals nothing usable about
// the stored secret; the hash is compared again in constant time. It returns
// ErrUnauthorized for unknown, malformed, or revoked tokens and records use
// at most once per minute.
func (s *Store) AuthenticateLeetgrinderToken(ctx context.Context, plain string, now time.Time) (leetgrinder.APIToken, error) {
	if !leetgrinder.WellFormedAPIToken(plain) {
		return leetgrinder.APIToken{}, ErrUnauthorized
	}
	hash := leetgrinder.HashAPIToken(plain)
	var stored []byte
	var t leetgrinder.APIToken
	var used, revoked sql.NullTime
	err := s.DB.QueryRowContext(ctx, "SELECT token_hash,"+leetgrinderTokenColumns+" FROM leetgrinder_api_tokens WHERE token_hash=$1", hash).Scan(&stored, &t.ID, &t.Name, &t.CreatedAt, &used, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return leetgrinder.APIToken{}, ErrUnauthorized
	}
	if err != nil {
		return leetgrinder.APIToken{}, err
	}
	if subtle.ConstantTimeCompare(stored, hash) != 1 || revoked.Valid {
		return leetgrinder.APIToken{}, ErrUnauthorized
	}
	if used.Valid {
		t.LastUsedAt = &used.Time
	}
	if !used.Valid || now.Sub(used.Time) >= leetgrinderTokenTouchInterval {
		// The WHERE clause repeats the throttle so concurrent requests write once.
		// A failed touch must not fail an otherwise valid request.
		res, err := s.DB.ExecContext(ctx, "UPDATE leetgrinder_api_tokens SET last_used_at=$2 WHERE id=$1 AND revoked_at IS NULL AND (last_used_at IS NULL OR last_used_at <= $3)", t.ID, now, now.Add(-leetgrinderTokenTouchInterval))
		if err != nil {
			log.Printf("leetgrinder: record token use: %v", err)
		} else if n, _ := res.RowsAffected(); n == 1 {
			t.LastUsedAt = &now
		}
	}
	return t, nil
}
