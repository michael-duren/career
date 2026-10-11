import assert from 'node:assert/strict';
import test from 'node:test';
import { currentGoals, goalNoteBody, goalPace, journalLine, logSlot, upcomingGoals, weekTotals } from '../src/lib/goal-progress.ts';
import { buildLogDocs } from '../src/lib/logs.ts';
import type { SchedulerWeek } from '../src/lib/scheduler.ts';
import type { Goal } from '../src/lib/timeline.ts';

const goal = (id: string, startDate: string, endDate: string, status: Goal['status'] = 'planned'): Goal => ({
  id, title: id, status, dependsOn: [], startDate, endDate, color: '#fff', dailyHours: 1, createdAt: '', updatedAt: '', notes: [], metadata: {},
  steps: [{ id: `${id}-a`, title: 'A', done: true }, { id: `${id}-b`, title: 'B', done: false }],
});

test('current goals are open goals covering today or marked active, soonest end first', () => {
  const goals = [goal('late', '2026-10-01', '2026-12-31'), goal('soon', '2026-09-01', '2026-10-20'), goal('done', '2026-10-01', '2026-10-30', 'done'),
    goal('active-early', '2026-10-20', '2026-11-01', 'active'), goal('future', '2026-10-15', '2026-11-01'), goal('far', '2027-01-01', '2027-02-01')];
  assert.deepEqual(currentGoals(goals, '2026-10-11').map(g => g.id), ['soon', 'active-early', 'late']);
  assert.deepEqual(upcomingGoals(goals, '2026-10-11').map(g => g.id), ['future']);
});

test('goal pace compares logged hours with hours expected through today', () => {
  const pace = goalPace(goal('g', '2026-10-05', '2026-10-18'), { totalHours: 5, weekHours: { '2026-10-05': 5 } }, '2026-10-07', '2026-10-05');
  assert.deepEqual(pace, { logged: 5, thisWeek: 5, expectedToDate: 3, plannedTotal: 14, stepsDone: 1, stepsTotal: 2 });
  assert.equal(goalPace({ ...goal('u', '2026-10-05', '2026-10-18'), dailyHours: undefined }, undefined, '2026-10-07', '2026-10-05').expectedToDate, undefined);
  assert.equal(goalPace(goal('f', '2026-11-01', '2026-11-18'), undefined, '2026-10-07', '2026-10-05').expectedToDate, 0);
});

const week = (sessions: Partial<SchedulerWeek['sessions'][number]>[]): SchedulerWeek => ({
  revision: 'r', week: '2026-10-05', settings: { timeZone: 'UTC', defaultDay: { start: '06:00', end: '22:00', nextDay: false }, weekdays: {}, dates: {} },
  days: [{ date: '2026-10-07', interval: { start: '06:00', end: '22:00', nextDay: false }, start: '2026-10-07T06:00:00Z', end: '2026-10-07T22:00:00Z', valid: true }],
  goals: [], rules: [], busy: [], warnings: [], warningTargets: [], remainingCapacityHours: 0,
  sessions: sessions.map((s, i) => ({ id: String(i), date: '2026-10-07', assignment: { title: 'x' }, plan: null, actual: null, state: 'accepted', conflictIds: [], exception: false, ...s })),
});

test('log slot ends at now and steps back over recorded work but not plans or skipped work', () => {
  const now = new Date('2026-10-07T12:10:30Z');
  assert.deepEqual(logSlot(week([]), '2026-10-07', now, 60), { start: '2026-10-07T11:10:00.000Z', end: '2026-10-07T12:10:00.000Z' });
  const busy = week([
    { actual: { status: 'explicit', date: '2026-10-07', start: '2026-10-07T11:30:00Z', end: '2026-10-07T12:30:00Z' } },
    { actual: { status: 'skipped', date: '2026-10-07', start: '2026-10-07T10:00:00Z', end: '2026-10-07T11:30:00Z' } },
    { plan: { start: '2026-10-07T10:00:00Z', end: '2026-10-07T11:30:00Z' } },
  ]);
  assert.deepEqual(logSlot(busy, '2026-10-07', now, 60), { start: '2026-10-07T10:30:00.000Z', end: '2026-10-07T11:30:00.000Z' });
  assert.deepEqual(logSlot(week([]), '2026-10-07', new Date('2026-10-09T00:00:00Z'), 30), { start: '2026-10-07T21:30:00.000Z', end: '2026-10-07T22:00:00.000Z' });
  assert.equal(logSlot(week([]), '2026-10-07', new Date('2026-10-07T06:30:00Z'), 60), null);
  assert.equal(logSlot(week([]), '2026-10-08', now, 60), null);
});

