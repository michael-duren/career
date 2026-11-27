import assert from 'node:assert/strict';
import test from 'node:test';
import { dayNumber, fitRange, goalProgress, goalState, goalTag, monthLabel, monthOffset, shiftGoal, timelineCounts, timelineHours, type Goal } from '../src/lib/timeline.ts';
import { updateWorkspaceEntry, type Workspace } from '../src/lib/workspace.ts';
const goal: Goal = { status: 'planned', dependsOn: [],
  id: '123', title: 'Systems', startDate: '2028-02-01', endDate: '2028-04-30', color: '#67e8f9',
  createdAt: '2026-09-12T00:00:00.000Z', updatedAt: '2026-09-12T00:00:00.000Z',
  notes: [{ id: 'note', body: 'Keep this', createdAt: '2026-09-12T00:00:00.000Z' }],
  metadata: { focus: 'OS' }, steps: [{ id: 'step', title: 'Memory', done: false }],
};
test('moving preserves duration and associated goal data across leap days', () => {
  const moved = shiftGoal(goal, 29, 'move');
  assert.equal(moved.startDate, '2028-03-01');
  assert.equal(dayNumber(moved.endDate) - dayNumber(moved.startDate), dayNumber(goal.endDate) - dayNumber(goal.startDate));
  assert.deepEqual(moved.notes, goal.notes);
  assert.deepEqual(moved.metadata, goal.metadata);
  assert.deepEqual(moved.steps, goal.steps);
  assert.deepEqual(shiftGoal(moved, -29, 'move'), goal);
});
test('resizing cannot invert dates and allows a one-day goal', () => {
  assert.equal(shiftGoal(goal, 999, 'start').startDate, goal.endDate);
  assert.equal(shiftGoal(goal, -999, 'end').endDate, goal.startDate);
  assert.equal(shiftGoal(goal, -10, 'start').endDate, goal.endDate);
  assert.equal(shiftGoal(goal, 10, 'end').startDate, goal.startDate);
});
test('calendar month spans clamp month ends and cross years', () => {
  assert.equal(monthOffset('2028-01-31', 1), '2028-02-29');
  assert.equal(monthOffset('2026-01-31', 1), '2026-02-28');
  for (const span of [6, 12, 24, 60, 120]) assert.ok(dayNumber(monthOffset('2026-09-01', span)) > dayNumber('2026-09-01'));
  assert.equal(monthOffset('2026-09-01', 120), '2036-09-01');
});
test('other workspace edits preserve goals', () => {
  const workspace: Workspace = { version: 2, notes: [], weeks: [], books: [], companies: [], documents: [], goals: [goal] };
  const changed = updateWorkspaceEntry(workspace, 'note', 'new', { id: 'new', title: 'A note', topic: 'OS', description: '', tags: [], body: 'Text' });
  assert.deepEqual(changed.goals, [goal]);
  assert.equal(changed.notes.length, 1);
});

test('workload counts inclusive overlap and excludes the saved version of the draft', async () => {
  const { workloadFor } = await import('../src/lib/timeline.ts');
  const draft = { ...goal, startDate: '2026-09-01', endDate: '2026-09-03', dailyHours: 2 };
  const result = workloadFor([draft, { ...draft, id: 'other', startDate: '2026-09-03', endDate: '2026-09-05', dailyHours: 1.5 }, { ...draft, id: 'later', startDate: '2026-09-04', endDate: '2026-09-05' }], draft);
  assert.equal(result.others.length, 1);
  assert.equal(result.peakHours, 3.5);
  assert.equal(result.peakCount, 2);
  assert.deepEqual(result.segments.map(s => [s.count, s.hours, s.end - s.start]), [[1, 2, 2], [2, 3.5, 1]]);
});
test('workload retains unknown estimates and zero-hour estimates distinctly', async () => {
  const { workloadFor } = await import('../src/lib/timeline.ts');
  const result = workloadFor([{ ...goal, id: 'unknown' }], { ...goal, dailyHours: 0 });
  assert.equal(result.segments[0].unknown, 1);
  assert.equal(result.peakHours, 0);
  assert.equal(result.peakCount, 2);
});
test('workload splits at boundaries instead of allocating a day for every date', async () => {
  const { workloadFor } = await import('../src/lib/timeline.ts');
  assert.equal(workloadFor([], { ...goal, startDate: '1900-01-01', endDate: '2200-12-31', dailyHours: 1 }).segments.length, 1);
  assert.equal(shiftGoal({ ...goal, dailyHours: 2.5 }, 10, 'move').dailyHours, 2.5);
});

