import assert from 'node:assert/strict';
import test from 'node:test';
import { formatDate, formatNumericDate, formatTime, formatTimeRange, formatWeekdayDate } from '../src/lib/format.ts';

test('times are 12-hour with no leading zero and a space before AM/PM', () => {
  assert.equal(formatTime(7 * 60 + 15), '7:15 AM');
  assert.equal(formatTime(17 * 60 + 45), '5:45 PM');
  assert.equal(formatTime(11 * 60 + 59), '11:59 AM');
  assert.equal(formatTime(23 * 60 + 5), '11:05 PM');
});

test('midnight and noon are 12 AM and 12 PM', () => {
  assert.equal(formatTime(0), '12:00 AM');
  assert.equal(formatTime(12 * 60), '12:00 PM');
  assert.equal(formatTime(30), '12:30 AM');
  assert.equal(formatTime(12 * 60 + 30), '12:30 PM');
  assert.equal(formatTime(0, { compact: true }), '12 AM');
  assert.equal(formatTime(12 * 60, { compact: true }), '12 PM');
});

test('compact drops :00 only when minutes are zero', () => {
  assert.equal(formatTime(9 * 60), '9:00 AM');
  assert.equal(formatTime(9 * 60, { compact: true }), '9 AM');
  assert.equal(formatTime(9 * 60 + 5, { compact: true }), '9:05 AM');
});

test('minutes wrap into one day', () => {
  assert.equal(formatTime(1440), '12:00 AM');
  assert.equal(formatTime(1440 + 90), '1:30 AM');
  assert.equal(formatTime(-15), '11:45 PM');
  assert.throws(() => formatTime(Number.NaN));
});

test('instants format in the given time zone', () => {
  assert.equal(formatTime('2026-11-24T17:20:00Z', { timeZone: 'UTC' }), '5:20 PM');
  assert.equal(formatTime(new Date('2026-11-24T00:00:00Z'), { timeZone: 'UTC' }), '12:00 AM');
  assert.equal(formatTime('2026-11-24T12:00:00Z', { timeZone: 'UTC', compact: true }), '12 PM');
  assert.equal(formatTime('2026-11-24T17:20:00Z', { timeZone: 'America/Chicago' }), '11:20 AM');
  assert.throws(() => formatTime('not a time'));
});

test('ranges use a spaced en dash', () => {
  assert.equal(formatTimeRange(7 * 60 + 15, 8 * 60 + 45), '7:15 AM – 8:45 AM');
  assert.equal(formatTimeRange(9 * 60, 16 * 60, { compact: true }), '9 AM – 4 PM');
  assert.equal(formatTimeRange(9 * 60, 16 * 60 + 30, { compact: true }), '9 AM – 4:30 PM');
  assert.equal(formatTimeRange(23 * 60, 24 * 60, { compact: true }), '11 PM – 12 AM');
});

test('dates', () => {
  assert.equal(formatDate('2026-11-24'), 'Nov 24');
  assert.equal(formatDate('2026-01-05'), 'Jan 5');
  assert.equal(formatWeekdayDate('2026-11-24'), 'Tue 24 Nov');
  assert.equal(formatNumericDate('2026-11-24'), '24.11.26');
  assert.equal(formatNumericDate('2030-03-07'), '07.03.30');
  assert.equal(formatDate('2026-11-24T03:00:00Z', 'America/Chicago'), 'Nov 23');
  assert.equal(formatWeekdayDate(new Date('2026-11-24T12:00:00Z'), 'UTC'), 'Tue 24 Nov');
  assert.throws(() => formatDate('2026-02-30'));
  assert.throws(() => formatDate('nope'));
});
