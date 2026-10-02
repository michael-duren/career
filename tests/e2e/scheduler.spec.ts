import { test, expect, type APIRequestContext, type Locator, type Page } from '@playwright/test';

const ownedGoals = new Set<string>();

// Regression coverage for three bugs found in production: recurring-delete
// not cascading to future occurrences, the session editor flashing open on a
// successful drag, and the subgoal picker being unavailable when editing an
// existing session. Each test creates its own disposable goal (real UUIDs -
// the API validates goal/step ids as UUIDs) so runs don't collide, mirroring
// Form fields are looked up scoped to the open
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
  ownedGoals.add(id);
  return goal;
}
async function goToNextWeek(page: Page) {
  // The heading text (and aria-busy) update from local state as soon as the
  // click handler runs, ahead of the fetch that actually repopulates the
  // grid - waiting on either of those, then reading [data-scheduler-date]
  // (which comes from the *fetched* week, not local state), can still race
  // and return the previous week's date. Wait on the one attribute this
  // function actually hands back instead.
  const firstDay = page.locator('[data-scheduler-date]').first();
  const before = await firstDay.getAttribute('data-scheduler-date');
  await page.getByRole('button', { name: 'Next week' }).click();
  await expect(firstDay).not.toHaveAttribute('data-scheduler-date', before ?? '');
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  return firstDay.getAttribute('data-scheduler-date');
}
async function visibleBox(locator: Locator) {
  await locator.scrollIntoViewIfNeeded();
  const box = await locator.boundingBox();
  if (!box) throw new Error('Drag source is unavailable.');
  return box;
}
function sessionFor(page: Page, title: string) {
  return page.locator('[data-session-id]').filter({ hasText: title });
}
// Browser input exercises pointer capture and hit testing on the real page.
async function mouseDrag(page: Page, from: { x: number; y: number }, to: { x: number; y: number }, steps = 8) {
  const viewport = page.viewportSize();
  if (!viewport) throw new Error('Browser viewport is unavailable.');
  for (const point of [from, to]) {
    expect(point.x).toBeGreaterThanOrEqual(0);
    expect(point.x).toBeLessThan(viewport.width);
    expect(point.y).toBeGreaterThanOrEqual(0);
    expect(point.y).toBeLessThan(viewport.height);
  }
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps });
  await page.mouse.up();
}
function localTimeOf(instant: string) {
  return new Intl.DateTimeFormat('en-US', { timeZone: 'America/Chicago', hour12: false, hour: '2-digit', minute: '2-digit' }).format(new Date(instant));
}

test.beforeEach(async ({ page }) => {
  ownedGoals.clear();
  await page.goto('/weekly-scheduler');
  await expect(page.locator('.scheduler-grid')).toBeVisible();
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
});

test.afterEach(async ({ request, baseURL }) => {
  if (!ownedGoals.size) return;
  const { revisions } = await (await request.get('/api/goals')).json();
  for (const id of ownedGoals) {
    const response = await request.delete('/api/goals', { headers: { origin: baseURL! }, data: { id, revision: revisions[id] } });
    expect(response.ok(), `Could not remove test goal ${id}: ${await response.text()}`).toBeTruthy();
  }
  ownedGoals.clear();
});

