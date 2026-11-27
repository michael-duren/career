import { test, expect, type APIRequestContext } from '@playwright/test';
import { MODULES, isEnabled, type ModuleId } from '../../src/lib/modules.ts';

// A turned-off module is gone, not just hidden: every page route it owns (legacy redirects
// included) and its API answer 404, and so do the modules that depend on it. Each probe is
// first checked with the module on, so a 404 can only come from the module guard.

// One GET per module that answers with the module on (entry kinds are probed as well).
const API_PROBES: Record<ModuleId, string[]> = {
  timeline: ['/api/goals'],
  schedule: ['/api/scheduler/week'],
  leetgrinder: ['/api/leetgrinder/today'],
  companies: ['/api/connections/queue'],
  people: ['/api/connections'],
  bookshelf: [],
  journal: ['/api/summaries/progress'],
  notes: [],
  audio: [],
  agents: ['/api/agent-context'],
};

type Settings = { modules: Record<string, boolean> };

const probes = (id: ModuleId) => {
  const def = MODULES.find(m => m.id === id)!;
  return [...def.routes, ...def.kinds.map(kind => `/api/entries/${kind}`), ...API_PROBES[id]];
};

async function status(request: APIRequestContext, path: string) {
  return (await request.get(path, { maxRedirects: 0 })).status();
}

async function patchModules(request: APIRequestContext, baseURL: string, modules: Record<string, boolean>) {
  const response = await request.patch('/api/settings', { headers: { origin: baseURL }, data: { modules } });
  expect(response.status(), await response.text()).toBe(200);
}

let original: Settings;
test.beforeAll(async ({ request }) => {
  original = await (await request.get('/api/settings')).json();
});
test.afterEach(async ({ request, baseURL }) => {
  await patchModules(request, baseURL!, original.modules);
});

for (const module of MODULES) {
  test(`${module.label} off: its routes and dependents 404`, async ({ request, baseURL }) => {
    const allOn = Object.fromEntries(MODULES.map(m => [m.id, true]));
    await patchModules(request, baseURL!, allOn);
    const off = { ...allOn, [module.id]: false };
    // The module plus everything that needs it (Timeline off takes Schedule and Agents with it).
    const gone = MODULES.filter(m => !isEnabled({ modules: off }, m.id)).map(m => m.id);
    expect(gone).toContain(module.id);
    const paths = gone.flatMap(probes);

    for (const path of paths) expect(await status(request, path), `${path} with every module on`).not.toBe(404);

    await patchModules(request, baseURL!, off);
    for (const path of paths) expect(await status(request, path), `${path} with ${module.id} off`).toBe(404);
    // The other modules keep working.
    for (const other of MODULES.filter(m => !gone.includes(m.id))) {
      expect(await status(request, other.routes[0]), `${other.routes[0]} with ${module.id} off`).not.toBe(404);
    }
  });
}
