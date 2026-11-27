import test from 'node:test';
import assert from 'node:assert/strict';
import { CATCH_UP_GOAL_KEY, avatarStack, catchUpQueue, currentStep, isSnoozed, lastContactNote, lastContactTag, linkableGoals, linkedGoal, queueCandidates, queueLanes, snoozedPeople, withCatchUpLink, companySlug, connectionsByCompany, employerKey, employers, trackedNames, markCaughtUp, newCompanyEntry, normalizeCompany, nextDue, relativeDays, initials, matchesQuery, type Connection } from '../src/lib/connections.ts';
import type { Goal } from '../src/lib/timeline.ts';

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

test('connections group by company, most recent touchpoint first, ignoring unlinked people', () => {
  const groups = connectionsByCompany([
    { ...base, id: 'old', name: 'Old', companySlug: 'grafana-labs', lastContactedOn: '2026-01-01' },
    { ...base, id: 'connected', name: 'Connected', companySlug: 'grafana-labs', connectedOn: '2026-06-01' },
    { ...base, id: 'never-b', name: 'Bea', companySlug: 'grafana-labs' },
    { ...base, id: 'unlinked', name: 'Unlinked' },
    { ...base, id: 'recent', name: 'Recent', companySlug: 'grafana-labs', lastContactedOn: '2026-09-01' },
    { ...base, id: 'never-a', name: 'Abe', companySlug: 'grafana-labs' },
    { ...base, id: 'other', name: 'Other', companySlug: 'datadog' },
  ]);
  assert.deepEqual([...groups.keys()], ['grafana-labs', 'datadog']);
  assert.deepEqual(groups.get('grafana-labs')?.map(c => c.id), ['recent', 'connected', 'old', 'never-a', 'never-b']);
  assert.equal(connectionsByCompany([]).size, 0);
});

test('avatar stacks cap circles and never show +1', () => {
  assert.deepEqual(avatarStack([], 4), { shown: [], overflow: 0 });
  assert.deepEqual(avatarStack([1, 2, 3, 4], 4), { shown: [1, 2, 3, 4], overflow: 0 });
  assert.deepEqual(avatarStack([1, 2, 3, 4, 5], 4), { shown: [1, 2, 3], overflow: 2 });
  assert.deepEqual(avatarStack([1, 2, 3, 4, 5, 6, 7, 8, 9], 4), { shown: [1, 2, 3], overflow: 6 });
  assert.deepEqual(avatarStack([1, 2, 3, 4, 5], 5), { shown: [1, 2, 3, 4, 5], overflow: 0 });
  assert.deepEqual(avatarStack([1, 2, 3, 4, 5, 6], 5), { shown: [1, 2, 3, 4], overflow: 2 });
  assert.deepEqual(avatarStack([1], 1), { shown: [1], overflow: 0 });
  // With room for one circle, it becomes the "+N" badge.
  assert.deepEqual(avatarStack([1, 2], 1), { shown: [], overflow: 2 });
});

test('company names normalize like the Go matcher', () => {
  assert.equal(normalizeCompany('Maximus, Inc.'), 'maximus');
  assert.equal(normalizeCompany('The Home Depot'), 'home depot');
  assert.equal(normalizeCompany('AT&T'), 'at and t');
  assert.equal(normalizeCompany('  '), '');
  assert.equal(normalizeCompany('Acme Co LLC'), 'acme');
  assert.equal(normalizeCompany('Siemens AG'), 'siemens');
  assert.equal(normalizeCompany('Booking BV'), 'booking');
  assert.equal(normalizeCompany('SAP SA GmbH'), 'sap');
  assert.equal(normalizeCompany('Johnson & Co'), 'johnson and');
});

test('employers group linked people by company and the rest by normalized name, most first', () => {
  const people = [
    { ...base, id: 'a', companyName: 'Maximus Inc' },
    { ...base, id: 'b', companyName: 'maximus' },
    { ...base, id: 'c', companyName: 'Grafana', companySlug: 'grafana-labs' },
    { ...base, id: 'd', companyName: 'Grafana Labs', companySlug: 'grafana-labs' },
    // Unlinked, but the name is already on the board: counts toward the tracked company.
    { ...base, id: 'e', companyName: 'Grafana Labs, Inc.' },
    { ...base, id: 'f', companyName: 'Acme' },
    { ...base, id: 'g', companyName: '' },
  ];
  const titles = new Map([['grafana-labs', 'Grafana Labs']]);
  assert.deepEqual(employers(people, titles).map(e => [e.name, e.count, e.slug]), [['Grafana Labs', 3, 'grafana-labs'], ['Maximus Inc', 2, undefined], ['Acme', 1, undefined]]);
  const tracked = trackedNames(titles);
  assert.equal(employerKey(people[4], tracked), 'slug:grafana-labs');
  assert.equal(employerKey(people[1], tracked), 'name:maximus');
  assert.equal(employerKey(people[6], tracked), '');
  // Companies off: no titles, so everyone groups by LinkedIn name, links ignored.
  assert.deepEqual(employers(people).map(e => [e.name, e.count, e.slug]), [['Grafana Labs', 2, undefined], ['Maximus Inc', 2, undefined], ['Acme', 1, undefined], ['Grafana', 1, undefined]]);
});

