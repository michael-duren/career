import { addDays, clockLabel, localFields, localInstant, type Assignment, type SchedulerSession, type SchedulerWeek, type SessionDraft } from './scheduler.ts';

export type PlacementInput =
  | { kind: 'assignment'; assignment: Assignment; targetDate: string; startMinute: number; durationMinutes: number }
  | { kind: 'move'; session: SchedulerSession; targetDate: string; startMinute: number }
  | { kind: 'resize'; session: SchedulerSession; edge: 'start' | 'end'; deltaMinutes: number };

export type PlacementResult =
  | { kind: 'valid'; draft: SessionDraft; start: string; end: string; conflictIds: string[] }
  | { kind: 'invalid'; message: string };

const overlaps = (aStart: number, aEnd: number, bStart: string, bEnd: string) => aStart < Date.parse(bEnd) && Date.parse(bStart) < aEnd;
const dateForMinute = (date: string, minute: number) => addDays(date, Math.floor(minute / 1440));

export function proposePlacement(input: PlacementInput, week: SchedulerWeek, now: Date): PlacementResult {
  try {
    const zone = week.settings.timeZone;
    const session = input.kind === 'assignment' ? undefined : input.session;
    const interval = session?.actual?.status !== 'skipped' && session?.actual ? session.actual : session?.plan;
    if (session && !interval) return { kind: 'invalid', message: 'This session has no interval to place.' };

    let start: string;
    let end: string;
    let date: string;
    if (input.kind === 'resize') {
      if (!interval) return { kind: 'invalid', message: 'This session has no interval to resize.' };
      date = session?.actual?.date ?? session?.date ?? '';
      start = new Date(Date.parse(interval.start) + (input.edge === 'start' ? input.deltaMinutes * 60000 : 0)).toISOString();
      end = new Date(Date.parse(interval.end) + (input.edge === 'end' ? input.deltaMinutes * 60000 : 0)).toISOString();
    } else {
      date = input.targetDate;
      start = localInstant(dateForMinute(date, input.startMinute), clockLabel(input.startMinute).slice(0, 5), zone);
      const duration = input.kind === 'assignment' ? input.durationMinutes : interval ? (Date.parse(interval.end) - Date.parse(interval.start)) / 60000 : 0;
      end = new Date(Date.parse(start) + duration * 60000).toISOString();
    }
    const startMs = Date.parse(start), endMs = Date.parse(end);
    if (!Number.isFinite(startMs) || !Number.isFinite(endMs) || endMs <= startMs) return { kind: 'invalid', message: 'End must be after start.' };
    const mode = session ? (!session.actual && session.plan && Date.parse(session.plan.start) > now.getTime() ? 'plan' : 'actual') : startMs > now.getTime() ? 'plan' : 'actual';
    if (mode === 'actual' && endMs > now.getTime()) return { kind: 'invalid', message: 'Actual work cannot end in the future.' };
    if (mode === 'plan') {
      if (startMs <= now.getTime()) return { kind: 'invalid', message: 'Plans must start in the future.' };
      const day = week.days.find(value => value.date === date);
      if (!day || startMs < Date.parse(day.start) || endMs > Date.parse(day.end)) return { kind: 'invalid', message: 'Planned sessions must fit inside one scheduling day.' };
      const goalId = input.kind === 'assignment' ? input.assignment.goalId : session?.assignment.goalId;
      if (goalId) {
        const goal = week.goals.find(value => value.goal.id === goalId)?.goal;
        if (!goal || date < goal.startDate || date > goal.endDate || goal.eligibleFrom && date < goal.eligibleFrom || goal.stoppedDate && date > goal.stoppedDate || goal.pauses?.some(pause => date >= pause.from && date <= pause.to)) {
          return { kind: 'invalid', message: 'This goal is not eligible on the selected date.' };
        }
      }
    }
    const startFields = localFields(start, zone), endFields = localFields(end, zone);
    const assignment = input.kind === 'assignment' ? input.assignment : input.session.assignment;
    const draft: SessionDraft = { id: session?.id, ruleId: session?.ruleId, assignment, date, startDate: startFields.date, start: startFields.time, endDate: endFields.date, end: endFields.time, originalStart: start, originalEnd: end, mode, repeat: false, scope: 'date' };
    const conflictIds = mode === 'actual'
      ? week.sessions.filter(value => value.id !== session?.id && value.actual && value.actual.status !== 'skipped' && overlaps(startMs, endMs, value.actual.start, value.actual.end)).map(value => value.id)
      : [
        ...week.sessions.filter(value => value.id !== session?.id && value.state === 'accepted' && value.plan && overlaps(startMs, endMs, value.plan.start, value.plan.end)).map(value => value.id),
        ...week.busy.filter(value => overlaps(startMs, endMs, value.start, value.end)).map(value => value.id),
      ];
    return { kind: 'valid', draft, start, end, conflictIds };
  } catch (error) {
    return { kind: 'invalid', message: error instanceof Error ? error.message : 'Could not place the session.' };
  }
}
