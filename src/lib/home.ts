// Pure helpers for Home (docs/redesign/pages/home.md): the done count and
// "next" note, session durations, the ISO week and the streak scale.
import { formatTime } from './format.ts';
import type { SchedulerSession } from './scheduler.ts';

/** Checked off here, or recorded with explicit times in the scheduler. Skipped is not done. */
export const isDone = (session: SchedulerSession) => Boolean(session.actual && session.actual.status !== 'skipped');
export const isSkipped = (session: SchedulerSession) => session.actual?.status === 'skipped';

/**
 * The next session: the first one (in start order) neither done nor skipped that
 * has not ended yet, so a session in progress stays next until it ends.
 */
export function nextSession(sessions: SchedulerSession[], now: number): SchedulerSession | undefined {
  return sessions.find(session => session.plan && !isDone(session) && !isSkipped(session) && Date.parse(session.plan.end) > now);
}

/** The accent note: "1 of 3 done — next in 25 min", "… — next at 5:45 PM" or "… — now until 6:45 PM". */
export function progressNote(sessions: SchedulerSession[], now: number, timeZone?: string): string {
  const done = `${sessions.filter(isDone).length} of ${sessions.length} done`;
  const next = nextSession(sessions, now);
  if (!next?.plan) return done;
  const start = Date.parse(next.plan.start);
  if (start <= now) return `${done} — now until ${formatTime(next.plan.end, { timeZone })}`;
  const minutes = Math.ceil((start - now) / 60000);
  return `${done} — ${minutes < 60 ? `next in ${minutes} min` : `next at ${formatTime(next.plan.start, { timeZone })}`}`;
}

/** Planned length in minutes (0 for a session without a plan). */
export function sessionMinutes(session: SchedulerSession): number {
  if (!session.plan) return 0;
  return Math.max(0, Math.round((Date.parse(session.plan.end) - Date.parse(session.plan.start)) / 60000));
}

/** "45 MIN" under an hour, else hours to the quarter: "1 H", "1.5 H", "2.25 H". */
export function durationLabel(minutes: number): string {
  if (minutes < 60) return `${minutes} MIN`;
  return `${Math.round(minutes / 15) / 4} H`;
}

/** Hours between two instants for the day window, to the half hour: "15.5". */
export function windowHours(start: string, end: string): string {
  return String(Math.round((Date.parse(end) - Date.parse(start)) / 1800000) / 2);
}

/** ISO 8601 week number of a "YYYY-MM-DD" date (weeks start Monday; week 1 holds the first Thursday). */
export function isoWeek(date: string): number {
  const day = new Date(`${date}T12:00:00Z`);
  // Move to the Thursday of this week; its year is the ISO year.
  day.setUTCDate(day.getUTCDate() + 3 - (day.getUTCDay() + 6) % 7);
  const firstThursday = new Date(Date.UTC(day.getUTCFullYear(), 0, 4, 12));
  firstThursday.setUTCDate(firstThursday.getUTCDate() + 3 - (firstThursday.getUTCDay() + 6) % 7);
  return 1 + Math.round((day.getTime() - firstThursday.getTime()) / (7 * 86400000));
}

export const STREAK_TICKS = 60;

/**
 * The streak tick scale runs from 0 to the record (or the streak, if longer, or 1).
 * Above STREAK_TICKS ticks it is scaled down so the row stays readable.
 */
export function streakScale(streak: number, record: number): { count: number; filled: number; max: number } {
  const max = Math.max(1, streak, record);
  if (max <= STREAK_TICKS) return { count: max, filled: streak, max };
  return { count: STREAK_TICKS, filled: Math.round(streak / max * STREAK_TICKS), max };
}
