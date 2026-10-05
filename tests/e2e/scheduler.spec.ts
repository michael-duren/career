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

test('dragging a goal or subgoal with many uncovered hours places a one-hour block', async ({ page, request, baseURL }) => {
  const stepId = crypto.randomUUID();
  const goal = await createGoal(request, baseURL!, { dailyHours: 8, steps: [{ id: stepId, title: 'One-hour subgoal', done: false }] });
  const date = await goToNextWeek(page);
  const card = page.locator(`#scheduler-goal-${goal.id}`);
  await expect(card.locator('.scheduler-shortfall')).toBeVisible();
  await card.locator('summary', { hasText: 'Subgoals' }).click();
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const preview = day.locator('.scheduler-drop-preview');

  for (const [source, offset, start, end] of [[card.locator('.scheduler-goal-title'), 60, '06:00', '07:00'], [card.locator('.scheduler-step'), 180, '08:00', '09:00']] as const) {
    const from = await visibleBox(source);
    const target = (await day.boundingBox())!;
    await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2);
    await page.mouse.down();
    await page.mouse.move(target.x + target.width / 2, target.y + offset, { steps: 8 });
    await expect(preview).toContainText(`${start}–${end}`);
    await expect(preview).toHaveAttribute('data-duration-minutes', '60');
    await page.mouse.up();
    await expect(page.locator('.scheduler-status')).toContainText('Saved.');
  }

  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  const sessions = state.sessions.filter((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id);
  expect(sessions.map((s: { assignment: { stepId?: string }; plan: { start: string; end: string } }) => [s.assignment.stepId ?? null, localTimeOf(s.plan.start), localTimeOf(s.plan.end)]).sort())
    .toEqual([[null, '06:00', '07:00'], [stepId, '08:00', '09:00']].sort());
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
  await session.locator('.scheduler-block-main').scrollIntoViewIfNeeded();
  await session.locator('.scheduler-block-main').evaluate(element => window.scrollBy(0, element.getBoundingClientRect().top - 370));
  const source = await visibleBox(session.locator('.scheduler-block-main'));
  const target = (await nextDay.boundingBox())!;
  expect(target.y + 90).toBeLessThan(840);
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

// US clocks spring forward on the second Sunday of March. Returns the first such
// Sunday whose scheduler week (starting Monday) begins after today, so the test
// never targets a past week.
function nextSpringForward() {
  const today = new Intl.DateTimeFormat('en-CA', { timeZone: 'America/Chicago' }).format(new Date());
  for (let year = Number(today.slice(0, 4)); ; year++) {
    const firstSunday = 1 + (7 - new Date(Date.UTC(year, 2, 1)).getUTCDay()) % 7;
    const gap = new Date(Date.UTC(year, 2, firstSunday + 7));
    const targetWeek = new Date(gap.getTime() - 6 * 86400000).toISOString().slice(0, 10);
    if (targetWeek > today) return { gapDate: gap.toISOString().slice(0, 10), targetWeek };
  }
}

test('unavailable hours and ineligible dates show invalid previews and save nothing', async ({ page, request, baseURL }) => {
  const available = await createGoal(request, baseURL!, { dailyHours: 0 });
  let date = await goToNextWeek(page);
  // The ineligible goal starts the day after the target date, so it is listed
  // for the week but cannot be placed on its first day.
  const startDate = new Date(Date.parse(`${date}T00:00:00Z`) + 86400000).toISOString().slice(0, 10);
  const ineligible = await createGoal(request, baseURL!, { dailyHours: 0, startDate });
  // Reload so the sidebar lists the new goal; next week is still the same date.
  await page.reload();
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  date = await goToNextWeek(page);
  // Clear the override even on failure; a leftover one invalidates later tests' drops on this date.
  try {
    await setDateOverride(page, date!, '09:00', '12:00');
    const day = page.locator(`[data-scheduler-date="${date}"]`);
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
  } finally {
    await clearDateOverride(request, baseURL!, date!, date!);
  }
});

// Cancels a goal's remaining sessions in a week so a retried test starts from an empty column.
async function cancelGoalSessions(request: APIRequestContext, origin: string, week: string, goalId: string) {
  for (;;) {
    const state = await (await request.get(`/api/scheduler/week?week=${week}`)).json();
    const session = state.sessions.find((s: { state: string; assignment: { goalId?: string } }) => s.state !== 'canceled' && s.assignment.goalId === goalId);
    if (!session) return;
    const response = await request.post('/api/scheduler/mutate', { headers: { origin }, data: { action: 'cancel', week: state.week, revision: state.revision, id: session.id } });
    expect(response.ok(), await response.text()).toBeTruthy();
  }
}
// Double-clicks a day column at a local clock time, using the axis's first label as its origin.
async function doubleClickerFor(page: Page, date: string) {
  const label = (await page.locator('.scheduler-axis span').first().textContent())!.match(/^(\d+):(\d+) (AM|PM)$/)!;
  const axisStart = (Number(label[1]) % 12 + (label[3] === 'PM' ? 12 : 0)) * 60 + Number(label[2]);
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  return async (clock: string) => {
    const box = await visibleBox(day);
    await page.mouse.dblclick(box.x + box.width / 2, box.y + Number(clock.slice(0, 2)) * 60 + Number(clock.slice(3)) - axisStart);
  };
}

test('double-clicking empty grid space drafts an entry that fills the gap up to one hour', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!, { dailyHours: 0 });
  const monday = await goToNextWeek(page);
  const date = new Date(Date.parse(`${monday}T00:00:00Z`) + 2 * 86400000).toISOString().slice(0, 10);
  const dialog = page.locator('.scheduler-editor-dialog');
  try {
    await setDateOverride(page, date, '09:00', '17:00');
    const doubleClickAt = await doubleClickerFor(page, date);

    await page.getByRole('button', { name: `Add session on ${date}` }).click();
    const quickAddAssignment = await dialog.getByLabel('Assignment').inputValue();
    await dialog.getByRole('button', { name: 'Discard draft' }).click();
    await expect(dialog).toBeHidden();

    await doubleClickAt('10:07');
    await expect(dialog.getByLabel('Start time')).toHaveValue('10:00');
    await expect(dialog.getByLabel('End time')).toHaveValue('11:00');
    await expect(dialog.getByLabel('Assignment')).toHaveValue(quickAddAssignment);
    await dialog.getByRole('button', { name: 'Discard draft' }).click();
    await expect(dialog).toBeHidden();

    await doubleClickAt('10:37');
    await expect(dialog).toBeVisible();
    await expect(dialog.getByLabel('Start time')).toHaveValue('10:30');
    await expect(dialog.getByLabel('End time')).toHaveValue('11:30');
    await dialog.getByLabel('Assignment').selectOption(goal.id);
    await dialog.getByRole('button', { name: 'Save session' }).click();
    await expect(dialog).toBeHidden();

    await doubleClickAt('10:07');
    await expect(dialog.getByLabel('Scheduling date')).toHaveValue(date);
    await expect(dialog.getByLabel('Start time')).toHaveValue('10:00');
    await expect(dialog.getByLabel('End time')).toHaveValue('10:30');
    await dialog.getByRole('button', { name: 'Discard draft' }).click();
    await expect(dialog).toBeHidden();

    await doubleClickAt('11:00');
    await expect(dialog.getByRole('button', { name: 'Remove this session' })).toBeVisible();
    await expect(dialog.getByLabel('Start time')).toHaveValue('10:30');
    await dialog.getByRole('button', { name: 'Discard draft' }).click();
    await expect(dialog).toBeHidden();

    await doubleClickAt('09:05');
    await expect(dialog.getByLabel('Start time')).toHaveValue('09:00');
    await expect(dialog.getByLabel('End time')).toHaveValue('10:00');
    await dialog.getByRole('button', { name: 'Discard draft' }).click();
    await expect(dialog).toBeHidden();

    await doubleClickAt('08:00');
    await page.waitForTimeout(300);
    await expect(dialog).toBeHidden();
    const state = await (await request.get(`/api/scheduler/week?week=${monday}`)).json();
    const sessions = state.sessions.filter((s: { state: string; assignment: { goalId?: string } }) => s.state !== 'canceled' && s.assignment.goalId === goal.id);
    expect(sessions.map((s: { date: string; plan: { start: string; end: string } }) => [s.date, localTimeOf(s.plan.start), localTimeOf(s.plan.end)])).toEqual([[date, '10:30', '11:30']]);
  } finally {
    await cancelGoalSessions(request, baseURL!, monday!, goal.id);
    await clearDateOverride(request, baseURL!, monday!, date);
  }
});

