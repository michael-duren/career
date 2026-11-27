import { expect, test, type Page } from '@playwright/test';

// The Notes page (docs/redesign/pages/notes.md) against mocked notes, plus a real note page.
const note = (id: string, title: string, topic: string, updatedAt: string, extra: Record<string, unknown> = {}) =>
  ({ id, title, topic, description: '', tags: [], body: '', updatedAt, summary: { wordCount: 40, todoCount: 0, todoDone: 0 }, ...extra });
const notes = [
  note('systems/raft', 'Raft consensus', 'Systems', '2026-11-25T15:00:00Z', { description: 'Leader election and log replication.', summary: { wordCount: 1228, todoCount: 3, todoDone: 2 } }),
  note('sre/slo', 'SLOs and error budgets', 'SRE', '2026-11-26T15:00:00Z', { tags: ['burn-rate'] }),
  note('db/mvcc', 'Postgres MVCC', 'Databases', '2026-11-20T15:00:00Z', { description: 'Snapshots, vacuum and bloat.' }),
  note('systems/ebpf', 'eBPF basics', 'Systems', '2026-11-10T15:00:00Z'),
];
const raftDetail = { ...notes[0], summary: undefined, body: '# Raft consensus\n\nTerms and **elections**.', todos: [
  { id: '11111111-1111-4111-8111-111111111111', title: 'Read the paper', done: true },
  { id: '22222222-2222-4222-8222-222222222222', title: 'Leader election', done: true },
  { id: '33333333-3333-4333-8333-333333333333', title: 'Log replication', done: false },
] };

/** Serves the mocked list and details; returns the POST bodies the page sends. */
async function mockNotes(page: Page) {
  const saves: { entry: Record<string, unknown>; revision: string | null }[] = [];
  await page.route('**/api/entries/note**', async route => {
    const request = route.request();
    if (request.method() === 'POST') {
      const body = request.postDataJSON();
      saves.push(body);
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entry: { ...body.entry, updatedAt: '2026-11-27T15:00:00Z' }, revision: 'saved' }) });
    }
    const id = new URL(request.url()).searchParams.get('id');
    if (id) {
      const found = id === raftDetail.id ? raftDetail : notes.find(n => n.id === id);
      return route.fulfill({ status: found ? 200 : 404, contentType: 'application/json', body: JSON.stringify(found ? { entry: found, revision: 'r1' } : { error: 'not found' }) });
    }
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: notes.map(entry => ({ entry, revision: 'r1' })), nextOffset: null }) });
  });
  return saves;
}

const cardTitles = (page: Page) => page.locator('main a[href^="/notes/"]:visible > span:nth-child(2)');

test('desktop: header counts, topics, card grid, single-topic filter and search', async ({ page }) => {
  await mockNotes(page);
  await page.goto('/notes');
  const main = page.locator('main');
  await expect(main.getByText('4 notes · 3 topics').filter({ visible: true })).toHaveCount(1);
  await expect(main.getByRole('heading', { level: 1, name: 'Notes' })).toBeVisible();

  const topics = main.getByRole('navigation', { name: 'Topics' });
  await expect(topics.getByRole('button')).toHaveText([/^All\s*4$/, /^Databases\s*1$/, /^SRE\s*1$/, /^Systems\s*2$/]);
  await expect(topics.getByRole('button', { name: /^All/ })).toHaveAttribute('aria-pressed', 'true');

  // Most recently updated first; each card links to its note page.
  await expect(cardTitles(page)).toHaveText(['SLOs and error budgets', 'Raft consensus', 'Postgres MVCC', 'eBPF basics']);
  const raft = main.getByRole('link', { name: /Raft consensus/ });
  await expect(raft).toHaveAttribute('href', '/notes/systems/raft');
  await expect(raft).toContainText('Systems');
  await expect(raft).toContainText('Nov 25');
  await expect(raft).toContainText('1228 words');
  await expect(raft).toContainText('Leader election and log replication.');
  await expect(raft).toContainText('2/3');
  await expect(raft.getByRole('progressbar', { name: 'Raft consensus todos done' })).toHaveAttribute('aria-valuenow', '67');
  // No todos, no progress bar.
  await expect(main.getByRole('link', { name: /Postgres MVCC/ }).getByRole('progressbar')).toHaveCount(0);

  // One topic at a time.
  await topics.getByRole('button', { name: /^Systems/ }).click();
  await expect(topics.getByRole('button', { name: /^Systems/ })).toHaveAttribute('aria-pressed', 'true');
  await expect(topics.getByRole('button', { name: /^All/ })).toHaveAttribute('aria-pressed', 'false');
  await expect(cardTitles(page)).toHaveText(['Raft consensus', 'eBPF basics']);
  await expect(main.getByRole('heading', { name: 'RECENTLY UPDATED' }).locator('..')).toContainText('Systems');
  await topics.getByRole('button', { name: /^SRE/ }).click();
  await expect(cardTitles(page)).toHaveText(['SLOs and error budgets']);

  // Search covers titles, summaries, topics and tags within the chosen topic.
  await topics.getByRole('button', { name: /^All/ }).click();
  const search = main.getByRole('searchbox', { name: 'Search notes' });
  await search.fill('vacuum');
  await expect(cardTitles(page)).toHaveText(['Postgres MVCC']);
  await search.fill('burn-rate');
  await expect(cardTitles(page)).toHaveText(['SLOs and error budgets']);
  await search.fill('nothing like this');
  await expect(cardTitles(page)).toHaveCount(0);
  await expect(main.getByText('No notes match. Try another topic or fewer words.').filter({ visible: true })).toHaveCount(1);
});

