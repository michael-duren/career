import { useCallback, useEffect, useRef, useState } from 'react';
import { z } from 'zod';
import { mondayOf, schedulerRequest, SchedulerError, earlierUnchecked, todaysTodos, weekSchema, zonedDate, type SchedulerSession, type SchedulerWeek } from '../lib/scheduler';
import { localDate } from '../lib/workspace';
import { isEnabled, loadSettings, SETTINGS_CHANNEL, SETTINGS_EVENT, type ModuleId } from '../lib/modules';
import { formatNumericDate, formatTime } from '../lib/format';
import { durationLabel, isDone, isoWeek, isSkipped, nextSession, progressNote, sessionMinutes, streakScale, windowHours } from '../lib/home';
import type { Goal } from '../lib/timeline';
import { Box, Checkbox, Label, MeasurementLine, Progress, SkippedIcon, TickScale } from './ui';

type Modules = Record<'schedule' | 'timeline' | 'leetgrinder', boolean>;
type Today = { week: SchedulerWeek; today: string };

const leetSchema = z.object({
  met: z.boolean(), remaining: z.number(), streak: z.number(), record: z.number().default(0), dueCount: z.number(),
  goal: z.object({ new: z.number(), review: z.number() }),
  done: z.object({ new: z.number(), review: z.number() }),
  picks: z.array(z.object({ slug: z.string(), title: z.string(), difficulty: z.string().optional(), recall: z.number(), reason: z.string(), done: z.boolean(), number: z.number().optional() })),
});
type Leet = z.infer<typeof leetSchema>;

/** Section data: loaded, failed with a message, or still loading (undefined). */
type Part<T> = { value: T } | { error: string } | undefined;

async function loadWeek(monday: string): Promise<SchedulerWeek> {
  return weekSchema.parse(await schedulerRequest(`week?week=${monday}`));
}
/** Loads the week holding today in the scheduler's time zone, which can differ from the browser's. */
async function loadToday(): Promise<Today> {
  let week = await loadWeek(mondayOf(localDate()));
  const today = zonedDate(new Date(), week.settings.timeZone);
  if (mondayOf(today) !== week.week) week = await loadWeek(mondayOf(today));
  return { week, today };
}
async function getJSON(url: string, fallback: string): Promise<unknown> {
  const response = await fetch(url, { cache: 'no-store' }).catch(() => { throw new Error(fallback); });
  const body: unknown = await response.json().catch(() => null);
  if (!response.ok) throw new Error(body && typeof body === 'object' && 'error' in body && typeof body.error === 'string' ? body.error : fallback);
  return body;
}
async function loadGoals(): Promise<Goal[]> {
  const body = await getJSON('/api/goals', 'Could not load goals.') as { goals?: Goal[] } | null;
  return (body?.goals ?? []).filter(goal => goal.status === 'active');
}
async function loadLeet(): Promise<Leet> {
  const parsed = leetSchema.safeParse(await getJSON('/api/leetgrinder/today', 'Could not load Leetgrinder.'));
  if (!parsed.success) throw new Error('Could not load Leetgrinder.');
  return parsed.data;
}
/** Which Home sections are on. If settings cannot load, every section tries and reports its own failure. */
async function loadModules(): Promise<Modules> {
  const settings = await loadSettings().catch(error => { console.warn('Module settings could not be loaded; showing every Home section.', error); return undefined; });
  const on = (id: ModuleId) => !settings || isEnabled(settings, id);
  return { schedule: on('schedule'), timeline: on('timeline'), leetgrinder: on('leetgrinder') };
}
const message = (e: unknown, fallback: string) => e instanceof Error ? e.message : fallback;
const settle = <T,>(promise: Promise<T>, fallback: string): Promise<Part<T>> => promise.then(value => ({ value }), e => ({ error: message(e, fallback) }));

