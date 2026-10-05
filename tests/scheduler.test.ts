import assert from 'node:assert/strict';
import test from 'node:test';
import { addDays, mondayOf, snapMinutes, zonedDate, localInstant, localFields, clockLabel, displayClock, draftMutation, weekSchema } from '../src/lib/scheduler.ts';
import { proposePlacement } from '../src/lib/scheduler-placement.ts';

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
test('displayClock shows 12-hour times with no leading zero on the hour', () => {
  assert.equal(displayClock(0), '12:00 AM');
  assert.equal(displayClock(9 * 60), '9:00 AM');
  assert.equal(displayClock(12 * 60), '12:00 PM');
  assert.equal(displayClock(13 * 60 + 5), '1:05 PM');
  assert.equal(displayClock(23 * 60 + 59), '11:59 PM');
  assert.equal(displayClock(1505), '1:05 AM +1d');
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

const placementWeek = (day: string, zone = 'America/Chicago', start = '09:00', end = '17:00', nextDay = false) => {
  const dayStart = localInstant(day, start, zone);
  const dayEnd = localInstant(nextDay ? addDays(day, 1) : day, end, zone);
  return weekSchema.parse({ revision: 'r1', week: mondayOf(day), settings: { timeZone: zone, defaultDay: { start, end, nextDay }, weekdays: {}, dates: {} }, days: [{ valid: true, date: day, interval: { start, end, nextDay }, start: dayStart, end: dayEnd }], goals: [{ goal: { id: 'goal', title: 'Learn', color: '#123456', startDate: '2026-01-01', endDate: '2027-01-01', status: 'active', dailyHours: 1, selectedWeekdays: null, steps: [], dependsOn: [] }, requiredHours: 1, actualHours: 0, remainingScheduledHours: 0, uncoveredHours: 0, excessHours: 0, unscheduledStepIds: [] }], sessions: [], rules: [], busy: [], warnings: [], remainingCapacityHours: 8 });
};

test('moving a block subtracts the 30-minute grab offset before snapping', () => {
  const week = placementWeek('2026-10-05');
  const session = weekSchema.parse({ ...week, sessions: [{ id: 's', date: '2026-10-05', assignment: { goalId: 'goal', title: 'Learn' }, plan: { start: '2026-10-05T14:00:00Z', end: '2026-10-05T15:00:00Z' }, actual: null, state: 'accepted', exception: false, conflictIds: [] }] }).sessions[0];
  const result = proposePlacement({ kind: 'move', session, targetDate: '2026-10-05', startMinute: 570 }, week, new Date('2026-10-01T00:00:00Z'));
  assert.equal(result.kind, 'valid');
  if (result.kind === 'valid') assert.deepEqual([result.start, result.end, result.draft.start, result.draft.end], ['2026-10-05T14:30:00.000Z', '2026-10-05T15:30:00.000Z', '09:30', '10:30']);
});

test('move and resize retain both computed fold instants and elapsed duration', () => {
  const week = placementWeek('2026-11-01', 'America/Chicago', '00:00', '04:00');
  const session = weekSchema.parse({ ...week, sessions: [{ id: 's', date: '2026-11-01', assignment: { goalId: 'goal', title: 'Learn' }, plan: { start: '2026-11-01T06:30:00Z', end: '2026-11-01T07:30:00Z' }, actual: null, state: 'accepted', exception: false, conflictIds: [] }] }).sessions[0];
  const now = new Date('2026-10-01T00:00:00Z');
  const moved = proposePlacement({ kind: 'move', session, targetDate: '2026-11-01', startMinute: 90 }, week, now);
  assert.equal(moved.kind, 'valid');
  if (moved.kind === 'valid') {
    assert.deepEqual([moved.start, moved.end, moved.draft.originalStart, moved.draft.originalEnd], ['2026-11-01T06:30:00.000Z', '2026-11-01T07:30:00.000Z', '2026-11-01T06:30:00.000Z', '2026-11-01T07:30:00.000Z']);
    const request = draftMutation(moved.draft, week);
    if (request.action === 'session') {
      assert.deepEqual([request.session.plan?.start, request.session.plan?.end], ['2026-11-01T06:30:00.000Z', '2026-11-01T07:30:00.000Z']);
      assert.equal((Date.parse(request.session.plan!.end) - Date.parse(request.session.plan!.start)) / 60000, 60);
    }
  }
  const resized = proposePlacement({ kind: 'resize', session, edge: 'start', deltaMinutes: 60 }, week, now);
  assert.equal(resized.kind, 'invalid');
  const secondFold = weekSchema.parse({ ...week, sessions: [{ ...session, plan: { start: '2026-11-01T07:30:00Z', end: '2026-11-01T08:30:00Z' } }] }).sessions[0];
  const secondFoldMove = proposePlacement({ kind: 'move', session: secondFold, targetDate: '2026-11-01', startMinute: 105 }, week, now);
  assert.equal(secondFoldMove.kind, 'valid');
  if (secondFoldMove.kind === 'valid') {
    assert.deepEqual([secondFoldMove.start, secondFoldMove.end], ['2026-11-01T07:45:00.000Z', '2026-11-01T08:45:00.000Z']);
    const mutation = draftMutation(secondFoldMove.draft, week);
    if (mutation.action === 'session') assert.deepEqual([mutation.session.plan?.start, mutation.session.plan?.end], ['2026-11-01T07:45:00.000Z', '2026-11-01T08:45:00.000Z']);
  }
  const nextDay = weekSchema.parse({ ...week, sessions: [{ ...secondFold, date: '2026-11-02', plan: { start: '2026-11-02T07:30:00Z', end: '2026-11-02T08:30:00Z' } }] }).sessions[0];
  const movedFromOrdinaryDay = proposePlacement({ kind: 'move', session: nextDay, targetDate: '2026-11-01', startMinute: 105 }, week, now);
  assert.equal(movedFromOrdinaryDay.kind, 'valid');
  if (movedFromOrdinaryDay.kind === 'valid') assert.deepEqual([movedFromOrdinaryDay.start, movedFromOrdinaryDay.end], ['2026-11-01T06:45:00.000Z', '2026-11-01T07:45:00.000Z']);
  const ordinarySameDay = weekSchema.parse({ ...week, sessions: [{ ...secondFold, plan: { start: '2026-11-01T08:30:00Z', end: '2026-11-01T09:30:00Z' } }] }).sessions[0];
  const movedFromOrdinaryHour = proposePlacement({ kind: 'move', session: ordinarySameDay, targetDate: '2026-11-01', startMinute: 105 }, week, now);
  assert.equal(movedFromOrdinaryHour.kind, 'valid');
  if (movedFromOrdinaryHour.kind === 'valid') assert.deepEqual([movedFromOrdinaryHour.start, movedFromOrdinaryHour.end], ['2026-11-01T06:45:00.000Z', '2026-11-01T07:45:00.000Z']);
  const earlyFoldWithSeconds = weekSchema.parse({ ...week, sessions: [{ ...secondFold, plan: { start: '2026-11-01T06:30:45Z', end: '2026-11-01T07:30:45Z' } }] }).sessions[0];
  const movedFromEarlyFold = proposePlacement({ kind: 'move', session: earlyFoldWithSeconds, targetDate: '2026-11-01', startMinute: 105 }, week, now);
  assert.equal(movedFromEarlyFold.kind, 'valid');
  if (movedFromEarlyFold.kind === 'valid') assert.deepEqual([movedFromEarlyFold.start, movedFromEarlyFold.end], ['2026-11-01T06:45:00.000Z', '2026-11-01T07:45:00.000Z']);
  const endResize = proposePlacement({ kind: 'resize', session: secondFold, edge: 'end', deltaMinutes: -30 }, week, now);
  assert.equal(endResize.kind, 'valid');
  if (endResize.kind === 'valid') {
    assert.deepEqual([endResize.start, endResize.end], ['2026-11-01T07:30:00.000Z', '2026-11-01T08:00:00.000Z']);
    const mutation = draftMutation(endResize.draft, week);
    if (mutation.action === 'session') assert.deepEqual([mutation.session.plan?.start, mutation.session.plan?.end], ['2026-11-01T07:30:00.000Z', '2026-11-01T08:00:00.000Z']);
  }
});

test('placement rejects nonexistent spring wall time', () => {
  const week = placementWeek('2026-03-08', 'America/Chicago', '00:00', '04:00');
  const result = proposePlacement({ kind: 'assignment', assignment: { goalId: 'goal', title: 'Learn' }, targetDate: '2026-03-08', startMinute: 150, durationMinutes: 60 }, week, new Date('2026-01-01T00:00:00Z'));
  assert.equal(result.kind, 'invalid');
  if (result.kind === 'invalid') assert.match(result.message, /does not exist/);
});

test('spring resize follows elapsed minutes across the clock jump', () => {
  const week = placementWeek('2026-03-08', 'America/Chicago', '00:00', '04:00');
  const session = weekSchema.parse({ ...week, sessions: [{ id: 's', date: '2026-03-08', assignment: { goalId: 'goal', title: 'Learn' }, plan: { start: '2026-03-08T07:30:00Z', end: '2026-03-08T08:00:00Z' }, actual: null, state: 'accepted', exception: false, conflictIds: [] }] }).sessions[0];
  const result = proposePlacement({ kind: 'resize', session, edge: 'end', deltaMinutes: 30 }, week, new Date('2026-01-01T00:00:00Z'));
  assert.equal(result.kind, 'valid');
  if (result.kind === 'valid') {
    assert.deepEqual([result.start, result.end, result.draft.start, result.draft.end], ['2026-03-08T07:30:00.000Z', '2026-03-08T08:30:00.000Z', '01:30', '03:30']);
    const mutation = draftMutation(result.draft, week);
    if (mutation.action === 'session') assert.deepEqual([mutation.session.plan?.start, mutation.session.plan?.end], ['2026-03-08T07:30:00.000Z', '2026-03-08T08:30:00.000Z']);
  }
});

test('post-midnight actual proposal keeps the owning column date through submission', () => {
  const week = placementWeek('2026-09-27', 'America/Chicago', '09:00', '02:00', true);
  const session = weekSchema.parse({ ...week, sessions: [{ id: 's', date: '2026-09-27', assignment: { goalId: 'goal', title: 'Learn' }, plan: { start: '2026-09-27T14:00:00Z', end: '2026-09-27T15:00:00Z' }, actual: { status: 'explicit', date: '2026-09-27', start: '2026-09-27T14:00:00Z', end: '2026-09-27T15:00:00Z' }, state: 'accepted', exception: false, conflictIds: [] }] }).sessions[0];
  const result = proposePlacement({ kind: 'move', session, targetDate: '2026-09-27', startMinute: 1500 }, week, new Date('2026-10-02T00:00:00Z'));
  assert.equal(result.kind, 'valid');
  if (result.kind === 'valid') {
    assert.deepEqual([result.draft.date, result.draft.startDate, result.start, result.end], ['2026-09-27', '2026-09-28', '2026-09-28T06:00:00.000Z', '2026-09-28T07:00:00.000Z']);
    const request = draftMutation(result.draft, week);
    if (request.action === 'actual') assert.deepEqual([request.actual.date, request.actual.start, request.actual.end], ['2026-09-27', '2026-09-28T06:00:00.000Z', '2026-09-28T07:00:00.000Z']);
  }
});

test('proposal marks overlap but rejects invalid plan geometry and future actual ends', () => {
  const base = placementWeek('2026-10-05');
  const week = weekSchema.parse({ ...base, sessions: [{ id: 'other', date: '2026-10-05', assignment: { title: 'Busy' }, plan: { start: '2026-10-05T14:00:00Z', end: '2026-10-05T15:00:00Z' }, actual: null, state: 'accepted', exception: false, conflictIds: [] }] });
  const assignment = { goalId: 'goal', title: 'Learn' };
  const now = new Date('2026-10-01T00:00:00Z');
  const conflict = proposePlacement({ kind: 'assignment', assignment, targetDate: '2026-10-05', startMinute: 570, durationMinutes: 60 }, week, now);
  assert.equal(conflict.kind, 'valid');
  if (conflict.kind === 'valid') assert.deepEqual(conflict.conflictIds, ['other']);
  const outside = proposePlacement({ kind: 'assignment', assignment, targetDate: '2026-10-05', startMinute: 1005, durationMinutes: 60 }, week, now);
  assert.equal(outside.kind, 'invalid');
  const actual = proposePlacement({ kind: 'assignment', assignment, targetDate: '2026-10-01', startMinute: 570, durationMinutes: 60 }, week, new Date('2026-10-01T15:00:00Z'));
  assert.equal(actual.kind, 'invalid');
  if (actual.kind === 'invalid') assert.match(actual.message, /future/);
});

test('actual placement allows outside-hours work and reports actual overlap', () => {
  const base = placementWeek('2026-09-27');
  const week = weekSchema.parse({ ...base, sessions: [{ id: 'other', date: '2026-09-27', assignment: { goalId: 'goal', title: 'Learn' }, plan: null, actual: { status: 'explicit', date: '2026-09-27', start: '2026-09-27T08:00:00Z', end: '2026-09-27T09:00:00Z' }, state: 'accepted', exception: false, conflictIds: [] }] });
  const result = proposePlacement({ kind: 'assignment', assignment: { goalId: 'goal', title: 'Learn' }, targetDate: '2026-09-27', startMinute: 210, durationMinutes: 60 }, week, new Date('2026-10-02T00:00:00Z'));
  assert.equal(result.kind, 'valid');
  if (result.kind === 'valid') assert.deepEqual([result.draft.mode, result.start, result.end, result.conflictIds], ['actual', '2026-09-27T08:30:00.000Z', '2026-09-27T09:30:00.000Z', ['other']]);
});

test('planning rejects an ineligible goal date', () => {
  const base = placementWeek('2026-10-05');
  const week = weekSchema.parse({ ...base, goals: [{ ...base.goals[0], goal: { ...base.goals[0].goal, pauses: [{ from: '2026-10-05', to: '2026-10-05' }] } }] });
  const result = proposePlacement({ kind: 'assignment', assignment: { goalId: 'goal', title: 'Learn' }, targetDate: '2026-10-05', startMinute: 570, durationMinutes: 60 }, week, new Date('2026-10-01T00:00:00Z'));
  assert.equal(result.kind, 'invalid');
  if (result.kind === 'invalid') assert.match(result.message, /not eligible/);
});

test('display geometry includes early and late actuals and elapsed fold duration', async () => {
 const { displayAxis, intervalGeometry } = await import('../src/lib/scheduler.ts');
 const week=placementWeek('2026-11-01');
 week.sessions.push({id:'fold',date:'2026-11-01',state:'accepted',assignment:{title:'Fold'},plan:null,actual:{status:'explicit',date:'2026-11-01',start:'2026-11-01T06:30:00Z',end:'2026-11-01T07:30:00Z'},conflictIds:[],exception:false});
 week.sessions.push({...week.sessions[0],id:'late',actual:{status:'explicit',date:'2026-11-01',start:'2026-11-02T04:00:00Z',end:'2026-11-02T05:00:00Z'}});
 assert.deepEqual(intervalGeometry(week.sessions[0].actual!, '2026-11-01',week.settings.timeZone),{top:90,height:60});
 assert.deepEqual(displayAxis(week),{start:90,end:1380});
});
test('quick add uses the selected future day and available remaining duration', async () => {
 const { quickAddSlot } = await import('../src/lib/scheduler.ts');
 const week=placementWeek('2026-11-02');
 assert.deepEqual(quickAddSlot(week,'2026-11-02',new Date('2026-11-02T22:30:00Z'),60),{start:'2026-11-02T22:31:00.000Z',end:'2026-11-02T23:00:00.000Z'});
 assert.match(quickAddSlot(week,'2026-11-02',new Date('2026-11-03T00:00:00Z'),60).reason!,/No valid future slot/);
});
test('double-click gap fills from the quarter hour to the next block, day end, or one hour', async () => {
 const { gapAt } = await import('../src/lib/scheduler.ts');
 const week=placementWeek('2026-10-05');
 const at=(clock: string) => Number(clock.slice(0, 2)) * 60 + Number(clock.slice(3));
 const before=new Date('2026-10-01T00:00:00Z');
 assert.deepEqual(gapAt(week,'2026-10-05',at('10:07'),before),{start:at('10:00'),end:at('11:00')});
 assert.deepEqual(gapAt(week,'2026-10-05',at('16:40'),before),{start:at('16:30'),end:at('17:00')});
 assert.equal(gapAt(week,'2026-10-05',at('08:59'),before),null);
 assert.equal(gapAt(week,'2026-10-05',at('17:00'),before),null);
 assert.equal(gapAt(week,'2026-10-06',at('10:00'),before),null);
 const session = { id:'s', date:'2026-10-05', state:'accepted' as const, assignment:{title:'Block'}, plan:{start:'2026-10-05T15:30:00Z',end:'2026-10-05T16:00:00Z'}, actual:null, conflictIds:[], exception:false };
 week.sessions.push(session);
 assert.deepEqual(gapAt(week,'2026-10-05',at('10:07'),before),{start:at('10:00'),end:at('10:30')});
 assert.equal(gapAt(week,'2026-10-05',at('10:45'),before),null);
 assert.deepEqual(gapAt(week,'2026-10-05',at('11:05'),before),{start:at('11:00'),end:at('12:00')});
 week.busy.push({ id:'b', title:'Busy', start:'2026-10-05T17:10:00Z', end:'2026-10-05T17:40:00Z' });
 assert.deepEqual(gapAt(week,'2026-10-05',at('11:50'),before),{start:at('11:45'),end:at('12:10')});
 assert.equal(gapAt(week,'2026-10-05',at('12:20'),before),null);
 assert.deepEqual(gapAt(week,'2026-10-05',at('12:41'),before),{start:at('12:40'),end:at('13:40')});
 week.sessions.push({ ...session, id:'gone', state:'canceled' as const, plan:{start:'2026-10-05T19:00:00Z',end:'2026-10-05T20:00:00Z'} });
 assert.deepEqual(gapAt(week,'2026-10-05',at('13:50'),before),{start:at('13:45'),end:at('14:45')});
});
test('double-click gap is cut at now so the draft can be saved as actual work or a plan', async () => {
 const { gapAt } = await import('../src/lib/scheduler.ts');
 const week=placementWeek('2026-10-05');
 const at=(clock: string) => Number(clock.slice(0, 2)) * 60 + Number(clock.slice(3));
 const now=new Date('2026-10-05T15:40:30Z');
 assert.deepEqual(gapAt(week,'2026-10-05',at('10:07'),now),{start:at('10:00'),end:at('10:40')});
 assert.deepEqual(gapAt(week,'2026-10-05',at('10:40'),now),{start:at('10:41'),end:at('11:41')});
 assert.deepEqual(gapAt(week,'2026-10-05',at('10:42'),now),{start:at('10:41'),end:at('11:41')});
 assert.deepEqual(gapAt(week,'2026-10-05',at('10:50'),now),{start:at('10:45'),end:at('11:45')});
 assert.deepEqual(gapAt(week,'2026-10-05',at('09:10'),now),{start:at('09:00'),end:at('10:00')});
 week.busy.push({ id:'b', title:'Busy', start:'2026-10-05T16:00:00Z', end:'2026-10-05T17:00:00Z' });
 assert.deepEqual(gapAt(week,'2026-10-05',at('10:42'),now),{start:at('10:41'),end:at('11:00')});
});
test('double-click gap follows overnight days, active intervals, and sessions drawn without one', async () => {
 const { gapAt } = await import('../src/lib/scheduler.ts');
 const at=(clock: string) => Number(clock.slice(0, 2)) * 60 + Number(clock.slice(3));
 const before=new Date('2026-10-01T00:00:00Z');
 const night=placementWeek('2026-10-05','America/Chicago','20:00','02:00',true);
 assert.deepEqual(gapAt(night,'2026-10-05',at('23:50'),before),{start:at('23:45'),end:at('24:45')});
 assert.deepEqual(gapAt(night,'2026-10-05',at('25:55'),before),{start:at('25:45'),end:at('26:00')});
 const base = { date:'2026-10-05', state:'accepted' as const, assignment:{title:'Block'}, actual:null, conflictIds:[], exception:false };
 night.sessions.push({ ...base, id:'late', date:'2026-10-06', plan:{start:'2026-10-06T04:00:00Z',end:'2026-10-06T05:30:00Z'} });
 assert.deepEqual(gapAt(night,'2026-10-05',at('24:40'),before),{start:at('24:30'),end:at('25:30')});
 assert.equal(gapAt(night,'2026-10-05',at('24:20'),before),null);

 const week=placementWeek('2026-10-05');
 const moved = { ...base, id:'moved', plan:{start:'2026-10-05T19:00:00Z',end:'2026-10-05T20:00:00Z'}, actual:{status:'explicit' as const,date:'2026-10-05',start:'2026-10-05T20:00:00Z',end:'2026-10-05T21:00:00Z'} };
 week.sessions.push(moved);
 assert.equal(gapAt(week,'2026-10-05',at('15:30'),before),null);
 assert.deepEqual(gapAt(week,'2026-10-05',at('14:30'),before),{start:at('14:30'),end:at('15:00')});
 week.sessions[0] = { ...moved, actual:{ ...moved.actual, status:'skipped' as const } };
 assert.equal(gapAt(week,'2026-10-05',at('14:30'),before),null);
 assert.deepEqual(gapAt(week,'2026-10-05',at('15:30'),before),{start:at('15:30'),end:at('16:30')});
 week.sessions.push({ ...base, id:'unplaced', plan:null });
 assert.equal(gapAt(week,'2026-10-05',at('09:30'),before),null);
 assert.deepEqual(gapAt(week,'2026-10-05',at('10:07'),before),{start:at('10:00'),end:at('11:00')});
});
