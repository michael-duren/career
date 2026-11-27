import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { GROUP_LABELS, RECENT_KEY, RECENT_LIMIT, REGISTRY_ORDER, addRecent, contextLine, groupResults, resultHref, resultIconPath } from '../public/ui/site-search.js';
import { ITEM_TYPE_PATHS } from '../public/ui/item-type-icons.js';
import { MODULES } from '../src/lib/modules.ts';

test('search groups cover exactly the modules with search results, in registry order', () => {
  const searchable = MODULES.filter(m => m.search).map(m => m.id);
  assert.deepEqual(REGISTRY_ORDER, searchable);
  assert.deepEqual(GROUP_LABELS, { timeline: 'Goals', schedule: 'Schedule', leetgrinder: 'Problems', companies: 'Companies', people: 'People', bookshelf: 'Books', journal: 'Journal', notes: 'Notes', audio: 'Audio thoughts', '': 'Documents' });
  assert.deepEqual(Object.keys(GROUP_LABELS).sort(), [...searchable, ''].sort());
});

test('sign out clears recent searches', () => {
  assert.match(readFileSync('src/lib/auth.ts', 'utf8'), new RegExp(`removeItem\\("${RECENT_KEY}"\\)`));
});

test('results link to their pages', () => {
  assert.equal(resultHref({ kind: 'goal', id: 'a b' }), '/timeline?goal=a%20b');
  assert.equal(resultHref({ kind: 'session', id: 's1' }), '/weekly-scheduler');
  assert.equal(resultHref({ kind: 'problem', id: 'two-sum' }), '/leetgrinder/problem/two-sum');
  assert.equal(resultHref({ kind: 'note', id: 'systems/raft notes' }), '/notes/systems/raft%20notes');
  assert.equal(resultHref({ kind: 'connection', id: 'u/1' }), '/connections?id=u%2F1');
  assert.equal(resultHref({ kind: 'personal', id: 'p1' }), '/journal?tab=personal&id=p1');
  assert.equal(resultHref({ kind: 'week', id: '2026/week-01' }), '/journal?id=2026%2Fweek-01');
});

test('the context line is the metadata for title matches, else the text around the match', () => {
  assert.equal(contextLine({ titleMatch: true, context: 'Active · Sep 14 → Dec 31', excerpt: '- [x] Step' }), 'Active · Sep 14 → Dec 31');
  assert.equal(contextLine({ titleMatch: false, context: 'Systems · 23 words', excerpt: '## Raft\n\n- [ ] **Leader** election `code` two-sum' }), 'Raft Leader election code two-sum');
  assert.equal(contextLine({ titleMatch: true, context: '', excerpt: 'Body text' }), 'Body text');
  assert.equal(contextLine({ titleMatch: false, context: '', excerpt: '…- Read DDIA\n- CKA lab…' }), '…Read DDIA CKA lab…');
  assert.equal(contextLine({ titleMatch: false, context: '#1 · Easy', excerpt: '' }), '#1 · Easy');
});

test('groups follow the saved module order, documents last, with totals from every kind', () => {
  const results = [
    { kind: 'document', module: '', id: 'd' },
    { kind: 'week', module: 'journal', id: 'w' },
    { kind: 'goal', module: 'timeline', id: 'g' },
    { kind: 'personal', module: 'journal', id: 'p' },
    { kind: 'note', module: 'notes', id: 'n' },
  ];
  const groups = groupResults(results, { document: 1, week: 7, personal: 2, goal: 1, note: 1 }, ['notes', 'journal']);
  assert.deepEqual(groups.map(g => g.label), ['Notes', 'Journal', 'Goals', 'Documents']);
  assert.deepEqual(groups[1].results.map((r: { id: string }) => r.id), ['w', 'p']);
  assert.equal(groups[1].total, 9);
  // Without totals the shown results count.
  assert.equal(groupResults(results)[0].total, 1);
});

test('recent searches put the latest first without repeats', () => {
  assert.deepEqual(addRecent(['raft', 'Grafana'], ' grafana '), ['grafana', 'raft']);
  assert.deepEqual(addRecent(['a'], '  '), ['a']);
  assert.equal(addRecent(['a', 'b', 'c', 'd', 'e'], 'f').length, RECENT_LIMIT);
});

test('book results draw their item type icon, the same marks as the Bookshelf', () => {
  assert.equal(resultIconPath({ kind: 'book', type: 'paper' }), ITEM_TYPE_PATHS.paper);
  assert.equal(resultIconPath({ kind: 'book', type: 'course' }), ITEM_TYPE_PATHS.course);
  assert.equal(resultIconPath({ kind: 'book', type: 'book' }), ITEM_TYPE_PATHS.book);
  // An unknown or missing type still reads as a book; other kinds have no icon.
  assert.equal(resultIconPath({ kind: 'book' }), ITEM_TYPE_PATHS.book);
  assert.equal(resultIconPath({ kind: 'note', type: 'paper' }), undefined);
  // The React icons draw from the same paths.
  const icons = readFileSync('src/components/ui/icons.tsx', 'utf8');
  for (const type of ['book', 'paper', 'course']) assert.match(icons, new RegExp(`ITEM_TYPE_PATHS\\.${type}\\b`));
});
