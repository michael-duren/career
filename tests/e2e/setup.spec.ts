import { test, expect, type Page } from '@playwright/test';

// First-run setup (docs/redesign/pages/setup.md): focus → modules → connect → theme → first goal,
// Skip with defaults, and re-running it from Settings → General. Each test clears the real
// setupCompletedAt and restores every setting afterwards.

type Saved = { modules: Record<string, boolean>; moduleOrder: string[]; theme: string; setupCompletedAt: string | null; effective: Record<string, boolean> };

const readSettings = (page: Page): Promise<Saved> => page.evaluate(async () => (await fetch('/api/settings', { cache: 'no-store' })).json());

const patch = (page: Page, body: object) => page.evaluate(async value => {
  const response = await fetch('/api/settings', { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(value) });
  if (!response.ok) throw new Error(`PATCH /api/settings: ${response.status} ${await response.text()}`);
}, body);

async function restore(page: Page, original: Saved) {
  await patch(page, { modules: original.modules, moduleOrder: original.moduleOrder, theme: original.theme, setupCompletedAt: original.setupCompletedAt });
  await page.evaluate(() => localStorage.removeItem('career-theme'));
}

/** Opens /setup as a new install would see it and returns the settings to restore. */
async function openFreshSetup(page: Page): Promise<Saved> {
  await page.goto('/settings/general');
  const original = await readSettings(page);
  await patch(page, { setupCompletedAt: null });
  await page.goto('/setup');
  return original;
}

const main = (page: Page) => page.locator('main');
const heading = (page: Page) => main(page).getByRole('heading', { level: 1 });
const next = (page: Page) => page.getByRole('button', { name: 'CONTINUE →' });

test.use({ serviceWorkers: 'block' });

test('new users are sent to setup after login; existing users go where they asked', async ({ browser }) => {
  for (const setupCompletedAt of [null, '2026-11-01T12:00:00Z']) {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] }, serviceWorkers: 'block' });
    const page = await context.newPage();
    // Answer the login and the settings check without touching the shared session or the real settings.
    await page.route('**/api/auth/login', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ authenticated: true, username: 'someone' }) }));
    await page.route('**/api/settings', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ modules: {}, moduleOrder: [], theme: 'dark', setupCompletedAt }) }));
    await page.goto('/login?redirect=/books');
    await page.getByLabel('Username').fill('someone');
    await page.getByLabel('Password').fill('anything');
    // The mocked login sets no cookie, so the server bounces the next page back to login: check
    // where the page went first.
    const leaving = page.waitForRequest(request => request.isNavigationRequest() && !new URL(request.url()).pathname.startsWith('/login'));
    await page.getByRole('button', { name: 'Sign in' }).click();
    expect(new URL((await leaving).url()).pathname).toBe(setupCompletedAt ? '/books' : '/setup');
    await context.close();
  }
});

