import { z } from 'zod';

/**
 * The module registry. Every feature is a module the user can turn off; menus,
 * Home, search and the phone tab bar are generated from the enabled modules.
 * Home, Search, Settings and Setup are always on and are not modules.
 *
 * The server mirrors the ids, routes, API prefixes, entry kinds and MCP tools
 * in internal/modules/modules.go (a Go test checks the ids match).
 * See docs/redesign/modules.md.
 */
export const MODULE_IDS = ['timeline', 'schedule', 'leetgrinder', 'companies', 'people', 'bookshelf', 'journal', 'notes', 'audio', 'agents'] as const;
export type ModuleId = typeof MODULE_IDS[number];

export const THEMES = ['dark', 'vellum', 'cobalt', 'concrete'] as const;
export type Theme = typeof THEMES[number];

export interface ModuleDef {
  id: ModuleId;
  /** Menu label. */
  label: string;
  /** Short uppercase label for the header and phone tab bar. */
  short: string;
  /** Line-drawing icon name. */
  icon: string;
  /** Main route first; every route the module owns. */
  routes: string[];
  /** What the module adds to Home, if anything. */
  homeSection?: string;
  /** What global search finds for it, if anything. */
  search?: string;
  /** Entry kinds it owns in /api/entries and /api/search. */
  kinds: string[];
  reminders?: 'ntfy';
  mcpTools: string[];
  apiPrefixes: string[];
  /** Modules that must be on for this one to work. */
  dependsOn: ModuleId[];
  /** Parts that need another module; the module stays on without it. */
  partsNeed?: { part: string; module: ModuleId }[];
}

export const MODULES: readonly ModuleDef[] = [
  { id: 'timeline', label: 'Timeline', short: 'TIMELINE', icon: 'timeline', routes: ['/timeline'], homeSection: 'Active goals', search: 'goals', kinds: ['goal'], mcpTools: [], apiPrefixes: ['/api/goals'], dependsOn: [] },
  { id: 'schedule', label: 'Schedule', short: 'SCHEDULE', icon: 'schedule', routes: ['/weekly-scheduler'], homeSection: "Today's sessions, missed", search: 'sessions', kinds: [],
    mcpTools: ['get_weekly_schedule', 'save_schedule_session', 'remove_schedule_session', 'save_schedule_rule', 'end_schedule_rule', 'record_schedule_actual', 'update_schedule_settings'], apiPrefixes: ['/api/scheduler'], dependsOn: ['timeline'] },
  { id: 'leetgrinder', label: 'Leetgrinder', short: 'LEET', icon: 'leetgrinder', routes: ['/leetgrinder'], homeSection: 'Streak + review card', search: 'problems', kinds: [], reminders: 'ntfy',
    mcpTools: ['get_leetgrinder_today', 'list_leetgrinder_reviews', 'get_leetgrinder_stats', 'log_leetgrinder_attempt', 'list_leetgrinder_todos', 'create_leetgrinder_todo_set', 'add_leetgrinder_todo_problem', 'reorder_leetgrinder_set', 'remove_leetgrinder_todo_problem', 'remove_leetgrinder_todo_set'], apiPrefixes: ['/api/leetgrinder'], dependsOn: [] },
  { id: 'companies', label: 'Companies', short: 'COMPANIES', icon: 'companies', routes: ['/companies', '/manage/companies'], search: 'companies', kinds: ['company'], mcpTools: [], apiPrefixes: [], dependsOn: [] },
  { id: 'people', label: 'People', short: 'PEOPLE', icon: 'people', routes: ['/connections'], search: 'people', kinds: ['connection'], mcpTools: ['list_connection_companies', 'add_companies_to_queue'], apiPrefixes: ['/api/connections'], dependsOn: [],
    partsNeed: [{ part: 'Catch-up queue', module: 'companies' }] },
  { id: 'bookshelf', label: 'Bookshelf', short: 'BOOKS', icon: 'bookshelf', routes: ['/books', '/manage/books'], search: 'books', kinds: ['book'], mcpTools: [], apiPrefixes: [], dependsOn: [] },
  { id: 'journal', label: 'Journal', short: 'JOURNAL', icon: 'journal', routes: ['/journal'], search: 'entries', kinds: ['week', 'personal'], mcpTools: [], apiPrefixes: [], dependsOn: [] },
  { id: 'notes', label: 'Notes', short: 'NOTES', icon: 'notes', routes: ['/notes'], search: 'notes', kinds: ['note'], mcpTools: [], apiPrefixes: [], dependsOn: [] },
  { id: 'audio', label: 'Audio thoughts', short: 'AUDIO', icon: 'audio', routes: ['/audio-thoughts'], search: 'transcripts', kinds: ['run'], mcpTools: [], apiPrefixes: ['/api/running'], dependsOn: [] },
  { id: 'agents', label: 'Agent context', short: 'AGENTS', icon: 'agents', routes: ['/agents'], kinds: [], mcpTools: [], apiPrefixes: ['/api/agent-context'], dependsOn: ['timeline'] },
];

