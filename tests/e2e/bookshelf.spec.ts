import { expect, test, type Page } from '@playwright/test';

// The Bookshelf page (docs/redesign/pages/bookshelf.md) against mocked items.
const common = { authors: [], category: 'Systems', status: 'backlog', featured: false, priority: 'medium', tags: [], body: '' };
const chapters = (done: number, total: number) => `## Chapters\n${Array.from({ length: total }, (_, i) => `- [${i < done ? 'x' : ' '}] Chapter ${i + 1}`).join('\n')}\n`;
const items = [
  { ...common, slug: 'books/ddia', title: 'Designing Data-Intensive Applications', authors: ['Martin Kleppmann'], priority: 'high', status: 'reading', featured: true,
    url: 'https://example.com/ddia', body: `${chapters(2, 3)}## Log\n### 2026-09-24\n- Read replication\n`, updatedAt: '2026-09-24T15:00:00Z' },
  { ...common, slug: 'books/beej', title: "Beej's Guide to Network Programming", authors: ['Brian Hall'], category: 'Networking', priority: 'high', status: 'reading',
    body: chapters(4, 10), updatedAt: '2026-11-20T15:00:00Z' },
  { ...common, slug: 'books/done', title: 'The Go Programming Language', authors: ['Donovan', 'Kernighan'], category: 'Languages', status: 'completed', body: chapters(3, 3), updatedAt: '2026-12-01T15:00:00Z' },
  { ...common, slug: 'papers/mapreduce', title: 'MapReduce', authors: ['Jeffrey Dean'], type: 'paper', priority: 'low' },
  // The most recently updated in-progress item sorts last by priority and by title, so only the
  // recency rule can put it on the card.
  { ...common, slug: 'books/zig', title: 'Zig in Depth', authors: ['Some Author'], category: 'Languages', priority: 'low', status: 'paused',
    body: chapters(1, 4), updatedAt: '2026-11-25T15:00:00Z' },
  { ...common, slug: 'courses/k8s', title: 'Kubernetes for SREs', authors: ['Some School'], type: 'course', category: 'Online Course', status: 'paused', body: chapters(0, 6) },
].map(item => ({ type: 'book', ...item }));

async function mockBooks(page: Page, entries: unknown[] = items) {
  await page.route('**/api/entries/book?**', route => route.fulfill({
    status: 200, contentType: 'application/json', body: JSON.stringify({ entries: entries.map(entry => ({ entry, revision: 'test' })), nextOffset: null }),
  }));
}

test('papers remain visible after the first page of bookshelf items', async ({ page }) => {
  const books = Array.from({ length: 100 }, (_, index) => ({ ...common, slug: `book-${index}`, title: `Book ${index}`, type: 'book' }));
  const paper = { ...common, slug: 'paper-after-books', title: 'A short article', type: 'paper', url: 'https://example.com/article' };
  await page.route('**/api/entries/book?**', route => {
    const offset = Number(new URL(route.request().url()).searchParams.get('offset'));
    const entries = offset === 0 ? books : offset === 100 ? [paper] : [];
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: entries.map(entry => ({ entry, revision: 'test' })), nextOffset: offset === 0 ? 100 : null }) });
  });

  await page.goto('/books');
  const main = page.locator('main');
  await expect(main.getByText('101 books, courses & papers')).toBeVisible();
  await main.getByRole('button', { name: /^Paper\s*1$/ }).click();
  const catalog = main.getByRole('list', { name: 'Catalog' });
  await expect(catalog.getByRole('button')).toHaveCount(1);
  await expect(catalog.getByRole('button', { name: /A short article/ })).toBeVisible();
});

test('desktop: catalog order, in-progress card, filters and search', async ({ page }) => {
  await mockBooks(page);
  await page.goto('/books');
  const main = page.locator('main');
  const catalog = main.getByRole('list', { name: 'Catalog' });

  // Sorted by priority, then status (reading first, completed last), then title.
  await expect(catalog.getByRole('button')).toHaveText([/Beej/, /Designing Data/, /Kubernetes/, /Go Programming/, /Zig in Depth/, /MapReduce/]);
  await expect(catalog.getByRole('button', { name: /Beej/ }).getByRole('img', { name: '4 of 10 chapters' })).toBeVisible();
  await expect(catalog.getByRole('button', { name: /MapReduce/ })).toContainText('No progress tracked');

  // In progress: started and unfinished, most recently updated first. Zig (Nov 25) beats Beej
  // (Nov 20) and DDIA (Sep 24) though both sort before it; the completed Go book (Dec 1) and the
  // unstarted course don't count.
  const card = main.getByRole('complementary', { name: 'In progress' });
  await expect(card.getByRole('button', { name: 'Zig in Depth' })).toBeVisible();
  await expect(card.getByRole('img', { name: '1 of 4 chapters' })).toBeVisible();
  await expect(card.getByText('Last updated Nov 25')).toBeVisible();
  await expect(card.getByText('3 chapters to go')).toBeVisible();
  await expect(card.getByRole('link', { name: 'Log reading' })).toHaveAttribute('href', '/manage/books?id=books%2Fzig');
  // No URL: DETAILS opens the item instead of OPEN.
  await expect(card.getByRole('link', { name: /Open/ })).toHaveCount(0);
  await card.getByRole('button', { name: 'Details' }).click();
  await expect(page.getByRole('dialog', { name: 'Zig in Depth' })).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(main.getByRole('link', { name: '+ Add item' })).toHaveAttribute('href', '/manage/books?new=1');
  await expect(main.getByRole('link', { name: 'Manage', exact: true })).toHaveAttribute('href', '/manage/books');

  // Shelf filter, type filter (toggles off again), search by author.
  await main.getByRole('button', { name: /^Networking\s*1$/ }).click();
  await expect(catalog.getByRole('button')).toHaveText([/Beej/]);
  await main.getByRole('button', { name: /^All\s*6$/ }).click();
  await main.getByRole('button', { name: /^Course\s*1$/ }).click();
  await expect(catalog.getByRole('button')).toHaveText([/Kubernetes/]);
  await main.getByRole('button', { name: /^Course\s*1$/ }).click();
  await main.getByRole('searchbox', { name: 'Search title or author' }).fill('kleppmann');
  await expect(catalog.getByRole('button')).toHaveText([/Designing Data/]);
  await main.getByRole('searchbox', { name: 'Search title or author' }).fill('nothing like this');
  await expect(main.getByText('No items match "nothing like this".')).toBeVisible();
});

