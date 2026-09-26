import type { Goal } from './timeline.ts';

/** Mini goals are a goal's steps: ordered, dateless sub-goals persisted with the goal. */
export type MiniGoal = Goal['steps'][number];
export type StepMove = { stepId: string; from: string; to: string; index: number };

export const newMiniGoal = (title: string): MiniGoal => ({ id: crypto.randomUUID(), title: title.trim(), done: false });

/**
 * Convert "drop before the item at slot" (slot = items.length appends) into the
 * step's index after the move, matching the server. Returns null for a no-op.
 */
export function stepMove(goals: Goal[], stepId: string, to: string, slot: number): StepMove | null {
  const from = goals.find(g => g.steps.some(s => s.id === stepId));
  const target = goals.find(g => g.id === to);
  if (!from || !target) return null;
  const current = from.steps.findIndex(s => s.id === stepId);
  const bounded = Math.max(0, Math.min(slot, target.steps.length));
  if (from.id !== to) return { stepId, from: from.id, to, index: bounded };
  const index = bounded > current ? bounded - 1 : bounded;
  return index === current ? null : { stepId, from: from.id, to, index };
}

/** Apply a move locally, as the server does, for previews and tests. */
export function applyStepMove(goals: Goal[], move: StepMove): Goal[] {
  const step = goals.find(g => g.id === move.from)?.steps.find(s => s.id === move.stepId);
  if (!step || !goals.some(g => g.id === move.to)) return goals;
  return goals.map(goal => {
    let steps = goal.id === move.from ? goal.steps.filter(s => s.id !== move.stepId) : goal.steps;
    if (goal.id === move.to) steps = [...steps.slice(0, move.index), step, ...steps.slice(move.index)];
    return steps === goal.steps ? goal : { ...goal, steps };
  });
}
