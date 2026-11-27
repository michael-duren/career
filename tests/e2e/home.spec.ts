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
  // The step is the title and its goal the sub-label.
  await expect(list.getByRole('listitem').nth(1)).toContainText('Practice problems');
  await expect(list.getByRole('listitem').nth(1)).toContainText('Systems');
  await expect(page.getByText('1 of 3 done')).toBeVisible();
  const first = list.getByRole('checkbox').nth(1);
  await expect(first).not.toBeChecked();
  expect(writes).toHaveLength(0);

  await list.getByText('Practice problems', { exact: true }).click();
  await expect(first).toBeChecked();
  await expect(page.getByText('2 of 3 done')).toBeVisible();
  expect(writes[0]).toMatchObject({ action: 'actual', id: 'early', revision: 'r1', actual: { status: 'assumed', date: today, ...plan(9) } });

  await list.getByText('Practice problems', { exact: true }).click();
  await expect(first).not.toBeChecked();
  expect(writes[1]).toMatchObject({ action: 'clearActual', id: 'early', revision: 'r2' });
});

test('journal page combines the work journal, personal journal and logs', async ({ page }) => {
  await page.goto('/journal');
  await expect(page.getByRole('tab', { name: 'Work' })).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('tab', { name: 'Personal' }).click();
  await expect(page).toHaveURL(/\/journal\?tab=personal$/);
  const panel = page.getByRole('tabpanel', { name: 'Personal' });
  // Personal entries stay behind the lock until opened.
  await panel.getByRole('button', { name: /Private · open to read/ }).click();
  await expect(panel.getByRole('link', { name: 'Read as Markdown →' })).toHaveAttribute('href', '/api/agent-context?format=markdown&journal=personal');
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

const goals = [
  { id: 'g1', status: 'active', title: 'Systems', startDate: '2026-01-01', endDate: '2999-12-31', steps: [{ id: 's1', title: 'a', done: true }, { id: 's2', title: 'b', done: false }, { id: 's3', title: 'c', done: false }] },
  { id: 'g2', status: 'active', title: 'Overdue goal', startDate: '2020-01-01', endDate: '2020-02-01', steps: [] },
  { id: 'g3', status: 'done', title: 'Finished goal', startDate: '2020-01-01', endDate: '2020-02-01', steps: [] },
];
const leet = {
  date: today, goal: { new: 2, review: 1 }, done: { new: 2, review: 0, bonus: 0 }, met: false, remaining: 1, streak: 2, record: 29, dueCount: 61,
  picks: [{ slug: 'kth-largest-element-in-a-stream', title: 'Kth Largest Element in a Stream', difficulty: 'Easy', recall: 0.67, reason: 'complexity judged wrong', done: false, number: 703 }],
  newPicks: [], due: [],
};
async function mockSections(page: Page, modules: Record<string, boolean> = {}) {
  await page.route('**/api/goals', route => route.fulfill({ json: { goals, revisions: {} } }));
  await page.route('**/api/leetgrinder/today', route => route.fulfill({ json: leet }));
  await page.route('**/api/settings', route => route.request().method() === 'GET'
    ? route.fulfill({ json: { modules, moduleOrder: [], theme: 'dark', setupCompletedAt: '2026-01-01T00:00:00Z' } })
    : route.fallback());
}

test('home shows active goals and the Leetgrinder streak and review pick', async ({ page }) => {
  await mockWeek(page);
  await mockSections(page);
  await page.goto('/');
  const goalList = page.getByRole('list', { name: 'Active goals' });
  await expect(goalList.getByRole('listitem')).toHaveCount(2);
  await expect(goalList.getByRole('listitem').first()).toContainText('1 / 3');
  await expect(goalList.getByRole('progressbar', { name: 'Systems: 1 of 3 steps' })).toHaveAttribute('aria-valuenow', '33');
  await expect(goalList.getByRole('listitem').nth(1)).toContainText('LATE');
  await expect(goalList.getByRole('link', { name: 'Systems' })).toHaveAttribute('href', '/timeline?goal=g1');
  await expect(page.getByText('Record 29')).toBeVisible();
  await expect(page.getByText('1 to go')).toBeVisible();
  const review = page.getByRole('region', { name: "Today's review" });
  await expect(review).toContainText('Review · #703 · Easy');
  await expect(review).toContainText('Recall 67% · complexity judged wrong');
  await expect(review.getByRole('link', { name: /open problem/i })).toHaveAttribute('href', '/leetgrinder/problem/kth-largest-element-in-a-stream');
  await expect(page.getByRole('link', { name: /61\s*Due/ })).toHaveAttribute('href', '/leetgrinder/reviews');
});

test('home sections disappear when their module is off', async ({ page }) => {
  await mockWeek(page);
  await mockSections(page, { leetgrinder: false, timeline: false });
  await page.goto('/');
  // Schedule needs Timeline, so all three sections are off.
  await expect(page.getByText('Schedule, Timeline and Leetgrinder are turned off')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'GOAL SESSIONS' })).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'ACTIVE GOALS' })).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'LEETGRINDER' })).toHaveCount(0);
});

test('home keeps sessions and goals when only Leetgrinder is off', async ({ page }) => {
  await mockWeek(page);
  await mockSections(page, { leetgrinder: false });
  await page.goto('/');
  await expect(page.getByRole('list', { name: "Today's todos" })).toBeVisible();
  await expect(page.getByRole('list', { name: 'Active goals' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'LEETGRINDER' })).toHaveCount(0);
});

