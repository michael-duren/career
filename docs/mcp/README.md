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

Kinds: `goal`, `work_journal`, `personal_journal`, `note`, `page`, `book`, `company`,
`connection`, `audio_thought`. Audio thoughts are private and only returned when
requested by kind. Prompt `career_conversation` offers a guided entry point.

Write tools (need **Allow read and edit**):

- `create_career_entry`: create a goal, company or note. IDs, slugs and timestamps are
  generated as random UUIDs, like the website.
- `update_career_entry`: patch a goal, company or note. Pass the `revision` from
  `read_career_entry` and only changed fields. A stale revision is rejected instead of
  overwriting newer edits. New steps, notes and todos may omit IDs.

Saves use the website's validation. There is no delete tool. Claude clients ask before
running write tools unless you allow them permanently. Request
bodies are limited to 64 KiB.

## Verify

```sh
make test-postgres   # includes the OAuth flow and MCP tool tests
```

`internal/server/mcp_test.go` covers discovery, registration, login redirect, consent,
cross-origin approval rejection, PKCE, code replay, refresh rotation, read tools, and write
tools (scope enforcement, revision conflicts, validation) through the Go MCP client.

## History

An earlier TypeScript MCP (Netlify Functions with an external OAuth provider) lives in
`src/lib/mcp-*.ts`, `scripts/career-mcp.mjs` and [oauth-plan.md](oauth-plan.md). It is not
deployed. The Go server replaces it.
