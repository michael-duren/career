import type { Note } from './workspace';
import { formatDate } from './format.ts';

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

export interface TopicCount { name: string; count: number }

/** Topics in use with how many notes each holds, alphabetical. */
export function topicCounts(notes: Pick<Note, 'topic'>[]): TopicCount[] {
  const counts = new Map<string, number>();
  for (const note of notes) counts.set(note.topic, (counts.get(note.topic) ?? 0) + 1);
  return [...counts].map(([name, count]) => ({ name, count })).sort((a, b) => a.name.localeCompare(b.name));
}

/** True when the note's title, topic, summary or tags contain the query (case-insensitive). */
export function matchesNoteSearch(note: Pick<Note, 'title' | 'topic' | 'description' | 'tags'>, query: string): boolean {
  const needle = query.trim().toLowerCase();
  if (!needle) return true;
  return [note.title, note.topic, note.description ?? '', ...note.tags].join(' ').toLowerCase().includes(needle);
}

/** Notes in one topic (or all when topic is null) matching the search, most recently updated first. */
export function filterNotes<T extends Pick<Note, 'title' | 'topic' | 'description' | 'tags' | 'updatedAt'>>(notes: T[], topic: string | null, query: string): T[] {
  return notes.filter(note => (topic === null || note.topic === topic) && matchesNoteSearch(note, query))
    .sort((a, b) => (b.updatedAt ?? '').localeCompare(a.updatedAt ?? '') || a.title.localeCompare(b.title));
}

/** "Nov 25" for this year, "Nov 25 2025" for other years; '' when missing or invalid. */
export function noteCardDate(iso: string | undefined, now = new Date()): string {
  if (!iso) return '';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '';
  const label = formatDate(date);
  return date.getFullYear() === now.getFullYear() ? label : `${label} ${date.getFullYear()}`;
}

/** The body without a leading "# Title" heading that repeats the note's title (the reader already shows it). */
export function withoutTitleHeading(body: string, title: string): string {
  // An optional closing run of #s counts only after whitespace (CommonMark), so "# C#" keeps its #.
  const match = /^\s*#[ \t]+(.+?)(?:[ \t]+#+)?[ \t]*(?:\r?\n|$)/.exec(body);
  return match && match[1].trim() === title.trim() ? body.slice(match[0].length).replace(/^\s*\n/, '') : body;
}