test('a card opens the note in place; the reader keeps the editor and todos', async ({ page }) => {
  const saves = await mockNotes(page);
  await page.goto('/notes');
  const main = page.locator('main');
  await main.getByRole('link', { name: /Raft consensus/ }).click();

  await expect(main.getByRole('button', { name: '← Back to notes' })).toBeFocused();
  expect(new URL(page.url()).pathname).toBe('/notes');
  await expect(main.getByRole('heading', { level: 1, name: 'Raft consensus' })).toHaveCount(1);
  // The body's leading "# Raft consensus" is not repeated under the title.
  await expect(main.locator('.prose-ink strong')).toHaveText('elections');
  await expect(main.locator('.prose-ink h1')).toHaveCount(0);

  // Checking a todo while reading saves it straight away with the loaded revision.
  await main.getByRole('checkbox', { name: 'Mark Log replication done' }).click();
  await expect(main.getByRole('status')).toHaveText('Saved.');
  expect(saves).toHaveLength(1);
  expect(saves[0].revision).toBe('r1');
  expect((saves[0].entry.todos as { done: boolean }[]).map(todo => todo.done)).toEqual([true, true, true]);

  // Edit and save the summary.
  await main.getByRole('button', { name: 'Edit' }).click();
  await main.getByRole('textbox', { name: 'Summary' }).fill('Terms, elections and logs.');
  await main.getByRole('button', { name: 'Save' }).click();
  await expect(main.getByRole('paragraph').filter({ hasText: 'Terms, elections and logs.' })).toBeVisible();
  expect(saves[1]).toMatchObject({ revision: 'saved', entry: { id: 'systems/raft', topic: 'Systems', description: 'Terms, elections and logs.' } });

  await main.getByRole('button', { name: '← Back to notes' }).click();
  await expect(main.getByRole('heading', { level: 1, name: 'Notes' })).toBeVisible();
  await expect(main.getByRole('link', { name: /Raft consensus/ })).toBeFocused();
});

test('+ New note starts in the chosen topic and saves a new note', async ({ page }) => {
  const saves = await mockNotes(page);
  await page.goto('/notes');
  const main = page.locator('main');
  await main.getByRole('navigation', { name: 'Topics' }).getByRole('button', { name: /^Databases/ }).click();
  await main.getByRole('button', { name: '+ New note' }).click();

  await expect(main.getByRole('combobox', { name: 'Topic' })).toHaveValue('Databases');
  await main.getByRole('textbox', { name: 'Title' }).fill('Index types');
  await main.getByRole('textbox', { name: 'Markdown' }).fill('B-tree, hash, GIN.');
  await main.getByRole('button', { name: 'Save' }).click();

  await expect(main.getByRole('heading', { level: 1, name: 'Index types' })).toBeVisible();
  expect(saves).toHaveLength(1);
  expect(saves[0]).toMatchObject({ revision: null, entry: { title: 'Index types', topic: 'Databases', body: 'B-tree, hash, GIN.', tags: [] } });
  expect(saves[0].entry.id).toMatch(/^[0-9a-f-]{36}$/);
});

test.describe('phone', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('title with +, search, topic chips and note rows', async ({ page }) => {
    await mockNotes(page);
    await page.goto('/notes');
    const main = page.locator('main');
    await expect(main.getByRole('heading', { level: 1, name: 'Notes' })).toBeVisible();
    await expect(main.getByRole('navigation', { name: 'Topics' })).toBeHidden();
    const chips = main.getByRole('group', { name: 'Topics' });
    await expect(chips.getByRole('button')).toHaveText([/^All 4$/, /^Databases 1$/, /^SRE 1$/, /^Systems 2$/]);
    await expect(cardTitles(page)).toHaveText(['SLOs and error budgets', 'Raft consensus', 'Postgres MVCC', 'eBPF basics']);

    await chips.getByRole('button', { name: /^Systems/ }).click();
    await expect(chips.getByRole('button', { name: /^Systems/ })).toHaveAttribute('aria-pressed', 'true');
    await expect(cardTitles(page)).toHaveText(['Raft consensus', 'eBPF basics']);
    await main.getByRole('searchbox', { name: 'Search notes' }).fill('ebpf');
    await expect(cardTitles(page)).toHaveText(['eBPF basics']);

    await main.getByRole('button', { name: 'New note' }).click();
    await expect(main.getByRole('combobox', { name: 'Topic' })).toHaveValue('Systems');
    const width = await page.evaluate(() => document.documentElement.scrollWidth);
    expect(width).toBeLessThanOrEqual(390);
  });
});

test('a note page opens the reader and Back shows that note’s topic', async ({ page, baseURL }) => {
  // Unique per run so retries against a reused workspace don't collide.
  const id = `e2e/notes-page-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const created = await page.request.post('/api/entries/note', { headers: { Origin: baseURL! }, data: { revision: null, entry: { id, title: 'Playwright note page', topic: 'E2E Topic', description: '', tags: [], body: 'Read me.' } } });
  expect(created.status()).toBe(200);
  const { revision } = await created.json();
  try {
    await page.goto(`/notes/${id}`);
    const main = page.locator('main');
    await expect(main.getByRole('heading', { level: 1, name: 'Playwright note page' })).toBeVisible();
    await expect(main.getByText('E2E Topic', { exact: true }).filter({ visible: true })).toHaveCount(1);
    await main.getByRole('button', { name: '← Back to notes' }).click();
    await expect(page).toHaveURL(/\/notes$/);
    const topics = main.getByRole('navigation', { name: 'Topics' });
    await expect(topics.getByRole('button', { name: /^E2E Topic/ })).toHaveAttribute('aria-pressed', 'true');
    await expect(cardTitles(page)).toHaveText(['Playwright note page']);
  } finally {
    await page.request.delete('/api/entries/note', { headers: { Origin: baseURL! }, data: { id, revision } });
  }
});
