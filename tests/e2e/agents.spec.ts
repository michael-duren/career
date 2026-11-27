import { expect, test, type Page } from '@playwright/test';

// The Agent context page (docs/redesign/pages/agent-context.md) against a mocked API.
const goal = (id: string, status: string, endDate: string, steps: boolean[]) => ({
  id, title: id, status, dependsOn: [], startDate: '2026-09-01', endDate, color: '#999999', notes: [], metadata: {},
  createdAt: '2026-09-01T00:00:00Z', updatedAt: '2026-09-01T00:00:00Z',
  steps: steps.map((done, i) => ({ id: `${id}-${i}`, title: `Step ${i}`, done })),
});
const goals = [
  goal('LeetCode Curriculum', 'active', '2026-12-31', [true, false, false]),
  goal('OA Prep', 'active', '2026-11-18', [false]), // late, still counted as active
  goal('eBPF Book', 'planned', '2027-02-13', [false, false]),
  goal('Old talk', 'done', '2026-10-01', [true]),
];
// Long enough that the preview has to cut it.
const markdown = `# Agent context\n\n- Rule one\n\n# Current timeline goals\n\n${Array.from({ length: 80 }, (_, i) => `line ${i}`).join('\n')}\nTHE LAST LINE`;
const json = { rules: ['Rule one'], goals, journals: { work: [], personal: [] }, references: { notes: [], pages: [], books: [], companies: [] }, tail: 'x'.repeat(5000) };

async function mockContext(page: Page, { fail = false } = {}) {
  const state = { fail };
  await page.route('**/api/agent-context**', route => {
    if (state.fail) return route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'Database is down.' }) });
    const markdownView = new URL(route.request().url()).searchParams.get('format') === 'markdown';
    return markdownView
      ? route.fulfill({ status: 200, contentType: 'text/markdown', body: markdown })
      : route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(json) });
  });
  return state;
}

test.beforeEach(async ({ page, context }) => {
  await page.clock.setFixedTime(new Date('2026-11-27T12:00:00'));
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
});

test('shows stats, a cut preview, and copies the full output of the selected format', async ({ page }) => {
  await mockContext(page);
  await page.goto('/agents');
  const main = page.locator('main');
  await expect(main.getByRole('heading', { level: 1, name: 'Agent context' })).toBeVisible();
  const stats = main.getByRole('complementary', { name: 'About this output' });
  const stat = (label: string) => stats.locator('dl > div').filter({ has: page.getByText(label, { exact: true }) }).locator('dd');
  await expect(stat('Goals')).toHaveText('04');
  await expect(stat('Active')).toHaveText('02');
  await expect(stat('Steps')).toHaveText('07');
  await expect(stat('Characters')).toHaveText(markdown.length.toLocaleString('en-US'));

  const preview = main.getByRole('tabpanel', { name: 'Markdown preview' });
  await expect(preview).toContainText('agent-context.md');
  await expect(preview.locator('pre')).toContainText('line 0');
  await expect(preview.locator('pre')).not.toContainText('THE LAST LINE');
  await expect(preview).toContainText('more characters. Copy takes all of it.');
  await expect(stats.getByRole('link', { name: /GET \/api\/agent-context/ })).toHaveAttribute('href', '/api/agent-context?format=markdown');

  await main.getByRole('button', { name: 'Copy' }).click();
  await expect(main.getByRole('button', { name: 'Copied ✓' })).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(markdown);

  await main.getByRole('tab', { name: 'JSON' }).click();
  await expect(main.getByRole('button', { name: 'Copy' })).toBeVisible();
  const jsonPreview = main.getByRole('tabpanel', { name: 'JSON preview' });
  await expect(jsonPreview).toContainText('agent-context.json');
  await expect(jsonPreview.locator('pre')).toContainText('"rules": [');
  const pretty = JSON.stringify(json, null, 2);
  await expect(stat('Characters')).toHaveText(pretty.length.toLocaleString('en-US'));
  await expect(stats.getByRole('link', { name: /GET \/api\/agent-context/ })).toHaveAttribute('href', '/api/agent-context');
  await main.getByRole('button', { name: 'Copy' }).click();
  await expect(main.getByRole('button', { name: 'Copied ✓' })).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(pretty);

  // Keyboard: arrows move between formats.
  await main.getByRole('tab', { name: 'JSON' }).press('ArrowLeft');
  await expect(main.getByRole('tab', { name: 'Markdown' })).toHaveAttribute('aria-selected', 'true');
  await expect(main.getByRole('tab', { name: 'Markdown' })).toBeFocused();
});

test('a failed load shows the error and retries', async ({ page }) => {
  const state = await mockContext(page, { fail: true });
  await page.goto('/agents');
  const main = page.locator('main');
  await expect(main.getByRole('alert')).toContainText('Database is down.');
  await expect(main.getByRole('button', { name: 'Copy' })).toBeDisabled();
  state.fail = false;
  await main.getByRole('button', { name: 'Retry' }).click();
  await expect(main.getByRole('alert')).toHaveCount(0);
  await expect(main.getByRole('tabpanel', { name: 'Markdown preview' }).locator('pre')).toContainText('line 0');
  await expect(main.getByRole('button', { name: 'Copy' })).toBeEnabled();
});

test.describe('phone', () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test('segmented control, copy, preview and endpoint note', async ({ page }) => {
    await mockContext(page);
    await page.goto('/agents');
    const main = page.locator('main');
    await expect(main.getByText('4 goals · 7 steps')).toBeVisible();
    await expect(main.getByRole('complementary', { name: 'About this output' })).toBeHidden();
    await expect(main.getByRole('link', { name: 'Also at GET /api/agent-context?format=markdown' })).toBeVisible();
    await main.getByRole('tab', { name: 'JSON' }).tap();
    await expect(main.getByRole('link', { name: 'Also at GET /api/agent-context' })).toHaveAttribute('href', '/api/agent-context');
    await main.getByRole('button', { name: 'Copy' }).tap();
    await expect(main.getByRole('button', { name: 'Copied ✓' })).toBeVisible();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(JSON.stringify(json, null, 2));
    expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(0);
  });
});