test('double-clicking today splits the gap at now into recorded work or a future plan', async ({ page, request, baseURL }) => {
  // Recorded work needs a goal to assign; without one a past double-click opens nothing.
  const goal = await createGoal(request, baseURL!, { startDate: '2030-01-07', endDate: '2030-12-31', dailyHours: 0 });
  await controlledWeek(page, request);
  await page.clock.pauseAt(new Date('2030-01-07T16:40:30Z'));
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  const dialog = page.locator('.scheduler-editor-dialog');
  try {
    await setDateOverride(page, '2030-01-07', '09:00', '17:00');
    const doubleClickAt = await doubleClickerFor(page, '2030-01-07');
    for (const [clock, start, end, actual] of [['10:07', '10:00', '10:40', true], ['10:42', '10:41', '11:41', false], ['10:50', '10:45', '11:45', false]] as const) {
      await doubleClickAt(clock);
      await expect(dialog.getByLabel('Start time')).toHaveValue(start);
      await expect(dialog.getByLabel('End time')).toHaveValue(end);
      await expect(dialog.getByRole('heading', { name: actual ? 'Record actual work' : 'New session', exact: true })).toBeVisible();
      await dialog.getByRole('button', { name: 'Discard draft' }).click();
      await expect(dialog).toBeHidden();
    }
    // The server runs on real time, so the 2030 plan cut at now is in its future and saves.
    await doubleClickAt('10:42');
    await dialog.getByLabel('Assignment').selectOption(goal.id);
    await dialog.getByRole('button', { name: 'Save session' }).click();
    await expect(dialog).toBeHidden();
    const state = await (await request.get('/api/scheduler/week?week=2030-01-07')).json();
    const saved = state.sessions.filter((s: { state: string; assignment: { goalId?: string } }) => s.state !== 'canceled' && s.assignment.goalId === goal.id);
    expect(saved.map((s: { plan: { start: string; end: string } }) => [localTimeOf(s.plan.start), localTimeOf(s.plan.end)])).toEqual([['10:41', '11:41']]);
  } finally {
    await cancelGoalSessions(request, baseURL!, '2030-01-07', goal.id);
    await clearDateOverride(request, baseURL!, '2030-01-07', '2030-01-07');
  }
});

