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
export function localFields(instant: string, timezone: string) {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: timezone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23', timeZoneName: 'shortOffset' }).formatToParts(new Date(instant));
  const part = (type: Intl.DateTimeFormatPartTypes) => parts.find(p => p.type === type)?.value ?? '';
  return { date: `${part('year')}-${part('month')}-${part('day')}`, time: `${part('hour')}:${part('minute')}`, offset: part('timeZoneName') };
}
export function zonedDate(now: Date, timezone: string): string { return localFields(now.toISOString(), timezone).date; }
export function localInstant(date: string, time: string, timezone: string): string {
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
  return new Date(first).toISOString();
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
const goalSchema = z.object({ id: z.string(), title: z.string(), color: z.string(), startDate: z.string(), endDate: z.string(), status: z.string(), dailyHours: z.number().nullable(), selectedWeekdays: z.array(z.number()).nullable(), steps: list(z.object({ id: z.string(), title: z.string(), completed: z.boolean() })), dependsOn: list(z.string()) });
export const weekSchema = z.object({ revision: z.string(), week: z.string(), settings: settingsSchema, days: list(z.object({ date: z.string(), interval: dayIntervalSchema, start: z.string(), end: z.string() })), goals: list(z.object({ goal: goalSchema, requiredHours: z.number().nullable(), actualHours: z.number(), remainingScheduledHours: z.number(), uncoveredHours: z.number(), excessHours: z.number(), unscheduledStepIds: list(z.string()) })), sessions: list(sessionSchema), rules: list(ruleSchema), busy: list(planSchema.extend({ id: z.string(), title: z.string() })), warnings: list(z.string()), remainingCapacityHours: z.number() });
export type SchedulerWeek = z.infer<typeof weekSchema>;
export type SchedulerSession = z.infer<typeof sessionSchema>;
export type Settings = z.infer<typeof settingsSchema>;
export type Assignment = z.infer<typeof assignmentSchema>;
export type Mutation = { revision: string; week: string } & (
  { action: 'settings'; settings: Settings } |
  { action: 'rule'; id?: string; rule: z.infer<typeof ruleSchema>; effectiveFrom?: string } |
  { action: 'session'; id?: string; session: SchedulerSession } |
  { action: 'cancel'; id: string } |
  { action: 'actual'; id?: string; session?: SchedulerSession; actual: z.infer<typeof actualSchema> }
);
export type SessionDraft = { id?: string; ruleId?: string; assignment: Assignment; date: string; startDate?: string; start: string; endDate: string; end: string; mode: 'plan' | 'actual'; repeat: boolean; scope: 'date' | 'future'; originalStart?: string; originalEnd?: string };
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
