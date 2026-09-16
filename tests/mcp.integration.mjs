import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, readdir, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js';
import { startOAuthServer } from './fixtures/oauth-server.mjs';
import { createCareerOAuth } from '../scripts/lib/mcp-oauth.mjs';

const port = 4346;
const base = `http://127.0.0.1:${port}`;
const issuer = await startOAuthServer({ resource: `${base}/api/mcp` });
const directory = await mkdtemp(join(tmpdir(), 'mcp-integration-oauth-'));
const oauthEnv = { CAREER_MCP_URL: `${base}/api/mcp`, CAREER_MCP_CREDENTIAL_DIR: directory, CAREER_MCP_OAUTH_ISSUER: issuer.issuer };
const dev = spawn(process.execPath, ['scripts/dev.mjs', '--host', '127.0.0.1', '--port', String(port)], {
  env: { ...process.env, CAREER_MCP_AUTH_MODE: 'oauth', CAREER_MCP_RESOURCE: `${base}/api/mcp`, CAREER_MCP_OAUTH_ISSUER: issuer.issuer,
    CAREER_MCP_OAUTH_JWKS_URI: `${issuer.issuer}/jwks`, CAREER_MCP_OAUTH_AUDIENCE: `${base}/api/mcp`, CAREER_MCP_OWNER_SUBJECT: 'owner',
    CAREER_MCP_TOKEN_PROFILE: 'at+jwt', CAREER_MCP_ALLOWED_ORIGINS: 'https://chat.example', CAREER_MCP_SERVICE_IDENTITIES: '[]',
    CONTEXT: 'dev', BRANCH: 'mcp-oauth-verification' },
  stdio: 'ignore', detached: true,
});
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
const clients = [];
try {
  let ready = false;
  for (let i = 0; i < 120; i++) {
    if (dev.exitCode !== null) throw new Error('Local server exited before becoming ready');
    try { if ((await fetch(`${base}/api/mcp`)).status === 401) { ready = true; break; } } catch {}
    await pause(250);
  }
  assert.ok(ready, 'App starts and routes MCP through OAuth authentication');
  for (const path of ['/.well-known/oauth-protected-resource', '/.well-known/oauth-protected-resource/api/mcp']) {
    const discovery = await fetch(`${base}${path}`, { redirect: 'manual' });
    assert.equal(discovery.status, 200, `${path} bypasses website login`);
    assert.equal((await discovery.json()).resource, `${base}/api/mcp`);
  }
  const preflight = await fetch(`${base}/api/mcp`, { method: 'OPTIONS', headers: { origin: 'https://chat.example', 'access-control-request-method': 'POST', 'access-control-request-headers': 'Authorization, Content-Type, MCP-Protocol-Version' } });
  assert.equal(preflight.status, 204);
  assert.equal(preflight.headers.get('access-control-allow-origin'), 'https://chat.example');
  assert.equal((await fetch(`${base}/api/mcp`, { headers: { cookie: 'auth_token=invalid' } })).status, 401);
  const login = spawn(process.execPath, ['scripts/career-mcp.mjs', 'login'], { env: { ...process.env, ...oauthEnv }, stdio: ['ignore', 'pipe', 'pipe'] });
  let loginOutput = ''; let loginStdout = ''; let browser;
  login.stdout.on('data', chunk => { loginStdout += chunk; });
  login.stderr.on('data', chunk => {
    loginOutput += chunk;
    const match = loginOutput.match(/http:\/\/127\.0\.0\.1:\d+\/authorize\?\S+/);
    if (match && !browser) browser = issuer.authorize(new URL(match[0])).then(callback => fetch(callback));
  });
  const loginTimeout = setTimeout(() => login.kill('SIGTERM'), 15000);
  try { assert.equal(await new Promise(resolve => login.once('exit', resolve)), 0, 'CLI OAuth login succeeds'); await browser; }
  finally { clearTimeout(loginTimeout); }
  assert.equal(loginStdout, '', 'Login does not write credentials or non-MCP output to stdout');
  assert.doesNotMatch(loginOutput, /access_token|refresh_token|code_verifier/);
  const oauth = await createCareerOAuth(new URL(`${base}/api/mcp`), oauthEnv);
  const credentialFile = join(directory, (await readdir(directory)).find(name => name.endsWith('.json')));
  const credential = JSON.parse(await readFile(credentialFile, 'utf8'));
  const token = credential.tokens.access_token;
  const client = new Client({ name: 'http-integration', version: '1' });
  clients.push(client);
  await client.connect(new StreamableHTTPClientTransport(new URL(`${base}/api/mcp`), { fetch: oauth.fetch }));
  assert.equal((await client.listTools()).tools.length, 4);
  const overview = await client.callTool({ name: 'get_career_overview', arguments: {} });
  assert.ok(!overview.isError);
  assert.ok(JSON.parse(overview.content[0].text).counts);
  for (const [name, args] of [['list_career_entries', { kind: 'note', limit: 1 }], ['search_career_context', { query: 'career' }]]) {
    assert.ok(!(await client.callTool({ name, arguments: args })).isError);
  }
  const entries = JSON.parse((await client.callTool({ name: 'list_career_entries', arguments: { kind: 'note', limit: 1 } })).content[0].text).entries;
  assert.ok(entries.length > 0);
  assert.ok(!(await client.callTool({ name: 'read_career_entry', arguments: { kind: 'note', id: entries[0].id } })).isError);
  const browserApi = await fetch(`${base}/api/workspace`, { headers: { Authorization: `Bearer ${token}` } });
  assert.equal(browserApi.status, 401, 'MCP credential grants no website write access');
  const local = new Client({ name: 'stdio-integration', version: '1' });
  clients.push(local);
  credential.expiresAt = 0; await writeFile(credentialFile, JSON.stringify(credential), { mode: 0o600 });
  const other = new Client({ name: 'second-stdio-integration', version: '1' }); clients.push(other);
  await Promise.all([local, other].map(c => c.connect(new StdioClientTransport({ command: process.execPath, args: ['scripts/career-mcp.mjs'], env: oauthEnv, stderr: 'pipe' }))));
  assert.equal(issuer.counts.refreshes, 1, 'Separate bridge processes serialize refresh-token rotation');
  assert.equal((await local.listTools()).tools.length, 4);
  assert.ok(!(await local.callTool({ name: 'get_career_overview', arguments: {} })).isError);
  assert.equal((await local.listResources()).resources[0].uri, 'career://overview');
  assert.match(JSON.stringify(await local.readResource({ uri: 'career://overview' })), /source of truth/);
  assert.equal((await local.listPrompts()).prompts[0].name, 'career_conversation');
  const stranger = await issuer.mint({}, { subject: 'stranger' });
  assert.equal((await fetch(`${base}/api/mcp`, { headers: { authorization: `Bearer ${stranger}` } })).status, 403);
  const logout = spawn(process.execPath, ['scripts/career-mcp.mjs', 'logout'], { env: { ...process.env, ...oauthEnv }, stdio: 'ignore' });
  assert.equal(await new Promise(resolve => logout.once('exit', resolve)), 0);
  await assert.rejects(oauth.fetch(`${base}/api/mcp`), /login/);
  console.log('OAuth discovery, CLI PKCE login, HTTP tools, CORS, owner restriction, website isolation, cross-process refresh, stdio and revocation passed.');
} finally {
  await Promise.allSettled(clients.map(client => client.close()));
  try { process.kill(-dev.pid, 'SIGTERM'); } catch {}
  await issuer.close(); await rm(directory, { recursive: true, force: true });
}