test('double-clicking a past gap in a week without goals opens nothing', async ({ page, request }) => {
  await controlledWeek(page, request);
  await page.route('**/api/scheduler/week?*', async route => {
    const response = await request.get(`/api/scheduler/week?week=${new URL(route.request().url()).searchParams.get('week')}`);
    await route.fulfill({ json: { ...(await response.json()), goals: [] } });
  });
  await page.reload();
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  await page.clock.pauseAt(new Date('2030-01-07T16:40:30Z'));
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  await expect(page.locator('.scheduler-goal')).toHaveCount(0);
  const dialog = page.locator('.scheduler-editor-dialog');
  const doubleClickAt = await doubleClickerFor(page, '2030-01-07');
  await doubleClickAt('10:07');
  await page.waitForTimeout(300);
  await expect(dialog).toBeHidden();
  await doubleClickAt('10:50');
  await expect(dialog.getByRole('heading', { name: 'New session', exact: true })).toBeVisible();
  await expect(dialog.getByLabel('Start time')).toHaveValue('10:45');
});

test('spring daylight gap shows a specific invalid preview and saves nothing', async ({ page, request, baseURL }) => {
  // Reaching next March can take up to ~52 week navigations.
  test.setTimeout(120_000);
  const goal = await createGoal(request, baseURL!, { dailyHours: 0, startDate: '2026-01-01' });
  await page.reload();
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  const { gapDate, targetWeek } = nextSpringForward();
  try {
    await setDateOverride(page, gapDate, '00:00', '04:00');
    for (let i = 0; i < 80 && (await page.locator('[data-scheduler-date]').first().getAttribute('data-scheduler-date'))! < targetWeek; i++) await goToNextWeek(page);
    await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', targetWeek);
    await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
    const day = page.locator(`[data-scheduler-date="${gapDate}"]`);
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
  } finally {
    await clearDateOverride(request, baseURL!, targetWeek, gapDate);
  }
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
  const title = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`));
  await client.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: title.x + title.width / 2, y: title.y + title.height / 2, id: 3 }] });
  await client.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await expect(page.locator('.scheduler-editor-dialog')).toBeVisible();
  await page.locator('.scheduler-editor-dialog').getByRole('button', { name: 'Discard draft' }).click();

  await page.evaluate(() => scrollTo(0, 0));
  const cancelHandle = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-handle`));
  await client.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: cancelHandle.x + cancelHandle.width / 2, y: cancelHandle.y + cancelHandle.height / 2, id: 4 }] });
  await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: 220, y: 830, id: 4 }] });
  await expect.poll(async () => (await day.boundingBox())?.y).toBeLessThan(650);
  const cancelDay = (await day.boundingBox())!;
  await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: cancelDay.x + cancelDay.width / 2, y: cancelDay.y + 120, id: 4 }] });
  await expect(day.locator('.scheduler-drop-preview')).toBeVisible();
  await client.send('Input.dispatchTouchEvent', { type: 'touchCancel', touchPoints: [] });
  await expect(day.locator('.scheduler-drop-preview')).toHaveCount(0);
  expect(writes).toBe(1);
  await page.evaluate(() => scrollTo(0, 0));
  const titleAfterCancel = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`));
  await client.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: titleAfterCancel.x + titleAfterCancel.width / 2, y: titleAfterCancel.y + titleAfterCancel.height / 2, id: 5 }] });
  await client.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await expect(page.locator('.scheduler-editor-dialog')).toBeVisible();
  expect(writes).toBe(1);
});

test('tablet touch can start a visible drag handle', async ({ page, request, baseURL }) => {
  await page.setViewportSize({ width: 820, height: 1112 });
  const client = await page.context().newCDPSession(page);
  await client.send('Emulation.setTouchEmulationEnabled', { enabled: true, maxTouchPoints: 1 });
  const goal = await createGoal(request, baseURL!, { dailyHours: 0 });
  const date = await goToNextWeek(page);
  const handle = page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-handle`);
  await expect(handle).toBeVisible();
  const source = await visibleBox(handle);
  const day = page.locator(`[data-scheduler-date="${date}"]`);
  const target = (await day.boundingBox())!;
  await client.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: source.x + source.width / 2, y: source.y + source.height / 2, id: 1 }] });
  await client.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: target.x + target.width / 2, y: target.y + 60, id: 1 }] });
  await expect(day.locator('.scheduler-drop-preview')).toHaveAttribute('data-start', '06:00');
  await client.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await expect(page.locator('.scheduler-status')).toContainText('Saved.');
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

