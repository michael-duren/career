# Migration todos: Netlify → Go, PostgreSQL, and homelab k3s

## General Notes

`migration.md` is both the migration instructions and the authoritative migration state file. Keep progress here, not only in chat, agent memory, or a separate todo file. Supporting documents may hold detailed evidence, but this file must identify the current item, outcome, remaining work, and exact next action.

This directory is the Git worktree for the `migration` branch and its [migration pull request](https://github.com/michael-duren/career-strategy/pull/1). Do not implement numbered migration steps directly in this worktree. For each numbered step (example: 2. Establish the Go service and relational PostgreSQL model
), create a separate Git worktree on a dedicated topic branch based on the current `migration` branch, complete and validate that step there, after work is complete update the migration.md in the `migration` worktree, the user will review and commit the work, making a pull request themselves into the migration branch.

Because this repository uses a bare-worktree layout, start in `migration/`, then go up one directory to the repository container before adding a worktree from `../.bare` (for example, `cd .. && git --git-dir=.bare worktree add <worktree-path> -b <topic-branch> migration`). Return to the new worktree to implement the step.

### Worker loop and state rules

1. Read these General Notes, the Migration state below, and the assigned numbered section (including its file map, validation, and item records). Stable IDs such as `3.1` identify individual checkboxes; `C.1` identifies a completion criterion. Start at the recorded item unless the user assigns another. Inspect `git status --short` and preserve unrelated work. Use the listed files and follow their direct imports only as needed; if a location moved or a dependency is missing, search the relevant directory and record the corrected path here.
2. Record the active step/item and status before editing. Read the original checkbox instructions and any appended differences together. Implement the item, including required build/test coverage and fixes for migration errors. Reuse completed foundation work called out below. If an item needs a later step to pass end-to-end validation, leave it unchecked, record that dependency, and continue only with independent work.
3. **The user executes any command expected to take more than a couple of seconds (use two seconds as the threshold).** This includes builds/checks/test suites, dependency installation, database startup/migrations/imports/exports/restores, image builds/imports, Ansible applies, deployment/rollout waits, and performance/browser runs. Prepare the files first, then provide a copyable command with the exact working directory, target environment, prerequisites, expected success result, and the short output/exit status needed back. Record this handoff under the item and in Migration state. Do not launch these commands in the background or poll them to save tokens. If an unexpectedly slow command has already started, record it and avoid repeated polling or duplicate execution; do not interrupt a database operation unsafely.
   **Package every user-run command in a checked-in shell script.** The user must not have to copy multi-line commands from chat or reconstruct commands from prose. Give the script safe subcommands when a workflow has multiple phases, validate its shell syntax, make it executable, and record only the short script invocation in `migration.md` and chat. Scripts must prompt securely for required secrets or use the repository's existing documented secret mechanism; never invent a secret-management dependency.
4. Run quick, bounded inspections locally. For required long build/test checks, let the user execute them; inspect their reported results, fix migration-related failures, and hand back only the necessary reruns. A prepared command, earlier foundation test, skipped check, or missing user result is not a pass. Keep the item `awaiting-user-command` until required results arrive. While waiting, work only on independent items and preserve the pending item in state.
5. **Immediately after each checkbox is finished**, change only its marker from `[ ]` to `[x]` and append a dated record beneath that item. Record changed file paths, what was implemented, validation commands/environment/results (including whether user-reported), and any remaining dependencies. Preserve the original instruction text and stable ID. If anything differs from the plan, append `Differences:` with the original expectation, actual behavior, reason, and follow-up item; never silently replace the original requirement or mark an unmet requirement complete. Record failures and partial work even when the checkbox stays open. A waived requirement needs an explicit user decision recorded here.
6. Update Migration state after every item and before any pause, handoff, or context compaction. At each item boundary, save a concise checkpoint, then compact context when supported. If compaction is not available, leave the same checkpoint for the next context; do not claim compaction happened. After compaction, resume by rereading this file rather than scanning the repository. Mark a numbered section complete only when all its checkboxes and required validation are complete; mark final criteria only with the linked acceptance evidence.

Use this record beneath an item when work starts (it is a template, not a new checklist):

```text
State (YYYY-MM-DD): in-progress | awaiting-user-command | blocked | complete
Changed files: repository-relative paths; absolute paths for other repositories.
Implemented/evidence: concise outcome; link to detailed evidence if needed.
Validation: command, cwd, target, result/exit status, date, agent-run or user-reported.
Differences: none, or expectation → actual behavior, reason, and follow-up item.
Pending user command: exact command + cwd + prerequisites + expected output; or none.
Next action/dependencies: exact remaining action and item IDs; or none.
```

Do not store passwords, tokens, connection strings containing secrets, private exports, or private content in this file or Git. Record private artifact paths and nonsecret checksums instead. Load secrets through the documented environment mechanism. External repository paths below are references and intended edit locations; follow the permissions applicable to those repositories.

### Migration state

- Last updated: 2026-09-14 (step 6 deployed; live LAN checks passed).
- Active numbered step / next item: **6 / 6.4, 6.7–6.8**, status **awaiting-external-acceptance**. Items **6.1–6.3 and 6.5–6.6 are complete**. Implementation merged into migration; tested release remains topic commit 684ce848098bb563cdcd54575582b691c7b507bd.
- Completed: steps **1–4**, step 5 items **5.1 and 5.3–5.5**. Items **5.2 and 5.6** still need app-pod and populated-restore/pre-cutover evidence.
- Current environment: career deployed in context default, namespace career-strategy. Pod career-fc6559c8b-qrmn2 is Ready with zero restarts on k8s-w-2; migration Job career-migrate-9vdjw completed on k8s-w-1. Agent-run app-pod check-db reports PostgreSQL ready, schema version 1. LAN ingress with Host career.homelab returns healthz/readyz 200, protected HTML 303 to login and unauthenticated API 401/private,no-store. Public routing has not changed.
- Validation received (2026-09-14): user reported `./scripts/deploy-homelab.sh check` reached its final PASS line. Static build produced 17 pages; Go race-enabled source tests and HTTP smoke passed (readiness, auth, scoped CRUD, conflicts, origin checks, core protection and excluded MCP routes). Script completion also confirms preceding dependency/Astro/TypeScript/domain checks succeeded. Real-PostgreSQL race suite is not yet accepted.
- Validation received (2026-09-14, subsequent user report): `./scripts/deploy-homelab.sh check-postgres` passed against local career_dev on port 5433. `go test -race ./...` reported database tests passing in 1.847s, config/server cached passes, and cmd/api with no test files. This supersedes the pending PostgreSQL race status above.
- Validation received (2026-09-14, user-reported): image `career-strategy:684ce848098bb563cdcd54575582b691c7b507bd` built and runtime inventory passed; container stayed alive with unavailable DB, readiness returned 503, and SIGTERM exited 0. Item 6.1 is complete; deployed outage/recovery remains open under 6.4.
- Validation update (2026-09-14, user-reported): server manifest validation passed from migration at 752940ce1e91e66954c412c320ce7b9db748b612. Deployment failed before image import with missing local image career-strategy:752940ce1e91e66954c412c320ce7b9db748b612. Step 6 was merged, but the built/tested image is tagged with topic commit 684ce848098bb563cdcd54575582b691c7b507bd. Agent confirmed the topic worktree is clean at that commit and the two commits differ only in migration.md; no runtime differences.
- Validation update (2026-09-14, user-reported): deploy PASS for career-strategy:684ce848098bb563cdcd54575582b691c7b507bd, previous=none. Migration Job completed and rollout succeeded. Independent agent live checks above passed at 21:17 UTC.
- Pending user command: none immediately; next external dependency is dedicated career tunnel setup and a staging HTTPS origin before final production cutover. Do not repeat deploy or secrets.
- Next action: establish whether a dedicated career Cloudflare tunnel already exists, then prepare its pinned-image and staging route handoff. Deployed DB-outage/recovery acceptance remains tied to 7.10. Step 5 pod-check still needs exact DB identity/source-IP evidence: existing psql command is incompatible with this runtime, although app-pod check-db now proves authenticated connectivity/schema readiness. Record acceptance here to keep the release worktree clean.
- Known dependencies: tunnel token, reviewed/tested cloudflared digest, confirmed LAN DNS allocation and external HTTPS acceptance remain pending. Staging outage/recovery and public routing acceptance depend on steps 7–8. Preserve step 5's pending pod/restore work. Step 6 is not complete.

### File and validation conventions

Paths are relative to `/home/mduren/Code/career-strategy/migration` unless absolute. Directory paths name a bounded group to inspect, not an invitation to scan the whole repository. **Create** means a proposed file does not exist yet; update this map if implementation chooses another location. Generated `dist/`, `.astro/`, and `.migration-private/` are inspection/output locations, not source files to edit. Each numbered section has a local file map keyed to its checkbox IDs and a validation handoff; read those alongside the original requirements.

All commands in validation handoffs below are **user-run** under the two-second rule. Unless stated otherwise, cwd is `/home/mduren/Code/career-strategy/migration`. Prepare scripts/config first and record exact commands and prerequisites under the active item before asking the user to run them. Do not treat planned scripts or legacy browser suites as already compatible with Go.

## Plan

Plan based on the application and homelab repositories inspected on 2026-09-14. Steps 1–2 were implemented and verified locally on 2026-09-14. Local PostgreSQL was started and seeded; no production infrastructure or data was changed. See [captured behavior and production inputs](docs/migration/current-behavior.md) and [local commands/API/validation results](docs/migration/local-development.md).

Target: one Go service serves the static Astro build and same-origin browser APIs, with PostgreSQL on the existing homelab database VM. Keep Astro, React islands, Tailwind, Markdown editing, and the existing single-user workflow. Node is needed for building and frontend development, not in the production runtime.

**Existing Go boilerplate:** A Go Blueprint application using Chi already exists in this repository. Build on the existing `go.mod`, `cmd/api/main.go` entry point, `internal/server/` HTTP server and Chi routes, and `internal/database/` PostgreSQL service (using `pgx` through `database/sql`). The scaffold also includes Go tests, `docker-compose.yml`, `.air.toml`, and Makefile build/run/test targets. Extend this application in place; do not generate another Go module or a parallel server entry point. The generated `/` and `/health` handlers, permissive CORS, and `BLUEPRINT_DB_*` configuration are starter behavior to adapt during the migration, not completed application features.

**Out of scope:** MCP and its OAuth integration. Do not port MCP endpoints, discovery, token validation, or client integration during this migration. Browser username/password login and protected routes remain in scope. Keep the ordinary session-authenticated JSON/Markdown agent-context export.

## 1. Capture the current behavior and migration inputs

### Files and verification for step 1

| Items | Read / update locations                                                                                                                                                                                                                                                                                                                                                             |
| ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1.1   | `src/pages/api/workspace.ts`, `src/pages/api/goals.ts`, `src/pages/api/agent-context.ts`, `src/lib/workspace.ts`, `src/lib/workspace-server.ts`, `src/lib/agent-context.ts`; contract record: `docs/migration/current-behavior.md`.                                                                                                                                                 |
| 1.2   | `src/pages/index.astro`, `src/pages/books.astro`, `src/pages/companies.astro`, `src/pages/companies/[...slug].astro`, `src/pages/logs.astro`, `src/pages/progress.astro`, `src/pages/agents.astro`, `src/pages/login.astro`, `src/pages/notes/`, `src/pages/documents/`, `src/pages/manage/[kind].astro`, `src/pages/2026/`; route inventory: `docs/migration/current-behavior.md`. |
| 1.3   | `scripts/capture-migration-fixtures.mjs`, `tests/fixtures/migration/workspace-v2.json`, `tests/fixtures/migration/expected-projections.json`, `src/lib/books.ts`, `src/lib/companies.ts`, `src/lib/progress.ts`, `src/lib/timeline.ts`, `src/components/WorkspaceEditor.tsx`; evidence: `docs/migration/current-behavior.md`.                                                       |
| 1.4   | `scripts/inspect-migration-source.mjs`, `src/lib/workspace-server.ts`, `netlify.toml`, `docs/migration/current-behavior.md`; private Netlify CLI credentials are read by the script, never copied into this file.                                                                                                                                                                   |
| 1.5   | `docs/migration/current-behavior.md`, `netlify.toml`, `/home/mduren/Code/rubber-duck/deploy/homelab/ingress.yaml`, `/home/mduren/Code/rubber-duck/deploy/homelab/cloudflared.yaml`, `/home/mduren/Code/home-infra/infra/ansible/inventory.ini`; live DNS/provider state must be verified at deployment.                                                                             |

Validation handoff if revisiting: `node scripts/capture-migration-fixtures.mjs` regenerates synthetic expectations (review the diff); `node scripts/inspect-migration-source.mjs` requires the intended production Netlify CLI login and reports metadata only. Compare contracts/fixtures to the captured document. No build is needed for documentation-only changes; source changes use the relevant step 2–4 checks.

