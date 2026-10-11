import { expect, test } from '@playwright/test';

test('delete an accidental attempt from problem history', async ({ page }) => {
  const slug = `delete-check-${Date.now()}`;
  await page.goto(`/leetgrinder/problem/${slug}`);
  for (const notes of ['Keep this attempt', 'Accidental attempt']) {
    await page.getByRole('tab', { name: 'Record an attempt' }).click();
    const form = page.getByRole('tabpanel').locator('form.attempt-form');
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
  // With no attempts left the page opens on the form.
  await expect(page.getByRole('tab', { name: 'Record an attempt' })).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('tab', { name: /^Attempts/ }).click();
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

// The problem page splits attempts, the attempt form and the optimal panel
// into tabs instead of squeezing them side by side.
test('problem page sections are tabs', async ({ page }) => {
  const slug = `tabs-check-${Date.now()}`;
  await page.goto(`/leetgrinder/problem/${slug}`);

  const record = page.getByRole('tab', { name: 'Record an attempt' });
  const attempts = page.getByRole('tab', { name: /^Attempts/ });
  const optimal = page.getByRole('tab', { name: 'Optimal complexity' });
  // A problem with no attempts opens on the form, with the other panels hidden.
  await expect(record).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('tabpanel')).toHaveCount(1);
  await expect(page.getByRole('tabpanel')).toHaveAttribute('aria-labelledby', 'tab-record');
  await expect(page.getByText('No attempts recorded yet.')).toBeHidden();

  // Arrow keys move between tabs.
  await record.focus();
  await page.keyboard.press('ArrowRight');
  await expect(optimal).toBeFocused();
  await expect(optimal).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByText('appears after your first attempt')).toBeVisible();
  await page.keyboard.press('Home');
  await expect(attempts).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByText('No attempts recorded yet.')).toBeVisible();

  // After saving, the page opens on the attempts, which now count one.
  await record.click();
  const form = page.getByRole('tabpanel').locator('form.attempt-form');
  await form.getByLabel('Outcome').selectOption('unfinished');
  await form.getByLabel('Notes').fill('Tabbed attempt');
  await form.getByRole('button', { name: 'Save attempt' }).click();
  await expect(attempts).toHaveAttribute('aria-selected', 'true');
  await expect(attempts).toContainText('1');
  await expect(page.locator('article.attempt')).toContainText('Tabbed attempt');
  await expect(page.getByRole('heading', { name: 'Record an attempt' })).toBeHidden();

  // A link to the optimal panel opens its tab.
  await page.goto(`/leetgrinder/problem/${slug}#optimal`);
  await expect(optimal).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByRole('heading', { name: 'Optimal complexity' })).toBeVisible();
});