export function moduleById(id: ModuleId): ModuleDef {
  return MODULES.find(m => m.id === id)!;
}

/** The module that owns a page path, matching whole path segments. */
export function moduleForPath(path: string): ModuleDef | undefined {
  return MODULES.find(m => m.routes.some(route => path === route || path.startsWith(`${route}/`)));
}

export const settingsSchema = z.object({
  modules: z.record(z.string(), z.boolean()),
  moduleOrder: z.array(z.string()),
  theme: z.enum(THEMES),
  setupCompletedAt: z.string().nullable(),
  effective: z.record(z.string(), z.boolean()).optional(),
});
export type Settings = z.infer<typeof settingsSchema>;

/** Settings with every module on, as for an existing workspace. */
export const DEFAULT_SETTINGS: Settings = {
  modules: Object.fromEntries(MODULE_IDS.map(id => [id, true])),
  moduleOrder: [...MODULE_IDS], theme: 'dark', setupCompletedAt: null,
};

/** Whether a module is usable: switched on (missing counts as on) and every dependency usable. */
export function isEnabled(settings: Pick<Settings, 'modules'>, id: ModuleId, seen: ModuleId[] = []): boolean {
  if (seen.includes(id) || settings.modules[id] === false) return false;
  return moduleById(id).dependsOn.every(dep => isEnabled(settings, dep, [...seen, id]));
}

/**
 * The labels of the dependencies keeping a module's switch locked, for
 * "needs Timeline". Empty when nothing blocks it.
 */
export function lockedBy(settings: Pick<Settings, 'modules'>, id: ModuleId): string[] {
  return moduleById(id).dependsOn.filter(dep => !isEnabled(settings, dep)).map(dep => moduleById(dep).label);
}

/** Every module in the user's order: stored order first, then the rest in registry order. */
export function orderedModules(settings: Pick<Settings, 'moduleOrder'>): ModuleDef[] {
  const ids = [...new Set([...settings.moduleOrder, ...MODULE_IDS])].filter((id): id is ModuleId => (MODULE_IDS as readonly string[]).includes(id));
  return ids.map(moduleById);
}

/** Usable modules in the user's order. */
export function enabledModules(settings: Pick<Settings, 'modules' | 'moduleOrder'>): ModuleDef[] {
  return orderedModules(settings).filter(m => isEnabled(settings, m.id));
}

/** The phone tab bar's module tabs: the first 3 usable modules in order (Home and More frame them). */
export function tabBarModules(settings: Pick<Settings, 'modules' | 'moduleOrder'>): ModuleDef[] {
  return enabledModules(settings).slice(0, 3);
}

/** Whether a part of a module that needs another module (People's catch-up queue) is shown. */
export function partEnabled(settings: Pick<Settings, 'modules'>, id: ModuleId, part: string): boolean {
  const need = moduleById(id).partsNeed?.find(p => p.part === part);
  return isEnabled(settings, id) && (!need || isEnabled(settings, need.module));
}

/** Setup focus picks and the modules each turns on. */
export const FOCUS_SUGGESTIONS: Record<string, ModuleId[]> = {
  'Interview prep': ['leetgrinder', 'timeline', 'schedule', 'companies'],
  Learning: ['bookshelf', 'notes', 'timeline'],
  Networking: ['people', 'companies'],
  'Planning my time': ['timeline', 'schedule', 'agents'],
  Reflecting: ['journal', 'audio'],
};

/** The modules switched on by a set of setup focus picks; the rest are off. */
export function modulesForFocus(focuses: string[]): Record<ModuleId, boolean> {
  const on = new Set(focuses.flatMap(f => FOCUS_SUGGESTIONS[f] ?? []));
  return Object.fromEntries(MODULE_IDS.map(id => [id, on.has(id)])) as Record<ModuleId, boolean>;
}

export type SettingsPatch = Partial<{ modules: Partial<Record<ModuleId, boolean>>; moduleOrder: ModuleId[]; theme: Theme; setupCompletedAt: string | null }>;

async function settingsRequest(init?: RequestInit): Promise<Settings> {
  const response = await fetch('/api/settings', { cache: 'no-store', ...init });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(typeof body.error === 'string' ? body.error : 'Settings could not be loaded. Try again.');
  return settingsSchema.parse(body);
}

export function loadSettings(): Promise<Settings> {
  return settingsRequest();
}

export function saveSettings(patch: SettingsPatch): Promise<Settings> {
  return settingsRequest({ method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(patch) });
}
