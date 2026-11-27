import { formatDate, formatTime, formatWeekdayDate } from './format.ts';

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
/** "Tue 24 Nov · 7:15 AM" in the browser zone, or '' for a bad timestamp. */
export const thoughtDate = (startedAt: string) => {
  const date = new Date(startedAt);
  return Number.isNaN(date.getTime()) ? '' : `${formatWeekdayDate(date)} · ${formatTime(date)}`;
};
/** A recordings row's date and time: "NOV 1" and "10:11 AM", with the year when it isn't this year. */
export function recordingStamp(startedAt: string, now = new Date()): { date: string; time: string } {
  const date = new Date(startedAt);
  if (Number.isNaN(date.getTime())) return { date: '', time: '' };
  const day = formatDate(date).toUpperCase();
  return { date: date.getFullYear() === now.getFullYear() ? day : `${day} ${date.getFullYear()}`, time: formatTime(date) };
}
/** Search over a thought's title, tags and transcript (the list's excerpt, or the body). */
export function matchesThought(entry: { title: string; tags: string[]; body?: string; excerpt?: string }, query: string): boolean {
  const needle = query.trim().toLowerCase();
  return !needle || `${entry.title} ${entry.tags.join(' ')} ${thoughtExcerpt(entry)}`.toLowerCase().includes(needle);
}
/**
 * Polyline points for the recorder's waveform: the last levels (0..1) drawn as a line that
 * swings above and below the middle, newest at the right. No levels gives a flat line.
 */
export function waveformPoints(levels: number[], { width = 360, height = 44, step = 5 } = {}): string {
  const count = Math.floor(width / step) + 1, mid = height / 2, amp = mid - 2;
  const recent = levels.slice(-count), offset = count - recent.length;
  return Array.from({ length: count }, (_, i) => {
    const level = i < offset ? 0 : Math.max(0, Math.min(1, recent[i - offset] || 0));
    const y = mid - (i % 2 ? -1 : 1) * level * amp;
    return `${i * step},${Math.round(y * 10) / 10}`;
  }).join(' ');
}
