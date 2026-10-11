export const PLAN_START = new Date('2026-04-20T00:00:00Z');
export const APPLICATION_DATE = new Date('2027-01-01T00:00:00Z');
export const TRACK_KEYS = ['ostep', 'ebpf', 'database', 'systemDesign'] as const;
export type TrackKey = (typeof TRACK_KEYS)[number];
export const TRACK_META: Record<string, { name: string; color: string }> = {
  ostep: { name: 'OSTEP', color: '#a78bfa' },
  ebpf: { name: 'eBPF', color: '#60a5fa' },
  database: { name: 'Database project', color: '#fbbf24' },
  systemDesign: { name: 'System Design', color: '#34d399' },
  algorithms: { name: 'Algorithms (historical)', color: '#94a3b8' },
  osOss: { name: 'OS & OSS (historical)', color: '#a1a1aa' },
  languages: { name: 'Languages (historical)', color: '#d6d3d1' },
};
export interface WeekEntry {
  week: number;
  year: number;
  dates: string;
  slug: string;
  body: string;
  hours: Record<string, number>;
  tags: string[];
  targets?: Record<string, number>;
}
export function weekNumberFor(date: Date): number {
  return Math.floor((date.getTime() - PLAN_START.getTime()) / (7 * 86400000)) + 1;
}
export function totalHours(week: WeekEntry): number {
  return Object.values(week.hours).reduce((sum, hours) => sum + hours, 0);
}
export interface GoalTrack { id: string; title: string; color: string; weekHours: Record<string, number> }
export const goalTrackKey = (id: string) => `goal:${id}`;
const isoDate = (date: Date) => date.toISOString().slice(0, 10);

/**
 * Adds hours recorded in the weekly scheduler to journal weeks as goal tracks,
 * creating weeks that have scheduler work but no journal entry.
 */
export function withGoalHours(weeks: WeekEntry[], goals: GoalTrack[]): WeekEntry[] {
  const byStart = new Map(weeks.map(week => [week.dates.slice(0, 10), { ...week, hours: { ...week.hours } }]));
  for (const goal of goals) {
    for (const [monday, hours] of Object.entries(goal.weekHours)) {
      const start = new Date(`${monday}T00:00:00Z`);
      if (!(hours > 0) || start < PLAN_START) continue;
      let week = byStart.get(monday);
      if (!week) {
        const number = weekNumberFor(start);
        week = { week: number, year: start.getUTCFullYear(), dates: `${monday} to ${isoDate(new Date(start.getTime() + 6 * 86400000))}`, slug: `scheduler/${monday}`, body: '', hours: {}, tags: [] };
        byStart.set(monday, week);
      }
      week.hours[goalTrackKey(goal.id)] = (week.hours[goalTrackKey(goal.id)] ?? 0) + hours;
    }
  }
  return [...byStart.values()];
}

export function buildDashboard(weeks: WeekEntry[], now = new Date(), goals: GoalTrack[] = []) {
  // With goal tracks, the older journal tracks are history: label them apart from same-named goals.
  const legacy = goals.length ? Object.fromEntries(Object.entries(TRACK_META).map(([key, value]) => [key, { ...value, name: value.name.includes('(historical)') ? value.name : `${value.name} (journal)` }])) : TRACK_META;
  const meta: Record<string, { name: string; color: string }> = { ...legacy, ...Object.fromEntries(goals.map(goal => [goalTrackKey(goal.id), { name: goal.title, color: goal.color }])) };
  const currentWeek = Math.max(1, weekNumberFor(now));
  const sorted = weeks.filter(w => new Date(w.dates.slice(0, 10) + 'T00:00:00Z') <= now).sort((a, b) => a.dates.localeCompare(b.dates));
  const current = sorted.find(w => w.week === currentWeek);
  const recent = Array.from({ length: Math.min(4, currentWeek) }, (_, i) => sorted.find(w => w.week === currentWeek - i));
  const keys = [...new Set([...(goals.length ? [] : TRACK_KEYS), ...sorted.flatMap(w => Object.keys(w.hours).filter(k => w.hours[k] > 0))])];
  const tracks = keys.map(key => ({ key, ...(meta[key] ?? { name: key, color: '#fb7185' }),
    total: sorted.reduce((sum, w) => sum + (w.hours[key] ?? 0), 0),
    current: current?.hours[key] ?? 0,
    target: current?.targets?.[key],
    values: sorted.map(w => w.hours[key] ?? 0),
  }));
  return {
    currentWeek,
    daysToApplication: Math.max(0, Math.ceil((APPLICATION_DATE.getTime() - now.getTime()) / 86400000)),
    totalHours: sorted.reduce((sum, w) => sum + totalHours(w), 0),
    weeksLogged: sorted.filter(w => totalHours(w) > 0).length,
    recentAverage: recent.reduce((sum, w) => sum + (w ? totalHours(w) : 0), 0) / (recent.length || 1),
    currentHours: current ? totalHours(current) : 0,
    currentDates: current?.dates ?? '',
    tracks, weeks: sorted,
  };
}
export type DashboardData = ReturnType<typeof buildDashboard>;