Completed: contracts, route inventory, synthetic fixtures/parser expectations, live production identity/schema/ETag/counts, public DNS and deployment inputs are recorded in `docs/migration/current-behavior.md`. LAN hostname allocation must still be confirmed immediately before deployment; failed resolution alone is not proof of availability. Interactive caller changes are tracked in steps 3–4.

- [x] **1.1.** Record the browser API contracts from `src/pages/api/workspace.ts`, `goals.ts`, and `agent-context.ts`: methods, payloads, response shapes, validation, status codes, and revision behavior. Document the existing `{ data, revision }` and `{ goals, revision }` contracts, then migrate interactive callers to the scoped reads and entity revisions in steps 2–3. Keep a full-workspace compatibility/export adapter only where needed.
- [x] **1.2.** Inventory every route that reads `readWorkspace()` or `getJournalEntries()`, including home, books, companies/detail, logs, progress, agents, documents, and reference pages under `/2026/`. Inventory request-dependent behavior in login, notes/detail, documents, and `/manage/[kind]`.
- [x] **1.3.** Capture representative fixtures and expected results for notes, work journals, personal journals, books/courses, companies/contacts, documents, and timeline goals. Include nested IDs, Unicode, Markdown/HTML, deleted seed entries, timestamps, optional fields, and unsaved-draft/conflict behavior.
- [x] **1.4.** Identify the production Netlify site/store and current workspace versions. Production uses `personal-workspace` / `content`; nonproduction builds currently derive isolated `personal-workspace-preview-*` stores. Never use a preview export as the production source.
- [x] **1.5.** Record the current public hostname, DNS route, and deployment settings for cutover and rollback. Use `career.homelab` as the proposed LAN hostname; confirm its availability before deployment. Prefer retaining the current public origin to preserve bookmarks and browser-local drafts.

## 2. Establish the Go service and relational PostgreSQL model

### Files and verification for step 2

| Items   | Read / update locations                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 2.1     | `go.mod`, `go.sum`, `cmd/api/main.go`, `internal/server/server.go`, `internal/server/routes.go`, `internal/server/auth.go`, `internal/database/database.go`.                                                                                                                                                                                                                                                                                                 |
| 2.2     | `internal/config/config.go`, `internal/config/config_test.go`, `.env.example`, `internal/server/server.go`, `internal/database/database.go`; timing evidence: `docs/migration/current-behavior.md`, `docs/migration/local-development.md`.                                                                                                                                                                                                                   |
| 2.3–2.5 | `internal/database/migrations/001_workspace.sql`, `internal/database/model.go`, `internal/database/children.go`, `internal/database/transfer.go`; source fields: `src/lib/workspace.ts`, `src/lib/workspace-server.ts`, `src/lib/books.ts`, `src/lib/companies.ts`, `src/lib/progress.ts`, `src/lib/timeline.ts`; fixtures: `tests/fixtures/migration/`. Add new versioned SQL files under `internal/database/migrations/` for changes to an applied schema. |
| 2.6     | `internal/database/migrations/001_workspace.sql`, `internal/database/store.go`, `internal/database/database_test.go` (query-plan checks).                                                                                                                                                                                                                                                                                                                    |
| 2.7     | `internal/database/projections.go`, `internal/database/children.go`, `internal/database/store.go`, `cmd/api/main.go`; parser references: `src/lib/books.ts`, `src/lib/companies.ts`, `src/lib/checklist.ts`, `src/lib/logs.ts`; `tests/fixtures/migration/expected-projections.json`.                                                                                                                                                                        |
| 2.8–2.9 | `internal/database/store.go`, `internal/database/children.go`, `internal/database/transfer.go`, `internal/server/routes.go`, `internal/database/database_test.go`.                                                                                                                                                                                                                                                                                           |
| 2.10    | `internal/database/model.go`, `internal/database/transfer.go`, `internal/server/routes.go`, `internal/database/model_test.go`, `tests/fixtures/migration/workspace-v2.json`.                                                                                                                                                                                                                                                                                 |
| 2.11    | `internal/database/migrate.go`, `internal/database/migrations/`, `cmd/api/main.go`, `Makefile`.                                                                                                                                                                                                                                                                                                                                                              |
| 2.12    | `cmd/api/main.go`, `internal/database/transfer.go`, `scripts/export-seed.mjs`, `src/lib/workspace-seed.ts`, `src/content.config.ts`, `src/content/notes/`, `src/content/docs/`, `src/content/books/`, `src/content/companies/`, `src/content/progress/`, `Makefile`; generated private artifact: `.migration-private/seed-v2.json`.                                                                                                                          |

Validation handoff: `make postgres-up migrate check-db` for the local development database, then `make test-postgres` (real PostgreSQL/race checks), `go build -o /tmp/career-migration ./cmd/api`, and `node tests/go-api.integration.mjs`. Configuration/model/handler tests live in `internal/config/config_test.go`, `internal/database/model_test.go`, `internal/server/routes_test.go`; full DB tests in `internal/database/database_test.go`. Follow `docs/migration/local-development.md` for empty-target seed/round-trip commands; do not rerun seed import against the already seeded workspace. Preserve historical evidence and append new outcomes per item.

Completed locally: versioned relational migrations, validated configuration, scoped authenticated Go APIs, entity concurrency, transactional children/projections, streaming import/export, explicit seeds and projection rebuild, and real-PostgreSQL/race/round-trip checks. Run `make postgres-up migrate check-db` to verify PostgreSQL. Source timing and indexed SQL plans are recorded; full browser/ingress performance acceptance remains step 7. The current Astro UI still uses Netlify until steps 3–4.

- [x] **2.1.** Extend the existing Go Blueprint/Chi application at `cmd/api/`, `internal/server/`, and `internal/database/` with HTTP handlers, browser authentication, workspace operations, and PostgreSQL persistence. Retain Chi with `net/http` and `pgx`; add explicit SQL and versioned SQL migrations.
- [x] **2.2.** Add configuration for `DATABASE_URL`, `AUTH_USERNAME`, `AUTH_PASSWORD_HASH`, `JWT_SECRET`, `PUBLIC_ORIGIN`, listen address, and static build directory. Validate required production settings at startup and keep local development defaults explicitly local.
      The reported 1–2 second data interactions are a migration priority. Current handlers load the entire Blobs workspace, and book/company views derive data by parsing Markdown. Measure storage, decoding, Markdown processing, SQL, response size, and browser rendering separately to verify the bottlenecks. Do not reproduce whole-workspace reads/writes in PostgreSQL: relational storage and scoped APIs are required in this migration, not a later optimization.

- [x] **2.3.** Implement the following concrete tables in versioned SQL migrations. Match fields against `src/lib/workspace.ts`, `workspace-server.ts`, `books.ts`, `companies.ts`, `progress.ts`, and `timeline.ts` before import. Use SQL columns for structured fields and `TEXT` for original Markdown; no singleton workspace JSONB payload or per-entity JSON payload standing in for these tables.

| Table                      | Columns and relationships                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `workspace_metadata`       | Singleton `id SMALLINT PRIMARY KEY CHECK (id = 1)`, source `version INTEGER`, nullable `catalog_version INTEGER`, nullable `journals_version INTEGER`, `change_sequence BIGINT NOT NULL`, `updated_at TIMESTAMPTZ`. Metadata and export provenance only; no content document.                                                                                                                                                                                                                 |
| `notes`                    | `id TEXT PRIMARY KEY`, `title TEXT`, `topic TEXT`, `description TEXT`, `tags TEXT[]`, `body TEXT`.                                                                                                                                                                                                                                                                                                                                                                                            |
| `documents`                | `id TEXT PRIMARY KEY`, `title TEXT`, `description TEXT`, `tags TEXT[]`, `body TEXT`, `is_core BOOLEAN` for delete protection.                                                                                                                                                                                                                                                                                                                                                                 |
| `personal_journal_entries` | `id TEXT PRIMARY KEY`, `title TEXT`, `entry_date DATE NULL`, `description TEXT`, `tags TEXT[]`, `body TEXT`. Map the existing undated `date: ""` to SQL NULL and back.                                                                                                                                                                                                                                                                                                                        |
| `journal_weeks`            | `slug TEXT PRIMARY KEY`, `week INTEGER`, `year INTEGER`, original `dates TEXT`, validated `start_date DATE`, `end_date DATE`, `tags TEXT[]`, `body TEXT`, `targets_present BOOLEAN`. Preserve the original dates string for export.                                                                                                                                                                                                                                                           |
| `journal_week_metrics`     | `week_slug TEXT REFERENCES journal_weeks(slug) ON DELETE CASCADE`, `track TEXT`, nullable `hours NUMERIC`, nullable `target NUMERIC`; primary key `(week_slug, track)`. At least one value must exist. Preserve arbitrary historical track keys and distinguish absent values from zero.                                                                                                                                                                                                      |
| `books`                    | `slug TEXT PRIMARY KEY`, `title TEXT`, nullable `edition TEXT`, `authors TEXT[]`, `category TEXT`, `type TEXT` (book/course), nullable `url TEXT`, `cover TEXT`, `isbn TEXT`, `status TEXT`, `featured BOOLEAN`, `priority TEXT`, `tags TEXT[]`, `body TEXT`, nullable `started DATE`, `finished DATE`, `rating NUMERIC`. Optional explicit progress uses nullable `progress_unit TEXT`, `progress_total INTEGER`, `progress_completed INTEGER`, constrained to be all absent or all present. |
| `companies`                | `slug TEXT PRIMARY KEY`, `title TEXT`, `category TEXT`, `type TEXT CHECK (type = 'company')`, `url TEXT`, nullable `cover TEXT`, `status TEXT`, `featured BOOLEAN`, `priority TEXT`, `tags TEXT[]`, `body TEXT`, `contacts_present BOOLEAN`.                                                                                                                                                                                                                                                  |
| `company_contacts`         | `company_slug TEXT REFERENCES companies(slug) ON DELETE CASCADE`, `id UUID`, `position INTEGER`, `name TEXT`, `role TEXT`, `email TEXT`, `url TEXT`, `notes TEXT`; primary key `(company_slug, id)`, unique `(company_slug, position)`.                                                                                                                                                                                                                                                       |
| `goals`                    | `id UUID PRIMARY KEY`, `title TEXT`, `start_date DATE`, `end_date DATE`, `color TEXT`, nullable `daily_hours NUMERIC`, `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`.                                                                                                                                                                                                                                                                                                                    |
| `goal_notes`               | `goal_id UUID REFERENCES goals(id) ON DELETE CASCADE`, `id UUID`, `position INTEGER`, `body TEXT`, `created_at TIMESTAMPTZ`; primary key `(goal_id, id)`, unique `(goal_id, position)`.                                                                                                                                                                                                                                                                                                       |
| `goal_steps`               | `goal_id UUID REFERENCES goals(id) ON DELETE CASCADE`, `id UUID`, `position INTEGER`, `title TEXT`, `done BOOLEAN`; primary key `(goal_id, id)`, unique `(goal_id, position)`.                                                                                                                                                                                                                                                                                                                |
| `goal_metadata`            | `goal_id UUID REFERENCES goals(id) ON DELETE CASCADE`, `key TEXT`, `value TEXT`; primary key `(goal_id, key)`.                                                                                                                                                                                                                                                                                                                                                                                |
| `migration_imports`        | `id UUID PRIMARY KEY`, `source_store TEXT`, `source_key TEXT`, nullable `source_revision TEXT`, `source_checksum TEXT`, `imported_at TIMESTAMPTZ`, `archive_path TEXT`, source schema markers. Record imported checksums uniquely per source store/key. Keep exact original bytes in restricted backup files outside the served tree and Git.                                                                                                                                                 |

