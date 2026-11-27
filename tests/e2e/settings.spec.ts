import { test, expect, type Page } from '@playwright/test';

// Settings → Modules, Theme and Account (docs/redesign/pages/settings-modules.md). Every change
// saves at once and the header and phone tab bar follow without a reload.

type Saved = { modules: Record<string, boolean>; moduleOrder: string[]; theme: string; effective: Record<string, boolean> };

const readSettings = (page: Page): Promise<Saved> => page.evaluate(async () => (await fetch('/api/settings', { cache: 'no-store' })).json());

async function restore(page: Page, original: Saved) {
  await page.evaluate(async body => {
    const response = await fetch('/api/settings', { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    if (!response.ok) throw new Error(`PATCH /api/settings: ${response.status}`);
  }, { modules: original.modules, moduleOrder: original.moduleOrder, theme: original.theme });
  await page.evaluate(() => localStorage.removeItem('career-theme'));
}

// A service worker would send requests itself, past page.route: the mocked logout below would
// really end the shared session.
test.use({ serviceWorkers: 'block' });

const main = (page: Page) => page.locator('main');
const orderList = (page: Page) => main(page).getByRole('list', { name: 'Module order' });
const preview = (page: Page) => main(page).getByRole('list', { name: 'Phone tab bar preview' });

test('/settings opens Modules, and the settings nav marks the section', async ({ page }) => {
  await page.goto('/settings');
  await expect(page).toHaveURL(/\/settings\/modules$/);
  const nav = page.getByRole('navigation', { name: 'Settings' });
  await expect(nav.getByRole('link')).toHaveText(['General', 'Modules', 'Theme', 'Account']);
  await expect(nav.locator('[aria-current="page"]')).toHaveText('Modules');
  await expect(main(page).getByRole('heading', { level: 1 })).toHaveText('Modules');
});

test.describe('wide enough for every header link', () => {
  test.use({ viewport: { width: 1920, height: 1000 } });

  test('switching modules saves at once, enforces dependencies and updates the header', async ({ page }) => {
    const countsRequest = page.waitForRequest(request => request.url().includes('/api/counts'));
    await page.goto('/settings/modules');
    // Turned-off modules need their counts too, for "paused · N kept".
    expect(new URL((await countsRequest).url()).searchParams.get('all')).toBe('1');
    const original = await readSettings(page);
    const journalLink = page.locator('[data-app-menu] > li[data-module-id="journal"]');
    try {
      await expect(main(page).getByText('10 of 10 on')).toBeVisible();
      await expect(journalLink).toBeVisible();
      const row = (id: string) => main(page).locator(`li[data-module-id="${id}"]`).first();
      await expect(row('timeline')).toContainText('Schedule and Agent context use this');

      // Journal off: paused with the real entry count, gone from the header without a reload.
      const journalCount = (await page.evaluate(async () => (await fetch('/api/counts')).json())).counts.journal as number;
      await main(page).getByRole('switch', { name: 'Journal' }).click();
      await expect(main(page).getByRole('switch', { name: 'Journal' })).toHaveAttribute('aria-checked', 'false');
      await expect(row('journal')).toContainText(journalCount ? `paused · ${journalCount} ${journalCount === 1 ? 'entry' : 'entries'} kept` : 'paused');
      await expect(main(page).getByText('9 of 10 on')).toBeVisible();
      await expect(journalLink).toHaveAttribute('hidden', '');
      await expect(page.locator('#app-more-menu li[data-module-id="journal"]')).toHaveCount(0);
      await expect.poll(async () => (await readSettings(page)).modules.journal).toBe(false);

      // Timeline off locks Schedule and Agent context; the tab bar preview fills from what's left.
      await main(page).getByRole('switch', { name: 'Timeline' }).click();
      await expect(main(page).getByRole('switch', { name: 'Schedule' })).toBeDisabled();
      await expect(row('schedule')).toContainText('needs Timeline');
      await expect(row('agents')).toContainText('needs Timeline');
      await expect(main(page).getByText('6 of 10 on')).toBeVisible();
      await expect(preview(page).locator('li')).toHaveText(['HOME', 'LEET', 'COMPANIES', 'PEOPLE', 'MORE']);
      await expect.poll(async () => (await readSettings(page)).effective.schedule).toBe(false);

      // Companies off keeps People but flags its catch-up queue.
      await main(page).getByRole('switch', { name: 'Companies' }).click();
      await expect(row('people')).toContainText('catch-up queue needs Companies');

      // Turning Timeline back on restores Schedule, whose own switch stayed on.
      await main(page).getByRole('switch', { name: 'Timeline' }).click();
      await expect(main(page).getByRole('switch', { name: 'Schedule' })).toHaveAttribute('aria-checked', 'true');
      await expect.poll(async () => (await readSettings(page)).effective.schedule).toBe(true);
    } finally {
      await restore(page, original);
    }
  });
});

test('quick changes save one at a time, in order', async ({ page }) => {
  await page.goto('/settings/modules');
  const original = await readSettings(page);
  const bodies: unknown[] = [];
  let release = () => {};
  const held = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/api/settings', async route => {
    if (route.request().method() !== 'PATCH') return route.continue();
    bodies.push(route.request().postDataJSON());
    if (bodies.length === 1) await held;
    await route.continue();
  });
  try {
    await expect(main(page).getByText('10 of 10 on')).toBeVisible();
    await main(page).getByRole('switch', { name: 'Notes' }).click();
    await main(page).getByRole('switch', { name: 'Bookshelf' }).click();
    await expect(main(page).getByText('8 of 10 on')).toBeVisible();
    // The second change waits for the first to answer.
    await page.waitForTimeout(300);
    expect(bodies).toEqual([{ modules: { notes: false } }]);
    release();
    await expect.poll(() => bodies).toEqual([{ modules: { notes: false } }, { modules: { bookshelf: false } }]);
    await expect.poll(async () => { const saved = await readSettings(page); return [saved.modules.notes, saved.modules.bookshelf]; }).toEqual([false, false]);
    await expect(main(page).getByText('8 of 10 on')).toBeVisible();
  } finally {
    release();
    await page.unrouteAll({ behavior: 'wait' });
    await restore(page, original);
  }
});

