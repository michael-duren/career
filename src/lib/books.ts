import { checklist } from './checklist.ts';
// ---------------------------------------------------------------------------
// Books & courses model
//
// Each saved book or course contains metadata and a Markdown body. The metadata
// carries the metric-bearing fields; the body carries two schematic blocks:
//
//   ## Chapters        — a checkbox list. Each "- [x] ..." is a finished unit,
//                        each "- [ ] ..." an unfinished one. This is the primary
//                        progress signal — no need to also touch frontmatter.
//   ## Log             — dated "### YYYY-MM-DD" sub-sections of reading notes.
//
// buildShelf() parses the raw entries (progress, chapters, log) into `all`; it
// also derives a featured item, groups and stats that the Bookshelf page no
// longer shows. The page itself uses the helpers at the bottom of this file
// (inProgress, sortCatalog, ...). Adding a new book is handled through the
// workspace editor.
// ---------------------------------------------------------------------------

export const BOOK_CATEGORIES = [
  'Computer Science',
  'Networking',
  'OS',
  'Systems',
  'Distributed Systems',
  'Languages',
  'Career',
  'Online Course',
] as const;

export type BookCategory = (typeof BOOK_CATEGORIES)[number];
export type BookStatus = 'backlog' | 'reading' | 'paused' | 'completed' | 'reference';
export type BookType = 'book' | 'course' | 'paper';
export type ProgressUnit = 'chapter' | 'page' | 'module' | 'section' | 'lecture';

// Raw shape coming out of the content collection (frontmatter + body).
export interface BookFrontmatter {
  title: string;
  edition?: string;
  authors: string[];
  category: BookCategory;
  type: BookType;
  url?: string;
  cover?: string;
  isbn?: string;
  status: BookStatus;
  featured: boolean;
  progress?: { unit: ProgressUnit; total: number; completed: number };
  started?: string;
  finished?: string;
  rating?: number;
  priority: 'high' | 'medium' | 'low';
  tags: string[];
}

export interface RawBook extends BookFrontmatter {
  slug: string;
  body: string;
  /** Set by the API on saved entries (ISO timestamp); absent on imported seeds. */
  updatedAt?: string;
}

export interface ChapterItem {
  index: number;
  done: boolean;
  label: string;
}

export interface LogNote {
  date: string;
  text: string;
}

export interface Book extends RawBook {
  chapters: ChapterItem[];
  log: LogNote[];
  /** Resolved progress unit (frontmatter override → else "chapter"). */
  unit: ProgressUnit;
  /** Units finished and total, from frontmatter override or the checklist. */
  completed: number;
  total: number;
  percent: number;
  /** Most recent dated log note, if any. */
  lastNote: LogNote | null;
}

export interface CategoryGroup {
  category: string;
  books: Book[];
}

export interface ShelfStats {
  totalBooks: number;
  totalCourses: number;
  reading: number;
  completed: number;
  backlog: number;
  /** Sum of finished units across everything (chapters + modules + …). */
  unitsDone: number;
  unitsTotal: number;
  /** Average completion across non-backlog items, 0–100. */
  avgCompletion: number;
  byCategory: { category: string; count: number; done: number }[];
}

export interface ShelfData {
  featured: Book | null;
  currentlyReading: Book[];
  groups: CategoryGroup[];
  stats: ShelfStats;
  all: Book[];
}

// "- [x] 3. Processes" / "* [ ] Intro" → { done, label }
function parseChapters(body: string): ChapterItem[] {
  return checklist(body).filter(task => /^(chapters|modules|sections)$/.test(task.section)).map(task => ({ index: task.index, done: task.checked, label: task.label }));
}

