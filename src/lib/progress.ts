export const PLAN_START = new Date('2026-04-20T00:00:00Z');
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
