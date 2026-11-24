import { test, expect, type Page } from '@playwright/test';

// Home is today's todo list: the day's planned goal sessions, checked off by hand.
// The scheduler API is mocked so the list is deterministic at any time of day;
// the server side of completing a session is covered by the Go scheduler tests.

const today = new Date().toISOString().slice(0, 10);
function mondayOf(date: string) {
  const day = new Date(`${date}T12:00:00Z`);
  day.setUTCDate(day.getUTCDate() - (day.getUTCDay() + 6) % 7);
  return day.toISOString().slice(0, 10);
}
const plan = (hour: number) => ({ start: `${today}T${String(hour).padStart(2, '0')}:00:00Z`, end: `${today}T${String(hour + 1).padStart(2, '0')}:00:00Z` });
const session = (id: string, title: string, hour: number) => ({
  id, date: today, assignment: { goalId: 'goal', title, goalTitle: 'Systems', color: '#86efac' },
  plan: plan(hour), actual: null as null | Record<string, string>, state: 'accepted', conflictIds: [], exception: false,
});

async function mockWeek(page: Page) {
  const state = {
    revision: 'r1', week: mondayOf(today), days: [], goals: [], rules: [], busy: [], warnings: [], warningTargets: [], remainingCapacityHours: 0,
    settings: { timeZone: 'UTC', defaultDay: { start: '06:00', end: '22:00', nextDay: false }, weekdays: {}, dates: {} },
    sessions: [session('late', 'Read chapter 4', 15), session('early', 'Practice problems', 9), { ...session('lunch', 'Lunch', 12), assignment: { title: 'Lunch' } }],
  };
  const writes: Record<string, unknown>[] = [];
  await page.route('**/api/scheduler/week?*', route => route.fulfill({ json: state }));
  await page.route('**/api/scheduler/mutate', async route => {
    const body = route.request().postDataJSON();
    writes.push(body);
    const target = state.sessions.find(item => item.id === body.id)!;
    target.actual = body.action === 'clearActual' ? null : body.actual;
    state.revision = `r${writes.length + 1}`;
    await route.fulfill({ json: state });
  });
  return writes;
}

test('home lists today\'s goal sessions and checks them off only when clicked', async ({ page }) => {
  const writes = await mockWeek(page);
  await page.goto('/');
  const list = page.getByRole('list', { name: "Today's todos" });
  await expect(list.getByRole('listitem')).toHaveCount(2);
  await expect(list.getByRole('listitem').first()).toContainText('Systems › Practice problems');
  await expect(page.getByText('0 of 2 done')).toBeVisible();
  const first = list.getByRole('checkbox').first();
  await expect(first).not.toBeChecked();
  expect(writes).toHaveLength(0);

  await list.getByText('Systems › Practice problems').click();
  await expect(first).toBeChecked();
  await expect(page.getByText('1 of 2 done')).toBeVisible();
  expect(writes[0]).toMatchObject({ action: 'actual', id: 'early', revision: 'r1', actual: { status: 'assumed', date: today, ...plan(9) } });

  await list.getByText('Systems › Practice problems').click();
  await expect(first).not.toBeChecked();
  expect(writes[1]).toMatchObject({ action: 'clearActual', id: 'early', revision: 'r2' });
});

test('journal page combines the work journal, personal journal and logs', async ({ page }) => {
  await page.goto('/journal');
  await expect(page.getByRole('tab', { name: 'Work' })).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('tab', { name: 'Personal' }).click();
  await expect(page).toHaveURL(/\/journal\?tab=personal$/);
  await expect(page.getByRole('tabpanel', { name: 'Personal' })).toContainText('Read personal journal as Markdown');
});

test('old journal, logs and personal journal URLs open their journal tabs', async ({ page }) => {
  await page.goto('/logs');
  await expect(page).toHaveURL(/\/journal\?tab=logs$/);
  await expect(page.getByRole('tab', { name: 'Logs' })).toHaveAttribute('aria-selected', 'true');
  await page.goto('/personal-journal');
  await expect(page.getByRole('tab', { name: 'Personal' })).toHaveAttribute('aria-selected', 'true');
  await page.goto('/?tab=journal');
  await expect(page).toHaveURL(/\/journal$/);
  await expect(page.getByRole('tab', { name: 'Work' })).toHaveAttribute('aria-selected', 'true');
});