test('held assignment shows its snapped destination before the matching interval is saved', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!, { dailyHours: 0 });
  const date = await goToNextWeek(page);
  const source = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`));
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const target = (await day.boundingBox())!;
  await page.mouse.move(source.x + source.width / 2, source.y + source.height / 2);
  await page.mouse.down();
  await page.mouse.move(target.x + target.width / 2, target.y + 60, { steps: 8 });
  const preview = day.locator('.scheduler-drop-preview');
  await expect(preview).toBeVisible();
  await expect(preview).toContainText(date!);
  await expect(preview).toContainText('06:00–07:00');
  await expect(preview).toHaveAttribute('data-duration-minutes', '60');
  expect((await preview.boundingBox())?.height).toBe(60);
  expect(Math.abs((await preview.boundingBox())!.y - (target.y + 60))).toBeLessThan(2);
  await page.mouse.up();
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');
  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const session = state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  expect(localTimeOf(session.plan.start)).toBe('06:00');
  expect(localTimeOf(session.plan.end)).toBe('07:00');
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
  // Scoped to this goal's own session id, not just "the first session on the
  // page" - the suite doesn't clean up between specs and CI retries failed
  // tests, so an unscoped selector can end up dragging an unrelated session
  // left over from an earlier run.
  const sessionId = await session.getAttribute('data-session-id');
  const handleSelector = `[data-session-id="${sessionId}"] .scheduler-block-main`;
  const source = (await page.locator(handleSelector).boundingBox())!;
  const targetDate = await page.locator('[data-scheduler-date]').nth(1).getAttribute('data-scheduler-date');
  const target = (await page.locator('[data-scheduler-date]').nth(1).boundingBox())!;

  // Checking dialog.toBeHidden() only *after* the drag settles wouldn't catch
  // a flash regression - it's a retrying assertion, and the dialog reliably
  // ends up closed again by the time it's checked whether or not it flashed
  // open in between (setDraft(proposal) opening it, then setDraft(null)
  // closing it once the save resolved). A MutationObserver records every
  // "open" the attribute actually took during the gesture, independent of
  // when this test happens to look.
  await page.evaluate(() => {
    const w = window as unknown as { __dialogFlashed?: boolean };
    w.__dialogFlashed = false;
    const el = document.querySelector('.scheduler-editor-dialog')!;
    new MutationObserver(() => { if (el.hasAttribute('open')) w.__dialogFlashed = true; }).observe(el, { attributes: true, attributeFilter: ['open'] });
  });
  await mouseDrag(page, { x: source.x + source.width / 2, y: source.y + source.height / 2 }, { x: target.x + target.width / 2, y: target.y + 80 }, 10);
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');
  expect(await page.evaluate(() => (window as unknown as { __dialogFlashed?: boolean }).__dialogFlashed)).toBe(false);

  // The dialog must never appear for a successful drag save - it used to
  // flash open (setDraft before the async save resolved) even on success.
  await expect(dialog).toBeHidden();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);

  await expect.poll(async () => {
    const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
    const moved = state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
    return moved && { date: moved.date, start: localTimeOf(moved.plan.start), end: localTimeOf(moved.plan.end) };
  }).toEqual({ date: targetDate, start: '05:45', end: '06:45' });
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

test('creating a fixed commitment with no goal saves it as a plain reservation', async ({ page, request, baseURL }) => {
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
  const cleanup = await request.post('/api/scheduler/mutate', { headers: { origin: baseURL! }, data: { action: 'cancel', week: date, revision: state.revision, id: session.id } });
  expect(cleanup.ok(), await cleanup.text()).toBeTruthy();
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

test('the block\'s delete icon on a recurring session cascades to future occurrences too', async ({ page, request, baseURL }) => {
  // The dialog's "Remove this session" + "Apply change to" selector is one
  // delete path; the block's own "x" icon (a window.confirm, not the dialog)
  // is the other and more immediately visible one - it needs its own
  // coverage since it builds the mutation differently (see deleteSession in
  // WeeklyScheduler.tsx, which always sends scope: 'future' for a recurring
  // session from this icon, with no "this date only" option here).
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('16:00');
  await dialog.getByLabel('End time').fill('17:00');
  await dialog.getByLabel('Repeat weekly from this date').check();
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  const laterWeek = await goToNextWeek(page);

  // The delete icon is a hover-reveal control (opacity/pointer-events only
  // switch on via .scheduler-block:hover in CSS) - Playwright's actionability
  // check evaluates hittability before it moves the mouse, so it never
  // becomes clickable on its own; hovering the block first is what actually
  // triggers the CSS that makes it interactive.
  const block = sessionFor(page, goal.title).first();
  await block.hover();
  page.once('dialog', d => { expect(d.message()).toContain('future occurrences'); void d.accept(); });
  await block.getByRole('button', { name: `Delete ${goal.title}` }).click();
  await expect(page.locator('[role="alert"]')).toHaveCount(0);
  await expect(sessionFor(page, goal.title)).toHaveCount(0);

  const state = await (await request.get(`/api/scheduler/week?week=${laterWeek}`)).json();
  expect(state.sessions.some((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id)).toBe(false);
  const rule = state.rules.find((r: { assignment: { goalId: string } }) => r.assignment.goalId === goal.id);
  expect(rule?.effectiveTo).toBeDefined();
  expect(rule.effectiveTo < laterWeek!).toBe(true);
});

test('resizing a session by its bottom edge extends its duration', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!);
  const date = await goToNextWeek(page);
  // CI retries share the same disposable app, so use another day if an
  // earlier attempt already reserved this slot.
  const sessionDate = await page.locator('[data-scheduler-date]').nth(test.info().retry).getAttribute('data-scheduler-date');
  if (!sessionDate) throw new Error('Scheduling day is unavailable.');
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(sessionDate);
  await dialog.getByLabel('Start time').fill('09:00');
  await dialog.getByLabel('End time').fill('10:00');
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();

  // Resolve this goal's session rather than an unrelated session left by a retry.
  const sessionId = await sessionFor(page, goal.title).getAttribute('data-session-id');
  const edgeSelector = `[data-session-id="${sessionId}"] .scheduler-edge-bottom`;
  const box = (await page.locator(edgeSelector).boundingBox())!;
  await mouseDrag(page, { x: box.x + box.width / 2, y: box.y + box.height / 2 }, { x: box.x + box.width / 2, y: box.y + box.height / 2 + 60 }, 6);
  await expect(page.locator('[role="alert"]')).toHaveCount(0);

  await expect.poll(async () => {
    const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
    const session = state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
    return session && {
      end: localTimeOf(session.plan.end),
      minutes: (Date.parse(session.plan.end) - Date.parse(session.plan.start)) / 60000,
    };
  }).toEqual({ end: '11:00', minutes: 120 });
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

async function createPlannedSession(page: Page, goal: { id: string; title: string }, date: string, start: string, end: string) {
  const dialog = page.locator('.scheduler-editor-dialog');
  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date);
  await dialog.getByLabel('Start time').fill(start);
  await dialog.getByLabel('End time').fill(end);
  await dialog.getByRole('button', { name: 'Save session' }).click();
  await expect(dialog).toBeHidden();
  return sessionFor(page, goal.title);
}

test('held move and both resize edges preview the interval that the API stores', async ({ page, request, baseURL }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  const goal = await createGoal(request, baseURL!, { dailyHours: 0 });
  const date = await goToNextWeek(page);
  const session = await createPlannedSession(page, goal, date!, '06:00', '07:00');
  const nextDay = page.locator('[data-scheduler-date]').nth(1);
  const targetDate = (await nextDay.getAttribute('data-scheduler-date'))!;
  const source = await visibleBox(session.locator('.scheduler-block-main'));
  const target = (await nextDay.boundingBox())!;
  await page.mouse.move(source.x + source.width / 2, source.y + source.height / 2);
  await page.mouse.down();
  await page.mouse.move(target.x + target.width / 2, target.y + 90, { steps: 8 });
  let preview = nextDay.locator('.scheduler-drop-preview');
  await expect(preview).toHaveAttribute('data-start', '06:00');
  await expect(preview).toHaveAttribute('data-end', '07:00');
  await page.mouse.up();
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');
  await expect(session).toHaveCount(1);
  let block = (await session.boundingBox())!;
  expect(Math.abs(block.y - (target.y + 60))).toBeLessThan(2);

  const topEdge = (await session.locator('.scheduler-edge-top').boundingBox())!;
  await page.mouse.move(topEdge.x + topEdge.width / 2, topEdge.y + topEdge.height / 2);
  await page.mouse.down();
  await page.mouse.move(topEdge.x + topEdge.width / 2, topEdge.y + topEdge.height / 2 + 15, { steps: 4 });
  preview = nextDay.locator('.scheduler-drop-preview');
  await expect(preview).toHaveAttribute('data-start', '06:15');
  await expect(preview).toHaveAttribute('data-duration-minutes', '45');
  await page.mouse.up();
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');

  const bottomEdge = (await session.locator('.scheduler-edge-bottom').boundingBox())!;
  await page.mouse.move(bottomEdge.x + bottomEdge.width / 2, bottomEdge.y + bottomEdge.height / 2);
  await page.mouse.down();
  await page.mouse.move(bottomEdge.x + bottomEdge.width / 2, bottomEdge.y + bottomEdge.height / 2 + 30, { steps: 4 });
  preview = nextDay.locator('.scheduler-drop-preview');
  await expect(preview).toHaveAttribute('data-end', '07:30');
  await expect(preview).toHaveAttribute('data-duration-minutes', '75');
  await page.mouse.up();
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');
  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const saved = state.sessions.find((value: { assignment: { goalId: string } }) => value.assignment.goalId === goal.id);
  expect([saved.date, localTimeOf(saved.plan.start), localTimeOf(saved.plan.end), (Date.parse(saved.plan.end) - Date.parse(saved.plan.start)) / 60000]).toEqual([targetDate, '06:15', '07:30', 75]);
  block = (await session.boundingBox())!;
  expect(Math.abs(block.height - 75)).toBeLessThan(2);
});

test('known overlap appears during the held drag and a rejected drop retains its draft', async ({ page, request, baseURL }) => {
  const goalA = await createGoal(request, baseURL!, { dailyHours: 0 });
  const goalB = await createGoal(request, baseURL!, { dailyHours: 0 });
  const date = await goToNextWeek(page);
  await createPlannedSession(page, goalA, date!, '05:00', '06:00');
  const source = await visibleBox(page.locator(`#scheduler-goal-${goalB.id} .scheduler-goal-title`));
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const target = (await day.boundingBox())!;
  await page.mouse.move(source.x + source.width / 2, source.y + source.height / 2);
  await page.mouse.down();
  await page.mouse.move(target.x + target.width / 2, target.y + 30, { steps: 6 });
  const preview = day.locator('.scheduler-drop-preview');
  await expect(preview).toHaveClass(/is-conflicting/);
  await expect(preview).toContainText('Overlaps 1 known reservation');
  await page.mouse.up();
  await expect(page.locator('.scheduler-editor-dialog')).toBeVisible();
  await expect(page.locator('.scheduler-editor-dialog [role="alert"]')).toContainText('draft');
  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  expect(state.sessions.some((value: { assignment: { goalId: string } }) => value.assignment.goalId === goalB.id)).toBe(false);
});

