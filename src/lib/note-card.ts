import type { Note } from './workspace';

export const WORDS_PER_MINUTE = 200;

export interface NoteStats { words: number; minutes: number; todos: number; todosDone: number }

export function countWords(body: string): number {
  const text = body.trim();
  return text ? text.split(/\s+/).length : 0;
}

/** List results carry a server summary; a note saved in this session carries its body instead. */
export function noteStats(note: Pick<Note, 'body' | 'todos' | 'summary'>): NoteStats {
  const words = note.summary?.wordCount ?? countWords(note.body ?? '');
  const todos = note.summary?.todoCount ?? note.todos?.length ?? 0;
  const todosDone = note.summary?.todoDone ?? note.todos?.filter(todo => todo.done).length ?? 0;
  return { words, minutes: words ? Math.max(1, Math.round(words / WORDS_PER_MINUTE)) : 0, todos, todosDone };
}

/** "Today", "Yesterday", "3 days ago", then a short calendar date. */
export function noteDate(iso: string | undefined, now = new Date()): string {
  if (!iso) return '';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  const day = (d: Date) => Date.UTC(d.getFullYear(), d.getMonth(), d.getDate());
  const days = Math.round((day(now) - day(date)) / 86_400_000);
  if (days <= 0) return 'Today';
  if (days === 1) return 'Yesterday';
  if (days < 7) return `${days} days ago`;
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', ...(date.getFullYear() === now.getFullYear() ? {} : { year: 'numeric' }) });
}
