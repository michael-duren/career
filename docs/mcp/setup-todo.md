# Setup TODO

> **Status (2026-09-19):** the app now runs as a Go service on the homelab k3s cluster and Netlify is no longer used. The Go server does not serve MCP, so the Netlify-hosted MCP endpoint and OAuth setup described here are not deployed. Treat this as a plan to revisit if MCP is ported to Go.

## MCP OAuth

- [ ] Choose an OAuth provider that supports MCP OAuth, PKCE `S256`, CIMD or DCR, refresh tokens, JWKS, and revocation. WorkOS AuthKit is the current candidate.
- [ ] Decide the production MCP URL: `https://your-domain.example/api/mcp`.
- [ ] Create a staging OAuth environment with a separate issuer, audience, owner identity, and synthetic workspace data.
- [ ] Configure the provider's issuer URL and JWKS URL.
- [ ] Configure a MCP-specific audience matching the MCP URL.
- [ ] Enable the `career:read` scope and describe that consent grants access to work and personal journals, biography, goals, company contacts, and supporting documents.
- [ ] Enable Authorization Code with PKCE `S256`.
- [ ] Enable rotating refresh tokens and set access tokens to 10 minutes or less.
- [ ] Enable Client ID Metadata Documents (CIMD) and Dynamic Client Registration (DCR).
- [ ] Register exact hosted callbacks for ChatGPT, Claude, and other clients.
- [ ] Allow standards-compatible loopback callbacks for local CLI clients.
- [ ] Restrict provider sign-in and consent to the workspace owner.
- [ ] Record the owner's immutable OAuth `sub` claim.

## Netlify configuration

- [ ] Set these Function environment variables in staging:

  ```dotenv
  CAREER_MCP_AUTH_MODE=oauth
  CAREER_MCP_RESOURCE=https://your-domain.example/api/mcp
  CAREER_MCP_OAUTH_ISSUER=https://your-provider.example
  CAREER_MCP_OAUTH_JWKS_URI=https://your-provider.example/.well-known/jwks.json
  CAREER_MCP_OAUTH_AUDIENCE=https://your-domain.example/api/mcp
  CAREER_MCP_OWNER_SUBJECT=your-provider-subject
  CAREER_MCP_TOKEN_PROFILE=at+jwt
  CAREER_MCP_ALLOWED_ORIGINS=
  CAREER_MCP_SERVICE_IDENTITIES=[]
  ```

- [ ] Deploy staging.
- [ ] Run `npm run check:mcp-oauth` against staging.
- [ ] Set `CAREER_MCP_CHECK_ACCESS_TOKEN` temporarily to validate a real provider token without printing it.
- [ ] Confirm the token has the expected issuer, audience, access-token type, owner subject, scope, and lifetime.

## Client connections

- [ ] Connect Codex with the remote MCP URL, then run `codex mcp login career`.
- [ ] Add the MCP through ChatGPT's connector/plugin interface and complete OAuth.
- [ ] Add a remote custom connector in Claude and complete OAuth.
- [ ] Add the remote server in Claude Code and authenticate with `/mcp`.
- [ ] Configure Gemini CLI and authenticate with `/mcp auth`.
- [ ] Test remote OAuth in the installed Cursor version.
- [ ] Record each client's version, registration method, first login, tool calls, refresh, reconnect, and revocation result.

## Local and unattended agents

- [ ] Set `CAREER_MCP_URL` locally.
- [ ] Run `node scripts/career-mcp.mjs login` and complete browser authorization.
- [ ] Configure stdio-only clients to launch `node scripts/career-mcp.mjs`.
- [ ] Keep the credential directory private and outside Git.
- [ ] Create separate confidential OAuth clients for unattended agents.
- [ ] Enroll each service subject and client ID in `CAREER_MCP_SERVICE_IDENTITIES`.
- [ ] Store service client secrets in the agent's secret manager.
- [ ] Verify concurrent refresh, expiration, revocation, and rejection of unregistered services.

## Production rollout

- [ ] Repeat the provider and client checks against production with real configuration.
- [ ] Verify all four MCP tools and supported resources/prompts in every target client.
- [ ] Confirm website cookies and website JWTs cannot authenticate MCP.
- [ ] Confirm MCP tokens cannot access website editing APIs.
- [ ] Remove `CAREER_MCP_TOKEN` and all static-token client configuration.
- [ ] Keep `CAREER_MCP_AUTH_MODE=disabled` as the emergency rollback setting.
- [ ] Document the provider, issuer, audience, token profile, owner mapping, callbacks, lifetimes, and revocation behavior in `docs/mcp/README.md`.

See [MCP setup and verification](README.md) and [the implementation plan](oauth-plan.md) for details.
