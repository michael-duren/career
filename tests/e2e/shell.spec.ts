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
  const original = await page.evaluate(async () => (await fetch('/api/settings')).json());
  try {
    // Simulates what saveSettings() announces (tests/shell.test.ts covers that it does):
    // career:settings on this window...
    const saved = await patchSettings(page, { modules: { notes: false, timeline: false } });
    await page.evaluate(settings => window.dispatchEvent(new CustomEvent('career:settings', { detail: settings })), saved);
    // Turning Timeline off also hides Schedule and Agents (they depend on it).
    await expect(inlineLinks(page)).toHaveText(['HOME', 'LEET', 'COMPANIES', 'PEOPLE', 'BOOKS', 'JOURNAL', 'AUDIO']);

    // ...and a career-settings BroadcastChannel message from another tab.
    const other = await context.newPage();
    await other.goto('/books');
    const reordered = await patchSettings(other, { modules: { notes: true, timeline: true }, moduleOrder: ['notes', 'bookshelf'] });
    await other.evaluate(settings => { new BroadcastChannel('career-settings').postMessage(settings); }, reordered);
    await expect(inlineLinks(page)).toHaveText(['HOME', 'NOTES', 'BOOKS', 'TIMELINE', 'SCHEDULE', 'LEET', 'COMPANIES', 'PEOPLE', 'JOURNAL', 'AUDIO', 'AGENTS']);
    expect(await page.evaluate(() => (window as unknown as { sameDocument?: boolean }).sameDocument)).toBe(true);
    await other.close();
  } finally {
    await patchSettings(page, { modules: original.modules, moduleOrder: original.moduleOrder });
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

test.describe('phone (< 768px)', () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true });

  const tabBar = (page: Page) => page.locator('[data-app-tabbar]');
  const visibleTabs = (page: Page) => tabBar(page).locator('.app-tab:visible');

  test('top bar and tab bar replace the menu; the current tab is marked', async ({ page }) => {
    await page.goto('/weekly-scheduler');
    await expect(page.locator('[data-app-menu]')).toBeHidden();
    await expect(page.getByRole('link', { name: 'Settings' })).toBeHidden();
    await expect(visibleTabs(page)).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'LEET', 'MORE']);
    await expect(tabBar(page).locator('[aria-current="page"]')).toHaveText('SCHEDULE');
    // 62px + 16px bottom padding (no safe-area inset in the test browser); the page is padded to match.
    expect((await tabBar(page).boundingBox())!.height).toBe(78);
    expect(await page.evaluate(() => getComputedStyle(document.body).paddingBottom)).toBe('78px');
    for (const tab of await visibleTabs(page).all()) expect((await tab.boundingBox())!.width).toBeGreaterThanOrEqual(44);
    const toggle = (await page.getByRole('button', { name: 'Search everything' }).boundingBox())!;
    expect([toggle.width, toggle.height]).toEqual([44, 44]);
    // A module not in the bar marks More, which lists it.
    await page.goto('/companies');
    await expect(tabBar(page).locator('[aria-current="page"]')).toHaveText('MORE');
    // The Go-rendered Leetgrinder pages get the same bar.
    await page.goto('/leetgrinder/problems');
    await expect(tabBar(page).locator('[aria-current="page"]')).toHaveText('LEET');
    // Safe areas need viewport-fit=cover on every layout (Astro and Go).
    await expect(page.locator('meta[name="viewport"]')).toHaveAttribute('content', /viewport-fit=cover/);
    await page.goto('/books');
    await expect(page.locator('meta[name="viewport"]')).toHaveAttribute('content', /viewport-fit=cover/);
  });

  test('the search button opens the search field and Escape closes it', async ({ page }) => {
    await page.goto('/books');
    const toggle = page.getByRole('button', { name: 'Search everything' });
    const search = page.getByRole('searchbox', { name: 'Search everything' });
    await expect(search).toBeHidden();
    await toggle.tap();
    await expect(search).toBeFocused();
    await expect(toggle).toHaveAttribute('aria-expanded', 'true');
    await page.keyboard.press('Escape');
    await expect(search).toBeHidden();
    await expect(toggle).toBeFocused();
  });

  test('turning a tab module off fills its slot, and More lists the rest', async ({ page }) => {
    await page.goto('/more');
    const original = await page.evaluate(async () => (await fetch('/api/settings')).json());
    const more = page.getByRole('navigation', { name: 'More pages' });
    await expect(more.getByRole('link')).toHaveCount(7);
    await expect(more.getByRole('link').first()).toContainText('Companies');
    try {
      const saved = await patchSettings(page, { modules: { leetgrinder: false } });
      await page.evaluate(settings => window.dispatchEvent(new CustomEvent('career:settings', { detail: settings })), saved);
      await expect(visibleTabs(page)).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'COMPANIES', 'MORE']);
      await expect(more.getByRole('link').first()).toContainText('People');
      await expect(page.getByRole('link', { name: /Modules/ })).toContainText('9 of 10 on');
      // The next page draws the same tabs.
      await page.goto('/books');
      await expect(visibleTabs(page)).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'COMPANIES', 'MORE']);
    } finally {
      await patchSettings(page, { modules: original.modules, moduleOrder: original.moduleOrder });
    }
  });

  test('the More page picks a theme', async ({ page }) => {
    await page.goto('/more');
    const original = await page.evaluate(async () => (await fetch('/api/settings')).json());
    try {
      await page.getByText('VELLUM', { exact: true }).tap();
      await expect(page.locator('html')).toHaveAttribute('data-theme', 'vellum');
      await expect(page.getByRole('radio', { name: 'VELLUM' })).toBeChecked();
      await expect.poll(async () => (await page.evaluate(async () => (await fetch('/api/settings')).json())).theme).toBe('vellum');
      await page.reload();
      await expect(page.locator('html')).toHaveAttribute('data-theme', 'vellum');
    } finally {
      await patchSettings(page, { theme: original.theme });
      await page.evaluate(() => localStorage.removeItem('career-theme'));
    }
  });

  test('the next page draws tabs from the cached settings before /api/settings answers', async ({ page }) => {
    await page.goto('/books');
    // Let this page's own settings request land (it rewrites the cache) before seeding it.
    await expect.poll(() => page.evaluate(() => localStorage.getItem('career-shell-settings'))).toContain('moduleOrder');
    const cached = { modules: { timeline: true }, moduleOrder: ['notes', 'audio', 'bookshelf'], effective: {} };
    await page.evaluate(value => localStorage.setItem('career-shell-settings', JSON.stringify(value)), cached);
    let release = () => {};
    const held = new Promise<void>(resolve => { release = resolve; });
    await page.route('**/api/settings', async route => { await held; await route.continue(); });
    try {
      await page.goto('/books');
      await expect(visibleTabs(page)).toHaveText(['HOME', 'NOTES', 'AUDIO', 'BOOKS', 'MORE']);
      await expect(tabBar(page).locator('[aria-current="page"]')).toHaveText('BOOKS');
    } finally {
      release();
      await page.unrouteAll({ behavior: 'wait' });
    }
    // Once the request answers, the saved settings win and replace the cache.
    await expect(visibleTabs(page)).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'LEET', 'MORE']);

    // A corrupt cache is dropped and the rendered tabs stay until settings load.
    await page.evaluate(() => localStorage.setItem('career-shell-settings', '{not json'));
    await page.goto('/books');
    await expect(visibleTabs(page)).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'LEET', 'MORE']);
    await expect.poll(() => page.evaluate(() => localStorage.getItem('career-shell-settings'))).toContain('moduleOrder');
  });

  test('More keeps the session and says so when sign out fails', async ({ page }) => {
    await page.goto('/more');
    await page.route('**/api/auth/logout', route => route.fulfill({ status: 500, body: '{}' }));
    await page.getByRole('button', { name: 'SIGN OUT' }).tap();
    await expect(page.getByRole('alert')).toContainText('Sign out failed');
    await expect(page).toHaveURL(/\/more$/);
  });
});
