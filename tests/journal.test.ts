import test from 'node:test';
import assert from 'node:assert/strict';
import { hoursText, journalCounts, latestEntryDate, parseWeekBody, personalDate, personalMatches, sectionEmpty, staleNote, weekBounds, weekHoursLabel, weekMatches, weekRange } from '../src/lib/journal.ts';

const week = { week: 48, dates: '2026-11-23 to 2026-11-29', tags: ['infra'], body: '' };

test('week ranges read as short dates and keep both months across a month end', () => {
  assert.equal(weekRange('2026-11-23 to 2026-11-29'), 'Nov 23 – 29');
  assert.equal(weekRange('2026-07-27 to 2026-08-02'), 'Jul 27 – Aug 2');
  assert.equal(weekRange('2026-12-28 to 2027-01-03'), 'Dec 28 – Jan 3');
  assert.equal(weekRange('someday'), 'someday');
  assert.equal(weekRange('2026-02-30 to 2026-03-06'), '2026-02-30 to 2026-03-06');
  assert.deepEqual(weekBounds('2026-11-23 to 2026-11-29'), { start: '2026-11-23', end: '2026-11-29' });
  assert.equal(weekBounds('none'), null);
});

test('a week body splits into dated days, blockers and notes, always in that order', () => {
  const sections = parseWeekBody('## Notes\n- Remember RAID\n\n## What I did\n\n### Monday — Nov 23\n- Solved 3 graph problems\n- Read DDIA\n  chapter 5\n\n### Wednesday\n* CKA lab\n\n## Blockers\n-\n', '2026-11-23');
  assert.deepEqual(sections.map(section => section.kind), ['what', 'blockers', 'notes']);
  assert.deepEqual(sections[0].groups, [
    { label: 'Monday · Nov 23', date: '2026-11-23', items: ['Solved 3 graph problems', 'Read DDIA chapter 5'] },
    { label: 'Wednesday · Nov 25', date: '2026-11-25', items: ['CKA lab'] },
  ]);
  assert.equal(sectionEmpty(sections[1]), true, 'a lone "-" is the empty template, not an item');
  assert.deepEqual(sections[2].groups, [{ label: '', items: ['Remember RAID'] }]);
});

test('a heading date wins over its weekday, and undated weeks keep plain day labels', () => {
  // "Friday — Oct 16" written in the week of Oct 12: the weekday index would say Oct 16 too.
  assert.equal(parseWeekBody('## What I did\n### Friday — Oct 16\n- x\n', '2026-10-12')[0].groups[0].date, '2026-10-16');
  // A heading whose date sits outside its week keeps the written date.
  assert.equal(parseWeekBody('## What I did\n### Monday — Oct 19\n- x\n', '2026-10-12')[0].groups[0].date, '2026-10-19');
  assert.deepEqual(parseWeekBody('## What I did\n### Tuesday\n- x\n', null)[0].groups[0], { label: 'Tuesday', date: undefined, items: ['x'] });
  assert.deepEqual(parseWeekBody('## What I did\n### Misc\n- x\n', '2026-10-12')[0].groups[0], { label: 'Misc', items: ['x'] });
});

test('code, tables and custom sections are kept as Markdown instead of mangled into list items', () => {
  const sections = parseWeekBody('Intro line\n\n## What I did\n```go\n## not a heading\n```\n\n## Retro\n| a | b |\n', '2026-11-23');
  assert.equal(sections[0].rich, true);
  assert.match(sections[0].markdown, /## not a heading/);
  assert.equal(sections.length, 5, 'standard three, then the preamble and Retro in body order');
  assert.deepEqual(sections.slice(3).map(section => [section.title, section.rich]), [['', false], ['Retro', true]]);
  assert.deepEqual(sections[3].groups, [{ label: '', items: ['Intro line'] }]);
});

test('the latest entry is the newest written day, falling back to the start of a week with undated writing', () => {
  assert.equal(latestEntryDate([
    { dates: '2026-11-16 to 2026-11-22', body: '## What I did\n### Friday\n- done\n' },
    { dates: '2026-11-23 to 2026-11-29', body: '## What I did\n### Monday — Nov 23\n- a\n### Wednesday\n- b\n### Thursday\n' },
  ]), '2026-11-25', 'an empty Thursday heading does not count');
  assert.equal(latestEntryDate([{ dates: '2026-11-02 to 2026-11-08', body: '## Notes\n- undated thought\n' }]), '2026-11-02');
  assert.equal(latestEntryDate([{ dates: '2026-11-02 to 2026-11-08', body: '## What I did\n\n## Blockers\n\n## Notes\n' }]), null);
  assert.equal(latestEntryDate([]), null);
});

test('the catch-up note appears only when the latest entry is more than 7 days old', () => {
  assert.equal(staleNote('2026-10-16', '2026-11-25'), 'last entry Oct 16 — catch up?');
  assert.equal(staleNote('2026-11-18', '2026-11-25'), null, 'exactly 7 days is not stale');
  assert.equal(staleNote('2026-11-17', '2026-11-25'), 'last entry Nov 17 — catch up?');
  assert.equal(staleNote(null, '2026-11-25'), null);
});

test('journal search needs every word somewhere in the week or entry', () => {
  assert.equal(weekMatches({ ...week, body: 'Fixed the Alloy PR' }, 'alloy pr'), true);
  assert.equal(weekMatches({ ...week, body: 'Fixed the Alloy PR' }, 'alloy loki'), false);
  assert.equal(weekMatches(week, 'w48'), true);
  assert.equal(weekMatches(week, 'nov 23'), true);
  assert.equal(weekMatches(week, 'infra'), true);
  assert.equal(weekMatches(week, '  '), true);
  const entry = { title: 'Mock interview', description: 'Went well', date: '2026-11-22', tags: ['interview'] };
  assert.equal(personalMatches(entry, 'mock well'), true);
  assert.equal(personalMatches(entry, 'nov 22'), true);
  assert.equal(personalMatches(entry, 'run'), false);
});

test('labels: counts, hours and personal dates', () => {
  assert.equal(journalCounts(15, 2), '15 work weeks · 2 personal');
  assert.equal(journalCounts(1, 0), '1 work week · 0 personal');
  assert.equal(hoursText(2.25), '2.3');
  assert.equal(weekHoursLabel(6.5, 11), '6.5 h logged this week', 'recorded goal hours come first');
  assert.equal(weekHoursLabel(0, 11), '11 h in earlier tracks');
  assert.equal(weekHoursLabel(0, 0), 'No hours logged this week');
  assert.equal(personalDate('2026-11-24'), 'Nov 24');
  assert.equal(personalDate(''), 'Undated');
});

test('log records split bullets into their own parts and keep typed track names', async () => {
  const { buildLogDocs, trackLabel } = await import('../src/lib/logs.ts');
  const [doc] = buildLogDocs([{ week: 48, year: 2026, slug: '2026/week-48', dates: '2026-11-23 to 2026-11-29', hours: {}, tags: [], body: '## What I did\n### Monday — Nov 23\n- Solved 3 graph problems\n- LC: two sum\n  wrapped line\nloose text\n' }]);
  assert.deepEqual(doc.parts, [{ label: '', text: 'Solved 3 graph problems' }, { label: 'LC', text: 'two sum wrapped line loose text' }]);
  assert.equal(doc.slug, '2026/week-48');
  assert.equal(trackLabel('LeetCode'), 'LeetCode');
  assert.equal(trackLabel('systemDesign'), 'System Design');
  assert.equal(trackLabel('cloudNative'), 'Cloud Native');
});
