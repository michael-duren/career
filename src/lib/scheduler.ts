import { z } from 'zod';

export function addDays(date: string, count: number): string {
  const value = new Date(`${date}T12:00:00Z`);
  value.setUTCDate(value.getUTCDate() + count);
  return value.toISOString().slice(0, 10);
}
export function mondayOf(date: string): string {
  return addDays(date, -((new Date(`${date}T12:00:00Z`).getUTCDay() + 6) % 7));
}
export function snapMinutes(minutes: number): number { return Math.round(minutes / 15) * 15; }
export function clockLabel(minutes: number): string {
  const day = Math.floor(minutes / 1440);
  const within = ((minutes % 1440) + 1440) % 1440;
  return `${String(Math.floor(within / 60)).padStart(2, '0')}:${String(within % 60).padStart(2, '0')}${day ? ` +${day}d` : ''}`;
}
// displayClock is for on-screen labels only (12-hour, no leading zero on the
// hour); clockLabel stays 24-hour since its output is parsed back via
// .slice(0, 5) to build actual time values.
export function displayClock(minutes: number): string {
  const day = Math.floor(minutes / 1440);
  const within = ((minutes % 1440) + 1440) % 1440;
  const hour24 = Math.floor(within / 60), minute = within % 60;
  const hour12 = hour24 % 12 || 12;
  return `${hour12}:${String(minute).padStart(2, '0')} ${hour24 < 12 ? 'AM' : 'PM'}${day ? ` +${day}d` : ''}`;
}
export function localFields(instant: string, timezone: string) {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: timezone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23', timeZoneName: 'shortOffset' }).formatToParts(new Date(instant));
  const part = (type: Intl.DateTimeFormatPartTypes) => parts.find(p => p.type === type)?.value ?? '';
  return { date: `${part('year')}-${part('month')}-${part('day')}`, time: `${part('hour')}:${part('minute')}`, offset: part('timeZoneName') };
}
export function zonedDate(now: Date, timezone: string): string { return localFields(now.toISOString(), timezone).date; }
export function localInstant(date: string, time: string, timezone: string, preferredOffset?: string): string {
  const wall = Date.parse(`${date}T${time}:00Z`);
  if (!Number.isFinite(wall)) throw new Error('Enter a valid date and time.');
  const offsets = new Set<number>();
  for (const delta of [-36, -12, 0, 12, 36]) {
    const sample = wall + delta * 3600000;
    const local = localFields(new Date(sample).toISOString(), timezone);
    offsets.add(Date.parse(`${local.date}T${local.time}:00Z`) - sample);
  }
  const matches = [...offsets].map(offset => wall - offset).filter(candidate => {
    const local = localFields(new Date(candidate).toISOString(), timezone);
    return local.date === date && local.time === time;
  }).sort((a, b) => a - b);
  const first = matches[0];
  if (first === undefined) throw new Error('This local time does not exist because the clocks change. Choose another time.');
  const preferred = preferredOffset && matches.find(candidate => localFields(new Date(candidate).toISOString(), timezone).offset === preferredOffset);
  return new Date(preferred ?? first).toISOString();
}
export function minuteOf(instant: string, date: string, timezone: string): number {
  const local = localFields(instant, timezone);
  return (Date.parse(`${local.date}T12:00Z`) - Date.parse(`${date}T12:00Z`)) / 60000 + Number(local.time.slice(0, 2)) * 60 + Number(local.time.slice(3));
}