- [x] **2.4.** Give all seven top-level content tables (`notes`, `documents`, `personal_journal_entries`, `journal_weeks`, `books`, `companies`, `goals`) a non-null opaque `revision TEXT` and `position BIGINT` to preserve collection order. Retain optional legacy `updated_at TIMESTAMPTZ` on non-goal entries without inventing source timestamps. Required fields are `NOT NULL`; explicitly map absent optional fields, empty arrays/maps, empty strings, and nulls at the API/export boundary. Add presence flags where row absence alone cannot preserve the source distinction (including optional top-level collections). Reject unsupported values during dry-run instead of silently coercing them.
- [x] **2.5.** Add foreign keys, nonnegative positions, and checks matching current validation for enums, hours, rating, progress, and goal date ranges. Keep nested/slash-containing IDs as text; do not replace them with generated UUIDs. Preserve array ordering and duplicates where currently permitted. Store date-only values as `DATE` and timestamps as `TIMESTAMPTZ`; retain source timestamp representation in import provenance if needed for lossless export.
- [x] **2.6.** Add indexes for actual query paths: notes `(topic, title, id)`, documents `(title, id)`, personal journals `(entry_date DESC, id)`, weeks `(start_date DESC, slug)` and `(year, week)`, books/companies `(status, category, priority, slug)`, and goal start/end dates. Child primary keys already support owner lookups. Add tag GIN or maintained full-text search indexes only for implemented filters/search; verify query plans with representative data and `EXPLAIN (ANALYZE, BUFFERS)`.
- [x] **2.7.** Parse Markdown at import/save time for derived board/shelf data. Add `book_summaries` (`book_slug` FK/PK, chapter counts, excerpt, source revision, parser version) and `company_summaries` (`company_slug` FK/PK, why text, step counts, source revision, parser version), plus owner-specific checklist/log projection tables with cascading FKs, ordered positions, source line indices, labels/text, checked state, and log dates where present. Markdown remains authoritative: update these projections in the same transaction as the owning row, preserve explicit book progress overrides, and provide an explicit rebuild command when parser behavior changes. Routine list/dashboard requests must not reparse all Markdown bodies. Render/sanitize only requested detail content; preserve checklist source-line semantics and reject stale toggles.
- [x] **2.8.** Implement entity-level optimistic concurrency: update/delete the addressed row only where its revision matches, rotate the revision on success, and return HTTP 409 on stale writes. Null revision means insert-if-absent with unique-key conflicts mapped to 409. Child edits (contacts, goal notes/steps/metadata) participate in the parent's revision and transaction. Independent entity edits must succeed concurrently; never overwrite unrelated rows. Increment `workspace_metadata.change_sequence` transactionally for consistent export provenance, not as an interactive global conflict token.
- [x] **2.9.** Use explicit column projections, bounded pagination with deterministic ordering, SQL filtering/aggregation, and batched child queries. List responses omit Markdown bodies and unrequested children. A detail read fetches one entity; a save returns the changed entity/revision or delete acknowledgement. No request may load/serialize the entire workspace just to read or change one entry. Full exports reconstruct the legacy shape from all tables under one consistent read transaction and run only on explicit export/compatibility requests.
- [x] **2.10.** Preserve existing per-entry and per-request validation limits. Replace the old 4,000,000-byte whole-workspace save limit with bounded entity requests and paginated reads; do not impose that aggregate cap on the relational database. Stream full exports/imports with separately documented limits so database growth does not break backup or agent-context export. Validate relational round trips against fixtures, including optional-field presence, array order, Unicode, date-only values, and exact Markdown bytes.
- [x] **2.11.** Add an explicit migration command with a database lock and migration history. Run it as a release step before rollout; application startup checks schema compatibility. Keep initial schema changes compatible with the preceding application image.
- [x] **2.12.** Add an explicit seed/import command. Export the existing Astro content collections to a versioned seed artifact using current parsing and migration rules, then transactionally populate these tables and derived projections only into an empty workspace. Seed artifacts belong outside the served static directory. Database failures must return errors, never trigger reseeding or restore deleted content.

## 3. Move browser APIs and authentication into Go

### Files and verification for step 3

Foundation already present: scoped `/api/entries/{kind}`, cookie auth, origin/content-type checks, throttling, exports, and entity concurrency exist in Go (step 2). These checkboxes remain open because browser integration and full endpoint/page behavior still need verification. Extend those implementations and record the actual delta beneath each item.

| Items   | Read / update locations                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 3.1     | Callers: `src/components/WorkspaceEditor.tsx`, `src/components/BookShelf.tsx`, `src/components/CompanyBoard.tsx`, `src/components/LogExplorer.tsx`, `src/components/ProgressDashboard.tsx`, `src/components/SiteSearch.astro`, `src/components/ContentPage.astro`, `src/components/EntryFields.tsx`; legacy contract: `src/pages/api/workspace.ts`, `src/lib/workspace.ts`, `src/lib/workspace-server.ts`; Go: `internal/server/routes.go`, `internal/database/store.go`, `internal/database/model.go`, `internal/database/projections.go`; current API reference: `docs/migration/local-development.md`. Page entry points are in step 4's map. |
| 3.2     | `src/pages/api/goals.ts`, `src/components/GoalTimeline.tsx`, `src/components/GoalSchedule.tsx`, `src/pages/timeline.astro`, `src/lib/timeline.ts`, `internal/server/routes.go`, `internal/database/store.go`, `internal/database/children.go`, `tests/timeline.test.ts`.                                                                                                                                                                                                                                                                                                                                                                         |
| 3.3     | `src/pages/api/agent-context.ts`, `src/lib/agent-context.ts`, `src/pages/agents.astro`, `internal/server/routes.go`, `internal/database/transfer.go`, `internal/database/store.go`.                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| 3.4–3.5 | `netlify/functions/auth.js`, `src/pages/login.astro`, `src/lib/auth.ts`, `src/layouts/Layout.astro`, `src/lib/server-auth.ts`, `scripts/dev-auth.mjs`, `internal/server/auth.go`, `internal/server/routes.go`, `internal/config/config.go`, `AUTH_README.md`.                                                                                                                                                                                                                                                                                                                                                                                    |
| 3.6     | `src/middleware.ts`, `src/pages/login.astro`, `internal/server/auth.go`, `internal/server/routes.go`, `internal/server/routes_test.go`; public paths: `public/`, `src/components/PwaHead.astro`.                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| 3.7     | `internal/server/auth.go`, `internal/server/routes.go`, `internal/server/server.go`, `internal/config/config.go`, `internal/database/database.go`, `.env.example`, `internal/server/routes_test.go`.                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| 3.8     | `src/pages/api/mcp.ts`, `src/pages/.well-known/oauth-protected-resource.ts`, `src/pages/.well-known/oauth-protected-resource/api/mcp.ts`, `src/pages/agents.astro`, `src/lib/mcp-auth.ts`, `src/lib/mcp-discovery.ts`, `src/lib/mcp-http.ts`, `src/lib/mcp-server.ts`, `package.json`; deferred suites: `tests/mcp.test.ts`, `tests/mcp-oauth.test.ts`, `tests/mcp-bridge.test.mjs`, `tests/mcp.integration.mjs`; docs: `MCP.md`, `MCP-OAUTH-PLAN.md`.                                                                                                                                                                                           |

Validation handoff: `go test ./...`, `go build -o /tmp/career-migration ./cmd/api`, `node tests/go-api.integration.mjs` after local DB setup from step 2; adapt/extend `internal/server/routes_test.go` and `tests/go-api.integration.mjs` for new contracts. For frontend changes: `npx astro check`, `npm run tc -- --noEmit`, `npm run build`, and `node --test tests/workspace.test.ts tests/timeline.test.ts tests/search.test.ts tests/journals.test.ts tests/companies.test.ts`. Browser/auth suites listed in step 7 require conversion before they validate Go. Record interim build dependencies on step 4; do not call API smoke alone browser parity.

- [x] **3.1.** Replace interactive full-workspace calls with scoped collection/detail endpoints for all six entry kinds (for example, `/api/entries/{kind}` plus a query parameter for nested IDs), and page-specific dashboard summaries. Support server-side filters, bounded pagination, targeted saves/deletes, and entity revisions. Preserve core-document delete protection, validation, timestamps, conflict draft recovery, and storage errors. Port domain operations from `src/lib/workspace.ts` and schemas from `workspace-server.ts`. If retaining `/api/workspace` temporarily, isolate it as an explicit compatibility adapter and remove every routine page/editor dependency on it before completion.
  State (2026-09-14): in-progress
  Changed files: `migration.md`.
  Implemented/evidence: Began step 3 in the dedicated `migration-step-3` worktree and topic branch; the step 2 scoped Go API is the starting contract.
  Validation: bounded repository inspection only; implementation validation pending.
  Differences: none.
  Pending user command: none yet; exact validation commands will be recorded after implementation.
  Next action/dependencies: replace routine browser `/api/workspace` callers with scoped entry and summary reads, beginning with the item 3.1 file map.
  State (2026-09-14): in-progress
  Changed files: `src/components/WorkspaceEditor.tsx`, `src/components/CompanyBoard.tsx`, `src/components/SiteSearch.astro`, `internal/database/search.go`, `internal/server/routes.go`.
  Implemented/evidence: Editors use bounded paginated collection reads, detail reads, targeted entity saves/deletes, entity revisions, and explicit `/api/export`; site search uses one bounded server-side SQL query; company mutations use the scoped company endpoint. Existing Go validation, timestamps, core protection, conflict, and storage mappings remain authoritative.
  Validation: Go tests/build, TypeScript compile, Astro check/build, 31 frontend unit tests, and expanded Go/PostgreSQL API integration passed agent-run on 2026-09-14.
  Differences: Page-specific dashboard data still enters through Astro build/runtime props; moving those reads to browser summary APIs belongs to item 4.2 and cannot be end-to-end verified until the static shells exist.
  Pending user command: none.
  Next action/dependencies: complete item 4.2's browser dashboard/summary integration, then close 3.1.
  State (2026-09-14): complete
  Changed files: `src/components/StaticDataViews.tsx`, `internal/database/summaries.go`, `internal/server/routes.go`; prior item 3.1 files retained.
  Implemented/evidence: Static browser pages now use scoped detail/collection calls and a body-free progress summary endpoint; no routine browser dependency on `/api/workspace` remains.
  Validation: final step 4 frontend, Go, PostgreSQL integration, and browser/PWA suites passed agent-run on 2026-09-14.
  Differences: none.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **3.2.** Adapt `GET/POST/DELETE /api/goals` to goal tables, goal-level revisions, visible-date-range reads, and targeted mutation responses; fetch notes/steps only when needed. Update timeline callers for the new contract while preserving UUIDs, date-only ranges, inclusive scheduling semantics, daily-hour estimates, notes, metadata, steps, and server-owned timestamps.
  State (2026-09-14): complete
  Changed files: `internal/server/context.go`, `internal/server/routes.go`, `src/components/GoalTimeline.tsx`.
  Implemented/evidence: Added range-filtered reads and targeted goal saves/deletes using per-goal revisions; timeline reloads its visible range and merges mutation results locally.
  Validation: `go test ./...`, Go build, TypeScript compile, Astro check/build, 31 frontend unit tests, and expanded Go/PostgreSQL API integration all passed agent-run on 2026-09-14.
  Differences: none.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **3.3.** Port `GET /api/agent-context` with JSON/Markdown output and optional work/personal journal filtering. Keep saved timeline goals authoritative and preserve complete exports independently of browser JavaScript. This browser-session endpoint does not require MCP or OAuth.
  State (2026-09-14): complete
  Changed files: `internal/server/context.go`, `internal/server/routes.go`.
  Implemented/evidence: Go now emits authenticated JSON or Markdown agent context from a repeatable-read database export, with optional work/personal filtering and saved goals as the current timeline.
  Validation: expanded Go/PostgreSQL integration covered JSON/Markdown, journal validation, and authoritative saved goals; all required checks passed agent-run on 2026-09-14.
  Differences: none.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **3.4.** Replace `/.netlify/functions/auth` with Go login, session verification, and logout routes under `/api/auth/`. Reuse the bcrypt password hash and 24-hour HS256 session lifetime initially; enforce the configured username, algorithm, signature, and expiration.
  State (2026-09-14): complete
  Changed files: `src/lib/auth.ts`, `src/pages/login.astro`; Go foundation remains in `internal/server/auth.go` and `internal/server/routes.go`.
  Implemented/evidence: Browser callers now use `/api/auth/login`, `/verify`, and `/logout`; no browser bearer token is accepted or stored.
  Validation: Go auth unit tests, Go build, TypeScript/Astro checks, and API integration passed agent-run on 2026-09-14.
  Differences: none.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **3.5.** Have Go set and clear the session cookie using `HttpOnly`, `Secure` on HTTPS, `SameSite=Lax`, and `Path=/`. Update login, `src/lib/auth.ts`, and logout callers to use cookie-based verification instead of JavaScript-managed tokens/localStorage. Clear obsolete token keys and require a fresh login at cutover.
  State (2026-09-14): complete
  Changed files: `src/lib/auth.ts`, `src/pages/login.astro`, `src/layouts/DocsLayout.astro`.
  Implemented/evidence: Login/logout rely on Go's HttpOnly `session` cookie; legacy local token keys are cleared and redirect targets are restricted to local paths.
  Validation: Go auth cookie assertions, TypeScript/Astro checks, build, and integration passed agent-run on 2026-09-14.
  Differences: display username remains a generic static value until item 4.2 hydrates session state.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **3.6.** Port the protection rules from `src/middleware.ts` into Go: redirect unauthenticated page requests to login, return JSON 401 for private APIs, and allow only the intended login/public assets and health probes without a session. Validate login redirect targets as local paths.
  State (2026-09-14): complete
  Changed files: `src/pages/login.astro`; API protection foundation remains in `internal/server/auth.go`.
  Implemented/evidence: Private Go APIs return JSON 401 and login redirects reject external/network-path targets.
  Validation: `internal/server/routes_test.go` verifies protected-page redirects, public login access, and private API 401s; Go tests passed agent-run on 2026-09-14.
  Differences: Static handlers are added in step 4, but the access policy now wraps the router and will govern those handlers.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **3.7.** Preserve same-origin checks on mutations and JSON content-type requirements. Compare against configured `PUBLIC_ORIGIN` behind ingress; trust forwarded headers only from the intended proxy path. Add login throttling and bounded request reads, timeouts, and database connection limits.
  State (2026-09-14): complete
  Changed files: no new delta; step 2 foundation in `internal/server/auth.go`, `internal/server/routes.go`, `internal/server/server.go`, `internal/config/config.go`, and `internal/database/database.go` is retained.
  Implemented/evidence: New goal mutations reuse the same origin/content-type/body-bound enforcement; no forwarded headers are trusted.
  Validation: Go unit and expanded integration tests passed agent-run on 2026-09-14.
  Differences: none.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **3.8.** Remove MCP/discovery routes from the new served build and hide their connection instructions in `/agents`. Return 404 for excluded paths rather than routing them to a generic HTML page. Separate deferred MCP tests/scripts from the migration's required checks; do not recreate their OAuth configuration or secrets.
  State (2026-09-14): complete
  Changed files: `src/pages/agents.astro`, `package.json`; Go route exclusion remains in `internal/server/routes.go`.
  Implemented/evidence: Removed MCP UI/instructions, kept Go without MCP/discovery routes, and moved MCP unit suites to `test:mcp:deferred` outside the required `npm test` set.
  Validation: Go API smoke passed its MCP/discovery 404 assertions agent-run on 2026-09-14; Astro check/build passed.
  Differences: Astro MCP source routes remain until item 4.9 removes obsolete server routes; they are not registered by Go.
  Pending user command: none.
  Next action/dependencies: final obsolete Astro source removal remains item 4.9.

