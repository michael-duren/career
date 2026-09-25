import test from 'node:test';
import assert from 'node:assert/strict';
import { catchUpQueue, markCaughtUp, nextDue, relativeDays, initials, matchesQuery, type Connection } from '../src/lib/connections.ts';

const base: Connection = { id: 'x', name: 'Ada Lovelace', role: 'SRE', companyName: 'Grafana Labs', email: '', queued: false, notes: '', tags: ['observability'] };
const today = '2026-09-25';

test('cadence falls back to connection date and treats never-contacted people as due', () => {
  assert.equal(nextDue(base, today), undefined);
  assert.equal(nextDue({ ...base, cadenceDays: 30, connectedOn: '2026-09-01' }, today), '2026-10-01');
  assert.equal(nextDue({ ...base, cadenceDays: 30, connectedOn: '2026-01-01', lastContactedOn: '2026-09-10' }, today), '2026-10-10');
  assert.equal(nextDue({ ...base, cadenceDays: 30 }, today), today);
});

test('queue holds manual and overdue people, most overdue first', () => {
  const queue = catchUpQueue([
    { ...base, id: 'not-due', name: 'Not due', cadenceDays: 90, lastContactedOn: '2026-09-01' },
    { ...base, id: 'manual', name: 'Manual', queued: true },
    { ...base, id: 'late', name: 'Late', cadenceDays: 30, lastContactedOn: '2026-06-01' },
    { ...base, id: 'slightly', name: 'Slightly', cadenceDays: 14, lastContactedOn: '2026-09-10' },
    { ...base, id: 'none', name: 'No cadence' },
  ], today);
  assert.deepEqual(queue.map(item => [item.connection.id, item.overdueDays, item.reason]), [['late', 86, 'cadence'], ['slightly', 1, 'cadence'], ['manual', 0, 'manual']]);
});

test('catching up removes a person from the queue', () => {
  const late = { ...base, cadenceDays: 30, lastContactedOn: '2026-06-01', queued: true };
  const done = markCaughtUp(late, today);
  assert.equal(done.lastContactedOn, today);
  assert.equal(done.queued, false);
  assert.deepEqual(catchUpQueue([done], today), []);
});

test('formatting helpers', () => {
  assert.equal(relativeDays(undefined, today), 'Never');
  assert.equal(relativeDays(today, today), 'Today');
  assert.equal(relativeDays('2026-09-20', today), '5d ago');
  assert.equal(relativeDays('2026-03-25', today), '6mo ago');
  assert.equal(relativeDays('2023-09-25', today), '3y ago');
  assert.equal(initials('ada  byron lovelace'), 'AB');
  assert.equal(initials(''), '?');
  assert.ok(matchesQuery(base, 'grafana'));
  assert.ok(matchesQuery(base, 'OBSERV'));
  assert.ok(!matchesQuery(base, 'datadog'));
});