test('journal lines carry the goal as a Logs facet and goal notes carry the date', () => {
  const line = journalLine({ goalTitle: 'OSTEP', stepTitle: 'Semaphores', hours: 1.5, note: ' Finished the chapter ' });
  assert.equal(line, 'Goal: OSTEP › Semaphores (1.5h) — Finished the chapter');
  assert.equal(journalLine({ hours: 0, note: ' plain ' }), 'plain');
  assert.equal(goalNoteBody({ date: '2026-10-07', hours: 1.25, note: 'Done' }), '2026-10-07 · 1.25h logged — Done');
  assert.equal(goalNoteBody({ date: '2026-10-07', stepTitle: 'Semaphores', hours: 0, note: 'Stuck' }), '2026-10-07 · Semaphores — Stuck');
  const [doc] = buildLogDocs([{ week: 25, year: 2026, dates: '2026-10-05 to 2026-10-11', slug: '2026/week-25', hours: {}, tags: [], body: `## What I did\n\n### Wednesday — Oct 7\n${line}\n` }]);
  assert.deepEqual(doc.parts, [{ label: 'Goal', text: 'OSTEP › Semaphores (1.5h) — Finished the chapter' }]);
});

test('week totals add required, recorded, scheduled and uncovered hours across goals', () => {
  const w = week([]);
  const summary = (requiredHours: number | null, actualHours: number, remainingScheduledHours: number, uncoveredHours: number) =>
    ({ goal: { id: 'g', title: 'g', color: '', startDate: '', endDate: '', status: '', dailyHours: null, selectedWeekdays: null, pauses: [], steps: [], dependsOn: [] }, requiredHours, actualHours, remainingScheduledHours, uncoveredHours, excessHours: 0, unscheduledStepIds: [] });
  w.goals = [summary(7, 3, 2, 2), summary(null, 1, 0, 0)];
  assert.deepEqual(weekTotals(w), { required: 7, actual: 4, scheduled: 2, uncovered: 2 });
});

test('progress weeks gain scheduler goal hours, including weeks without a journal entry', async () => {
  const { buildDashboard, withGoalHours } = await import('../src/lib/progress.ts');
  const journal = [{ week: 25, year: 2026, dates: '2026-10-05 to 2026-10-11', slug: '2026/week-25', body: '', tags: [], hours: { ostep: 2 } }];
  const tracks = [{ id: 'g', title: 'OSTEP goal', color: '#67e8f9', weekHours: { '2026-10-05': 3, '2026-10-12': 1.5, '2026-01-05': 9 } }];
  const weeks = withGoalHours(journal, tracks);
  assert.deepEqual(journal[0].hours, { ostep: 2 });
  assert.deepEqual(weeks.map(w => [w.week, w.dates, w.hours]), [
    [25, '2026-10-05 to 2026-10-11', { ostep: 2, 'goal:g': 3 }],
    [26, '2026-10-12 to 2026-10-18', { 'goal:g': 1.5 }],
  ]);
  const dashboard = buildDashboard(weeks, new Date('2026-10-14T12:00:00Z'), tracks);
  const goal = dashboard.tracks.find(track => track.key === 'goal:g');
  assert.equal(goal?.name, 'OSTEP goal');
  assert.equal(goal?.current, 1.5);
  assert.equal(dashboard.totalHours, 6.5);
});

test('with goal tracks, empty journal tracks drop out and used ones are labelled as journal history', async () => {
  const { buildDashboard } = await import('../src/lib/progress.ts');
  const weeks = [{ week: 25, year: 2026, dates: '2026-10-05 to 2026-10-11', slug: 'w', body: '', tags: [], hours: { ostep: 2, 'goal:g': 1 } }];
  const names = buildDashboard(weeks, new Date('2026-10-08T12:00:00Z'), [{ id: 'g', title: 'OSTEP', color: '#fff', weekHours: {} }]).tracks.map(t => t.name);
  assert.deepEqual(names, ['OSTEP (journal)', 'OSTEP']);
  assert.deepEqual(buildDashboard(weeks, new Date('2026-10-08T12:00:00Z')).tracks.map(t => t.key).slice(0, 4), ['ostep', 'ebpf', 'database', 'systemDesign']);
});

test('save steps resume after a partial failure without repeating saved steps', async () => {
  const { runSaveSteps } = await import('../src/lib/goal-progress.ts');
  const calls: string[] = [];
  let failNote = true;
  const steps = [
    { key: 'hours', label: '1h logged', run: async () => { calls.push('hours'); } },
    { key: 'goal', label: 'note added', run: async () => { calls.push('goal'); if (failNote) throw new Error('conflict'); } },
    { key: 'journal', label: 'journal updated', run: async () => { calls.push('journal'); } },
  ];
  const done = new Map<string, string>();
  await assert.rejects(runSaveSteps(steps, done), /conflict/);
  assert.deepEqual([...done.keys()], ['hours']);
  failNote = false;
  await runSaveSteps(steps, done);
  assert.deepEqual(calls, ['hours', 'goal', 'goal', 'journal']);
  assert.deepEqual([...done.values()], ['1h logged', 'note added', 'journal updated']);
});