## 4. Convert Astro to a static build served by Go

### Files and verification for step 4

| Items | Read / update locations                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 4.1   | `astro.config.mjs`, `package.json`, `package-lock.json`, `src/content.config.ts`, `tsconfig.json`; server reads in the page files below.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| 4.2   | Pages: `src/pages/index.astro`, `src/pages/books.astro`, `src/pages/companies.astro`, `src/pages/logs.astro`, `src/pages/progress.astro`, `src/pages/agents.astro`, `src/pages/timeline.astro`, `src/pages/journal.astro`, `src/pages/personal-journal.astro`, `src/pages/notes/index.astro`, `src/pages/documents/index.astro`; islands: all callers in 3.1–3.2 plus `src/components/MarkdownPreview.tsx`; shared rendering: `src/components/ContentPage.astro`, `src/layouts/Layout.astro`, `src/layouts/DocsLayout.astro`, `src/components/Sidebar.astro`; data/parsing: `src/lib/search.ts`, `src/lib/books.ts`, `src/lib/companies.ts`, `src/lib/logs.ts`, `src/lib/progress.ts`, `src/lib/checklist.ts`. |
| 4.3   | `src/pages/notes/[...id].astro`, `src/pages/documents/[...id].astro`, `src/pages/companies/[...slug].astro`, `src/components/WorkspaceEditor.tsx`, `src/components/CompanyBoard.tsx`, `src/components/ContentPage.astro`, `internal/server/routes.go`; create replacement static-shell source files under `src/pages/` and record their exact paths here.                                                                                                                                                                                                                                                                                                                                                      |
| 4.4   | `internal/server/routes.go`, `internal/database/store.go`, `internal/server/routes_test.go`, `src/pages/2026/career-study-plan.astro`, `src/pages/2026/algorithms/`, `src/pages/2026/system-design/`, `src/pages/2026/os-oss/`, `src/pages/2026/languages/index.astro`, `src/pages/2026/random/vintage-computers.astro`; alias inventory: `docs/migration/current-behavior.md`.                                                                                                                                                                                                                                                                                                                                |
| 4.5   | `src/pages/manage/[kind].astro`, `src/components/WorkspaceEditor.tsx`, `src/pages/companies/[...slug].astro`, `src/pages/login.astro`, `internal/server/routes.go`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| 4.6   | `src/lib/workspace-seed.ts`, `src/lib/workspace-server.ts`, `src/content.config.ts`, `src/content/`, `scripts/export-seed.mjs`, `astro.config.mjs`, `public/`; inspect generated `dist/` and `.astro/` for leaks; `.migration-private/` must stay excluded.                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| 4.7   | `internal/server/routes.go`, `internal/server/server.go`, `internal/config/config.go`, `internal/server/routes_test.go`; generated `dist/` route/layout inspection.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| 4.8   | `internal/server/routes.go`, `internal/server/auth.go`, `public/sw.js`, `public/manifest.webmanifest`, `public/offline.html`, `public/icons/`, `src/components/PwaHead.astro`, `tests/pwa.browser.mjs`, `PWA.md`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| 4.9   | `src/pages/api/workspace.ts`, `src/pages/api/goals.ts`, `src/pages/api/agent-context.ts`, `src/pages/api/mcp.ts`, `src/pages/.well-known/`, `src/middleware.ts`, `src/lib/workspace-server.ts`, `src/lib/server-auth.ts`, `src/lib/auth.ts`, `astro.config.mjs`, `scripts/dev.mjs`; retain tooling dependencies required by step 7's one-time export until safe to remove.                                                                                                                                                                                                                                                                                                                                     |

Validation handoff: `npx astro check`, `npm run tc -- --noEmit`, `npm run build`, `go test ./...`, `go build -o /tmp/career-migration ./cmd/api`. Prepare an isolated static build without DB/Netlify credentials (including no auto-loaded credential file); document the exact invocation. Inspect generated output and exercise Go with `STATIC_DIR` pointing to that output. After adapting the step 7 suites, have the user run `node tests/workspace.browser.mjs`, `node tests/journals.browser.mjs`, `node tests/pwa.browser.mjs` against the Go-served app, including new-after-build nested URLs, 404s, login redirects, and offline privacy. Record required server/browser/environment setup first.

