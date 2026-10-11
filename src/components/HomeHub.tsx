import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react';
import { CalendarDays, ChartGantt, ChevronLeft, ChevronRight, ListFilter, NotebookPen, Target } from 'lucide-react';
import { DocumentView, LogsView } from './StaticDataViews';
import { WorkspaceEditor, appendToWorkJournal } from './WorkspaceEditor';
import { buildLogDocs, type LogDoc } from '../lib/logs';
import { currentGoals, formatHours, goalNoteBody, goalPace, journalLine, logSlot, runSaveSteps, upcomingGoals, weekTotals, type GoalHours, type SaveStep } from '../lib/goal-progress';
import { addDays, localInstant, mondayOf, schedulerRequest, SchedulerError, weekSchema, zonedDate, type SchedulerWeek } from '../lib/scheduler';
import type { Goal } from '../lib/timeline';
import { localDate, newWeek, type JournalWeek } from '../lib/workspace';

const field = 'mt-1 w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500';
const button = 'rounded-lg border border-zinc-700 px-3 py-2 text-sm hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed';
const primary = `${button} bg-blue-600 border-blue-500 text-white hover:bg-blue-500`;
const card = 'rounded-xl border border-zinc-800 bg-zinc-900 p-5';

const TABS = [
  { id: 'overview', label: 'Overview', icon: Target },
  { id: 'week', label: 'Week', icon: CalendarDays },
  { id: 'journal', label: 'Journal', icon: NotebookPen },
  { id: 'logs', label: 'Logs', icon: ListFilter },
] as const;
type Tab = (typeof TABS)[number]['id'];
const isTab = (value: string | null): value is Tab => TABS.some(tab => tab.id === value);