test('new companies get a URL-safe slug and the outreach checklist', () => {
  assert.equal(companySlug('Prime Digital Academy'), 'prime-digital-academy');
  assert.equal(companySlug('Café & Co.'), 'cafe-co');
  assert.equal(companySlug('!!!'), '');
  const entry = newCompanyEntry('York Solutions', 'york-solutions');
  assert.equal(entry.status, 'not_started');
  assert.equal(entry.url, 'https://www.linkedin.com/search/results/companies/?keywords=York%20Solutions');
  assert.match(entry.body, /^## Why\n\n## Steps\n\n- \[ \] Identify one person/);
  assert.match(entry.body, /## Log\n$/);
});

test('queue lanes hold staged people by name and leave snoozed ones out until their date', () => {
  const people: Connection[] = [
    { ...base, id: 'b', name: 'Bea', stage: 'messaged' },
    { ...base, id: 'a', name: 'Al', stage: 'messaged' },
    { ...base, id: 's', name: 'Sleepy', stage: 'to_contact', snoozedUntil: '2026-09-26' },
    { ...base, id: 'w', name: 'Woken', stage: 'to_contact', snoozedUntil: today },
    { ...base, id: 'n', name: 'Not queued' },
  ];
  assert.deepEqual(queueLanes(people, today).map(lane => [lane.stage, lane.people.map(c => c.id)]),
    [['to_contact', ['w']], ['messaged', ['a', 'b']], ['follow_up', []], ['referral_ask', []]]);
  assert.equal(isSnoozed(people[2], today), true);
  assert.equal(isSnoozed(people[3], today), false);
  assert.deepEqual(snoozedPeople(people, today).map(c => c.id), ['s']);
});

test('queue candidates are linked people without a stage, most recently in touch first', () => {
  const people: Connection[] = [
    { ...base, id: 'old', name: 'Old', companySlug: 'x', lastContactedOn: '2025-01-01' },
    { ...base, id: 'new', name: 'New', companySlug: 'x', lastContactedOn: '2026-09-01' },
    { ...base, id: 'staged', name: 'Staged', companySlug: 'x', stage: 'to_contact' },
    { ...base, id: 'unlinked', name: 'Unlinked' },
  ];
  assert.deepEqual(queueCandidates(people).map(c => c.id), ['new', 'old']);
});

test('last contact wording on cards and in the panel', () => {
  assert.equal(lastContactTag(base), 'NO CONTACT LOGGED');
  assert.equal(lastContactTag({ ...base, lastContactedOn: '2026-09-11' }), 'LAST · SEP 11');
  assert.equal(lastContactNote(base, today), 'never contacted');
  assert.equal(lastContactNote({ ...base, lastContactedOn: '2026-09-11' }, today), 'last contact Sep 11 · 14 days ago');
  assert.equal(lastContactNote({ ...base, lastContactedOn: '2026-09-24' }, today), 'last contact Sep 24 · 1 day ago');
  assert.equal(lastContactNote({ ...base, lastContactedOn: today }, today), 'last contact Sep 25 · today');
});

test('the linked goal comes from goal metadata, preferring an active goal', () => {
  const goal = (id: string, status: Goal['status'], linked: boolean, steps: Goal['steps'] = []): Goal => ({
    id, status, title: id, dependsOn: [], startDate: '2026-01-01', endDate: '2026-12-31', color: '#000000', createdAt: '', updatedAt: '', notes: [],
    metadata: linked ? { [CATCH_UP_GOAL_KEY]: 'linked' } : {}, steps,
  });
  assert.equal(linkedGoal([goal('a', 'planned', true), goal('b', 'active', true)])?.id, 'b');
  assert.equal(linkedGoal([goal('a', 'dropped', true), goal('b', 'active', false)]), undefined);
  assert.deepEqual(linkableGoals([goal('d', 'done', false), goal('p', 'planned', false), goal('x', 'dropped', false)]).map(g => g.id), ['p']);
  assert.deepEqual(withCatchUpLink(goal('a', 'active', false), true).metadata, { [CATCH_UP_GOAL_KEY]: 'linked' });
  assert.deepEqual(withCatchUpLink({ ...goal('a', 'active', true), metadata: { [CATCH_UP_GOAL_KEY]: 'linked', purpose: 'career' } }, false).metadata, { purpose: 'career' });
  const steps = [{ id: '1', title: 'one', done: true }, { id: '2', title: 'two', done: false }];
  assert.equal(currentStep(goal('a', 'active', true)), 0);
  assert.equal(currentStep(goal('a', 'active', true, steps)), 2);
  assert.equal(currentStep(goal('a', 'active', true, steps.map(s => ({ ...s, done: true })))), 3);
});