- [x] **4.1.** Change `astro.config.mjs` from `output: "server"` with the Netlify adapter to entirely static output after removing runtime server dependencies. This repo uses Astro 6; follow its installed version rather than Astro 5 examples in the local skill. Astro prerenders static routes at build time, so live workspace reads must move out of frontmatter. See [Astro rendering documentation](https://docs.astro.build/en/guides/on-demand-rendering/).
  State (2026-09-14): in-progress
  Changed files: `migration.md`.
  Implemented/evidence: Created the dedicated `migration-step-4` worktree and topic branch from the current `migration` branch.
  Validation: bounded repository and worktree inspection only; implementation validation pending.
  Differences: none.
  Pending user command: none yet; exact validation commands will be recorded after implementation.
  Next action/dependencies: remove build-time workspace/session reads, then switch Astro 6 to static output without the Netlify adapter.
  State (2026-09-14): in-progress
  Changed files: `astro.config.mjs`, `package.json`, `src/pages/`, `src/components/StaticDataViews.tsx`, `src/components/WorkspaceEditor.tsx`, `src/components/CompanyBoard.tsx`, `internal/server/static.go`, `internal/server/routes.go`, `internal/server/auth.go`, `scripts/dev.mjs` in the `migration-step-4` worktree.
  Implemented/evidence: Astro is configured for static output without the Netlify adapter; build-time workspace/session reads and Astro server routes were removed; scoped browser loaders, reusable detail shells, finite manage shells, and explicit Go static/dynamic routing were added.
  Validation: `gofmt`, `git diff --check`, and compile-only `go test ./internal/server -run '^$'` passed agent-run on 2026-09-14 (the bounded command unexpectedly took 4 seconds); required full checks remain pending.
  Differences: the Netlify blobs dependency is retained for the step 7 one-time export tooling; only the obsolete Astro Netlify adapter is removed.
  Pending user command: from `/home/mduren/Code/career-strategy/migration-step-4`, run `npm install --package-lock-only --ignore-scripts --no-audit --no-fund`, then `npx astro check && npm run tc -- --noEmit && env -u DATABASE_URL -u NETLIFY_AUTH_TOKEN -u NETLIFY_SITE_ID npm run build`, then `go test ./... && go build -o /tmp/career-migration-step-4 ./cmd/api`; expected result is exit 0 for all commands and a static `dist/` build without credentials.
  Next action/dependencies: inspect results and generated output, fix failures, then validate Go-served static/browser behavior for items 4.2–4.9.
  State (2026-09-14): in-progress
  Changed files: `migration.md`.
  Implemented/evidence: User explicitly instructed the agent to run the previously handed-off commands; this is recorded as the step 4 validation-command waiver.
  Validation: full suite starting agent-run.
  Differences: normal commands-over-two-seconds handoff waived by explicit user direction for step 4.
  Pending user command: none.
  Next action/dependencies: run and inspect the full validation suite.
  State (2026-09-14): complete
  Changed files: `astro.config.mjs`, `package.json`, `package-lock.json`, static page and shell sources in the `migration-step-4` worktree.
  Implemented/evidence: Astro 6 now produces a fully static 17-page build without the Netlify adapter or runtime workspace/session reads.
  Validation: `npx --no-install astro check`, TypeScript compile, and credential-free `npm run build` passed agent-run on 2026-09-14.
  Differences: Netlify blobs remains temporarily for step 7 export tooling; no Netlify runtime adapter remains.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **4.2.** Replace server-provided workspace props with authenticated, scoped collection/detail/summary fetches and loading/error/session-expiry states. Fetch only the visible page data, reuse it across islands during the current navigation, and refresh the changed entity/affected summaries after mutations. Do not download the full workspace, parse all Markdown client-side, or issue a request per list item. Update home, books, companies, logs, progress, agents, and editable document/reference views. Preserve search, filters, checklists, contacts, timeline editing, sanitized Markdown rendering, and local draft recovery.
  State (2026-09-14): complete
  Changed files: `src/components/StaticDataViews.tsx`, `src/components/CompanyBoard.tsx`, `src/components/WorkspaceEditor.tsx`, dashboard pages, `internal/database/summaries.go`, `internal/server/routes.go`.
  Implemented/evidence: Added bounded scoped loaders, shared per-view results, session-expiry redirects, mutation refresh events, and a body-free progress summary; existing sanitized previews, filters, editors, contacts, checklists, timeline, and drafts remain active.
  Validation: Astro/TypeScript builds, 30 frontend unit tests, PostgreSQL integration, and PWA/browser checks passed agent-run.
  Differences: log search intentionally receives the bounded week-detail collection because journal bodies are its visible searchable data; progress/home receive body-free summaries.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **4.3.** Build reusable static shells for `/notes/<id>`, `/documents/<id>`, and `/companies/<slug>`. Go maps these URL families to the appropriate shell; client code reads the identifier from the URL. Do not prerender only today's database IDs: entries created after deployment must open without rebuilding.
  State (2026-09-14): complete
  Changed files: `src/pages/static-shells/`, `src/components/StaticDataViews.tsx`, `src/components/WorkspaceEditor.tsx`, `internal/server/static.go`.
  Implemented/evidence: Three reusable shells resolve URL identifiers in the browser; integration created an entry after build and opened its nested URL through Go.
  Validation: credential-free static build and post-build dynamic-route integration passed agent-run.
  Differences: shell output lives under `static-shells/` because Astro excludes underscore-prefixed page directories.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **4.4.** Have Go validate dynamic-route IDs against saved data where needed to preserve HTTP 404 behavior. Preserve nested IDs, URL encoding, direct navigation, refresh, and known reference aliases under `/2026/`. Keep fixed routes distinct from dynamic detail routes.
  State (2026-09-14): complete
  Changed files: `internal/server/static.go`, removed runtime `src/pages/2026/` sources.
  Implemented/evidence: Go validates decoded nested IDs through PostgreSQL, serves fixed reference aliases separately, and returns real 404s after deletion.
  Validation: Go/PostgreSQL integration and the 71-route mobile browser audit passed agent-run.
  Differences: known reference routes reuse the document shell rather than duplicate Astro pages.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **4.5.** Generate the finite `/manage/books` and `/manage/companies` shells, move query-string handling into browser code, and preserve the company-editor redirect in Go. Remove request/cookie reads and redirects from Astro build-time execution.
  State (2026-09-14): complete
  Changed files: `src/pages/manage/books.astro`, `src/pages/manage/companies.astro`, `src/pages/documents/index.astro`, `src/pages/login.astro`, `internal/server/static.go`.
  Implemented/evidence: Finite shells build statically; editors read query strings in the browser and Go owns the company redirect.
  Validation: Astro static route inventory and Go integration passed agent-run.
  Differences: none.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **4.6.** Keep private content and seed data out of publicly accessible JavaScript, generated JSON, source maps, and public assets. Public frontend bundles should contain application code, not saved workspace snapshots. Protect static HTML through Go before serving it.
  State (2026-09-14): complete
  Changed files: static page/component changes above; `internal/server/auth.go`, `internal/server/static.go`.
  Implemented/evidence: Generated assets contain no verbatim private source paragraphs, credential markers, or source maps; Go protects all application HTML.
  Validation: inspected all `dist` HTML/JS/JSON/map candidates against 82 source Markdown files: zero paragraph hits and zero source maps.
  Differences: seed sources remain repository-only for step 7 export tooling and are not emitted into `dist`.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **4.7.** Serve the final `dist/` directory from Go with correct MIME types, directory-index handling, and an explicit route map. Return real 404s for unknown pages/assets/APIs and reject traversal paths; do not use a universal SPA fallback.
  State (2026-09-14): complete
  Changed files: `internal/server/static.go`, `internal/server/routes.go`, `tests/go-api.integration.mjs`.
  Implemented/evidence: Added explicit fixed/dynamic maps, MIME-aware file serving, containment checks, and non-HTML unknown API/page/asset 404s.
  Validation: Go unit/build and PostgreSQL-backed HTTP integration passed agent-run.
  Differences: none.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **4.8.** Preserve `private, no-store` on protected HTML and API responses; cache only appropriate public assets, with immutable caching for hashed build assets. Preserve the PWA manifest/icons/offline page and network-only treatment of private navigation/API data in `public/sw.js`.
  State (2026-09-14): complete
  Changed files: `internal/server/static.go`, `internal/server/auth.go`, `tests/pwa.browser.mjs`.
  Implemented/evidence: Protected HTML/API responses remain private/no-store, hashed Astro assets are immutable, and only the existing public PWA resources are cached offline.
  Validation: integration header assertions, 71-route/four-width layout audit, and PWA focus/cache/offline/reconnection interactions passed agent-run.
  Differences: none.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **4.9.** Remove Astro API routes, middleware, and obsolete server-only imports once Go replacements work. Verify the build succeeds without PostgreSQL access, Netlify credentials, or a Netlify emulator.
  State (2026-09-14): complete
  Changed files: removed `src/pages/api/`, `src/pages/.well-known/`, `src/middleware.ts`, `src/lib/workspace-server.ts`, `src/lib/server-auth.ts`; updated `astro.config.mjs`, `package.json`, `package-lock.json`, `scripts/dev.mjs`, `tests/workspace.test.ts`.
  Implemented/evidence: Removed Astro runtime/API/auth middleware and Netlify adapter; retained Netlify blobs only for step 7 export tooling.
  Validation: Astro check, TypeScript, and static build passed with DB and Netlify credential variables unset; Go and browser suites passed agent-run.
  Differences: deferred MCP tooling/libs remain excluded from the served build, as required by step 3.8.
  Pending user command: none.
  Next action/dependencies: none.

## 5. Provision a dedicated homelab database and backups

### Files and verification for step 5

External infrastructure root: `/home/mduren/Code/home-infra/infra`. No dedicated career-strategy playbooks exist yet.

| Items   | Read / update locations                                                                                                                                                                                                                                                                                             |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 5.1–5.2 | Reference: `/home/mduren/Code/home-infra/infra/ansible/postgres.yml`, `/home/mduren/Code/home-infra/infra/ansible/inventory.ini`. Create: `/home/mduren/Code/home-infra/infra/ansible/career-strategy-db.yml`. Live `pg_hba.conf` location must be obtained from the DB host (`SHOW hba_file`), then recorded here. |
| 5.3     | `/home/mduren/Code/home-infra/infra/ansible/Makefile`, `/home/mduren/Code/home-infra/infra/Makefile`; create app operations documentation at `deploy/homelab/README.md` with the exact collection install, secret injection, and playbook commands.                                                                 |
| 5.4     | `/home/mduren/Code/home-infra/infra/ansible/postgres.yml`, new career DB playbook; reference `/home/mduren/Code/rubber-duck/deploy/homelab/deployment.yaml`; app `.env.example`, `internal/config/config.go`, new `deploy/homelab/README.md`. Record live PostgreSQL config/certificate locations after inspection. |
| 5.5     | `/home/mduren/Code/home-infra/infra/ansible/duck-backup.yml`, `/home/mduren/Code/home-infra/infra/ansible/Makefile`; create `/home/mduren/Code/home-infra/infra/ansible/career-strategy-backup.yml`; document timer/unit/archive/retention paths in `deploy/homelab/README.md` after choosing them.                 |
| 5.6     | New career backup playbook and `deploy/homelab/README.md`; monitoring references: `/home/mduren/Code/home-infra/infra/ansible/prometheus.yml`, `/home/mduren/Code/home-infra/infra/ansible/grafana.yml`; app reads: `cmd/api/main.go`, `internal/database/transfer.go`, `tests/go-api.integration.mjs`.             |

Validation handoff: prepare exact `ansible-playbook` syntax/check, apply, and second-apply commands with cwd `/home/mduren/Code/home-infra/infra/ansible`, inventory `inventory.ini`, and the dedicated playbook only. The user runs these, live pod connectivity/TLS checks, backup, and disposable restore. Capture exit status, idempotence recap, counts/read checks, backup verification, and monitoring evidence without secrets. Commands depend on newly created playbooks and verified hosts; record concrete commands before execution. App code changes additionally use step 2 checks.

Reference: `/home/mduren/Code/home-infra/infra/ansible/postgres.yml` provisions PostgreSQL 17 on the `db` inventory group. Rubber Duck uses `pgdb` at `192.168.20.103:5432`; the playbook notes that pod traffic arrives as node IPs through SNAT.

- [x] **5.1.** Add `/home/mduren/Code/home-infra/infra/ansible/career-strategy-db.yml` targeting the existing `db` host. Create a dedicated `career_strategy` login and owned database idempotently; leave `duckserver` and its objects unchanged. Use `community.postgresql`, the existing privilege-escalation pattern, and a required secret variable protected with `no_log`.
  State (2026-09-14): in-progress
  Changed files: `migration.md` in the `migration-step-5` worktree.
  Implemented/evidence: Created the dedicated `migration-step-5` worktree and topic branch from the current `migration` branch at the user's direction.
  Validation: bounded repository inspection only; implementation validation pending.
  Differences: Step 5 began before the completed step 4 branch was merged into `migration`; step 5 infrastructure work is independent, and the branches must be reconciled before merge.
  Pending user command: none yet; exact dedicated playbook validation commands will be recorded after implementation.
  Next action/dependencies: create the dedicated database playbook without changing the existing Duck database or PostgreSQL installation.
  State (2026-09-14): awaiting-user-command
  Changed files: `/home/mduren/Code/home-infra/infra/ansible/career-strategy-db.yml`, `/home/mduren/Code/home-infra/infra/ansible/Makefile`, `deploy/homelab/README.md`.
  Implemented/evidence: Added a dedicated `db`-host play that requires a protected password, creates only the `career_strategy` login/database, and uses the existing postgres become-user pattern; no PostgreSQL package/install or Duck task is present.
  Validation: `git diff --check` passed agent-run 2026-09-14; required Ansible syntax/check/apply/idempotence results pending user execution.
  Differences: none.
  Pending user command: run the collection install and DB syntax/check/apply/two-pass sequence in `deploy/homelab/README.md`; return exit statuses and play recaps without secrets.
  Next action/dependencies: record successful live results and mark 5.1 complete.
  State (2026-09-14): complete
  Changed files: no additional files.
  Implemented/evidence: The user ran `./deploy/homelab/run-step5.sh database`; because the script uses `set -e`, reaching the final recap confirms syntax check, check mode, initial apply, and second apply all exited successfully. The dedicated play did not contain tasks targeting Duck objects.
  Validation: final second-apply recap user-reported 2026-09-14: `pgdb ok=9 changed=0 unreachable=0 failed=0 skipped=0 rescued=0 ignored=0`.
  Differences: none.
  Pending user command: none for item 5.1.
  Next action/dependencies: preserve the non-secret HBA/TLS output for items 5.2 and 5.4, then run the backup phase.
- [ ] **5.2.** Add a database/user-specific SCRAM `pg_hba.conf` entry for the verified cluster source subnet, following the existing `192.168.20.0/24` pattern. Reload PostgreSQL for HBA changes and verify connectivity from an actual app pod. Do not reinstall PostgreSQL or rerun unrelated cluster provisioning.
  State (2026-09-14): awaiting-user-command
  Changed files: `/home/mduren/Code/home-infra/infra/ansible/career-strategy-db.yml`, `deploy/homelab/README.md`.
  Implemented/evidence: The play discovers `SHOW hba_file`, manages one database/user-specific `scram-sha-256` line for `192.168.20.0/24`, and reloads rather than restarts PostgreSQL.
  Validation: live HBA path/subnet and app-pod connectivity are pending user execution.
  Differences: Actual app-pod verification depends on the workload created in step 6; this item remains open until that evidence exists.
  Pending user command: apply the DB playbook and report its HBA path; after step 6, run the documented in-pod `psql` query and report database/user/client address.
  Next action/dependencies: items 5.1 and 6.2–6.3.
  State (2026-09-14): awaiting-user-command
  Changed files: `/home/mduren/Code/home-infra/infra/ansible/career-strategy-db.yml`.
  Implemented/evidence: User-reported live DB runs confirmed active HBA file `/etc/postgresql/17/main/pg_hba.conf` and configured source subnet `192.168.20.0/24`; replaced deprecated `db` query arguments with `login_db` after the reported collection warning.
  Validation: check/apply/idempotence sequence passed user-run 2026-09-14; final recap `ok=9 changed=0 unreachable=0 failed=0`. Actual app-pod connectivity remains pending step 6.
  Differences: none.
  Pending user command: run `./deploy/homelab/run-step5.sh pod-check` after an app pod exists.
  Next action/dependencies: step 6 app workload.
- [x] **5.3.** Add a Makefile target and document secure secret injection and required Ansible collections. Run syntax/check review, apply the dedicated playbook, then rerun it to verify idempotence.
  State (2026-09-14): awaiting-user-command
  Changed files: `/home/mduren/Code/home-infra/infra/ansible/Makefile`, `deploy/homelab/README.md`, `deploy/homelab/run-step5.sh`.
  Implemented/evidence: Added dedicated Make targets, a hidden interactive password prompt, collection prerequisites, exact syntax/check/apply/idempotence commands, and a single shell entry point with phased subcommands. General migration instructions now require checked-in shell scripts for all user-run commands.
  Validation: `git diff --check` passed agent-run 2026-09-14; all Ansible commands are pending under the two-second handoff rule.
  Differences: none.
  Pending user command: execute the documented command sequence and return the second-apply `changed`/`failed` recap.
  Next action/dependencies: successful 5.1 and 5.5 playbook applies.
  State (2026-09-14): complete
  Changed files: no additional files.
  Implemented/evidence: Both dedicated Make targets and the secure interactive wrapper were exercised. The DB workflow reached an idempotent second apply; backup resources are installed and live verification passed.
  Validation: DB second apply user-reported `changed=0 failed=0`; backup verification user-reported rc=0 for both worker and Prometheus on 2026-09-14.
  Differences: The initial backup check-mode and oneshot-status verification defects were fixed and their failures remain recorded below item 5.5.
  Pending user command: none.
  Next action/dependencies: none.
- [x] **5.4.** Verify PostgreSQL TLS availability and explicitly document the chosen connection mode. Rubber Duck currently uses `sslmode=disable` on the LAN; if enabling certificate-verified TLS, provision the server certificate/trust configuration before requiring it in this app.
  State (2026-09-14): awaiting-user-command
  Changed files: `/home/mduren/Code/home-infra/infra/ansible/career-strategy-db.yml`, `deploy/homelab/README.md`.
  Implemented/evidence: The play reports `ssl`, CA, certificate, and key settings; documentation selects `sslmode=disable` on the LAN unless live evidence supports trusted `verify-full` configuration.
  Validation: live PostgreSQL setting output pending user execution.
  Differences: none.
  Pending user command: return the playbook's non-secret TLS settings.
  Next action/dependencies: record live certificate paths/settings and confirm or revise the documented choice.
  State (2026-09-14): complete
  Changed files: no additional files.
  Implemented/evidence: Live PostgreSQL reports TLS enabled but no CA file and only Debian snake-oil certificate/key paths. Therefore the documented initial `sslmode=disable` LAN choice is confirmed; certificate verification is not claimed.
  Validation: user-reported playbook output 2026-09-14: `ssl=on`, `ssl_ca_file` empty, `ssl_cert_file=/etc/ssl/certs/ssl-cert-snakeoil.pem`, `ssl_key_file=/etc/ssl/private/ssl-cert-snakeoil.key`; final apply was idempotent with `changed=0 failed=0`.
  Differences: TLS transport is technically available, but certificate-verified TLS is not configured; provisioning a trusted certificate/CA remains a prerequisite before switching to `verify-full`.
  Pending user command: none.
  Next action/dependencies: retain `sslmode=disable` for step 6 deployment configuration.
- [x] **5.5.** Add `career-strategy-backup.yml` and a Makefile target, adapting `duck-backup.yml`'s worker-host pull, mount check, `pg_dump -Fc`, archive verification, retention, and systemd timer. Give this database separate filenames and pruning rules; preserve Duck backups. Confirm backup storage capacity and set a documented recovery target, initially daily backups/at most 24 hours of data loss outside cutover.
  State (2026-09-14): awaiting-user-command
  Changed files: `/home/mduren/Code/home-infra/infra/ansible/career-strategy-backup.yml`, `/home/mduren/Code/home-infra/infra/ansible/Makefile`, `/home/mduren/Code/home-infra/infra/ansible/templates/prometheus.yml.j2`, `deploy/homelab/README.md`.
  Implemented/evidence: Added a separate worker-pull backup, mount guard, custom-format archive/TOC verification, atomic rename, isolated directory/name/pruning, 90-day retention, daily persistent timer, capacity check, and 24-hour recovery-point objective.
  Validation: static diff check passed; syntax/check/apply/idempotence, live archive, timer, and capacity evidence pending user execution.
  Differences: none.
  Pending user command: run the documented backup sequence and return recaps plus non-secret `df`, service, timer, and archive output.
  Next action/dependencies: successful live backup and capacity confirmation.
  State (2026-09-14): awaiting-user-command
  Changed files: `/home/mduren/Code/home-infra/infra/ansible/career-strategy-backup.yml`.
  Implemented/evidence: Fixed dry-run behavior by skipping timer and node-exporter activation when `ansible_check_mode` is true; those services cannot be addressed during check mode because their package/unit files are intentionally not created.
  Validation: first user-run backup check reached `pve-worker ok=10 changed=5 failed=1`; failure was `career-strategy-db-backup.timer` not found during check mode. This is recorded as a validation failure, not a pass. Shell/static validation of the fix is agent-run; rerun pending.
  Differences: The initial planned check run exposed an Ansible check-mode dependency and required the explicit skip; real apply behavior is unchanged.
  Pending user command: rerun `./deploy/homelab/run-step5.sh backup` and return its output.
  Next action/dependencies: successful check/apply/idempotence, backup, capacity, timer, and Prometheus results.
  State (2026-09-14): awaiting-user-command
  Changed files: `deploy/homelab/run-step5.sh`, `deploy/homelab/README.md`.
  Implemented/evidence: User-run backup completed successfully and created a verified archive, but the combined verification command returned rc=3 because `systemctl status` treats an inactive, successfully completed oneshot as nonzero. Replaced that check with explicit systemd `Result=success` and `ExecMainStatus=0` assertions and added a verification-only subcommand.
  Validation: user-reported 2026-09-14: backup service exited `status=0/SUCCESS`, logged `OK`, created `/mnt/duckbackup/pgdump/career-strategy/career_strategy-2026-09-14_1242.dump`; storage was 424G total, 402G available, 1% used. The wrapper's final command returned rc=3 solely at the status display, so timer/archive continuation and Prometheus config checks did not execute.
  Differences: Verification now tests systemd's result fields instead of relying on `systemctl status` exit semantics for a oneshot service.
  Pending user command: run `./deploy/homelab/run-step5.sh verify-backup` and return its output.
  Next action/dependencies: confirm timer, archive listing, and Prometheus configuration.
  State (2026-09-14): complete
  Changed files: no additional files.
  Implemented/evidence: Live verification confirmed the dedicated backup service completes successfully, the timer is scheduled, two isolated Career Strategy archives exist, capacity is ample, and Prometheus loads the backup rules.
  Validation: user-run `./deploy/homelab/run-step5.sh verify-backup` exited successfully 2026-09-14. Worker reported `Result=success`, `ExecMainStatus=0`, next timer at 2026-09-15 03:55:12 CDT, archives at 12:42 and 12:45, and 402G available of 424G. `promtool` reported valid configuration, one rule file, and two valid rules.
  Differences: none.
  Pending user command: run the same backup service immediately before cutover under item 5.6.
  Next action/dependencies: item 5.6 pre-cutover backup.
- [ ] **5.6.** Restore a backup into a disposable database and verify counts, representative entries, and application reads. Document credentials/role recreation as well as data restore. Make backup failure visible in the homelab's monitoring and take a fresh backup immediately before cutover.
  State (2026-09-14): awaiting-user-command
  Changed files: `/home/mduren/Code/home-infra/infra/ansible/career-strategy-backup.yml`, `/home/mduren/Code/home-infra/infra/ansible/templates/prometheus.yml.j2`, `deploy/homelab/README.md`.
  Implemented/evidence: Documented role recreation and disposable restore; added node-exporter backup metrics, Prometheus scrape config, and failed/stale rules; documented pre-cutover fresh backup.
  Validation: restore counts/representative rows/application reads and Prometheus visibility pending user execution after a live backup contains migrated data.
  Differences: Alert rules are visible in Prometheus, but notifications remain dependent on the homelab's broader Alertmanager workstream.
  Pending user command: run the documented archive/Prometheus checks and disposable restore drill, then report non-secret results; run a fresh backup again immediately before cutover.
  Next action/dependencies: live database contents, successful 5.5 backup, and cutover timing in step 8.

## 6. Build and deploy to k3s

### Files and verification for step 6

Reference deployment root: `/home/mduren/Code/rubber-duck/deploy/homelab/`. Step 5 already supplied app `deploy/homelab/README.md` and `run-step5.sh`; extend those operations docs. Step 6 implementation is in `/home/mduren/Code/career-strategy/migration-step-6`. Additional actual files beyond the proposed map: `deploy/homelab/configmap.yaml`, `scripts/check-container.sh`.

| Items   | Read / update locations                                                                                                                                                                                                                                                                                                                              |
| ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 6.1     | Create `Dockerfile`, `.dockerignore`; inputs: `package.json`, `package-lock.json`, `astro.config.mjs`, `go.mod`, `go.sum`, `cmd/api/main.go`, `internal/`, generated `dist/`; exclusions: `.env`, `.migration-private/`, `.netlify/`, `.astro/`, `src/content/`, `.gitignore`.                                                                       |
| 6.2     | Create `deploy/homelab/namespace.yaml`, `deploy/homelab/deployment.yaml`, `deploy/homelab/service.yaml`, `deploy/homelab/ingress.yaml`; reference same filenames under `/home/mduren/Code/rubber-duck/deploy/homelab/`.                                                                                                                              |
| 6.3     | New deployment/service manifests and `scripts/deploy-homelab.sh`, `.env.example`, `internal/config/config.go`; reference `/home/mduren/Code/rubber-duck/deploy/homelab/networkpolicy.yaml`; create app `deploy/homelab/networkpolicy.yaml` only if implementing one. Secret values stay outside tracked manifests.                                   |
| 6.4     | `cmd/api/main.go`, `internal/server/routes.go`, `internal/server/server.go`, `internal/database/database.go`, `internal/database/migrate.go`, `internal/server/routes_test.go`; new `deploy/homelab/deployment.yaml`. Health/readiness/shutdown foundation already exists from step 2; verify it and add deployed probes/outage acceptance.          |
| 6.5–6.6 | Reference `/home/mduren/Code/rubber-duck/scripts/deploy-homelab.sh`; create `scripts/deploy-homelab.sh`, `deploy/homelab/migration-job.yaml`, `deploy/homelab/README.md`; update `Makefile` and new deployment manifests. Record verified kubecontext, kubeconfig path, eligible nodes/architectures, current/previous image tags in the item state. |
| 6.7     | Reference `/home/mduren/Code/rubber-duck/deploy/homelab/cloudflared.yaml`; create `deploy/homelab/cloudflared.yaml` if an app tunnel is needed; new ingress/deployment/script files, `internal/server/auth.go`, `internal/config/config.go`, `docs/migration/current-behavior.md`.                                                                   |
| 6.8     | Create `deploy/homelab/README.md`; refer to step 5 playbooks, `scripts/deploy-homelab.sh`, `Makefile`, `docs/migration/current-behavior.md`.                                                                                                                                                                                                         |

Validation handoff: build/test with step 4 commands first; prepare exact image build, manifest validation, node import, migration Job, rollout, probe/outage, and ingress smoke commands for the user. Record cwd, verified kubecontext/namespace, immutable image tag, secret-loading prerequisites, and expected success for each. Use the newly documented release commands rather than inventing flags before the script exists. Record liveness surviving DB loss, readiness recovering, graceful shutdown, and successful HTTPS auth/mutations through ingress.

Reference: `/home/mduren/Code/rubber-duck/deploy/homelab/` and `scripts/deploy-homelab.sh` use Traefik, a ClusterIP Service, and local image imports into each node's containerd. They also include a Cloudflare tunnel deployment. The grading RBAC and isolated grading-pod network policy are specific to Rubber Duck.

- [x] **6.1.** Add a multi-stage Dockerfile: install locked frontend dependencies/build Astro, compile Go, then copy the binary and `dist/` into a non-root runtime image with CA certificates. Exclude credentials, private exports, development storage, and repository seed files from the runtime image and served tree.

  State (2026-09-14): complete.
  Changed files: Dockerfile, .dockerignore, scripts/deploy-homelab.sh, scripts/check-container.sh in migration-step-6.
  Implemented/evidence: production image `career-strategy:684ce848098bb563cdcd54575582b691c7b507bd` built successfully with runtime inventory checks; source/build context excludes private state and seeds.
  Validation: user-reported `./scripts/deploy-homelab.sh build` and `./scripts/check-container.sh` both reached PASS in the step 6 worktree. Runtime is non-root, includes static output, has no Node/source tree, and passed unavailable-DB startup and graceful shutdown checks.
  Differences: none. Earlier pending records below are historical and superseded by this completion.
  Pending user command: none for 6.1.
  Next action/dependencies: 6.2–6.6 cluster deployment; 6.7 external HTTPS acceptance.

  State (2026-09-14): awaiting-user-command.
  Changed files: Dockerfile, .dockerignore, scripts/check-container.sh (in dedicated step 6 worktree).
  Implemented/evidence: Added locked frontend/static and Go stages, CA-equipped non-root runtime, allowlisted build context excluding seeds/private state.
  Validation: agent-run bash syntax, YAML parse and git diff --check passed 2026-09-14. Required build/runtime/deployment checks have not run.
  Differences: Build and runtime inventory still require user results.
  Validation update (2026-09-14, user-reported): PostgreSQL race check passed, database package 1.847s. Source and PostgreSQL checks are accepted. Next user commands are `./scripts/deploy-homelab.sh build` and `./scripts/check-container.sh` in the clean step 6 worktree; historical pending check commands below are superseded. Container acceptance remains open.
  Validation update (2026-09-14, user-reported): source check script reached final PASS; Astro built 17 pages, Go race-enabled source tests and HTTP smoke passed. Container build/inventory remain pending; this does not complete 6.1.
  Pending user command: ./scripts/deploy-homelab.sh check-postgres; then build and ./scripts/check-container.sh after user commit. Cwd: /home/mduren/Code/career-strategy/migration-step-6. Prerequisites/results: deploy/homelab/README.md.
  Next action/dependencies: inspect user check results first; remaining acceptance as above.

- [x] **6.2.** Add `deploy/homelab/namespace.yaml`, `deployment.yaml`, `service.yaml`, and `ingress.yaml` for a separate `career-strategy` namespace, Go port 8080, and proposed `career.homelab` ingress. Start with one replica and explicit CPU/memory requests and limits.

  State (2026-09-14): complete.
  Changed files: step 6 deployment manifests and scripts/deploy-homelab.sh, committed at 684ce848098bb563cdcd54575582b691c7b507bd.
  Implemented/evidence: Namespace, one-replica Deployment, Service and Traefik ingress created. Agent verified one Ready pod and LAN Host-routed /readyz=200, /=303 to login. Explicit resources remain as validated manifests.
  Validation: user-reported deploy PASS; bounded agent-run live kubectl/HTTP checks passed at 21:17 UTC. Context default, namespace career-strategy, API https://192.168.20.3:6443.
  Differences: none; earlier pending records below are historical and superseded.
  Pending user command: none for 6.2.
  Next action/dependencies: remaining 6.4 deployed outage/recovery, 6.7 HTTPS and 6.8 operational acceptance.

  State (2026-09-14): awaiting-user-command.
  Changed files: deploy/homelab/namespace.yaml, deployment.yaml, service.yaml, ingress.yaml (in dedicated step 6 worktree).
  Implemented/evidence: Added separate namespace, port 8080, one replica, resource limits and Traefik LAN ingress.
  Validation: agent-run bash syntax, YAML parse and git diff --check passed 2026-09-14. Required build/runtime/deployment checks have not run.
  Differences: Server-side manifest validation and deployment pending.
  Pending user command: ./scripts/deploy-homelab.sh validate after secrets; then deploy. Cwd: /home/mduren/Code/career-strategy/migration-step-6. Prerequisites/results: deploy/homelab/README.md.
  Next action/dependencies: inspect user check results first; remaining acceptance as above.

- [x] **6.3.** Inject database/browser-auth secrets through Kubernetes Secrets and nonsecret settings separately. Disable service-account token automount; this app does not need Rubber Duck's pod/configmap permissions. If adding NetworkPolicy, allow the actual ingress path, DNS, and PostgreSQL traffic and test them.

  State (2026-09-14): complete.
  Changed files: step 6 deployment manifests and scripts/deploy-homelab.sh, committed at 684ce848098bb563cdcd54575582b691c7b507bd.
  Implemented/evidence: Deployment successfully consumed career-secrets and career-config. Agent verified automountServiceAccountToken=false and ran /app/career check-db in the app pod: PostgreSQL ready, schema version 1. No secret values inspected or logged.
  Validation: user-reported deploy PASS; bounded agent-run live kubectl/HTTP checks passed at 21:17 UTC. Context default, namespace career-strategy, API https://192.168.20.3:6443.
  Differences: none; earlier pending records below are historical and superseded.
  Pending user command: none for 6.3.
  Next action/dependencies: remaining 6.4 deployed outage/recovery, 6.7 HTTPS and 6.8 operational acceptance.

  State (2026-09-14): awaiting-user-command.
  Changed files: deploy/homelab/configmap.yaml, deployment.yaml, migration-job.yaml, scripts/deploy-homelab.sh (in dedicated step 6 worktree).
  Implemented/evidence: Separated ConfigMap and Secret; disabled token automount; hidden prompts and restricted temporary secret files, server-side apply without logged values.
  Validation: agent-run bash syntax, YAML parse and git diff --check passed 2026-09-14. Required build/runtime/deployment checks have not run.
  Differences: No NetworkPolicy added (optional in plan). Secret loading and live DB auth pending.
  Pending user command: ./scripts/deploy-homelab.sh secrets after dependency install and step 5 DB password is available. Cwd: /home/mduren/Code/career-strategy/migration-step-6. Prerequisites/results: deploy/homelab/README.md.
  Next action/dependencies: inspect user check results first; remaining acceptance as above.

- [ ] **6.4.** Adapt the scaffold's `/health` handler into `/healthz` for process liveness and `/readyz` for database/schema readiness; remove the database health check's fatal exit on connection failure. Retain and verify the existing graceful SIGTERM shutdown in `cmd/api/main.go`, adding database cleanup. Use startup/readiness/liveness probes that allow normal startup and do not restart the process solely because PostgreSQL is temporarily unavailable.

  State (2026-09-14): awaiting-user-command.
  Changed files: cmd/api/main.go, deploy/homelab/deployment.yaml, scripts/check-container.sh (in dedicated step 6 worktree).
  Implemented/evidence: Reused health/readiness and deferred DB Close/SIGTERM shutdown. Removed startup schema exit so DB outages do not terminate a starting process; added probes and container regression check.
  Validation: agent-run bash syntax, YAML parse and git diff --check passed 2026-09-14. Required build/runtime/deployment checks have not run.
  Differences: Existing /health alias retained. Deployed outage/recovery requires step 7 staging.
  Validation update (2026-09-14, user-reported): production image 684ce848098bb563cdcd54575582b691c7b507bd passed container check: liveness survives unavailable DB, readiness=503, SIGTERM exit=0. Remaining: deployed probe behavior and DB recovery under 7.10; prior container-check handoff below is superseded.
  Validation update (2026-09-14, user-reported): PostgreSQL race check passed. Next is the production-container unavailable-DB startup and SIGTERM check after image build; deployed recovery still depends on staging.
  Validation update (2026-09-14, user-reported): Go race-enabled source tests and HTTP smoke including readiness passed. Container DB-unavailable startup/SIGTERM and deployed outage/recovery are still pending; this does not complete 6.4.
  Pending user command: ./scripts/check-container.sh after image build; stage DB-loss/recovery acceptance in 7.10. Cwd: /home/mduren/Code/career-strategy/migration-step-6. Prerequisites/results: deploy/homelab/README.md.
  Next action/dependencies: inspect user check results first; remaining acceptance as above.

- [x] **6.5.** Add `scripts/deploy-homelab.sh` and Makefile targets following Rubber Duck's registry-free import pattern. Use immutable commit-based image tags with `imagePullPolicy: Never`; import onto all eligible nodes before applying the Deployment or migration Job. Verify current node addresses/architectures instead of assuming the reference script's list remains correct.

  State (2026-09-14): complete.
  Changed files: step 6 deployment manifests and scripts/deploy-homelab.sh, committed at 684ce848098bb563cdcd54575582b691c7b507bd.
  Implemented/evidence: Release script completed immutable image imports before the migration Job and Deployment. Live deployed image is career-strategy:684ce848098bb563cdcd54575582b691c7b507bd; eligible amd64 node discovery/import workflow passed. Current pod runs on k8s-w-2; migration ran on k8s-w-1.
  Validation: user-reported deploy PASS; bounded agent-run live kubectl/HTTP checks passed at 21:17 UTC. Context default, namespace career-strategy, API https://192.168.20.3:6443.
  Differences: none; earlier pending records below are historical and superseded.
  Pending user command: none for 6.5.
  Next action/dependencies: remaining 6.4 deployed outage/recovery, 6.7 HTTPS and 6.8 operational acceptance.

  State (2026-09-14): awaiting-user-command.
  Changed files: scripts/deploy-homelab.sh, Makefile, deploy/homelab/deployment.yaml, migration-job.yaml (in dedicated step 6 worktree).
  Implemented/evidence: Added clean-commit tags, Never pull policy, live eligible-node discovery, imports before workloads, previous-image retention and new-node sync.
  Validation: agent-run bash syntax, YAML parse and git diff --check passed 2026-09-14. Required build/runtime/deployment checks have not run.
  Differences: Live node inspection passed; build/import not run.
  Pending user command: ./scripts/deploy-homelab.sh build after review/commit; inspect and deploy after prerequisites. Cwd: /home/mduren/Code/career-strategy/migration-step-6. Prerequisites/results: deploy/homelab/README.md.
  Next action/dependencies: inspect user check results first; remaining acceptance as above.

- [x] **6.6.** Make deployment check the intended kubecontext/namespace, create/update secrets without logging values, run the migration Job to completion, apply manifests, and wait for rollout. Preserve the previous image on every eligible node for rollback. Document how new nodes receive imported images.

  State (2026-09-14): complete.
  Changed files: step 6 deployment manifests and scripts/deploy-homelab.sh, committed at 684ce848098bb563cdcd54575582b691c7b507bd.
  Implemented/evidence: User-reported migration Job career-migrate-9vdjw completed and Deployment rolled out successfully. Agent verified Ready pod with zero restarts. First release reports previous=none, so no prior image exists to retain. Future-release retention/rollback code is present; a rollback rehearsal remains final criterion C.5.
  Validation: user-reported deploy PASS; bounded agent-run live kubectl/HTTP checks passed at 21:17 UTC. Context default, namespace career-strategy, API https://192.168.20.3:6443.
  Differences: none; earlier pending records below are historical and superseded.
  Pending user command: none for 6.6.
  Next action/dependencies: remaining 6.4 deployed outage/recovery, 6.7 HTTPS and 6.8 operational acceptance.

  State (2026-09-14): awaiting-user-command.
  Changed files: scripts/deploy-homelab.sh, deploy/homelab/migration-job.yaml, README.md (in dedicated step 6 worktree).
  Implemented/evidence: Checks exact context/API endpoint and namespace, securely loads secrets, awaits unique migration Job before Deployment, records rollback image before rollout.
  Validation: agent-run bash syntax, YAML parse and git diff --check passed 2026-09-14. Required build/runtime/deployment checks have not run.
  Differences: Migration/rollout/rollback not run; no prior app image currently recorded.
  Validation update (2026-09-14): user reported manifest dry-run PASS and deploy failure from merged migration commit 752940c because its image tag was never built. Failure occurred at local image lookup before import/migration/rollout. Agent confirmed only migration.md differs from the built/tested clean topic commit 684ce84. Rerun deploy from /home/mduren/Code/career-strategy/migration-step-6 to reuse the tested image; this supersedes earlier pending commands below.
  Pending user command: ./scripts/deploy-homelab.sh deploy after successful checks/build/secrets/validate. Cwd: /home/mduren/Code/career-strategy/migration-step-6. Prerequisites/results: deploy/homelab/README.md.
  Next action/dependencies: inspect user check results first; remaining acceptance as above.

- [ ] **6.7.** Configure HTTPS for the final public hostname, using the existing tunnel approach if public access is retained. Add app-specific tunnel routing/secrets and pin a tested cloudflared image if deploying a new tunnel. Verify secure cookies, redirects, origin checks, and cache behavior through the actual external path, not just port-forwarding.

  State (2026-09-14): awaiting-user-command.
  Changed files: deploy/homelab/cloudflared.yaml, configmap.yaml, scripts/deploy-homelab.sh, README.md (in dedicated step 6 worktree).
  Implemented/evidence: Prepared dedicated remotely managed tunnel, token Secret and digest-only image prompt; documented final origin and service route.
  Validation: agent-run bash syntax, YAML parse and git diff --check passed 2026-09-14. Required build/runtime/deployment checks have not run.
  Differences: Tested image pin and provider route expected → unresolved input and acceptance, not a completed pin. Final DNS activation deferred to 8.3 to preserve cutover ordering.
  Pending user command: Tunnel phase only after dedicated provider configuration and reviewed digest; external HTTPS checks after staged routing. Cwd: /home/mduren/Code/career-strategy/migration-step-6. Prerequisites/results: deploy/homelab/README.md.
  Next action/dependencies: inspect user check results first; remaining acceptance as above.

- [ ] **6.8.** Document LAN DNS, public DNS/tunnel routing, kubeconfig location, release commands, logs, rollback, restore, and backup checks in `deploy/homelab/README.md`. Verify live cluster and DB state during implementation; repository configuration alone is not proof of availability.

  State (2026-09-14): awaiting-user-command.
  Changed files: deploy/homelab/README.md, scripts/deploy-homelab.sh (in dedicated step 6 worktree).
  Implemented/evidence: Documented kubeconfig, LAN/public routing, release, rollback, new nodes and step 5 backup/restore links. Live nodes/ingress and DB TCP inspected.
  Validation: agent-run bash syntax, YAML parse and git diff --check passed 2026-09-14. Required build/runtime/deployment checks have not run.
  Differences: DB TCP success is not authenticated/schema evidence; DNS availability and deployed acceptance remain pending.
  Validation update (2026-09-14, user-reported): first documented source-check handoff passed; next is PostgreSQL race coverage.
  Pending user command: ./scripts/deploy-homelab.sh check-postgres; follow remaining documented phases after acceptance. Cwd: /home/mduren/Code/career-strategy/migration-step-6. Prerequisites/results: deploy/homelab/README.md.
  Next action/dependencies: inspect user check results first; remaining acceptance as above.


## 7. Migrate data and prove parity

### Files and verification for step 7

Import/seed tooling, local PostgreSQL configuration, and database/race tests already exist from step 2. Reuse them; the remaining acceptance includes production backup-key export, browser migration, staging, restore, and performance at scale.

| Items | Read / update locations                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 7.1   | `scripts/inspect-migration-source.mjs`, `src/lib/workspace-server.ts`, `docs/migration/current-behavior.md`; create `scripts/export-netlify-workspace.mjs` for all production keys and metadata; archives under `.migration-private/` or a restricted external backup directory, never `public/`.                                                                                                                                                          |
| 7.2   | `cmd/api/main.go`, `internal/database/transfer.go`, `internal/database/model.go`, `internal/database/children.go`, `internal/database/projections.go`, `internal/database/migrations/001_workspace.sql`, `internal/database/database_test.go`, `docs/migration/local-development.md`.                                                                                                                                                                      |
| 7.3   | `src/lib/workspace.ts`, `src/lib/workspace-server.ts`, `src/lib/workspace-seed.ts`, `scripts/export-seed.mjs`, `tests/workspace.test.ts`, `tests/journals.test.ts`; create `scripts/normalize-migration-export.mjs` only if needed; original/normalized archives remain private.                                                                                                                                                                           |
| 7.4   | `internal/database/transfer.go`, `internal/database/database_test.go`, `tests/fixtures/migration/workspace-v2.json`; create `scripts/compare-migration-export.mjs` for source/target comparison and a redacted evidence report in `docs/migration/parity.md`.                                                                                                                                                                                              |
| 7.5   | `docker-compose.yml`, `.air.toml`, `Makefile`, `.env.example`, `scripts/dev.mjs`, `scripts/dev-auth.mjs`, `astro.config.mjs`, `package.json`, `tests/dev-auth.test.mjs`, `docs/migration/local-development.md`.                                                                                                                                                                                                                                            |
| 7.6   | `internal/server/routes_test.go`, `internal/config/config_test.go`, `internal/database/database_test.go`, `internal/database/model_test.go`, `tests/go-api.integration.mjs`, `tests/fixtures/migration/`, `Makefile`.                                                                                                                                                                                                                                      |
| 7.7   | `tests/workspace.integration.mjs`, `tests/journals.integration.mjs`, `tests/workspace.browser.mjs`, `tests/journals.browser.mjs`, `tests/pwa.browser.mjs`, `tests/dev-auth.test.mjs`, `test-auth.js`, `tests/go-api.integration.mjs`, `package.json`; keep `tests/workspace.test.ts`, `tests/timeline.test.ts`, `tests/search.test.ts`, `tests/journals.test.ts`, `tests/companies.test.ts`; deferred MCP files listed in 3.8.                             |
| 7.8   | Browser suites in 7.7, `src/components/MarkdownPreview.tsx`, `src/components/WorkspaceEditor.tsx`, `src/components/CompanyBoard.tsx`, `src/components/GoalTimeline.tsx`, `public/sw.js`; production container/ingress definitions from step 6; create `docs/migration/parity.md` for scenario results.                                                                                                                                                     |
| 7.9   | Baselines: `docs/migration/current-behavior.md`, `docs/migration/local-development.md`; instrumentation targets: `internal/server/server.go`, `internal/server/routes.go`, `internal/database/store.go`, `internal/database/database_test.go`; create `scripts/benchmark-migration.mjs` and `docs/migration/performance.md` for fixture generation/run instructions, measurements, and budget decisions. Private production-sized inputs stay outside Git. |
| 7.10  | `deploy/homelab/` files created in step 6, `scripts/deploy-homelab.sh`, `deploy/homelab/README.md`, career DB/backup playbooks from step 5; create `deploy/homelab/staging/` if separate manifests are needed and record exact filenames here.                                                                                                                                                                                                             |

Validation handoff: local DB setup from step 2, `make test-postgres`, `go build -o /tmp/career-migration ./cmd/api`, `node tests/go-api.integration.mjs`, `npx astro check`, `npm run tc -- --noEmit`, `npm run build`, and the five in-scope domain tests from step 3. After conversion, run `node --test tests/dev-auth.test.mjs`, `node tests/workspace.integration.mjs`, `node tests/journals.integration.mjs`, `node tests/workspace.browser.mjs`, `node tests/journals.browser.mjs`, `node tests/pwa.browser.mjs` with recorded server/browser/env prerequisites. Current `npm test` includes deferred MCP tests; update script separation before using it as the required migration suite. Have the user execute production export, import dry-run/commit, comparison, benchmarks, staging rollout, and backup/restore with explicit target identifiers. Record each result under the owning checkbox and link detailed reports. No production cutover until acceptance passes.

- [ ] **7.1.** Before removing the Blobs dependency, add a one-time export utility for production `content` plus all existing backup keys, including `backup-before-v2`, `backup-before-catalog-v1`, and `backup-before-journals-v2`. Record store identity, export time, source ETag, checksums, and schema markers. Keep exports outside Git and public assets.
- [ ] **7.2.** Implement dry-run validation and transactional import into an empty PostgreSQL workspace. Preserve every collection, Markdown body, identifier, timestamp, contact, goal detail, and optional field. Assign fresh opaque revisions to each imported entity and initialize the export change sequence, retaining the original ETag as provenance. Populate child tables and derived projections in the same transaction. Reject unsupported versions or invalid data with a useful report.
- [ ] **7.3.** If legacy normalization is needed, apply the existing workspace/catalog/journal migration behavior to an exported copy and archive both versions. Never silently drop unrecognized fields or merge seed content back into a current production workspace.
- [ ] **7.4.** Make repeat import safe: detect the same source checksum and refuse to overwrite an already modified target. Compare all collection counts, IDs, and parsed values against the exported/normalized source; byte-check Markdown bodies and report differences before accepting the import.
- [ ] **7.5.** Adapt the existing `docker-compose.yml`, `.air.toml`, and Makefile targets for local PostgreSQL development, and add a launcher that runs Go plus Astro development with same-origin API routing. Replace Netlify development storage/auth emulation. Give development and staging separate databases and credentials with no production write access.
- [ ] **7.6.** Add Go tests for handlers, browser auth, validation, and real-PostgreSQL persistence. Prove two concurrent edits of the same entity revision produce one success and one conflict; prove edits to different entities both persist, including a goal edit racing a note edit. Test parent/child transaction rollback, cascading deletes, and consistent full exports during concurrent writes. Test failed import rollback, missing DB behavior, and restart persistence.
- [ ] **7.7.** Adapt `tests/workspace.integration.mjs`, journal integration tests, browser suites, and auth tests to the Go runtime and cookie login. Keep relevant existing domain tests; defer MCP/OAuth suites explicitly. Run Go tests/race checks, TypeScript/Astro checks, production build, and the applicable integration/browser suites.
- [ ] **7.8.** Test login/logout/expiry, all entry CRUD, export, timeline drag/resize, company contacts/checklists, journals/dashboard updates, document aliases, direct dynamic URLs created after build, conflict draft recovery, Markdown sanitization, and PWA offline privacy against the production container.
- [ ] **7.9.** Measure baseline and migrated p50/p95 timings for representative list, detail, dashboard, timeline, and save interactions using production-sized and 10× fixtures. Record cold/warm runs, request/query counts, payload sizes, database time, handler time, and browser time through the actual ingress. Initial acceptance budgets on the homelab LAN: p95 ordinary data API reads/writes ≤100 ms server-side, visible data updates ≤300 ms after an interaction, and navigation to usable data ≤500 ms in a warmed browser; measure cold navigation and public-origin network latency separately. Full exports are measured separately. Investigate budget misses before cutover; verify unrelated body growth does not increase detail-read/save payloads or trigger workspace-wide parsing.
- [ ] **7.10.** Deploy a staging namespace/database using a restored copy. Verify ingress, migrations, DB connectivity, pod rescheduling across eligible nodes, readiness during a DB outage, and backup restore before changing production routing.

## 8. Cut over, retain rollback, and remove Netlify

### Files and verification for step 8

| Items | Read / update locations                                                                                                                                                                                                                                                                                                                                                   |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 8.1   | Old mutation paths: `src/pages/api/workspace.ts`, `src/pages/api/goals.ts`, `src/pages/api/mcp.ts`, `src/lib/mcp-server.ts`, `src/lib/workspace-server.ts`; source/routing identity: `docs/migration/current-behavior.md`; step 7's `scripts/export-netlify-workspace.mjs`; record freeze enforcement/deployment and private final archive path under this item.          |
| 8.2   | `cmd/api/main.go`, `internal/database/transfer.go`, `scripts/compare-migration-export.mjs`, `scripts/deploy-homelab.sh`, `deploy/homelab/migration-job.yaml`, `deploy/homelab/deployment.yaml`, `deploy/homelab/README.md` (new files from earlier steps).                                                                                                                |
| 8.3   | `deploy/homelab/ingress.yaml`, app tunnel config if created, `docs/migration/current-behavior.md`, `src/pages/login.astro`, `src/lib/auth.ts`, `src/components/WorkspaceEditor.tsx`, `src/components/GoalTimeline.tsx`; live DNS/tunnel provider routing cannot be changed by editing repo files alone.                                                                   |
| 8.4   | Create `scripts/restore-netlify-workspace.mjs` and document/test it in `deploy/homelab/README.md` before cutover; use `internal/database/transfer.go`, `src/lib/workspace-server.ts`, production identity from `docs/migration/current-behavior.md`, and step 7 comparison tooling. Preserve a tagged revision with the required Netlify dependencies/tools for recovery. |
| 8.5   | `deploy/homelab/README.md`, new career backup playbook, monitoring config from 5.6, new `docs/migration/parity.md`; record observation start/end, image, private backups and verified restore evidence here.                                                                                                                                                              |
| 8.6   | `package.json`, `package-lock.json`, `astro.config.mjs`, `netlify/functions/auth.js`, `netlify.toml`, `scripts/check-netlify-build.mjs`, `scripts/dev.mjs`, `scripts/dev-auth.mjs`, `.env.example`, `.gitignore`; `src/lib/auth.ts`, `src/lib/server-auth.ts`, `src/lib/workspace-server.ts` for obsolete imports; retain access to the tagged recovery tooling from 8.4. |
| 8.7   | `README.md`, `AUTH_README.md`, `.env.example`, `docs/migration/local-development.md`, `deploy/homelab/README.md`, `CLAUDE.md` (existing agent stack/context), `MCP.md`, `MCP-OAUTH-PLAN.md`, `src/pages/agents.astro`, `PWA.md`; bounded final audit: `src/`, `scripts/`, `netlify/` if still present, `package.json`, `astro.config.mjs`, `Makefile`, deployment files.  |
| 8.8   | `deploy/homelab/README.md`, observation/backup records in this file, `docs/migration/parity.md`, `package.json`, `Dockerfile`, deployment manifests; retirement of site/store access and migration credentials occurs in the respective live provider/secret stores.                                                                                                      |

Validation handoff: the user executes write freeze, final export/import/compare, release, routing, controlled writes, recovery rehearsal, dependency/lockfile regeneration, and eventual retirement using the concrete commands prepared in `deploy/homelab/README.md`. Capture source revision/checksum, target identity, release image, routing outcome, cross-session saved content, and rollback/restore evidence under each item. Repeat the relevant step 7 build/tests after cleanup; run the final container with only documented production inputs. Waiting out the observation window is pending state, not an agent polling loop. Do not check off 8.8 before its recorded end date and verified backup/retirement results.

- [ ] **8.1.** Schedule a short write freeze on the old app. Disable all old mutation endpoints, take the final consistent export, and record its revision/checksum. Keep old writes blocked throughout DNS/tunnel propagation so only one backend accepts saves.
- [ ] **8.2.** Import the final snapshot into production PostgreSQL, verify it, deploy the tested image, and smoke-test through the final hostname before enabling new writes. Keep the prior Netlify deployment and exports available for recovery.
- [ ] **8.3.** Switch routing, require fresh browser login/reload, and verify representative saved content and a controlled write/read from another session. Preserve/recover browser-local drafts before changing origins if a hostname change is necessary; localStorage does not transfer between origins.
- [ ] **8.4.** Define rollback triggers such as missing data, persistent auth failures, or failed saves. Before any new Go writes, rollback can restore old routing and unfreeze Netlify. After Go accepts writes, freeze Go first and export/reconcile its latest data back into Blobs using a prepared, tested recovery utility before reopening Netlify; a DNS-only rollback would lose new changes.
- [ ] **8.5.** Observe logs, readiness, conflicts, DB connectivity, and at least one scheduled backup/verified restore. Keep old production read-only for an agreed observation window, proposed seven days, and retain dated exports under the backup policy.
- [ ] **8.6.** Remove `@astrojs/netlify`, `@netlify/blobs`, obsolete auth-only JavaScript dependencies, `netlify/functions/auth.js`, `netlify.toml`, `scripts/check-netlify-build.mjs`, and Netlify-specific npm scripts/configuration; regenerate the lockfile. Preserve a tagged pre-migration revision and the recovery utility until rollback retirement.
- [ ] **8.7.** Update `README.md`, `AUTH_README.md`, environment examples, and developer/deployment instructions. Mark MCP/OAuth unavailable in this deployment. Search active runtime/build paths for remaining Netlify URLs, environment assumptions, SDK imports, and emulator behavior.
- [ ] **8.8.** After the observation window and final backup verification, retire Netlify hosting/storage access and migration credentials. Confirm the app builds and runs with only Go, the static build, PostgreSQL, and homelab deployment settings.

## Completion criteria

Use the item record format above for final acceptance as well. Evidence locations: **C.1** → step 7.9, `docs/migration/performance.md`; **C.2** → steps 4/6/8, `Dockerfile`, `deploy/homelab/README.md`, final `package.json`/`astro.config.mjs`; **C.3** → steps 2/7, `internal/database/migrations/`, `docs/migration/parity.md`; **C.4** → steps 3/4/7.8, `docs/migration/parity.md` and browser-suite results; **C.5** → steps 5/6/8, `deploy/homelab/README.md` and recorded release/restore/rollback results. Proposed reports/manifests are created in their owning steps. Each criterion needs deployed evidence where specified, not merely passing local source tests.

- [ ] **C.1.** Routine data interactions use scoped indexed queries and entity revisions, avoid whole-workspace serialization and repeated Markdown parsing, and meet the measured step 7 performance budgets.

- [ ] **C.2.** The Go container serves the Astro build and all in-scope APIs through homelab ingress, with no Node server or Netlify runtime dependency.
- [ ] **C.3.** PostgreSQL stores production entities and their relationships in the step 2 tables, with a verified export round trip; concurrent edits, deleted entries, and restart persistence behave correctly.
- [ ] **C.4.** Browser login, protected pages, all editing workflows, dynamic URLs, exports, and private cache behavior pass checks against the deployed container.
- [ ] **C.5.** A fresh deployment, database restore, and rollback procedure have been exercised and documented. MCP/OAuth remain explicitly deferred.
