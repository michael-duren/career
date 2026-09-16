package database

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
)

//go:embed migrations/*.sql
var migrations embed.FS

const SchemaVersion = 1

func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(724193601)"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())"); err != nil {
		return err
	}
	sql, err := migrations.ReadFile("migrations/001_workspace.sql")
	if err != nil {
		return err
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(sql))
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT COALESCE(max(version),0) FROM schema_migrations").Scan(&n); err != nil {
		return err
	}
	if n > SchemaVersion {
		return fmt.Errorf("database schema is newer than application")
	}
	if n == 1 {
		var stored string
		if err = tx.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version=1").Scan(&stored); err != nil {
			return err
		}
		if stored != checksum {
			return fmt.Errorf("applied migration checksum mismatch")
		}
	} else {
		if _, err = tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version,checksum) VALUES(1,$1)", checksum); err != nil {
			return err
		}
	}
	return tx.Commit()
}