test('a conflict added after the preview keeps the released draft editable', async ({ page, request, baseURL }) => {
  const goalA = await createGoal(request, baseURL!, { dailyHours: 0 });
  const goalB = await createGoal(request, baseURL!, { dailyHours: 0 });
  const date = await goToNextWeek(page);
  const source = await visibleBox(page.locator(`#scheduler-goal-${goalA.id} .scheduler-goal-title`));
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const target = (await day.boundingBox())!;
  await page.mouse.move(source.x + source.width / 2, source.y + source.height / 2);
  await page.mouse.down();
  await page.mouse.move(target.x + target.width / 2, target.y + 60, { steps: 8 });
  await expect(day.locator('.scheduler-drop-preview')).toHaveClass(/is-valid/);
  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const competing = await request.post('/api/scheduler/mutate', { headers: { origin: baseURL! }, data: {
    action: 'session', week: date, revision: state.revision,
    session: { id: '', date, assignment: { goalId: goalB.id, title: goalB.title }, plan: { start: `${date}T11:00:00Z`, end: `${date}T12:00:00Z` }, actual: null, state: 'accepted', exception: false, conflictIds: [] },
  } });
  expect(competing.ok(), await competing.text()).toBeTruthy();
  await page.mouse.up();
  const dialog = page.locator('.scheduler-editor-dialog');
  await expect(dialog).toBeVisible();
  await expect(dialog.getByLabel('Start time')).toHaveValue('06:00');
  await expect(dialog.locator('[role="alert"]')).toContainText('draft');
  const updated = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  expect(updated.sessions.some((value: { assignment: { goalId: string }; state: string }) => value.assignment.goalId === goalA.id && value.state === 'accepted')).toBe(false);
});

