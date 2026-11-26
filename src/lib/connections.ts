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
  /** Catch-up queue stage; absent when the person is not in the queue. */
  stage?: ConnectionStage;
  /** Hidden from the catch-up queue until this date (YYYY-MM-DD). */
  snoozedUntil?: string;
  notes: string;
  tags: string[];
  /** Read-only photo revision; absent when no photo is stored. */
  photo?: string;
  updatedAt?: string;
}
/** Catch-up queue stages in order (internal/database/connection_queue.go). */
export const CONNECTION_STAGES = ['to_contact', 'messaged', 'follow_up', 'referral_ask'] as const;
export type ConnectionStage = typeof CONNECTION_STAGES[number];
/** This week's "sent" count against the weekly target, Monday to Sunday. */
export interface CatchUpWeek { start: string; end: string; sent: number; target: number }
/** GET /api/connections/queue: everyone queued, snoozed included, by stage then name. */
export interface ConnectionQueue { today: string; week: CatchUpWeek; connections: Connection[]; revisions: Record<string, string> }
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

/** Group linked connections by company slug, most recent touchpoint first. */
export function connectionsByCompany(connections: Connection[]): Map<string, Connection[]> {
  const groups = new Map<string, Connection[]>();
  for (const connection of connections) {
    if (!connection.companySlug) continue;
    const group = groups.get(connection.companySlug) ?? [];
    group.push(connection);
    groups.set(connection.companySlug, group);
  }
  for (const group of groups.values()) group.sort((a, b) => (lastTouch(b) ?? '').localeCompare(lastTouch(a) ?? '') || a.name.localeCompare(b.name));
  return groups;
}

/** The people to show in an avatar stack of at most `max` circles, and how many hide behind "+N". */
export function avatarStack<T>(people: T[], max: number): { shown: T[]; overflow: number } {
  // "+1" would take the same space as the one person it hides.
  const shown = people.length > max ? people.slice(0, max - 1) : people;
  return { shown, overflow: people.length - shown.length };
}

const LEGAL_SUFFIXES = new Set(['inc', 'llc', 'ltd', 'limited', 'corp', 'corporation', 'co', 'company', 'gmbh', 'plc', 'sa', 'ag', 'bv', 'the']);

/** Lowercase, drop punctuation and legal suffixes; mirrors linkedin.NormalizeCompany in Go. */
export function normalizeCompany(name: string): string {
  return name.toLowerCase().replaceAll('&', ' and ').replace(/[^a-z0-9]+/g, ' ').split(' ').filter(word => word && !LEGAL_SUFFIXES.has(word)).join(' ');
}

/** One employer in the network: a tracked company (slug) or a LinkedIn company name. */
export interface Employer { key: string; name: string; count: number; slug?: string }

/**
 * People per employer, most first. Linked people count toward their tracked company; the rest
 * group by LinkedIn company name with case, punctuation and legal suffixes ignored.
 */
export function employers(connections: Connection[], titles: Map<string, string>): Employer[] {
  const groups = new Map<string, Employer>();
  for (const c of connections) {
    const key = c.companySlug ? `slug:${c.companySlug}` : `name:${normalizeCompany(c.companyName)}`;
    if (key === 'name:') continue;
    const group = groups.get(key);
    if (group) group.count++;
    else groups.set(key, c.companySlug
      ? { key, name: titles.get(c.companySlug) ?? (c.companyName.trim() || c.companySlug), count: 1, slug: c.companySlug }
      : { key, name: c.companyName.trim(), count: 1 });
  }
  return [...groups.values()].sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
}

/** A URL-safe company slug from its name ("Prime Digital Academy" -> "prime-digital-academy"). */
export function companySlug(title: string): string {
  return title.normalize('NFKD').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 80);
}

/**
 * A not-started company for the companies board, like add_companies_to_queue builds
 * (internal/server/mcp_connections.go): the website's outreach checklist and an empty Log.
 */
export function newCompanyEntry(title: string, slug: string) {
  return {
    slug, title, category: 'From connections', type: 'company', status: 'not_started', featured: false, priority: 'medium', tags: [],
    url: `https://www.linkedin.com/search/results/companies/?keywords=${encodeURIComponent(title)}`,
    body: '## Why\n\n## Steps\n\n- [ ] Identify one person at the company to connect with\n- [ ] Reach out and start building a relationship\n- [ ] Research team & open roles\n- [ ] Tailor resume/cover letter\n- [ ] Apply\n\n## Log\n',
  };
}
