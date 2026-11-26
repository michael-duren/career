import { test } from 'node:test';
import assert from 'node:assert/strict';
import { DEFAULT_SETTINGS, MODULES, MODULE_IDS, enabledModules, isEnabled, lockedBy, moduleForPath, modulesForFocus, moreModules, orderedModules, partEnabled, tabBarModules } from '../src/lib/modules.ts';

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
