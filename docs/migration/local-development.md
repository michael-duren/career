# Local development (Go + PostgreSQL)

> Production runs on the homelab k3s cluster; Netlify is no longer used. Sections below that
> mention migration steps are historical context from the 2026-09 migration.

Start PostgreSQL, apply versioned migrations, and verify a real SQL connection:

```sh
make postgres-up migrate check-db
```

Expected final line: `PostgreSQL is ready; schema version 1`. PostgreSQL 17 binds only to `127.0.0.1:5433`, using database/user `career_dev` and local-only password `career_dev_local`. `make postgres-check` also runs `pg_isready` inside the container. The named volume persists across container restarts and `docker compose down`; no command above removes it. A custom `POSTGRES_LOCAL_PORT` also requires a matching `DATABASE_URL`.

## Run the API

```sh
make go-dev
```

Then, in another terminal:

```sh
curl --fail http://127.0.0.1:8080/readyz
```

`/healthz` reports process liveness; `/readyz` checks PostgreSQL and schema compatibility and returns 503 during a database outage without exiting the process. The old `/health` is a readiness alias. Startup fails with a useful error if migrations have not been applied. Run `migrate` explicitly before each release; startup never changes the schema or seeds data.

Go serves both the authenticated APIs and the static Astro build from `STATIC_DIR` (run `npm run build` first to produce `dist/`). `npm run dev` runs the Astro dev server for frontend-only work; build and run Go for authenticated API testing.

## Authentication and configuration

Run `npm ci` and `npm run setup` to create `.env`. Press Enter to use the local
login `admin` / `password123`, or answer `n` to choose a username and password.
The script hashes the password, generates a random JWT secret, and uses the local
Docker Compose database credentials automatically. Existing `.env` files are kept.
For no prompts, use `npm run setup -- --defaults`. Copying `.env.example` also
works for local development with the same login and a fixed development JWT secret.

The Go executable reads `.env` without overwriting process environment. Supported configuration: `APP_ENV`, `DATABASE_URL`, `AUTH_USERNAME`, `AUTH_PASSWORD_HASH`, `JWT_SECRET`, `PUBLIC_ORIGIN`, `LISTEN_ADDR`, `STATIC_DIR`. Development defaults are explicitly loopback-only. Production must supply database URL, listen address, username, public origin, bcrypt password hash and a non-placeholder JWT secret of at least 32 characters. Use a credential-appropriate PostgreSQL URL and explicit TLS mode for the homelab later.

To generate a bcrypt hash locally without putting a password in shell history:

```sh
read -rs 'migration_password?Local login password: '
printf '%s' "$migration_password" | go run ./cmd/api hash-password
unset migration_password
```

Put the output in `.env` as a **single-quoted** `AUTH_PASSWORD_HASH` to preserve its dollar signs. Setup handles this quoting automatically. An unset development password hash disables login; a malformed configured hash is a startup error.

Login is `POST /api/auth/login` with JSON `{ "username": "...", "password": "..." }` and exact `Origin: http://localhost:8080` (the development `PUBLIC_ORIGIN`; override both when using another origin). It sets an HttpOnly, SameSite=Lax, Path=/ cookie, Secure for HTTPS, with a 24-hour HS256 lifetime. `GET /api/auth/verify` verifies it; `POST /api/auth/logout` clears it. No bearer/localStorage token is returned. Failed/successful login attempts are bounded per direct peer address; forwarded headers are not trusted.

## Scoped API contract

For `kind` use `note`, `document`, `personal`, `week`, `book`, `company`, or `goal`:

- `GET /api/entries/{kind}?limit=50&offset=0` → `{ entries: [{ entry, revision }], nextOffset: number | null }`. Maximum 100, deterministic domain ordering. Lists omit Markdown bodies and children. Book/company lists include persisted summaries.
- `GET /api/entries/{kind}?id=<URL-encoded-ID>` → `{ entry, revision }`. IDs may contain slashes. Detail reads retrieve only that entity and its children.
- `POST /api/entries/{kind}` with `{ entry, revision }` → changed `{ entry, revision }`. Null revision means insert-if-absent. Existing revisions must match; unrelated entities do not conflict. Normalized saves use server timestamps and append to the legacy collection order.
- `DELETE /api/entries/{kind}` with `{ id, revision }` → `{ deleted: true }`. Core document deletion is rejected.
- `POST /api/entries/book/toggle` (or company) with `{ id, revision, index, checked }` → changed entity. The source-line index must identify a task in the matching revision, outside a fenced block.
- `GET /api/export` → streamed legacy workspace JSON under one repeatable-read snapshot. This is an explicit export, never a routine editor fetch. Entity revisions stay in the scoped API; metadata change sequence is internal export provenance rather than a global write token.

