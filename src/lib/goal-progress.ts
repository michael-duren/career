import { dayNumber, timelineHours, type Goal } from './timeline.ts';

/** Recorded scheduler work for one goal: all-time total and totals keyed by week Monday. */
export interface GoalHours { totalHours: number; weekHours: Record<string, number> }

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

export function formatHours(hours: number): string {
  return hours.toLocaleString('en-US', { maximumFractionDigits: 2 });
}
