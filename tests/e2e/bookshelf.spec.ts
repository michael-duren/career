import { expect, test } from '@playwright/test';

test('papers remain visible after the first page of bookshelf items', async ({ page }) => {
  const common = { authors: [], category: 'Systems', status: 'backlog', featured: false, priority: 'medium', tags: [], body: '' };
  const books = Array.from({ length: 100 }, (_, index) => ({ ...common, slug: `book-${index}`, title: `Book ${index}`, type: 'book' }));
  const paper = { ...common, slug: 'paper-after-books', title: 'A short article', type: 'paper', url: 'https://example.com/article' };
  await page.route('**/api/entries/book?**', route => {
    const offset = Number(new URL(route.request().url()).searchParams.get('offset'));
    const entries = offset === 0 ? books : offset === 100 ? [paper] : [];
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: entries.map(entry => ({ entry, revision: 'test' })), nextOffset: offset === 0 ? 100 : null }) });
  });

  await page.goto('/books');
  await expect(page.getByRole('tab', { name: 'Papers 1' })).toBeVisible();
  await page.getByRole('tab', { name: 'Papers 1' }).click();
  await expect(page.getByRole('button', { name: /A short article/ })).toBeVisible();
  await expect(page.getByText('No papers yet')).toHaveCount(0);
});
