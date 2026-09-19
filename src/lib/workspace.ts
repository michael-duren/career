import type { RawBook } from './books';
import type { RawCompany } from './companies';
import { PLAN_START, weekNumberFor, type WeekEntry } from './progress.ts';

export interface Note {
  id: string;
  title: string;
  topic: string;
  description: string;
  tags: string[];
  body: string;
  updatedAt?: string;
}
export interface JournalWeek extends WeekEntry { updatedAt?: string }
export interface Document {
  id: string;
  title: string;
  description: string;
  tags: string[];
  body: string;
  updatedAt?: string;
}
export type WorkspaceBook = RawBook & { updatedAt?: string };
export type WorkspaceCompany = RawCompany & { updatedAt?: string };
export interface PersonalJournalEntry extends Document { date: string }
export interface RunningNote { id: string; title: string; runDate: string; startedAt: string; distanceKm?: number; durationMin?: number; tags: string[]; body: string; updatedAt?: string }
export type EntryKind = 'run' | 'personal' | 'note' | 'week' | 'book' | 'company' | 'document';
export type WorkspaceEntry = RunningNote | Note | JournalWeek | WorkspaceBook | WorkspaceCompany | Document | PersonalJournalEntry;
export interface LegacyWorkspace {
  version: 1;
  notes: Note[];
  weeks: JournalWeek[];
}
export interface Workspace {
  journalsVersion?: 2;
  personalJournal?: PersonalJournalEntry[];
  runningNotes?: RunningNote[];
  /** Saved status and dependsOn are authoritative; readiness is derived. */
  goals?: import('./timeline').Goal[];
  version: 2;
  catalogVersion?: 1;
  notes: Note[];
  weeks: JournalWeek[];
  books: WorkspaceBook[];
  companies: WorkspaceCompany[];
  documents: Document[];
}
export function migrateWorkspace(existing: LegacyWorkspace | Workspace | null, seed: Workspace): Workspace {
  if (!existing) return seed;
  if (existing.version === 2) return existing;
  if (existing.version !== 1 || !Array.isArray(existing.notes) || !Array.isArray(existing.weeks)) {
    throw new Error('Unsupported workspace version. Existing data was not changed.');
  }
  return { ...existing, version: 2, books: seed.books, companies: seed.companies, documents: seed.documents };
}

function replaceSection(body: string, heading: string, replacement: string): string {
  const pattern = new RegExp(`(^## ${heading}\\s*\\r?\\n)[\\s\\S]*?(?=^## |$(?![\\s\\S]))`, 'mi');
  return pattern.test(body) ? body.replace(pattern, (_match, header) => `${header}\n${replacement.trim()}\n\n`) : `${body.trim()}\n\n## ${heading}\n\n${replacement.trim()}\n`;
}

