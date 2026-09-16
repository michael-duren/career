// Run against the local Netlify-backed dev server, never a deployed site.
import assert from 'node:assert/strict';
import { isEmptyStarterWeek, newWeek } from '../src/lib/workspace.ts';
import { readFile } from 'node:fs/promises';

const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:4321';
assert.ok(['localhost', '127.0.0.1'].includes(new URL(base).hostname));
// Exercise the actual function emulator: legacy login used to erase the Blobs
// environment, which tests that minted their own JWT could not catch.
const login = await fetch(`${base}/.netlify/functions/auth`, {
  method: 'POST',
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify({ action: 'login', username: process.env.AUTH_USERNAME || 'admin', password: process.env.AUTH_PASSWORD || 'local-dev-password' }),
});
assert.equal(login.status, 200, await login.clone().text());
const { token } = await login.json();
const headers = { cookie: `auth_token=${token}`, origin: base, 'content-type': 'application/json' };
const get = async () => {
  const response = await fetch(`${base}/api/workspace`, { headers });
  assert.equal(response.status, 200, await response.clone().text());
  return response.json();
};
const post = input => fetch(`${base}/api/workspace`, { method: 'POST', headers, body: JSON.stringify(input) });
assert.equal((await fetch(`${base}/api/workspace`)).status, 401);
assert.equal((await fetch(`${base}/api/workspace`, { headers: { cookie: 'auth_token=forged' } })).status, 401);
assert.equal((await fetch(`${base}/api/workspace`, { method: 'POST', headers: { ...headers, origin: 'https://elsewhere.example' }, body: '{}' })).status, 403);
let beforeMigration;
try { beforeMigration = JSON.parse(await readFile('.netlify/blobs-serve/entries/0/site:personal-workspace/content', 'utf8')); } catch { /* Fresh local store. */ }
let snapshot = await get();
assert.equal(snapshot.data.version, 2);
assert.ok(snapshot.data.books.some(book => book.slug === 'network-programming-using-internet-sockets'));
assert.ok(snapshot.data.companies.length > 0);
assert.ok(snapshot.data.documents.some(page => page.id === '2026/career-study-plan'));
if (beforeMigration?.version === 1) {
  assert.deepEqual(snapshot.data.notes, beforeMigration.notes);
  assert.deepEqual(snapshot.data.weeks, beforeMigration.weeks.filter(w => !isEmptyStarterWeek(w)));
  const backup = JSON.parse(await readFile('.netlify/blobs-serve/entries/0/site:personal-workspace/backup-before-v2', 'utf8'));
  assert.deepEqual(backup, beforeMigration);
}