Notes support `topic`; books/companies support `status`, `category`, `priority`; goals support inclusive overlap `from`/`to` date filters. Other filters are rejected. Mutations require exact configured Origin and JSON content type. Save reads are capped at 150,000 bytes, deletes/toggles at 2,000, login at 4,096. Conflict is 409, missing entity 404, validation 400, storage failure 503. Private responses are no-store. The Go router serves the read-only MCP endpoint and its OAuth routes; see [MCP](../mcp/README.md).

## Seeds, imports, and exports

```sh
make seed-export
make seed-dry-run
make seed-import
```

Run import only against an **empty** workspace. Seed export uses installed Astro 6.1.7 `sync` and its schema-validated content cache, then the same extracted seed and journal migration functions as the current application. Missing required seed collections fail the command. Files go in ignored `.migration-private/`, never `public/` or `dist/`. The Astro cache format is checked and tied to the installed/locked version; revisit this exporter on an Astro upgrade.

A normalized, version-2 production export can later use:

```sh
go run ./cmd/api import --file /restricted/path/content.json --source-store personal-workspace --source-key content --source-revision 'THE_EXACT_ETAG' --dry-run
```

Drop `--dry-run` only for the intended empty target. The command archives exact source bytes with mode 0600 beneath a mode-0700 private directory, records SHA-256/store/key/ETag/schema markers, acquires an import lock, validates one entity at a time, and writes everything in one transaction. Dry runs roll back SQL writes. Unknown fields/versions/invalid values fail rather than being normalized or silently dropped. Any populated workspace is rejected, including a repeated import; no merge, upsert restore, or automatic reseed occurs. SQL migration history includes checksums and an advisory lock. Dry runs can consume sequence numbers; position gaps have no semantic effect.

Imports have an explicit **256 MiB archive limit**, independent of the 150 KB interactive request limit and the old 4 MB aggregate cap. Full exports have no aggregate byte limit and stream bounded batches; HTTP exports are subject to the server's 30-second write timeout, so use the CLI for large backups. Check command exit status before accepting an export; a broken HTTP stream is not a valid backup. Preserve exact private original archives alongside database backups to retain source provenance.

Export securely with a restrictive shell umask:

```sh
(umask 077; go run ./cmd/api export > .migration-private/export-v2.json)
```

`make rebuild-projections` explicitly recomputes book/company summaries, ordered checklist/log rows and parser versions from authoritative Markdown. It locks owner rows and updates projections transactionally. Normal reads never parse every Markdown body. Detail APIs return original Markdown as JSON; rendering/sanitization stays with requested detail views when the frontend is migrated.

## Verification performed (2026-09-14, during migration)

- `make test-postgres`: Go tests and race detector, using unique disposable schemas in the local database (not deleting the development workspace).
- `npm test`: 47 existing tests passed.
- `npx astro check`: zero errors/warnings, 73 existing hints.
- `npm run build`: build passed (Netlify server build at the time; now a static build).
- Seed dry-run and committed local seed import passed: 20 books, 28 companies, 13 documents, 4 notes, 13 work weeks, 1 personal journal.
- Fixture tests cover relational round trips, source timestamp spellings, optional presence, Unicode, exact Markdown, source-parser parity, concurrency, transaction rollback, core protection, stale toggles, consistent export snapshots, reconnect persistence, and query plans.

To run the HTTP smoke test:

```sh
go build -o /tmp/career-migration ./cmd/api
node tests/go-api.integration.mjs
```

It uses port 8088, temporary local auth credentials, creates/removes one synthetic note, and shuts down the Go process. It does not exercise homelab ingress. `node tests/pwa.browser.mjs` is the browser smoke test against a running Go server.