// Dated reading notes under "## Log": "### 2026-06-15" headings followed by lines.
function parseLog(body: string): LogNote[] {
  const lines = body.split('\n');
  const notes: LogNote[] = [];
  let inLog = false;
  let date: string | null = null;
  let buf: string[] = [];

  const flush = () => {
    if (date) {
      const text = buf
        .map((l) => l.replace(/^\s*[-*]\s+/, '').trim())
        .filter(Boolean)
        .join(' ');
      if (text) notes.push({ date, text });
    }
    buf = [];
  };

  for (const raw of lines) {
    const line = raw.trimEnd();
    if (/^##\s+/.test(line) && !/^###/.test(line)) {
      flush();
      date = null;
      inLog = /log|notes|journal/i.test(line);
      continue;
    }
    if (inLog && /^###\s+/.test(line)) {
      flush();
      date = line.replace(/^###\s+/, '').trim();
      continue;
    }
    if (inLog && date) buf.push(line);
  }
  flush();
  // Newest first.
  return notes.sort((a, b) => (a.date < b.date ? 1 : a.date > b.date ? -1 : 0));
}

function toBook(raw: RawBook): Book {
  const chapters = parseChapters(raw.body);
  const log = parseLog(raw.body);

  const checklistTotal = chapters.length;
  const checklistDone = chapters.filter((c) => c.done).length;

  // Frontmatter progress wins when present (page-counted books); otherwise the
  // checklist is the source of truth.
  const total = raw.progress?.total ?? checklistTotal;
  let completed = raw.progress ? raw.progress.completed : checklistDone;
  // A completed book is 100% even if the checklist wasn't fully ticked.
  if (raw.status === 'completed' && total > 0) completed = total;
  const percent = total > 0 ? Math.round((completed / total) * 100) : 0;
  const unit = raw.progress?.unit ?? (raw.type === 'course' ? 'module' : raw.type === 'paper' ? 'section' : 'chapter');

  return {
    ...raw,
    chapters,
    log,
    unit,
    completed,
    total,
    percent,
    lastNote: log[0] ?? null,
  };
}

const STATUS_RANK: Record<BookStatus, number> = {
  reading: 0,
  paused: 1,
  backlog: 2,
  reference: 3,
  completed: 4,
};

const PRIORITY_RANK: Record<'high' | 'medium' | 'low', number> = {
  high: 0,
  medium: 1,
  low: 2,
};

export function buildShelf(raws: RawBook[]): ShelfData {
  const all = raws.map(toBook).sort((a, b) => {
    if (STATUS_RANK[a.status] !== STATUS_RANK[b.status])
      return STATUS_RANK[a.status] - STATUS_RANK[b.status];
    if (PRIORITY_RANK[a.priority] !== PRIORITY_RANK[b.priority])
      return PRIORITY_RANK[a.priority] - PRIORITY_RANK[b.priority];
    return a.title.localeCompare(b.title);
  });

  // The hero: explicit featured flag wins; else the first currently-reading book.
  const reading = all.filter((b) => b.status === 'reading');
  const featured = all.find((b) => b.featured) ?? reading[0] ?? null;
  const currentlyReading = reading.filter((b) => b.slug !== featured?.slug);

  // Group by category, in the canonical category order.
  const groups: CategoryGroup[] = BOOK_CATEGORIES.map((category) => ({
    category,
    books: all.filter((b) => b.category === category),
  })).filter((g) => g.books.length > 0);

  // Roll-up stats.
  const unitsDone = all.reduce((s, b) => s + b.completed, 0);
  const unitsTotal = all.reduce((s, b) => s + b.total, 0);
  const tracked = all.filter((b) => b.status !== 'backlog' && b.total > 0);
  const avgCompletion =
    tracked.length > 0
      ? Math.round(tracked.reduce((s, b) => s + b.percent, 0) / tracked.length)
      : 0;

  const byCategory = groups.map((g) => ({
    category: g.category,
    count: g.books.length,
    done: g.books.filter((b) => b.status === 'completed').length,
  }));

  const stats: ShelfStats = {
    totalBooks: all.filter((b) => b.type === 'book').length,
    totalCourses: all.filter((b) => b.type === 'course').length,
    reading: reading.length,
    completed: all.filter((b) => b.status === 'completed').length,
    backlog: all.filter((b) => b.status === 'backlog').length,
    unitsDone,
    unitsTotal,
    avgCompletion,
    byCategory,
  };

  return { featured, currentlyReading, groups, stats, all };
}

// ---------------------------------------------------------------------------
// Bookshelf page helpers (docs/redesign/pages/bookshelf.md)
// ---------------------------------------------------------------------------

/** When the item last changed: the saved timestamp, else the newest log date. "" when unknown. */
export function lastUpdated(book: Book): string {
  return book.updatedAt || book.lastNote?.date || '';
}

function updatedKey(book: Book): number {
  const value = lastUpdated(book);
  const time = value ? Date.parse(/^\d{4}-\d{2}-\d{2}$/.test(value) ? `${value}T00:00:00Z` : value) : NaN;
  return Number.isNaN(time) ? -Infinity : time;
}

/** In progress = started (completed units > 0) and not finished, most recently updated first. */
export function inProgress(books: Book[]): Book[] {
  return books
    .filter((b) => b.status !== 'completed' && b.completed > 0 && b.completed < b.total)
    .sort((a, b) => updatedKey(b) - updatedKey(a) || a.title.localeCompare(b.title));
}

/** Catalog order: priority, then status (reading first, completed last), then title. */
export function sortCatalog(books: Book[]): Book[] {
  return [...books].sort((a, b) =>
    PRIORITY_RANK[a.priority] - PRIORITY_RANK[b.priority]
    || STATUS_RANK[a.status] - STATUS_RANK[b.status]
    || a.title.localeCompare(b.title));
}

/** Case-insensitive match on the title or any author. An empty query matches everything. */
export function matchesBookSearch(book: Pick<Book, 'title' | 'authors'>, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  return book.title.toLowerCase().includes(q) || book.authors.some((a) => a.toLowerCase().includes(q));
}

/** Categories that hold at least one item, alphabetically, with their counts. */
export function shelves(books: Pick<Book, 'category'>[]): { category: string; count: number }[] {
  const counts = new Map<string, number>();
  for (const b of books) counts.set(b.category, (counts.get(b.category) ?? 0) + 1);
  return [...counts].map(([category, count]) => ({ category, count })).sort((a, b) => a.category.localeCompare(b.category));
}

/**
 * Progress drawn as `count` cells or ticks with `filled` of them done. Up to `max` units
 * get one mark each; longer items (page counts) scale onto `max` marks.
 */
export function progressMarks(completed: number, total: number, max: number): { count: number; filled: number } {
  if (total <= 0) return { count: 0, filled: 0 };
  const done = Math.min(Math.max(completed, 0), total);
  if (total <= max) return { count: total, filled: done };
  // Never show a started item as empty or an unfinished one as full.
  const scaled = Math.round((done / total) * max);
  const filled = done > 0 && scaled === 0 ? 1 : done < total && scaled === max ? max - 1 : scaled;
  return { count: max, filled };
}

/** "chapter" / "chapters" (and pages, modules, ...). */
export function unitLabel(unit: ProgressUnit, count: number): string {
  return count === 1 ? unit : `${unit}s`;
}
