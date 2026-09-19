# Migration inputs and existing behavior

> **Historical (captured 2026-09-14).** Describes the Netlify deployment before the migration.
> Production now runs on the homelab k3s cluster (Go + PostgreSQL); Netlify is no longer used.

Captured 2026-09-14 from the current source, the read-only Netlify API/Blobs API, and local DNS. This document completes the discovery work in migration step 1. The browser still uses the existing Astro/Netlify implementation until steps 3–4 wire it to Go.

## Production identity and rollback inputs

| Input | Observed value |
| --- | --- |
| Public origin | `https://career.michaelduren.com` |
| Netlify site | `velvety-druid-ef18d0`, ID `8381ccc5-0c16-416b-af75-34a3e7117a58` |
| Repository / branch | `michael-duren/career-strategy`, `main` |
| Published deploy | `6aa732b2797a4500082c4789`, commit `6303b33d375a1f3e25775fd620e2cf121179ac72`, published `2026-09-13T23:33:49.454Z` |
| Build | `npm run build`, publish `dist`, Netlify adapter; `netlify.toml` functions directory `netlify/functions` |
| Production store / key | `personal-workspace` / `content`, strong consistency |
| Source ETag at `2026-09-14T11:30:17Z` | `"d2d3ac76b5de4a7ad835660fcd166fd4"` (quotes are part of the ETag) |
| Source markers | `version: 2`, `catalogVersion: 1`, `journalsVersion: 2` |
| Source counts | notes 9, weeks 13, books 19, companies 28, documents 13, personalJournal 2, goals 9 |
| DNS | Netlify reports managed DNS; public origin and Netlify alias both resolve to `18.208.88.157` / `98.84.224.111` from this machine |
| Proposed LAN name | `career.homelab` did not resolve; no matching declaration found in the homelab repo. This does **not** prove the name is unallocated: confirm with authoritative LAN DNS before deployment. |

The checkout was not linked to Netlify. The site was identified by its exact repository path, then the production store was read explicitly by site ID. `scripts/inspect-migration-source.mjs` repeats this read-only inspection using the existing Netlify CLI login; it prints only metadata, counts, size and timings. No production data, DNS, deployments, or infrastructure were changed.

The preview store formula in `astro.config.mjs` is `personal-workspace-preview-${sanitized(REVIEW_ID || BRANCH || CONTEXT).slice(0,32)}` whenever `CONTEXT` is set and is not `production`. Never substitute such a store for production. Export `content` and the `backup-before-v2`, `backup-before-catalog-v1`, and `backup-before-journals-v2` keys in step 7, then recapture ETag/checksum during the final write freeze. Preserve the current origin to retain bookmarks and localStorage drafts. The deploy above is a rollback reference, not a guarantee of Netlify retention; explicitly retain it before cutover.

## Browser API contracts before migration

All three APIs require the configured user's valid signed session. Successful/private responses are `Cache-Control: private, no-store`. Errors are JSON `{ error: string }`. Middleware can return a session-expiry 401 before the route-specific error. Unsupported methods use Astro's default method handling.

| Endpoint | Request | Success | Validation and failure behavior |
| --- | --- | --- | --- |
| `GET /api/workspace` | No payload | `{ data: Workspace, revision: string \| null }` | 401 unauthenticated; 503 storage/read/migration failure |
| `POST /api/workspace` | `{ kind, action: "save", entry, revision }` or `{ kind, action: "delete", id, revision }` | Entire `{ data, revision }` workspace | 150,000-byte request; 400 malformed JSON/kind/action/revision/entry, or delete of core document `index` / `2026/career-study-plan`; 403 nonmatching Origin; 415 non-JSON; 413 oversized entry; 409 global revision conflict; 503 save/read failure |
| `GET /api/goals` | No payload | `{ goals: data.goals ?? [], revision }` | 401 / 503 |
| `POST /api/goals` | `{ goal, revision }` | Entire `{ goals, revision }` collection | 150,000 bytes; goal schema below; same origin/content-type checks; 400 / 401 / 403 / 409 / 413 / 415 / 503 |
| `DELETE /api/goals` | `{ id: UUID, revision }` | Entire `{ goals, revision }` collection | 2,000 bytes; same origin/content-type checks; 400 / 401 / 403 / 409 / 413 / 415 / 503 |
| `GET /api/agent-context` | `format=json\|markdown` (default JSON), optional `journal=work\|personal` | Full JSON `{ revision, rules, goals, journals: {work, personal}, references: {notes,pages,books,companies} }`; filtered JSON `{ revision,journal,entries }`; or complete Markdown | 400 invalid format/journal; 401 unauthenticated; 503 storage error. Markdown content type `text/markdown; charset=utf-8`; source bodies fenced, not truncated. |

