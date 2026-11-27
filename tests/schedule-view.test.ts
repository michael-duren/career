import assert from 'node:assert/strict';
import test from 'node:test';
import { blockState, dayParts, goalHours, goalPatterns, hoursCell, isoWeek, weekRangeLabel } from '../src/lib/schedule-view.ts';
import { displayClock } from '../src/lib/scheduler.ts';

test('isoWeek follows ISO 8601 across year boundaries', () => {
  assert.equal(isoWeek('2026-11-23'), 48);
  assert.equal(isoWeek('2026-12-28'), 53);
  assert.equal(isoWeek('2027-01-03'), 53);
  assert.equal(isoWeek('2027-01-04'), 1);
  assert.equal(isoWeek('2024-12-30'), 1);
});
test('weekRangeLabel names one month, or both when the week crosses months or years', () => {
  assert.equal(weekRangeLabel('2026-11-23'), '23 — 29 November');
  assert.equal(weekRangeLabel('2026-11-30'), '30 November — 6 December');
  assert.equal(weekRangeLabel('2026-12-28', true), '28 Dec — 3 Jan');
  assert.equal(weekRangeLabel('2026-11-23', true), '23 — 29 Nov');
});
test('dayParts gives the column label pieces', () => {
  assert.deepEqual(dayParts('2026-11-26'), { dow: 'THU', num: 26 });
});
test('goal patterns follow week order and fall back to an outline after the third goal', () => {
  const goals = ['a', 'b', 'c', 'd'].map(id => ({ goal: { id } })) as never;
  const patterns = goalPatterns({ goals });
  assert.deepEqual([...patterns.values()], ['diagonal', 'crosshatch', 'dots', 'outline']);
  assert.equal(patterns.get('missing'), undefined);
});
test('blockState distinguishes plans, recorded work and skips', () => {
  assert.equal(blockState({ actual: null }), 'planned');
  assert.equal(blockState({ actual: { status: 'assumed', date: '', start: '', end: '' } }), 'done');
  assert.equal(blockState({ actual: { status: 'explicit', date: '', start: '', end: '' } }), 'done');
  assert.equal(blockState({ actual: { status: 'skipped', date: '', start: '', end: '' } }), 'skipped');
});
test('goalHours plans recorded plus scheduled hours against the requirement', () => {
  assert.deepEqual(goalHours({ requiredHours: 10.5, actualHours: 3, remainingScheduledHours: 2, uncoveredHours: 5.5 }), { required: 10.5, planned: 5, delta: -5.5, short: true });
  assert.deepEqual(goalHours({ requiredHours: 4, actualHours: 1, remainingScheduledHours: 5, uncoveredHours: 0 }), { required: 4, planned: 6, delta: 2, short: false });
  assert.deepEqual(goalHours({ requiredHours: null, actualHours: 0, remainingScheduledHours: 1, uncoveredHours: 0 }), { required: null, planned: 1, delta: null, short: false });
});
test('hoursCell shows one decimal, signs deltas and dashes missing values', () => {
  assert.equal(hoursCell(10.5), '10.5');
  assert.equal(hoursCell(4), '4.0');
  assert.equal(hoursCell(5.5, true), '+5.5');
  assert.equal(hoursCell(-1.5, true), '−1.5');
  assert.equal(hoursCell(0.01, true), '0.0');
  assert.equal(hoursCell(null), '—');
});
test('displayClock compact drops :00 on the hour for axis labels', () => {
  assert.equal(displayClock(6 * 60, true), '6 AM');
  assert.equal(displayClock(5 * 60 + 30, true), '5:30 AM');
  assert.equal(displayClock(1440 + 60, true), '1 AM +1d');
});
