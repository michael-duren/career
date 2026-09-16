# Private career MCP with OAuth

The read-only MCP endpoint is `https://YOUR-SITE/api/mcp`. It uses stateless Streamable HTTP and reads the same live Netlify Blobs workspace as the website. OAuth grants access only to the configured owner and explicitly enrolled service identities.

The application and local integration tests are implemented. A live authorization provider has **not** been configured or qualified, and hosted client compatibility has **not** been tested against this deployment. Follow the setup and rollout checks below before enabling production access.

## Configure the authorization provider

Use a managed MCP-compatible OAuth authorization server. The app is the resource server; the provider hosts login, consent, client registration, authorization codes, refresh-token rotation and revocation. Website login remains separate.

1. Create a staging authorization environment and resource identifier equal to your staging `/api/mcp` URL. Use a stable HTTPS staging hostname and synthetic workspace data; keep production and staging audiences and identities separate.
2. Enable Authorization Code with PKCE `S256`, public clients, and rotating refresh tokens. Configure access tokens for **at most 600 seconds**; the verifier enforces this limit with 5 seconds of clock tolerance. A 30-day refresh-token idle lifetime is the suggested provider setting.
3. Enable `career:read`. Its consent description must include access to work and personal journals, biography, company contacts, goals, and all supporting documents. This is one scope for the complete current MCP surface, not collection-level sharing.
4. Enable CIMD and DCR for broad client compatibility, plus manually registered clients where needed. Configure exact hosted callback URIs shown by each client and RFC 8252 loopback callbacks for CLI clients. Do not use wildcard hosted redirects. The local bridge uses DCR by default or a pre-registered client ID.
5. Restrict sign-in/consent to your account. Obtain the account's immutable subject. The application independently checks this subject; creating another provider account cannot grant access to the shared workspace.
6. Configure a **MCP-specific audience** and JWT access-token profile: default `typ: at+jwt` (RFC 9068), or an explicitly selected provider profile with the signed claim `token_use: access`. Plain `typ: JWT` without that discriminator is rejected, as are ID tokens, opaque tokens, and website JWTs. The app accepts RS256, ES256, PS256 or EdDSA signatures and requires `iss`, `aud`, `sub`, `iat`, `exp`, and `scope` containing `career:read`.
7. Copy the exact issuer and `jwks_uri` from the provider's trusted discovery document. Verify a real issued token meets the profile; a provider advertising OAuth alone is insufficient.
8. For machine clients, explicitly create a confidential Client Credentials application with `career:read`. Tokens must identify its own `sub` and `client_id`; enroll both values below. Machine subjects must differ from the owner subject. DCR does not enroll machine identities.

