import assert from 'node:assert/strict';
import test from 'node:test';
import { addDays, mondayOf, snapMinutes, zonedDate, localInstant, localFields, clockLabel } from '../src/lib/scheduler.ts';

test('Monday weeks cross month/year and leap-day boundaries without browser timezone', () => {
  assert.equal(mondayOf('2027-01-03'), '2026-12-28');
  assert.equal(mondayOf('2028-03-01'), '2028-02-28');
  assert.equal(addDays('2028-02-28', 2), '2028-03-01');
});
test('drag positions snap to quarter hours while labels preserve manual minute precision', () => {
  assert.equal(snapMinutes(68), 75);
  assert.equal(snapMinutes(66), 60);
  assert.equal(clockLabel(1505), '01:05 +1d');
});
test('calendar today follows saved scheduler timezone', () => {
  assert.equal(zonedDate(new Date('2026-09-28T02:00:00Z'), 'America/Chicago'), '2026-09-27');
});
test('local editor resolves repeated time early and rejects spring gaps', () => {
  assert.equal(localInstant('2026-11-01', '01:30', 'America/Chicago'), '2026-11-01T06:30:00.000Z');
  assert.throws(() => localInstant('2026-03-08', '02:30', 'America/Chicago'), /does not exist/);
  assert.equal(localInstant('2026-09-28', '05:17', 'Asia/Kolkata'), '2026-09-27T23:47:00.000Z');
  assert.deepEqual(localFields('2026-09-27T23:47:00Z', 'Asia/Kolkata'), { date: '2026-09-28', time: '05:17', offset: 'GMT+5:30' });
});

test('response parsing normalizes server null collections and rejects malformed revisions', async () => {
  const { weekSchema } = await import('../src/lib/scheduler.ts');
  const value = { revision: 'r1', week: '2026-09-28', settings: { timeZone: 'UTC', defaultDay: { start: '05:00', end: '20:30', nextDay: false }, weekdays: null, dates: null }, days: [], goals: null, sessions: null, rules: null, busy: null, warnings: null, remainingCapacityHours: 0 };
  assert.deepEqual(weekSchema.parse(value).sessions, []);
  assert.equal(weekSchema.safeParse({ ...value, revision: 5 }).success, false);
});
test('editor requests preserve revision, minute precision, actual replacement and recurrence scope', async () => {
  const { weekSchema, draftMutation } = await import('../src/lib/scheduler.ts');
  const week = weekSchema.parse({ revision: 'r1', week: '2026-09-28', settings: { timeZone: 'America/Chicago', defaultDay: { start: '05:00', end: '20:30', nextDay: false }, weekdays: {}, dates: {} }, days: [], goals: [], sessions: [], rules: [], busy: [], warnings: [], remainingCapacityHours: 0 });
  const draft = { id: 'session', ruleId: 'rule', assignment: { goalId: 'goal', title: 'Learn' }, date: '2026-09-28', start: '09:07', endDate: '2026-09-28', end: '10:22', mode: 'plan' as const, repeat: false, scope: 'date' as const };
  const request = draftMutation(draft, week);
  assert.equal(request.revision, 'r1');
  assert.equal(request.action, 'session');
  if (request.action === 'session') { assert.equal(request.session.plan?.start, '2026-09-28T14:07:00.000Z'); assert.equal(request.session.exception, true); }
  const rule = draftMutation({ ...draft, scope: 'future' }, week);
  assert.equal(rule.action, 'rule');
  if (rule.action === 'rule') assert.equal(rule.rule.durationMinutes, 75);
  const actual = draftMutation({ ...draft, mode: 'actual' }, week);
  assert.equal(actual.action, 'actual');
  if (actual.action === 'actual') { assert.equal(actual.id, 'session'); assert.equal(actual.actual.status, 'explicit'); }
});

test('unchanged editor times retain the original instants across a repeated hour', async () => {
  const { weekSchema, draftMutation } = await import('../src/lib/scheduler.ts');
  const week = weekSchema.parse({ revision: 'r1', week: '2026-10-26', settings: { timeZone: 'America/Chicago', defaultDay: { start: '00:00', end: '00:00', nextDay: true }, weekdays: {}, dates: {} }, days: [], goals: [], sessions: [], rules: [], busy: [], warnings: [], remainingCapacityHours: 0 });
  const mutation = draftMutation({ id: 'historical', assignment: { goalId: 'goal', title: 'Learn' }, date: '2026-11-01', start: '01:30', endDate: '2026-11-01', end: '01:30', originalStart: '2026-11-01T06:30:00Z', originalEnd: '2026-11-01T07:30:00Z', mode: 'actual', repeat: false, scope: 'date' }, week);
  assert.equal(mutation.action, 'actual');
  if (mutation.action === 'actual') assert.equal((Date.parse(mutation.actual.end) - Date.parse(mutation.actual.start)) / 3600000, 1);
});
