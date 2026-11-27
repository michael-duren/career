// Display helpers for the weekly scheduler page (docs/redesign/pages/schedule.md).
// Pure functions so node --test can cover them; the page lives in WeeklyScheduler*.tsx.
import { goalPattern, type GoalPatternName } from './ui.ts';
import type { SchedulerSession, SchedulerWeek } from './scheduler.ts';

const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];
const DAYS = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT'];

const utc = (date: string) => new Date(`${date}T12:00:00Z`);

// isoWeek is the ISO 8601 week number of a YYYY-MM-DD date (weeks start Monday; week 1 holds the first Thursday).
export function isoWeek(date: string): number {
  const day = utc(date);
  const thursday = new Date(day);
  thursday.setUTCDate(day.getUTCDate() + 3 - ((day.getUTCDay() + 6) % 7));
  const firstThursday = new Date(Date.UTC(thursday.getUTCFullYear(), 0, 4, 12));
  firstThursday.setUTCDate(firstThursday.getUTCDate() + 3 - ((firstThursday.getUTCDay() + 6) % 7));
  return 1 + Math.round((thursday.getTime() - firstThursday.getTime()) / (7 * 86400000));
}

// weekRangeLabel renders a Monday-to-Sunday week as "23 — 29 November", or "30 November — 6 December"
// across months. short uses three-letter months for the phone title ("23 — 29 Nov").
export function weekRangeLabel(monday: string, short = false): string {
  const start = utc(monday);
  const end = new Date(start);
  end.setUTCDate(start.getUTCDate() + 6);
  const month = (date: Date) => (short ? MONTHS[date.getUTCMonth()].slice(0, 3) : MONTHS[date.getUTCMonth()]);
  if (start.getUTCMonth() === end.getUTCMonth()) return `${start.getUTCDate()} — ${end.getUTCDate()} ${month(end)}`;
  return `${start.getUTCDate()} ${month(start)} — ${end.getUTCDate()} ${month(end)}`;
}

// dayParts splits a date into the column/strip label pieces: { dow: 'MON', num: 23 }.
export function dayParts(date: string): { dow: string; num: number } {
  const day = utc(date);
  return { dow: DAYS[day.getUTCDay()], num: day.getUTCDate() };
}

// goalPatterns gives each goal of the week its pattern by position in week.goals (which the API
// sorts by title, so a goal keeps its pattern from week to week while the set of goals is the same).
// Goals after the third, and goals not in this week's list, fall back to an outline.
export function goalPatterns(week: Pick<SchedulerWeek, 'goals'>): Map<string, GoalPatternName> {
  return new Map(week.goals.map((summary, index) => [summary.goal.id, goalPattern(index)]));
}

export type BlockState = 'planned' | 'done' | 'skipped';
// blockState: recorded work (assumed or explicit) is done, a skipped actual is skipped, anything else is still a plan.
export function blockState(session: Pick<SchedulerSession, 'actual'>): BlockState {
  if (session.actual?.status === 'skipped') return 'skipped';
  return session.actual ? 'done' : 'planned';
}

export type GoalHours = { required: number | null; planned: number; delta: number | null; short: boolean };
// goalHours is one GOAL HOURS row: required hours, planned (recorded plus still scheduled), and
// the difference, which is negative when the week falls short.
export function goalHours(summary: Pick<SchedulerWeek['goals'][number], 'requiredHours' | 'actualHours' | 'remainingScheduledHours' | 'uncoveredHours'>): GoalHours {
  const planned = summary.actualHours + summary.remainingScheduledHours;
  const required = summary.requiredHours;
  return { required, planned, delta: required === null ? null : planned - required, short: summary.uncoveredHours > 0 };
}

// hoursCell renders a table number with one decimal ("10.5", "4.0"); deltas get a sign ("+5.5", "−1.5").
export function hoursCell(value: number | null, signed = false): string {
  if (value === null) return '—';
  const rounded = Math.round(value * 10) / 10;
  const text = Math.abs(rounded).toFixed(1);
  if (!signed || rounded === 0) return text;
  return `${rounded > 0 ? '+' : '−'}${text}`;
}