test('Escape and lost pointer capture clear the preview without writing', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!, { dailyHours: 0 });
  const date = await goToNextWeek(page);
  let writes = 0;
  await page.route('**/api/scheduler/mutate', async route => { writes++; await route.continue(); });
  const source = page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`);
  const from = await visibleBox(source);
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const to = (await day.boundingBox())!;
  for (const cancel of ['escape', 'capture'] as const) {
    await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2);
    await page.mouse.down();
    await page.mouse.move(to.x + to.width / 2, to.y + 60, { steps: 5 });
    await expect(day.locator('.scheduler-drop-preview')).toBeVisible();
    if (cancel === 'escape') await page.keyboard.press('Escape');
    else {
      const captured = await source.evaluate(element => {
      const pointer = (element as HTMLElement);
      const ids = [];
      for (let id = 1; id < 20; id++) if (pointer.hasPointerCapture(id)) { ids.push(id); pointer.releasePointerCapture(id); }
      return ids;
      });
      expect(captured.length).toBeGreaterThan(0);
      await page.mouse.move(to.x + to.width / 2 + 1, to.y + 60);
    }
    await expect(day.locator('.scheduler-drop-preview')).toHaveCount(0);
    await page.mouse.up();
  }
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2);
  await page.mouse.down();
  await page.mouse.move(to.x + to.width / 2, to.y + 60, { steps: 5 });
  await expect(day.locator('.scheduler-drop-preview')).toBeVisible();
  await page.mouse.move(1270, 10, { steps: 5 });
  await expect(page.locator('.scheduler-preview-outside')).toContainText('Place inside a scheduling day');
  await page.mouse.up();
  await expect(page.locator('.scheduler-preview-outside')).toHaveCount(0);
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2);
  await page.mouse.down();
  await page.mouse.move(to.x + to.width / 2, to.y + 60, { steps: 5 });
  await expect(day.locator('.scheduler-drop-preview')).toBeVisible();
  await page.evaluate(() => (document.querySelector('button[aria-label="Next week"]') as HTMLButtonElement).click());
  await expect(day.locator('.scheduler-drop-preview')).toHaveCount(0);
  await page.mouse.up();
  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  expect(state.sessions.some((value: { assignment: { goalId: string } }) => value.assignment.goalId === goal.id)).toBe(false);
  expect(writes).toBe(0);
});

test('viewport edge scrolling reaches a target below a 1280 by 900 screen', async ({ page, request, baseURL }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  const goal = await createGoal(request, baseURL!, { dailyHours: 0 });
  const date = await goToNextWeek(page);
  const source = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`));
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const dayBox = (await day.boundingBox())!;
  const initialScroll = await page.evaluate(() => scrollY);
  await page.mouse.move(source.x + source.width / 2, source.y + source.height / 2);
  await page.mouse.down();
  await page.mouse.move(dayBox.x + dayBox.width / 2, 890, { steps: 8 });
  await expect.poll(() => page.evaluate(() => scrollY)).toBeGreaterThan(initialScroll + 150);
  await expect(day.locator('.scheduler-drop-preview')).toBeVisible();
  await page.mouse.up();
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');
});