The old revision is a **global Blobs ETag**. Every mutation first reads the whole workspace and compares its ETag, then writes using `onlyIfMatch` (or `onlyIfNew` for null). An unrelated edit can therefore conflict. A missing blob may seed once; storage errors never mean missing content. `readWorkspace()` performs version/catalog/journal migrations with backup keys and up to three optimistic retries. The old aggregate save cap is 4,000,000 bytes.

Saving removes the previous entry from its collection and appends the replacement; deletion filters by ID. Saves stamp non-goal `updatedAt`; goal saves preserve an existing `createdAt` and set server-owned `updatedAt`. Nested child edits replace their parent's children. Goals in agent context sort by start date then UUID; work/personal journals sort newest first; empty generated work scaffolding is omitted. Saved timeline goals alone define current goals; historical journals must not recreate deleted goals.

## Validation inventory

`src/lib/workspace-server.ts` is the entry-schema reference; `src/pages/api/goals.ts` defines goals.

- Entry ID/slug: 1–200 characters, `[a-zA-Z0-9_/-]+`, including nested IDs. Title trimmed, 1–200. Body ≤100,000 JavaScript string units; description ≤1,000. Tags: ≤30, each trimmed 1–80; ordering and duplicates allowed. Notes additionally have trimmed topic 1–100.
- Personal journal date is a valid `YYYY-MM-DD` or `""`. Weeks have integer week 1–10,000 and year 2000–2200, original `YYYY-MM-DD to YYYY-MM-DD`, arbitrary track keys ≤80, hours/targets 0–168. Optional targets distinguish absent from `{}`. Go additionally validates actual week calendar dates/range; unsupported historical values fail dry run.
- Books: category in the eight `BOOK_CATEGORIES`; type book/course; status backlog/reading/paused/completed/reference; priority high/medium/low; required featured boolean. Authors ≤30, each ≤200 before trim/filter. Optional edition/isbn ≤1,000; optional URL/cover HTTP(S), empty interactive values become absent. Dates are valid calendar dates, empty interactive values become absent. Rating 0–5. Progress unit chapter/page/module/section/lecture; total/completed integers 0–100,000, completed ≤total, all three present together.
- Companies: trimmed category 1–200; type company; required HTTP(S) URL; optional cover; status not_started/applied/interviewing/offer/rejected/passed; same priority/featured/tags. Optional contacts ≤200: UUID, trimmed name 1–200 and role ≤200, email empty or valid ≤254, URL empty or HTTP(S), notes ≤5,000. Go rejects duplicate child IDs instead of losing rows to a primary key conflict.
- Goals: UUID, trimmed title 1–200, valid inclusive start/end dates in 1900–2200 with end ≥start, `#RRGGBB` color, optional dailyHours 0–24, required creation/update timestamps. Notes ≤500 (UUID, trimmed body 1–20,000, timestamp); steps ≤200 (UUID, trimmed title 1–500, boolean done); metadata ≤50 entries (trimmed key 1–80, value ≤2,000). Zero hours remains distinct from an unknown estimate.

Imports validate without trimming/reseeding/coercion and reject unknown fields, unknown versions, explicit nulls where the source schema does not allow them, invalid dates, and duplicate IDs. Interactive Go saves apply existing trimming/empty-optional normalization. Markdown bytes are never normalized. SQL stores date-only values as DATE and timestamps as TIMESTAMPTZ; import provenance preserves original timestamp spelling, including goal-note timestamps.

## Route inventory

The direct callers of `readWorkspace()` are `/`, `/books`, `/companies`, `/companies/[...slug]`, `/agents`, `/documents/[...id]`, `/api/workspace`, `/api/goals`, `/api/agent-context`, and the excluded `/api/mcp`. `getJournalEntries()` wraps `readWorkspace()` and serves `/logs` and `/progress`.

The `/2026/` document readers are:

