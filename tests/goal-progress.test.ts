import assert from 'node:assert/strict';
import test from 'node:test';
import { goalPace } from '../src/lib/goal-progress.ts';
import type { Goal } from '../src/lib/timeline.ts';

const goal = (id: string, startDate: string, endDate: string, status: Goal['status'] = 'planned'): Goal => ({
  id, title: id, status, dependsOn: [], startDate, endDate, color: '#fff', dailyHours: 1, createdAt: '', updatedAt: '', notes: [], metadata: {},
  steps: [{ id: `${id}-a`, title: 'A', done: true }, { id: `${id}-b`, title: 'B', done: false }],
});

test('goal pace compares logged hours with hours expected through today', () => {
  const pace = goalPace(goal('g', '2026-10-05', '2026-10-18'), { totalHours: 5, weekHours: { '2026-10-05': 5 } }, '2026-10-07', '2026-10-05');
  assert.deepEqual(pace, { logged: 5, thisWeek: 5, expectedToDate: 3, plannedTotal: 14, stepsDone: 1, stepsTotal: 2 });
  assert.equal(goalPace({ ...goal('u', '2026-10-05', '2026-10-18'), dailyHours: undefined }, undefined, '2026-10-07', '2026-10-05').expectedToDate, undefined);
  assert.equal(goalPace(goal('f', '2026-11-01', '2026-11-18'), undefined, '2026-10-07', '2026-10-05').expectedToDate, 0);
});
