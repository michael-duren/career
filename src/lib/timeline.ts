export interface Goal {
  id: string;
  status: 'planned' | 'active' | 'done' | 'dropped';
  dependsOn: string[];
  title: string;
  startDate: string;
  endDate: string;
  color: string;
  dailyHours?: number;
  selectedWeekdays?: number[];
  createdAt: string;
  updatedAt: string;
  notes: { id: string; body: string; createdAt: string }[];
  metadata: Record<string, string>;
  steps: { id: string; title: string; done: boolean }[];
}
export const DAY = 86400000;
export const dayNumber = (date: string) => Date.parse(`${date}T00:00:00Z`) / DAY;
export const dayString = (day: number) => new Date(day * DAY).toISOString().slice(0, 10);
const allWeekdays = [1, 2, 3, 4, 5, 6, 7];
export function goalWeekdays(goal: Goal): number[] {
  const days = goal.selectedWeekdays ?? allWeekdays;
  if (days.some(day => !Number.isInteger(day) || day < 1 || day > 7) || new Set(days).size !== days.length) {
    throw new TypeError('selectedWeekdays must contain unique ISO weekdays from 1 through 7');
  }
  return days;
}
const isoWeekday = (day: number) => ((day + 3) % 7 + 7) % 7 + 1;
function selectedDayCount(selected: number[], start: number, end: number): number {
  const length = Math.max(0, end - start);
  const fullWeeks = Math.floor(length / 7);
  let count = fullWeeks * selected.length;
  for (let offset = fullWeeks * 7; offset < length; offset++) {
    if (selected.includes(isoWeekday(start + offset))) count++;
  }
  return count;
}
export function shiftGoal(goal: Goal, days: number, mode: 'move' | 'start' | 'end'): Goal {
  const start = dayNumber(goal.startDate), end = dayNumber(goal.endDate);
  const lower = dayNumber('1900-01-01'), upper = dayNumber('2200-12-31');
  const delta = Math.max(lower - start, Math.min(upper - end, days));
  return { ...goal,
    startDate: dayString(mode === 'move' ? start + delta : mode === 'start' ? Math.max(lower, Math.min(end, start + days)) : start),
    endDate: dayString(mode === 'move' ? end + delta : mode === 'end' ? Math.min(upper, Math.max(start, end + days)) : end),
  };
}
export function monthOffset(date: string, months: number): string {
  const parsed = new Date(`${date}T00:00:00Z`);
  const day = parsed.getUTCDate();
  parsed.setUTCDate(1);
  parsed.setUTCMonth(parsed.getUTCMonth() + months);
  const last = new Date(Date.UTC(parsed.getUTCFullYear(), parsed.getUTCMonth() + 1, 0)).getUTCDate();
  parsed.setUTCDate(Math.min(day, last));
  return parsed.toISOString().slice(0, 10);
}

/** Inclusive date ranges; split only at goal boundaries, even for multi-year plans. */
export function workloadFor(goals: Goal[], draft: Goal) {
  const start = dayNumber(draft.startDate), end = dayNumber(draft.endDate) + 1;
  const others = goals.filter(g => g.id !== draft.id && dayNumber(g.startDate) < end && dayNumber(g.endDate) >= start);
  const all = [...others, draft];
  const policies = all.map(goal => ({ goal, start: dayNumber(goal.startDate), end: dayNumber(goal.endDate) + 1, selected: goalWeekdays(goal) }));
  const boundaries = [...new Set([start, end, ...policies.flatMap(policy => [Math.max(start, policy.start), Math.min(end, policy.end)])])].sort((a, b) => a - b);
  const segments: { start: number; end: number; weekdays: number[]; count: number; hours: number; unknown: number }[] = [];
  for (let index = 0; index < boundaries.length - 1; index++) {
    const from = boundaries[index], to = boundaries[index + 1];
    const active = policies.filter(policy => policy.start <= from && policy.end >= to);
    const patterns = new Map<string, { weekdays: number[]; count: number; hours: number; unknown: number }>();
    for (const weekday of allWeekdays) {
      const required = active.filter(policy => policy.selected.includes(weekday));
      const count = required.length;
      const hours = required.reduce((sum, policy) => sum + (policy.goal.dailyHours ?? 0), 0);
      const unknown = required.filter(policy => policy.goal.dailyHours === undefined).length;
      const key = `${count}:${hours}:${unknown}`;
      const pattern = patterns.get(key);
      if (pattern) pattern.weekdays.push(weekday);
      else patterns.set(key, { weekdays: [weekday], count, hours, unknown });
    }
    for (const pattern of patterns.values()) {
      const previous = segments.at(-1);
      if (previous && previous.end === from && previous.count === pattern.count && previous.hours === pattern.hours
        && previous.unknown === pattern.unknown && previous.weekdays.join() === pattern.weekdays.join()) previous.end = to;
      else segments.push({ start: from, end: to, ...pattern });
    }
  }
  return { others, segments, peakHours: Math.max(0, ...segments.map(s => s.hours)), peakCount: Math.max(0, ...segments.map(s => s.count)) };
}

/** Average over every visible calendar day, including days with no goals. End is exclusive. */
export function timelineHours(goals: Goal[], start: number, end: number) {
  let total = 0, unknown = 0;
  for (const goal of goals) {
    const from = Math.max(start, dayNumber(goal.startDate)), to = Math.min(end, dayNumber(goal.endDate) + 1);
    const selected = goalWeekdays(goal);
    const applicable = selectedDayCount(selected, from, to);
    if (!applicable) continue;
    if (goal.dailyHours === undefined) unknown++;
    else total += applicable * goal.dailyHours;
  }
  return { total, average: end > start ? total / (end - start) : 0, unknown };
}
