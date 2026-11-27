import { useEffect, useRef, useState, type CSSProperties, type MouseEvent, type PointerEvent } from 'react';
import { dayNumber, dayString, fitRange, goalProgress, goalState, goalTag, monthLabel, monthName, monthOffset, shiftGoal, timelineCounts, timelineHours, type Goal } from '../lib/timeline';
import { localDate } from '../lib/workspace';
import { formatHours, goalPace, type GoalHours } from '../lib/goal-progress';
import { formatDate, formatTime } from '../lib/format';
import { mondayOf } from '../lib/scheduler';
import { cn } from '../lib/utils';
import '../styles/timeline.css';
import { dependencyCycle } from '../lib/goal-dependencies';
import { GoalDependencies } from './GoalDependencies';
import { GoalSchedule } from './GoalSchedule';
import { Button, HatchFill, Label, MeasurementLine, SegmentedTabs, TodayMarker } from './ui';

const RANGE_TABS = [
  { value: 'fit', label: 'Fit' }, { value: '6', label: '6M' }, { value: '12', label: '1Y' },
  { value: '24', label: '2Y' }, { value: '60', label: '5Y' }, { value: '120', label: '10Y' },
] as const;
// New goals still get a color: the Weekly Scheduler draws goals with it. The timeline itself is ink only.
const colors = ['#67e8f9', '#c4b5fd', '#fda4af', '#86efac', '#fcd34d', '#93c5fd', '#fdba74'];
const pad = (value: number) => String(value).padStart(2, '0');
// "Nov 24, 2026" for a YYYY-MM-DD date.
const longDate = (date: string) => `${formatDate(date)}, ${date.slice(0, 4)}`;
// "Nov 24, 2026 · 5:20 PM" for an instant, in the browser's zone.
function stamp(instant: string) {
  try {
    return `${formatDate(instant)}, ${new Date(instant).getFullYear()} · ${formatTime(instant)}`;
  } catch {
    return instant;
  }
}
const lineState = (goal: Goal, today: string) => goalState(goal, today) === 'late' ? 'late' : goal.status === 'planned' || goal.status === 'dropped' ? 'plan' : 'default';
const stepsText = (goal: Goal, today: string) => goal.steps.length ? `${goal.steps.filter(s => s.done).length} / ${goal.steps.length} steps` : goalState(goal, today);

function Steps({ goal, className }: { goal: Goal; className?: string }) {
  return <ol className={cn('tl-steps', className)}>
    {goal.steps.map((step, index) => <li key={step.id} className={step.done ? 'is-done' : undefined}>
      <Label>{pad(index + 1)}</Label><span>{step.title}{step.done && <span className="sr-only"> (done)</span>}</span>
    </li>)}
    {!goal.steps.length && <li className="tl-steps__empty"><Label>No steps yet</Label></li>}
  </ol>;
}

