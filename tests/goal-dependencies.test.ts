import { test } from 'node:test';
import assert from 'node:assert/strict';
import { dependencyState, dependencyCycle, criticalPath, statusSuggestions } from '../src/lib/goal-dependencies.ts';
import type { Goal } from '../src/lib/timeline.ts';
const goal = (id: string, days = 1, dependsOn: string[] = [], status: Goal['status'] = 'planned'): Goal => ({ id, title: id, status, dependsOn, startDate: '2026-01-01', endDate: `2026-01-${String(days).padStart(2, '0')}`, color: '#ffffff', createdAt: '', updatedAt: '', notes: [], steps: [], metadata: {} });
test('ready, blocked, dropped warnings and inclusive date conflicts', () => {
 const a = goal('a'), b = goal('b', 2, ['a']);
 assert.equal(dependencyState(a, [a,b]).ready, true);
 assert.equal(dependencyState(b, [a,b]).blocked, true);
 assert.deepEqual(dependencyState(b, [a,b]).blockers, [a]);
 assert.deepEqual(dependencyState(b, [a,b]).conflicts, [a]);
 a.status = 'dropped';
 assert.equal(dependencyState(b, [a,b]).ready, true);
 assert.deepEqual(dependencyState(b, [a,b]).dropped, [a]);
 b.status = 'done'; assert.equal(dependencyState(b, [a,b]).ready, false); assert.equal(dependencyState(b, [a,b]).blocked, false);
});
test('longest chain includes done goals and inclusive calendar-day weights', () => {
 const goals = [goal('a', 3, [], 'done'), goal('b', 2, ['a']), goal('c', 7), goal('d', 4, ['b','c'])];
 assert.deepEqual(criticalPath(goals, 'd'), { ids: ['c','d'], days: 11 });
 goals[1].endDate = '2026-01-08';
 assert.deepEqual(criticalPath(goals, 'd'), { ids: ['a','b','d'], days: 15 });
});
test('cycle and self-edge checks, missing prerequisites and manual suggestions', () => {
 const a = goal('a'), b = goal('b', 1, ['a']);
 assert.deepEqual(dependencyCycle([a,b], {...a, dependsOn: ['b']}), ['a','b','a']);
 assert.deepEqual(dependencyCycle([a], {...a, dependsOn: ['a']}), ['a','a']);
 assert.equal(dependencyCycle([a,b], b), null);
 assert.equal(dependencyState(goal('c', 1, ['missing']), []).blocked, true);
 assert.equal(statusSuggestions(a, '2026-01-02')[0].status, 'active');
 a.steps = [{id:'x', title:'step', done:true}]; assert.equal(statusSuggestions(a, '2026-01-02')[0].status, 'done');
});
