import { test, expect, type Page } from '@playwright/test';

// The timeline page draws goals from GET /api/goals. Goals are mocked relative to today so the
// late/active split is deterministic; saving goals is covered by the Go goal tests.

const day = (offset: number) => new Date(Date.now() + offset * 86400000).toISOString().slice(0, 10);
const goal = (id: string, title: string, status: string, startDate: string, endDate: string, steps: { id: string; title: string; done: boolean }[] = []) => ({
  id, title, status, startDate, endDate, steps, color: '#86efac', dailyHours: 1, dependsOn: [], notes: [], metadata: {},
  createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-02T15:30:00Z',
});
const goals = [
  goal('late', 'Overdue reading', 'active', day(-80), day(-10)),
  goal('sys', 'System design prep', 'active', day(-30), day(60), [{ id: 's1', title: 'Caching', done: true }, { id: 's2', title: 'Queues', done: false }]),
  goal('done', 'Resume refresh', 'done', day(-60), day(-40)),
  goal('next', 'Apply to companies', 'planned', day(20), day(120)),
];

async function mockGoals(page: Page, list = goals) {
  const saved: Record<string, unknown>[] = [];
  await page.route('**/api/scheduler/goal-hours', route => route.fulfill({ json: { goals: {} } }));
  await page.route('**/api/goals', async route => {
    if (route.request().method() === 'GET') return route.fulfill({ json: { goals: list, revisions: Object.fromEntries(list.map(g => [g.id, 'r1'])) } });
    const body = route.request().postDataJSON();
    saved.push(body);
    await route.fulfill({ json: { goal: body.goal, revision: 'r2' } });
  });
  return saved;
}
// Compared with a fixed width: under mobile emulation innerWidth can follow overflowing content.
const overflow = (page: Page, width: number) => page.evaluate(w => document.documentElement.scrollWidth - w, width);

test.describe('desktop', () => {
  test('header counts, late tag and the selected goal panel', async ({ page }) => {
    await mockGoals(page);
    await page.goto('/timeline');
    const main = page.locator('main');
    const counts = main.getByRole('definition');
    await expect(counts).toHaveText(['02', '01', '01', '01']);
    await expect(main.getByRole('heading', { level: 1, name: 'Goal timeline' })).toBeVisible();
    // Fit: the axis starts at the first goal's month, not a fixed Aug-Jul window.
    const first = new Date(`${day(-80)}T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', year: 'numeric', timeZone: 'UTC' });
    await expect(main.locator('.tl-header .ui-label').first()).toHaveText(new RegExp(`^${first} – `));
    await expect(main.getByRole('button', { name: /^Overdue reading: .*, late$/ })).toBeVisible();
    await expect(main.locator('.tl-desktop .tl-row--late .tl-tag')).toHaveText('late');

    const panel = main.getByRole('region', { name: 'Selected goal' });
    await expect(panel.getByRole('heading', { name: 'System design prep' })).toBeVisible();
    await expect(panel.locator('li.is-done')).toHaveText(/Caching/);
    await expect(panel).toContainText('1 / 2 steps');
    await main.getByRole('button', { name: 'Apply to companies', exact: true }).click();
    await expect(panel.getByRole('heading', { name: 'Apply to companies' })).toBeVisible();
    await expect(panel).toContainText('No steps yet');

    await panel.getByRole('button', { name: 'Edit goal' }).click();
    const dialog = page.getByRole('dialog', { name: 'Goal details' });
    await expect(dialog.getByLabel('Name')).toHaveValue('Apply to companies');
    // 15:30 UTC is 9:30 AM in America/Chicago (the configured test zone).
    await expect(dialog).toContainText('Last updated Jan 2, 2026 · 9:30 AM');
  });

  test('range tabs change the axis and dragging a span saves new dates', async ({ page }) => {
    const saved = await mockGoals(page);
    await page.goto('/timeline');
    const main = page.locator('main');
    await main.getByRole('tab', { name: '2Y' }).click();
    await expect(main.getByRole('tab', { name: '2Y' })).toHaveAttribute('aria-selected', 'true');
    await main.getByRole('tab', { name: 'Fit' }).click();
    const label = main.locator('.tl-header .ui-label').first();
    const fit = await label.textContent();
    await main.getByRole('tab', { name: '2Y' }).click();
    await expect(label).not.toHaveText(fit!);
    await main.getByRole('tab', { name: 'Fit' }).click();
    await expect(label).toHaveText(fit!);

    const span = main.getByRole('button', { name: /^Apply to companies: / });
    const box = (await span.boundingBox())!;
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();
    await page.mouse.move(box.x + box.width / 2 + 80, box.y + box.height / 2, { steps: 5 });
    await page.mouse.up();
    await expect.poll(() => saved.length).toBe(1);
    const moved = (saved[0] as { goal: { startDate: string; endDate: string } }).goal;
    const length = (g: { startDate: string; endDate: string }) => Date.parse(g.endDate) - Date.parse(g.startDate);
    expect(moved.startDate > day(20)).toBe(true);
    expect(length(moved)).toBe(length({ startDate: day(20), endDate: day(120) }));
    expect((saved[0] as { revision: string }).revision).toBe('r1');
    await page.waitForTimeout(300);
    expect(saved.length).toBe(1);
  });

  test('a goal spanning the whole axis keeps its tag inside the timeline', async ({ page }) => {
    await mockGoals(page, [goal('all', 'Whole year', 'active', day(-150), day(150), [{ id: 'x', title: 'X', done: false }])]);
    await page.goto('/timeline');
    const track = (await page.locator('main .tl-desktop .tl-track').first().boundingBox())!;
    const tag = (await page.locator('main .tl-desktop .tl-tag').boundingBox())!;
    expect(tag.x).toBeGreaterThanOrEqual(track.x);
    expect(tag.x + tag.width).toBeLessThanOrEqual(track.x + track.width);
  });

  test('a ?goal= link opens that goal', async ({ page }) => {
    await mockGoals(page);
    await page.goto('/timeline?goal=sys');
    await expect(page.getByRole('dialog', { name: 'Goal details' }).getByLabel('Name')).toHaveValue('System design prep');
  });
});

test.describe('phone', () => {
  test.use({ viewport: { width: 360, height: 740 }, isMobile: true, hasTouch: true });

  test('tapping a goal expands its steps inline', async ({ page }) => {
    await mockGoals(page);
    await page.goto('/timeline');
    const main = page.locator('main');
    const row = main.getByRole('button', { name: /System design prep/, expanded: false });
    await row.tap();
    await expect(main.getByRole('button', { name: /System design prep/, expanded: true })).toBeVisible();
    const detail = main.locator('.tl-mdetail');
    await expect(detail).toContainText('Caching');
    await expect(detail).toContainText('Queues');
    await expect(main.locator('.tl-mrow.tl-row--late .tl-tag')).toHaveText('late');
    expect(await overflow(page, 360)).toBeLessThanOrEqual(0);
    await detail.getByRole('button', { name: 'Edit goal' }).tap();
    await expect(page.getByRole('dialog', { name: 'Goal details' })).toBeVisible();
  });

  test('the empty state fits the screen', async ({ page }) => {
    await mockGoals(page, []);
    await page.goto('/timeline');
    const empty = page.locator('main .tl-empty:visible');
    await expect(empty).toHaveText(/Space for your next goal/);
    const box = (await empty.boundingBox())!;
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(360);
    expect(await overflow(page, 360)).toBeLessThanOrEqual(0);
  });
});
