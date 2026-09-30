# Retired: the Leetgrinder 84-day curriculum

Leetgrinder began as a twelve-week data structures and algorithms curriculum: 84 numbered sessions, one a day, with 252 core and 48 optional LeetCode problems, a lesson and readings for each session (with a side-by-side example player for weeks 1 to 4), a calendar schedule with ahead/behind status, and "Finish day" completion. It was replaced by a tracker for any LeetCode problem with a daily goal and spaced-repetition reviews (see `docs/superpowers/specs/2026-09-30-leetgrinder-tracker-design.md`).

Everything here is kept for later and is not built, tested, or generated:

- `retired/go.mod` declares a separate stub module, so `go build ./...`, `go vet ./...` and `go test ./...` in the main module skip this directory.
- Templates are renamed `*.templ.retired`, so `go tool templ generate` ignores them.
- The npm scripts `test:leetgrinder:player` and `test:leetgrinder:browser` were removed; their tests are in `tests/`.

Layout mirrors the repository: `internal/leetgrinder/` (curriculum, readings, lessons, examples, the lesson player, day and about page templates in `views_curriculum.templ.retired`, and the schedule and dashboard code they used), `docs/`, `scripts/leetgrinder/`, `cmd/verify-leetgrinder/`, `cmd/leetgrinder-preview/`, and `tests/`. The curriculum-era specs and plans stay in `docs/superpowers/` as history.

The last commit where the curriculum ran is `f73111e5f57ee3976bd0f5434e734d58adb3a9fa`.

The curated optimal complexities of all 300 problems live on in `internal/leetgrinder/catalog_seed.json`, which seeds the problem catalog.

## Reviving it

1. Move `internal/leetgrinder/*` back into the package, rename `*.templ.retired` to `*.templ`, and restore the day, about and lesson routes in `internal/server/leetgrinder.go` (compare with the commit above).
2. Add a migration that recreates `leetgrinder_completed_days (day INTEGER PRIMARY KEY CHECK (day BETWEEN 1 AND 84))`, restore `SetLeetgrinderDay`, and add `CompletedDays` back to the state loader and export.
3. Add the schedule settings back: `start_date DATE` and `daily_hours` on `leetgrinder_settings`, the schedule form and `/leetgrinder/settings/schedule` route, and the schedule gates and `behind_schedule` reminder in the notification worker.
4. Restore the two npm scripts and move `scripts/leetgrinder/`, `cmd/verify-leetgrinder/` and `cmd/leetgrinder-preview/` back.
