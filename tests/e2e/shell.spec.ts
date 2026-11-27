import { test, expect, type Page } from '@playwright/test';

// The app header (docs/redesign/shell.md): Home plus every enabled module in the user's
// order, the current page marked, overflow in MORE +, search, settings and a 12-hour clock.
// The same markup is rendered by Astro (src/components/AppHeader.astro) and Go (internal/shell).

// Wide enough for every link, so MORE + (overflow only) stays hidden.
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
  await expect(page.getByRole('button', { name: 'MORE +' })).toBeHidden();
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
  // Sign out lives in Settings → Account now; MORE + holds only overflow links.
  await expect(moreMenu.getByRole('button', { name: 'SIGN OUT' })).toHaveCount(0);
  await page.keyboard.press('Escape');
  await expect(moreMenu).toBeHidden();
  await expect(more).toBeFocused();

  await more.click();
  await moreMenu.getByRole('link', { name: 'AGENTS' }).click();
  await expect(page).toHaveURL(/\/agents$/);

  // Wider again: every link returns to the bar and MORE + hides.
  await page.setViewportSize({ width: 1920, height: 800 });
  await expect(inlineLinks(page)).toHaveCount(11);
  await expect(more).toBeHidden();
});

test('"/" jumps to the search field, but not while typing', async ({ page }) => {
  await page.goto('/books');
  await page.locator('body').press('/');
  const search = page.getByRole('searchbox', { name: 'Search everything' });
  await expect(search).toBeFocused();
  await search.fill('a/b');
  await expect(search).toHaveValue('a/b');
});

// A fixed /api/search answer: one result each from Timeline, Schedule and Leetgrinder, two of
// seven journal matches and a document, as the API returns them (kinds in any order).
const SEARCH_ANSWER = {
  results: [
    { kind: 'document', id: 'home', title: 'Home', module: '', context: '', excerpt: '# Focus\n\nAlloy work this week', titleMatch: false },
    { kind: 'goal', id: 'g1', title: 'Contribute to Alloy', module: 'timeline', context: 'Active · Sep 14 → Dec 31', excerpt: '', titleMatch: true },
    { kind: 'week', id: '2026/week-26', title: 'Week 26 · Oct 16', module: 'journal', context: '', excerpt: 'Did some stuff on **Alloy** in the morning.', titleMatch: false },
    { kind: 'personal', id: 'p1', title: 'Alloy day', module: 'journal', context: 'Oct 17, 2026', excerpt: 'More Alloy', titleMatch: true },
    { kind: 'session', id: 's1', title: 'Contribute to Alloy', module: 'schedule', context: 'Today · 7:00 PM – 8:00 PM', excerpt: '', titleMatch: true },
    { kind: 'problem', id: 'alloy-tower', title: 'Alloy Tower', module: 'leetgrinder', context: '#9999 · Hard', excerpt: '', titleMatch: true },
  ],
  totals: { document: 1, goal: 1, week: 6, personal: 1, session: 1, problem: 1 },
};

async function mockSearch(page: Page) {
  const queries: string[] = [];
  await page.route('**/api/search?*', route => {
    queries.push(new URL(route.request().url()).search);
    return route.fulfill({ json: SEARCH_ANSWER });
  });
  return queries;
}

test('the header search groups results by module with counts, context and recent searches', async ({ page }) => {
  const queries = await mockSearch(page);
  // A saved module order (as shell.js caches it) puts Journal before Timeline's goals.
  await page.addInitScript(() => {
    if (sessionStorage.getItem('seeded')) return;
    sessionStorage.setItem('seeded', '1');
    localStorage.removeItem('career-recent-searches');
    localStorage.setItem('career-shell-settings', JSON.stringify({ modules: {}, moduleOrder: ['journal', 'timeline', 'schedule', 'leetgrinder'] }));
  });
  await page.route('**/api/settings', route => route.request().method() === 'GET'
    ? route.fulfill({ json: { modules: {}, moduleOrder: ['journal', 'timeline', 'schedule', 'leetgrinder'], theme: 'dark', setupCompletedAt: '2026-01-01T00:00:00Z' } })
    : route.fallback());
  await page.goto('/books');
  const search = page.getByRole('searchbox', { name: 'Search everything' });
  await search.fill('alloy');
  const panel = page.locator('#search-panel');
  await expect(panel.locator('#search-status')).toHaveText('11 results for “alloy”');
  expect(queries.at(-1)).toBe('?q=alloy');
  // The saved module order, documents last.
  await expect(panel.locator('.search-group h2')).toHaveText(['Journal', 'Goals', 'Schedule', 'Problems', 'Documents']);
  const journal = panel.getByRole('list', { name: 'Journal' });
  await expect(panel.locator('.search-group-count')).toHaveText(['2 of 7', '1', '1', '1', '1']);
  // Title matches show their metadata; body matches show the text around the match.
  await expect(journal.getByRole('link')).toHaveText(['Week 26 · Oct 16Did some stuff on Alloy in the morning.', 'Alloy dayOct 17, 2026']);
  await expect(panel.getByRole('link', { name: /Contribute to Alloy\s*Active/ })).toHaveAttribute('href', '/timeline?goal=g1');
  await expect(panel.getByRole('list', { name: 'Problems' }).getByRole('link')).toHaveAttribute('href', '/leetgrinder/problem/alloy-tower');

  // Arrow keys walk the results in order and back to the field.
  await search.press('ArrowDown');
  await expect(journal.getByRole('link').first()).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await expect(journal.getByRole('link').nth(1)).toBeFocused();
  await page.keyboard.press('ArrowUp');
  await page.keyboard.press('ArrowUp');
  await expect(search).toBeFocused();

  // Enter opens the first result and remembers the query.
  await Promise.all([page.waitForURL(url => url.pathname === '/journal' && url.searchParams.get('id') === '2026/week-26', { waitUntil: 'commit' }), search.press('Enter')]);
  await page.goto('/books');
  await page.locator('body').press('/');
  const chips = page.getByRole('group', { name: 'Recent' }).getByRole('button');
  await expect(chips).toHaveText(['alloy']);
  await search.press('ArrowDown');
  await expect(chips.first()).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(search).toHaveValue('alloy');
  await expect(panel.locator('.search-group')).toHaveCount(5);
  await page.keyboard.press('Escape');
  await expect(panel).toBeHidden();
});

