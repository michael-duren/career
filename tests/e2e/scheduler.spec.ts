import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

// Regression coverage for three bugs found in production: recurring-delete
// not cascading to future occurrences, the session editor flashing open on a
// successful drag, and the subgoal picker being unavailable when editing an
// existing session. Each test creates its own disposable goal (real UUIDs -
// the API validates goal/step ids as UUIDs) so runs don't collide, mirroring
// tests/scheduler.browser.mjs. Form fields are looked up scoped to the open
// dialog, not the whole page: Playwright's getByLabel/getByRole name matching
// is substring-based by default, and other sessions' resize-edge buttons
// (aria-label "Change start time of <title>") otherwise collide with "Start
// time" etc.
async function createGoal(request: APIRequestContext, origin: string, overrides: Record<string, unknown> = {}) {
  const id = crypto.randomUUID();
  const startDate = new Date().toISOString().slice(0, 10);
  const goal = {
    id, title: `Playwright goal ${id}`, status: 'planned', startDate, endDate: `${Number(startDate.slice(0, 4)) + 1}-12-31`,
    dailyHours: 1, selectedWeekdays: [1, 2, 3, 4, 5, 6, 7], color: '#67e8f9', dependsOn: [], steps: [], notes: [], metadata: {},
    createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(), ...overrides,
  };
  // Mutating endpoints require a matching Origin header (see internal/server/auth.go's
  // mutation() check) - a real browser sends this automatically, a bare request doesn't.
  const response = await request.post('/api/goals', { headers: { origin }, data: { revision: null, goal } });
  expect(response.ok(), await response.text()).toBeTruthy();
  return goal;
}
async function goToNextWeek(page: Page) {
  // The week heading updates synchronously with the click (it just formats
  // local state), before the new week's data has necessarily loaded - unlike
  // aria-busy, which can still read "false" from *before* the click if we
  // check it too early, since the load only flips it to "true" on the next
  // render pass. Waiting for the heading to actually change first avoids
  // reading stale (pre-navigation) day/session data below.
  const heading = page.locator('.scheduler-toolbar h2');
  const before = await heading.textContent();
  await page.getByRole('button', { name: 'Next week' }).click();
  await expect(heading).not.toHaveText(before ?? '');
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  return page.locator('[data-scheduler-date]').first().getAttribute('data-scheduler-date');
}
function sessionFor(page: Page, title: string) {
  return page.locator('[data-session-id]').filter({ hasText: title });
}
// Drives the component's pointer-capture-based drag/resize (see the comment
// on the drag test below) by dispatching PointerEvents directly at `selector`
// rather than relying on Playwright's mouse actions, which never reach it.
async function dispatchPointerDrag(page: Page, selector: string, from: { x: number; y: number }, to: { x: number; y: number }, steps = 8) {
  await page.evaluate(({ selector, fx, fy, tx, ty, steps }) => {
    const el = document.querySelector(selector) as HTMLElement;
    const fire = (type: string, x: number, y: number) => el.dispatchEvent(new PointerEvent(type, { bubbles: true, cancelable: true, pointerId: 1, isPrimary: true, clientX: x, clientY: y, button: 0, buttons: type === 'pointerup' ? 0 : 1 }));
    fire('pointerdown', fx, fy);
    for (let i = 1; i <= steps; i++) fire('pointermove', fx + (tx - fx) * i / steps, fy + (ty - fy) * i / steps);
    fire('pointerup', tx, ty);
  }, { selector, fx: from.x, fy: from.y, tx: to.x, ty: to.y, steps });
}
function localTimeOf(instant: string) {
  return new Intl.DateTimeFormat('en-US', { timeZone: 'America/Chicago', hour12: false, hour: '2-digit', minute: '2-digit' }).format(new Date(instant));
}

test.beforeEach(async ({ page }) => {
  await page.goto('/weekly-scheduler');
  await expect(page.locator('.scheduler-grid')).toBeVisible();
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
});

