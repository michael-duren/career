import { test } from 'node:test';
import assert from 'node:assert/strict';
import { DEFAULT_SETTINGS, MODULES, MODULE_DESCRIPTIONS, MODULE_IDS, appearsIn, enabledModules, isEnabled, lockedBy, moduleForPath, moduleStatus, modulesForFocus, moreModules, moveModule, orderedModules, partEnabled, tabBarHint, tabBarModules } from '../src/lib/modules.ts';

const settings = (modules: Record<string, boolean>, moduleOrder: string[] = [...MODULE_IDS]) => ({ modules: { ...DEFAULT_SETTINGS.modules, ...modules }, moduleOrder });

test('registry ids are unique and dependencies exist', () => {
  assert.deepEqual(MODULES.map(m => m.id), [...MODULE_IDS]);
  for (const m of MODULES) for (const dep of m.dependsOn) assert.ok(MODULE_IDS.includes(dep), `${m.id} depends on unknown ${dep}`);
});

test('a module whose dependency is off is effectively off and locked', () => {
  const s = settings({ timeline: false });
  assert.equal(isEnabled(s, 'timeline'), false);
  assert.equal(isEnabled(s, 'schedule'), false);
  assert.equal(isEnabled(s, 'agents'), false);
  assert.equal(isEnabled(s, 'leetgrinder'), true);
  assert.deepEqual(lockedBy(s, 'schedule'), ['Timeline']);
  assert.deepEqual(lockedBy(settings({}), 'schedule'), []);
  // The stored switch is kept, so turning Timeline back on restores Schedule.
  assert.equal(isEnabled(settings({ timeline: true }), 'schedule'), true);
});

test('missing switches count as on', () => {
  assert.equal(isEnabled({ modules: {} }, 'notes'), true);
});

test('turning off Companies keeps People but hides its catch-up queue', () => {
  const s = settings({ companies: false });
  assert.equal(isEnabled(s, 'people'), true);
  assert.equal(partEnabled(s, 'people', 'Catch-up queue'), false);
  assert.equal(partEnabled(settings({}), 'people', 'Catch-up queue'), true);
});

test('tab bar takes the first 3 enabled modules in the user order', () => {
  const order = ['notes', 'leetgrinder', 'journal', 'timeline'];
  assert.deepEqual(tabBarModules(settings({}, order)).map(m => m.id), ['notes', 'leetgrinder', 'journal']);
  assert.deepEqual(tabBarModules(settings({ leetgrinder: false }, order)).map(m => m.id), ['notes', 'journal', 'timeline']);
  // Unknown and repeated ids are dropped; missing ids follow in registry order.
  assert.deepEqual(orderedModules({ moduleOrder: ['audio', 'bogus', 'audio'] }).map(m => m.id), ['audio', ...MODULE_IDS.filter(id => id !== 'audio')]);
  assert.equal(enabledModules(settings({ timeline: false })).some(m => m.id === 'schedule'), false);
});

test('the More page lists every usable module the tab bar has no room for, in order', () => {
  const order = ['notes', 'leetgrinder', 'journal', 'timeline'];
  assert.deepEqual(moreModules(settings({}, order)).map(m => m.id), ['timeline', 'schedule', 'companies', 'people', 'bookshelf', 'audio', 'agents']);
  // Turning a tab's module off pulls the next module into the bar and out of More.
  assert.deepEqual(moreModules(settings({ leetgrinder: false, people: false }, order)).map(m => m.id), ['schedule', 'companies', 'bookshelf', 'audio', 'agents']);
  for (const s of [settings({}), settings({ timeline: false }, order)]) {
    assert.deepEqual([...tabBarModules(s), ...moreModules(s)], enabledModules(s));
  }
});

test('maps page paths to modules on whole segments', () => {
  assert.equal(moduleForPath('/notes')?.id, 'notes');
  assert.equal(moduleForPath('/notes/systems/db')?.id, 'notes');
  assert.equal(moduleForPath('/manage/companies')?.id, 'companies');
  assert.equal(moduleForPath('/notesy'), undefined);
  assert.equal(moduleForPath('/'), undefined);
});

test('setup focus picks pre-select modules', () => {
  const picked = modulesForFocus(['Networking', 'Reflecting']);
  assert.deepEqual(MODULE_IDS.filter(id => picked[id]), ['people', 'companies', 'journal', 'audio'].sort((a, b) => MODULE_IDS.indexOf(a as never) - MODULE_IDS.indexOf(b as never)));
});

test('Settings → Modules status: locked, paused with real counts, blocked parts, and users', () => {
  const all = settings({});
  assert.deepEqual(moduleStatus(all, 'timeline'), { text: 'Schedule and Agent context use this', tone: 'quiet' });
  assert.deepEqual(moduleStatus(settings({ agents: false }), 'timeline'), { text: 'Schedule uses this', tone: 'quiet' });
  assert.equal(moduleStatus(all, 'notes'), null);
  const noTimeline = settings({ timeline: false });
  assert.deepEqual(moduleStatus(noTimeline, 'schedule'), { text: 'needs Timeline', tone: 'accent' });
  assert.deepEqual(moduleStatus(noTimeline, 'timeline', { timeline: 16 }), { text: 'paused · 16 goals kept', tone: 'muted' });
  assert.deepEqual(moduleStatus(settings({ audio: false }), 'audio', { audio: 1 }), { text: 'paused · 1 recording kept', tone: 'muted' });
  // Nothing saved, an unknown count, or a module that saves nothing countable: just "paused".
  assert.equal(moduleStatus(settings({ audio: false }), 'audio', { audio: 0 })?.text, 'paused');
  assert.equal(moduleStatus(settings({ audio: false }), 'audio')?.text, 'paused');
  assert.equal(moduleStatus(settings({ schedule: false }), 'schedule', { schedule: 3 })?.text, 'paused');
  assert.deepEqual(moduleStatus(settings({ companies: false }), 'people'), { text: 'catch-up queue needs Companies', tone: 'accent' });
});

test('every module has a description and appears in the menu', () => {
  for (const m of MODULES) {
    assert.ok(MODULE_DESCRIPTIONS[m.id], `${m.id} has no description`);
    assert.equal(appearsIn(m)[0], 'Menu');
  }
  assert.deepEqual(appearsIn(MODULES.find(m => m.id === 'leetgrinder')!), ['Menu', 'Home', 'Search', 'Reminders']);
  assert.deepEqual(appearsIn(MODULES.find(m => m.id === 'agents')!), ['Menu']);
});

test('tab bar hint names the module that slides into the last tab', () => {
  assert.equal(tabBarHint(settings({})), 'turn off Leetgrinder and Companies slides in');
  assert.equal(tabBarHint(settings({ leetgrinder: false })), 'turn off Companies and People slides in');
  const fewOn = Object.fromEntries(MODULE_IDS.map(id => [id, ['notes', 'journal'].includes(id)]));
  assert.equal(tabBarHint(settings(fewOn)), '');
});

test('moveModule moves one id and clamps the target', () => {
  assert.deepEqual(moveModule(['timeline', 'schedule', 'notes'], 0, 2), ['schedule', 'notes', 'timeline']);
  assert.deepEqual(moveModule(['timeline', 'schedule', 'notes'], 2, 0), ['notes', 'timeline', 'schedule']);
  assert.deepEqual(moveModule(['timeline', 'schedule', 'notes'], 1, 9), ['timeline', 'notes', 'schedule']);
});
