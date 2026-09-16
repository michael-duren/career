import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, rm, readdir, readFile, writeFile, stat, chmod } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { spawn } from 'node:child_process';
import { createCareerOAuth, credentialStore, validateCallback, discoverCareerOAuth, mcpUrl } from '../scripts/lib/mcp-oauth.mjs';
import { startOAuthServer } from './fixtures/oauth-server.mjs';
import { startAuthorization } from '@modelcontextprotocol/sdk/client/auth.js';

test('OAuth login, private persistence, concurrent renewal, reconnect, machine tokens and revocation', async () => {
  const fixture = await startOAuthServer();
  const directory = await mkdtemp(join(tmpdir(), 'career-oauth-'));
  const env = { CAREER_MCP_CREDENTIAL_DIR: directory };
  try {
    const url = new URL(fixture.resource);
    const oauth = await createCareerOAuth(url, env);
    await assert.rejects(oauth.fetch(url), /login/);
    await oauth.login(async authorization => {
      assert.equal(authorization.searchParams.get('code_challenge_method'), 'S256');
      assert.equal(authorization.searchParams.get('resource'), fixture.resource);
      const callback = await fixture.authorize(authorization);
      assert.equal((await fetch(callback)).status, 200);
    });
    assert.equal((await oauth.fetch(url)).status, 200);
    assert.equal(fixture.counts.exchanges, 1);
    const file = join(directory, (await readdir(directory)).find(name => name.endsWith('.json')));
    assert.equal((await stat(file)).mode & 0o077, 0);
    const record = JSON.parse(await readFile(file, 'utf8'));
    const previousRefresh = record.tokens.refresh_token;
    record.expiresAt = 0;
    await writeFile(file, JSON.stringify(record), { mode: 0o600 });
    const second = await createCareerOAuth(url, env);
    const responses = await Promise.all([oauth.fetch(url), second.fetch(url), oauth.fetch(url)]);
    assert.ok(responses.every(r => r.status === 200)); assert.equal(fixture.counts.refreshes, 1);
    const renewed = JSON.parse(await readFile(file, 'utf8'));
    assert.notEqual(renewed.tokens.refresh_token, previousRefresh);
    assert.equal((await second.fetch(url)).status, 200);
    await oauth.logout(); assert.equal(fixture.counts.revocations, 1);
    await assert.rejects(second.fetch(url), /login/);
    const revokeCheck = await fetch(`${fixture.issuer}/token`, { method: 'POST', body: new URLSearchParams({ grant_type: 'refresh_token', resource: fixture.resource, client_id: renewed.client.client_id, refresh_token: renewed.tokens.refresh_token }) });
    assert.equal(revokeCheck.status, 400);
    await assert.rejects(oauth.fetch(new URL('http://127.0.0.1:1/api/mcp')), /another URL/);
    fixture.clients.set('scheduler', { client_id: 'scheduler', client_secret: 'test-only-secret', machine: true });
    const machine = await createCareerOAuth(url, { ...env, CAREER_MCP_GRANT_TYPE: 'client_credentials', CAREER_MCP_CLIENT_ID: 'scheduler', CAREER_MCP_CLIENT_SECRET: 'test-only-secret' });
    assert.equal((await machine.fetch(url)).status, 200); assert.equal(fixture.counts.machine, 1);
  } finally { await fixture.close(); await rm(directory, { recursive: true, force: true }); }
});

test('callback state/issuer/redirect, consent denial and discovery mismatch are rejected', async () => {
  const redirect = 'http://127.0.0.1:1234/callback';
  const metadata = { issuer: 'https://issuer.example', authorization_response_iss_parameter_supported: true };
  const callback = new URL(`${redirect}?state=expected&code=one&iss=https%3A%2F%2Fissuer.example`);
  assert.equal(validateCallback(callback, redirect, 'expected', metadata), 'one');
  for (const [key, value] of [['state', 'wrong'], ['iss', 'https://wrong.example'], ['error', 'access_denied'], ['code', '']]) {
    const bad = new URL(callback); bad.searchParams.set(key, value); assert.throws(() => validateCallback(bad, redirect, 'expected', metadata));
  }
  const missing = new URL(callback); missing.searchParams.delete('iss'); assert.throws(() => validateCallback(missing, redirect, 'expected', metadata));
  const duplicate = new URL(callback); duplicate.searchParams.append('state', 'expected'); assert.throws(() => validateCallback(duplicate, redirect, 'expected', metadata));
  assert.throws(() => validateCallback(callback, 'http://127.0.0.1:9999/callback', 'expected', metadata));
  for (const url of ['http://remote.example/api/mcp', 'https://user:secret@remote.example/api/mcp', 'https://remote.example/api/mcp?token=secret']) assert.throws(() => mcpUrl(url));
  const fixture = await startOAuthServer();
  const directory = await mkdtemp(join(tmpdir(), 'career-denied-'));
  try {
    await assert.rejects(discoverCareerOAuth(new URL(fixture.resource), 'https://wrong.example'), /issuer changed/);
    const oauth = await createCareerOAuth(new URL(fixture.resource), { CAREER_MCP_CREDENTIAL_DIR: directory });
    await assert.rejects(oauth.login(async authorization => {
      authorization.searchParams.set('test_deny', 'true');
      const callback = await fixture.authorize(authorization); assert.equal((await fetch(callback)).status, 400);
    }), /callback rejected/);
    assert.equal(fixture.counts.exchanges, 0);
  } finally { await fixture.close(); await rm(directory, { recursive: true, force: true }); }
});

