import assert from 'node:assert/strict';
import test from 'node:test';
import { appendCompanyNote, buildBoard, type RawCompany } from '../src/lib/companies.ts';
import { toggleTask } from '../src/lib/checklist.ts';
const company: RawCompany = {
  slug: 'test', title: 'Test', type: 'company', category: 'Infra', url: 'https://example.com', status: 'not_started', priority: 'high', featured: false, tags: [],
  body: '## Why\nA reason\n\n## Steps\n- [ ] Apply\n- [x] Research\n\n## Log\n- Earlier note\n\n## Other\nKeep this',
};
test('company steps use source indices and preserve notes', () => {
  const before = buildBoard([company]).categories[0].companies[0];
  const changed = { ...company, body: toggleTask(company.body, before.steps[0].index, true) };
  const after = buildBoard([changed]).categories[0].companies[0];
  assert.equal(after.completed, 2);
  assert.deepEqual(after.logEntries, ['Earlier note']);
  assert.equal(toggleTask(changed.body, after.steps[0].index, false), company.body);
});
test('quick company notes preserve multiline text, earlier notes, steps and later sections', () => {
  const body = appendCompanyNote(company.body, 'Follow up\nAfter the meetup', '2026-09-13');
  const parsed = buildBoard([{ ...company, body }]).categories[0].companies[0];
  assert.deepEqual(parsed.logEntries, ['2026-09-13: Follow up\nAfter the meetup', 'Earlier note']);
  assert.equal(parsed.completed, 1);
  assert.ok(body.endsWith('## Other\nKeep this'));
});
test('quick notes create a missing Log section and ignore blank notes', () => {
  const body = '## Steps\n- [ ] Apply';
  assert.equal(appendCompanyNote(body, '   ', '2026-09-13'), body);
  assert.deepEqual(buildBoard([{ ...company, body: appendCompanyNote(body, 'New note', '2026-09-13') }]).categories[0].companies[0].logEntries, ['2026-09-13: New note']);
});

test('research follows relationship steps for saved companies and toggles retain source indices', () => {
  const body = '## Steps\n- [x] Research team & open roles\n- [ ] Identify one person at the company to connect with\n- [ ] Reach out and start building a relationship\n- [ ] Apply';
  const steps = buildBoard([{ ...company, body }]).categories[0].companies[0].steps;
  assert.deepEqual(steps.map(step => step.label), ['Identify one person at the company to connect with', 'Reach out and start building a relationship', 'Research team & open roles', 'Apply']);
  assert.equal(steps[2].completed, true);
  assert.match(toggleTask(body, steps[2].index, false), /- \[ \] Research team & open roles/);
});