test('dragging a session to another day saves without flashing the editor or erroring', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('09:00');
  await dialog.getByLabel('End time').fill('10:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  const session = sessionFor(page, goal.title);
  await expect(session).toBeVisible();
  const handle = session.locator('.scheduler-block-main');
  const source = (await handle.boundingBox())!;
  const targetDate = await page.locator('[data-scheduler-date]').nth(1).getAttribute('data-scheduler-date');
  const target = (await page.locator('[data-scheduler-date]').nth(1).boundingBox())!;

  // The component implements dragging itself via pointer capture (see
  // WeeklyScheduler.tsx's beginPointerDrag/pointerDragMove), not native HTML5
  // drag-and-drop or Playwright's mouse actions - page.mouse.* never reached
  // its onPointerMove handler in testing (setPointerCapture routes real OS
  // input, not JS-dispatched events, and dispatching directly on the element
  // sidesteps that entirely). Dispatching PointerEvents straight at the
  // element is what actually drives it.
  await page.evaluate(([sx, sy, tx, ty]) => {
    const el = document.querySelector('[data-session-id] .scheduler-block-main') as HTMLElement;
    const fire = (type: string, x: number, y: number) => el.dispatchEvent(new PointerEvent(type, { bubbles: true, cancelable: true, pointerId: 1, isPrimary: true, clientX: x, clientY: y, button: 0, buttons: type === 'pointerup' ? 0 : 1 }));
    fire('pointerdown', sx, sy);
    for (let i = 1; i <= 10; i++) fire('pointermove', sx + (tx - sx) * i / 10, sy + (ty - sy) * i / 10);
    fire('pointerup', tx, ty);
  }, [source.x + source.width / 2, source.y + source.height / 2, target.x + target.width / 2, target.y + 80]);

  // The dialog must never appear for a successful drag save - it used to
  // flash open (setDraft before the async save resolved) even on success.
  await expect(dialog).toBeHidden();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');

  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const moved = state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  expect(moved?.date).toBe(targetDate);
});

test('deleting a recurring occurrence with "future" scope removes it from later weeks too', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('11:00');
  await dialog.getByLabel('End time').fill('12:00');
  await dialog.getByLabel('Repeat weekly from this date').check();
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  const laterWeek = await goToNextWeek(page); // one week further out: a later occurrence of the same rule

  const occurrence = sessionFor(page, goal.title).first();
  await expect(occurrence).toBeVisible();
  await occurrence.locator('.scheduler-block-main').click();
  await dialog.getByLabel('Apply change to').selectOption('future');
  await dialog.getByRole('button', { name: 'Remove this session' }).click();
  await expect(dialog).toBeHidden();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);

  const stillFurtherOut = await (await request.get(`/api/scheduler/week?week=${laterWeek}`)).json();
  // The API keeps ended rules around (for history) rather than deleting them,
  // so check it was actually ended before this week - not that it vanished -
  // and that no session got generated from it for this week either way.
  const rule = stillFurtherOut.rules.find((r: { assignment: { goalId: string } }) => r.assignment.goalId === goal.id);
  expect(rule?.effectiveTo).toBeDefined();
  expect(rule.effectiveTo < laterWeek!).toBe(true);
  expect(stillFurtherOut.sessions.some((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id)).toBe(false);
});

test('editing an existing planned session can assign it to a subgoal', async ({ page, request, baseURL }) => {
  const stepId = crypto.randomUUID();
  const goal = await createGoal(request, baseURL!, { steps: [{ id: stepId, title: 'Playwright subgoal', done: false }] });
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('13:00');
  await dialog.getByLabel('End time').fill('14:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  await sessionFor(page, goal.title).locator('.scheduler-block-main').click();
  await expect(dialog.getByLabel('Subgoal')).toBeVisible();
  await dialog.getByLabel('Subgoal').selectOption({ label: 'Playwright subgoal' });
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);

  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const session = state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  expect(session?.assignment.stepId).toBe(stepId);
});

test('creating a fixed commitment with no goal saves it as a plain reservation', async ({ page, request }) => {
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');
  const title = `Commute ${crypto.randomUUID()}`;

  await page.getByRole('button', { name: 'Add commitment' }).click();
  await dialog.getByLabel('Commitment name').fill(title);
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('15:00');
  await dialog.getByLabel('End time').fill('16:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);

  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const session = state.sessions.find((s: { assignment: { title: string } }) => s.assignment.title === title);
  expect(session?.assignment.goalId).toBeUndefined();
  expect(session?.plan).toBeTruthy();
});

test('editing an existing planned session updates its time', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('17:00');
  await dialog.getByLabel('End time').fill('18:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  await sessionFor(page, goal.title).locator('.scheduler-block-main').click();
  await dialog.getByLabel('Start time').fill('07:00');
  await dialog.getByLabel('End time').fill('08:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);

  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const session = state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  expect(localTimeOf(session.plan.start)).toBe('07:00');
});

test('deleting a single planned session removes it', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('06:00');
  await dialog.getByLabel('End time').fill('06:30');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  await sessionFor(page, goal.title).locator('.scheduler-block-main').click();
  await dialog.getByRole('button', { name: 'Remove this session' }).click();
  await expect(dialog).toBeHidden();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);
  // Canceled sessions are excluded from the calendar (state !== 'canceled')
  // but still returned by the API as a tombstone, not deleted outright.
  await expect(sessionFor(page, goal.title)).toHaveCount(0);

  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const session = state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  expect(session?.state).toBe('canceled');
});