/** Apply factual catalog fixes once while retaining user-written notes and logs. */
export function updateBookCatalog(existing: Workspace, seed: Workspace): Workspace {
  if (existing.catalogVersion === 1) return existing;
  const correctedBySlug = new Map(seed.books.map(book => [book.slug, book]));
  const targets = new Set(['ostep', 'network-programming-using-internet-sockets', 'database-internals', 'system-design-interview-vol-1', 'system-design-interview-vol-2']);
  const books = existing.books.map(saved => {
    const corrected = correctedBySlug.get(saved.slug);
    if (!corrected || !targets.has(saved.slug)) return saved;
    let body = saved.body;
    if (saved.slug === 'network-programming-using-internet-sockets') {
      if (/^- \[[ xX]\] TODO\s*$/m.test(body)) body = replaceSection(body, 'Chapters', corrected.body.match(/^## Chapters\s*\n([\s\S]*?)(?=^## )/m)?.[1] ?? '');
      if (body.includes('Current reading for the freeCodeCamp eBPF course')) body = replaceSection(body, 'Why', corrected.body.match(/^## Why\s*\n([\s\S]*?)(?=^## )/m)?.[1] ?? '');
    }
    if (saved.slug === 'ostep') {
      const old = body.match(/^## Chapters\s*\n([\s\S]*?)(?=^## )/m)?.[1] ?? '';
      const oldTasks = old.split('\n').filter(line => /^\s*- \[[ xX]\]/.test(line));
      if (oldTasks.length === 14 && oldTasks[0]?.includes('Intro: Processes')) {
        const completedGroups = oldTasks.map(line => /\[[xX]\]/.test(line));
        const groups = [[4, 5], [6], [7, 8, 9, 10], [13, 14], [15, 16], [18, 19, 20], [21, 22], [26, 27, 28, 29], [30, 31], [32, 33], [36, 37], [39, 40], [41, 42, 43], [45, 48, 49, 50]];
        const completed = new Set(groups.flatMap((chapters, index) => completedGroups[index] ? chapters : []));
        const newList = (corrected.body.match(/^## Chapters\s*\n([\s\S]*?)(?=^## )/m)?.[1] ?? '').split('\n').map(line => {
          const chapter = /^- \[ \] (\d+)\./.exec(line)?.[1];
          return chapter && completed.has(Number(chapter)) ? line.replace('[ ]', '[x]') : line;
        }).join('\n');
        body = replaceSection(body, 'Chapters', newList);
      }
    }
    return {
      ...saved,
      ...(saved.slug === 'network-programming-using-internet-sockets' ? { title: corrected.title, authors: corrected.authors, edition: corrected.edition, tags: corrected.tags } : {}),
      ...Object.fromEntries(['cover', 'isbn', 'url', 'edition', 'authors'].flatMap(key => {
        const current = saved[key as keyof typeof saved];
        const populated = Array.isArray(current) ? current.length > 0 : Boolean(current);
        return populated ? [] : [[key, corrected[key as keyof typeof corrected]]];
      }).filter(([, value]) => value !== undefined)),
      body,
    } as WorkspaceBook;
  });
  return { ...existing, catalogVersion: 1, books };
}
export const collectionKey = { run: 'runningNotes', personal: 'personalJournal', note: 'notes', week: 'weeks', book: 'books', company: 'companies', document: 'documents' } as const;
export const getEntryId = (entry: WorkspaceEntry) => 'id' in entry ? entry.id : entry.slug;
export const getEntryTitle = (entry: WorkspaceEntry) => 'title' in entry ? entry.title : `Week ${entry.week} · ${entry.dates}`;
export function workspaceEntries(data: Workspace, kind: EntryKind): WorkspaceEntry[] { return data[collectionKey[kind]] ?? []; }
export function updateWorkspaceEntry(data: Workspace, kind: EntryKind, id: string, entry?: WorkspaceEntry): Workspace {
  const entries = workspaceEntries(data, kind).filter(e => getEntryId(e) !== id);
  return { ...data, [collectionKey[kind]]: entry ? [...entries, entry] : entries };
}
export interface Snapshot { data: Workspace; revision: string | null }

export function localDate(date = new Date()): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

export function newWeek(date: string): JournalWeek {
  const day = new Date(`${date}T00:00:00Z`);
  const week = weekNumberFor(day);
  const start = new Date(PLAN_START.getTime() + (week - 1) * 7 * 86400000);
  const end = new Date(start.getTime() + 6 * 86400000);
  return {
    slug: `${start.getUTCFullYear()}/week-${String(week).padStart(2, '0')}`,
    week, year: start.getUTCFullYear(),
    dates: `${start.toISOString().slice(0, 10)} to ${end.toISOString().slice(0, 10)}`,
    hours: {}, tags: [], body: '## What I did\n\n## Blockers\n\n## Notes\n',
  };
}

/** Insert into the existing daily section without rewriting other journal text. */
export function appendDailyEntry(body: string, date: string, text: string): string {
  const day = new Date(`${date}T00:00:00Z`);
  const weekday = day.toLocaleDateString('en-US', { weekday: 'long', timeZone: 'UTC' });
  const label = day.toLocaleDateString('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' });
  const lines = body.split('\n');
  let section = lines.findIndex(line => /^## What I did\s*$/i.test(line));
  if (section === -1) return `## What I did\n\n### ${weekday} — ${label}\n${text.trim()}\n\n${body}`;
  let end = lines.findIndex((line, i) => i > section && /^## /.test(line));
  if (end === -1) end = lines.length;
  const heading = lines.findIndex((line, i) => i > section && i < end && line.startsWith(`### ${weekday}`));
  if (heading !== -1) {
    const next = lines.findIndex((line, i) => i > heading && i < end && /^### /.test(line));
    const at = next === -1 ? end : next;
    lines.splice(at, 0, text.trim(), '');
  } else {
    lines.splice(end, 0, `### ${weekday} — ${label}`, text.trim(), '');
  }
  return lines.join('\n');
}

/** Recognize only the generated scaffolding; preserve custom headings and all metrics. */
export function isEmptyStarterWeek(week: JournalWeek): boolean {
  if (week.updatedAt || week.tags.length || Object.values(week.hours).some(Boolean)
    || Object.values(week.targets ?? {}).some(Boolean)) return false;
  return week.body.split('\n').every(line => {
    const text = line.trim();
    return !text || /^[-*]$/.test(text) || /^## (What I did|Blockers|Notes)$/.test(text)
      || /^### (Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)(?: [—–-] [A-Z][a-z]{2} \d{1,2})?$/.test(text);
  });
}

/** One-time journal migration; never re-add a deleted personal entry. */
export function migrateJournals(data: Workspace, bio: string): Workspace {
  if (data.journalsVersion === 2) return data;
  const companies = data.companies.map(company => {
    // Add the relationship-first steps to old saved checklists while preserving
    // the user's existing checked state and all other company notes.
    if (company.body.includes('Identify one person at the company to connect with')) return company;
    const body = company.body.replace(
      /(^- \[[ xX]\] Research team & open roles\r?\n)(^- \[[ xX]\] Tailor resume\/cover letter\r?\n)(^- \[[ xX]\] Submit application)/m,
      '$1- [ ] Identify one person at the company to connect with\n- [ ] Reach out and start building a relationship\n$2$3',
    );
    return body === company.body ? company : { ...company, body };
  });
  return { ...data, journalsVersion: 2, companies,
    weeks: data.weeks.filter(week => !isEmptyStarterWeek(week)),
    personalJournal: data.personalJournal ?? (bio.trim() ? [{
      id: 'bio', title: 'Personal history and background', date: '', description: 'Background imported from bio.md; dates and plans in this entry are historical context.',
      tags: ['background'], body: bio,
    }] : []),
  };
}
