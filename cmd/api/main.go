package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/otel"
	"github.com/michael-duren/career-strategy/internal/server"
	otelapi "go.opentelemetry.io/otel"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
func run() error {
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "hash-password" {
		password, err := io.ReadAll(io.LimitReader(os.Stdin, 73))
		if err != nil {
			return err
		}
		if len(password) > 72 {
			return fmt.Errorf("bcrypt password must be at most 72 bytes (use printf without newline)")
		}
		hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err == nil {
			fmt.Println(string(hash))
		}
		return err
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(c.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	switch command {
	case "migrate":
		return db.Migrate(ctx)
	case "check-db":
		if err := db.CheckSchema(ctx); err != nil {
			return err
		}
		fmt.Println("PostgreSQL is ready; schema version", database.SchemaVersion)
		return nil
	case "rebuild-projections":
		return db.Rebuild(ctx)
	case "export":
		return db.Export(ctx, os.Stdout)
	case "import", "seed":
		f := flag.NewFlagSet(command, flag.ContinueOnError)
		path := f.String("file", "", "version 2 workspace JSON")
		dry := f.Bool("dry-run", false, "validate and roll back")
		store := f.String("source-store", "", "source identity (required)")
		key := f.String("source-key", "content", "source key")
		revision := f.String("source-revision", "", "source ETag")
		if err = f.Parse(os.Args[2:]); err != nil {
			return err
		}
		if *path == "" || *store == "" {
			return fmt.Errorf("--file and --source-store required")
		}
		file, err := os.Open(*path)
		if err != nil {
			return err
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return err
		}
		const max = 256 << 20
		if info.Size() > max {
			return fmt.Errorf("archive exceeds 256 MiB import limit")
		}
		dir := ".migration-private/archives"
		if err = os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		if err = os.Chmod(".migration-private", 0700); err != nil {
			return err
		}
		archive, err := os.CreateTemp(dir, "source-*.json")
		if err != nil {
			return err
		}
		defer archive.Close()
		h := sha256.New()
		n, err := io.Copy(io.MultiWriter(archive, h), io.LimitReader(file, max+1))
		if err != nil {
			return err
		}
		if n > max {
			return fmt.Errorf("archive exceeds 256 MiB import limit")
		}
		if err = archive.Sync(); err != nil {
			return err
		}
		if _, err = archive.Seek(0, 0); err != nil {
			return err
		}
		absolute, _ := filepath.Abs(archive.Name())
		counts, err := db.Import(ctx, archive, database.Source{Store: *store, Key: *key, Revision: *revision, Checksum: fmt.Sprintf("%x", h.Sum(nil)), Archive: absolute}, *dry)
		if err != nil {
			return err
		}
		fmt.Printf("dry_run=%v counts=%v archive=%s\n", *dry, counts, absolute)
		return nil
	case "serve":
		// Readiness owns database/schema availability. Keep the process alive
		// during database outages, including when a pod starts during one.
		if c.PasswordHash != "" {
			if _, err = bcrypt.Cost([]byte(c.PasswordHash)); err != nil {
				return fmt.Errorf("invalid AUTH_PASSWORD_HASH: %w", err)
			}
		}
		shutdownOtel, err := otel.Setup(ctx, otel.Options{Endpoint: c.OTelEndpoint, ServiceName: c.OTelServiceName, ExportInterval: c.OTelExportInterval})
		if err != nil {
			return fmt.Errorf("setup opentelemetry: %w", err)
		}
		defer func() {
			flush, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			if err := shutdownOtel(flush); err != nil {
				log.Printf("shutdown opentelemetry: %v", err)
			}
		}()
		dbMetrics, err := db.RegisterMetrics(otelapi.GetMeterProvider())
		if err != nil {
			return fmt.Errorf("register database metrics: %w", err)
		}
		defer dbMetrics.Unregister()
		srv := server.NewServer(c, db)
		done := make(chan error, 1)
		go func() { done <- srv.ListenAndServe() }()
		log.Printf("Go API listening at %s", c.ListenAddr)
		select {
		case err = <-done:
			if err != http.ErrServerClosed {
				return err
			}
		case <-ctx.Done():
			shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			if err = srv.Shutdown(shutdown); err != nil {
				return err
			}
			err = <-done
			if err != http.ErrServerClosed {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown command %q: serve, migrate, check-db, seed, import, export, rebuild-projections, hash-password", command)
	}
}
