import assert from 'node:assert/strict';
import test from 'node:test';
import { appendDailyEntry, localDate, newWeek } from '../src/lib/workspace.ts';

test('quick entries preserve existing day content, later days and weekly sections', () => {
  const source = '## What I did\n\n### Monday — Sep 7\nOriginal Monday\n\n### Tuesday — Sep 8\nOriginal Tuesday\n\n## Blockers\nA blocker\n\n## Notes\nA note';
  const result = appendDailyEntry(source, '2026-09-07', '**New reflection**');
  assert.match(result, /Original Monday[\s\S]*\*\*New reflection\*\*[\s\S]*### Tuesday/);
  assert.equal(result.match(/### Monday/g)?.length, 1);
  assert.ok(result.endsWith('## Blockers\nA blocker\n\n## Notes\nA note'));
});

test('creates a missing weekday inside What I did, before Blockers', () => {
  const result = appendDailyEntry('## What I did\n\n## Blockers\nKeep this', '2026-09-11', '- Learned something');
  assert.match(result, /### Friday — Sep 11\n- Learned something\n\n## Blockers/);
});

test('preserves unstructured journals when adding a daily section', () => {
  const original = '# Older journal\nSome text\n```js\nconst x = 1;\n```';
  const result = appendDailyEntry(original, '2026-09-11', 'Today');
  assert.ok(result.endsWith(original));
  assert.ok(result.startsWith('## What I did\n\n### Friday — Sep 11\nToday'));
});

test('week calculation follows the existing April 20 plan start, including year boundaries', () => {
  assert.equal(newWeek('2026-04-20').week, 1);
  assert.equal(newWeek('2026-09-11').dates, '2026-09-07 to 2026-09-13');
  assert.equal(newWeek('2026-09-13').week, 21);
  assert.equal(newWeek('2026-09-14').week, 22);
  assert.equal(newWeek('2027-01-01').dates, '2026-12-28 to 2027-01-03');
});

test('local dates use the calendar date, not a UTC truncation', () => {
  const date = new Date(2026, 8, 11, 23, 59);
  assert.equal(localDate(date), '2026-09-11');
});

test('v1 migration retains saved content and intentionally empty collections', async () => {
  const { migrateWorkspace } = await import('../src/lib/workspace.ts');
  const existing = { version: 1 as const, notes: [], weeks: [{ ...newWeek('2026-09-11'), body: 'Saved in the UI', hours: { ostep: 4 } }] };
  const seeds = {
    version: 2 as const,
    notes: [{ id: 'old-file', title: 'Do not resurrect', topic: 'General', description: '', tags: [], body: '' }],
    weeks: [], books: [], companies: [],
    documents: [{ id: 'index', title: 'Home', description: '', tags: [], body: 'Home page' }],
  };
  const migrated = migrateWorkspace(existing, seeds);
  assert.deepEqual(migrated.notes, []);
  assert.deepEqual(migrated.weeks, existing.weeks);
  assert.deepEqual(migrated.documents, seeds.documents);
  assert.equal(migrated.version, 2);
  assert.equal(existing.version, 1);
  assert.equal(migrateWorkspace(migrated, { ...seeds, documents: [] }), migrated);
});

test('collection edits preserve all other collections and deletions stay deleted', async () => {
  const { migrateWorkspace, updateWorkspaceEntry } = await import('../src/lib/workspace.ts');
  const data = { version: 2 as const, notes: [], weeks: [], books: [], companies: [], documents: [{ id: 'reference', title: 'Reference', description: '', tags: [], body: 'Original' }] };
  const changed = updateWorkspaceEntry(data, 'document', 'reference', { ...data.documents[0], body: 'Edited' });
  assert.equal(changed.documents[0].body, 'Edited');
  assert.equal(changed.notes, data.notes);
  assert.equal(data.documents[0].body, 'Original');
  const removed = updateWorkspaceEntry(changed, 'document', 'reference');
  assert.deepEqual(migrateWorkspace(removed, data).documents, []);
});

test('v2 does not import bio from files or overwrite an existing saved note', async () => {
  const { migrateWorkspace } = await import('../src/lib/workspace.ts');
  const bio = { id: 'bio', title: 'Bio', topic: 'General', description: '', tags: [], body: 'File bio' };
  const seed = { version: 2 as const, notes: [bio], weeks: [], books: [], companies: [], documents: [] };
  assert.deepEqual(migrateWorkspace({ version: 1, notes: [], weeks: [] }, seed).notes, []);
  const savedBio = { ...bio, body: 'Saved bio' };
  assert.deepEqual(migrateWorkspace({ version: 1, notes: [savedBio], weeks: [] }, seed).notes, [savedBio]);
});

test('catalog fixes preserve notes and map existing OSTEP progress', async () => {
  const { updateBookCatalog } = await import('../src/lib/workspace.ts');
  const base = { version: 2 as const, notes: [], weeks: [], companies: [], documents: [] };
  const oldTasks = Array.from({ length: 14 }, (_, index) => `- [${index === 2 ? 'x' : ' '}] ${index === 0 ? 'Intro: Processes & the Process API' : `Old group ${index + 1}`}`).join('\n');
  const chapters = Array.from({ length: 57 }, (_, index) => `- [ ] ${index + 1}. Chapter ${index + 1}`).join('\n');
  const existing = { ...base, books: [{ slug: 'ostep', title: 'OSTEP', authors: [], category: 'OS' as const, type: 'book' as const, status: 'reading' as const, featured: true, priority: 'high' as const, tags: [], body: `## Why\nMy reason\n\n## Chapters\n${oldTasks}\n\n## Log\n### 2026-01-01\nMy note` }] };
  const seed = { ...base, catalogVersion: 1 as const, books: [{ ...existing.books[0], cover: 'https://example.com/cover.jpg', body: `## Why\nSeed reason\n\n## Chapters\n${chapters}\n\n## Log\n` }] };
  const updated = updateBookCatalog(existing, seed);
  assert.equal(updated.catalogVersion, 1);
  assert.equal(updated.books[0].body.match(/^- \[[ xX]\]/gm)?.length, 57);
  assert.match(updated.books[0].body, /- \[x\] 7\. Chapter 7/);
  assert.match(updated.books[0].body, /My reason/);
  assert.match(updated.books[0].body, /My note/);
  assert.equal(updated.books[0].cover, 'https://example.com/cover.jpg');
  assert.deepEqual(updated.books[0].authors, []);
  assert.equal(updateBookCatalog(updated, { ...seed, books: [] }), updated);
});

test('catalog fixes fill missing authors without overwriting user metadata', async () => {
  const { updateBookCatalog } = await import('../src/lib/workspace.ts');
  const base = { version: 2 as const, notes: [], weeks: [], companies: [], documents: [] };
  const template = { slug: 'database-internals', title: 'Database Internals', category: 'Distributed Systems' as const, type: 'book' as const, status: 'backlog' as const, featured: false, priority: 'high' as const, tags: [], body: '## Log\nMy note' };
  const existing = { ...base, books: [{ ...template, authors: [] }] };
  const seed = { ...base, catalogVersion: 1 as const, books: [{ ...template, authors: ['Alex Petrov'], cover: 'https://example.com/database.jpg' }] };
  const updated = updateBookCatalog(existing, seed);
  assert.deepEqual(updated.books[0].authors, ['Alex Petrov']);
  assert.equal(updated.books[0].cover, 'https://example.com/database.jpg');
  assert.match(updated.books[0].body, /My note/);
  const custom = updateBookCatalog({ ...existing, books: [{ ...template, authors: ['Custom Author'] }] }, seed);
  assert.deepEqual(custom.books[0].authors, ['Custom Author']);
});