test('the next session is circled and the note moves as time passes', async ({ page }) => {
  await page.clock.install({ time: new Date(`${today}T08:30:00Z`) });
  await mockWeek(page);
  await mockSections(page);
  await page.goto('/');
  const list = page.getByRole('list', { name: "Today's todos" });
  await expect(page.getByText('1 of 3 done — next in 30 min')).toBeVisible();
  await expect(list.getByRole('listitem').nth(1).locator('.ui-check__ring')).toHaveCount(1);
  await expect(list.locator('.ui-check__ring')).toHaveCount(1);
  // Past the end of the 9 AM session the ring moves to the 3 PM one.
  await page.clock.runFor('02:00:00');
  await expect(page.getByText('1 of 3 done — next at 3:00 PM')).toBeVisible();
  await expect(list.getByRole('listitem').nth(2).locator('.ui-check__ring')).toHaveCount(1);
  // Checking it off moves the ring on; with nothing left the note is just the count.
  await list.getByText('Read chapter 4', { exact: true }).click();
  await expect(page.getByText(/^2 of 3 done$/)).toBeVisible();
  await expect(list.locator('.ui-check__ring')).toHaveCount(0);
});

test('home shows the empty state with a link to the scheduler', async ({ page }) => {
  await page.route('**/api/scheduler/week?*', route => route.fulfill({ json: {
    revision: 'r1', week: mondayOf(today), days: [], goals: [], rules: [], busy: [], warnings: [], warningTargets: [], remainingCapacityHours: 0, sessions: [],
    settings: { timeZone: 'UTC', defaultDay: { start: '06:00', end: '22:00', nextDay: false }, weekdays: {}, dates: {} },
  } }));
  await mockSections(page);
  await page.goto('/');
  await expect(page.getByText('Nothing scheduled today.')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Plan in the scheduler.' })).toHaveAttribute('href', '/weekly-scheduler');
});

test('checking a session off while other sections load does not leave them loading', async ({ page }) => {
  await mockWeek(page);
  await mockSections(page);
  let releaseGoals!: () => void;
  const goalsHeld = new Promise<void>(resolve => { releaseGoals = resolve; });
  await page.route('**/api/goals', async route => { await goalsHeld; await route.fulfill({ json: { goals, revisions: {} } }); });
  await page.goto('/');
  await expect(page.getByText('Loading goals…')).toBeVisible();
  const list = page.getByRole('list', { name: "Today's todos" });
  await list.getByText('Practice problems', { exact: true }).click();
  await expect(page.getByText('2 of 3 done')).toBeVisible();
  releaseGoals();
  await expect(page.getByRole('list', { name: 'Active goals' }).getByRole('listitem')).toHaveCount(2);
});

test('a section that fails to load does not hide the others', async ({ page }) => {
  await mockWeek(page);
  await mockSections(page);
  await page.route('**/api/leetgrinder/today', route => route.fulfill({ status: 503, json: { error: 'Your progress is unavailable; please retry.' } }));
  await page.goto('/');
  await expect(page.getByRole('alert').filter({ hasText: 'Your progress is unavailable; please retry.' })).toBeVisible();
  await expect(page.getByRole('list', { name: "Today's todos" }).getByRole('listitem')).toHaveCount(3);
  await expect(page.getByRole('list', { name: 'Active goals' }).getByRole('listitem')).toHaveCount(2);
});

test('a skipped session is struck through and counts toward the total', async ({ page }) => {
  await page.route('**/api/scheduler/week?*', route => route.fulfill({ json: {
    revision: 'r1', week: mondayOf(today), days: [], goals: [], rules: [], busy: [], warnings: [], warningTargets: [], remainingCapacityHours: 0,
    settings: { timeZone: 'UTC', defaultDay: { start: '06:00', end: '22:00', nextDay: false }, weekdays: {}, dates: {} },
    sessions: [{ ...session('skip', 'Skipped review', 9), actual: { status: 'skipped', date: today, ...plan(9) } }, session('open', 'Open work', 22)],
  } }));
  await mockSections(page);
  await page.goto('/');
  const list = page.getByRole('list', { name: "Today's todos" });
  await expect(list.getByRole('listitem').first()).toContainText('skipped');
  await expect(list.getByRole('listitem').first().getByRole('checkbox')).not.toBeChecked();
  await expect(page.getByText(/^0 of 2 done/)).toBeVisible();
});

test('a failed check-off shows the error after the list reloads', async ({ page }) => {
  await mockWeek(page);
  await mockSections(page);
  await page.route('**/api/scheduler/mutate', route => route.fulfill({ status: 500, json: { error: 'Saving failed. Please retry.' } }));
  await page.goto('/');
  const list = page.getByRole('list', { name: "Today's todos" });
  await list.getByText('Practice problems', { exact: true }).click();
  await expect(page.getByRole('alert').filter({ hasText: 'Saving failed. Please retry.' })).toBeVisible();
  await expect(list.getByRole('checkbox').nth(1)).not.toBeChecked();
  await page.waitForTimeout(500);
  await expect(page.getByRole('alert').filter({ hasText: 'Saving failed. Please retry.' })).toBeVisible();
});
