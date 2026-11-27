import { test } from 'node:test';
import assert from 'node:assert/strict';
import { MODULE_IDS } from '../src/lib/modules.ts';
import {
  FOCUS_CARDS, SETUP_STEPS, connectionsFor, effectiveModules, finishPatch, goalDraftError, goalFromDraft, localDate, mcpURL, skipPatch,
  suggestedModules, switchesFrom,
} from '../src/lib/setup.ts';

const onIds = (switches: Record<string, boolean>) => MODULE_IDS.filter(id => switches[id]);

test('five steps, and every focus card turns on modules', () => {
  assert.deepEqual(SETUP_STEPS.map(s => s.label), ['Focus', 'Modules', 'Connect', 'Theme', 'First goal']);
  assert.deepEqual(FOCUS_CARDS.map(c => c.name), ['Interview prep', 'Learning', 'Networking', 'Planning my time', 'Reflecting']);
  for (const card of FOCUS_CARDS) assert.ok(onIds(suggestedModules([card.name])).length > 0, card.name);
});

test('focus picks suggest their modules; no picks suggests everything', () => {
  assert.deepEqual(onIds(suggestedModules(['Interview prep', 'Networking'])), ['timeline', 'schedule', 'leetgrinder', 'companies', 'people']);
  assert.deepEqual(onIds(suggestedModules(['Reflecting'])), ['journal', 'audio']);
  assert.deepEqual(onIds(suggestedModules([])), [...MODULE_IDS]);
});

test('saved settings become switches, missing modules counting as on', () => {
  assert.deepEqual(onIds(switchesFrom({ journal: false, audio: false })), MODULE_IDS.filter(id => id !== 'journal' && id !== 'audio'));
});

test('dependencies apply: a switched-on module needing an off one is off', () => {
  const switches = { ...suggestedModules(['Planning my time']), timeline: false };
  assert.deepEqual(onIds(effectiveModules(switches)), []);
  assert.equal(switches.schedule, true, 'the raw switch is kept, so turning Timeline back on restores Schedule');
});

test('Connect lists only the integrations of modules that are on', () => {
  assert.deepEqual(connectionsFor(effectiveModules(suggestedModules(['Reflecting']))), []);
  assert.deepEqual(connectionsFor(effectiveModules(suggestedModules(['Interview prep']))).map(c => c.name), ['Google Calendar', 'Reminders (ntfy)', 'Browser extension']);
  assert.deepEqual(connectionsFor(effectiveModules(suggestedModules([]))).map(c => c.cta), ['CONNECT', 'UPLOAD .ZIP', 'SET UP', 'INSTALL', 'COPY URL']);
  assert.equal(mcpURL('https://career.example'), 'https://career.example/api/mcp');
});

test('Skip turns everything on with the dark theme; Finish saves what is effectively on', () => {
  const now = new Date('2026-11-26T15:00:00Z');
  assert.deepEqual(skipPatch(now), { modules: Object.fromEntries(MODULE_IDS.map(id => [id, true])), theme: 'dark', setupCompletedAt: '2026-11-26T15:00:00.000Z' });
  const finish = finishPatch({ ...suggestedModules(['Planning my time']), timeline: false }, now);
  assert.equal(finish.setupCompletedAt, '2026-11-26T15:00:00.000Z');
  assert.equal(finish.theme, undefined, 'the theme saved itself on step 4');
  // Schedule is sent off, since the server refuses to newly turn on a blocked module.
  assert.equal(finish.modules?.schedule, false);
  assert.equal(Object.keys(finish.modules ?? {}).length, MODULE_IDS.length);
});

test('the first goal is optional, needs both dates, and ends on or after its start', () => {
  assert.equal(goalDraftError({ title: ' ', start: '', end: '' }), '');
  assert.match(goalDraftError({ title: 'Ship it', start: '2026-11-26', end: '' }), /start and an end/);
  assert.match(goalDraftError({ title: 'Ship it', start: '2026-11-26', end: '2026-11-01' }), /before the start/);
  assert.equal(goalDraftError({ title: 'Ship it', start: '2026-11-26', end: '2026-11-26' }), '');
  assert.equal(goalFromDraft({ title: '', start: '2026-11-26', end: '2026-12-31' }, 'id', new Date()), null);
  const goal = goalFromDraft({ title: ' Finish NeetCode 150 ', start: '2026-11-26', end: '2026-12-31' }, '8c1f5bd2-4ad5-4c22-9a43-2f2a1d3f0e11', new Date('2026-11-26T15:00:00Z'));
  assert.equal(goal?.title, 'Finish NeetCode 150');
  assert.equal(goal?.startDate, '2026-11-26');
  assert.equal(goal?.endDate, '2026-12-31');
  assert.equal(goal?.status, 'planned');
  assert.equal(goal?.createdAt, '2026-11-26T15:00:00.000Z');
});

test('localDate is the local calendar day', () => {
  assert.equal(localDate(new Date(2026, 0, 5, 23, 30)), '2026-01-05');
});
