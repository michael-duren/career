import assert from 'node:assert/strict';
import test from 'node:test';
import { durationLabel, isoWeek, sessionMinutes, nextSession, progressNote, streakScale, STREAK_TICKS, windowHours } from '../src/lib/home.ts';
import type { SchedulerSession } from '../src/lib/scheduler.ts';

const session = (id: string, start: string, end: string, actual: SchedulerSession['actual'] = null): SchedulerSession => ({
  id, date: '2026-11-24', assignment: { goalId: 'g', title: id }, plan: { start, end } as SchedulerSession['plan'], actual, state: 'accepted', conflictIds: [], exception: false,
});
const at = (time: string) => Date.parse(`2026-11-24T${time}:00Z`);
const day = [
  session('a', '2026-11-24T07:15:00Z', '2026-11-24T08:45:00Z', { status: 'assumed' } as SchedulerSession['actual']),
  session('b', '2026-11-24T17:45:00Z', '2026-11-24T18:45:00Z'),
  session('c', '2026-11-24T19:00:00Z', '2026-11-24T20:00:00Z'),
];

test('the next session is the first unchecked one that has not ended', () => {
  assert.equal(nextSession(day, at('07:00'))?.id, 'b');
  assert.equal(nextSession(day, at('18:00'))?.id, 'b');
  assert.equal(nextSession(day, at('18:45'))?.id, 'c');
  assert.equal(nextSession(day, at('20:00')), undefined);
  const skipped = [session('s', '2026-11-24T17:45:00Z', '2026-11-24T18:45:00Z', { status: 'skipped' } as SchedulerSession['actual']), day[2]];
  assert.equal(nextSession(skipped, at('17:00'))?.id, 'c');
});

test('the note counts done sessions and says when the next one starts', () => {
  assert.equal(progressNote(day, at('17:20'), 'UTC'), '1 of 3 done — next in 25 min');
  assert.equal(progressNote(day, at('12:00'), 'UTC'), '1 of 3 done — next at 5:45 PM');
  assert.equal(progressNote(day, at('18:00'), 'UTC'), '1 of 3 done — now until 6:45 PM');
  assert.equal(progressNote(day, at('17:45'), 'America/Chicago'), '1 of 3 done — now until 12:45 PM');
  assert.equal(progressNote(day, at('21:00'), 'UTC'), '1 of 3 done');
  assert.equal(progressNote(day, at('16:45'), 'UTC'), '1 of 3 done — next at 5:45 PM');
  assert.equal(progressNote(day, at('16:46'), 'UTC'), '1 of 3 done — next in 59 min');
  assert.equal(progressNote(day, Date.parse('2026-11-24T17:44:30Z'), 'UTC'), '1 of 3 done — next in 1 min');
  assert.equal(progressNote([], at('12:00'), 'UTC'), '0 of 0 done');
});

test('durations read in minutes under an hour, else hours', () => {
  assert.equal(durationLabel(45), '45 MIN');
  assert.equal(durationLabel(60), '1 H');
  assert.equal(durationLabel(90), '1.5 H');
  assert.equal(durationLabel(135), '2.25 H');
  assert.equal(sessionMinutes(day[0]), 90);
  assert.equal(sessionMinutes({ ...day[0], plan: null }), 0);
  assert.equal(windowHours('2026-11-24T05:00:00Z', '2026-11-24T20:30:00Z'), '15.5');
});

test('ISO week numbers', () => {
  assert.equal(isoWeek('2026-11-24'), 48);
  assert.equal(isoWeek('2026-01-01'), 1);
  assert.equal(isoWeek('2027-01-01'), 53);
  assert.equal(isoWeek('2024-12-30'), 1);
  assert.equal(isoWeek('2021-01-03'), 53);
});

test('the streak scale runs to the record and scales down past the cap', () => {
  assert.deepEqual(streakScale(2, 29), { count: 29, filled: 2, max: 29 });
  assert.deepEqual(streakScale(0, 0), { count: 1, filled: 0, max: 1 });
  assert.deepEqual(streakScale(31, 29), { count: 31, filled: 31, max: 31 });
  assert.deepEqual(streakScale(60, 120), { count: STREAK_TICKS, filled: 30, max: 120 });
});