- `/2026/career-study-plan`, `/2026/languages/`, `/2026/random/vintage-computers`
- `/2026/system-design/`, `/2026/system-design/ddia-read`, `/2026/system-design/hello-interview`, `/2026/system-design/alex-xu-vol1`
- `/2026/os-oss/`, `/2026/os-oss/container-internals`, `/2026/os-oss/ostep`, `/2026/os-oss/conference-talk`, `/2026/os-oss/build-runtime`

`/2026/algorithms/`, `/2026/algorithms/mit-6006`, `/2026/algorithms/mit-6042j`, `/2026/algorithms/leetcode`, and `/2026/os-oss/podman-contributions` redirect to `/2026/career-study-plan` with 302.

Request-dependent behavior:

| Route | Runtime dependency |
| --- | --- |
| `/login` | Reads `Astro.request` to verify auth and redirects signed-in users home. Browser redirect query is used after login. |
| `/notes`, `/notes/[...id]` | Editor loads workspace in browser; detail receives `Astro.params.id`, including nested IDs. No saved-note frontmatter read. |
| `/documents/[...id]` | Looks up `Astro.params.id` in saved documents; returns 404 if missing. |
| `/companies/[...slug]` | Looks up saved company; sets 404 for missing company and renders error content. |
| `/manage/[kind]` | Only `books`/`companies`; other kinds return 404. Reads query `id`; company edits redirect to encoded company detail, while books receive initial editor ID. |
| All private routes | `src/middleware.ts` checks the cookie; unauthenticated pages redirect to `/login?redirect=<pathname>`, private APIs return JSON 401. |

Current auth is Netlify username/password with bcrypt and a 24-hour HS256 token. `src/lib/auth.ts` keeps `auth_token` and `auth_username` in localStorage and JavaScript manages the cookie. The Go foundation uses an HttpOnly `session` cookie; browser caller migration and obsolete-token cleanup are explicitly step 3.

## Fixtures and expected user behavior

`tests/fixtures/migration/workspace-v2.json` is synthetic and contains all seven entity kinds, nested IDs, Unicode/emoji, duplicate ordered arrays, exact CRLF Markdown with HTML, optional/empty fields, undated and dated personal entries, unknown historical metric keys with zero values, empty targets, book/course progress, company contacts, core documents, and goal notes/steps/metadata. `expected-projections.json` was captured from the existing TypeScript shelf/board/checklist parsers by `scripts/capture-migration-fixtures.mjs`, not invented from the Go implementation.

Go tests compare complete relational exports to these fixtures; also exercise absent/empty top-level collections, empty/absent contacts, target-only metrics, dry-run and failed-import rollback, duplicate imports, deleted children, core protection, stale toggles, independent concurrent edits, same-revision conflicts, consistent snapshots during a write, and persistence after reconnect. Seeded entries deleted after import remain deleted; reads never load seed files.

Workspace editor drafts use `career-workspace:v1:<auth_username-or-user>:<kind>` in localStorage and retain `{ entry, revision }`. Reload/list refresh must not replace a dirty draft; save errors and 409 retain it. Selecting another entry asks before discarding. Quick journal drafts use kind `quick` and preserve date/body. GoalTimeline keeps its draft in React memory, warns before unload, and offers “Load latest (keep draft)” after conflicts; it does not currently persist the goal draft across a browser restart. Do not claim stronger draft recovery during migration. Preserve these behaviors when callers change in step 3.

## Initial timing evidence

A single read-only production-store fetch on this workstation transferred 167,908 bytes: 546.71 ms storage/network and 0.90 ms JSON decoding. This is not a p50/p95 or browser rendering measurement. It supports investigating storage latency separately, but does not prove the cause of the reported 1–2 second interactions.

Local PostgreSQL 17 `EXPLAIN (ANALYZE, BUFFERS)` with 1,000 notes and 10,000-character bodies: filtered 50-note list used `notes_topic_idx` (index-only scan, 26 buffer hits, 0.047 ms); one detail used `notes_pkey` (3 hits, 0.028 ms); revision-checked update used `notes_pkey` (12 hits total, 0.193 ms). These single warm SQL executions are not end-to-end budgets. Other domain indexes are installed; planner choices on small collections can legitimately be sequential scans. Production-sized/10× ingress, cold/warm, response/render/Markdown timings remain step 7 acceptance work.
