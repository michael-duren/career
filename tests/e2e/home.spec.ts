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
const yesterday = new Date(Date.now() - 86400000).toISOString().slice(0, 10);
const session = (id: string, title: string, hour: number, date = today) => ({
  id, date, assignment: { goalId: 'goal', title, goalTitle: 'Systems', color: '#86efac' },
  plan: date === today ? plan(hour) : { start: `${date}T${String(hour).padStart(2, '0')}:00:00Z`, end: `${date}T${String(hour + 1).padStart(2, '0')}:00:00Z` }, actual: null as null | Record<string, string>, state: 'accepted', conflictIds: [], exception: false,
});

async function mockWeek(page: Page) {
  const state = {
    revision: 'r1', week: mondayOf(today), days: [], goals: [], rules: [], busy: [], warnings: [], warningTargets: [], remainingCapacityHours: 0,
    settings: { timeZone: 'UTC', defaultDay: { start: '06:00', end: '22:00', nextDay: false }, weekdays: {}, dates: {} },
    sessions: [session('late', 'Read chapter 4', 15), session('early', 'Practice problems', 9), { ...session('lunch', 'Lunch', 12), assignment: { title: 'Lunch' } },
      { ...session('logged', 'Logged in scheduler', 7), actual: { status: 'explicit', date: today, start: `${today}T07:15:00Z`, end: `${today}T08:00:00Z` } },
      session('missed', 'Missed review', 10, yesterday)],
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
  await expect(list.getByRole('listitem')).toHaveCount(3);
  // Work recorded with explicit times in the scheduler shows as done but can't be unchecked here.
  const logged = list.getByRole('listitem').first();
  await expect(logged).toContainText('recorded in scheduler');
  await expect(logged.getByRole('checkbox')).toBeChecked();
  await expect(logged.getByRole('checkbox')).toBeDisabled();
  await expect(list.getByRole('listitem').nth(1)).toContainText('Systems › Practice problems');
  await expect(page.getByText('1 of 3 done')).toBeVisible();
  const first = list.getByRole('checkbox').nth(1);
  await expect(first).not.toBeChecked();
  expect(writes).toHaveLength(0);

  await list.getByText('Systems › Practice problems').click();
  await expect(first).toBeChecked();
  await expect(page.getByText('2 of 3 done')).toBeVisible();
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

test('home lists unchecked sessions from earlier in the week and keeps them after checking', async ({ page }) => {
  const writes = await mockWeek(page);
  await page.goto('/');
  const earlier = page.getByRole('list', { name: 'Earlier unchecked todos' });
  await expect(earlier.getByRole('listitem')).toHaveCount(1);
  await expect(earlier).toContainText('Systems › Missed review');
  await earlier.getByText('Systems › Missed review').click();
  await expect(earlier.getByRole('checkbox')).toBeChecked();
  expect(writes[0]).toMatchObject({ action: 'actual', id: 'missed', actual: { status: 'assumed', date: yesterday } });
  // Still listed, so a mistaken check can be undone right away.
  await earlier.getByText('Systems › Missed review').click();
  await expect(earlier.getByRole('checkbox')).not.toBeChecked();
  expect(writes[1]).toMatchObject({ action: 'clearActual', id: 'missed' });
});
