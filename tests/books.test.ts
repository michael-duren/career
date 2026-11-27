import assert from 'node:assert/strict';
import test from 'node:test';
import { buildShelf, inProgress, lastUpdated, matchesBookSearch, progressMarks, shelves, sortCatalog, unitLabel, type RawBook } from '../src/lib/books.ts';

const raw = (slug: string, extra: Partial<RawBook> = {}): RawBook => ({
  slug, title: slug, authors: [], category: 'Systems', type: 'book', status: 'backlog', featured: false, priority: 'medium', tags: [], body: '', ...extra,
});
const chapters = (done: number, total: number) => `## Chapters\n${Array.from({ length: total }, (_, i) => `- [${i < done ? 'x' : ' '}] Chapter ${i + 1}`).join('\n')}\n`;

test('in progress means started and unfinished, most recently updated first', () => {
  const { all } = buildShelf([
    raw('not-started', { status: 'reading', body: chapters(0, 5) }),
    raw('finished-checklist', { status: 'reading', body: chapters(5, 5) }),
    raw('completed-status', { status: 'completed', body: chapters(2, 5) }),
    raw('older', { status: 'paused', body: chapters(1, 5), updatedAt: '2026-10-01T12:00:00Z' }),
    raw('newer', { status: 'reading', body: chapters(3, 5), updatedAt: '2026-11-20T12:00:00Z' }),
    raw('log-only', { status: 'reading', body: `${chapters(2, 4)}## Log\n### 2026-11-01\n- Read\n` }),
    raw('pages', { status: 'reading', progress: { unit: 'page', total: 300, completed: 40 } }),
    raw('no-units', { status: 'reading' }),
  ]);
  assert.deepEqual(inProgress(all).map(b => b.slug), ['newer', 'log-only', 'older', 'pages']);
});

test('last updated prefers the saved timestamp, then the newest log date', () => {
  const log = '## Log\n### 2026-09-24\n- Notes\n### 2026-09-01\n- Older\n';
  const { all } = buildShelf([raw('a', { body: log }), raw('b', { updatedAt: '2026-11-02T00:00:00Z' }), raw('c'), raw('d', { body: log, updatedAt: '2026-08-01T00:00:00Z' })]);
  assert.deepEqual(all.map(b => [b.slug, lastUpdated(b)]), [['a', '2026-09-24'], ['b', '2026-11-02T00:00:00Z'], ['c', ''], ['d', '2026-08-01T00:00:00Z']]);
});

test('catalog sorts by priority, then status, then title', () => {
  const { all } = buildShelf([
    raw('Low reading', { priority: 'low', status: 'reading' }),
    raw('High done', { priority: 'high', status: 'completed' }),
    raw('High backlog', { priority: 'high', status: 'backlog' }),
    raw('Medium b', { priority: 'medium' }),
    raw('Medium a', { priority: 'medium' }),
    raw('High reading', { priority: 'high', status: 'reading' }),
  ]);
  assert.deepEqual(sortCatalog(all).map(b => b.slug), ['High reading', 'High backlog', 'High done', 'Medium a', 'Medium b', 'Low reading']);
});

test('search matches title or author, case-insensitively', () => {
  const book = { title: 'Designing Data-Intensive Applications', authors: ['Martin Kleppmann'] };
  assert.equal(matchesBookSearch(book, ''), true);
  assert.equal(matchesBookSearch(book, '  data-INTENSIVE '), true);
  assert.equal(matchesBookSearch(book, 'kleppmann'), true);
  assert.equal(matchesBookSearch(book, 'petrov'), false);
});

test('shelves lists used categories alphabetically with counts', () => {
  assert.deepEqual(shelves([{ category: 'Systems' }, { category: 'Career' }, { category: 'Systems' }]), [{ category: 'Career', count: 1 }, { category: 'Systems', count: 2 }]);
});

test('progress marks draw one per unit, and scale long items without hiding progress', () => {
  assert.deepEqual(progressMarks(4, 10, 10), { count: 10, filled: 4 });
  assert.deepEqual(progressMarks(2, 3, 10), { count: 3, filled: 2 });
  assert.deepEqual(progressMarks(0, 0, 10), { count: 0, filled: 0 });
  assert.deepEqual(progressMarks(150, 300, 10), { count: 10, filled: 5 });
  assert.deepEqual(progressMarks(1, 300, 10), { count: 10, filled: 1 });
  assert.deepEqual(progressMarks(299, 300, 10), { count: 10, filled: 9 });
  assert.deepEqual(progressMarks(300, 300, 10), { count: 10, filled: 10 });
  assert.deepEqual(progressMarks(12, 10, 10), { count: 10, filled: 10 });
  assert.deepEqual(progressMarks(0, 6, 10), { count: 6, filled: 0 });
  assert.deepEqual(progressMarks(0, 300, 12), { count: 12, filled: 0 });
  assert.deepEqual(progressMarks(11, 12, 12), { count: 12, filled: 11 });
  assert.deepEqual(progressMarks(100, 300, 12), { count: 12, filled: 4 });
});

test('unit labels pluralize', () => {
  assert.equal(unitLabel('chapter', 1), 'chapter');
  assert.equal(unitLabel('page', 0), 'pages');
  assert.equal(unitLabel('module', 6), 'modules');
});
