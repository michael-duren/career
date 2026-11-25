// Shared display formatting for the redesign (docs/redesign/design-system.md,
// Formatting). Times are 12-hour with no leading zero and a space before
// AM/PM; ranges use a spaced en dash. These are for on-screen labels only:
// never parse their output back into time values.

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

export type TimeOptions = {
  // compact drops ":00" on the hour ("9 AM") for tight places.
  compact?: boolean;
  // timeZone is used when formatting an instant; defaults to the browser zone.
  timeZone?: string;
};

// A time of day: minutes since midnight (wrapped into one day), or an instant.
export type TimeValue = number | string | Date;

function minutesOf(value: TimeValue, timeZone?: string): number {
  if (typeof value === 'number') {
    if (!Number.isFinite(value)) throw new Error('Invalid time.');
    return ((Math.trunc(value) % 1440) + 1440) % 1440;
  }
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) throw new Error('Invalid time.');
  const parts = new Intl.DateTimeFormat('en-US', { timeZone, hour: 'numeric', minute: 'numeric', hourCycle: 'h23' }).formatToParts(date);
  const part = (type: Intl.DateTimeFormatPartTypes) => Number(parts.find(p => p.type === type)?.value ?? 0);
  return (part('hour') % 24) * 60 + part('minute');
}

// formatTime renders "7:15 AM", "12:00 PM", or "9 AM" when compact.
export function formatTime(value: TimeValue, options: TimeOptions = {}): string {
  const minutes = minutesOf(value, options.timeZone);
  const hour24 = Math.floor(minutes / 60), minute = minutes % 60;
  const hour = hour24 % 12 || 12;
  const meridiem = hour24 < 12 ? 'AM' : 'PM';
  if (options.compact && minute === 0) return `${hour} ${meridiem}`;
  return `${hour}:${String(minute).padStart(2, '0')} ${meridiem}`;
}

// formatTimeRange renders "7:15 AM – 8:45 AM", or "9 AM – 4 PM" when compact.
export function formatTimeRange(start: TimeValue, end: TimeValue, options: TimeOptions = {}): string {
  return `${formatTime(start, options)} – ${formatTime(end, options)}`;
}

// A calendar date: "YYYY-MM-DD" (read as that date, no zone shift) or an
// instant (read in timeZone, default the browser zone). Any other string goes
// through new Date(), so one without an offset is read as runtime-local time.
export type DateValue = string | Date;

function dateFields(value: DateValue, timeZone?: string): { year: number; month: number; day: number; weekday: number } {
  if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(value)) {
    const [year, month, day] = value.split('-').map(Number);
    const date = new Date(Date.UTC(year, month - 1, day));
    if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) throw new Error('Invalid date.');
    return { year, month, day, weekday: date.getUTCDay() };
  }
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) throw new Error('Invalid date.');
  const parts = new Intl.DateTimeFormat('en-US', { timeZone, year: 'numeric', month: 'numeric', day: 'numeric' }).formatToParts(date);
  const part = (type: Intl.DateTimeFormatPartTypes) => Number(parts.find(p => p.type === type)?.value ?? 0);
  const year = part('year'), month = part('month'), day = part('day');
  return { year, month, day, weekday: new Date(Date.UTC(year, month - 1, day)).getUTCDay() };
}

// formatDate renders "Nov 24".
export function formatDate(value: DateValue, timeZone?: string): string {
  const { month, day } = dateFields(value, timeZone);
  return `${MONTHS[month - 1]} ${day}`;
}

// formatWeekdayDate renders "Tue 24 Nov".
export function formatWeekdayDate(value: DateValue, timeZone?: string): string {
  const { month, day, weekday } = dateFields(value, timeZone);
  return `${WEEKDAYS[weekday]} ${day} ${MONTHS[month - 1]}`;
}

// formatNumericDate renders "24.11.26"; only for the large Home date.
export function formatNumericDate(value: DateValue, timeZone?: string): string {
  const { year, month, day } = dateFields(value, timeZone);
  return `${String(day).padStart(2, '0')}.${String(month).padStart(2, '0')}.${String(year % 100).padStart(2, '0')}`;
}
