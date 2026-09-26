// Run after make postgres-up migrate; uses a disposable note and removes it.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { setTimeout } from 'node:timers/promises';
import bcrypt from 'bcryptjs';
const origin = 'http://127.0.0.1:8088';
const child = spawn(process.env.GO_BINARY || '/tmp/career-migration', ['serve'], { env: { ...process.env, APP_ENV: 'development', LISTEN_ADDR: '127.0.0.1:8088', PUBLIC_ORIGIN: origin, STATIC_DIR: process.env.STATIC_DIR || 'dist', AUTH_USERNAME: 'migration-test', AUTH_PASSWORD_HASH: bcrypt.hashSync('local-test-password', 4), JWT_SECRET: 'local-migration-test-secret-32-characters', DATABASE_URL: process.env.TEST_DATABASE_URL || 'postgres://career_dev:career_dev_local@127.0.0.1:5433/career_dev?sslmode=disable' }, stdio: ['ignore', 'ignore', 'pipe'] });
let errors = ''; child.stderr.on('data', chunk => errors += chunk); child.on('error', error => errors += error.message);
let cookie = ''; let revision; let goalRevision; const id = `migration-test/${crypto.randomUUID()}`; const goalId = crypto.randomUUID();
async function request(path, method = 'GET', body, extra = {}) {
  return fetch(origin + path, { method, headers: { Origin: origin, 'Content-Type': 'application/json', Cookie: cookie, ...extra }, body: body === undefined ? undefined : JSON.stringify(body) });
}
try {
  let response;
  let ready = false;
  for (let attempt = 0; attempt < 100; attempt++) { try { const r = await request('/readyz'); if (r.ok) { ready = true; break; } } catch {} await setTimeout(100); }
  assert.ok(ready, `Go service did not become ready: ${errors}`);
  assert.equal((await request('/api/entries/note')).status, 401);
  response = await fetch(origin + '/', { redirect: 'manual' }); assert.equal(response.status, 303); assert.match(response.headers.get('location'), /^\/login/);
  response = await request('/login'); assert.equal(response.status, 200); assert.match(response.headers.get('content-type'), /text\/html/);
  response = await request('/api/mcp', 'POST', {}); assert.equal(response.status, 401); assert.match(response.headers.get('www-authenticate'), /resource_metadata=/);
  response = await request('/.well-known/oauth-protected-resource'); assert.equal(response.status, 200); assert.equal((await response.json()).resource, `${origin}/api/mcp`);
  response = await request('/api/auth/login', 'POST', { username: 'migration-test', password: 'local-test-password' });
  assert.equal(response.status, 200); cookie = response.headers.get('set-cookie').split(';')[0];
  response = await request('/'); assert.equal(response.status, 200); assert.equal(response.headers.get('cache-control'), 'private, no-store');
  response = await request('/api/summaries/progress'); assert.equal(response.status, 200); assert.ok((await response.json()).entries.every(entry => !('body' in entry)));
  response = await request('/_astro/does-not-exist.js'); assert.equal(response.status, 404);
  response = await request('/unknown-page'); assert.equal(response.status, 404);
  response = await request('/api/unknown'); assert.equal(response.status, 404); assert.ok(!(await response.text()).includes('<html'));
  const entry = { id, title: 'HTTP round trip 🦆', topic: 'Migration test', description: '', tags: ['test'], body: '**Exact Markdown**\r\n' };
  response = await request('/api/entries/note', 'POST', { entry, revision: null }); assert.equal(response.status, 200); const saved = await response.json(); revision = saved.revision; assert.equal(saved.entry.body, entry.body);
  assert.equal((await request('/api/entries/note', 'POST', { entry, revision: null })).status, 409);
  assert.equal((await request('/api/entries/note', 'POST', { entry, revision }, { Origin: 'https://evil.example' })).status, 403);
  response = await request(`/api/entries/note?id=${encodeURIComponent(id)}`); assert.equal(response.status, 200); assert.equal((await response.json()).revision, revision);
  response = await request(`/notes/${id}`); assert.equal(response.status, 200); assert.match(await response.text(), /<title>Note<\/title>/);
  response = await request('/api/search?q=round%20trip'); assert.equal(response.status, 200); assert.ok((await response.json()).results.some(item => item.id === id));
  const goal = { id: goalId, title: 'Migration API goal', startDate: '2026-09-01', endDate: '2026-09-30', dailyHours: 1, color: '#67e8f9', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(), notes: [], metadata: {}, steps: [] };
  response = await request('/api/goals', 'POST', { goal, revision: null }); assert.equal(response.status, 200); goalRevision = (await response.json()).revision;
  response = await request('/api/goals?from=2026-09-15&to=2026-09-15'); assert.equal(response.status, 200); assert.ok((await response.json()).goals.some(item => item.id === goalId));
  response = await request('/api/agent-context?journal=work'); assert.equal(response.status, 200); assert.equal((await response.json()).journal, 'work');
  response = await request('/api/agent-context?format=markdown'); assert.equal(response.status, 200); assert.match(await response.text(), /Migration API goal/);
  assert.equal((await request('/api/agent-context?journal=invalid')).status, 400);
  response = await request('/api/entries/note?topic=Migration%20test&limit=1'); const page = await response.json(); assert.equal(page.entries.length, 1); assert.ok(!('body' in page.entries[0].entry));
  response = await request('/api/entries/note', 'POST', { entry: { ...entry, title: 'Updated' }, revision }); assert.equal(response.status, 200); const updated = await response.json(); assert.notEqual(updated.revision, revision);
  assert.equal((await request('/api/entries/note', 'DELETE', { id, revision })).status, 409); revision = updated.revision;
  assert.equal((await request('/api/entries/document', 'DELETE', { id: 'index', revision: 'irrelevant' })).status, 400);
  assert.equal((await request('/api/entries/note', 'DELETE', { id, revision })).status, 200); revision = undefined;
  assert.equal((await request(`/notes/${id}`)).status, 404);
  assert.equal((await request('/api/goals', 'DELETE', { id: goalId, revision: goalRevision })).status, 200); goalRevision = undefined;
  assert.equal((await request(`/api/entries/note?id=${encodeURIComponent(id)}`)).status, 404);
  assert.equal((await request('/api/auth/logout', 'POST', {})).status, 200);
  console.log('Go HTTP smoke passed: readiness, cookie login, scoped CRUD, conflicts, origin checks, core protection, excluded MCP routes.');
} finally {
  if (revision) await request('/api/entries/note', 'DELETE', { id, revision }).catch(() => {});
  if (goalRevision) await request('/api/goals', 'DELETE', { id: goalId, revision: goalRevision }).catch(() => {});
  if (child.exitCode === null && child.signalCode === null) { const stopped = new Promise(resolve => child.once('exit', resolve)); child.kill('SIGTERM'); await stopped; }
}
