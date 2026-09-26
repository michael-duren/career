// Mirrors ThoughtTitle in internal/database/running.go so thoughts typed in the
// editor are named the same way as transcribed ones.
export const THOUGHT_PLACEHOLDER = 'New audio thought';
const autoTitle = /^(New audio thought|Run \d{4}-\d{2}-\d{2} \d{2}:\d{2}|Audio thought \d{4}-\d{2}-\d{2})$/;
export const isAutoTitle = (title: string) => autoTitle.test(title);
export function thoughtTitle(body: string): string {
  const words = body.trim().split(/\s+/).filter(Boolean);
  if (!words.length) return '';
  let title = words.slice(0, 6).join(' '), cut = words.length > 6;
  const chars = [...title];
  if (chars.length > 60) { title = chars.slice(0, 60).join(''); cut = true; }
  return cut ? `${title.replace(/[ .,;:!?-]+$/, '')}...` : title;
}
export const thoughtExcerpt = (entry: { body?: string; excerpt?: string }) => (entry.excerpt ?? entry.body ?? '').replace(/\s+/g, ' ').trim();
export const thoughtDate = (startedAt: string) => {
  const date = new Date(startedAt);
  return Number.isNaN(date.getTime()) ? '' : `${date.toLocaleDateString(undefined, { weekday: 'short', month: 'short', day: 'numeric' })} · ${date.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })}`;
};
