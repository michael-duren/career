package database

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

type Service interface {
	Health() map[string]string
	Close() error
}
type Store struct{ DB *sql.DB }

func Open(url string) (*Store, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	return &Store{DB: db}, nil
}
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.CheckSchema(ctx); err != nil {
		return map[string]string{"status": "down"}
	}
	return map[string]string{"status": "up"}
}
func (s *Store) CheckSchema(ctx context.Context) error {
	var n int
	if err := s.DB.QueryRowContext(ctx, "SELECT COALESCE(max(version),0) FROM schema_migrations").Scan(&n); err != nil {
		return fmt.Errorf("database/schema unavailable: %w", err)
	}
	if n != SchemaVersion {
		return fmt.Errorf("schema version %d; run migrate (expected %d)", n, SchemaVersion)
	}
	return nil
}
