// Exercise only a local dev server with an isolated REVIEW_ID/CONTEXT store.
import assert from 'node:assert/strict';
const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:4322';
assert.ok(['localhost', '127.0.0.1'].includes(new URL(base).hostname));
const login = await fetch(`${base}/.netlify/functions/auth`, { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ action: 'login', username: 'admin', password: 'local-dev-password' }) });
assert.equal(login.status, 200);
const { token } = await login.json();
const headers = { cookie: `auth_token=${token}`, origin: base, 'content-type': 'application/json' };
const request = (path, method = 'GET', body) => fetch(`${base}${path}`, { method, headers, ...(body ? { body: JSON.stringify(body) } : {}) });
const workspace = async () => { const r = await request('/api/workspace'); assert.equal(r.status, 200); return r.json(); };
const context = async () => { const r = await request('/api/agent-context'); assert.equal(r.status, 200); assert.equal(r.headers.get('cache-control'), 'private, no-store'); return r.json(); };
assert.equal((await fetch(`${base}/api/agent-context`)).status, 401);
assert.equal((await fetch(`${base}/api/goals`, { method: 'DELETE', headers: { origin: base, 'content-type': 'application/json' } })).status, 401);
assert.equal((await request('/api/agent-context?journal=invalid')).status, 400);
let snapshot = await workspace();
assert.equal(snapshot.data.journalsVersion, 2);
assert.ok(snapshot.data.personalJournal.some(e => e.id === 'bio'));
const id = crypto.randomUUID();
const personal = { id, title: 'Integration personal entry', date: '2026-09-12', description: '', tags: [], body: `Life reflection ${id}` };
const now = new Date().toISOString();
let goal = { id: crypto.randomUUID(), title: 'Integration goal', startDate: '2026-09-12', endDate: '2026-12-12', color: '#67e8f9', createdAt: now, updatedAt: now, notes: [{ id: crypto.randomUUID(), body: 'Goal note', createdAt: now }], steps: [{ id: crypto.randomUUID(), title: 'First step', done: false }], metadata: { focus: 'test' } };
try {
  let r = await request('/api/workspace', 'POST', { action: 'save', kind: 'personal', entry: personal, revision: snapshot.revision });
  assert.equal(r.status, 200);
  let saved = await r.json();
  assert.deepEqual(saved.data.weeks, snapshot.data.weeks);
  assert.equal((await request('/api/workspace', 'POST', { action: 'save', kind: 'personal', entry: personal, revision: snapshot.revision })).status, 409);
  assert.equal((await request('/api/workspace', 'POST', { action: 'save', kind: 'personal', entry: { ...personal, date: '2026-02-30' }, revision: saved.revision })).status, 400);
  for (const format of ['markdown', 'json']) {
    const work = await (await request(`/api/agent-context?format=${format}&journal=work`)).text();
    const life = await (await request(`/api/agent-context?format=${format}&journal=personal`)).text();
    assert.ok(!work.includes(personal.body));
    assert.ok(life.includes(personal.body));
  }
  r = await request('/api/goals', 'POST', { goal, revision: saved.revision });
  assert.equal(r.status, 200);
  let goals = await r.json();
  assert.ok((await context()).goals.some(g => g.id === goal.id));
  const stale = goals.revision;
  goal = { ...goal, startDate: '2026-10-12', endDate: '2027-01-12' };
  r = await request('/api/goals', 'POST', { goal, revision: goals.revision });
  assert.equal(r.status, 200);
  goals = await r.json();
  const moved = (await context()).goals.find(g => g.id === goal.id);
  assert.equal(moved.startDate, '2026-10-12');
  assert.deepEqual(moved.notes, goal.notes);
  const markdown = await (await request('/api/agent-context?format=markdown')).text();
  assert.ok(markdown.includes('2026-10-12 → 2027-01-12'));
  assert.equal((await request('/api/goals', 'DELETE', { id: goal.id, revision: stale })).status, 409);
  assert.equal((await fetch(`${base}/api/goals`, { method: 'DELETE', headers: { ...headers, origin: 'https://invalid.example' }, body: JSON.stringify({ id: goal.id, revision: goals.revision }) })).status, 403);
  r = await request('/api/goals', 'DELETE', { id: goal.id, revision: goals.revision });
  assert.equal(r.status, 200);
  assert.ok(!(await context()).goals.some(g => g.id === goal.id));
  assert.ok(!(await (await request('/api/agent-context?format=markdown')).text()).includes(goal.title));
  for (const path of ['/agents', '/journal', '/personal-journal']) {
    r = await request(path);
    assert.equal(r.status, 200, path);
    const html = await r.text();
    assert.ok(html.includes('Personal journal'));
    assert.ok(html.includes('Work journal'));
    assert.ok(html.includes('Agent context'));
  }
} finally {
  const current = await workspace();
  if (current.data.goals?.some(g => g.id === goal.id)) await request('/api/goals', 'DELETE', { id: goal.id, revision: current.revision });
  const latest = await workspace();
  await request('/api/workspace', 'POST', { action: 'delete', kind: 'personal', id, revision: latest.revision });
}
console.log('Journal separation, authenticated exports, goal moves/deletes, conflicts, and SSR pages passed.');
