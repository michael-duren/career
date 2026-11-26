package database

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/modules"
)

// migrateThrough applies the migrations up to and including version.
func migrateThrough(t *testing.T, s *Store, version int) {
	t.Helper()
	if _, err := s.DB.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		v, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil {
			t.Fatal(err)
		}
		if v > version {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.DB.Exec(string(sql)); err != nil {
			t.Fatalf("%s: %v", file.Name(), err)
		}
		if _, err = s.DB.Exec("INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", v, fmt.Sprintf("%x", sha256.Sum256(sql))); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAppSettingsNewInstall(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	got, err := s.AppSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.SetupCompletedAt != nil || got.Theme != "dark" || len(got.Order) != len(modules.Registry) {
		t.Fatalf("new install: %+v", got)
	}
	for id, on := range got.Modules {
		if !on {
			t.Errorf("%s off by default", id)
		}
	}

	theme, at := "vellum", time.Date(2026, 11, 2, 15, 4, 5, 0, time.UTC)
	saved, err := s.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"timeline": false, "notes": false}, Order: []string{"notes", "leetgrinder"}, Theme: &theme, SetupCompletedAt: &at})
	if err != nil {
		t.Fatal(err)
	}
	got, err = s.AppSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Modules["timeline"] || got.Modules["notes"] || !got.Modules["schedule"] || got.Enabled("schedule") || got.Order[0] != "notes" || got.Order[1] != "leetgrinder" ||
		got.Theme != "vellum" || got.SetupCompletedAt == nil || !got.SetupCompletedAt.Equal(at) || !saved.SetupCompletedAt.Equal(at) {
		t.Fatalf("saved: %+v", got)
	}

	if _, err = s.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"agents": false}}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"agents": true}}); !errors.Is(err, ErrInvalid) || err.Error() != "Agent context needs Timeline" {
		t.Fatal("turned on a module whose dependency is off:", err)
	}
	// A rejected change saves nothing.
	if got, _ = s.AppSettings(ctx); got.Modules["agents"] {
		t.Fatal("rejected change was saved")
	}
}

func TestAppSettingsMigrationKeepsExistingWorkspaces(t *testing.T) {
	ctx := context.Background()
	// Any install migrated before this release counts as existing, with or
	// without content.
	for _, content := range []bool{true, false} {
		existing := emptyStore(t)
		migrateThrough(t, existing, 35)
		if content {
			if _, err := existing.DB.Exec(`INSERT INTO leetgrinder_todo_sets(id,title) VALUES('11111111-1111-4111-8111-111111111111','A')`); err != nil {
				t.Fatal(err)
			}
		}
		if err := existing.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		got, err := existing.AppSettings(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got.SetupCompletedAt == nil {
			t.Fatalf("content=%v: an existing install must skip setup", content)
		}
		for _, id := range modules.IDs() {
			if !got.Enabled(id) {
				t.Errorf("%s off after migration", id)
			}
		}
	}

	// A new database migrates in one transaction and still needs setup.
	fresh := testStore(t)
	if got, err := fresh.AppSettings(ctx); err != nil || got.SetupCompletedAt != nil {
		t.Fatal("a new install still needs setup", got, err)
	}
}

func TestAppSettingsUnreadableRowFails(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if _, err := s.DB.Exec(`INSERT INTO app_settings(id,modules) VALUES(1,'{"notes":"yes"}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppSettings(ctx); err == nil {
		t.Fatal("an unreadable settings row must be an error, not defaults")
	}
}