test('the header search works on the Go-rendered Leetgrinder pages', async ({ page }) => {
  await mockSearch(page);
  await page.goto('/leetgrinder/problems');
  await page.locator('body').press('/');
  await page.keyboard.type('alloy');
  await expect(page.locator('#search-panel .search-group h2')).toHaveText(['Goals', 'Schedule', 'Problems', 'Journal', 'Documents']);
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

  test('the search button opens the full-screen search; Cancel and Escape close it', async ({ page }) => {
    await mockSearch(page);
    await page.goto('/books');
    await page.evaluate(() => localStorage.setItem('career-recent-searches', JSON.stringify(['kth largest', 'raft'])));
    await page.evaluate(() => window.scrollTo(0, 300));
    const toggle = page.getByRole('button', { name: 'Search everything' });
    const search = page.getByRole('searchbox', { name: 'Search everything' });
    await expect(search).toBeHidden();
    await toggle.tap();
    await expect(search).toBeFocused();
    await expect(toggle).toHaveAttribute('aria-expanded', 'true');
    // It covers the page from under the top bar (scrolled back into view) to the tab bar.
    const box = (await page.locator('.site-search').boundingBox())!;
    const bar = (await page.locator('.app-header-row').boundingBox())!;
    const tabBar = (await page.locator('[data-app-tabbar]').boundingBox())!;
    expect(Math.abs(box.y - (bar.y + bar.height))).toBeLessThanOrEqual(1);
    expect(Math.abs(box.y + box.height - tabBar.y)).toBeLessThanOrEqual(1);
    expect([box.x, box.width]).toEqual([0, page.viewportSize()!.width]);
    await expect(page.getByRole('group', { name: 'Recent' }).getByRole('button')).toHaveText(['kth largest', 'raft']);
    await search.fill('alloy');
    await expect(page.locator('.search-group h2')).toHaveText(['Goals', 'Schedule', 'Problems', 'Journal', 'Documents']);
    await expect(page.getByRole('group', { name: 'Recent' })).toBeHidden();
    // Tapping inside the page keeps it open.
    await page.locator('#search-status').tap();
    await expect(page.locator('.search-group')).toHaveCount(5);
    await page.getByRole('button', { name: 'Cancel' }).tap();
    await expect(search).toBeHidden();
    await expect(toggle).toBeFocused();
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
    await toggle.tap();
    await expect(search).toBeFocused();
    await page.keyboard.press('Escape');
    await expect(search).toBeHidden();
    await expect(toggle).toBeFocused();
    // Opening a result goes to its page.
    await toggle.tap();
    await page.getByRole('list', { name: 'Problems' }).getByRole('link').tap();
    await expect(page).toHaveURL('/leetgrinder/problem/alloy-tower');
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

    // A corrupt cache is ignored (the rendered tabs show) and replaced once settings load.
    await page.evaluate(() => localStorage.setItem('career-shell-settings', '{not json'));
    await page.goto('/books');
    await expect(visibleTabs(page)).toHaveText(['HOME', 'TIMELINE', 'SCHEDULE', 'LEET', 'MORE']);
    await expect.poll(() => page.evaluate(() => localStorage.getItem('career-shell-settings'))).toContain('moduleOrder');
  });

  test.describe('sign out', () => {
    // A service worker would send the logout itself, past page.route, and end the shared session.
    test.use({ serviceWorkers: 'block' });

    test('More keeps the session and says so when sign out fails', async ({ page }) => {
      await page.goto('/more');
      await page.route('**/api/auth/logout', route => route.fulfill({ status: 500, body: '{}' }));
      await page.getByRole('button', { name: 'SIGN OUT' }).tap();
      await expect(page.getByRole('alert')).toContainText('Sign out failed');
      await expect(page).toHaveURL(/\/more$/);
    });
  });
});
