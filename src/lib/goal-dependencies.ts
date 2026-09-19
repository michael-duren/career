import { dayNumber, type Goal } from './timeline.ts';
export const finished = (g: Goal) => g.status === 'done' || g.status === 'dropped';
export function dependencyState(goal: Goal, goals: Goal[]) {
  const prerequisites = (goal.dependsOn ?? []).map(id => goals.find(g => g.id === id)).filter((g): g is Goal => !!g);
  const blockers = prerequisites.filter(g => !finished(g));
  const missing = (goal.dependsOn ?? []).filter(id => !goals.some(g => g.id === id));
  return { ready: !finished(goal) && !blockers.length && !missing.length,
    blocked: !finished(goal) && (!!blockers.length || !!missing.length), blockers, missing,
    dropped: prerequisites.filter(g => g.status === 'dropped'),
    conflicts: prerequisites.filter(g => goal.startDate <= g.endDate) };
}
export function dependencyCycle(goals: Goal[], draft: Goal): string[] | null {
  const byId = new Map([...goals, draft].map(g => [g.id, g]));
  const visited = new Set<string>();
  function visit(id: string, path: string[]): string[] | null {
    if (path.includes(id)) return [...path.slice(path.indexOf(id)), id];
    if (visited.has(id)) return null;
    for (const next of byId.get(id)?.dependsOn ?? []) { const cycle = visit(next, [...path, id]); if (cycle) return cycle; }
    visited.add(id); return null;
  }
  return visit(draft.id, []);
}
export function criticalPath(goals: Goal[], target: string): { ids: string[]; days: number } {
  const byId = new Map(goals.map(g => [g.id, g]));
  const memo = new Map<string, { ids: string[]; days: number }>();
  const visiting = new Set<string>();
  function walk(id: string): { ids: string[]; days: number } {
    if (visiting.has(id)) throw new Error('Dependency cycle');
    if (memo.has(id)) return memo.get(id)!;
    const goal = byId.get(id); if (!goal) return { ids: [], days: 0 };
    visiting.add(id);
    const best = (goal.dependsOn ?? []).map(walk).reduce((a, b) => b.days > a.days ? b : a, { ids: [] as string[], days: 0 });
    const result = { ids: [...best.ids, id], days: best.days + dayNumber(goal.endDate) - dayNumber(goal.startDate) + 1 };
    visiting.delete(id); memo.set(id, result); return result;
  }
  return walk(target);
}
export function statusSuggestions(goal: Goal, today: string) {
  if (finished(goal)) return [];
  const suggestions: { status: Goal['status']; label: string }[] = [];
  if (goal.steps.length && goal.steps.every(s => s.done)) suggestions.push({ status: 'done', label: 'All steps done, mark as done?' });
  if (goal.status === 'planned' && goal.startDate < today) suggestions.push({ status: 'active', label: 'Start date passed, mark as active?' });
  return suggestions;
}