async function setDateOverride(page: Page, date: string, start: string, end: string) {
  await page.getByRole('button', { name: 'Day settings' }).click();
  const settings = page.getByRole('region', { name: 'Scheduler settings' });
  await settings.getByText('Date overrides', { exact: true }).click();
  await settings.getByLabel('Date', { exact: true }).fill(date);
  await settings.getByRole('button', { name: 'Add override' }).click();
  const override = settings.locator('fieldset').last();
  await override.getByLabel('Start', { exact: true }).fill(start);
  await override.getByLabel('End', { exact: true }).fill(end);
  await settings.getByRole('button', { name: 'Save settings' }).click();
  await expect(settings).toBeHidden();
}
async function clearDateOverride(request: APIRequestContext, origin: string, weekDate: string, overrideDate: string) {
  const state = await (await request.get(`/api/scheduler/week?week=${weekDate}`)).json();
  const dates = { ...state.settings.dates };
  delete dates[overrideDate];
  const response = await request.post('/api/scheduler/mutate', { headers: { origin }, data: { action: 'settings', week: weekDate, revision: state.revision, settings: { ...state.settings, dates } } });
  expect(response.ok(), await response.text()).toBeTruthy();
}

test('unavailable hours and ineligible dates show invalid previews and save nothing', async ({ page, request, baseURL }) => {
  const available = await createGoal(request, baseURL!, { dailyHours: 0 });
  const ineligible = await createGoal(request, baseURL!, { dailyHours: 0, startDate: '2026-10-06' });
  const date = await goToNextWeek(page);
  await setDateOverride(page, date!, '09:00', '12:00');
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const target = (await day.boundingBox())!;
  for (const [goal, minute, explanation] of [
    [available, 60, 'fit inside one scheduling day'],
    [ineligible, 270, 'not eligible'],
  ] as const) {
    const source = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`));
    const currentTarget = (await day.boundingBox())!;
    await page.mouse.move(source.x + source.width / 2, source.y + source.height / 2);
    await page.mouse.down();
    await page.mouse.move(currentTarget.x + currentTarget.width / 2, currentTarget.y + minute, { steps: 8 });
    const preview = day.locator('.scheduler-drop-preview');
    await expect(preview).toHaveClass(/is-invalid/);
    await expect(preview).toContainText(explanation);
    await page.mouse.up();
  }
  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  expect(state.sessions.some((value: { assignment: { goalId: string } }) => new Set<string>([available.id, ineligible.id]).has(value.assignment.goalId))).toBe(false);
  await clearDateOverride(request, baseURL!, date!, date!);
});

test('spring daylight gap shows a specific invalid preview and saves nothing', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!, { dailyHours: 0, startDate: '2026-01-01' });
  await page.reload();
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  await setDateOverride(page, '2027-03-14', '00:00', '04:00');
  const targetWeek = '2027-03-08';
  for (let i = 0; i < 23; i++) await goToNextWeek(page);
  await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', targetWeek);
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  const day = page.locator('[data-scheduler-date="2027-03-14"]');
  const source = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`));
  const target = (await day.boundingBox())!;
  await page.mouse.move(source.x + source.width / 2, source.y + source.height / 2);
  await page.mouse.down();
  await page.mouse.move(target.x + target.width / 2, target.y + 150, { steps: 8 });
  const preview = day.locator('.scheduler-drop-preview');
  await expect(preview).toHaveClass(/is-invalid/);
  await expect(preview).toContainText('does not exist');
  await page.mouse.up();
  const state = await (await request.get(`/api/scheduler/week?week=${targetWeek}`)).json();
  expect(state.sessions.some((value: { assignment: { goalId: string } }) => value.assignment.goalId === goal.id)).toBe(false);
  await clearDateOverride(request, baseURL!, targetWeek, '2027-03-14');
});

