import assert from 'node:assert/strict';
import test from 'node:test';
import { agentEndpoint, agentFileName, agentStats, padCount, previewText } from '../src/lib/agent-view.ts';
import type { Goal } from '../src/lib/timeline.ts';

const goal = (id: string, status: Goal['status'], endDate: string, steps: boolean[]): Goal => ({
  id, title: id, status, dependsOn: [], startDate: '2026-01-01', endDate, color: '#fff',
  createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z', notes: [], metadata: {},
  steps: steps.map((done, i) => ({ id: `${id}-${i}`, title: `Step ${i}`, done })),
});

test('agent stats count goals, active goals (late ones too) and every step', () => {
  const goals = [
    goal('a', 'active', '2026-12-31', [true, false]),
    goal('late', 'active', '2026-11-01', [false]),
    goal('p', 'planned', '2027-01-31', [false, false, false]),
    goal('d', 'done', '2026-10-01', []),
  ];
  assert.deepEqual(agentStats(goals, '2026-11-27'), { goals: 4, active: 2, steps: 6 });
  assert.deepEqual(agentStats([], '2026-11-27'), { goals: 0, active: 0, steps: 0 });
});

test('endpoint and file name follow the format', () => {
  assert.equal(agentEndpoint('markdown'), '/api/agent-context?format=markdown');
  assert.equal(agentEndpoint('json'), '/api/agent-context');
  assert.equal(agentFileName('markdown'), 'agent-context.md');
  assert.equal(agentFileName('json'), 'agent-context.json');
  assert.equal(padCount(6), '06');
  assert.equal(padCount(123), '123');
});

test('the preview keeps short output whole and cuts long output on a line', () => {
  assert.deepEqual(previewText('a\nb'), { text: 'a\nb', truncated: false, hiddenChars: 0 });
  const lines = Array.from({ length: 100 }, (_, i) => `line ${i}`).join('\n');
  const byLines = previewText(lines, 3);
  assert.equal(byLines.text, 'line 0\nline 1\nline 2');
  assert.equal(byLines.truncated, true);
  assert.equal(byLines.hiddenChars, lines.length - byLines.text.length);
  const byChars = previewText('aaaa\nbbbb\ncccc', 60, 7);
  assert.equal(byChars.text, 'aaaa');
  assert.equal(previewText('x'.repeat(20), 60, 5).text, 'xxxxx');
});
