import { test, expect, type Page } from '@playwright/test';

// The app header (docs/redesign/shell.md): Home plus every enabled module in the user's
// order, the current page marked, overflow in MORE +, search, settings and a 12-hour clock.
// The same markup is rendered by Astro (src/components/AppHeader.astro) and Go (internal/shell).

// Wide enough for every link next to MORE + (which always shows: it holds Sign out).
test.use({ viewport: { width: 1920, height: 900 } });

const menu = (page: Page) => page.getByRole('navigation', { name: 'Main' });
const inlineLinks = (page: Page) => page.locator('[data-app-menu] > li[data-module-id]:visible > a');

async function patchSettings(page: Page, patch: object) {
  return page.evaluate(async body => {
    const response = await fetch('/api/settings', { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    if (!response.ok) throw new Error(`PATCH /api/settings: ${response.status}`);
    return response.json();
  }, patch);
}

test('lists Home and every module with the current page marked', async ({ page }) => {
  await page.goto('/books');
  await expect(inlineLinks(page)).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'LEET', 'COMPANIES', 'PEOPLE', 'BOOKS', 'JOURNAL', 'NOTES', 'AUDIO', 'AGENTS']);
  await expect(menu(page).locator('[aria-current="page"]')).toHaveText(['BOOKS']);
  await expect(page.getByRole('link', { name: 'Settings' })).toHaveAttribute('href', '/settings/modules');
  await expect(page.locator('[data-app-clock]')).toHaveText(/^(1[0-2]|[1-9]):[0-5]\d (AM|PM)$/);
});

test('marks the owning module on its secondary routes', async ({ page }) => {
  await page.goto('/manage/companies');
  await expect(menu(page).locator('[aria-current="page"]')).toHaveText(['COMPANIES']);
});

test('the Go-rendered Leetgrinder pages get the same header with LEET current', async ({ page }) => {
  for (const path of ['/leetgrinder', '/leetgrinder/problems']) {
    await page.goto(path);
    await expect(menu(page).locator('[aria-current="page"]')).toHaveText(['LEET']);
    await expect(inlineLinks(page).first()).toHaveText('HOME');
    await expect(page.locator('[data-app-clock]')).toHaveText(/ (AM|PM)$/);
  }
});

test('the menu follows module changes without a reload, in this tab and others', async ({ page, context }) => {
  await page.goto('/books');
  await expect(inlineLinks(page)).toHaveCount(11);
  await page.evaluate(() => { (window as unknown as { sameDocument: boolean }).sameDocument = true; });
  try {
    // This tab: saveSettings() fires career:settings on the window.
    const saved = await patchSettings(page, { modules: { notes: false, timeline: false } });
    await page.evaluate(settings => window.dispatchEvent(new CustomEvent('career:settings', { detail: settings })), saved);
    // Turning Timeline off also hides Schedule and Agents (they depend on it).
    await expect(inlineLinks(page)).toHaveText(['HOME', 'LEET', 'COMPANIES', 'PEOPLE', 'BOOKS', 'JOURNAL', 'AUDIO']);

    // Another tab: saveSettings() posts on the career-settings BroadcastChannel.
    const other = await context.newPage();
    await other.goto('/books');
    const reordered = await patchSettings(other, { modules: { notes: true, timeline: true }, moduleOrder: ['notes', 'bookshelf'] });
    await other.evaluate(settings => { new BroadcastChannel('career-settings').postMessage(settings); }, reordered);
    await expect(inlineLinks(page)).toHaveText(['HOME', 'NOTES', 'BOOKS', 'TIMELINE', 'SCHEDULE', 'LEET', 'COMPANIES', 'PEOPLE', 'JOURNAL', 'AUDIO', 'AGENTS']);
    expect(await page.evaluate(() => (window as unknown as { sameDocument?: boolean }).sameDocument)).toBe(true);
    await other.close();
  } finally {
    await patchSettings(page, { modules: { notes: true, timeline: true }, moduleOrder: ['timeline', 'schedule', 'leetgrinder', 'companies', 'people', 'bookshelf', 'journal', 'notes', 'audio', 'agents'] });
  }
});

test('links that do not fit move into MORE +, which opens, navigates and closes', async ({ page }) => {
  await page.setViewportSize({ width: 800, height: 800 });
  await page.goto('/leetgrinder');
  const more = page.getByRole('button', { name: 'MORE +' });
  const moreMenu = page.locator('#app-more-menu');
  await expect(moreMenu.locator('[data-module-id]')).not.toHaveCount(0);
  const inline = await inlineLinks(page).count();
  expect(inline).toBeGreaterThan(0);
  expect(inline).toBeLessThan(11);
  // Everything still fits on one row without horizontal scroll.
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  // LEET is in the overflow here, so MORE + carries the current-page underline.
  await expect(page.locator('[data-app-more]')).toHaveClass(/is-current/);

  await more.click();
  await expect(more).toHaveAttribute('aria-expanded', 'true');
  await expect(moreMenu).toBeVisible();
  await expect(moreMenu.getByRole('button', { name: 'SIGN OUT' })).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(moreMenu).toBeHidden();
  await expect(more).toBeFocused();

  await more.click();
  await moreMenu.getByRole('link', { name: 'AGENTS' }).click();
  await expect(page).toHaveURL(/\/agents$/);

  // Wider again: every link returns to the bar.
  await page.setViewportSize({ width: 1920, height: 800 });
  await expect(inlineLinks(page)).toHaveCount(11);
});

test('"/" jumps to the search field, but not while typing', async ({ page }) => {
  await page.goto('/books');
  await page.locator('body').press('/');
  const search = page.getByRole('searchbox', { name: 'Search everything' });
  await expect(search).toBeFocused();
  await search.fill('a/b');
  await expect(search).toHaveValue('a/b');
});
