import type { SchedulerWeek } from './scheduler.ts';
import { dayNumber, timelineHours, type Goal } from './timeline.ts';

/** Recorded scheduler work for one goal: all-time total and totals keyed by week Monday. */
export interface GoalHours { totalHours: number; weekHours: Record<string, number> }

const closed = (goal: Goal) => goal.status === 'done' || goal.status === 'dropped';

/** Open goals whose dates cover today, plus any goal marked active; soonest end first. */
export function currentGoals(goals: Goal[], today: string): Goal[] {
  return goals
    .filter(goal => !closed(goal) && (goal.status === 'active' || (goal.startDate <= today && goal.endDate >= today)))
    .sort((a, b) => a.endDate.localeCompare(b.endDate) || a.title.localeCompare(b.title));
}

/** Open goals that start after today and within the next `days` days. */
export function upcomingGoals(goals: Goal[], today: string, days = 14): Goal[] {
  const limit = dayNumber(today) + days;
  return goals
    .filter(goal => !closed(goal) && goal.status !== 'active' && goal.startDate > today && dayNumber(goal.startDate) <= limit)
    .sort((a, b) => a.startDate.localeCompare(b.startDate) || a.title.localeCompare(b.title));
}

/** Expected hours on the goal's selected weekdays between two inclusive dates, or undefined without an estimate. */
export function expectedHours(goal: Goal, from: string, to: string): number | undefined {
  if (goal.dailyHours === undefined) return undefined;
  return timelineHours([goal], Math.max(dayNumber(from), dayNumber(goal.startDate)), Math.min(dayNumber(to), dayNumber(goal.endDate)) + 1).total;
}

export interface GoalPace {
  logged: number;
  thisWeek: number;
  expectedToDate?: number;
  plannedTotal?: number;
  stepsDone: number;
  stepsTotal: number;
}
export function goalPace(goal: Goal, hours: GoalHours | undefined, today: string, weekStart: string): GoalPace {
  return {
    logged: hours?.totalHours ?? 0,
    thisWeek: hours?.weekHours[weekStart] ?? 0,
    expectedToDate: today < goal.startDate ? 0 : expectedHours(goal, goal.startDate, today),
    plannedTotal: expectedHours(goal, goal.startDate, goal.endDate),
    stepsDone: goal.steps.filter(step => step.done).length,
    stepsTotal: goal.steps.length,
  };
}

/**
 * Finds the latest `minutes`-long interval on `date` that ends no later than now and
 * the end of the scheduling day and does not overlap recorded work, which the scheduler
 * rejects. Returns ISO instants, or null when the day has no such gap.
 */
export function logSlot(week: SchedulerWeek, date: string, now: Date, minutes: number): { start: string; end: string } | null {
  const day = week.days.find(item => item.date === date);
  if (!day?.valid || minutes <= 0) return null;
  const dayStart = Date.parse(day.start), length = minutes * 60000;
  const recorded = week.sessions.flatMap(session => session.actual && session.actual.status !== 'skipped' ? [{ start: Date.parse(session.actual.start), end: Date.parse(session.actual.end) }] : []);
  let end = Math.floor(Math.min(Date.parse(day.end), now.getTime()) / 60000) * 60000;
  while (end - length >= dayStart) {
    const start = end - length;
    const overlapping = recorded.filter(item => item.start < end && start < item.end);
    if (!overlapping.length) return { start: new Date(start).toISOString(), end: new Date(end).toISOString() };
    end = Math.min(...overlapping.map(item => item.start));
  }
  return null;
}

export function formatHours(hours: number): string {
  return hours.toLocaleString('en-US', { maximumFractionDigits: 2 });
}

/** One daily journal line; the "Goal:" label becomes a facet in the Logs explorer. */
export function journalLine({ goalTitle, stepTitle, hours, note }: { goalTitle?: string; stepTitle?: string; hours: number; note: string }): string {
  const text = note.trim();
  if (!goalTitle) return text;
  return `Goal: ${goalTitle}${stepTitle ? ` › ${stepTitle}` : ''}${hours > 0 ? ` (${formatHours(hours)}h)` : ''}${text ? ` — ${text}` : ''}`;
}

/** Note body saved on the timeline goal. */
export function goalNoteBody({ date, stepTitle, hours, note }: { date: string; stepTitle?: string; hours: number; note: string }): string {
  const parts = [date, stepTitle, hours > 0 ? `${formatHours(hours)}h logged` : ''].filter(Boolean).join(' · ');
  return note.trim() ? `${parts} — ${note.trim()}` : parts;
}

/** Totals across every goal in a scheduler week. */
export function weekTotals(week: SchedulerWeek) {
  return week.goals.reduce((sum, item) => ({
    required: sum.required + (item.requiredHours ?? 0),
    actual: sum.actual + item.actualHours,
    scheduled: sum.scheduled + item.remainingScheduledHours,
    uncovered: sum.uncovered + item.uncoveredHours,
  }), { required: 0, actual: 0, scheduled: 0, uncovered: 0 });
}

export interface SaveStep { key: string; label: string; run: () => Promise<void> }
/**
 * Runs save steps in order, skipping any that `done` already records, so retrying
 * after a partial failure never repeats a saved step. Finished steps are added to `done`.
 */
export async function runSaveSteps(steps: SaveStep[], done: Map<string, string>): Promise<void> {
  for (const step of steps) {
    if (done.has(step.key)) continue;
    await step.run();
    done.set(step.key, step.label);
  }
}
