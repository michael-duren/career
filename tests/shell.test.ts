import { test } from 'node:test';
import assert from 'node:assert/strict';
import { SETTINGS_CHANNEL as SHELL_CHANNEL, SETTINGS_EVENT as SHELL_EVENT, currentLink, fitCount, formatClock, menuOrder } from '../public/ui/shell.js';
import { DEFAULT_SETTINGS, MODULES, MODULE_IDS, SETTINGS_CHANNEL, SETTINGS_EVENT, enabledModules, moduleForPath, saveSettings, type Settings } from '../src/lib/modules.ts';
import { formatTime } from '../src/lib/format.ts';

const ids = [...MODULE_IDS];

test('the header and modules.ts agree on the settings event and channel names', () => {
  assert.equal(SHELL_EVENT, SETTINGS_EVENT);
  assert.equal(SHELL_CHANNEL, SETTINGS_CHANNEL);
});

test('menuOrder matches enabledModules: user order, dependencies applied by effective', () => {
  const cases: Settings[] = [
    DEFAULT_SETTINGS,
    { ...DEFAULT_SETTINGS, moduleOrder: ['notes', 'leetgrinder', 'timeline'] },
    { ...DEFAULT_SETTINGS, modules: { ...DEFAULT_SETTINGS.modules, timeline: false, audio: false } },
    { ...DEFAULT_SETTINGS, moduleOrder: ['agents', 'unknown', 'agents', 'books'], modules: { notes: false } },
  ];
  for (const settings of cases) {
    const effective = Object.fromEntries(ids.map(id => [id, enabledModules(settings).some(m => m.id === id)]));
    assert.deepEqual(menuOrder({ ...settings, effective }, ids), enabledModules(settings).map(m => m.id));
  }
  // Without `effective` the stored switches decide.
  assert.deepEqual(menuOrder({ modules: { notes: false }, moduleOrder: [] }, ['notes', 'audio']), ['audio']);
});

test('fitCount counts the leading links that fit, gaps included', () => {
  assert.equal(fitCount([50, 50, 50], 170, 10), 3);
  assert.equal(fitCount([50, 50, 50], 169, 10), 2);
  assert.equal(fitCount([50, 50, 50], 49, 10), 0);
  assert.equal(fitCount([], 0, 10), 0);
});

test('formatClock is 12-hour like formatTime', () => {
  for (const [h, m, text] of [[0, 5, '12:05 AM'], [12, 0, '12:00 PM'], [17, 20, '5:20 PM'], [9, 7, '9:07 AM']] as const) {
    const date = new Date(2026, 10, 24, h, m);
    assert.equal(formatClock(date), text);
    assert.equal(formatClock(date), formatTime(date));
  }
});

test('currentLink matches whole path segments like moduleForPath', () => {
  const items = [{ dataset: { routes: '/' } }, ...MODULES.map(m => ({ dataset: { routes: m.routes.join(' ') }, id: m.id }))];
  for (const path of ['/companies', '/manage/companies/', '/logs/2026', '/leetgrinder/problems', '/booksmith', '/2026/career-study-plan']) {
    const want = moduleForPath(path.replace(/\/$/, ''))?.id;
    assert.equal((currentLink(items, path) as { id?: string } | undefined)?.id, want, path);
  }
  assert.equal(currentLink(items, '/'), items[0]);
  assert.equal(currentLink([{ dataset: {} }], '/notes'), undefined);
});

test('saveSettings announces the saved settings on the window and to other tabs', async t => {
  const saved = { ...DEFAULT_SETTINGS, modules: { ...DEFAULT_SETTINGS.modules, notes: false } };
  const g = globalThis as unknown as { window?: EventTarget; fetch: typeof fetch };
  const realFetch = g.fetch;
  g.window = new EventTarget();
  g.fetch = async () => new Response(JSON.stringify(saved), { status: 200 });
  t.after(() => { delete g.window; g.fetch = realFetch; });
  const other = new BroadcastChannel(SETTINGS_CHANNEL);
  t.after(() => other.close());
  const broadcast = new Promise(resolve => other.addEventListener('message', event => resolve((event as MessageEvent).data), { once: true }));
  let detail: unknown;
  g.window.addEventListener(SETTINGS_EVENT, event => { detail = (event as CustomEvent).detail; });
  assert.deepEqual(await saveSettings({ modules: { notes: false } }), saved);
  assert.deepEqual(detail, saved);
  assert.deepEqual(await broadcast, saved);
});
