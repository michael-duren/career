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
psql 'postgres://career_dev:career_dev_local@127.0.0.1:5433/career_dev?sslmode=disable' -c 'CREATE DATABASE career_scheduler_manual_test'
TEST_DATABASE_URL='postgres://career_dev:career_dev_local@127.0.0.1:5433/career_scheduler_manual_test?sslmode=disable' go test -v ./internal/database -run 'Scheduler|ImportDoesNotFabricate'
psql 'postgres://career_dev:career_dev_local@127.0.0.1:5433/career_dev?sslmode=disable' -c 'DROP DATABASE career_scheduler_manual_test'
```

For the browser suite, run `scripts/verify-scheduler-e2e.sh`. It generates the ignored Templ Go files required to compile the server, builds the site, creates a uniquely named database, starts an authenticated local app, runs `tests/e2e/scheduler.spec.ts`, then stops the app and drops that database. It uses port 4539 by default; set `SCHEDULER_TEST_PORT` if that port is occupied. The script uses the local PostgreSQL credentials from `.env.example`; set `SCHEDULER_TEST_PG_ROOT` to another PostgreSQL server URL if needed.

The browser project sets a 1280×1400 viewport after the Playwright device preset so the full baseline day is visible to real mouse input. Smaller viewport and scrolling behavior need their own tests.
