import { useCallback, useEffect, useRef, useState } from 'react';
import { Check } from 'lucide-react';
import { mondayOf, schedulerRequest, SchedulerError, earlierUnchecked, todaysTodos, weekSchema, zonedDate, type SchedulerSession, type SchedulerWeek } from '../lib/scheduler';
import { localDate } from '../lib/workspace';
import { isEnabled, loadSettings } from '../lib/modules';

type Today = { week: SchedulerWeek; today: string };

async function loadWeek(monday: string): Promise<SchedulerWeek> {
  return weekSchema.parse(await schedulerRequest(`week?week=${monday}`));
}
/** Loads the week holding today in the scheduler's time zone, which can differ from the browser's, or 'off' while the schedule module is off. */
// Module settings are read once per page load; the schedule cannot be turned off from Home.
let scheduleOn: Promise<boolean> | undefined;
async function loadToday(): Promise<Today | 'off'> {
  // If settings cannot load, try the schedule anyway; its own error explains a failure.
  scheduleOn ??= loadSettings().then(settings => isEnabled(settings, 'schedule'), () => true);
  if (!await scheduleOn) return 'off';
  let week = await loadWeek(mondayOf(localDate()));
  const today = zonedDate(new Date(), week.settings.timeZone);
  if (mondayOf(today) !== week.week) week = await loadWeek(mondayOf(today));
  return { week, today };
}

const isDone = (session: SchedulerSession) => Boolean(session.actual && session.actual.status !== 'skipped');

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

export function HomeHub() {
  const [data, setData] = useState<Today>();
  const [scheduleOff, setScheduleOff] = useState(false);
  const [error, setError] = useState('');
  const [pending, setPending] = useState<string>();
  // Earlier sessions checked off here stay listed so a mistaken click can be undone.
  const [touched, setTouched] = useState<Set<string>>(() => new Set());
  const latest = useRef(0);
  const reload = useCallback(() => {
    const request = ++latest.current;
    loadToday().then(value => {
      if (request !== latest.current) return;
      setScheduleOff(value === 'off'); setData(value === 'off' ? undefined : value); setError('');
    })
      .catch(e => { if (request === latest.current) setError(e instanceof Error ? e.message : 'Could not load today.'); });
  }, []);
  useEffect(() => {
    if (redirectLegacyTab()) return;
    reload();
    // Coming back to the tab picks up a new day and changes made in the scheduler.
    const onVisible = () => { if (document.visibilityState === 'visible') reload(); };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, [reload]);

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
      setData({ week, today: data.today });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save. Please retry.');
      reload();
    } finally { setPending(undefined); }
  }

  const todos = data ? todaysTodos(data.week, data.today) : [];
  const earlier = data ? earlierUnchecked(data.week, data.today, touched) : [];
  const doneCount = todos.filter(isDone).length;
  const zone = data?.week.settings.timeZone;
  const clock = (instant: string) => new Date(instant).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit', timeZone: zone });
  const heading = data ? new Date(`${data.today}T12:00:00Z`).toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', timeZone: 'UTC' }) : 'Today';
  const row = (session: SchedulerSession, showDay = false) => <TodoRow key={session.id} session={session} clock={clock} showDay={showDay} pending={pending} onToggle={() => void toggle(session)} />;

  return <div className="mx-auto max-w-2xl space-y-6">
    <header className="flex flex-wrap items-baseline justify-between gap-2">
      <div>
        <h1 className="text-3xl font-bold">{heading}</h1>
        {data && todos.length > 0 && <p className="text-zinc-400" aria-live="polite">{doneCount} of {todos.length} done</p>}
      </div>
      {!scheduleOff && <a className="text-sm text-blue-400 hover:underline" href="/weekly-scheduler">Plan in scheduler →</a>}
    </header>
    {error && <p role="alert" className="text-red-300">{error}</p>}
    {scheduleOff ? <p className="text-zinc-400">The schedule is turned off, so there are no sessions to show.</p>
      : !data ? !error && <p role="status" className="text-zinc-400">Loading today…</p>
      : todos.length === 0 ? <p className="text-zinc-400">Nothing scheduled today. <a className="text-blue-400 underline" href="/weekly-scheduler">Plan in the weekly scheduler.</a></p>
      : <ul className="divide-y divide-zinc-800 rounded-xl border border-zinc-800 bg-zinc-900" aria-label="Today's todos">{todos.map(session => row(session))}</ul>}
    {earlier.length > 0 && <section className="space-y-2" aria-labelledby="earlier-heading">
      <h2 id="earlier-heading" className="text-sm font-medium text-amber-300">Not checked off earlier this week</h2>
      <ul className="divide-y divide-zinc-800 rounded-xl border border-zinc-800 bg-zinc-900" aria-label="Earlier unchecked todos">{earlier.map(session => row(session, true))}</ul>
    </section>}
  </div>;
}

function TodoRow({ session, clock, showDay, pending, onToggle }: { session: SchedulerSession; clock: (instant: string) => string; showDay: boolean; pending?: string; onToggle: () => void }) {
  const done = isDone(session);
  // Explicit times were recorded in the scheduler; only checked-off sessions can be unchecked here.
  const recorded = session.actual?.status === 'explicit';
  const title = session.assignment.goalTitle && session.assignment.title !== session.assignment.goalTitle ? `${session.assignment.goalTitle} › ${session.assignment.title}` : session.assignment.title;
  const day = showDay ? `${new Date(`${session.date}T12:00:00Z`).toLocaleDateString('en-US', { weekday: 'short', timeZone: 'UTC' })} · ` : '';
  return <li>
    <label className={`flex ${recorded ? '' : 'cursor-pointer'} items-center gap-4 px-4 py-3 hover:bg-zinc-800/50 ${pending === session.id ? 'opacity-60' : ''}`}>
      <input type="checkbox" className="peer sr-only" checked={done} disabled={pending !== undefined || recorded} onChange={onToggle} />
      <span aria-hidden className={`flex size-6 shrink-0 items-center justify-center rounded-md border-2 peer-focus-visible:ring-2 peer-focus-visible:ring-blue-500 ${done ? 'border-transparent' : 'border-zinc-600'}`} style={done ? { background: session.assignment.color } : { borderColor: session.assignment.color }}>
        {done && <Check className="size-4 text-zinc-950" strokeWidth={3} />}
      </span>
      <span className="min-w-0 flex-1">
        <span className={`block ${done ? 'text-zinc-500 line-through' : 'text-zinc-100'}`}>{title}</span>
        <span className="block text-sm text-zinc-500">{day}{clock(session.plan!.start)} – {clock(session.plan!.end)}{session.actual?.status === 'skipped' ? ' · skipped' : recorded ? ' · recorded in scheduler' : ''}</span>
      </span>
    </label>
  </li>;
}