assert.ok(snapshot.data.notes.some(n => n.id === 'ostep'));
assert.ok(snapshot.data.weeks.every(w => !isEmptyStarterWeek(w)));
const entry = { id: `test-${crypto.randomUUID()}`, title: 'Integration test note', topic: 'Testing', description: '', tags: ['test'], body: '**Hello**\n\n<script>alert(1)</script>' };
const beforeNotes = snapshot.data.notes.length;
let response = await post({ action: 'save', kind: 'note', revision: snapshot.revision, entry });
assert.equal(response.status, 200, await response.clone().text());
const saved = await response.json();
assert.equal(saved.data.notes.length, beforeNotes + 1);
assert.equal((await get()).data.notes.find(n => n.id === entry.id).body, entry.body);
response = await post({ action: 'save', kind: 'note', revision: snapshot.revision, entry: { ...entry, title: 'Stale overwrite' } });
assert.equal(response.status, 409);
response = await post({ action: 'save', kind: 'note', revision: saved.revision, entry: { ...entry, title: '' } });
assert.equal(response.status, 400);
response = await post({ action: 'delete', kind: 'note', revision: saved.revision, id: entry.id });
assert.equal(response.status, 200);
snapshot = await response.json();
assert.equal(snapshot.data.notes.length, beforeNotes);
assert.ok(!snapshot.data.notes.some(n => n.id === entry.id));
const originalWeek = snapshot.data.weeks.find(w => w.week === 21);
const week = structuredClone(originalWeek ?? newWeek('2026-09-07'));
const edited = { ...week, body: `${week.body}\nAPI integration check\n`, hours: { ...week.hours, ostep: 1.25 } };
response = await post({ action: 'save', kind: 'week', revision: snapshot.revision, entry: edited });
assert.equal(response.status, 200);
snapshot = await response.json();
try {
  const progress = await fetch(`${base}/progress`, { headers });
  assert.equal(progress.status, 200);
  assert.ok((await progress.text()).includes('1.25'));
  const logs = await fetch(`${base}/logs`, { headers });
  assert.equal(logs.status, 200);
  assert.ok((await logs.text()).includes('API integration check'));
  for (const path of ['/', '/journal', '/notes', '/notes/ostep', '/books', '/companies', '/manage/books', '/manage/companies', '/documents', '/2026/career-study-plan', '/2026/os-oss/ostep', '/2026/random/vintage-computers']) {
    const page = await fetch(`${base}${path}`, { headers });
    assert.equal(page.status, 200, path);
    assert.equal(page.headers.get('cache-control'), 'private, no-store');
  }
} finally {
  const current = await get();
  response = await post(originalWeek ? { action: 'save', kind: 'week', revision: current.revision, entry: week } : { action: 'delete', kind: 'week', revision: current.revision, id: week.slug });
  assert.equal(response.status, 200);
}
for (const [kind, collection, template, route] of [
  ['book', 'books', { title: 'Integration book', authors: ['Test Author'], category: 'Networking', type: 'book', status: 'reading', featured: false, priority: 'medium', tags: [], body: '## Chapters\n- [x] One\n- [ ] Two', progress: { unit: 'page', completed: 1, total: 4 } }, '/books'],
  ['company', 'companies', { title: 'Integration company', category: 'Testing', type: 'company', url: 'https://example.com', status: 'applied', featured: false, priority: 'high', tags: [], body: '## Steps\n- [x] Apply\n- [ ] Interview' }, '/companies'],
  ['document', 'documents', { title: 'Integration page', description: 'Test page', tags: [], body: '# Test page\n\n<div class="grid"><img src="/images/vintage/pdp11.jpg" alt="PDP" onerror="alert(1)" /></div>\n<script>alert(1)</script>' }, null],
]) {
  const id = `integration-${crypto.randomUUID()}`;
  const entry = { ...template, [kind === 'document' ? 'id' : 'slug']: id };
  try {
    const before = await get();
    let result = await post({ kind, action: 'save', entry, revision: before.revision });
    assert.equal(result.status, 200, await result.clone().text());
    const created = await result.json();
    assert.ok(created.data[collection].some(item => (item.id || item.slug) === id));
    const page = await fetch(`${base}${route || `/documents/${id}`}`, { headers });
    assert.equal(page.status, 200);
    const html = await page.text();
    assert.ok(html.includes(template.title));
    if (kind === 'document') {
      assert.ok(html.includes('/images/vintage/pdp11.jpg'));
      assert.ok(!html.includes('onerror='));
      assert.ok(!html.includes('<script>alert(1)</script>'));
    }
    result = await post({ kind, action: 'save', entry: { ...entry, body: 'Edited content' }, revision: created.revision });
    assert.equal(result.status, 200);
    const edited = await result.json();
    assert.equal(edited.data[collection].find(item => (item.id || item.slug) === id).body, 'Edited content');
    if (kind !== 'document') {
      result = await post({ kind, action: 'save', entry: { ...entry, url: 'javascript:alert(1)' }, revision: edited.revision });
      assert.equal(result.status, 400);
    }
  } finally {
    const current = await get();
    const result = await post({ kind, action: 'delete', id, revision: current.revision });
    assert.equal(result.status, 200);
  }
  assert.ok(!(await get()).data[collection].some(item => (item.id || item.slug) === id));
}
const current = await get();
assert.equal((await post({ kind: 'document', action: 'delete', id: 'index', revision: current.revision })).status, 400);
const vintage = await fetch(`${base}/2026/random/vintage-computers`, { headers });
assert.ok((await vintage.text()).includes('/images/vintage/pdp11.jpg'));
console.log('API integration passed: authentication, CSRF, migration, CRUD, validation, stale-write protection, and journal/dashboard integration.');
