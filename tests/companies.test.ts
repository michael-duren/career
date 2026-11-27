import assert from 'node:assert/strict';
import test from 'node:test';
import { appendCompanyNote, buildBoard, companyDomain, companyLogoUrl, matchesCompanySearch, sortCompanies, type RawCompany } from '../src/lib/companies.ts';
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

test('company logo domain comes from the URL host, without www', () => {
  assert.equal(companyDomain('https://www.Grafana.com/careers?x=1'), 'grafana.com');
  assert.equal(companyDomain('http://jobs.example.co.uk'), 'jobs.example.co.uk');
  assert.equal(companyDomain('https://datadoghq.com.'), 'datadoghq.com');
  for (const url of [undefined, '', 'not a url', 'mailto:hr@example.com', 'https://localhost:8080/', 'ftp://example.com', 'http://192.168.1.1/', 'http://[::1]/']) assert.equal(companyDomain(url), undefined, String(url));
  assert.equal(companyLogoUrl('https://www.fly.io/jobs'), '/api/companies/logo?domain=fly.io');
  assert.equal(companyLogoUrl('nope'), undefined);
});

test('company search matches every word against name, category, tags and why', () => {
  const [parsed] = buildBoard([{ ...company, title: 'Grafana Labs', category: 'Observability / Infra', tags: ['remote', 'go'], body: '## Why\nLoki and Tempo teams.\n' }]).categories[0].companies;
  for (const query of ['', '  ', 'grafana', 'LABS', 'observability', 'remote', 'tempo', 'grafana remote', 'loki infra']) assert.equal(matchesCompanySearch(parsed, query), true, query);
  for (const query of ['datadog', 'grafana datadog', 'onsite']) assert.equal(matchesCompanySearch(parsed, query), false, query);
});

test('companies sort by priority, then featured, then name', () => {
  const sorted = sortCompanies([
    { title: 'Low', priority: 'low', featured: true },
    { title: 'Beta', priority: 'high', featured: false },
    { title: 'Alpha', priority: 'high', featured: false },
    { title: 'Zeta', priority: 'high', featured: true },
    { title: 'Mid', priority: 'medium', featured: false },
  ] as const);
  assert.deepEqual(sorted.map(c => c.title), ['Zeta', 'Alpha', 'Beta', 'Mid', 'Low']);
});