test('mobile touch scrolls the requirement list and drags only from its handle', async ({ page, request, baseURL }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const client = await page.context().newCDPSession(page);
  await client.send('Emulation.setTouchEmulationEnabled', { enabled: true, maxTouchPoints: 1 });
  const goal = await createGoal(request, baseURL!, { dailyHours: 0 });
  const date = await goToNextWeek(page);
  let writes = 0;
  await page.route('**/api/scheduler/mutate', async route => { writes++; await route.continue(); });
  const source = await visibleBox(page.locator('.scheduler-goal-title').first());
  const initialScroll = await page.evaluate(() => scrollY + (document.querySelector('.scheduler-goals')?.scrollTop ?? 0));
  const x = source.x + source.width / 2, y = source.y + source.height / 2;
  await client.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y, id: 1 }] });
  for (let step = 1; step <= 5; step++) await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x, y: y - step * 35, id: 1 }] });
  await client.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await expect.poll(() => page.evaluate(() => scrollY + (document.querySelector('.scheduler-goals')?.scrollTop ?? 0))).toBeGreaterThan(initialScroll + 50);
  await page.waitForTimeout(300);
  await page.evaluate(() => scrollTo(0, 0));
  await expect.poll(() => page.evaluate(() => scrollY)).toBe(0);
  const handle = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-handle`));
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const target = (await day.boundingBox())!;
  await client.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: handle.x + handle.width / 2, y: handle.y + handle.height / 2, id: 2 }] });
  for (let step = 1; step <= 8; step++) await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: handle.x + handle.width / 2 + (target.x + target.width / 2 - handle.x - handle.width / 2) * step / 8, y: handle.y + handle.height / 2 + (830 - handle.y - handle.height / 2) * step / 8, id: 2 }] });
  await expect.poll(async () => (await day.boundingBox())?.y).toBeLessThan(650);
  await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: 220, y: 500, id: 2 }] });
  await page.waitForTimeout(80);
  const visibleDay = (await day.boundingBox())!;
  await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: visibleDay.x + visibleDay.width / 2, y: visibleDay.y + 60, id: 2 }] });
  const preview = day.locator('.scheduler-drop-preview');
  await expect(preview).toHaveAttribute('data-start', '06:00');
  const previewStart = await preview.getAttribute('data-start');
  const previewEnd = await preview.getAttribute('data-end');
  await client.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');
  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const saved = state.sessions.find((value: { assignment: { goalId: string } }) => value.assignment.goalId === goal.id);
  expect([localTimeOf(saved.plan.start), localTimeOf(saved.plan.end)]).toEqual([previewStart, previewEnd]);
  expect(writes).toBe(1);

  await page.evaluate(() => scrollTo(0, 0));
  const cancelHandle = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-handle`));
  await client.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: cancelHandle.x + cancelHandle.width / 2, y: cancelHandle.y + cancelHandle.height / 2, id: 3 }] });
  await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: 220, y: 830, id: 3 }] });
  await expect.poll(async () => (await day.boundingBox())?.y).toBeLessThan(650);
  const cancelDay = (await day.boundingBox())!;
  await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: cancelDay.x + cancelDay.width / 2, y: cancelDay.y + 120, id: 3 }] });
  await expect(day.locator('.scheduler-drop-preview')).toBeVisible();
  await client.send('Input.dispatchTouchEvent', { type: 'touchCancel', touchPoints: [] });
  await expect(day.locator('.scheduler-drop-preview')).toHaveCount(0);
  expect(writes).toBe(1);
});

