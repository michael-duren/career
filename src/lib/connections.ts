export interface Connection {
  id: string;
  name: string;
  role: string;
  companyName: string;
  companySlug?: string;
  email: string;
  url?: string;
  connectedOn?: string;
  lastContactedOn?: string;
  cadenceDays?: number;
  queued: boolean;
  notes: string;
  tags: string[];
  /** Read-only photo revision; absent when no photo is stored. */
  photo?: string;
  updatedAt?: string;
}
export interface ConnectionImport { parsed: number; created: number; updated: number; unchanged: number; skipped: number; linked: number; messagesMatched: number }
export interface QueueItem { connection: Connection; due: string; overdueDays: number; reason: 'manual' | 'cadence' }

export const CADENCES = [14, 30, 60, 90, 180, 365] as const;
const DAY = 86400000;

function utc(date: string): number { return Date.parse(`${date}T00:00:00Z`); }
export function addDays(date: string, days: number): string { return new Date(utc(date) + days * DAY).toISOString().slice(0, 10); }
export function daysBetween(from: string, to: string): number { return Math.round((utc(to) - utc(from)) / DAY); }

/** The last known touchpoint: a conversation, else the day you connected. */
export function lastTouch(c: Connection): string | undefined { return c.lastContactedOn ?? c.connectedOn; }

/** Next catch-up date, or undefined when no cadence is set. Never-contacted people are due immediately. */
export function nextDue(c: Connection, today: string): string | undefined {
  if (!c.cadenceDays) return undefined;
  const touched = lastTouch(c);
  return touched ? addDays(touched, c.cadenceDays) : today;
}

/** People to catch up with: manually queued or past their cadence, most overdue first. */
export function catchUpQueue(connections: Connection[], today: string): QueueItem[] {
  const items: QueueItem[] = [];
  for (const connection of connections) {
    const due = nextDue(connection, today);
    if (due && due <= today) items.push({ connection, due, overdueDays: daysBetween(due, today), reason: 'cadence' });
    else if (connection.queued) items.push({ connection, due: today, overdueDays: 0, reason: 'manual' });
  }
  return items.sort((a, b) => b.overdueDays - a.overdueDays || a.connection.name.localeCompare(b.connection.name));
}

/** Logging a conversation clears the manual flag and restarts the cadence. */
export function markCaughtUp(c: Connection, today: string): Connection { return { ...c, lastContactedOn: today, queued: false }; }

export function initials(name: string): string {
  return name.split(/\s+/).filter(Boolean).map(word => word[0]).join('').slice(0, 2).toUpperCase() || '?';
}

export function relativeDays(date: string | undefined, today: string): string {
  if (!date) return 'Never';
  const days = daysBetween(date, today);
  if (days <= 0) return days === 0 ? 'Today' : `In ${-days}d`;
  if (days < 60) return `${days}d ago`;
  if (days < 730) return `${Math.round(days / 30)}mo ago`;
  return `${Math.round(days / 365)}y ago`;
}

export function matchesQuery(c: Connection, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  return [c.name, c.role, c.companyName, c.email, c.notes, ...c.tags].some(value => value.toLowerCase().includes(q));
}

export function newConnection(company?: { slug: string; title: string }): Connection {
  return { id: crypto.randomUUID(), name: '', role: '', companyName: company?.title ?? '', companySlug: company?.slug, email: '', queued: false, notes: '', tags: [] };
}

export function photoURL(c: Connection): string | undefined {
  return c.photo ? `/api/connections/photo?id=${encodeURIComponent(c.id)}&v=${encodeURIComponent(c.photo)}` : undefined;
}
