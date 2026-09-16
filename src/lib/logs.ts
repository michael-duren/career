import { TRACK_META, type WeekEntry } from './progress';

// ---------------------------------------------------------------------------
// Log document model
//
// The progress dashboard treats each week as a row of metrics. The Log Explorer
// instead treats every daily journal entry (and the Blockers / Notes sections)
// as an individual "document" — a Kibana Discover record — so they can be
// searched, faceted, and expanded one by one. Weekly structured fields (hours,
// leetcode, tags, PRs) ride along on each document as context.
// ---------------------------------------------------------------------------

export interface LogPart {
  /** Marker prefix found at the start of a line, e.g. "LC" / "OSS". "" = freeform. */
  label: string;
  text: string;
}

export type LogSection = 'day' | 'blockers' | 'notes';

export interface LogDoc {
  id: string;
  week: number;
  year: number;
  weekDates: string;
  section: LogSection;
  /** Weekday name for `day` sections, otherwise "". */
  day: string;
  /** Display heading, e.g. "Monday — Jun 22" or "Blockers". */
  dateLabel: string;
  /** 0=Mon … 6=Sun, 7=blockers, 8=notes — for ordering. */
  dayIndex: number;
  tags: string[];
  /** Weekly hours by track. Keys are dynamic — new tracks just appear here. */
  hours: Record<string, number>;
  /** Hour keys with > 0 hours that week — used for the "track" facet. */
  activeTracks: string[];
  parts: LogPart[];
  /** Flattened text of all parts — the search haystack body. */
  message: string;
  sortKey: number;
}

export interface WeekSummary {
  week: number;
  year: number;
  dates: string;
  hours: Record<string, number>;
  tags: string[];
}

const DAY_ORDER = [
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
  'Sunday',
];

function dayIndexOf(name: string): number {
  const i = DAY_ORDER.indexOf(name);
  return i === -1 ? 6 : i;
}

// Known track colors mirror the dashboard; unknown (future) tracks get a stable
// generated color so newly added hour keys render consistently across reloads.
const KNOWN_TRACK_COLORS: Record<string, string> = {
  algorithms: '#60a5fa',
  systemDesign: '#34d399',
  osOss: '#a78bfa',
  languages: '#fbbf24',
};

export function trackColor(key: string): string {
  if (TRACK_META[key]) return TRACK_META[key].color;
  if (KNOWN_TRACK_COLORS[key]) return KNOWN_TRACK_COLORS[key];
  let h = 0;
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) % 360;
  return `hsl(${h} 60% 62%)`;
}

const MONTHS = [
  'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
  'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
];

/** "2026-06-22 to 2026-06-28" → "Jun 22" (the week's start date). */
export function shortDate(dates: string): string {
  const m = /(\d{4})-(\d{2})-(\d{2})/.exec(dates);
  if (!m) return dates;
  const month = MONTHS[parseInt(m[2], 10) - 1] ?? m[2];
  return `${month} ${parseInt(m[3], 10)}`;
}

export function trackLabel(key: string): string {
  if (TRACK_META[key]) return TRACK_META[key].name;
  return key
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/^./, (c) => c.toUpperCase());
}

// Split a block of lines into labeled parts. Lines beginning with a short marker
// like "LC:" or "OSS:" start a new part; following wrapped lines append to it.
function toParts(lines: string[]): LogPart[] {
  const parts: LogPart[] = [];
  let cur: LogPart | null = null;
  for (const l of lines) {
    const t = l.trim();
    if (t === '' || t === '-' || t === '*') continue;
    const m = /^([A-Z][A-Za-z]{0,11}):\s*(.*)$/.exec(t);
    if (m) {
      cur = { label: m[1], text: m[2] };
      parts.push(cur);
    } else if (cur) {
      cur.text += (cur.text ? ' ' : '') + t;
    } else {
      cur = { label: '', text: t };
      parts.push(cur);
    }
  }
  return parts.filter((p) => p.text.trim() !== '' || p.label !== '');
}

function partsToMessage(parts: LogPart[]): string {
  return parts.map((p) => (p.label ? `${p.label}: ${p.text}` : p.text)).join('\n');
}

function makeDoc(
  w: WeekEntry,
  activeTracks: string[],
  section: LogSection,
  dateLabel: string,
  day: string,
  dayIndex: number,
  parts: LogPart[],
): LogDoc {
  return {
    id: `${w.year}-w${w.week}-${section}-${day || section}`,
    week: w.week,
    year: w.year,
    weekDates: w.dates,
    section,
    day,
    dateLabel,
    dayIndex,
    tags: w.tags,
    hours: w.hours,
    activeTracks,
    parts,
    message: partsToMessage(parts),
    sortKey: w.week * 100 + dayIndex,
  };
}

/** Parse all week entries into individual, newest-first log documents. */
export function buildLogDocs(weeks: WeekEntry[]): LogDoc[] {
  const docs: LogDoc[] = [];

  for (const w of weeks) {
    const activeTracks = Object.keys(w.hours).filter((k) => (w.hours[k] || 0) > 0);
    const lines = (w.body || '').split('\n');

    let section: 'what' | 'blockers' | 'notes' | 'other' = 'other';
    let dayHeading: string | null = null;
    let buf: string[] = [];

    const flushDay = () => {
      if (section !== 'what') return;
      if (section === 'what' && dayHeading) {
        const parts = toParts(buf);
        if (parts.length) {
          const day = dayHeading.split(/[\s—–-]/)[0].trim();
          docs.push(
            makeDoc(w, activeTracks, 'day', dayHeading, day, dayIndexOf(day), parts),
          );
        }
      }
      buf = [];
    };

    const flushBlock = () => {
      if (section !== 'blockers' && section !== 'notes') return;
      if (section === 'blockers' || section === 'notes') {
        const parts = toParts(buf);
        if (parts.length) {
          const label = section === 'blockers' ? 'Blockers' : 'Notes';
          docs.push(
            makeDoc(w, activeTracks, section, label, '', section === 'blockers' ? 7 : 8, parts),
          );
        }
      }
      buf = [];
    };

    for (const raw of lines) {
      const line = raw.trimEnd();

      if (/^##\s+/.test(line) && !/^###\s+/.test(line)) {
        flushDay();
        flushBlock();
        dayHeading = null;
        const title = line.replace(/^##\s+/, '').toLowerCase();
        section = title.includes('blocker')
          ? 'blockers'
          : title.includes('note')
            ? 'notes'
            : title.includes('what')
              ? 'what'
              : 'other';
        continue;
      }

      if (/^###\s+/.test(line)) {
        if (section === 'what') {
          flushDay();
          dayHeading = line.replace(/^###\s+/, '').trim();
        }
        continue;
      }

      buf.push(line);
    }

    flushDay();
    flushBlock();
  }

  docs.sort((a, b) => b.week - a.week || b.dayIndex - a.dayIndex);
  return docs;
}

/** Union of every hour key across all weeks, in first-seen order. */
export function allHourKeys(weeks: WeekSummary[]): string[] {
  const seen: string[] = [];
  for (const w of weeks) {
    for (const k of Object.keys(w.hours)) {
      if (!seen.includes(k)) seen.push(k);
    }
  }
  return seen;
}