test('future actual work shows an invalid preview and sends no mutation', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!, { dailyHours: 0 });
  const first = (await page.locator('[data-scheduler-date]').first().getAttribute('data-scheduler-date'))!;
  const targetWeek = new Date(`${first}T12:00:00Z`);
  targetWeek.setUTCDate(targetWeek.getUTCDate() + 7);
  const date = targetWeek.toISOString().slice(0, 10);
  let writes = 0;
  await page.route('**/api/scheduler/mutate', async route => { writes++; await route.continue(); });
  await page.route('**/api/scheduler/week?**', async route => {
    const response = await route.fetch();
    const state = await response.json();
    if (state.week === date) state.sessions.push({
      id: 'future-actual-preview-fixture', date, assignment: { goalId: goal.id, title: goal.title },
      plan: null, actual: { status: 'explicit', date, start: `${date}T10:00:00Z`, end: `${date}T11:00:00Z` },
      state: 'accepted', exception: false, conflictIds: [],
    });
    await route.fulfill({ response, json: state });
  });
  await goToNextWeek(page);
  const block = sessionFor(page, goal.title);
  await expect(block).toBeVisible();
  const source = await visibleBox(block.locator('.scheduler-block-main'));
  const nextDay = page.locator('[data-scheduler-date]').nth(1);
  const target = (await nextDay.boundingBox())!;
  await page.mouse.move(source.x + source.width / 2, source.y + source.height / 2);
  await page.mouse.down();
  await page.mouse.move(target.x + target.width / 2, target.y + 90, { steps: 8 });
  const preview = nextDay.locator('.scheduler-drop-preview');
  await expect(preview).toHaveClass(/is-invalid/);
  await expect(preview).toContainText('Actual work cannot end in the future');
  await page.mouse.up();
  expect(writes).toBe(0);
});
