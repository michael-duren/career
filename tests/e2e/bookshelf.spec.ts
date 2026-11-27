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
  await expect(catalog.getByRole('button')).toHaveText([/Beej/, /Designing Data/, /Kubernetes/, /Go Programming/, /MapReduce/]);
  await expect(catalog.getByRole('button', { name: /Beej/ }).getByRole('img', { name: '4 of 10 chapters' })).toBeVisible();
  await expect(catalog.getByRole('button', { name: /MapReduce/ })).toContainText('No progress tracked');

  // In progress: started and unfinished, most recently updated first (Beej, not the featured DDIA
  // and not the newer but completed Go book).
  const card = main.getByRole('complementary', { name: 'In progress' });
  await expect(card.getByRole('button', { name: "Beej's Guide to Network Programming" })).toBeVisible();
  await expect(card.getByRole('img', { name: '4 of 10 chapters' })).toBeVisible();
  await expect(card.getByText('Last updated Nov 20')).toBeVisible();
  await expect(card.getByText('6 chapters to go')).toBeVisible();
  await expect(card.getByRole('link', { name: 'Log reading' })).toHaveAttribute('href', '/manage/books?id=books%2Fbeej');
  await expect(main.getByRole('link', { name: '+ Add item' })).toHaveAttribute('href', '/manage/books?new=1');
  await expect(main.getByRole('link', { name: 'Manage', exact: true })).toHaveAttribute('href', '/manage/books');

  // Shelf filter, type filter (toggles off again), search by author.
  await main.getByRole('button', { name: /^Networking\s*1$/ }).click();
  await expect(catalog.getByRole('button')).toHaveText([/Beej/]);
  await main.getByRole('button', { name: /^All\s*5$/ }).click();
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
  await expect(dialog.getByText('Done: Chapter 1')).toBeAttached();
  await expect(dialog.getByText('Read replication')).toBeVisible();
  await expect(dialog.getByRole('link', { name: 'Edit details & progress' })).toHaveAttribute('href', '/manage/books?id=books%2Fddia');
  await expect(dialog.getByRole('link', { name: /Open source/ })).toHaveAttribute('href', 'https://example.com/ddia');
  await expect(page).toHaveURL(/\/books\?id=books%2Fddia$/);
  await dialog.getByRole('button', { name: 'Close' }).click();
  await expect(dialog).toBeHidden();
  await expect(page).toHaveURL(/\/books$/);

  await page.goto('/books?id=papers%2Fmapreduce');
  await expect(page.getByRole('dialog', { name: 'MapReduce' })).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog', { name: 'MapReduce' })).toBeHidden();
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
    await expect(main.getByText('5 items')).toBeVisible();
    await expect(main.getByRole('link', { name: 'Add item' })).toHaveAttribute('href', '/manage/books?new=1');
    const catalog = main.getByRole('list', { name: 'Catalog' });
    await expect(catalog.getByRole('button')).toHaveCount(5);
    const chips = main.getByRole('group', { name: 'Shelves and types' });
    await chips.getByRole('button', { name: 'Languages' }).click();
    await expect(catalog.getByRole('button')).toHaveText([/Go Programming/]);
    await chips.getByRole('button', { name: 'All' }).click();
    await chips.getByRole('button', { name: 'Paper' }).click();
    await expect(catalog.getByRole('button')).toHaveText([/MapReduce/]);
    // The in-progress card opens the item's details.
    await main.getByRole('button', { name: /Beej's Guide/ }).click();
    await expect(page.getByRole('dialog', { name: "Beej's Guide to Network Programming" })).toBeVisible();
  });
});
