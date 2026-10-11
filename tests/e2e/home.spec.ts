import { test, expect, type APIRequestContext } from '@playwright/test';

// The home page combines current goals, the week, the work journal and logs.
// Logging work there must land in every place that reads it: the weekly
// scheduler (hours), the timeline goal (note, mini goal), and the journal.
// Like the other specs, it expects a disposable workspace: afterAll removes
// its goals, but the recorded session and journal line remain.
const ownedGoals = new Set<string>();

function chicagoDate(offsetDays = 0) {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'America/Chicago' }).format(new Date(Date.now() + offsetDays * 86400000));
}

async function createGoal(request: APIRequestContext, origin: string) {
  const id = crypto.randomUUID(), stepId = crypto.randomUUID();
  const goal = {
    id, title: `Home goal ${id.slice(0, 8)}`, status: 'active', startDate: chicagoDate(-30), endDate: chicagoDate(60),
    dailyHours: 1, selectedWeekdays: [1, 2, 3, 4, 5, 6, 7], color: '#86efac', dependsOn: [], notes: [], metadata: {},
    steps: [{ id: stepId, title: `Step ${id.slice(0, 8)}`, done: false }],
    createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
  };
  const response = await request.post('/api/goals', { headers: { origin }, data: { revision: null, goal } });
  expect(response.ok(), await response.text()).toBeTruthy();
  ownedGoals.add(id);
  return goal;
}

test.afterAll(async ({ request, baseURL }) => {
  for (const id of ownedGoals) {
    const list = await (await request.get('/api/goals')).json();
    await request.delete('/api/goals', { headers: { origin: baseURL! }, data: { id, revision: list.revisions[id] } });
  }
});

test('logging work from home syncs hours, goal notes and the journal', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const step = goal.steps[0];
  const yesterday = chicagoDate(-1);

  await page.goto('/');
  const card = page.locator('article').filter({ has: page.getByRole('heading', { name: goal.title }) });
  await expect(card).toContainText('0h logged');
  await expect(page.getByRole('heading', { name: /^This week ·/ })).toBeVisible();

  const form = page.locator('section').filter({ has: page.getByRole('heading', { name: 'A moment to reflect' }) });
  await form.getByRole('combobox', { name: /^Goal/ }).selectOption(goal.id);
  await form.getByRole('combobox', { name: /^Mini goal/ }).selectOption(step.id);
  await form.getByLabel('Date', { exact: true }).fill(yesterday);
  await form.getByLabel('Hours', { exact: true }).fill('1.5');
  await form.getByLabel(`Mark “${step.title}” done`).check();
  await form.getByLabel('Reflection').fill('Worked through the examples');
  await form.getByRole('button', { name: 'Log work' }).click();
  await expect(form.getByRole('status')).toContainText(`1.5h logged to ${goal.title}`);
  await expect(form.getByRole('status')).toContainText('journal updated');
  await expect(card).toContainText('1.5h logged');

  // Scheduler holds the hours as a recorded session.
  const hours = await (await request.get('/api/scheduler/goal-hours')).json();
  expect(hours.goals[goal.id].totalHours).toBe(1.5);
  // The timeline goal holds the note and the completed mini goal.
  const saved = (await (await request.get('/api/goals')).json()).goals.find((g: { id: string }) => g.id === goal.id);
  expect(saved.notes[0].body).toBe(`${yesterday} · ${step.title} · 1.5h logged — Worked through the examples`);
  expect(saved.steps[0].done).toBe(true);

  // Week view shows the session and the journal line on that day.
  await page.getByRole('tab', { name: 'Week' }).click();
  await expect(page).toHaveURL(/tab=week/);
  const weekPanel = page.getByRole('tabpanel', { name: 'Week' });
  // On a Monday, yesterday belongs to the previous week.
  if (new Date(`${chicagoDate(0)}T12:00:00Z`).getUTCDay() === 1) await weekPanel.getByRole('button', { name: 'Previous week' }).click();
  await expect(weekPanel).toContainText(`${goal.title} › ${step.title} · 1.5h`);
  await expect(weekPanel).toContainText(`${goal.title} › ${step.title} (1.5h) — Worked through the examples`);

  // Logs explorer and journal editor live in the same page.
  await page.getByRole('tab', { name: 'Logs' }).click();
  await expect(page.getByRole('tabpanel', { name: 'Logs' })).toContainText('Worked through the examples');
  await page.getByRole('tab', { name: 'Journal' }).click();
  await expect(page.getByRole('tabpanel', { name: 'Journal' }).getByRole('link', { name: '+ Add entry' })).toBeVisible();

  // Timeline reads the same logged hours.
  await page.goto(`/timeline`);
  await expect(page.locator('.timeline-list')).toContainText(`${goal.title} · ${goal.startDate} → ${goal.endDate} · 1.5h logged`);
});

test('old journal and logs URLs open their home tabs', async ({ page }) => {
  await page.goto('/logs');
  await expect(page).toHaveURL(/\/\?tab=logs$/);
  await expect(page.getByRole('tab', { name: 'Logs' })).toHaveAttribute('aria-selected', 'true');
  await page.goto('/journal');
  await expect(page.getByRole('tab', { name: 'Journal' })).toHaveAttribute('aria-selected', 'true');
});