test('resizing a session by its bottom edge extends its duration', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('18:00');
  await dialog.getByLabel('End time').fill('19:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  // ":has-text" is a Playwright-only pseudo-selector - it doesn't exist for
  // the native document.querySelector that dispatchPointerDrag runs inside
  // page.evaluate, so resolve the real session id first and build a plain
  // attribute selector from it.
  const sessionId = await sessionFor(page, goal.title).getAttribute('data-session-id');
  const edgeSelector = `[data-session-id="${sessionId}"] .scheduler-edge-bottom`;
  const box = (await page.locator(edgeSelector).boundingBox())!;
  await dispatchPointerDrag(page, edgeSelector, { x: box.x + box.width / 2, y: box.y }, { x: box.x + box.width / 2, y: box.y + 60 }, 6);
  await page.waitForTimeout(200);
  await expect(page.locator('[role="alert"]')).toHaveCount(0);

  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const session = state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  const minutes = (Date.parse(session.plan.end) - Date.parse(session.plan.start)) / 60000;
  expect(minutes).toBeGreaterThan(60);
});

test('overlapping sessions are flagged as a conflict and the failed draft is kept open for editing', async ({ page, request, baseURL }) => {
  const goalA = await createGoal(request, baseURL!);
  const goalB = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goalA.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('05:00');
  await dialog.getByLabel('End time').fill('06:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  await page.locator(`#scheduler-goal-${goalB.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('05:30');
  await dialog.getByLabel('End time').fill('06:30');
  await dialog.getByRole('button', { name: 'Save session' }).click();

  // The overlapping save is rejected: the draft stays open for correction and
  // the existing reservation it collides with is highlighted, matching the
  // established "Overlap retains the editor draft" behavior.
  await expect(page.locator('[role="alert"]')).toContainText('draft');
  await expect(page.locator('.scheduler-conflict')).toHaveCount(1);
  await expect(dialog).toBeVisible();
  await dialog.getByRole('button', { name: 'Discard draft' }).click();
  await expect(dialog).toBeHidden();

  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  expect(state.sessions.some((s: { assignment: { goalId: string } }) => s.assignment.goalId === goalB.id)).toBe(false);
});

test('editing a single occurrence with "This date" leaves the rule and other occurrences unchanged', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('08:00');
  await dialog.getByLabel('End time').fill('09:00');
  await dialog.getByLabel('Repeat weekly from this date').check();
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  const laterWeek = await goToNextWeek(page);
  await sessionFor(page, goal.title).first().locator('.scheduler-block-main').click();
  // "Apply change to" defaults to "This date" - leave it alone.
  await dialog.getByLabel('Start time').fill('10:00');
  await dialog.getByLabel('End time').fill('11:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);

  const changedWeek = await (await request.get(`/api/scheduler/week?week=${laterWeek}`)).json();
  const changedSession = changedWeek.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  expect(localTimeOf(changedSession.plan.start)).toBe('10:00');
  expect(changedWeek.rules.find((r: { assignment: { goalId: string } }) => r.assignment.goalId === goal.id)?.localStart).toBe('08:00');

  const evenLaterWeek = await goToNextWeek(page);
  const untouchedWeek = await (await request.get(`/api/scheduler/week?week=${evenLaterWeek}`)).json();
  const untouchedSession = untouchedWeek.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  expect(localTimeOf(untouchedSession.plan.start)).toBe('08:00');
});

test('a narrow viewport shows only the selected day and never scrolls horizontally', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('14:00');
  await dialog.getByLabel('End time').fill('15:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator('.scheduler-day-picker')).toBeVisible();
  const visibleDays = await page.locator('.scheduler-day').evaluateAll(days => days.filter(d => getComputedStyle(d).display !== 'none').length);
  expect(visibleDays).toBe(1);
  const scrollsHorizontally = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
  expect(scrollsHorizontally).toBe(false);
});
