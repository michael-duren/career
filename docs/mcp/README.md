# Career MCP server

The Go service serves an MCP endpoint at `https://career.duckgc.com/api/mcp`
(`PUBLIC_ORIGIN` + `/api/mcp`). It uses stateless Streamable HTTP with JSON responses and
reads the same PostgreSQL workspace as the website.

## Authorization

The server is its own single-owner OAuth 2.1 authorization server. No external provider or
extra secret is needed:

- Discovery: `/.well-known/oauth-protected-resource[/api/mcp]` (RFC 9728) and
  `/.well-known/oauth-authorization-server` (RFC 8414).
- Dynamic client registration at `/oauth/register` (RFC 7591). Clients are public
  (`token_endpoint_auth_method: none`); redirect URIs must be HTTPS or loopback HTTP.
  Loopback callbacks may use any port (RFC 8252).
- `/oauth/authorize` requires the website login, then shows a consent page where you choose
  **Allow read only** (`career:read`) or **Allow read and edit** (`career:read career:write`).
  The client's requested scope does not decide this. Authorization Code with PKCE `S256` only.
- `/oauth/token` issues 1-hour access tokens and 30-day rotating refresh tokens with the
  chosen scope, audience `/api/mcp`. Refreshing keeps the scope; to change it, reconnect.

Client IDs, codes and tokens are HMAC-signed claim sets keyed from `JWT_SECRET` with a
separate key per purpose, so every replica verifies them without shared sessions. Codes and
refresh tokens are single use: their IDs are recorded in `oauth_consumed_tokens`
(migration 007), so replay fails on every replica. Website session cookies and website JWTs
cannot authenticate MCP, and MCP tokens cannot call website APIs.

**Revoke everything:** rotate `JWT_SECRET` and restart. This signs out the website too and
invalidates every MCP client registration and token; reconnect clients afterwards.
Individual grants cannot be revoked; an issued access token stays valid for up to one hour.
Refresh tokens are single use: if a client loses a refresh response, it must reconnect.

## Connect clients

**Claude (web, desktop, mobile):** Settings → Connectors → Add custom connector, URL
`https://career.duckgc.com/api/mcp`. Leave the OAuth client ID/secret fields empty. Claude
registers itself, opens the consent page (sign in to the website if asked) and returns.
Enable the connector in a chat from the tools menu.

**Claude Code:**

```sh
claude mcp add --transport http career https://career.duckgc.com/api/mcp
```

Then run `/mcp` in Claude Code and authenticate `career`.

Other clients that support remote MCP with OAuth discovery and dynamic registration (Codex,
Gemini CLI, Cursor) use the same URL.

The endpoint must be reachable from the client. Claude's hosted connectors call it from
Anthropic's servers, so it must be publicly reachable through the Cloudflare tunnel. An
access policy in front of `/api/mcp`, `/oauth/*` or `/.well-known/*` that requires a browser
login blocks the connector.

## Tools

- `get_career_overview`: current timeline goals with status, dates and step progress,
  entry counts per kind, and interpretation rules. Start here.
- `list_career_entries`: page through one kind without bodies.
- `search_career_context`: case-insensitive phrase search with excerpts.
- `read_career_entry`: one entry with all metadata and body as JSON, in chunks.
- `list_connection_companies`: employers of your LinkedIn connections, grouped by
  normalized company name (case and suffixes like Inc ignored), with people count, most
  recent conversation date, a few people with roles, and `trackedSlug` when the company is
  already on the companies board. Filter by `query`, `role` (count only matching people),
  `minConnections` and `untracked`; sort by `connections`, `recent` or `name`; page with
  `offset`/`limit` (1-100).
- `list_leetgrinder_todos`: named problem sets and individual problems. Returns set and
  item IDs for removal. A problem is done once a solved or struggled attempt is logged.
  Sets keep done problems with `done: true` and report `problemCount` and
  `remainingCount`; individual problems leave the list once done.

Kinds: `goal`, `work_journal`, `personal_journal`, `note`, `page`, `book`, `company`,
`connection`, `audio_thought`. Audio thoughts are private and only returned when
requested by kind. Prompt `career_conversation` offers a guided entry point.

