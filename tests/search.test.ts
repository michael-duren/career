import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createSearch, searchEntries } from '../src/lib/search.ts';

test('finds typos in titles and text deep within content', () => {
  const search = createSearch([
    { title: 'Database notes', kind: 'Note', href: '/notes/database', body: 'intro '.repeat(200) + 'transaction isolation' },
    { title: 'Bookshelf', kind: 'Navigation', href: '/books', body: '' },
  ]);
  assert.equal(search.search('databse')[0].item.href, '/notes/database');
  assert.equal(search.search('isolaton')[0].item.href, '/notes/database');
  assert.equal(search.search('zzqqxxww').length, 0);
});

test('indexes live workspace collections with usable destinations', () => {
  const entries = searchEntries({ version: 2,
    notes: [{ id: 'new/note', title: 'Fresh note', topic: 'topic', description: '', tags: [], body: 'saved text' }],
    documents: [{ id: 'new/page', title: 'Fresh page', description: '', tags: [], body: '' }],
    weeks: [{ slug: '2026/week-01', week: 1, year: 2026, dates: '2026-01-01 to 2026-01-07', hours: {}, tags: [], body: 'journal text' }],
    books: [], companies: [], goals: [],
  });
  assert.equal(entries.find(e => e.title === 'Fresh note')?.href, '/notes/new/note');
  assert.equal(entries.find(e => e.title === 'Fresh page')?.href, '/documents/new/page');
  assert.equal(entries.find(e => e.kind === 'Work journal')?.href, '/journal?id=2026%2Fweek-01');
  assert.ok(entries.some(e => e.href === '/timeline'));
});