async function controlledWeek(page: Page, request: APIRequestContext, date = '2030-01-07') {
  await page.clock.install({ time: new Date(`${date}T08:00:00Z`) });
  const state = await (await request.get(`/api/scheduler/week?week=${date}`)).json();
  await page.route('**/api/scheduler/week?*', async route => {
    const requested = new URL(route.request().url()).searchParams.get('week');
    const response = await request.get(`/api/scheduler/week?week=${requested}`);
    await route.fulfill({ json: await response.json() });
  });
  await page.goto('/weekly-scheduler');
  await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', date);
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  return state;
}

async function pendingCommitment(page: Page) {
  await page.getByRole('button', { name: 'Add commitment', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: 'Session editor' });
  await dialog.getByLabel('Commitment name').fill('Request ordering commitment');
  await dialog.getByLabel('Start time', { exact: true }).fill('11:00');
  await dialog.getByLabel('End time', { exact: true }).fill('12:00');
  return dialog;
}

test('request coordination keeps Jan 14 visible when a delayed Jan 7 save completes', async ({ page, request }) => {
  const jan7 = await controlledWeek(page, request);
  let finish: () => void = () => {};
  const gate = new Promise<void>(resolve => { finish = resolve; });
  let writes = 0;
  await page.route('**/api/scheduler/mutate', async route => { writes++; await gate; await route.fulfill({ json: jan7 }); });
  const dialog = await pendingCommitment(page);
  await dialog.getByRole('button', { name: 'Save session', exact: true }).click();
  await expect.poll(() => writes).toBe(1);
  await page.evaluate(() => document.querySelector<HTMLButtonElement>('[aria-label="Next week"]')?.click());
  await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', '2030-01-14');
  finish();
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  await expect(page.locator('.scheduler-toolbar h2')).toContainText('Jan 14');
  await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', '2030-01-14');
});

test('request coordination retains failed proposal ownership after navigation and guards rapid submit', async ({ page, request }) => {
  const jan7 = await controlledWeek(page, request);
  let finish: () => void = () => {};
  const gate = new Promise<void>(resolve => { finish = resolve; });
  const writes: { week: string; revision: string; session: { plan: { start: string } } }[] = [];
  await page.route('**/api/scheduler/mutate', async route => {
    writes.push(route.request().postDataJSON());
    if (writes.length === 1) { await gate; await route.fulfill({ status: 409, json: { error: 'Another edit changed the revision', current: { ...jan7, revision: 'new-owning-revision' }, conflictIds: [] } }); }
    else await route.fulfill({ json: { ...jan7, revision: 'accepted-revision' } });
  });
  const dialog = await pendingCommitment(page);
  await dialog.evaluate(formDialog => { const form = formDialog.querySelector('form')!; form.requestSubmit(); form.requestSubmit(); });
  await expect.poll(() => writes.length).toBe(1);
  await page.evaluate(() => document.querySelector<HTMLButtonElement>('[aria-label="Next week"]')?.click());
  await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', '2030-01-14');
  finish();
  await expect(dialog.getByRole('alert')).toContainText('Review and retry');
  expect(writes).toHaveLength(1);
  await expect(dialog.getByLabel('Scheduling date', { exact: true })).toHaveValue('2030-01-07');
  await dialog.getByRole('button', { name: 'Save session', exact: true }).click();
  await expect(dialog).toBeHidden();
  expect(writes[1]).toMatchObject({ week: '2030-01-07', revision: 'new-owning-revision', session: { plan: { start: '2030-01-07T17:00:00.000Z' } } });
  await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', '2030-01-14');
  await page.unrouteAll({ behavior: 'wait' });
});

test('request coordination rejects a delayed read after a newer save and clears resolved errors', async ({ page, request }) => {
  const state = await controlledWeek(page, request);
  let finishSave: () => void = () => {};
  let finishRead: () => void = () => {};
  const saveGate = new Promise<void>(resolve => { finishSave = resolve; });
  const readGate = new Promise<void>(resolve => { finishRead = resolve; });
  let saving = false, reading = false;
  await page.route('**/api/scheduler/mutate', async route => { saving = true; await saveGate; await route.fulfill({ json: { ...state, warnings: ['Newly saved state'], revision: 'new' } }); });
  const dialog = await pendingCommitment(page);
  await dialog.getByRole('button', { name: 'Save session', exact: true }).click();
  await expect.poll(() => saving).toBe(true);
  await page.route('**/api/scheduler/week?*', async route => { reading = true; await readGate; await route.fulfill({ json: state }); });
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect.poll(() => reading).toBe(true);
  finishSave();
  await expect(page.locator('.scheduler-warnings')).toContainText('Newly saved state');
  finishRead();
  await expect(page.locator('.weekly-scheduler')).toHaveAttribute('aria-busy', 'false');
  await expect(page.locator('.scheduler-warnings')).toContainText('Newly saved state');
});

