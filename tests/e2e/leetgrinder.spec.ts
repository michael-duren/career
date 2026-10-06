import { expect, test } from '@playwright/test';

test('delete an accidental attempt from problem history', async ({ page }) => {
  const slug = `delete-check-${Date.now()}`;
  await page.goto(`/leetgrinder/problem/${slug}`);
  for (const notes of ['Keep this attempt', 'Accidental attempt']) {
    const form = page.locator('form.attempt-form').first();
    await form.getByLabel('Outcome').selectOption('unfinished');
    await form.getByLabel('Notes').fill(notes);
    await form.getByRole('button', { name: 'Save attempt' }).click();
    await expect(page.locator('article.attempt').filter({ hasText: notes })).toBeVisible();
  }

  const accidental = page.locator('article.attempt').filter({ hasText: 'Accidental attempt' });
  await expect(accidental.getByRole('button', { name: 'Delete attempt', exact: true })).toBeHidden();
  await accidental.getByText('Delete this attempt', { exact: true }).click();
  await expect(accidental.getByText(/This cannot be undone/)).toBeVisible();
  await accidental.getByRole('button', { name: 'Delete attempt', exact: true }).click();
  await expect(page.locator('article.attempt')).toHaveCount(1);
  await expect(page.locator('article.attempt')).toContainText('Keep this attempt');
  await page.reload();
  await expect(page.locator('article.attempt')).toHaveCount(1);

  const remaining = page.locator('article.attempt');
  await remaining.getByText('Delete this attempt', { exact: true }).click();
  await remaining.getByRole('button', { name: 'Delete attempt', exact: true }).click();
  await expect(page.getByText('No attempts recorded yet.')).toBeVisible();
});

// Layout guards for the Leetgrinder attempt form and problems table: the
// approach radios stay radio-sized, and a struggle's flag fits the table
// without pushing columns out of view.
test('attempt form and problems table keep their layout with flags', async ({ page }) => {
  const slug = `layout-check-${Date.now()}`;
  await page.goto(`/leetgrinder/problem/${slug}`);

  const form = page.locator('form.attempt-form').first();
  const radio = form.getByRole('radio', { name: 'Took a simpler approach for time' });
  const box = await radio.boundingBox();
  expect(box && box.width).toBeLessThan(30);

  await form.getByLabel('Outcome').selectOption('struggled');
  await form.locator('select[name="timeComplexity"]').selectOption('O(n)');
  await form.locator('select[name="spaceComplexity"]').selectOption('O(1)');
  await radio.check();
  await form.getByLabel('I want to review this problem again soon').check();
  await form.getByRole('button', { name: 'Save attempt' }).click();
  await expect(page.getByText('Took a simpler approach for time · Marked for review')).toBeVisible();

  await page.goto(`/leetgrinder/problems?q=${slug}`);
  const row = page.getByRole('row', { name: new RegExp(slug) });
  await expect(row.getByText('Flagged: struggled')).toBeVisible();
  // Every column, the LeetCode link included, fits without scrolling.
  const scroll = page.locator('.problem-table');
  const overflow = await scroll.evaluate(el => el.scrollWidth - el.clientWidth);
  expect(overflow).toBeLessThanOrEqual(0);
  await expect(row.getByRole('link', { name: /on LeetCode/ })).toBeInViewport();
});
