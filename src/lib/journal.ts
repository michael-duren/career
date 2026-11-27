// Pure helpers behind the Journal page (docs/redesign/pages/journal.md). Kept free of React so
// node --test can cover them.
import { formatDate } from './format.ts';
import type { WeekEntry } from './progress.ts';

const WEEKDAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
const MONTHS = ['jan', 'feb', 'mar', 'apr', 'may', 'jun', 'jul', 'aug', 'sep', 'oct', 'nov', 'dec'];
const DAY_MS = 86400000;
// "Nov 23", "November 23", "Sept. 4": real month names only, so "Marketing 2 pager" is not March 2.
const MONTH_DAY = /\b(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)(?:uary|ruary|ch|il|ust|t|tember|ober|ember|(?<=jun)e|(?<=jul)y)?\.?\s+(\d{1,2})\b/i;

/** The ISO dates in a week's "2026-11-23 to 2026-11-29" range, or null when it doesn't parse. */
export function weekBounds(dates: string): { start: string; end: string } | null {
  const found = dates.match(/\d{4}-\d{2}-\d{2}/g);
  if (!found || found.length === 0) return null;
  return { start: found[0], end: found[1] ?? found[0] };
}

function validDate(iso: string): boolean {
  const date = new Date(`${iso}T00:00:00Z`);
  return !Number.isNaN(date.getTime()) && date.toISOString().slice(0, 10) === iso;
}

/** "Nov 23 – 29", or "Jul 27 – Aug 2" across a month; the raw text when it doesn't parse. */
export function weekRange(dates: string): string {
  const bounds = weekBounds(dates);
  if (!bounds || !validDate(bounds.start) || !validDate(bounds.end)) return dates;
  const start = formatDate(bounds.start), end = formatDate(bounds.end);
  if (start === end) return start;
  return start.slice(0, 3) === end.slice(0, 3) ? `${start} – ${end.slice(4)}` : `${start} – ${end}`;
}

function addDays(iso: string, days: number): string {
  return new Date(new Date(`${iso}T00:00:00Z`).getTime() + days * DAY_MS).toISOString().slice(0, 10);
}

export type JournalSectionKind = 'what' | 'blockers' | 'notes' | 'other';
/** One day under What I did (label "Monday · Nov 23"), or the single unlabeled group of other sections. */
export interface JournalGroup { label: string; date?: string; items: string[] }
export interface JournalSection {
  kind: JournalSectionKind;
  title: string;
  groups: JournalGroup[];
  /** The section's raw Markdown, used instead of the ruled list when `rich` is set. */
  markdown: string;
  /** Code, tables or quotes that a plain ruled list would mangle. */
  rich: boolean;
}

const STANDARD: { kind: Exclude<JournalSectionKind, 'other'>; title: string }[] = [
  { kind: 'what', title: 'What I did' },
  { kind: 'blockers', title: 'Blockers' },
  { kind: 'notes', title: 'Notes' },
];

function sectionKind(title: string): JournalSectionKind {
  const text = title.trim().toLowerCase();
  if (/^what\b/.test(text)) return 'what';
  if (/^blockers?\b/.test(text)) return 'blockers';
  if (/^notes?\b/.test(text)) return 'notes';
  return 'other';
}

/** Turns "Wednesday" or "Monday — Nov 23" into a label and, when it can, the day's date. */
function dayGroup(heading: string, weekStart: string | null): JournalGroup {
  const text = heading.trim();
  const index = WEEKDAYS.findIndex(day => text.toLowerCase().startsWith(day.toLowerCase()));
  if (index === -1) return { label: text, items: [] };
  const day = WEEKDAYS[index];
  let date: string | undefined;
  const named = MONTH_DAY.exec(text.slice(day.length));
  if (weekStart && validDate(weekStart)) {
    date = addDays(weekStart, index);
    // A heading's own month and day win when they fall in a different week than its weekday.
    const month = named ? MONTHS.indexOf(named[1].toLowerCase()) : -1;
    if (named && month >= 0) {
      const year = Number(weekStart.slice(0, 4));
      const candidates = [year - 1, year, year + 1].map(y => `${y}-${String(month + 1).padStart(2, '0')}-${String(Number(named[2])).padStart(2, '0')}`).filter(validDate);
      const nearest = candidates.sort((a, b) => Math.abs(Date.parse(a) - Date.parse(date!)) - Math.abs(Date.parse(b) - Date.parse(date!)))[0];
      if (nearest) date = nearest;
    }
  }
  return { label: date ? `${day} · ${formatDate(date)}` : day, date, items: [] };
}

function addLine(group: JournalGroup, raw: string) {
  const text = raw.trim();
  if (!text || /^[-*+]$/.test(text)) return;
  const bullet = /^(?:[-*+]|\d+[.)])\s+(.*)$/.exec(text);
  // An indented line without a bullet continues the item above it.
  if (!bullet && /^\s/.test(raw) && group.items.length) {
    group.items[group.items.length - 1] += ` ${text}`;
    return;
  }
  group.items.push(bullet ? bullet[1] : text);
}

/**
 * Splits a week's Markdown into What I did (grouped by "### Day" headings), Blockers, Notes,
 * then any other "## " sections in body order. The three standard sections are always present.
 */
