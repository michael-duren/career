# Verify the weekly scheduler

From the repository root, start the local PostgreSQL service and install the locked dependencies:

```sh
make postgres-up
npm ci --ignore-scripts --no-audit --no-fund
```

Run the client and domain checks:

```sh
node --test tests/scheduler.test.ts
go test ./internal/scheduler
```

Run database tests against a separate disposable database. The tests create and remove their own schemas inside it:

```sh
psql 'postgres://career_dev:career_dev_local@127.0.0.1:5433/postgres?sslmode=disable' -c 'CREATE DATABASE career_scheduler_manual_test'
TEST_DATABASE_URL='postgres://career_dev:career_dev_local@127.0.0.1:5433/career_scheduler_manual_test?sslmode=disable' go test -v ./internal/scheduler ./internal/database ./internal/server
psql 'postgres://career_dev:career_dev_local@127.0.0.1:5433/postgres?sslmode=disable' -c 'DROP DATABASE career_scheduler_manual_test'
```

For the browser suite, run `scripts/verify-scheduler-e2e.sh`. It generates the ignored Templ Go files required to compile the server, builds the site, creates a uniquely named database, starts an authenticated local app, runs `tests/e2e/scheduler.spec.ts`, then stops the app and drops only the database it created. It rejects an occupied app port before creating the database. The default port is 4539; set `SCHEDULER_TEST_PORT` to a free port if needed. The script uses the local PostgreSQL credentials from `.env.example`; set `SCHEDULER_TEST_PG_ROOT` to another PostgreSQL server URL if needed. Run `bash tests/verify-scheduler-e2e.test.sh` to check the runner's database and port safety behavior without starting the app.

The browser project sets a 1280×1400 viewport after the Playwright device preset so the full baseline day is visible to real mouse input. Smaller viewport and scrolling behavior need their own tests.

The Google server tests inject `scheduler.GoogleClient` with an `httptest` provider and a controlled server clock. Run `go test -race ./internal/server -run SchedulerGoogle` with the same disposable `TEST_DATABASE_URL` to check worker/fetch concurrency. No live Google credentials are required. Never point these checks at shared `career_dev`. The tests create isolated schemas inside the disposable database, including all migrations.

The full scheduler browser suite covers real pointer create/move/resize previews, both resize edges, gesture cancellation, viewport scrolling, touch handles, recurring date/future changes, rejected drafts, week/save races, visible refresh and Google health, narrow-screen editors, DST folds and invalid dates, and outside-hours actual work. Run frontend builds and browser checks sequentially because they share `dist`.
