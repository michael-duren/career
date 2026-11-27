import { FOCUS_SUGGESTIONS, MODULE_IDS, isEnabled, type ModuleId, type SettingsPatch } from './modules.ts';
import { DEFAULT_GOAL_COLOR, type Goal } from './timeline.ts';

// First-run setup (docs/redesign/pages/setup.md): focus → modules → connect → theme → first goal.
// Pure helpers behind src/components/SetupWizard.tsx, kept free of React so node --test covers them.

/** The five steps; short is the phone label. */
export const SETUP_STEPS = [
  { label: 'Focus', short: 'Focus' },
  { label: 'Modules', short: 'Modules' },
  { label: 'Connect', short: 'Connect' },
  { label: 'Theme', short: 'Theme' },
  { label: 'First goal', short: 'Goal' },
] as const;

export type Focus = keyof typeof FOCUS_SUGGESTIONS;

/** The focus cards of step 1, in order. The modules each turns on come from FOCUS_SUGGESTIONS. */
export const FOCUS_CARDS: { name: Focus; description: string }[] = [
  { name: 'Interview prep', description: 'LeetCode practice, target companies, a study plan' },
  { name: 'Learning', description: 'Books, courses, papers and notes' },
  { name: 'Networking', description: 'Your connections and a catch-up queue' },
  { name: 'Planning my time', description: 'Goals with dates and a weekly schedule' },
  { name: 'Reflecting', description: 'A weekly journal and voice memos' },
];

export type ModuleSwitches = Record<ModuleId, boolean>;

/**
 * The modules the focus picks suggest. No picks narrows nothing, so every module is
 * suggested on; otherwise only the picked focuses' modules are.
 */
export function suggestedModules(focuses: readonly string[]): ModuleSwitches {
  const on = new Set(focuses.flatMap(f => FOCUS_SUGGESTIONS[f] ?? []));
  return Object.fromEntries(MODULE_IDS.map(id => [id, focuses.length === 0 || on.has(id)])) as ModuleSwitches;
}

/** Every module's switch from saved settings (a missing module counts as on). */
export function switchesFrom(modules: Partial<Record<string, boolean>>): ModuleSwitches {
  return Object.fromEntries(MODULE_IDS.map(id => [id, modules[id] !== false])) as ModuleSwitches;
}

/**
 * What is really on once dependencies apply: a switched-on module whose dependency is off is
 * off. This is what Finish saves, since the server refuses to newly turn on a blocked module.
 */
export function effectiveModules(switches: ModuleSwitches): ModuleSwitches {
  return Object.fromEntries(MODULE_IDS.map(id => [id, isEnabled({ modules: switches }, id)])) as ModuleSwitches;
}

export interface SetupConnection {
  module: ModuleId;
  name: string;
  description: string;
  cta: string;
  /** The page that sets it up (opened in a new tab so setup keeps its place). */
  href?: string;
  /** Copy the MCP endpoint instead of opening a page. */
  copy?: 'mcp';
}

/** Step 3's integrations, each shown only when the module it serves is on. */
export const SETUP_CONNECTIONS: SetupConnection[] = [
  { module: 'schedule', name: 'Google Calendar', description: 'Read busy time and export accepted plans to a calendar', cta: 'CONNECT', href: '/weekly-scheduler' },
  { module: 'people', name: 'LinkedIn export', description: 'Import your connections and messages from a LinkedIn data export', cta: 'UPLOAD .ZIP', href: '/connections?import=1' },
  { module: 'leetgrinder', name: 'Reminders (ntfy)', description: 'Daily nudges for LeetCode reviews', cta: 'SET UP', href: '/leetgrinder/settings#ntfy' },
  { module: 'leetgrinder', name: 'Browser extension', description: 'Log LeetCode and NeetCode submissions automatically', cta: 'INSTALL', href: '/leetgrinder/settings#api-tokens' },
  { module: 'agents', name: 'Claude (MCP)', description: 'Let Claude read your goals and schedule', cta: 'COPY URL', copy: 'mcp' },
];

export function connectionsFor(effective: ModuleSwitches): SetupConnection[] {
  return SETUP_CONNECTIONS.filter(c => effective[c.module]);
}

/** The MCP endpoint an AI client connects to. */
export const mcpURL = (origin: string) => `${origin}/api/mcp`;

/** Skip setup: every module on, the dark theme, setup done. */
export function skipPatch(now: Date): SettingsPatch {
  return { modules: Object.fromEntries(MODULE_IDS.map(id => [id, true])), theme: 'dark', setupCompletedAt: now.toISOString() };
}

/** Finish: save what is really on and mark setup done (the theme saved itself on step 4). */
export function finishPatch(switches: ModuleSwitches, now: Date): SettingsPatch {
  return { modules: effectiveModules(switches), setupCompletedAt: now.toISOString() };
}

export interface GoalDraft {
  title: string;
  start: string;
  end: string;
}

/** Why the first-goal fields can't be saved, or '' when they can. A blank name adds no goal. */
export function goalDraftError({ title, start, end }: GoalDraft): string {
  if (!title.trim()) return '';
  if (!start || !end) return 'Give the goal a start and an end date, or clear its name to skip it.';
  if (end < start) return 'The end date is before the start date.';
  return '';
}

/** The goal POST /api/goals saves for the draft, or null when the name is blank. */
export function goalFromDraft(draft: GoalDraft, id: string, now: Date): Goal | null {
  const title = draft.title.trim();
  if (!title) return null;
  const at = now.toISOString();
  return {
    id, title, startDate: draft.start, endDate: draft.end, status: 'planned', dependsOn: [],
    // Goals are drawn by pattern in the Architect UI; the stored color is only a legacy field.
    color: DEFAULT_GOAL_COLOR, dailyHours: 1, createdAt: at, updatedAt: at, notes: [], metadata: {}, steps: [],
  };
}

/** Today in the browser's time zone, as an <input type="date"> value. */
export function localDate(now: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}
