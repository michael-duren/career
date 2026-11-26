import { z } from 'zod';
import { THEMES, type Theme } from './theme.ts';

/**
 * The module registry. Every feature is a module the user can turn off. The new
 * shell, Home, search, Settings and Setup build their menus, tab bar and sections
 * from it (the helpers below are for them); today the header and a few pages use it.
 * Home, Search, Settings and Setup are always on and are not modules.
 *
 * The server mirrors the ids, labels, short labels, routes, API prefixes, entry
 * kinds, MCP tools and dependencies in internal/modules/modules.go; TestRegistryMatchesFrontend
 * checks the two agree.
 * See docs/redesign/modules.md.
 */
export const MODULE_IDS = ['timeline', 'schedule', 'leetgrinder', 'companies', 'people', 'bookshelf', 'journal', 'notes', 'audio', 'agents'] as const;
export type ModuleId = typeof MODULE_IDS[number];

export interface ModuleDef {
  id: ModuleId;
  /** Menu label. */
  label: string;
  /** Short uppercase label for the header and phone tab bar. */
  short: string;
  /** Line-drawing icon name. */
  icon: string;
  /** Main route first; every route the module owns, legacy redirects included. */
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
  /** MCP tools that also need another module (the server hides them without it). */
  toolsNeed?: Partial<Record<string, ModuleId>>;
}

export const MODULES: readonly ModuleDef[] = [
  { id: 'timeline', label: 'Timeline', short: 'TIMELINE', icon: 'timeline', routes: ['/timeline'], homeSection: 'Active goals', search: 'goals', kinds: ['goal'], mcpTools: [], apiPrefixes: ['/api/goals'], dependsOn: [] },
  { id: 'schedule', label: 'Schedule', short: 'SCHEDULE', icon: 'schedule', routes: ['/weekly-scheduler'], homeSection: "Today's sessions, missed", search: 'sessions', kinds: [],
    mcpTools: ['get_weekly_schedule', 'save_schedule_session', 'remove_schedule_session', 'save_schedule_rule', 'end_schedule_rule', 'record_schedule_actual', 'update_schedule_settings'], apiPrefixes: ['/api/scheduler'], dependsOn: ['timeline'] },
  { id: 'leetgrinder', label: 'Leetgrinder', short: 'LEET', icon: 'leetgrinder', routes: ['/leetgrinder', '/bootcamp'], homeSection: 'Streak + review card', search: 'problems', kinds: [], reminders: 'ntfy',
    mcpTools: ['get_leetgrinder_today', 'list_leetgrinder_reviews', 'get_leetgrinder_stats', 'log_leetgrinder_attempt', 'list_leetgrinder_todos', 'create_leetgrinder_todo_set', 'add_leetgrinder_todo_problem', 'reorder_leetgrinder_set', 'remove_leetgrinder_todo_problem', 'remove_leetgrinder_todo_set'], apiPrefixes: ['/api/leetgrinder'], dependsOn: [] },
  { id: 'companies', label: 'Companies', short: 'COMPANIES', icon: 'companies', routes: ['/companies', '/manage/companies'], search: 'companies', kinds: ['company'], mcpTools: [], apiPrefixes: [], dependsOn: [] },
  { id: 'people', label: 'People', short: 'PEOPLE', icon: 'people', routes: ['/connections'], search: 'people', kinds: ['connection'], mcpTools: ['list_connection_companies', 'add_companies_to_queue'], apiPrefixes: ['/api/connections'], dependsOn: [],
    partsNeed: [{ part: 'Catch-up queue', module: 'companies' }], toolsNeed: { list_connection_companies: 'companies', add_companies_to_queue: 'companies' } },
  { id: 'bookshelf', label: 'Bookshelf', short: 'BOOKS', icon: 'bookshelf', routes: ['/books', '/manage/books'], search: 'books', kinds: ['book'], mcpTools: [], apiPrefixes: [], dependsOn: [] },
  { id: 'journal', label: 'Journal', short: 'JOURNAL', icon: 'journal', routes: ['/journal', '/logs', '/personal-journal'], search: 'entries', kinds: ['week', 'personal'], mcpTools: [], apiPrefixes: ['/api/summaries/progress'], dependsOn: [] },
  { id: 'notes', label: 'Notes', short: 'NOTES', icon: 'notes', routes: ['/notes'], search: 'notes', kinds: ['note'], mcpTools: [], apiPrefixes: [], dependsOn: [] },
  { id: 'audio', label: 'Audio thoughts', short: 'AUDIO', icon: 'audio', routes: ['/audio-thoughts', '/running'], search: 'transcripts', kinds: ['run'], mcpTools: [], apiPrefixes: ['/api/running'], dependsOn: [] },
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

/** Settings before any are saved: every module on, dark theme, setup not done. */
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

/** The phone More page's modules: usable ones the tab bar has no room for, in order. */
export function moreModules(settings: Pick<Settings, 'modules' | 'moduleOrder'>): ModuleDef[] {
  const tabs = new Set(tabBarModules(settings).map(m => m.id));
  return enabledModules(settings).filter(m => !tabs.has(m.id));
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

async function settingsRequest(fallback: string, init?: RequestInit): Promise<Settings> {
  const response = await fetch('/api/settings', { cache: 'no-store', ...init }).catch(() => { throw new Error(fallback); });
  const body: unknown = await response.json().catch(() => null);
  if (!response.ok) {
    const error = body && typeof body === 'object' && 'error' in body && typeof body.error === 'string' ? body.error : fallback;
    throw new Error(error);
  }
  const parsed = settingsSchema.safeParse(body);
  if (!parsed.success) throw new Error(fallback);
  return parsed.data;
}

export function loadSettings(): Promise<Settings> {
  return settingsRequest('Settings could not be loaded. Try again.');
}

/**
 * Whether module id is usable, for pages that hide a part owned by another module.
 * If settings cannot load it logs and answers true, so the part's own request reports the failure.
 */
export function moduleEnabled(id: ModuleId): Promise<boolean> {
  return loadSettings().then(settings => isEnabled(settings, id), error => {
    console.warn(`Module settings could not be loaded; assuming ${id} is on.`, error);
    return true;
  });
}

/**
 * Saved settings are announced so the header (public/ui/shell.js, which mirrors these names)
 * updates its menu without a reload: a SETTINGS_EVENT on this window, and a message on
 * SETTINGS_CHANNEL for other tabs.
 */
export const SETTINGS_EVENT = 'career:settings';
export const SETTINGS_CHANNEL = 'career-settings';

export function announceSettings(settings: Settings): void {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(SETTINGS_EVENT, { detail: settings }));
  if (typeof BroadcastChannel === 'function') {
    const channel = new BroadcastChannel(SETTINGS_CHANNEL);
    channel.postMessage(settings);
    channel.close();
  }
}

export async function saveSettings(patch: SettingsPatch): Promise<Settings> {
  const saved = await settingsRequest('Settings could not be saved. Try again.', { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(patch) });
  // The save succeeded; a failed announcement only leaves other menus stale until reload.
  try {
    announceSettings(saved);
  } catch (error) {
    console.warn('Saved settings could not be announced to open menus.', error);
  }
  return saved;
}