test('workload stays bounded by weekday patterns across centuries and many goals', async () => {
  const { workloadFor } = await import('../src/lib/timeline.ts');
  const draft = { ...goal, startDate: '1900-01-01', endDate: '2200-12-31', dailyHours: 1, selectedWeekdays: [1, 3, 5] };
  const goals = Array.from({ length: 100 }, (_, index) => ({ ...draft, id: `goal-${index}` }));
  assert.ok(workloadFor(goals, { ...draft, id: 'draft' }).segments.length <= 7);
});

test('workload groups selected and unselected weekdays without expanding dates', async () => {
  const { workloadFor } = await import('../src/lib/timeline.ts');
  const result = workloadFor([], { ...goal, startDate: '2026-09-07', endDate: '2026-09-13', dailyHours: 2, selectedWeekdays: [1, 3, 5] });
  assert.deepEqual(result.segments.map(segment => ({ weekdays: segment.weekdays, count: segment.count, hours: segment.hours })), [
    { weekdays: [1, 3, 5], count: 1, hours: 2 },
    { weekdays: [2, 4, 6, 7], count: 0, hours: 0 },
  ]);
});


test('timeline daily average clips goals to the view, adds overlapping hours and includes empty days', () => {
  const start = dayNumber('2026-09-01'), end = dayNumber('2026-09-11');
  const result = timelineHours([
    { ...goal, startDate: '2026-08-20', endDate: '2026-09-03', dailyHours: 2 },
    { ...goal, id: 'second', startDate: '2026-09-03', endDate: '2026-09-05', dailyHours: 4 },
    { ...goal, id: 'outside', startDate: '2026-09-11', endDate: '2026-09-30', dailyHours: 8 },
  ], start, end);
  assert.deepEqual(result, { total: 18, average: 1.8, unknown: 0 });
  assert.deepEqual(timelineHours([], start, end), { total: 0, average: 0, unknown: 0 });
});
test('timeline hours mark unestimated visible goals without treating zero-hour goals as unknown', () => {
  const start = dayNumber(goal.startDate), end = dayNumber(goal.endDate) + 1;
  assert.deepEqual(timelineHours([goal, { ...goal, id: 'zero', dailyHours: 0 }], start, end), { total: 0, average: 0, unknown: 1 });
});

test('timeline hours default missing weekdays to every day', () => {
  const start = dayNumber('2026-09-07'), end = dayNumber('2026-09-14');
  assert.deepEqual(timelineHours([{ ...goal, startDate: '2026-09-07', endDate: '2026-09-13', dailyHours: 2 }], start, end),
    { total: 14, average: 2, unknown: 0 });
});

test('timeline hours count only selected weekdays and allow an empty selection', () => {
  const start = dayNumber('2026-09-07'), end = dayNumber('2026-09-14');
  assert.deepEqual(timelineHours([
    { ...goal, startDate: '2026-09-07', endDate: '2026-09-13', dailyHours: 2, selectedWeekdays: [1, 3, 5] },
    { ...goal, id: 'none', startDate: '2026-09-07', endDate: '2026-09-13', dailyHours: 8, selectedWeekdays: [] },
  ], start, end), { total: 6, average: 6 / 7, unknown: 0 });
});

test('timeline hours clip selected weekdays to partial goal ranges', () => {
  const start = dayNumber('2026-09-07'), end = dayNumber('2026-09-14');
  assert.deepEqual(timelineHours([
    { ...goal, startDate: '2026-09-10', endDate: '2026-09-13', dailyHours: 1, selectedWeekdays: [1, 4, 7] },
  ], start, end), { total: 2, average: 2 / 7, unknown: 0 });
});

test('timeline calculations reject invalid and duplicate ISO weekdays', () => {
  const start = dayNumber('2026-09-07'), end = dayNumber('2026-09-14');
  for (const selectedWeekdays of [[0], [8], [1.5], [1, 1]]) {
    assert.throws(() => timelineHours([{ ...goal, startDate: '2026-09-07', endDate: '2026-09-13', dailyHours: 1, selectedWeekdays }], start, end),
      /selectedWeekdays/);
  }
});

