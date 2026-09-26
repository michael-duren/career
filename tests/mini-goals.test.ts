import { test } from 'node:test';
import assert from 'node:assert/strict';
import { applyStepMove, newMiniGoal, stepMove } from '../src/lib/mini-goals.ts';
import type { Goal } from '../src/lib/timeline.ts';
const goal = (id: string, ...steps: string[]): Goal => ({ id, title: id, status: 'planned', dependsOn: [], startDate: '2026-01-01', endDate: '2026-01-02', color: '#ffffff', createdAt: '', updatedAt: '', notes: [], metadata: {}, steps: steps.map(s => ({ id: s, title: s, done: false })) });
const titles = (goals: Goal[], id: string) => goals.find(g => g.id === id)!.steps.map(s => s.id);
test('new mini goals are trimmed, undone and have their own ID', () => {
  const a = newMiniGoal('  Draft outline '), b = newMiniGoal('Draft outline');
  assert.equal(a.title, 'Draft outline'); assert.equal(a.done, false); assert.notEqual(a.id, b.id);
});
test('dropping within a goal accounts for the removed slot', () => {
  const goals = [goal('a', 'a1', 'a2', 'a3')];
  assert.deepEqual(stepMove(goals, 'a1', 'a', 3), { stepId: 'a1', from: 'a', to: 'a', index: 2 });
  assert.deepEqual(stepMove(goals, 'a3', 'a', 0), { stepId: 'a3', from: 'a', to: 'a', index: 0 });
  assert.equal(stepMove(goals, 'a2', 'a', 1), null, 'dropping before itself is a no-op');
  assert.equal(stepMove(goals, 'a2', 'a', 2), null, 'dropping just after itself is a no-op');
  assert.deepEqual(titles(applyStepMove(goals, stepMove(goals, 'a1', 'a', 3)!), 'a'), ['a2', 'a3', 'a1']);
  assert.deepEqual(titles(applyStepMove(goals, stepMove(goals, 'a3', 'a', 1)!), 'a'), ['a1', 'a3', 'a2']);
});
test('moving between goals keeps the step and clamps the slot', () => {
  const goals = [goal('a', 'a1', 'a2'), goal('b', 'b1'), goal('c')];
  const move = stepMove(goals, 'a2', 'b', 0)!;
  assert.deepEqual(move, { stepId: 'a2', from: 'a', to: 'b', index: 0 });
  const moved = applyStepMove(goals, move);
  assert.deepEqual(titles(moved, 'a'), ['a1']); assert.deepEqual(titles(moved, 'b'), ['a2', 'b1']);
  assert.equal(moved[2], goals[2], 'untouched goals keep their identity');
  assert.deepEqual(stepMove(goals, 'a1', 'c', 9), { stepId: 'a1', from: 'a', to: 'c', index: 0 });
  assert.equal(stepMove(goals, 'missing', 'b', 0), null);
  assert.equal(stepMove(goals, 'a1', 'missing', 0), null);
  assert.equal(applyStepMove(goals, { stepId: 'a1', from: 'a', to: 'missing', index: 0 }), goals);
});