test('visible local refresh updates worker actuals, clock editability and reconnect health without availability retries', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!, { title: 'Clock commitment', startDate: '2030-01-07', endDate: '2030-12-31' });
  const state = await controlledWeek(page, request);
  await page.clock.setSystemTime(new Date('2030-01-07T10:59:30Z'));
  const session = { actual: null, exception: false, id: crypto.randomUUID(), date: state.week, assignment: { goalId: goal.id, title: 'Clock commitment' }, state: 'accepted', plan: { start: '2030-01-07T11:00:00Z', end: '2030-01-07T11:00:30Z' } };
  let reads = 0, healthReads = 0, availability = 0;
  await page.route('**/api/scheduler/google/status', route => { healthReads++; return route.fulfill({ json: { configured: true, connected: true, reconnectRequired: healthReads > 1, revision: 'google-health' } }); });
  await page.route('**/api/scheduler/google/refresh', route => { availability++; return route.fulfill({ json: {} }); });
  await page.route('**/api/scheduler/week?*', route => { reads++; return route.fulfill({ json: { ...state, goals: state.goals.map((summary: { actualHours: number }) => ({ ...summary, actualHours: reads > 1 ? 1 / 120 : 0 })), sessions: [{ ...session, actual: reads > 1 ? { ...session.plan, date: state.week, status: 'assumed' } : null }] } }); });
  await page.reload();
  await expect(page.getByRole('button', { name: 'Delete Clock commitment', exact: true })).toBeVisible();
  await page.clock.fastForward(31000);
  await expect(page.getByRole('button', { name: 'Delete Clock commitment', exact: true })).toHaveCount(0);
  await sessionFor(page, 'Clock commitment').locator('.scheduler-block-main').click();
  const clockDialog = page.getByRole('dialog', { name: 'Session editor' });
  await expect(clockDialog.getByRole('button', { name: 'Restore as planned' })).toBeDisabled();
  await page.clock.fastForward(30000);
  await expect(clockDialog.getByRole('button', { name: 'Restore as planned' })).toBeEnabled();
  await clockDialog.getByRole('button', { name: 'Discard draft' }).click();
  await expect(sessionFor(page, 'Clock commitment')).toContainText('assumed actual');
  await expect(page.locator(`#scheduler-goal-${goal.id} dd`).nth(1)).toHaveText('0.01h');
  await expect(page.locator('.scheduler-google summary')).toContainText('Reconnect needed');
  await page.clock.fastForward(600000);
  expect(availability).toBe(0);
});

test('Google selection conflict retains choices and reloads current revision for explicit retry', async ({ page, request }) => {
  await controlledWeek(page, request);
  let calendarReads = 0;
  const submitted: { revision: string; calendarIds: string[] }[] = [];
  await page.route('**/api/scheduler/google/status', route => route.fulfill({ json: { configured: true, connected: true, reconnectRequired: false, revision: 'status-revision' } }));
  await page.route('**/api/scheduler/google/calendars', async route => {
    if (route.request().method() === 'GET') { calendarReads++; await route.fulfill({ json: { revision: calendarReads === 1 ? 'old-selection' : 'current-selection', calendars: [{ id: 'busy', summary: 'Busy calendar', selected: false }] } }); }
    else { submitted.push(route.request().postDataJSON()); await route.fulfill(submitted.length === 1 ? { status: 409, json: { error: 'Calendar selection changed', conflictIds: [] } } : { json: {} }); }
  });
  await page.reload();
  await page.locator('.scheduler-google summary').click();
  await page.getByRole('button', { name: 'Select busy calendars' }).click();
  await page.getByLabel('Busy calendar', { exact: true }).check();
  await page.getByRole('button', { name: 'Save calendar selection' }).click();
  await expect(page.locator('.scheduler-google [role="alert"]')).toContainText('Calendar selection changed');
  await expect.poll(() => calendarReads).toBe(2);
  await expect(page.getByLabel('Busy calendar', { exact: true })).toBeChecked();
  expect(submitted).toHaveLength(1);
  await page.getByRole('button', { name: 'Save calendar selection' }).click();
  await expect.poll(() => submitted.length).toBe(2);
  expect(submitted[1]).toEqual({ revision: 'current-selection', calendarIds: ['busy'] });
});