async function getJSON<T>(url: string): Promise<T> {
  const response = await fetch(url, { cache: 'no-store' });
  if (response.status === 401) {
    location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`);
    throw new Error('Your session expired. Sign in again.');
  }
  const value = await response.json();
  if (!response.ok) throw Object.assign(new Error(value.error || 'Request failed. Please retry.'), { status: response.status });
  return value;
}
async function loadWeek(monday: string): Promise<SchedulerWeek> {
  return weekSchema.parse(await schedulerRequest(`week?week=${monday}`));
}

/** Runs `load` now and after any workspace save, keeping only the latest result. */
function useLive<T>(load: () => Promise<T>, deps: unknown[]) {
  const [data, setData] = useState<T>();
  const [error, setError] = useState('');
  const latest = useRef(0);
  const reload = useCallback(() => {
    const request = ++latest.current;
    load().then(value => { if (request === latest.current) { setData(value); setError(''); } })
      .catch(e => { if (request === latest.current) setError(e instanceof Error ? e.message : 'Could not load this section.'); });
  }, deps);
  useEffect(() => {
    reload();
    window.addEventListener('workspace-saved', reload);
    return () => window.removeEventListener('workspace-saved', reload);
  }, [reload]);
  return { data, error, reload };
}
const loadGoals = async () => {
  const [saved, recorded] = await Promise.all([
    getJSON<{ goals: Goal[] }>('/api/goals'),
    // Logged hours are supplementary; goals still show without them, as on the timeline.
    getJSON<{ goals: Record<string, GoalHours> }>('/api/scheduler/goal-hours').catch(() => ({ goals: {} })),
  ]);
  return { goals: saved.goals, hours: recorded.goals };
};

function Bar({ value, max, color, label }: { value: number; max: number; color: string; label: string }) {
  const pct = max > 0 ? Math.min(100, value / max * 100) : value > 0 ? 100 : 0;
  return <div role="meter" aria-label={label} aria-valuemin={0} aria-valuemax={max || 1} aria-valuenow={value} className="h-2 w-full overflow-hidden rounded-full bg-zinc-800">
    <div className="h-full rounded-full" style={{ width: `${pct}%`, background: color }} />
  </div>;
}

function GoalCard({ goal, hours, today, weekStart }: { goal: Goal; hours?: GoalHours; today: string; weekStart: string }) {
  const pace = goalPace(goal, hours, today, weekStart);
  const latestNote = [...goal.notes].sort((a, b) => b.createdAt.localeCompare(a.createdAt))[0];
  const behind = pace.expectedToDate !== undefined ? pace.expectedToDate - pace.logged : 0;
  return <article className="rounded-lg border border-zinc-800 border-l-4 bg-zinc-950/40 p-4 space-y-3" style={{ borderLeftColor: goal.color }}>
    <div className="flex flex-wrap items-start justify-between gap-2">
      <div className="min-w-0">
        <h3 className="font-semibold text-zinc-100">{goal.title}</h3>
        <p className="text-xs text-zinc-500">{goal.status} · {goal.startDate} → {goal.endDate}{goal.dailyHours !== undefined ? ` · ${formatHours(goal.dailyHours)} h/day` : ''}</p>
      </div>
      <a className="shrink-0 text-sm text-blue-400 hover:underline" href={`/timeline?goal=${encodeURIComponent(goal.id)}`}>Timeline →</a>
    </div>
    <div className="space-y-1">
      <div className="flex justify-between text-sm"><span>{formatHours(pace.logged)}h logged{pace.plannedTotal !== undefined ? ` of ${formatHours(pace.plannedTotal)}h planned` : ''}</span><span className="text-zinc-400">{formatHours(pace.thisWeek)}h this week</span></div>
      <Bar value={pace.logged} max={pace.plannedTotal ?? 0} color={goal.color} label={`${goal.title} hours logged`} />
      {pace.expectedToDate !== undefined && pace.expectedToDate > 0 && <p className={`text-xs ${behind > 0.25 ? 'text-amber-300' : 'text-emerald-300'}`}>{behind > 0.25 ? `${formatHours(behind)}h behind` : 'On pace'} · {formatHours(pace.expectedToDate)}h expected by today</p>}
    </div>
    {pace.stepsTotal > 0 && <p className="text-sm text-zinc-400">{pace.stepsDone}/{pace.stepsTotal} mini goals done{goal.steps.find(step => !step.done) ? ` · next: ${goal.steps.find(step => !step.done)!.title}` : ''}</p>}
    {latestNote && <p className="line-clamp-2 text-sm text-zinc-400 border-l-2 border-zinc-700 pl-2">{latestNote.body}</p>}
  </article>;
}

function WeekGoalRows({ week }: { week: SchedulerWeek }) {
  if (!week.goals.length) return <p className="text-sm text-zinc-400">No goal work planned or recorded this week.</p>;
  return <ul className="space-y-3">{week.goals.map(item => {
    const target = item.requiredHours ?? 0;
    return <li key={item.goal.id} className="space-y-1">
      <div className="flex flex-wrap justify-between gap-2 text-sm">
        <span className="font-medium" style={{ color: item.goal.color }}>{item.goal.title}</span>
        <span className="text-zinc-400">{formatHours(item.actualHours)}h done · {formatHours(item.remainingScheduledHours)}h planned{item.requiredHours !== null ? ` · ${formatHours(target)}h target` : ''}</span>
      </div>
      <Bar value={item.actualHours} max={target} color={item.goal.color} label={`${item.goal.title} hours this week`} />
      {item.uncoveredHours > 0 && <p className="text-xs text-amber-300">{formatHours(item.uncoveredHours)}h not planned yet</p>}
    </li>;
  })}</ul>;
}

function WeekSummary({ week, onOpenWeek }: { week?: SchedulerWeek; onOpenWeek: () => void }) {
  if (!week) return <section className={card}><p role="status" className="text-zinc-400">Loading this week…</p></section>;
  const totals = weekTotals(week);
  return <section className={`${card} space-y-4`} aria-labelledby="week-summary-heading">
    <div className="flex flex-wrap items-baseline justify-between gap-2">
      <h2 id="week-summary-heading" className="text-xl font-semibold">This week · {week.week} → {addDays(week.week, 6)}</h2>
      <div className="flex gap-3 text-sm"><button type="button" className="text-blue-400 hover:underline" onClick={onOpenWeek}>Week view →</button><a className="text-blue-400 hover:underline" href="/weekly-scheduler">Plan in scheduler →</a></div>
    </div>
    <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">{[[totals.actual, 'Logged'], [totals.scheduled, 'Still planned'], [totals.required, 'Goal target'], [totals.uncovered, 'Unplanned']].map(([value, label]) =>
      <div key={label} className="rounded-lg bg-zinc-800/60 p-3"><dt className="text-xs text-zinc-400">{label}</dt><dd className="text-lg font-semibold">{formatHours(value as number)}h</dd></div>)}</dl>
    <WeekGoalRows week={week} />
  </section>;
}

function LogWorkForm({ goals, today, timeZone, onSaved }: { goals: Goal[]; today: string; timeZone?: string; onSaved: () => void }) {
  const current = currentGoals(goals, today);
  const others = goals.filter(goal => goal.status !== 'done' && goal.status !== 'dropped' && !current.includes(goal)).sort((a, b) => a.startDate.localeCompare(b.startDate));
  const [goalId, setGoalId] = useState('');
  const [stepId, setStepId] = useState('');
  const [completeStep, setCompleteStep] = useState(false);
  const [date, setDate] = useState(today);
  const [hours, setHours] = useState('1');
  const [startTime, setStartTime] = useState('');
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');
  // Steps an earlier, partly failed attempt saved. While any exist, what is being logged is locked so a retry finishes the same entry.
  const completed = useRef(new Map<string, string>());
  const [partial, setPartial] = useState(false);
  useEffect(() => { if (!goalId && current[0]) setGoalId(current[0].id); }, [current[0]?.id]);
  const goal = goals.find(item => item.id === goalId);
  const step = goal?.steps.find(item => item.id === stepId);
  const amount = goal ? Number(hours) || 0 : 0;

  async function recordHours(target: Goal) {
    const minutes = Math.round(amount * 60);
    let week = await loadWeek(mondayOf(date));
    for (let attempt = 0; ; attempt++) {
      const zone = week.settings.timeZone;
      let interval: { start: string; end: string } | null;
      if (startTime) {
        const start = localInstant(date, startTime, zone);
        interval = { start, end: new Date(Date.parse(start) + minutes * 60000).toISOString() };
      } else interval = logSlot(week, date, new Date(), minutes);
      if (!interval) throw new Error(`No free ${formatHours(amount)}h gap on ${date} before now. Enter a start time.`);
      try {
        await schedulerRequest('mutate', {
          action: 'actual', revision: week.revision, week: week.week,
          session: { id: '', date, assignment: { goalId: target.id, stepId: step?.id, title: step?.title ?? target.title, goalTitle: target.title, color: target.color }, plan: null, actual: null, state: 'accepted', exception: false, conflictIds: [] },
          actual: { status: 'explicit', date, start: interval.start, end: interval.end },
        });
        return;
      } catch (e) {
        // A stale revision returns the current week; retry once with it.
        if (attempt === 0 && e instanceof SchedulerError && e.current && e.current.revision !== week.revision && !e.conflictIds.length) { week = e.current; continue; }
        throw e;
      }
    }
  }
  async function saveGoalNote(target: Goal) {
    for (let attempt = 0; ; attempt++) {
      const saved = await getJSON<{ goals: Goal[]; revisions: Record<string, string> }>('/api/goals');
      const latest = saved.goals.find(item => item.id === target.id);
      if (!latest) throw new Error(`${target.title} is no longer on the timeline.`);
      const body = note.trim() ? goalNoteBody({ date, stepTitle: step?.title, hours: amount, note }) : '';
      const next: Goal = {
        ...latest,
        notes: body ? [{ id: crypto.randomUUID(), body, createdAt: new Date().toISOString() }, ...latest.notes] : latest.notes,
        steps: completeStep && step ? latest.steps.map(item => item.id === step.id ? { ...item, done: true } : item) : latest.steps,
      };
      const response = await fetch('/api/goals', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal: next, revision: saved.revisions[target.id] ?? null }) });
      const result = await response.json();
      if (response.ok) return;
      if (response.status === 409 && attempt === 0) continue;
      throw new Error(result.error || 'Could not save the goal note.');
    }
  }
  async function submit() {
    setBusy(true); setError(''); setStatus('');
    const done = completed.current;
    try {
      const steps: SaveStep[] = [];
      if (goal && amount > 0) steps.push({ key: 'hours', label: `${formatHours(amount)}h logged to ${goal.title}`, run: () => recordHours(goal) });
      if (goal && (note.trim() || (completeStep && step))) steps.push({ key: 'goal', label: note.trim() ? 'note added to the goal' : 'mini goal completed', run: () => saveGoalNote(goal) });
      const line = journalLine({ goalTitle: goal?.title, stepTitle: step?.title, hours: amount, note });
      if (line) steps.push({ key: 'journal', label: 'journal updated', run: () => appendToWorkJournal(date, line) });
      await runSaveSteps(steps, done);
      setStatus(`Saved: ${[...done.values()].join(', ')}.`);
      resetForm();
      onSaved();
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Could not save.';
      setError(done.size ? `${message} Already saved: ${[...done.values()].join(', ')}. Saving again finishes the rest.` : message);
      if (done.size) { setPartial(true); onSaved(); }
    } finally { setBusy(false); }
  }
  function resetForm() {
    completed.current = new Map(); setPartial(false);
    setNote(''); setStartTime(''); setCompleteStep(false); setStepId('');
  }
  const canSave = goal ? amount > 0 || Boolean(note.trim()) || (completeStep && Boolean(step)) : Boolean(note.trim());
  return <section className="rounded-xl border border-blue-900/60 bg-zinc-900 p-5 space-y-4" aria-labelledby="reflect-heading">
    <div>
      <h2 id="reflect-heading" className="text-xl font-semibold">A moment to reflect</h2>
      <p className="mt-1 text-sm text-zinc-400">Log time and notes against a timeline goal. Hours go to the weekly scheduler, notes to the goal and this week's journal.</p>
    </div>
    <form className="space-y-3" onSubmit={e => { e.preventDefault(); void submit(); }}>
      <fieldset disabled={busy} className="space-y-3">
        <fieldset disabled={partial} className="space-y-3" aria-describedby={partial ? 'partial-help' : undefined}>
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="block text-sm">Goal<select className={field} value={goalId} onChange={e => { setGoalId(e.target.value); setStepId(''); setCompleteStep(false); }}>
            <option value="">No goal (journal only)</option>
            {current.length > 0 && <optgroup label="Current goals">{current.map(item => <option key={item.id} value={item.id}>{item.title}</option>)}</optgroup>}
            {others.length > 0 && <optgroup label="Other open goals">{others.map(item => <option key={item.id} value={item.id}>{item.title}</option>)}</optgroup>}
          </select></label>
          {goal && goal.steps.some(item => !item.done) && <label className="block text-sm">Mini goal (optional)<select className={field} value={stepId} onChange={e => { setStepId(e.target.value); setCompleteStep(false); }}>
            <option value="">Whole goal</option>
            {goal.steps.filter(item => !item.done).map(item => <option key={item.id} value={item.id}>{item.title}</option>)}
          </select></label>}
        </div>
        <div className="grid gap-3 grid-cols-2 sm:grid-cols-3">
          <label className="block text-sm">Date<input className={field} type="date" required min="2026-04-20" max={today} value={date} onChange={e => setDate(e.target.value)} /></label>
          {goal && <label className="block text-sm">Hours<input className={field} type="number" min="0" max="16" step="0.25" value={hours} onChange={e => setHours(e.target.value)} /></label>}
          {goal && amount > 0 && <label className="block text-sm">Start time<input className={field} type="time" value={startTime} onChange={e => setStartTime(e.target.value)} aria-describedby="start-help" /><span id="start-help" className="text-xs text-zinc-500">Blank: latest free slot{timeZone ? ` (${timeZone})` : ''}</span></label>}
        </div>
        {step && <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={completeStep} onChange={e => setCompleteStep(e.target.checked)} />Mark “{step.title}” done</label>}
        </fieldset>
        <label className="block text-sm">What did you learn, finish, or get stuck on?<textarea aria-label="Reflection" className={`${field} min-h-28`} maxLength={20000} placeholder="Today I learned…" value={note} onChange={e => { setNote(e.target.value); setStatus(''); }} /></label>
        <div className="flex flex-wrap items-center gap-3">
          <button className={primary} type="submit" disabled={!canSave}>{busy ? 'Saving…' : goal && amount > 0 ? 'Log work' : 'Save reflection'}</button>
          {partial && <button type="button" className={button} onClick={() => { if (window.confirm('Start a new entry? Steps already saved stay saved; the rest of this entry is dropped.')) resetForm(); }}>Start over</button>}
          <a className="text-sm text-blue-400 hover:underline" href="/?tab=journal">Open journal →</a>
        </div>
      </fieldset>
    </form>
    {error && <p role="alert" className="text-sm text-red-300">{error}</p>}
    {partial && <p id="partial-help" className="text-sm text-zinc-400">Goal, date and hours are locked until this entry finishes saving, so nothing is logged twice.</p>}
    <p role="status" className="text-sm text-zinc-400">{status}</p>
  </section>;
}

function OverviewTab({ goals, hours, week, today, error, onSaved, onOpenWeek }: { goals?: Goal[]; hours: Record<string, GoalHours>; week?: SchedulerWeek; today: string; error: string; onSaved: () => void; onOpenWeek: () => void }) {
  const current = goals ? currentGoals(goals, today) : [];
  const upcoming = goals ? upcomingGoals(goals, today) : [];
  const weekStart = mondayOf(today);
  return <div className="space-y-6">
    {error && <p role="alert" className="text-red-300">{error}</p>}
    <section className={`${card} space-y-4`} aria-labelledby="current-goals-heading">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="current-goals-heading" className="text-xl font-semibold">Current goals</h2>
        <a className="inline-flex items-center gap-1 text-sm text-blue-400 hover:underline" href="/timeline"><ChartGantt className="size-4" aria-hidden />Timeline →</a>
      </div>
      {!goals ? <p role="status" className="text-zinc-400">Loading goals…</p> : current.length === 0 ? <p className="text-zinc-400">No goals cover today. <a className="text-blue-400 underline" href="/timeline">Plan one on the timeline.</a></p>
        : <div className="grid gap-3 md:grid-cols-2">{current.map(goal => <GoalCard key={goal.id} goal={goal} hours={hours[goal.id]} today={today} weekStart={weekStart} />)}</div>}
      {upcoming.length > 0 && <div className="space-y-1"><h3 className="text-sm font-medium text-zinc-400">Up next</h3><ul className="text-sm">{upcoming.map(goal => <li key={goal.id}><span style={{ color: goal.color }}>●</span> {goal.title} <span className="text-zinc-500">· starts {goal.startDate}</span></li>)}</ul></div>}
    </section>
    <WeekSummary week={week} onOpenWeek={onOpenWeek} />
    {goals && <LogWorkForm goals={goals} today={today} timeZone={week?.settings.timeZone} onSaved={onSaved} />}
    <details className={card}>
      <summary className="cursor-pointer text-lg font-semibold">Career strategy & study plan</summary>
      <div className="mt-4 space-y-4">
        <DocumentView id="index" home />
        <nav className="grid gap-3 sm:grid-cols-2" aria-label="Career resources">{[['/2026/career-study-plan', 'Study plan'], ['/books', 'Bookshelf'], ['/notes', 'Topic notes'], ['/companies', 'Application targets']].map(([href, label]) =>
          <a key={href} className="rounded-lg border border-zinc-800 p-4 text-blue-400 hover:border-blue-400" href={href}>{label} →</a>)}</nav>
      </div>
    </details>
  </div>;
}

function WeekTab({ weekStart, setWeekStart, today }: { weekStart: string; setWeekStart: (week: string) => void; today: string }) {
  const { data: week, error } = useLive(() => loadWeek(weekStart), [weekStart]);
  const [docs, setDocs] = useState<LogDoc[]>([]);
  useEffect(() => {
    let live = true;
    const load = () => getJSON<{ entry: JournalWeek }>(`/api/entries/week?id=${encodeURIComponent(newWeek(weekStart).slug)}`)
      .then(result => { if (live) setDocs(buildLogDocs([result.entry])); })
      .catch(e => { if (live) setDocs([]); if ((e as { status?: number }).status !== 404) console.warn(e); });
    void load();
    window.addEventListener('workspace-saved', load);
    return () => { live = false; window.removeEventListener('workspace-saved', load); };
  }, [weekStart]);
  const days = Array.from({ length: 7 }, (_, i) => addDays(weekStart, i));
  const label = (date: string) => new Date(`${date}T12:00:00Z`).toLocaleDateString('en-US', { weekday: 'long', month: 'short', day: 'numeric', timeZone: 'UTC' });
  return <div className="space-y-6">
    <div className="flex flex-wrap items-center gap-2">
      <button type="button" className={button} aria-label="Previous week" onClick={() => setWeekStart(addDays(weekStart, -7))}><ChevronLeft className="size-4" aria-hidden /></button>
      <button type="button" className={button} onClick={() => setWeekStart(mondayOf(today))}>This week</button>
      <button type="button" className={button} aria-label="Next week" onClick={() => setWeekStart(addDays(weekStart, 7))}><ChevronRight className="size-4" aria-hidden /></button>
      <span className="text-sm text-zinc-400">{weekStart} → {addDays(weekStart, 6)}</span>
      <a className="ml-auto text-sm text-blue-400 hover:underline" href="/weekly-scheduler">Plan in scheduler →</a>
    </div>
    {error && <p role="alert" className="text-red-300">{error}</p>}
    <section className={`${card} space-y-4`} aria-labelledby="week-goals-heading">
      <h2 id="week-goals-heading" className="text-xl font-semibold">Goal hours</h2>
      {week ? <WeekGoalRows week={week} /> : <p role="status" className="text-zinc-400">Loading…</p>}
    </section>
    <section className="space-y-3" aria-labelledby="week-days-heading">
      <h2 id="week-days-heading" className="text-xl font-semibold">Day by day</h2>
      {days.map((date, dayIndex) => {
        const sessions = (week?.sessions ?? []).filter(session => session.actual && session.actual.status !== 'skipped' && session.actual.date === date);
        // Journal weeks and scheduler weeks both start on Monday, so weekday position identifies the day.
        const entries = docs.filter(doc => doc.section === 'day' && doc.dayIndex === dayIndex);
        const total = sessions.reduce((sum, session) => sum + (Date.parse(session.actual!.end) - Date.parse(session.actual!.start)) / 3600000, 0);
        return <article key={date} className={`${card} space-y-2 ${date === today ? 'border-blue-800' : ''}`}>
          <h3 className="flex justify-between font-medium"><span>{label(date)}</span><span className="text-sm text-zinc-400">{formatHours(total)}h</span></h3>
          {sessions.length > 0 && <ul className="flex flex-wrap gap-2 text-xs">{sessions.map(session => <li key={session.id} className="rounded-full border border-zinc-700 px-2 py-0.5" style={{ borderColor: session.assignment.color }}>{session.assignment.goalTitle ?? session.assignment.title}{session.assignment.goalTitle && session.assignment.title !== session.assignment.goalTitle ? ` › ${session.assignment.title}` : ''} · {formatHours((Date.parse(session.actual!.end) - Date.parse(session.actual!.start)) / 3600000)}h</li>)}</ul>}
          {entries.flatMap(doc => doc.parts).map((part, i) => <p key={i} className="text-sm text-zinc-300">{part.label && <span className="font-semibold text-zinc-400">{part.label}: </span>}{part.text}</p>)}
          {!sessions.length && !entries.length && <p className="text-sm text-zinc-500">Nothing logged.</p>}
        </article>;
      })}
    </section>
  </div>;
}

function JournalTab() {
  return <div className="space-y-4">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p className="text-zinc-400">Small steps, written down. Revisit a week or edit its notes.</p>
      <a href="/journal/new" className="rounded-lg border border-blue-500 bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-500">+ Add entry</a>
    </div>
    <p className="text-sm"><a className="text-blue-400 underline" href="/personal-journal">Personal journal</a> · <a className="text-blue-400 underline" href="/api/agent-context?format=markdown&journal=work">Read work journal as Markdown</a></p>
    <WorkspaceEditor kind="week" />
  </div>;
}

export function HomeHub() {
  const [tab, setTabState] = useState<Tab>(() => { const value = new URLSearchParams(location.search).get('tab'); return isTab(value) ? value : 'overview'; });
  const [visited, setVisited] = useState<Set<Tab>>(() => new Set([tab]));
  const [timeZone, setTimeZone] = useState<string>();
  const today = timeZone ? zonedDate(new Date(), timeZone) : localDate();
  const [weekStart, setWeekStart] = useState(() => mondayOf(localDate()));
  const goals = useLive(loadGoals, []);
  const week = useLive(() => loadWeek(mondayOf(today)), [today]);
  useEffect(() => { if (week.data && week.data.settings.timeZone !== timeZone) setTimeZone(week.data.settings.timeZone); }, [week.data]);
  const reloadOverview = () => { goals.reload(); week.reload(); };
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  function setTab(next: Tab, focus = false) {
    setTabState(next); setVisited(current => new Set(current).add(next));
    const url = new URL(location.href);
    if (next === 'overview') url.searchParams.delete('tab'); else url.searchParams.set('tab', next);
    if (next !== 'journal') url.searchParams.delete('id');
    history.replaceState(history.state, '', url);
    if (focus) tabRefs.current[next]?.focus();
  }
  function onKey(event: KeyboardEvent) {
    const index = TABS.findIndex(item => item.id === tab);
    const move = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
    if (event.key === 'Home') { event.preventDefault(); setTab(TABS[0].id, true); }
    else if (event.key === 'End') { event.preventDefault(); setTab(TABS[TABS.length - 1].id, true); }
    else if (move) { event.preventDefault(); setTab(TABS[(index + move + TABS.length) % TABS.length].id, true); }
  }
  return <div className="space-y-6">
    <header className="space-y-1">
      <h1 className="text-3xl font-bold">Home</h1>
      <p className="text-zinc-400">Your timeline goals, this week's progress, and the journal behind them.</p>
    </header>
    <div role="tablist" aria-label="Home sections" className="flex gap-1 overflow-x-auto border-b border-zinc-800" onKeyDown={onKey}>
      {TABS.map(item => <button key={item.id} ref={node => { tabRefs.current[item.id] = node; }} role="tab" id={`tab-${item.id}`} aria-controls={`panel-${item.id}`} aria-selected={tab === item.id} tabIndex={tab === item.id ? 0 : -1}
        className={`inline-flex shrink-0 items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === item.id ? 'border-blue-500 text-zinc-100' : 'border-transparent text-zinc-400 hover:text-zinc-200'}`} onClick={() => setTab(item.id)}>
        <item.icon className="size-4" aria-hidden />{item.label}
      </button>)}
    </div>
    {TABS.map(item => visited.has(item.id) && <div key={item.id} role="tabpanel" id={`panel-${item.id}`} aria-labelledby={`tab-${item.id}`} hidden={tab !== item.id}>
      {item.id === 'overview' && <OverviewTab goals={goals.data?.goals} hours={goals.data?.hours ?? {}} week={week.data} today={today} error={goals.error || week.error} onSaved={reloadOverview} onOpenWeek={() => { setWeekStart(mondayOf(today)); setTab('week'); }} />}
      {item.id === 'week' && <WeekTab weekStart={weekStart} setWeekStart={setWeekStart} today={today} />}
      {item.id === 'journal' && <JournalTab />}
      {item.id === 'logs' && <LogsView />}
    </div>)}
  </div>;
}
