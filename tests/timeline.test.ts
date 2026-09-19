import assert from 'node:assert/strict';
import test from 'node:test';
import { dayNumber, monthOffset, shiftGoal, timelineHours, type Goal } from '../src/lib/timeline.ts';
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