export function GoalTimeline() {
  const [goals, setGoals] = useState<Goal[]>([]);
  const [revisions, setRevisions] = useState<Record<string, string>>({});
  const [logged, setLogged] = useState<Record<string, GoalHours>>({});
  const [loaded, setLoaded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('Loading goals…');
  // null = fit the axis to the goals' date range.
  const [view, setView] = useState<{ anchor: string; months: number } | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [draft, setDraft] = useState<Goal | null>(null);
  const [creating, setCreating] = useState(false);
  const [preview, setPreview] = useState<Goal | null>(null);
  const [note, setNote] = useState('');
  const [step, setStep] = useState('');
  const [metaKey, setMetaKey] = useState('');
  const [metaValue, setMetaValue] = useState('');
  const dialog = useRef<HTMLDialogElement>(null);
  const axis = useRef<HTMLDivElement>(null);
  const drag = useRef<{ goal: Goal; x: number; width: number; mode: 'move' | 'start' | 'end'; next: Goal; moved: boolean } | null>(null);
  const suppressClick = useRef(false);
  const saving = useRef(false);
  const todayDate = localDate();
  const thisMonth = todayDate.slice(0, 7) + '-01';
  const { anchor, months } = view ?? fitRange(goals, thisMonth);
  const start = dayNumber(anchor), end = dayNumber(monthOffset(anchor, months)), days = end - start;
  const today = dayNumber(todayDate);
  const pace = (goal: Goal) => goalPace(goal, logged[goal.id], todayDate, mondayOf(todayDate));
  const frac = (day: number) => (day - start) / days;
  const pct = (day: number) => `${frac(day) * 100}%`;
  async function load() {
    setBusy(true); setError('');
    try {
      const [response, hoursResponse] = await Promise.all([fetch('/api/goals', { cache: 'no-store' }), fetch('/api/scheduler/goal-hours', { cache: 'no-store' })]);
      const result = await response.json();
      if (!response.ok) throw new Error(result.error);
      // Logged hours are supplementary; the timeline stays editable without them.
      if (hoursResponse.ok) setLogged((await hoursResponse.json()).goals ?? {});
      setGoals(result.goals); setRevisions(result.revisions); setLoaded(true); setStatus('All changes saved');
      const selected = new URLSearchParams(location.search).get('goal');
      if (selected && !draft) { const goal = result.goals.find((g: Goal) => g.id === selected); if (goal) { setSelectedId(goal.id); setExpandedId(goal.id); open(goal); } history.replaceState(null, '', location.pathname); }
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not load goals.'); }
    finally { setBusy(false); }
  }
  // Every goal comes back in one response, so changing the visible range never needs a reload.
  useEffect(() => { void load(); }, []);
  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => { if (draft || saving.current) event.preventDefault(); };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [draft]);
  useEffect(() => { if (draft) dialog.current?.showModal(); else dialog.current?.close(); }, [!!draft]);
  function open(goal: Goal, isNew = false) {
    setDraft(structuredClone(goal)); setCreating(isNew); setNote(''); setStep(''); setMetaKey(''); setMetaValue('');
  }
  function create(date: string) {
    if (!loaded || busy) return;
    const now = new Date().toISOString();
    open({ id: crypto.randomUUID(), title: `Goal ${goals.length + 1}`, startDate: date,
      endDate: monthOffset(date, 3), dailyHours: 1, color: colors[Math.floor(Math.random() * colors.length)],
      status: 'planned', dependsOn: [], createdAt: now, updatedAt: now, notes: [], metadata: {}, steps: [] }, true);
  }
  async function save(goal: Goal) {
    if (saving.current) return;
    if (dependencyCycle(goals, goal)) { setError('Dependency cycle: choose different prerequisites.'); return; }
    saving.current = true; setBusy(true); setError(''); setStatus('Saving…');
    try {
      const response = await fetch('/api/goals', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal, revision: revisions[goal.id] ?? null }) });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error);
      setGoals(current => [...current.filter(item => item.id !== result.goal.id), result.goal]);
      setRevisions(current => ({ ...current, [result.goal.id]: result.revision })); setDraft(null); setStatus('All changes saved');
      setSelectedId(result.goal.id);
    } catch (e) {
      setDraft(goal); setNote(''); setStep(''); setMetaKey(''); setMetaValue(''); setCreating(false); setError(e instanceof Error ? e.message : 'Could not save. Please retry.'); setStatus('Unsaved changes');
    } finally { saving.current = false; setBusy(false); setPreview(null); }
  }
  async function remove(goal: Goal) {
    if (saving.current || !window.confirm(`Delete ${goal.title} and its notes and mini goals?${goals.filter(g => g.dependsOn.includes(goal.id)).length ? " Dependents will lose this prerequisite: " + goals.filter(g => g.dependsOn.includes(goal.id)).map(g => g.title).join(", ") : ""}`)) return;
    saving.current = true; setBusy(true); setError('');
    try {
      const response = await fetch('/api/goals', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: goal.id, revision: revisions[goal.id] }) });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error);
      setGoals(current => current.filter(item => item.id !== goal.id)); setRevisions(current => { const next = { ...current }; delete next[goal.id]; return next; }); setDraft(null); setStatus('Goal deleted'); await load();
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not delete goal.'); }
    finally { saving.current = false; setBusy(false); }
  }
  function pointerDown(event: PointerEvent<HTMLElement>, goal: Goal, mode: 'move' | 'start' | 'end') {
    if (busy || event.button !== 0) return;
    event.stopPropagation();
    event.currentTarget.setPointerCapture(event.pointerId);
    suppressClick.current = false;
    drag.current = { goal, mode, x: event.clientX, width: event.currentTarget.closest('[data-track]')!.getBoundingClientRect().width, next: goal, moved: false };
  }
  function pointerMove(event: PointerEvent<HTMLElement>) {
    const current = drag.current;
    if (!current) return;
    if (Math.abs(event.clientX - current.x) < 4 && !current.moved) return;
    current.moved = true;
    current.next = shiftGoal(current.goal, Math.round((event.clientX - current.x) / current.width * days), current.mode);
    setPreview(current.next);
  }
  function pointerUp() {
    const current = drag.current; drag.current = null;
    if (!current) return;
    suppressClick.current = current.moved;
    if (current.moved && (current.next.startDate !== current.goal.startDate || current.next.endDate !== current.goal.endDate)) void save(current.next);
    else setPreview(null);
  }
  const cancelDrag = () => { drag.current = null; setPreview(null); };
  // Click selects; a click that ended a drag does nothing.
  const pick = (goal: Goal) => { if (!suppressClick.current) setSelectedId(goal.id); suppressClick.current = false; };
  function createAt(event: MouseEvent<HTMLElement>) {
    const target = event.target as HTMLElement;
    if (target.closest('[data-goal], .tl-name') || !axis.current) return;
    const rect = axis.current.getBoundingClientRect();
    if (event.clientX < rect.left) return;
    create(dayString(start + Math.min(days - 1, Math.max(0, Math.floor((event.clientX - rect.left) / rect.width * days)))));
  }

  const visible = goals.filter(g => dayNumber(g.endDate) >= start && dayNumber(g.startDate) < end).sort((a, b) => a.startDate.localeCompare(b.startDate) || a.id.localeCompare(b.id));
  const counts = timelineCounts(visible, todayDate);
  const hours = timelineHours(goals.map(goal => preview?.id === goal.id ? preview : goal), start, end);
  const tickEvery = months <= 12 ? 1 : months <= 24 ? 2 : months <= 60 ? 6 : 12;
  const ticks = Array.from({ length: Math.ceil(months / tickEvery) }, (_, i) => monthOffset(anchor, i * tickEvery));
  const tickLabel = (tick: string) => tickEvery >= 6 ? `${monthName(tick)} ’${tick.slice(2, 4)}` : monthName(tick);
  const tickLetter = (tick: string) => tickEvery >= 6 ? `’${tick.slice(2, 4)}` : monthName(tick)[0];
  const todayInView = today >= start && today < end;
  const selected = visible.find(g => g.id === selectedId) ?? visible.find(g => goalState(g, todayDate) === 'active') ?? visible[0];
  const empty = loaded ? 'Space for your next goal. Choose New goal to plan a 3-month block.' : 'Loading your timeline…';
  // Where a goal's span sits on the axis (clipped to the view), in percent.
  function spanOf(goal: Goal) {
    const left = Math.max(start, dayNumber(goal.startDate)), right = Math.min(end, dayNumber(goal.endDate) + 1);
    return { left: (left - start) / days * 100, width: Math.max(0, right - left) / days * 100 };
  }
  // The tag sits after the span, or before it when the span ends near the right edge.
  const tagStyle = (left: number, width: number): CSSProperties => left + width > 86 ? { right: `calc(${100 - left}% + 10px)` } : { left: `calc(${left + width}% + 10px)` };

  return <div className="goal-timeline">
    <header className="tl-header">
      <div className="tl-header__title">
        <Label>{monthLabel(anchor)} – {monthLabel(dayString(end - 1))}</Label>
        <h1 className="m-0 font-extralight !text-[44px] !leading-[0.9] md:!text-[58px]">Goal timeline</h1>
      </div>
      <dl className="tl-counts" aria-label="Goals in view">
        {([['Active', counts.active], ['Planned', counts.planned], ['Done', counts.done], ['Late', counts.late]] as const).map(([name, count]) =>
          <div key={name} className={name === 'Late' ? 'text-accent' : undefined}><Label as="dt" className={name === 'Late' ? 'text-accent' : undefined}>{name}</Label><dd className="font-mono">{pad(count)}</dd></div>)}
      </dl>
    </header>

    <div className="tl-toolbar">
      <SegmentedTabs className="tl-ranges" label="Timeline span" tabs={RANGE_TABS} value={view ? String(view.months) : 'fit'}
        onChange={value => setView(value === 'fit' ? null : { anchor, months: Number(value) })} />
      <div className="tl-toolbar__actions">
        <Button size="sm" aria-label="Previous period" disabled={anchor <= '1900-01-01'} onClick={() => setView({ anchor: monthOffset(anchor, -months), months })}>←</Button>
        <Button size="sm" onClick={() => setView({ anchor: thisMonth, months })}>Today</Button>
        <Button size="sm" aria-label="Next period" disabled={monthOffset(anchor, months * 2) > '2201-01-01'} onClick={() => setView({ anchor: monthOffset(anchor, months), months })}>→</Button>
        <Button size="sm" variant="primary" className="tl-new" disabled={!loaded || busy} onClick={() => create(todayInView ? thisMonth : anchor)}>+ New goal</Button>
      </div>
    </div>
    <p role="status" className="tl-meta ui-label">{status} · {visible.length} of {goals.length} goals in view</p>
    {loaded && <p className="tl-meta ui-label">
      {hours.average.toLocaleString('en-US', { maximumFractionDigits: 2 })} expected h/day on average · {hours.total.toLocaleString('en-US', { maximumFractionDigits: 2 })} h on selected weekdays over {days} days
      {hours.unknown > 0 && <> · {hours.unknown} {hours.unknown === 1 ? 'goal has' : 'goals have'} no hour estimate, so these totals are incomplete</>}
    </p>}
    {error && <div role="alert" className="tl-error">{error} <Button size="sm" disabled={busy} onClick={() => void load()}>Load latest</Button></div>}

    <div className="tl-desktop">
      <div className="tl-head">
        <Label>Goal</Label>
        <div className="tl-axis" ref={axis}>{ticks.map(tick => <Label key={tick} className="tl-tick" style={{ left: pct(dayNumber(tick)) }}>{tickLabel(tick)}</Label>)}</div>
      </div>
      <div className="tl-body" onDoubleClick={createAt}>
        <div className="tl-guides" aria-hidden="true">{ticks.map(tick => <span key={tick} style={{ left: pct(dayNumber(tick)) }} />)}</div>
        {todayInView && visible.length > 0 && <TodayMarker className="tl-today" style={{ left: `calc(var(--tl-name) + (100% - var(--tl-name)) * ${frac(today)})` }} />}
        {preview && <div className="tl-guide" style={{ left: `calc(var(--tl-name) + (100% - var(--tl-name)) * ${frac(Math.max(start, Math.min(end - 1, dayNumber(preview.startDate))))})` }}>
          <span className={frac(dayNumber(preview.startDate)) > .75 ? 'tl-guide--left' : undefined}>Start: {longDate(preview.startDate)}</span>
        </div>}
        {!visible.length && <p className="tl-empty">{empty}</p>}
        {visible.map(saved => {
          const goal = preview?.id === saved.id ? preview : saved;
          const state = goalState(goal, todayDate), tag = goalTag(goal, todayDate);
          const { left, width } = spanOf(goal);
          const isSelected = selected?.id === goal.id;
          const handlers = { onPointerMove: pointerMove, onPointerUp: pointerUp, onPointerCancel: cancelDrag, disabled: busy };
          return <div key={goal.id} className={cn('tl-row', `tl-row--${state}`, isSelected && 'is-selected')}>
            <button type="button" className="tl-name" aria-pressed={isSelected} onClick={() => setSelectedId(goal.id)} onDoubleClick={() => open(goal)}>{goal.title}</button>
            <div className="tl-track" data-track>
              <div data-goal className="tl-span" style={{ left: `${left}%`, width: `${width}%` }}>
                <HatchFill className="tl-fill" style={{ width: `${goalProgress(goal) * 100}%` }} />
                <MeasurementLine className="tl-line" state={lineState(goal, todayDate)} />
                <button type="button" className="tl-hit" aria-label={`${goal.title}: ${longDate(goal.startDate)} – ${longDate(goal.endDate)}${tag ? `, ${tag}` : ''}`} title={`${goal.title}: ${longDate(goal.startDate)} – ${longDate(goal.endDate)} · double-click to edit`}
                  {...handlers} onPointerDown={e => pointerDown(e, goal, 'move')} onClick={() => pick(goal)} onDoubleClick={() => open(goal)} />
                <button type="button" className="tl-handle tl-handle--start" aria-label={`Resize start of ${goal.title}`} {...handlers} onPointerDown={e => pointerDown(e, goal, 'start')} onClick={() => pick(goal)} />
                <button type="button" className="tl-handle tl-handle--end" aria-label={`Resize end of ${goal.title}`} {...handlers} onPointerDown={e => pointerDown(e, goal, 'end')} onClick={() => pick(goal)} />
              </div>
              {tag && <span className={cn('tl-tag', state === 'late' && 'text-accent')} style={tagStyle(left, width)} aria-hidden="true">{tag}</span>}
            </div>
          </div>;
        })}
      </div>
      <p className="tl-help ui-label">Drag a span to move it, drag its ends to change dates, double-click a goal to edit it or empty space to add one.</p>
      {selected && <section className="tl-panel" aria-label="Selected goal" aria-live="polite">
        <div className="tl-panel__head">
          <h2 className="tl-panel__name">{selected.title}</h2>
          <Label>{longDate(selected.startDate)} – {longDate(selected.endDate)} · {stepsText(selected, todayDate)}</Label>
          <Button size="sm" className="tl-panel__edit" disabled={busy} onClick={() => open(selected)}>Edit goal</Button>
        </div>
        <Steps goal={selected} className="tl-steps--columns" />
      </section>}
    </div>

    <div className="tl-mobile">
      <div className="tl-maxis" aria-hidden="true">{ticks.map(tick => <Label key={tick} className="tl-tick" style={{ left: pct(dayNumber(tick)) }}>{tickLetter(tick)}</Label>)}</div>
      <div className="tl-mlist">
        {todayInView && visible.length > 0 && <TodayMarker label="" className="tl-mtoday" style={{ left: pct(today) }} />}
        {!visible.length && <p className="tl-empty">{empty}</p>}
        {visible.map(goal => {
          const state = goalState(goal, todayDate), isOpen = expandedId === goal.id;
          const { left, width } = spanOf(goal);
          return <div key={goal.id} className={cn('tl-mrow', `tl-row--${state}`, isOpen && 'is-selected')}>
            <button type="button" className="tl-mbtn" aria-expanded={isOpen} onClick={() => setExpandedId(isOpen ? null : goal.id)}>
              <span className="tl-mtop"><span className="tl-name">{goal.title}</span><span className={cn('tl-tag', state === 'late' && 'text-accent')}>{goalTag(goal, todayDate) || goal.status}</span></span>
              <span className="tl-mspan" aria-hidden="true"><span className="tl-span" style={{ left: `${left}%`, width: `max(${width}%, 4px)` }}>
                <HatchFill className="tl-fill" style={{ width: `${goalProgress(goal) * 100}%` }} />
                <MeasurementLine className="tl-line" state={lineState(goal, todayDate)} />
              </span></span>
            </button>
            {isOpen && <div className="tl-mdetail">
              <Label>{longDate(goal.startDate)} – {longDate(goal.endDate)} · {stepsText(goal, todayDate)}</Label>
              <Steps goal={goal} />
              <Button size="sm" disabled={busy} onClick={() => open(goal)}>Edit goal</Button>
            </div>}
          </div>;
        })}
      </div>
    </div>

    <details className="tl-all">
      <summary className="ui-label">All goals ({goals.length})</summary>
      <ul>{[...goals].sort((a, b) => a.startDate.localeCompare(b.startDate) || a.id.localeCompare(b.id)).map(goal => <li key={goal.id}>
        <button type="button" disabled={busy} onClick={() => open(goal)}>
          <span>{goal.title}</span>
          <Label>{longDate(goal.startDate)} – {longDate(goal.endDate)} · {goalState(goal, todayDate)} · {formatHours(pace(goal).logged)}h logged</Label>
        </button>
      </li>)}</ul>
    </details>

    <dialog ref={dialog} className="goal-dialog" aria-label={creating ? 'New goal' : 'Goal details'} onCancel={event => { event.preventDefault(); if (!busy) setDraft(null); }}>
      {draft && <form onSubmit={event => {
        event.preventDefault();
        const now = new Date().toISOString();
        void save({ ...draft,
          notes: note.trim() ? [{ id: crypto.randomUUID(), body: note.trim(), createdAt: now }, ...draft.notes] : draft.notes,
          steps: step.trim() ? [...draft.steps, { id: crypto.randomUUID(), title: step.trim(), done: false }] : draft.steps,
          metadata: metaKey.trim() ? { ...draft.metadata, [metaKey.trim()]: metaValue } : draft.metadata,
        });
      }}>
        <h2>{creating ? 'New goal' : 'Goal details'}</h2>
        {creating && <p className="timeline-help">Choose dates and daily hours, then check how this goal fits your schedule.</p>}
        {error && <div role="alert" className="tl-error">{error}<Button size="sm" disabled={busy} onClick={() => void load()}>Load latest (keep draft)</Button></div>}
        <fieldset disabled={busy}>
          <label>Name<input required maxLength={200} value={draft.title} onChange={e => setDraft({ ...draft, title: e.target.value })} /></label>
          <GoalDependencies draft={draft} goals={goals} onChange={setDraft} />
          <GoalSchedule draft={draft} goals={goals} onChange={setDraft} />
          {!creating && <>
            <h3>Logged work</h3>
            {(() => { const p = pace(draft); return <p className="timeline-help">{formatHours(p.logged)}h logged{p.plannedTotal !== undefined ? ` of ${formatHours(p.plannedTotal)}h planned` : ''}{p.expectedToDate ? ` · ${formatHours(p.expectedToDate)}h expected by today` : ''} · {formatHours(p.thisWeek)}h this week. <a href="/">Log work or a reflection</a> · <a href="/weekly-scheduler">Plan the week</a></p>; })()}
            <h3>Notes</h3>
            <label>Add a note<textarea maxLength={20000} value={note} onChange={e => setNote(e.target.value)} placeholder="What changed? What did you learn?" /></label>
            {[...draft.notes].sort((a, b) => b.createdAt.localeCompare(a.createdAt)).map(n => <article className="goal-note" key={n.id}><time dateTime={n.createdAt}>{stamp(n.createdAt)}</time><p>{n.body}</p></article>)}
            {!draft.notes.length && <p className="timeline-help">No notes yet.</p>}
            <h3>Mini goals</h3>
            {draft.steps.map(s => <label className="goal-step" key={s.id}><input type="checkbox" checked={s.done} onChange={e => setDraft({ ...draft, steps: draft.steps.map(item => item.id === s.id ? { ...item, done: e.target.checked } : item) })} /><span>{s.title}</span><button type="button" aria-label={`Remove ${s.title}`} onClick={() => setDraft({ ...draft, steps: draft.steps.filter(item => item.id !== s.id) })}>×</button></label>)}
            <label>New mini goal<input maxLength={500} value={step} onChange={e => setStep(e.target.value)} /></label>
            <button type="button" disabled={!step.trim()} onClick={() => { setDraft({ ...draft, steps: [...draft.steps, { id: crypto.randomUUID(), title: step.trim(), done: false }] }); setStep(''); }}>Add mini goal</button>
            <h3>Metadata</h3>
            {Object.entries(draft.metadata).map(([key, value]) => <label key={key}>{key}<input maxLength={2000} value={value} onChange={e => setDraft({ ...draft, metadata: { ...draft.metadata, [key]: e.target.value } })} /></label>)}
            <div className="goal-dates"><label>New field<input maxLength={80} placeholder="e.g. Focus area" value={metaKey} onChange={e => setMetaKey(e.target.value)} /></label><label>Value<input maxLength={2000} value={metaValue} onChange={e => setMetaValue(e.target.value)} /></label></div>
            <button type="button" disabled={!metaKey.trim()} onClick={() => { setDraft({ ...draft, metadata: { ...draft.metadata, [metaKey.trim()]: metaValue } }); setMetaKey(''); setMetaValue(''); }}>Add metadata</button>
            <p className="timeline-help">Last updated {stamp(draft.updatedAt)}</p>
          </>}
        </fieldset>
        <div className="goal-dialog__actions">
          {!creating && <Button variant="danger" disabled={busy} onClick={() => void remove(draft)}>Delete goal</Button>}
          <Button disabled={busy} onClick={() => setDraft(null)}>Cancel</Button>
          <Button type="submit" variant="primary" disabled={busy}>{busy ? 'Saving…' : creating ? 'Create goal' : 'Save goal'}</Button>
        </div>
      </form>}
    </dialog>
  </div>;
}