test('credential store rejects unsafe permissions and serializes updates', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'career-lock-'));
  try {
    const store = await credentialStore(directory, 'key');
    await Promise.all(Array.from({ length: 5 }, () => store.locked(async (data, save) => { data.n = (data.n || 0) + 1; await save(); })));
    await store.locked(async data => { assert.equal(data.n, 5); });
    await chmod(directory, 0o755); await assert.rejects(credentialStore(directory, 'key'), /permissions/);
  } finally { await rm(directory, { recursive: true, force: true }); }
});

test('PKCE mismatch, code replay, and refresh-token reuse fail at the authorization server', async () => {
  const fixture = await startOAuthServer();
  try {
    const metadata = await (await fetch(`${fixture.issuer}/.well-known/oauth-authorization-server`)).json();
    const redirectUrl = 'http://127.0.0.1:4567/callback';
    const client = await (await fetch(`${fixture.issuer}/register`, { method: 'POST', body: JSON.stringify({ redirect_uris: [redirectUrl], token_endpoint_auth_method: 'none' }) })).json();
    const grant = async () => {
      const started = await startAuthorization(fixture.issuer, { metadata, clientInformation: client, redirectUrl, scope: 'career:read', state: 'fixture', resource: new URL(fixture.resource) });
      const callback = await fixture.authorize(started.authorizationUrl);
      return new URLSearchParams({ client_id: client.client_id, grant_type: 'authorization_code', code: callback.searchParams.get('code'), code_verifier: started.codeVerifier, redirect_uri: redirectUrl, resource: fixture.resource });
    };
    const exchange = body => fetch(`${fixture.issuer}/token`, { method: 'POST', body });
    const bad = await grant(); bad.set('code_verifier', 'wrong'); assert.equal((await exchange(bad)).status, 400);
    const good = await grant(); const response = await exchange(good); assert.equal(response.status, 200);
    const tokens = await response.json(); assert.equal((await exchange(good)).status, 400);
    const refresh = new URLSearchParams({ grant_type: 'refresh_token', client_id: client.client_id, resource: fixture.resource, refresh_token: tokens.refresh_token });
    const renewed = await exchange(refresh); assert.equal(renewed.status, 200); const next = await renewed.json();
    assert.equal((await exchange(refresh)).status, 400);
    refresh.set('refresh_token', next.refresh_token); assert.equal((await exchange(refresh)).status, 400);
  } finally { await fixture.close(); }
});

test('qualification command verifies a token and identifies missing provider capabilities without leaking credentials', async () => {
  const fixture = await startOAuthServer();
  try {
    const token = await fixture.mint();
    const command = spawn(process.execPath, ['scripts/check-mcp-oauth.mjs'], { env: { ...process.env, NODE_ENV: 'development',
      CAREER_MCP_AUTH_MODE: 'oauth', CAREER_MCP_RESOURCE: fixture.resource, CAREER_MCP_OAUTH_AUDIENCE: fixture.resource,
      CAREER_MCP_OAUTH_ISSUER: fixture.issuer, CAREER_MCP_OAUTH_JWKS_URI: `${fixture.issuer}/jwks`, CAREER_MCP_OWNER_SUBJECT: 'owner',
      CAREER_MCP_SERVICE_IDENTITIES: '[]', CAREER_MCP_ALLOWED_ORIGINS: '', CAREER_MCP_TOKEN_PROFILE: 'at+jwt', CAREER_MCP_CHECK_ACCESS_TOKEN: token,
    }, stdio: ['ignore', 'pipe', 'pipe'] });
    let output = ''; command.stdout.on('data', chunk => { output += chunk; }); command.stderr.on('data', chunk => { output += chunk; });
    assert.equal(await new Promise(resolve => command.once('exit', resolve)), 1, 'Fixture lacks CIMD, so it cannot qualify as the broad-compatibility provider');
    assert.match(output, /clientMetadataDocuments: needs provider configuration/);
    assert.match(output, /accessToken: verified/); assert.ok(!output.includes(token));
  } finally { await fixture.close(); }
});