test('a failed save says so and shows what is really saved', async ({ page }) => {
  await page.goto('/settings/modules');
  await expect(main(page).getByText('10 of 10 on')).toBeVisible();
  await page.route('**/api/settings', route => route.request().method() === 'PATCH'
    ? route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: 'Database unavailable' }) })
    : route.continue());
  await main(page).getByRole('switch', { name: 'Notes' }).click();
  await expect(main(page).getByRole('alert')).toHaveText('Database unavailable');
  await expect(main(page).getByRole('switch', { name: 'Notes' })).toHaveAttribute('aria-checked', 'true');
});

test('the module order changes by keyboard and by dragging, and fills the tab bar', async ({ page }) => {
  await page.goto('/settings/modules');
  const original = await readSettings(page);
  try {
    await expect(orderList(page).locator('li').first()).toContainText('Timeline');
    const notes = main(page).getByRole('button', { name: /^Move Notes/ });
    await notes.focus();
    await page.keyboard.press('Home');
    await expect(orderList(page).locator('li').first()).toContainText('Notes');
    await expect(main(page).getByRole('button', { name: /^Move Notes, position 1 of 10/ })).toBeFocused();
    await page.keyboard.press('ArrowDown');
    await expect(orderList(page).locator('li').nth(1)).toContainText('Notes');
    await expect(preview(page).locator('li')).toHaveText(['HOME', 'TIMELINE', 'NOTES', 'SCHEDULE', 'MORE']);

    // Drag Bookshelf's handle to the top row.
    const handle = main(page).getByRole('button', { name: /^Move Bookshelf/ });
    const from = (await handle.boundingBox())!;
    const top = (await orderList(page).locator('li').first().boundingBox())!;
    await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2);
    await page.mouse.down();
    await page.mouse.move(from.x + from.width / 2, top.y + 4, { steps: 8 });
    await page.mouse.up();
    await expect(orderList(page).locator('li').first()).toContainText('Bookshelf');
    await expect.poll(async () => (await readSettings(page)).moduleOrder.slice(0, 3)).toEqual(['bookshelf', 'timeline', 'notes']);
    await expect(page.locator('[data-app-menu] > li[data-module-id]:visible > a').nth(1)).toHaveText('BOOKS');
  } finally {
    await restore(page, original);
  }
});

test('the Theme page applies and saves a theme', async ({ page }) => {
  await page.goto('/settings/theme');
  const original = await readSettings(page);
  try {
    await page.getByText('Concrete', { exact: true }).click();
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'concrete');
    await expect(page.getByRole('radio', { name: 'Concrete' })).toBeChecked();
    await expect.poll(async () => (await readSettings(page)).theme).toBe('concrete');
  } finally {
    await restore(page, original);
  }
});

test('Account signs out, and stays put when sign out fails', async ({ page }) => {
  await page.goto('/settings/account');
  await page.route('**/api/auth/logout', route => route.fulfill({ status: 500, body: '{}' }));
  await page.getByRole('button', { name: 'SIGN OUT' }).click();
  await expect(main(page).getByRole('alert')).toContainText('Sign out failed');
  await expect(page).toHaveURL(/\/settings\/account$/);
  // Answer the logout without ending the shared test session.
  await page.unroute('**/api/auth/logout');
  await page.route('**/api/auth/logout', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{}' }));
  await page.getByRole('button', { name: 'SIGN OUT' }).click();
  await expect(page).toHaveURL(/\/login$/);
});

test.describe('phone', () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true });

  test('Modules has a back link to More, switches and a tab bar preview that follows them', async ({ page }) => {
    await page.goto('/settings/modules');
    const original = await readSettings(page);
    try {
      await expect(page.getByRole('navigation', { name: 'Settings' })).toBeHidden();
      await expect(main(page).getByRole('link', { name: '← MORE' })).toHaveAttribute('href', '/more');
      await main(page).getByRole('switch', { name: 'Leetgrinder' }).tap();
      await expect(preview(page).locator('li')).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'COMPANIES', 'MORE']);
      await expect(page.locator('[data-app-tabbar] [data-module-id]:visible')).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'COMPANIES', 'MORE']);
      await expect(page.locator('[data-app-tabbar] [aria-current="page"]')).toHaveText('MORE');
    } finally {
      await restore(page, original);
    }
  });
});