test('a catalog row opens its details, and ?id= links open them directly', async ({ page }) => {
  await mockBooks(page);
  await page.goto('/books');
  const main = page.locator('main');
  await main.getByRole('list', { name: 'Catalog' }).getByRole('button', { name: /Designing Data/ }).click();
  const dialog = page.getByRole('dialog', { name: 'Designing Data-Intensive Applications' });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole('listitem').filter({ hasText: 'Chapter 1' })).toHaveText('Done: Chapter 1');
  await expect(dialog.getByRole('listitem').filter({ hasText: 'Chapter 3' })).toHaveText('Not done: Chapter 3');
  await expect(dialog.getByText('2 / 3 · 67%')).toBeVisible();
  await expect(dialog.getByText('Read replication')).toBeVisible();
  await expect(dialog.getByRole('link', { name: 'Edit details & progress' })).toHaveAttribute('href', '/manage/books?id=books%2Fddia');
  await expect(dialog.getByRole('link', { name: /Open source/ })).toHaveAttribute('href', 'https://example.com/ddia');
  await expect(page).toHaveURL(/\/books\?id=books%2Fddia$/);
  await dialog.getByRole('button', { name: 'Close' }).click();
  await expect(dialog).toBeHidden();
  // Focus goes back to the row that opened it.
  await expect(main.getByRole('list', { name: 'Catalog' }).getByRole('button', { name: /Designing Data/ })).toBeFocused();
  await expect(page).toHaveURL(/\/books$/);

  await page.goto('/books?id=papers%2Fmapreduce');
  await expect(page.getByRole('dialog', { name: 'MapReduce' })).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog', { name: 'MapReduce' })).toBeHidden();
  await expect(page).toHaveURL(/\/books$/);

  // A link to an item that's gone says so, and Dismiss clears it.
  await page.goto('/books?id=books%2Fgone');
  await expect(main.getByRole('alert')).toContainText('That item is no longer on the shelf.');
  await main.getByRole('button', { name: 'Dismiss' }).click();
  await expect(main.getByRole('alert')).toHaveCount(0);
  await expect(page).toHaveURL(/\/books$/);
});

test('an empty shelf offers to add the first item', async ({ page }) => {
  await mockBooks(page, []);
  await page.goto('/books');
  const main = page.locator('main');
  await expect(main.getByRole('link', { name: 'Add a book, course or paper' })).toHaveAttribute('href', '/manage/books?new=1');
  await expect(main.getByText('Nothing in progress.')).toBeVisible();
});

test.describe('phone', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('title, in-progress card, shelf and type chips, list rows', async ({ page }) => {
    await mockBooks(page);
    await page.goto('/books');
    const main = page.locator('main');
    await expect(main.getByText('6 items')).toBeVisible();
    await expect(main.getByRole('link', { name: 'Add item' })).toHaveAttribute('href', '/manage/books?new=1');
    const catalog = main.getByRole('list', { name: 'Catalog' });
    await expect(catalog.getByRole('button')).toHaveCount(6);
    const chips = main.getByRole('group', { name: 'Shelves and types' });
    await chips.getByRole('button', { name: 'Languages' }).click();
    await expect(catalog.getByRole('button')).toHaveText([/Go Programming/, /Zig in Depth/]);
    await chips.getByRole('button', { name: 'All' }).click();
    await chips.getByRole('button', { name: 'Paper' }).click();
    await expect(catalog.getByRole('button')).toHaveText([/MapReduce/]);
    // The in-progress card opens the item's details.
    await main.getByRole('button', { name: /Zig in Depth/ }).click();
    await expect(page.getByRole('dialog', { name: 'Zig in Depth' })).toBeVisible();
    await expect(page).toHaveURL(/\/books\?id=books%2Fzig$/);
  });
});
