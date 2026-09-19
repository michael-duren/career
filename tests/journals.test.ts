import test from 'node:test';
import assert from 'node:assert/strict';
import { isEmptyStarterWeek, migrateJournals, newWeek, updateWorkspaceEntry, type Workspace } from '../src/lib/workspace.ts';
import { agentContext, contextMarkdown, journalMarkdown } from '../src/lib/agent-context.ts';
import { shiftGoal, type Goal } from '../src/lib/timeline.ts';

const workspace = (): Workspace => ({ version: 2, notes: [], documents: [], weeks: [], books: [], companies: [] });
const goal: Goal = { status: 'planned', dependsOn: [], id: 'goal-1', title: 'Build a database', startDate: '2026-09-01', endDate: '2026-12-01', color: '#67e8f9', createdAt: '2026-09-01T00:00:00Z', updatedAt: '2026-09-01T00:00:00Z', steps: [{ id: 'step-1', title: 'Implement Raft', done: false }], notes: [{ id: 'note-1', body: 'Keep scope small', createdAt: '2026-09-01T00:00:00Z' }], metadata: { priority: 'high' } };

test('cleanup removes only untouched scaffolding and preserves content and metrics', () => {
  const empty = newWeek('2026-09-12');
  assert.ok(isEmptyStarterWeek({ ...empty, body: '## What I did\n### Monday — Sep 7\n-\n## Blockers\n-\n## Notes\n' }));
  for (const patch of [{ body: '# My reflection' }, { body: 'A hard day' }, { hours: { ostep: 1 } }, { targets: { ostep: 2 } }, { tags: ['important'] }, { updatedAt: '2026-09-12' }]) {
    assert.equal(isEmptyStarterWeek({ ...empty, ...patch }), false);
  }
  const written = { ...empty, slug: 'written', body: 'Actual reflection' };
  const data = { ...workspace(), weeks: [empty, written], goals: [goal] };
  const migrated = migrateJournals(data, '# Biography\nPersonal history');
  assert.deepEqual(migrated.weeks, [written]);
  assert.deepEqual(migrated.goals, data.goals);
  assert.equal(migrated.personalJournal?.[0].body, '# Biography\nPersonal history');
  assert.equal(data.weeks.length, 2);
  const deleted = updateWorkspaceEntry(migrated, 'personal', 'bio');
  assert.deepEqual(migrateJournals(deleted, 'Do not resurrect').personalJournal, []);
});

test('company checklist migration puts relationship building before applying', () => {
  const data = { ...workspace(), companies: [{ slug: 'example', title: 'Example', category: 'Infra', type: 'company' as const, url: 'https://example.com', status: 'not_started' as const, featured: false, priority: 'high' as const, tags: [], body: '## Steps\n\n- [x] Research team & open roles\n- [ ] Tailor resume/cover letter\n- [ ] Submit application\n' }] };
  const migrated = migrateJournals(data, '');
  const steps = migrated.companies[0].body.split('\n').filter(line => line.startsWith('- [')).map(line => line.slice(6));
  assert.deepEqual(steps, ['Research team & open roles', 'Identify one person at the company to connect with', 'Reach out and start building a relationship', 'Tailor resume/cover letter', 'Submit application']);
});

test('personal entries remain separate from work entries and survive unrelated edits', () => {
  const data = migrateJournals({ ...workspace(), weeks: [{ ...newWeek('2026-09-12'), body: 'Work only' }] }, 'Life only');
  const updated = updateWorkspaceEntry(data, 'personal', 'day', { id: 'day', date: '2026-09-12', title: 'Life', description: '', tags: [], body: 'Personal only' });
  assert.deepEqual(updated.weeks, data.weeks);
  assert.match(journalMarkdown(updated, 'personal'), /Personal only/);
  assert.doesNotMatch(journalMarkdown(updated, 'work'), /Personal only|Life only/);
  assert.doesNotMatch(journalMarkdown(updated, 'personal'), /Work only/);
});

test('agent goals track moves, details, and deletions without reviving historical plans', () => {
  const data = { ...workspace(), goals: [goal] };
  const moved = { ...data, goals: [shiftGoal(goal, 30, 'move')] };
  const context = agentContext({ data: moved, revision: 'r2' });
  assert.equal(context.goals[0].startDate, '2026-10-01');
  assert.deepEqual(context.goals[0].steps, goal.steps);
  assert.match(contextMarkdown({ data: moved, revision: 'r2' }), /2026-10-01 → 2026-12-31/);
  assert.match(contextMarkdown({ data: moved, revision: 'r2' }), /Implement Raft/);
  assert.match(contextMarkdown({ data: moved, revision: 'r2' }), /Keep scope small/);
  const removed = { ...moved, goals: [] };
  assert.deepEqual(agentContext({ data: removed, revision: 'r3' }).goals, []);
  assert.doesNotMatch(contextMarkdown({ data: removed, revision: 'r3' }), /Build a database/);
});

test('complete context includes every journal and reference body without truncation', () => {
  const body = 'An entry\n```js\nexample\n```\n' + 'long text '.repeat(1000) + 'THE END';
  const data = migrateJournals(workspace(), body);
  const markdown = contextMarkdown({ data, revision: 'revision' });
  assert.ok(markdown.includes(body));
  assert.match(markdown, /````markdown/);
  assert.equal(agentContext({ data, revision: 'revision' }).journals.personal[0].body, body);
});
