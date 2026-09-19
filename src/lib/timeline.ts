export interface Goal {
  id: string;
  status: 'planned' | 'active' | 'done' | 'dropped';
  dependsOn: string[];
  title: string;
  startDate: string;
  endDate: string;
  color: string;
  dailyHours?: number;
  createdAt: string;
  updatedAt: string;
  notes: { id: string; body: string; createdAt: string }[];
  metadata: Record<string, string>;
  steps: { id: string; title: string; done: boolean }[];
}
export const DAY = 86400000;
export const dayNumber = (date: string) => Date.parse(`${date}T00:00:00Z`) / DAY;
export const dayString = (day: number) => new Date(day * DAY).toISOString().slice(0, 10);
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
  const boundaries = [...new Set([start, end, ...all.flatMap(g => [Math.max(start, dayNumber(g.startDate)), Math.min(end, dayNumber(g.endDate) + 1)])])].sort((a, b) => a - b);
  const segments = boundaries.slice(0, -1).map((from, i) => {
    const active = all.filter(g => dayNumber(g.startDate) <= from && dayNumber(g.endDate) >= from);
    return { start: from, end: boundaries[i + 1], count: active.length,
      hours: active.reduce((sum, g) => sum + (g.dailyHours ?? 0), 0), unknown: active.filter(g => g.dailyHours === undefined).length };
  });
  return { others, segments, peakHours: Math.max(0, ...segments.map(s => s.hours)), peakCount: Math.max(0, ...segments.map(s => s.count)) };
}

/** Average over every visible calendar day, including days with no goals. End is exclusive. */
export function timelineHours(goals: Goal[], start: number, end: number) {
  let total = 0, unknown = 0;
  for (const goal of goals) {
    const overlap = Math.max(0, Math.min(end, dayNumber(goal.endDate) + 1) - Math.max(start, dayNumber(goal.startDate)));
    if (!overlap) continue;
    if (goal.dailyHours === undefined) unknown++;
    else total += overlap * goal.dailyHours;
  }
  return { total, average: end > start ? total / (end - start) : 0, unknown };
}
