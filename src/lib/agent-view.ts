import { timelineCounts, type Goal } from './timeline.ts';

// Helpers for the Agent context page (docs/redesign/pages/agent-context.md).

export type AgentFormat = 'markdown' | 'json';

export const AGENT_FORMATS = [
  { value: 'markdown', label: 'Markdown' },
  { value: 'json', label: 'JSON' },
] as const satisfies readonly { value: AgentFormat; label: string }[];

/** The API URL that returns the full output in this format. */
export function agentEndpoint(format: AgentFormat): string {
  return format === 'markdown' ? '/api/agent-context?format=markdown' : '/api/agent-context';
}

export function agentFileName(format: AgentFormat): string {
  return format === 'markdown' ? 'agent-context.md' : 'agent-context.json';
}

/** Sidebar counts: goals, active goals (late ones included, as on the Timeline), and steps. */
export function agentStats(goals: Goal[], today: string) {
  return {
    goals: goals.length,
    active: timelineCounts(goals, today).active,
    steps: goals.reduce((sum, goal) => sum + goal.steps.length, 0),
  };
}

/** Two-digit counts, as in the reference ("06"). */
export function padCount(value: number): string {
  return String(value).padStart(2, '0');
}

/**
 * The visible preview: the first maxLines lines, cut to maxChars. The page copies the full
 * text; the preview only says how much it left out.
 */
export function previewText(text: string, maxLines = 60, maxChars = 4000): { text: string; truncated: boolean; hiddenChars: number } {
  let shown = text.split('\n').slice(0, maxLines).join('\n');
  if (shown.length > maxChars) {
    // End on a whole line when there is one inside the limit.
    const cut = shown.lastIndexOf('\n', maxChars);
    shown = shown.slice(0, cut > 0 ? cut : maxChars);
  }
  return { text: shown, truncated: shown.length < text.length, hiddenChars: text.length - shown.length };
}