const list = <T extends z.ZodType>(schema: T) => z.array(schema).nullish().transform(value => value ?? []);
export const dayIntervalSchema = z.object({ start: z.string(), end: z.string(), nextDay: z.boolean() });
export const settingsSchema = z.object({ timeZone: z.string(), defaultDay: dayIntervalSchema, weekdays: z.record(z.string(), dayIntervalSchema).nullable().transform(v => v ?? {}), dates: z.record(z.string(), dayIntervalSchema).nullable().transform(v => v ?? {}) });
export const assignmentSchema = z.object({ goalId: z.string().optional(), stepId: z.string().optional(), title: z.string(), goalTitle: z.string().optional(), color: z.string().optional() });
const planSchema = z.object({ start: z.string(), end: z.string() });
const actualSchema = planSchema.extend({ status: z.enum(['assumed', 'explicit', 'skipped']), date: z.string(), outsideTimeline: z.boolean().optional() });
export const sessionSchema = z.object({ id: z.string(), ruleId: z.string().optional(), date: z.string(), assignment: assignmentSchema, plan: planSchema.nullable(), actual: actualSchema.nullable(), state: z.string(), attention: z.string().optional(), conflictIds: list(z.string()), exception: z.boolean() });
const ruleSchema = z.object({ id: z.string(), weekday: z.number(), localStart: z.string(), durationMinutes: z.number(), effectiveFrom: z.string(), effectiveTo: z.string().optional(), assignment: assignmentSchema });
const goalSchema = z.object({ id: z.string(), title: z.string(), color: z.string(), startDate: z.string(), endDate: z.string(), status: z.string(), dailyHours: z.number().nullable(), selectedWeekdays: z.array(z.number()).nullable(), eligibleFrom: z.string().optional(), stoppedDate: z.string().optional(), pauses: list(z.object({ from: z.string(), to: z.string() })), steps: list(z.object({ id: z.string(), title: z.string(), completed: z.boolean() })), dependsOn: list(z.string()) });
export const weekSchema = z.object({ revision: z.string(), week: z.string(), settings: settingsSchema, days: list(z.object({ date: z.string(), interval: dayIntervalSchema, start: z.string(), end: z.string(), valid: z.boolean(), reason: z.string().optional() })), goals: list(z.object({ goal: goalSchema, requiredHours: z.number().nullable(), actualHours: z.number(), remainingScheduledHours: z.number(), uncoveredHours: z.number(), excessHours: z.number(), unscheduledStepIds: list(z.string()) })), sessions: list(sessionSchema), rules: list(ruleSchema), busy: list(planSchema.extend({ id: z.string(), title: z.string() })), warnings: list(z.string()), warningTargets: list(z.object({ message: z.string(), date: z.string().optional(), goalId: z.string().optional(), sessionId: z.string().optional() })), remainingCapacityHours: z.number() });
export type SchedulerWeek = z.infer<typeof weekSchema>;
export type SchedulerSession = z.infer<typeof sessionSchema>;
export type Settings = z.infer<typeof settingsSchema>;
export type Assignment = z.infer<typeof assignmentSchema>;
export type Mutation = { revision: string; week: string } & (
  { action: 'settings'; settings: Settings } |
  { action: 'rule'; id?: string; rule: z.infer<typeof ruleSchema>; effectiveFrom?: string } |
  { action: 'session'; id?: string; session: SchedulerSession } |
  { action: 'cancel'; id: string; scope?: 'date' | 'future' } |
  { action: 'actual'; id?: string; session?: SchedulerSession; actual: z.infer<typeof actualSchema> }
);
export type SessionDraft = { id?: string; ruleId?: string; assignment: Assignment; date: string; startDate?: string; start: string; endDate: string; end: string; mode: 'plan' | 'actual'; repeat: boolean; scope: 'date' | 'future'; originalStart?: string; originalEnd?: string; explanation?: string };
export function draftMutation(draft: SessionDraft, week: SchedulerWeek): Mutation {
  const resolve = (date: string, time: string, original?: string) => {
    if (original) { const fields = localFields(original, week.settings.timeZone); if (fields.date === date && fields.time === time) return original; }
    return localInstant(date, time, week.settings.timeZone);
  };
  const start = resolve(draft.startDate ?? draft.date, draft.start, draft.originalStart);
  const end = resolve(draft.endDate, draft.end, draft.originalEnd);
  if (Date.parse(end) <= Date.parse(start)) throw new Error('End must be after start.');
  const base = { revision: week.revision, week: week.week };
  const session: SchedulerSession = { id: draft.id ?? '', ruleId: draft.ruleId, date: draft.date, assignment: draft.assignment, plan: draft.mode === 'plan' ? { start, end } : null, actual: null, state: 'accepted', exception: Boolean(draft.ruleId), conflictIds: [] };
  if (draft.mode === 'actual') return { ...base, action: 'actual', id: draft.id, session, actual: { status: 'explicit', date: draft.date, start, end } };
  if (draft.repeat || draft.ruleId && draft.scope === 'future') return { ...base, action: 'rule', id: draft.ruleId, effectiveFrom: draft.date, rule: { id: draft.ruleId ?? '', weekday: (new Date(`${draft.date}T12:00Z`).getUTCDay() + 6) % 7 + 1, localStart: draft.start, durationMinutes: (Date.parse(end) - Date.parse(start)) / 60000, effectiveFrom: draft.date, assignment: draft.assignment } };
  return { ...base, action: 'session', id: draft.id, session };
}
export class SchedulerError extends Error {
  current?: SchedulerWeek;
  conflictIds: string[];
  constructor(message: string, current?: SchedulerWeek, conflictIds: string[] = []) { super(message); this.current = current; this.conflictIds = conflictIds; }
}
export async function schedulerRequest(path: string, body?: unknown): Promise<unknown> {
  const response = await fetch(`/api/scheduler/${path}`, { cache: 'no-store', ...(body === undefined ? {} : { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }) });
  const value: unknown = await response.json();
  if (!response.ok) {
    const error = z.object({ error: z.string(), current: weekSchema.optional(), conflictIds: list(z.string()) }).safeParse(value);
    throw error.success ? new SchedulerError(error.data.error, error.data.current, error.data.conflictIds) : new Error(`Scheduler request failed (${response.status}).`);
  }
  return value;
}