test('the five steps save the picked modules, theme and first goal', async ({ page }) => {
  const original = await openFreshSetup(page);
  let goalId = '';
  try {
    const steps = page.getByRole('list', { name: 'Setup steps' });
    await expect(page.getByText('Setup · step 1 of 5')).toBeVisible();
    await expect(steps.getByRole('listitem')).toHaveText([/^focus$/i, /^modules\s+\(to do\)$/i, /^connect\s+\(to do\)$/i, /^theme\s+\(to do\)$/i, /^first goal\s+\(to do\)$/i], { useInnerText: true });
    await expect(page.getByRole('button', { name: '← BACK' })).toHaveCount(0);

    // 1 Focus: cards are checkboxes listing the modules they turn on.
    await expect(heading(page)).toHaveText('What are you working on?');
    const reflecting = main(page).getByRole('checkbox', { name: /Reflecting/ });
    await expect(main(page).getByText('Journal · Audio thoughts')).toBeVisible();
    await main(page).getByText('Reflecting', { exact: true }).click();
    await expect(reflecting).toBeChecked();
    await next(page).click();

    // 2 Modules: pre-set from the focus, suggested tags, dependency locks.
    await expect(page.getByText('Setup · step 2 of 5')).toBeVisible();
    await expect(heading(page)).toBeFocused();
    await expect(main(page).getByText('2 of 10 on')).toBeVisible();
    const row = (id: string) => main(page).locator(`li[data-module-id="${id}"]`);
    await expect(main(page).getByRole('switch', { name: 'Journal' })).toHaveAttribute('aria-checked', 'true');
    await expect(row('journal')).toContainText('suggested');
    await expect(main(page).getByRole('switch', { name: 'Schedule' })).toBeDisabled();
    await expect(row('schedule')).toContainText('needs Timeline');
    await main(page).getByRole('switch', { name: 'Timeline' }).click();
    await expect(main(page).getByRole('switch', { name: 'Agent context' })).toBeEnabled();
    await main(page).getByRole('switch', { name: 'Agent context' }).click();
    await main(page).getByRole('switch', { name: 'Audio thoughts' }).click();
    await expect(main(page).getByText('3 of 10 on')).toBeVisible();
    await next(page).click();

    // 3 Connect: only what the on modules use.
    await expect(heading(page)).toHaveText('Connect your tools');
    await expect(main(page).getByRole('listitem')).toHaveCount(1);
    await expect(main(page).getByRole('listitem')).toContainText('Claude (MCP)');
    await expect(main(page).getByRole('listitem')).toContainText('For Agent context');
    await next(page).click();

    // 4 Theme: picking applies it at once.
    await expect(heading(page)).toHaveText('Pick a look');
    await main(page).getByText('Vellum', { exact: true }).click();
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'vellum');
    await expect.poll(async () => (await readSettings(page)).theme).toBe('vellum');
    // Back keeps the choices.
    await page.getByRole('button', { name: '← BACK' }).click();
    await expect(heading(page)).toHaveText('Connect your tools');
    await next(page).click();
    await next(page).click();

    // 5 First goal, with the modules summary.
    await expect(heading(page)).toHaveText('Add your first goal');
    await expect(main(page).getByText('Timeline · Journal · Agent context')).toBeVisible();
    await main(page).getByLabel('Goal', { exact: true }).fill('Playwright setup goal');
    await main(page).getByLabel('Start', { exact: true }).fill('2026-12-01');
    await main(page).getByLabel('End', { exact: true }).fill('2026-11-01');
    await page.getByRole('button', { name: /FINISH/ }).click();
    await expect(main(page).getByRole('alert')).toHaveText('The end date is before the start date.');
    expect((await readSettings(page)).setupCompletedAt).toBeNull();
    await main(page).getByLabel('End', { exact: true }).fill('2027-01-31');
    await page.getByRole('button', { name: 'FINISH · OPEN HOME →' }).click();
    await expect(page).toHaveURL(/\/$/);

    const saved = await readSettings(page);
    expect(saved.setupCompletedAt).not.toBeNull();
    expect(Object.entries(saved.effective).filter(([, on]) => on).map(([id]) => id).sort()).toEqual(['agents', 'journal', 'timeline']);
    const goals = await page.evaluate(async () => (await fetch('/api/goals', { cache: 'no-store' })).json()) as { goals: { id: string; title: string; startDate: string; endDate: string }[] };
    const goal = goals.goals.find(g => g.title === 'Playwright setup goal');
    expect(goal).toMatchObject({ startDate: '2026-12-01', endDate: '2027-01-31' });
    goalId = goal!.id;
  } finally {
    if (goalId) {
      await page.evaluate(async id => {
        const { revisions } = await (await fetch('/api/goals', { cache: 'no-store' })).json();
        await fetch('/api/goals', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id, revision: revisions[id] }) });
      }, goalId);
    }
    await restore(page, original);
  }
});

test('Skip turns every module on, picks Dark and marks setup done', async ({ page }) => {
  const original = await openFreshSetup(page);
  try {
    await patch(page, { modules: { notes: false }, theme: 'cobalt' });
    await page.getByRole('button', { name: 'SKIP SETUP' }).click();
    await expect(page).toHaveURL(/\/$/);
    const saved = await readSettings(page);
    expect(saved.setupCompletedAt).not.toBeNull();
    expect(saved.theme).toBe('dark');
    expect(Object.values(saved.effective).every(Boolean)).toBe(true);
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  } finally {
    await restore(page, original);
  }
});

test('Settings → General runs setup again, starting from the saved modules', async ({ page }) => {
  await page.goto('/settings/general');
  const original = await readSettings(page);
  try {
    await patch(page, { modules: { notes: false } });
    const nav = page.getByRole('navigation', { name: 'Settings' });
    await expect(nav.locator('[aria-current="page"]')).toHaveText('General');
    await main(page).getByRole('link', { name: 'RUN SETUP AGAIN' }).click();
    await expect(page).toHaveURL(/\/setup$/);
    await next(page).click();
    await expect(main(page).getByText('9 of 10 on')).toBeVisible();
    await expect(main(page).getByRole('switch', { name: 'Notes' })).toHaveAttribute('aria-checked', 'false');
    await expect(main(page).getByText('suggested')).toHaveCount(0);
  } finally {
    await restore(page, original);
  }
});

test.describe('phone', () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true });

  test('one column, short step labels, theme previews two by two and the footer in reach', async ({ page }) => {
    const original = await openFreshSetup(page);
    try {
      await expect(page.getByText('Setup · step 1 of 5')).toBeHidden();
      await expect(page.getByRole('list', { name: 'Setup steps' }).getByRole('listitem').last()).toHaveText(/^goal\s+\(to do\)$/i, { useInnerText: true });
      await expect(next(page)).toBeInViewport();
      for (let i = 0; i < 3; i++) await next(page).tap();
      const previews = main(page).getByRole('radio');
      await expect(previews).toHaveCount(4);
      const [first, second, third] = await Promise.all([0, 1, 2].map(i => main(page).locator('label').nth(i).boundingBox()));
      expect(Math.round(first!.y)).toBe(Math.round(second!.y));
      expect(third!.y).toBeGreaterThan(first!.y);
      await next(page).tap();
      await expect(page.getByRole('button', { name: 'FINISH →' })).toBeInViewport();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    } finally {
      await restore(page, original);
    }
  });
});