// Old home tabs moved to the journal page.
function redirectLegacyTab() {
  const params = new URLSearchParams(location.search);
  const tab = params.get('tab');
  if (tab !== 'journal' && tab !== 'logs') return false;
  params.delete('tab');
  if (tab === 'logs') params.set('tab', 'logs');
  const query = params.toString();
  location.replace(`/journal${query ? `?${query}` : ''}`);
  return true;
}

const WEEKDAYS = ['SUNDAY', 'MONDAY', 'TUESDAY', 'WEDNESDAY', 'THURSDAY', 'FRIDAY', 'SATURDAY'];
const weekdayOf = (date: string) => WEEKDAYS[new Date(`${date}T12:00:00Z`).getUTCDay()];
const heading = 'm-0 text-[11px] font-medium tracking-[0.24em] md:text-xs';
const link = 'text-ink underline decoration-ink-30 underline-offset-4 hover:text-ink-hover';

export function HomeHub() {
  const [modules, setModules] = useState<Modules>();
  const [schedule, setSchedule] = useState<Part<Today>>();
  const [goals, setGoals] = useState<Part<Goal[]>>();
  const [leet, setLeet] = useState<Part<Leet>>();
  const [error, setError] = useState('');
  const [pending, setPending] = useState<string>();
  const [now, setNow] = useState(() => Date.now());
  // Earlier sessions checked off here stay listed so a mistaken click can be undone.
  const [touched, setTouched] = useState<Set<string>>(() => new Set());
  const latest = useRef(0);
  const reload = useCallback(() => {
    const request = ++latest.current;
    const current = () => request === latest.current;
    // Read on every reload, so a module changed in another tab shows when Home is visible again.
    void loadModules().then(on => {
      if (!current()) return;
      setModules(on); setNow(Date.now());
      if (on.schedule) void settle(loadToday(), 'Could not load today.').then(part => { if (current()) setSchedule(part); }); else setSchedule(undefined);
      if (on.timeline) void settle(loadGoals(), 'Could not load goals.').then(part => { if (current()) setGoals(part); }); else setGoals(undefined);
      if (on.leetgrinder) void settle(loadLeet(), 'Could not load Leetgrinder.').then(part => { if (current()) setLeet(part); }); else setLeet(undefined);
    });
  }, []);
  useEffect(() => {
    if (redirectLegacyTab()) return;
    reload();
    // Coming back to the tab picks up a new day and changes made in the scheduler.
    const onVisible = () => { if (document.visibilityState === 'visible') reload(); };
    document.addEventListener('visibilitychange', onVisible);
    // Modules switched in Settings (this tab or another) show without a reload.
    window.addEventListener(SETTINGS_EVENT, reload);
    const channel = typeof BroadcastChannel === 'function' ? new BroadcastChannel(SETTINGS_CHANNEL) : undefined;
    channel?.addEventListener('message', reload);
    // The next-session ring and note move as time passes.
    const tick = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => {
      document.removeEventListener('visibilitychange', onVisible);
      window.removeEventListener(SETTINGS_EVENT, reload);
      channel?.close();
      window.clearInterval(tick);
    };
  }, [reload]);

  const data = schedule && 'value' in schedule ? schedule.value : undefined;
  async function toggle(session: SchedulerSession) {
    if (!data) return;
    setPending(session.id); setError('');
    setTouched(current => new Set(current).add(session.id));
    const done = isDone(session);
    const body = (week: SchedulerWeek) => done
      ? { revision: week.revision, week: week.week, action: 'clearActual', id: session.id }
      : { revision: week.revision, week: week.week, action: 'actual', id: session.id, actual: { ...session.plan!, date: session.date, status: 'assumed' } };
    // Cancel any in-flight reload so it cannot overwrite the saved result.
    latest.current++;
    try {
      let week = data.week;
      for (let attempt = 0; ; attempt++) {
        try {
          week = weekSchema.parse(await schedulerRequest('mutate', body(week)));
          break;
        } catch (e) {
          // A stale revision returns the current week; retry once with it.
          if (attempt === 0 && e instanceof SchedulerError && e.current && e.current.revision !== week.revision && !e.conflictIds.length) { week = e.current; continue; }
          throw e;
        }
      }
      setSchedule({ value: { week, today: data.today } });
    } catch (e) {
      setError(message(e, 'Could not save. Please retry.'));
      reload();
    } finally { setPending(undefined); }
  }

  const zone = data?.week.settings.timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
  const today = data?.today ?? localDate();
  const todos = data ? todaysTodos(data.week, data.today) : [];
  const earlier = data ? earlierUnchecked(data.week, data.today, touched) : [];
  const next = nextSession(todos, now);
  const day = data?.week.days.find(d => d.date === data.today);
  const none = modules && !modules.schedule && !modules.timeline && !modules.leetgrinder;
  const left = modules && (modules.schedule || modules.timeline);

  return <div className="flex flex-col">
    <header className="grid grid-cols-[minmax(0,1fr)_auto] items-end gap-x-7 md:grid-cols-[auto_minmax(0,1fr)] lg:grid-cols-[auto_auto_minmax(0,1fr)]">
      <h1 className="m-0 font-extralight !text-[70px] !leading-[0.8] tracking-[-0.03em] md:!text-[128px]">
        <span aria-hidden="true">{formatNumericDate(today)}</span>
        <span className="ui-sr-only">Today, {new Date(`${today}T12:00:00Z`).toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', timeZone: 'UTC' })}</span>
      </h1>
      <div className="flex flex-col gap-1.5 pb-0.5 md:pb-1">
        <span aria-hidden="true" className="text-[13px] font-light tracking-[0.24em] md:text-[26px]"><span className="md:hidden">{weekdayOf(today).slice(0, 3)}</span><span className="hidden md:inline">{weekdayOf(today)}</span></span>
        <Label className="hidden md:inline">Week {isoWeek(today)} · {zone}</Label>
      </div>
      {todos.length > 0 && <p className="ui-note order-last col-span-2 m-0 mt-3.5 !text-[15px] md:order-none md:mt-4 md:!text-base lg:col-span-1 lg:mt-0 lg:justify-self-end lg:pb-2 lg:text-right" aria-live="polite">{progressNote(todos, now, zone)}</p>}
      {day && <div className="col-span-2 mt-4 flex items-center gap-2.5 md:mt-[18px] md:gap-3 lg:col-span-3">
        <MeasurementLine className="flex-1" />
        <Label className="text-[9px] md:text-[10px]">
          <span className="md:hidden">{formatTime(day.start, { timeZone: zone, compact: true })} — {formatTime(day.end, { timeZone: zone, compact: true })}</span>
          <span className="hidden md:inline">{formatTime(day.start, { timeZone: zone })} — {windowHours(day.start, day.end)} hours — {formatTime(day.end, { timeZone: zone })}</span>
        </Label>
        <MeasurementLine className="flex-1" />
      </div>}
    </header>
    {error && <p role="alert" className="mt-4 text-sm text-accent">{error}</p>}
    {!modules ? <p role="status" className="mt-8 text-ink-60">Loading today…</p>
      : none ? <p className="mt-8 text-ink-60">Nothing to show here: Schedule, Timeline and Leetgrinder are turned off. <a className={link} href="/settings/modules">Turn them on in Settings → Modules.</a></p>
      : <div className={`mt-[22px] grid gap-6 md:mt-[30px] md:gap-10 ${modules.leetgrinder && left ? 'xl:grid-cols-[minmax(0,680px)_minmax(300px,520px)] xl:gap-[58px]' : ''}`}>
        {left && <div className="flex min-w-0 flex-col gap-[22px] md:gap-[26px]">
          {modules.schedule && <section aria-labelledby="sessions-heading" className="flex flex-col">
            <div className="mb-1 flex items-center gap-2.5 md:mb-1.5">
              <h2 id="sessions-heading" className={heading}>GOAL SESSIONS</h2>
              <a href="/weekly-scheduler" className="ui-label ml-auto no-underline hover:!text-ink">Plan →</a>
            </div>
            {!schedule ? <p role="status" className="py-4 text-sm text-ink-60">Loading sessions…</p>
              : 'error' in schedule ? <p role="alert" className="py-4 text-sm text-accent">{schedule.error}</p>
              : todos.length === 0 ? <p className="py-4 text-ink-60">Nothing scheduled today. <a className={link} href="/weekly-scheduler">Plan in the scheduler.</a></p>
              : <ul className="m-0 list-none p-0" aria-label="Today's todos">{todos.map(session => <SessionRow key={session.id} session={session} zone={zone} next={session.id === next?.id} pending={pending} onToggle={() => void toggle(session)} />)}</ul>}
            {earlier.length > 0 && <ul className="m-0 list-none p-0" aria-label="Earlier unchecked todos">{earlier.map(session => <MissedRow key={session.id} session={session} zone={zone} pending={pending} onToggle={() => void toggle(session)} />)}</ul>}
          </section>}
          {modules.timeline && <GoalsSection goals={goals} today={today} />}
        </div>}
        {modules.leetgrinder && <LeetSection leet={leet} />}
      </div>}
  </div>;
}

function SessionRow({ session, zone, next, pending, onToggle }: { session: SchedulerSession; zone: string; next: boolean; pending?: string; onToggle: () => void }) {
  const done = isDone(session);
  const skipped = isSkipped(session);
  // Explicit times were recorded in the scheduler; only checked-off sessions can be unchecked here.
  const recorded = session.actual?.status === 'explicit';
  const { title, goalTitle } = session.assignment;
  const sub = [goalTitle && goalTitle !== title ? goalTitle : '', skipped ? 'skipped' : recorded ? 'recorded in scheduler' : ''].filter(Boolean).join(' · ');
  const minutes = sessionMinutes(session);
  return <li>
    <label className={`grid min-h-[68px] grid-cols-[34px_minmax(0,1fr)] items-center gap-3.5 border-b border-ink-14 py-2 md:min-h-16 md:grid-cols-[30px_76px_110px_minmax(0,1fr)_70px] md:gap-[18px] ${recorded || pending !== undefined ? '' : 'cursor-pointer'} ${done || skipped ? 'opacity-45' : ''} ${pending === session.id ? 'animate-pulse' : ''}`}>
      <span className="flex justify-center md:justify-start"><Checkbox size={18} className="md:!size-4" next={next} checked={done} disabled={pending !== undefined || recorded} onChange={onToggle} /></span>
      {/* Phone: time, duration line and title stacked; desktop: one grid column each. */}
      <span className="flex min-w-0 flex-col gap-[5px] md:contents">
        <span className="flex items-center gap-2.5 md:contents">
          <span className="font-mono text-xs md:text-[13px]">{formatTime(session.plan!.start, { timeZone: zone })}</span>
          <span className="flex items-center gap-2.5 md:flex-col md:items-start md:gap-1.5">
            <Label className="order-last text-[9px] md:order-none">{durationLabel(minutes)}</Label>
            <MeasurementLine className="max-md:!hidden" style={{ width: Math.min(110, minutes) }} />
            <MeasurementLine className="md:!hidden" style={{ width: Math.min(120, Math.round(minutes * 2 / 3)) }} />
          </span>
        </span>
        <span className="flex min-w-0 flex-col">
          <span className={`leading-tight ${next ? 'text-lg font-normal md:text-[21px]' : 'text-base font-light md:text-lg'} ${done || skipped ? 'line-through decoration-1' : ''}`}>{title}</span>
          {sub && <Label className="tracking-[0.1em]">{sub}</Label>}
        </span>
      </span>
      <span className="hidden text-right md:block">
        {next ? <span className="ui-note !text-[15px]">next →</span> : <Label>{skipped ? 'SKIPPED' : done ? 'DONE' : ''}</Label>}
      </span>
    </label>
  </li>;
}

function MissedRow({ session, zone, pending, onToggle }: { session: SchedulerSession; zone: string; pending?: string; onToggle: () => void }) {
  const done = isDone(session);
  const { title, goalTitle } = session.assignment;
  const weekday = new Date(`${session.date}T12:00:00Z`).toLocaleDateString('en-US', { weekday: 'short', timeZone: 'UTC' });
  const state = done ? 'done' : 'missed';
  return <li>
    <label className={`grid min-h-11 grid-cols-[34px_minmax(0,1fr)] items-center gap-3.5 py-2.5 text-sm md:min-h-10 md:grid-cols-[30px_76px_110px_minmax(0,1fr)] md:gap-[18px] md:py-1 ${done ? 'opacity-45' : 'text-accent'} ${pending !== undefined ? '' : 'cursor-pointer'}`}>
      <span className="flex justify-center md:justify-start"><Checkbox checked={done} disabled={pending !== undefined} onChange={onToggle} /></span>
      <span className="flex min-w-0 flex-col md:contents">
        <span className="font-mono text-[11px] uppercase md:text-xs md:normal-case"><span className="md:hidden">{weekday} </span>{formatTime(session.plan!.start, { timeZone: zone })}<span className="md:hidden"> · {state}</span></span>
        <span className="ui-label hidden items-center gap-1.5 !text-current md:flex"><SkippedIcon size={12} />{weekday} · {state}</span>
        <span className={`font-light ${done ? 'line-through' : ''}`}>{goalTitle && goalTitle !== title ? `${goalTitle} › ${title}` : title}</span>
      </span>
    </label>
  </li>;
}

function GoalsSection({ goals, today }: { goals: Part<Goal[]>; today: string }) {
  return <section aria-labelledby="goals-heading" className="flex flex-col">
    <div className="mb-1 flex items-center gap-2.5 md:mb-2.5">
      <h2 id="goals-heading" className={heading}>ACTIVE GOALS</h2>
      <a href="/timeline" className="ui-label ml-auto no-underline hover:!text-ink">Timeline →</a>
    </div>
    {!goals ? <p role="status" className="py-2 text-sm text-ink-60">Loading goals…</p>
      : 'error' in goals ? <p role="alert" className="py-2 text-sm text-accent">{goals.error}</p>
      : goals.value.length === 0 ? <p className="py-2 text-ink-60">No active goals. <a className={link} href="/timeline">Plan one on the timeline.</a></p>
      : <ul className="m-0 list-none p-0" aria-label="Active goals">{goals.value.map(goal => {
        const steps = goal.steps ?? [];
        const done = steps.filter(step => step.done).length;
        const late = goal.endDate < today;
        const label = late ? 'LATE' : steps.length ? `${done} / ${steps.length}` : 'NO STEPS';
        return <li key={goal.id} className="grid grid-cols-[minmax(0,1fr)_auto] items-baseline gap-x-5 gap-y-[7px] py-2 md:min-h-8 md:grid-cols-[200px_minmax(0,1fr)_90px] md:items-center md:py-0.5">
          <a href={`/timeline?goal=${encodeURIComponent(goal.id)}`} className="truncate text-[15px] font-light text-ink no-underline hover:text-ink-hover">{goal.title}</a>
          <span className={`text-right font-mono text-[11px] tracking-[0.08em] md:col-start-3 md:row-start-1 ${late ? 'text-accent' : 'text-ink-60'}`}>{label}</span>
          <Progress className="col-span-2 !h-3 md:col-span-1 md:col-start-2 md:row-start-1 md:!h-3.5" value={done} max={steps.length} late={late} label={`${goal.title}: ${steps.length ? `${done} of ${steps.length} steps` : 'no steps'}${late ? ', late' : ''}`} />
        </li>;
      })}</ul>}
  </section>;
}

function LeetSection({ leet }: { leet: Part<Leet> }) {
  const header = (meta?: string) => <div className="flex items-center gap-2.5">
    <h2 id="leet-heading" className={heading}><a href="/leetgrinder" className="text-ink no-underline hover:text-ink-hover">LEETGRINDER</a></h2>
    {meta && <Label className="ml-auto !text-ink">{meta}</Label>}
  </div>;
  if (!leet || 'error' in leet) return <aside aria-labelledby="leet-heading" className="flex flex-col gap-2.5">
    {header()}
    {leet ? <p role="alert" className="text-sm text-accent">{leet.error}</p> : <p role="status" className="text-sm text-ink-60">Loading Leetgrinder…</p>}
  </aside>;
  const value = leet.value;
  const scale = streakScale(value.streak, value.record);
  const pick = value.picks.find(p => !p.done);
  const count = (figure: string, label: string, href?: string) => {
    const body = <><span className="font-mono text-[13px] md:text-lg">{figure}</span><Label className="text-[8px] md:text-[10px]">{label}</Label></>;
    return href ? <a href={href} className="flex flex-col text-ink no-underline hover:text-ink-hover">{body}</a> : <span className="flex flex-col">{body}</span>;
  };
  return <aside aria-labelledby="leet-heading" className="flex min-w-0 flex-col gap-5">
    <div className="flex flex-col gap-2.5">
      {header(value.met ? 'Goal met ✓' : `${value.remaining} to go`)}
      <div className="flex items-end gap-3.5 md:gap-4">
        <span className="text-[64px] leading-[0.78] font-extralight tracking-[-0.03em] md:text-[84px]">{String(value.streak).padStart(2, '0')}</span>
        <span className="flex flex-col gap-[3px] pb-0.5 md:gap-1">
          <span className="text-[13px] font-light tracking-[0.12em] md:text-[15px]">DAY STREAK</span>
          <Label className="text-[9px] md:text-[10px]">Record {value.record}</Label>
        </span>
      </div>
      <TickScale className="mt-1" count={scale.count} filled={scale.filled} label={`Streak ${value.streak} days, record ${value.record}`} />
      <div className="flex justify-between"><Label>0</Label><Label>Days</Label><Label>{scale.max}</Label></div>
      <div className="mt-0.5 grid grid-cols-3 border-t border-ink-30 pt-2.5">
        {count(`${value.done.new}/${value.goal.new}`, 'New')}
        {count(`${value.done.review}/${value.goal.review}`, 'Review')}
        {count(String(value.dueCount), 'Due', '/leetgrinder/reviews')}
      </div>
    </div>
    {pick && <Box as="section" aria-label="Today's review" className="mx-[3px] flex flex-col gap-1.5 px-4 py-3.5 md:mx-2.5 md:px-[18px] md:py-4">
      <Label className="text-[9px] md:text-[10px]">Review{pick.number ? ` · #${pick.number}` : ''}{pick.difficulty ? ` · ${pick.difficulty}` : ''}</Label>
      <span className="text-lg leading-tight font-light md:text-xl">{pick.title}</span>
      <span className="text-xs text-ink-60">Recall {Math.round(pick.recall * 100)}%{pick.reason ? ` · ${pick.reason}` : ''}</span>
      <a href={`/leetgrinder/problem/${encodeURIComponent(pick.slug)}`} className="mt-2 flex h-11 items-center gap-3 self-start text-xs font-medium tracking-[0.2em] text-ink no-underline hover:text-ink-hover">
        OPEN PROBLEM <span aria-hidden="true" className="inline-block w-10 border-t border-ink" />→
      </a>
    </Box>}
  </aside>;
}