test('local refresh keeps drafts, clears resolved errors and pauses while the page is hidden', async ({ page, request }) => {
  const state = await controlledWeek(page, request);
  let failing = true, reads = 0, healthReads = 0;
  await page.route('**/api/scheduler/week?*', route => { reads++; return route.fulfill(failing ? { status: 503, json: { error: 'Temporary read failure' } } : { json: state }); });
  await page.route('**/api/scheduler/google/status', route => { healthReads++; return route.fulfill({ json: { configured: false, connected: false, reconnectRequired: false, revision: 'health' } }); });
  const dialog = await pendingCommitment(page);
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(dialog.getByRole('alert')).toContainText('Temporary read failure');
  failing = false;
  await dialog.getByRole('button', { name: 'Refresh schedule', exact: true }).click();
  await expect(dialog.getByRole('alert')).toHaveCount(0);
  await expect(dialog.getByLabel('Commitment name')).toHaveValue('Request ordering commitment');
  await page.evaluate(() => { Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' }); document.dispatchEvent(new Event('visibilitychange')); });
  const before = { reads, healthReads };
  await page.clock.fastForward(180000);
  expect({ reads, healthReads }).toEqual(before);
  await page.evaluate(() => { Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' }); document.dispatchEvent(new Event('visibilitychange')); });
  await expect.poll(() => reads).toBe(before.reads + 1);
  await dialog.getByRole('button', { name: 'Discard draft' }).click();
  await page.locator('.scheduler-toolbar').getByRole('button', { name: 'Refresh schedule', exact: true }).click();
  await expect.poll(() => reads).toBe(before.reads + 2);
});

test('failed drag after ordinary navigation opens its original week draft and saves with original settings', async ({ page, request, baseURL }) => {
  const goal = await createGoal(request, baseURL!, { startDate: '2030-01-07', endDate: '2030-12-31', dailyHours: 0 });
  const state = await controlledWeek(page, request);
  const next = await (await request.get('/api/scheduler/week?week=2030-01-14')).json();
  await page.route('**/api/scheduler/week?*', route => route.fulfill({ json: new URL(route.request().url()).searchParams.get('week') === state.week ? state : { ...next, settings: { ...next.settings, timeZone: 'UTC' } } }));
  let finish: () => void = () => {};
  const gate = new Promise<void>(resolve => { finish = resolve; });
  const writes: { week: string; revision: string; session: { plan: { start: string } } }[] = [];
  await page.route('**/api/scheduler/mutate', async route => { writes.push(route.request().postDataJSON()); if (writes.length === 1) { await gate; await route.fulfill({ status: 503, json: { error: 'Save temporarily unavailable' } }); } else await route.fulfill({ json: state }); });
  const source = await visibleBox(page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`));
  const day = (await page.locator('[data-scheduler-date]').first().boundingBox())!;
  await mouseDrag(page, { x: source.x + source.width / 2, y: source.y + source.height / 2 }, { x: day.x + day.width / 2, y: day.y + 60 });
  await expect.poll(() => writes.length).toBe(1);
  await expect(page.getByRole('button', { name: 'Add commitment', exact: true })).toBeDisabled();
  await expect(page.locator('.scheduler-edge').first()).toHaveCount(0);
  await page.getByRole('button', { name: 'Next week' }).click();
  await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', '2030-01-14');
  finish();
  const dialog = page.getByRole('dialog', { name: 'Session editor' });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole('alert')).toContainText('Save temporarily unavailable');
  await expect(dialog.getByLabel('Scheduling date', { exact: true })).toHaveValue('2030-01-07');
  await expect(dialog).toContainText('Times use America/Chicago');
  await dialog.getByRole('button', { name: 'Save session', exact: true }).click();
  await expect(dialog).toBeHidden();
  expect(writes[1]).toMatchObject({ week: '2030-01-07', revision: state.revision, session: { plan: { start: '2030-01-07T12:00:00.000Z' } } });
  await expect(page.locator('[data-scheduler-date]').first()).toHaveAttribute('data-scheduler-date', '2030-01-14');
});

test('an open plan draft locks its original plan at the start boundary and offers actual recording', async ({ page, request, baseURL }) => {
  const stepId = crypto.randomUUID();
  const goal = await createGoal(request, baseURL!, { steps: [{ id: stepId, title: 'Unsaved subgoal', done: false }], startDate: '2030-01-07', endDate: '2030-12-31', dailyHours: 0 });
  const state = await controlledWeek(page, request);
  await page.clock.setSystemTime(new Date('2030-01-07T10:59:30Z'));
  const session = { id: crypto.randomUUID(), date: state.week, assignment: { goalId: goal.id, title: goal.title }, state: 'accepted', plan: { start: '2030-01-07T11:00:00Z', end: '2030-01-07T11:30:00Z' }, actual: null, exception: false };
  await page.route('**/api/scheduler/week?*', route => route.fulfill({ json: { ...state, sessions: [session] } }));
  await page.reload();
  await sessionFor(page, goal.title).locator('.scheduler-block-main').click();
  const dialog = page.getByRole('dialog', { name: 'Session editor' });
  await expect(dialog.getByRole('button', { name: 'Save session', exact: true })).toBeEnabled();
  await expect(dialog.getByRole('button', { name: 'Remove this session' })).toBeVisible();
  await dialog.getByLabel('Subgoal').selectOption(stepId);
  await dialog.getByLabel('Start time', { exact: true }).fill('05:10');
  await dialog.getByLabel('End time', { exact: true }).fill('05:40');
  const writes: { week: string; revision: string; action: string; session: { assignment: { goalId: string; title: string } }; actual: { start: string; end: string } }[] = [];
  await page.route('**/api/scheduler/mutate', async route => { const mutation = route.request().postDataJSON(); writes.push(mutation); await route.fulfill({ json: { ...state, sessions: [{ ...session, actual: mutation.actual }] } }); });
  await page.clock.fastForward(31000);
  await expect(dialog.getByRole('button', { name: 'Save session', exact: true })).toBeDisabled();
  await expect(dialog.getByRole('button', { name: 'Remove this session' })).toHaveCount(0);
  await expect(dialog).toContainText('This session has started');
  await dialog.evaluate(element => element.querySelector('form')!.requestSubmit());
  expect(writes).toHaveLength(0);
  await dialog.getByRole('button', { name: 'Record actual work', exact: true }).click();
  await expect(dialog.getByRole('heading', { name: 'Record actual work', exact: true })).toBeVisible();
  await expect(dialog).toContainText(`Actual work is assigned to ${goal.title}.`);
  await expect(dialog).toContainText('Original plan: 05:00–05:30');
  await expect(dialog.getByLabel('Start time', { exact: true })).toHaveValue('05:10');
  await expect(dialog.getByLabel('End time', { exact: true })).toHaveValue('05:40');
  await dialog.getByRole('button', { name: 'Save session', exact: true }).click();
  await expect(dialog.getByRole('alert')).toContainText('Actual work cannot end in the future');
  expect(writes).toHaveLength(0);
  await page.clock.fastForward(41 * 60000);
  await dialog.getByRole('button', { name: 'Save session', exact: true }).click();
  await expect(dialog).toBeHidden();
  expect(writes).toEqual([expect.objectContaining({ week: state.week, revision: state.revision, action: 'actual', actual: expect.objectContaining({ start: '2030-01-07T11:10:00.000Z', end: '2030-01-07T11:40:00.000Z' }) })]);
  expect(writes[0].session.assignment).toEqual(session.assignment);
  await sessionFor(page, goal.title).locator('.scheduler-block-main').click();
  await expect(dialog).toContainText('Original plan: 05:00–05:30');
  await expect(dialog.getByLabel('Start time', { exact: true })).toHaveValue('05:10');
  await expect(dialog.getByLabel('End time', { exact: true })).toHaveValue('05:40');
});

test('obsolete Google health errors cannot replace a newer successful status', async ({ page, request }) => {
  await controlledWeek(page, request);
  let finish: () => void = () => {};
  const gate = new Promise<void>(resolve => { finish = resolve; });
  let reads = 0;
  await page.route('**/api/scheduler/google/status', async route => {
    reads++;
    if (reads === 1) { await gate; await route.fulfill({ status: 503, json: { error: 'Obsolete status failure' } }); }
    else await route.fulfill({ json: { configured: true, connected: true, reconnectRequired: false, revision: 'new-health' } });
  });
  await page.reload();
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(page.locator('.scheduler-google summary')).toContainText('Connected');
  const obsolete = page.waitForResponse(response => response.url().endsWith('/google/status') && response.status() === 503);
  finish();
  await obsolete;
  await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
  await expect(page.locator('.scheduler-google [role="alert"]')).toHaveCount(0);
});

test('geometry keeps adjacent short blocks selectable and outside-hours actuals discoverable', async ({page,request}) => {
 const state=await controlledWeek(page,request);
 const date=state.week;
 const make=(id:string,start:string,end:string)=>({id,date,state:'accepted',assignment:{title:id},plan:null,actual:{date,status:'explicit',start:`${date}T${start}:00Z`,end:`${date}T${end}:00Z`},conflictIds:[],exception:false});
 const sessions=[make('short one','09:00','09:15'),make('short two','09:15','09:30'),make('early actual','03:00','04:00'),make('late actual','22:00','23:00')];
 await page.route('**/api/scheduler/week?*',route=>route.fulfill({json:{...state,settings:{...state.settings,timeZone:'UTC'},sessions}}));
 await page.getByRole('button',{name:'Refresh schedule',exact:true}).first().click();
 for(const id of ['short one','short two']) {
  const block=page.locator(`[data-session-id="${id}"]`); const box=await visibleBox(block); expect(box.height).toBe(15);
  await page.mouse.move(box.x+box.width/2,box.y+box.height/2); await page.mouse.down(); await page.mouse.up();
  await expect(page.getByRole('dialog',{name:'Session editor'})).toBeVisible();
  await page.getByRole('button',{name:'Discard draft'}).click();
 }
 await page.setViewportSize({width:390,height:844});
 const touch=await page.context().newCDPSession(page); await touch.send('Emulation.setTouchEmulationEnabled',{enabled:true,maxTouchPoints:1});
 for (const id of ['short one','short two']) {
  const block=page.locator(`[data-session-id="${id}"]`); const box=await visibleBox(block); expect(box.height).toBe(15);
  const topEdge=(await block.locator('.scheduler-edge-top').boundingBox())!, bottomEdge=(await block.locator('.scheduler-edge-bottom').boundingBox())!;
  expect(topEdge.height).toBe(2); expect(bottomEdge.height).toBe(2); expect(topEdge.y+topEdge.height).toBeLessThan(bottomEdge.y);
  await touch.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{x:box.x+box.width/2,y:box.y+box.height/2,id:1}]});
  await touch.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
  await expect(page.getByRole('dialog',{name:'Session editor'})).toBeVisible(); await page.getByRole('button',{name:'Discard draft'}).click();
 }
 await touch.detach(); await page.setViewportSize({width:1280,height:1400});
 for(const id of ['early actual','late actual']) { const block=page.locator(`[data-session-id="${id}"]`); await block.scrollIntoViewIfNeeded(); await expect(block).toBeInViewport(); expect((await block.boundingBox())!.height).toBe(60); }
});
test('invalid dated column remains visible and warning opens its affected editable session', async ({page,request})=>{
 const state=await controlledWeek(page,request);
 const session={id:'attention',date:state.week,state:'attention',assignment:{title:'Affected session'},plan:{start:`${state.week}T10:00:00Z`,end:`${state.week}T11:00:00Z`},actual:null,attention:'Needs placement',conflictIds:[],exception:false};
 const message='Affected session needs placement';
 await page.route('**/api/scheduler/week?*',route=>route.fulfill({json:{...state,settings:{...state.settings,timeZone:'UTC'},days:state.days.map((day:Record<string,unknown>,i:number)=>i===6?{...day,valid:false,reason:'02:30 does not exist because clocks change'}:day),sessions:[session],warnings:[message],warningTargets:[{message,date:session.date,sessionId:session.id}]}}));
 await page.getByRole('button',{name:'Refresh schedule',exact:true}).first().click();
 await expect(page.locator('[data-scheduler-date]')).toHaveCount(7); await expect(page.getByText('02:30 does not exist because clocks change')).toBeVisible();
 await page.getByRole('button',{name:message,exact:true}).click();
 await expect(page.getByRole('dialog',{name:'Session editor'})).toBeVisible(); await expect(page.getByLabel('Start time',{exact:true})).toHaveValue('10:00');
});
test('fold actual shows elapsed minutes and both UTC offsets', async ({page,request})=>{
 const state=await controlledWeek(page,request,'2026-10-26');
 const session={id:'fold',date:'2026-11-01',state:'accepted',assignment:{title:'Repeated hour'},plan:null,actual:{date:'2026-11-01',status:'explicit',start:'2026-11-01T06:30:00Z',end:'2026-11-01T07:30:00Z'},conflictIds:[],exception:false};
 await page.route('**/api/scheduler/week?*',route=>route.fulfill({json:{...state,settings:{...state.settings,timeZone:'America/Chicago'},sessions:[session,{...session,id:'fold-second',assignment:{title:'Later repeated hour'},actual:{...session.actual,start:'2026-11-01T07:30:00Z',end:'2026-11-01T08:30:00Z'}}]}}));
 await page.getByRole('button',{name:'Refresh schedule',exact:true}).first().click();
 const block=page.locator('[data-session-id="fold"]'); await block.scrollIntoViewIfNeeded(); expect((await block.boundingBox())!.height).toBe(60); await expect(block).toContainText('60 min · GMT-5–GMT-6');
 const second=page.locator('[data-session-id="fold-second"]'); const a=(await block.boundingBox())!, b=(await second.boundingBox())!; expect(a.x+a.width).toBeLessThanOrEqual(b.x);
 for (const item of [block,second]) { await item.locator('.scheduler-block-main').click(); await expect(page.getByRole('dialog',{name:'Session editor'})).toBeVisible(); await page.getByRole('button',{name:'Discard draft'}).click(); }
});
test('quick add opens a fitting selected-day draft or a no-slot explanation',async({page,request})=>{
 const state=await controlledWeek(page,request);
 const date=state.week;
 await page.route('**/api/scheduler/week?*',route=>route.fulfill({json:{...state,settings:{...state.settings,timeZone:'UTC'},days:state.days.map((day:Record<string,unknown>,i:number)=>({...day,valid:true,start:`${day.date}T08:00:00Z`,end:`${day.date}T08:30:00Z`,interval:{start:'08:00',end:'08:30',nextDay:false}})),sessions:[]}}));
 await page.getByRole('button',{name:'Refresh schedule',exact:true}).first().click();
 await page.getByRole('button',{name:`Add session on ${date}`,exact:true}).click();
 const dialog=page.getByRole('dialog',{name:'Session editor'}); await expect(dialog.getByLabel('Start time',{exact:true})).toHaveValue('08:01'); await expect(dialog.getByLabel('End time',{exact:true})).toHaveValue('08:30');
 await page.getByRole('button',{name:'Discard draft'}).click(); await page.clock.fastForward(31*60000);
 await page.getByRole('button',{name:`Add session on ${date}`,exact:true}).click(); await expect(dialog.getByText(/No valid future slot/)).toBeVisible(); await expect(dialog.getByLabel('Start time',{exact:true})).toHaveValue('');
});