export function parseWeekBody(body: string, weekStart: string | null): JournalSection[] {
  const sections: JournalSection[] = [];
  let current: JournalSection | null = null;
  let fence = false;
  const start = (kind: JournalSectionKind, title: string) => {
    current = { kind, title, groups: [], markdown: '', rich: false };
    sections.push(current);
    return current;
  };
  for (const raw of body.replace(/\r\n?/g, '\n').split('\n')) {
    const heading = !fence && /^##\s+(.*)$/.exec(raw);
    if (heading && !raw.startsWith('###')) { start(sectionKind(heading[1]), heading[1].trim()); continue; }
    const section: JournalSection = current ?? start('other', '');
    section.markdown += `${raw}\n`;
    if (/^\s*(```|~~~)/.test(raw)) { fence = !fence; section.rich = true; continue; }
    if (fence) continue;
    // Tables, quotes, deep headings, nested or numbered lists and task boxes need real Markdown.
    if (/^\s*(\||>)/.test(raw) || /^#{4,}\s/.test(raw) || /^\s+([-*+]|\d+[.)])\s/.test(raw) || /^\s*\d+[.)]\s/.test(raw) || /^\s*[-*+]\s+\[[ xX]\]/.test(raw)) section.rich = true;
    const day = /^###\s+(.*)$/.exec(raw);
    if (day) { section.groups.push(section.kind === 'what' ? dayGroup(day[1], weekStart) : { label: day[1].trim(), items: [] }); continue; }
    if (!raw.trim()) continue;
    if (!section.groups.length) section.groups.push({ label: '', items: [] });
    addLine(section.groups[section.groups.length - 1], raw);
  }
  for (const section of sections) {
    section.markdown = section.markdown.trim();
    section.groups = section.groups.filter(group => group.items.length || group.label);
  }
  // Merge repeated standard sections and drop empty unnamed preambles.
  const merged = STANDARD.map(({ kind, title }) => {
    const parts = sections.filter(section => section.kind === kind);
    return {
      kind, title,
      groups: parts.flatMap(part => part.groups),
      markdown: parts.map(part => part.markdown).filter(Boolean).join('\n\n'),
      rich: parts.some(part => part.rich),
    };
  });
  const others = sections.filter(section => section.kind === 'other' && (section.title || section.markdown));
  return [...merged, ...others];
}

/** True when a section has nothing written in it. */
export function sectionEmpty(section: JournalSection): boolean {
  return section.rich ? !section.markdown.trim() : section.groups.every(group => group.items.length === 0);
}

/**
 * The date of the most recent written day across all weeks: the latest dated "### Day" group
 * with items, or, for a week with writing but no day headings, the week's start date.
 */
export function latestEntryDate(weeks: Pick<WeekEntry, 'dates' | 'body'>[]): string | null {
  let latest: string | null = null;
  for (const week of weeks) {
    const start = weekBounds(week.dates)?.start ?? null;
    const sections = parseWeekBody(week.body ?? '', start);
    const dated = sections.flatMap(section => section.groups).filter(group => group.date && group.items.length).map(group => group.date!);
    const written = sections.some(section => !sectionEmpty(section));
    const candidate = dated.sort().at(-1) ?? (written && start && validDate(start) ? start : null);
    if (candidate && (!latest || candidate > latest)) latest = candidate;
  }
  return latest;
}

/** The accent "catch up" note: only when the latest entry is more than 7 days before today. */
export function staleNote(latest: string | null, today: string): string | null {
  if (!latest || !validDate(latest) || !validDate(today)) return null;
  const days = Math.round((Date.parse(today) - Date.parse(latest)) / DAY_MS);
  return days > 7 ? `last entry ${formatDate(latest)} — catch up?` : null;
}

function tokens(query: string): string[] {
  return query.trim().toLowerCase().split(/\s+/).filter(Boolean);
}

/** Every search word appears in the week's number, dates, tags or text. */
export function weekMatches(week: Pick<WeekEntry, 'week' | 'dates' | 'tags' | 'body'>, query: string): boolean {
  const words = tokens(query);
  if (!words.length) return true;
  const hay = `w${week.week} week ${week.week} ${week.dates} ${weekRange(week.dates)} ${week.tags.join(' ')} ${week.body ?? ''}`.toLowerCase();
  return words.every(word => hay.includes(word));
}

/** Every search word appears in the entry's title, summary, date or tags (list results carry no body). */
export function personalMatches(entry: { title: string; description: string; date: string; tags: string[] }, query: string): boolean {
  const words = tokens(query);
  if (!words.length) return true;
  const hay = `${entry.title} ${entry.description} ${entry.date} ${entry.date && validDate(entry.date) ? formatDate(entry.date) : ''} ${entry.tags.join(' ')}`.toLowerCase();
  return words.every(word => hay.includes(word));
}

/** "4 work weeks · 1 personal". */
export function journalCounts(weeks: number, personal: number): string {
  return `${weeks} work week${weeks === 1 ? '' : 's'} · ${personal} personal`;
}

/** Hours rounded to at most one decimal: 2, 2.5. */
export function hoursText(hours: number): string {
  return `${Math.round(hours * 10) / 10}`;
}

/** The text on a week's measurement line: recorded goal hours first, then older per-track hours. */
export function weekHoursLabel(goalHours: number, legacyHours: number): string {
  if (goalHours > 0) return `${hoursText(goalHours)} h logged this week`;
  if (legacyHours > 0) return `${hoursText(legacyHours)} h in earlier tracks`;
  return 'No hours logged this week';
}

/** A personal entry's date for lists ("Nov 24"), or "Undated" for background entries. */
export function personalDate(date: string): string {
  return date && validDate(date) ? formatDate(date) : 'Undated';
}
