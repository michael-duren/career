import { test, expect, type APIRequestContext, type Page } from '@playwright/test';

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
    return state.sessions.find((s: { assignment: { goalId: string } }) => s.assignment.goalId === goal.id)?.date;
  }).toBe(targetDate);
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
  const dialog = page.locator('.scheduler-editor-dialog');

  await page.locator(`#scheduler-goal-${goal.id} .scheduler-goal-title`).click();
  await dialog.getByLabel('Scheduling date').fill(date!);
  await dialog.getByLabel('Start time').fill('17:00');
  await dialog.getByLabel('End time').fill('18:00');
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
  }).toEqual({ end: '19:00', minutes: 120 });
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
