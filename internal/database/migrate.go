package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrations embed.FS

const SchemaVersion = 3

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
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT COALESCE(max(version),0) FROM schema_migrations").Scan(&n); err != nil {
		return err
	}
	if n > SchemaVersion {
		return fmt.Errorf("database schema is newer than application")
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, file := range files {
		name := file.Name()
		version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return err
		}
		migrationSQL, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(migrationSQL))
		var stored string
		err = tx.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version=$1", version).Scan(&stored)
		if err == nil {
			if stored != checksum {
				return fmt.Errorf("applied migration checksum mismatch")
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(migrationSQL)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", version, checksum); err != nil {
			return err
		}

	}
	return tx.Commit()
}