test('late means neither done nor dropped, ending before today, with steps still open', () => {
  const today = '2028-05-01';
  assert.equal(goalState({ ...goal, status: 'active' }, today), 'late');
  assert.equal(goalState({ ...goal, status: 'planned' }, today), 'late');
  assert.equal(goalState({ ...goal, status: 'done' }, today), 'done');
  assert.equal(goalState({ ...goal, status: 'dropped' }, today), 'dropped');
  // Ending today is not late yet.
  assert.equal(goalState({ ...goal, status: 'active' }, goal.endDate), 'active');
  assert.equal(goalState(goal, '2028-03-01'), 'planned');
  // All steps done: not late, matching Home. No steps: late.
  assert.equal(goalState({ ...goal, status: 'active', steps: [{ id: 's', title: 'S', done: true }] }, today), 'active');
  assert.equal(goalState({ ...goal, status: 'active', steps: [] }, today), 'late');
  assert.equal(goalTag({ ...goal, status: 'dropped' }, today), 'dropped');
  assert.equal(goalTag({ ...goal, status: 'done', steps: [] }, today), 'done');
});
test('tags and progress come from status and steps', () => {
  const steps = [{ id: 'a', title: 'A', done: true }, { id: 'b', title: 'B', done: false }, { id: 'c', title: 'C', done: false }];
  assert.equal(goalTag({ ...goal, steps }, '2028-03-01'), '1/3');
  assert.equal(goalTag({ ...goal, steps: [] }, '2028-03-01'), '');
  assert.equal(goalTag({ ...goal, steps }, '2028-06-01'), 'late');
  assert.equal(goalTag({ ...goal, status: 'done', steps }, '2028-06-01'), 'done');
  assert.equal(goalProgress({ ...goal, steps }), 1 / 3);
  assert.equal(goalProgress({ ...goal, steps: [] }), 0);
  assert.equal(goalProgress({ ...goal, status: 'done', steps }), 1);
});
test('header counts count late goals under their status too and skip dropped ones', () => {
  const goals: Goal[] = [
    { ...goal, id: 'a', status: 'active' }, { ...goal, id: 'b', status: 'active', endDate: '2028-12-31' },
    { ...goal, id: 'c', status: 'planned', endDate: '2028-12-31' }, { ...goal, id: 'd', status: 'done' }, { ...goal, id: 'e', status: 'dropped' },
  ];
  assert.deepEqual(timelineCounts(goals, '2028-06-01'), { active: 2, planned: 1, done: 1, late: 1 });
});
test('fit range covers every goal month, at least six months, across years', () => {
  const goals: Goal[] = [{ ...goal, startDate: '2026-09-26', endDate: '2027-01-09' }, { ...goal, id: 'x', startDate: '2027-01-04', endDate: '2027-05-04' }];
  assert.deepEqual(fitRange(goals, '2026-11-26'), { anchor: '2026-09-01', months: 9 });
  assert.deepEqual(fitRange([goal], '2026-11-26'), { anchor: '2028-02-01', months: 6 });
  assert.deepEqual(fitRange([], '2026-11-26'), { anchor: '2026-11-01', months: 6 });
  const { anchor, months } = fitRange(goals, '2026-11-26');
  assert.ok(monthOffset(anchor, months) > '2027-05-04');
  assert.equal(monthLabel(anchor), 'Sep 2026');
  // Unsorted, with a middle goal ending last on the 31st, and a long single goal.
  const unsorted: Goal[] = [{ ...goal, id: 'b', startDate: '2027-02-01', endDate: '2027-03-01' }, { ...goal, id: 'c', startDate: '2026-12-10', endDate: '2027-08-31' }, { ...goal, id: 'a', startDate: '2026-11-05', endDate: '2026-12-01' }];
  assert.deepEqual(fitRange(unsorted, '2026-11-26'), { anchor: '2026-11-01', months: 10 });
  assert.deepEqual(fitRange([{ ...goal, startDate: '2026-01-15', endDate: '2027-03-02' }], '2026-11-26'), { anchor: '2026-01-01', months: 15 });
});