export function intervalGeometry(interval: {start: string; end: string}, date: string, zone: string) {
  return { top: minuteOf(interval.start, date, zone), height: (Date.parse(interval.end) - Date.parse(interval.start)) / 60000 };
}
export function displayAxis(week: SchedulerWeek) {
  const minutes = (clock: string) => Number(clock.slice(0, 2)) * 60 + Number(clock.slice(3));
  const starts = week.days.map(day => minutes(day.interval.start));
  const ends = week.days.map(day => minutes(day.interval.end) + (day.interval.nextDay ? 1440 : 0));
  for (const session of week.sessions) {
    if (session.state === 'canceled') continue;
    const interval = session.actual && session.actual.status !== 'skipped' ? session.actual : session.plan;
    if (!interval) continue;
    const geometry = intervalGeometry(interval, session.actual?.date ?? session.date, week.settings.timeZone);
    starts.push(geometry.top); ends.push(geometry.top + geometry.height);
  }
  return { start: Math.min(...starts, 300), end: Math.max(...ends, 1230) };
}
// gapAt finds the free gap for a double-click on empty grid space. clickMinute
// counts from local midnight of date, as on the grid axis. The gap starts at the
// quarter hour at or before the click (never inside an earlier block) and ends at
// the next session or busy block, the end of the scheduling day, or maxMinutes
// after its start, whichever is first. A gap containing now is cut at now: a
// click before now's minute gets the elapsed part (recorded as actual work), a
// click in or after now's minute gets a new span from the next minute, up to
// maxMinutes and still stopping at the next block or day end (a future plan).
// Timed sessions owned by neighbouring dates count as blocks where they overlap
// this column, even though the grid draws them only in their own column. Blocks
// end at their real wall-clock end; sessions are drawn by elapsed minutes, so
// across a daylight-saving change their drawn end differs from it. Whether the
// draft is actual work or a plan is decided by the caller; pass it the same now.
// Returns null for an unknown or invalid day, outside the day's interval, inside a block,
// when nothing is left after cutting at now, or when a daylight-saving change
// makes an edge a skipped wall time, makes the real span differ from its
// wall-clock length (a repeated hour shares grid rows with its first
// occurrence), or would move the draft to the other side of now.
export function gapAt(week: SchedulerWeek, date: string, clickMinute: number, now: Date, maxMinutes = 60): {start: number; end: number} | null {
  const day = week.days.find(day => day.date === date);
  if (!day?.valid) return null;
  const clock = (time: string) => Number(time.slice(0, 2)) * 60 + Number(time.slice(3));
  const dayStart = clock(day.interval.start), dayEnd = clock(day.interval.end) + (day.interval.nextDay ? 1440 : 0);
  if (clickMinute < dayStart || clickMinute >= dayEnd) return null;
  const zone = week.settings.timeZone;
  const blocks = [...week.busy, ...week.sessions.filter(session => session.state !== 'canceled')].flatMap(item => {
    const interval = 'assignment' in item ? (item.actual && item.actual.status !== 'skipped' ? item.actual : item.plan) : item;
    // The grid draws a session without an interval as an hour at its day's start.
    if (!interval) return 'assignment' in item && (item.actual?.date ?? item.date) === date ? [{ start: dayStart, end: dayStart + 60 }] : [];
    const { top, height } = intervalGeometry(interval, date, zone);
    // minuteOf drops seconds, which already floors the start; round the end up so the draft stays clear of
    // second-level edges. Across a daylight-saving change the wall-clock end differs from start + elapsed minutes; use the wall clock.
    const wallEnd = minuteOf(interval.end, date, zone) + (Date.parse(interval.end) % 60000) / 60000;
    return height > 0 ? [{ start: top, end: Math.ceil(wallEnd) }] : [];
  });
  if (blocks.some(block => block.start <= clickMinute && clickMinute < block.end)) return null;
  let start = Math.max(dayStart, Math.floor(clickMinute / 15) * 15, ...blocks.filter(block => block.end <= clickMinute).map(block => block.end));
  const limit = Math.min(dayEnd, ...blocks.filter(block => block.start > clickMinute).map(block => block.start));
  let end = Math.min(limit, start + maxMinutes);
  const nowMinute = minuteOf(now.toISOString(), date, zone);
  if (start <= nowMinute && nowMinute < end) {
    if (clickMinute < nowMinute) end = nowMinute;
    else { start = nowMinute + 1; end = Math.min(limit, start + maxMinutes); }
  }
  const instant = (minute: number) => { try { return Date.parse(localInstant(addDays(date, Math.floor(minute / 1440)), clockLabel(minute).slice(0, 5), zone)); } catch { return NaN; } };
  const from = instant(start), to = instant(end);
  // The resolved draft must keep its wall-clock length and stay on the clicked side of now. The past side is
  // defensive (the cut already ends at now's minute); the future side catches now inside a repeated hour.
  const sideOfNow = clickMinute < nowMinute ? to <= now.getTime() : from > now.getTime();
  return end > start && (to - from) / 60000 === end - start && sideOfNow ? { start, end } : null;
}
export function quickAddSlot(week: SchedulerWeek, date: string, now: Date, duration: number): {start: string; end: string; reason?: string} {
  const day = week.days.find(day => day.date === date);
  const fallback = {start: '', end: '', reason: day?.reason || 'No valid future slot is available on this scheduling date. Choose another date or record actual work.'};
  if (!day?.valid) return fallback;
  const end = Date.parse(day.end);
  let start = Math.max(Date.parse(day.start), (Math.floor(now.getTime() / 60000) + 1) * 60000);
  const blocked = [...week.busy, ...week.sessions.filter(session => session.state === 'accepted' && session.plan).map(session => session.plan!)].sort((a,b) => Date.parse(a.start)-Date.parse(b.start));
  for (const interval of blocked) {
    const a = Date.parse(interval.start), b = Date.parse(interval.end);
    if (b <= start || a >= end) continue;
    if (a > start) return {start:new Date(start).toISOString(), end:new Date(Math.min(a, end, start+duration*60000)).toISOString()};
    start = Math.max(start,b);
  }
  if (start >= end) return fallback;
  return {start:new Date(start).toISOString(),end:new Date(Math.min(end,start+duration*60000)).toISOString()};
}
