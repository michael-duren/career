export const PLAN_START = new Date('2026-04-20T00:00:00Z');
export const TRACK_META: Record<string, { name: string }> = {
  ostep: { name: 'OSTEP' },
  ebpf: { name: 'eBPF' },
  database: { name: 'Database project' },
  systemDesign: { name: 'System Design' },
  algorithms: { name: 'Algorithms (historical)' },
  osOss: { name: 'OS & OSS (historical)' },
  languages: { name: 'Languages (historical)' },
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