WorkOS AuthKit was identified as a candidate in the plan, not a verified selection. Provider/account selection must satisfy the resource audience, token profile, registration and renewal requirements above. No provider management API key is needed by the resource server. [Managed MCP OAuth example](https://workos.com/blog/how-to-add-authentication-to-your-mcp-server), [MCP authorization specification](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)

## Configure Netlify Functions

Set these values in the site's **Functions** environment, then deploy:

```dotenv
CAREER_MCP_AUTH_MODE=oauth
CAREER_MCP_RESOURCE=https://YOUR-SITE/api/mcp
CAREER_MCP_OAUTH_ISSUER=https://YOUR-AUTH-ISSUER
CAREER_MCP_OAUTH_JWKS_URI=https://YOUR-AUTH-ISSUER/actual-jwks-path
CAREER_MCP_OAUTH_AUDIENCE=https://YOUR-SITE/api/mcp
CAREER_MCP_OWNER_SUBJECT=YOUR-IMMUTABLE-SUBJECT
CAREER_MCP_TOKEN_PROFILE=at+jwt
CAREER_MCP_ALLOWED_ORIGINS=
CAREER_MCP_SERVICE_IDENTITIES=[]
```

`CAREER_MCP_OAUTH_AUDIENCE` defaults to the resource URL. Only override it with another identifier exclusively assigned to this MCP. The issuer must exactly match the token's `iss`, including any trailing slash. The signing-key URL is explicitly pinned through configuration, never read from a token header.

`CAREER_MCP_ALLOWED_ORIGINS` is an optional comma-separated list of exact browser origins, such as `https://your-browser-client.example`. The resource origin is also allowed. Hosted server-to-server clients and CLI clients normally send no Origin and need no CORS entry. `Origin: null` and unapproved origins are rejected. Website cookies cannot authenticate MCP; MCP tokens cannot authenticate website editing APIs.

For a service client, set `CAREER_MCP_SERVICE_IDENTITIES` to JSON such as `[{"subject":"SERVICE-SUBJECT","clientId":"SCHEDULER-CLIENT-ID"}]`. Enrollment is server configuration, not an MCP tool. Keep service client secrets in the agent's secret manager, not in the site's frontend or Git.

Missing mode defaults to `disabled`, even if an old `CAREER_MCP_TOKEN` exists. Invalid OAuth configuration returns 503. Emergency disable: set `CAREER_MCP_AUTH_MODE=disabled` and redeploy. Temporary legacy migration requires explicitly selecting `legacy` plus the previous random token (at least 32 characters); OAuth never falls back to it. The upgraded bridge is OAuth-only. Remove legacy configuration after migration.

## Discovery and qualification

Both public routes return the same metadata without a website login:

- `/.well-known/oauth-protected-resource/api/mcp`
- `/.well-known/oauth-protected-resource`

Unauthenticated MCP requests return HTTP 401 with `WWW-Authenticate` pointing to metadata and requesting `career:read`. Insufficient scope returns 403 with a scope challenge; disallowed identities return 403 without exposing the subject. The authorization server's discovery and OAuth endpoints live on its own domain.

After staging deployment, load the server configuration into a local shell and run:

```sh
npm run check:mcp-oauth
```

This read-only check validates discovery and signing-key configuration and reports advertised provider capabilities. Supply a real access token through the secret environment variable `CAREER_MCP_CHECK_ACCESS_TOKEN` to additionally validate its signature, identity, audience, token type, scope and lifetime. The command never prints the token or subject. Metadata alone does not prove sign-in, rotation, or client compatibility.

## Connect chat clients and coding agents

Use the remote `/api/mcp` URL in a client that supports Streamable HTTP and OAuth. Complete browser sign-in and consent separately for each product. Product/account restrictions still apply; OAuth cannot give MCP support to an unsupported chat interface.

For Codex, add this to its MCP configuration and then run `codex mcp login career`:

```toml
[mcp_servers.career]
url = "https://YOUR-SITE/api/mcp"
```

ChatGPT uses its supported MCP/plugin management flow; copy the exact callback displayed there. Claude uses a remote custom connector; Claude Code authenticates remote servers through `/mcp`. Gemini CLI supports OAuth discovery and `/mcp auth`. For Cursor or other IDEs, validate the installed version's remote OAuth support and use the local bridge if needed. [OpenAI authentication](https://developers.openai.com/plugins/build/auth), [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli), [Claude connectors](https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp), [Claude Code](https://code.claude.com/docs/en/mcp), [Gemini CLI](https://geminicli.com/docs/tools/mcp-server/)

OAuth does not automatically publish or install a plugin, or transfer a connection between unrelated products. Test all four tools after linking. Resources and prompts are optional entry points for clients that support them.

## Local stdio clients

Requires Node 22.18+ and this repository's installed dependencies. Set the URL in your shell and authorize once:

```sh
export CAREER_MCP_URL=https://YOUR-SITE/api/mcp
node scripts/career-mcp.mjs login
```

Open the printed authorization URL in a browser on the same machine. The bridge listens only on `127.0.0.1` for up to three minutes, verifies callback state/issuer and exchanges the code with PKCE. It does not automatically launch a GUI browser. With DCR it registers an exact temporary callback; reauthorization creates a new registration. For pre-registration set `CAREER_MCP_CLIENT_ID`, optionally `CAREER_MCP_CLIENT_SECRET`, and `CAREER_MCP_CALLBACK_PORT`; register `http://127.0.0.1:PORT/callback` exactly or use your provider's standards-compatible variable-port policy. Set `CAREER_MCP_SCOPES='career:read offline_access'` if the provider requires offline access to issue refresh tokens.

Configure the stdio client:

```json
{
  "mcpServers": {
    "career": {
      "command": "/absolute/path/to/node",
      "args": ["/absolute/path/to/career-strategy/scripts/career-mcp.mjs"],
      "env": {
        "CAREER_MCP_URL": "https://YOUR-SITE/api/mcp"
      }
    }
  }
}
```

Use the same client ID, scopes and credential directory settings for login and stdio startup when overriding defaults. The launcher intentionally does not load `.env` files. `npm run mcp` is a manual convenience; clients should launch `node` directly to keep npm output out of the protocol.

Credentials use the portable file fallback: `~/.career-mcp`, directory mode 0700 and files 0600. No OS-keychain integration is currently installed. `CAREER_MCP_CREDENTIAL_DIR` can override this with an equally private local directory. Records are separated by issuer/resource/client configuration. Do not put this directory in Git, a shared filesystem or a synced folder. Renewal is serialized across processes with a lock, and credentials are atomically replaced. If a process is forcibly killed, inspect the PID in the corresponding `.lock/pid` file; remove that lock only after verifying its process has exited. Locks are never stolen by age. Permission enforcement is intended for POSIX hosts; use a native OAuth client on platforms without equivalent filesystem permissions.

Normal startup renews tokens automatically and never opens an interactive flow. If the grant expires or is revoked, rerun `login`. To revoke and clear the saved grant:

```sh
node scripts/career-mcp.mjs logout
```

Logout requires a working provider revocation endpoint. If revocation fails, credentials are retained so you can retry; revoke through the provider dashboard if needed. Revoking a refresh grant stops renewal. Already issued JWTs can remain valid for up to 600 seconds plus 5 seconds clock tolerance. Removing a service enrollment or disabling MCP takes effect after redeploy. Machine client credentials must be disabled at the provider to prevent new tokens.

## Unattended agents

For owner-delegated work, authorize the bridge once on the agent host and securely persist its rotating refresh grant. For an independent enrolled service, use:

```dotenv
CAREER_MCP_URL=https://YOUR-SITE/api/mcp
CAREER_MCP_GRANT_TYPE=client_credentials
CAREER_MCP_CLIENT_ID=YOUR-SERVICE-CLIENT-ID
CAREER_MCP_CLIENT_SECRET=SECRET-FROM-AGENT-SECRET-MANAGER
```

The bridge requests short-lived tokens and renews them without browser interaction. It supports Basic or form client-secret authentication through the SDK; signed client assertions are not implemented in this bridge. A custom API agent can obtain OAuth tokens through its own credential manager and pass the access token to its remote MCP integration. Client Credentials is an extension, not a universally supported chat-client login option. [MCP machine authentication](https://modelcontextprotocol.io/extensions/auth/oauth-client-credentials)

## Tools and privacy

- `get_career_overview`: live goals, counts, revision, and context rules.
- `list_career_entries`: paginated IDs/titles for goals, journals, notes, pages, books and companies.
- `search_career_context`: word search with bounded excerpts and source IDs.
- `read_career_entry`: metadata and full text in bounded chunks; pass revisions to detect changes.

Resource `career://overview` and prompt `career_conversation` offer alternate entry points. Current timeline goals are authoritative; journals and references supply historical context. Every operation reloads saved workspace data. Storage failures produce an error. Existing initialization/migrations may run on the first read, as in the website. Suggested changes must be saved through the website.

There are no write tools, outbound website fetches, or API-key passthrough. MCP responses remain private/no-store and are not cached by the PWA. Actual MCP requests require a bearer token; OPTIONS handles browser preflight without one. Authenticated GET/SSE and DELETE return 405 because the server creates no persistent session. Request bodies remain limited to 64 KiB.

Server audit records contain generated request IDs, HTTP outcome, a provider-validated client ID when available, and known tool names. Tokens, subjects, request bodies, provider errors and career content are excluded. The local bridge writes interactive messages only to stderr and refuses remote HTTP and redirects carrying credentials.

## Verify and roll out

```sh
npm test
node tests/mcp.integration.mjs
npm run tc
npm run build
npm run check:netlify-build
```

Tests use an isolated, **test-only** authorization server and temporary credentials. Never deploy `tests/fixtures/oauth-server.mjs`: its authorization endpoint simulates an already authenticated owner. Tests cover denial, PKCE/code/refresh replay, token validation, key rotation, owner/service restrictions, public Astro discovery, CORS, real CLI login/logout, HTTP tools, stdio, and concurrent renewal across bridge processes.

Before production, record the actual provider, account tier/cost, issuer, audience, access-token profile, owner mapping, callback policy, and refresh lifetime. Then complete the staging matrix:

| Target | Required evidence | Current status |
| --- | --- | --- |
| Local HTTP + stdio | Login, four tools, refresh, reconnect, revocation | Automated local fixture verified |
| ChatGPT | Account/version/date, registration mode, OAuth, tools, renewal, revocation | Pending live provider/account |
| Codex | Same checks | Pending live provider/account |
| Claude connector + Claude Code | Same checks, each surface | Pending live provider/account |
| Gemini CLI + Cursor | Same checks, each installed client | Pending live provider/account |
| Unattended service | Enrolled identity, token renewal, denied unregistered service | Local fixture verified; live provider pending |

Only deploy production OAuth configuration after live staging passes. Connect each client, remove the old shared token, and retain `disabled` as the emergency rollback. See [the implementation plan](MCP-OAUTH-PLAN.md) for the original architecture and remaining external setup milestones.