Books, video courses, and papers (including short articles and blog posts) all use
`kind: "book"`. The entry's `type` field is `book`, `course`, or `paper`. List, search,
and read use `kind: "book"` for all three. To create a paper, pass
`{"kind":"book","entry":{"title":"...","type":"paper","category":"Systems","url":"https://..."}}`
to `create_career_entry`. For updates and deletes, read the item first and pass its
`id` and returned `revision` with `kind: "book"`.

Write tools (need **Allow read and edit**):

- `create_career_entry`: create a goal, company, note, bookshelf item or connection. IDs, slugs and
  timestamps are generated as random UUIDs, like the website.
- `update_career_entry`: patch a goal, company, note, bookshelf item or connection. Pass the
  `revision` from `read_career_entry` and only changed fields. A stale revision is
  rejected instead of overwriting newer edits. New steps, notes and todos may omit IDs.
- `delete_career_entry`: delete a goal, company, note, bookshelf item or connection. Needs the
  `revision` from `read_career_entry`; a stale revision is rejected instead of deleting
  the wrong version. Cannot be undone.
- `add_companies_to_queue`: batch-add 1-50 companies to the companies board as
  `not_started` with the website's outreach checklist and an empty Log, so no reach-out
  date is recorded. Only `title` is required; `category` defaults to "From connections",
  `url` to a LinkedIn company search link, `priority` to medium, and `why` fills the Why
  section (no `#`/`##` headings or unclosed code fences; `\r`, U+2028 and U+2029 count as
  line breaks). Titles need letters or digits. A name that `list_connection_companies`
  would report as tracked is skipped with `created: false` and the stored title and slug,
  so retries are safe. Tracked means connections at that employer are linked to a
  company, or the name equals a company's title or slug or starts with it as whole words,
  ignoring case and legal suffixes: with "Google" tracked, "Google DeepMind" is skipped.
  Use `create_career_entry` to add such a company deliberately. Within one batch only
  exact name repeats are skipped, so `["Acme", "Acme Robotics"]` creates both. Like a
  company created on the website, new companies pick up unlinked connections whose
  employer best matches them among all companies; the total is returned as
  `linkedConnections` and last-talked dates are not changed. The batch is all or nothing
  if any company fails validation.
- `create_leetgrinder_todo_set`: create a named set with up to 200 problems. Pass plain
  links or slugs in `problems`. Pass objects in `problemDetails` to copy each problem's
  `number`, `title`, `difficulty`, `topics`, and `metadata`. The set also accepts a
  `description` and `metadata`. Omit both problem arrays to create an empty set.
- `add_leetgrinder_todo_problem`: add a problem to a set with `setID`, or to the individual
  list without it. It accepts the same problem fields. Adding the same problem twice to one
  list keeps one entry; adding a done individual problem again queues it for another
  pass. `list_leetgrinder_todos` returns each entry's `sourceProblem` snapshot and
  catalog fields. Imported source snapshots stay on the catalog problem if
  a todo is removed.
- `remove_leetgrinder_todo_problem`: remove one todo entry by its ID.
- `remove_leetgrinder_todo_set`: remove a set and its todo entries by set ID. These tools
  do not delete attempt history or problem metadata.

Saves and deletes use the website's validation. Claude clients ask before
running write tools unless you allow them permanently. Request
bodies are limited to 64 KiB.

## Verify

```sh
make test-postgres   # includes the OAuth flow and MCP tool tests
```

`internal/server/mcp_test.go` covers discovery, registration, login redirect, consent,
cross-origin approval rejection, PKCE, code replay, refresh rotation, read tools, and write
tools (scope enforcement, revision conflicts, validation) and the connection company tools
through the Go MCP client. `internal/database/connection_companies_test.go` covers grouping,
filters, duplicate detection and linking.

## History

An earlier TypeScript MCP (Netlify Functions with an external OAuth provider) lives in
`src/lib/mcp-*.ts`, `scripts/career-mcp.mjs` and [oauth-plan.md](oauth-plan.md). It is not
deployed. The Go server replaces it.
