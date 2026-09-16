import { useEffect, useRef, useState, type PointerEvent } from 'react';
import { dayNumber, dayString, monthOffset, shiftGoal, timelineHours, type Goal } from '../lib/timeline';
import { localDate } from '../lib/workspace';
import '../styles/timeline.css';
import { GoalSchedule } from './GoalSchedule';

const ranges = [6, 12, 24, 60, 120];
const colors = ['#67e8f9', '#c4b5fd', '#fda4af', '#86efac', '#fcd34d', '#93c5fd', '#fdba74'];
export function GoalTimeline() {
  const [goals, setGoals] = useState<Goal[]>([]);
  const [revisions, setRevisions] = useState<Record<string, string>>({});
  const [loaded, setLoaded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('Loading goals…');
  const [months, setMonths] = useState(6);
  const [anchor, setAnchor] = useState(() => localDate().slice(0, 7) + '-01');
  const [draft, setDraft] = useState<Goal | null>(null);
  const [creating, setCreating] = useState(false);
  const [preview, setPreview] = useState<Goal | null>(null);
  const [note, setNote] = useState('');
  const [step, setStep] = useState('');
  const [metaKey, setMetaKey] = useState('');
  const [metaValue, setMetaValue] = useState('');
  const dialog = useRef<HTMLDialogElement>(null);
  const drag = useRef<{ goal: Goal; x: number; width: number; mode: 'move' | 'start' | 'end'; next: Goal; moved: boolean } | null>(null);
  const suppressClick = useRef(false);
  const saving = useRef(false);
  const start = dayNumber(anchor), end = dayNumber(monthOffset(anchor, months)), days = end - start;
  const today = dayNumber(localDate());
  const pct = (day: number) => `${(day - start) / days * 100}%`;
  async function load() {
    setBusy(true); setError('');
    try {
      const response = await fetch(`/api/goals?from=${anchor}&to=${dayString(end - 1)}`, { cache: 'no-store' });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error);
      setGoals(result.goals); setRevisions(result.revisions); setLoaded(true); setStatus('All changes saved');
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not load goals.'); }
    finally { setBusy(false); }
  }
  useEffect(() => { void load(); }, [anchor, months]);
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
      createdAt: now, updatedAt: now, notes: [], metadata: {}, steps: [] }, true);
  }
  async function save(goal: Goal) {
    if (saving.current) return;
    saving.current = true; setBusy(true); setError(''); setStatus('Saving…');
    try {
      const response = await fetch('/api/goals', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal, revision: revisions[goal.id] ?? null }) });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error);
      setGoals(current => [...current.filter(item => item.id !== result.goal.id), result.goal]);
      setRevisions(current => ({ ...current, [result.goal.id]: result.revision })); setDraft(null); setStatus('All changes saved');
    } catch (e) {
      setDraft(goal); setNote(''); setStep(''); setMetaKey(''); setMetaValue(''); setCreating(false); setError(e instanceof Error ? e.message : 'Could not save. Please retry.'); setStatus('Unsaved changes');
    } finally { saving.current = false; setBusy(false); setPreview(null); }
  }
  async function remove(goal: Goal) {
    if (saving.current || !window.confirm(`Delete ${goal.title} and its notes and substeps?`)) return;
    saving.current = true; setBusy(true); setError('');
    try {
      const response = await fetch('/api/goals', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: goal.id, revision: revisions[goal.id] }) });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error);
      setGoals(current => current.filter(item => item.id !== goal.id)); setRevisions(current => { const next = { ...current }; delete next[goal.id]; return next; }); setDraft(null); setStatus('Goal deleted');
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
  const visible = goals.filter(g => dayNumber(g.endDate) >= start && dayNumber(g.startDate) < end).sort((a, b) => a.startDate.localeCompare(b.startDate) || a.id.localeCompare(b.id));
  const hours = timelineHours(goals.map(goal => preview?.id === goal.id ? preview : goal), start, end);
  const tickEvery = months <= 24 ? 1 : months === 60 ? 6 : 12;
  const ticks = Array.from({ length: months / tickEvery }, (_, i) => monthOffset(anchor, i * tickEvery));
  return <div className="goal-timeline">
    <div className="timeline-toolbar">
      <div className="timeline-ranges" aria-label="Timeline span">{ranges.map(range => <button key={range} aria-pressed={months === range} onClick={() => setMonths(range)}>{range < 60 ? `${range} months` : `${range / 12} years`}</button>)}</div>
      <button disabled={!loaded || busy} onClick={() => create(anchor)}>+ New goal</button>
    </div>
    <div className="timeline-toolbar">
      <div className="timeline-ranges"><button aria-label="Previous period" disabled={anchor <= '1900-01-01'} onClick={() => setAnchor(monthOffset(anchor, -months))}>←</button><button onClick={() => setAnchor(localDate().slice(0, 7) + '-01')}>Today</button><button aria-label="Next period" disabled={monthOffset(anchor, months * 2) > '2201-01-01'} onClick={() => setAnchor(monthOffset(anchor, months))}>→</button></div>
      <span>{anchor} — {dayString(end - 1)}</span>
    </div>
    <p className="timeline-help">Choose New goal to plan a block. Tap a goal or select it from All goals below to edit its dates. You can also drag blocks to move them or drag their edges to resize.</p>
    <p role="status" className="timeline-help">{status} · {visible.length} of {goals.length} goals in view</p>
    {loaded && <div className="schedule-summary timeline-hours">
      <strong>{hours.average.toLocaleString('en-US', { maximumFractionDigits: 2 })} expected h/day on average</strong>
      <span>{hours.total.toLocaleString('en-US', { maximumFractionDigits: 2 })} total expected hours ÷ {days} calendar days in view, including days without goals.</span>
      {hours.unknown > 0 && <span>{hours.unknown} {hours.unknown === 1 ? 'goal has' : 'goals have'} no hour estimate; these totals are incomplete.</span>}
    </div>}
    {error && <div role="alert" className="timeline-error">{error} <button disabled={busy} onClick={() => void load()}>Load latest</button></div>}
    <div className="timeline-scroll">
      <div className="timeline-canvas" style={{ minWidth: months <= 12 ? 660 : 1000 }}>
        <div className="timeline-axis">{ticks.map(tick => <span key={tick} style={{ left: pct(dayNumber(tick)) }}>{new Date(`${tick}T00:00:00Z`).toLocaleDateString('en-US', { month: months < 120 ? 'short' : undefined, year: 'numeric', timeZone: 'UTC' })}</span>)}</div>
        <div data-track className="timeline-tracks" style={{ height: Math.max(240, visible.length * 72 + 80) }} onDoubleClick={event => {
          if ((event.target as HTMLElement).closest('[data-goal]')) return;
          const rect = event.currentTarget.getBoundingClientRect();
          create(dayString(start + Math.min(days - 1, Math.max(0, Math.floor((event.clientX - rect.left) / rect.width * days)))));
        }}>
          {ticks.map(tick => <div key={tick} className="timeline-gridline" style={{ left: pct(dayNumber(tick)) }} />)}
          {today >= start && today < end && <div className="timeline-today" style={{ left: pct(today) }}><span>Today</span></div>}
          {preview && <div className="timeline-drag-guide" style={{ left: pct(Math.max(start, Math.min(end - 1, dayNumber(preview.startDate)))) }}><span className={dayNumber(preview.startDate) > start + days * .75 ? 'guide-label-left' : ''}>Start: {preview.startDate}</span></div>}
          {!visible.length && <div className="timeline-empty">{loaded ? 'Space for your next goal. Choose New goal to plan a 3-month block.' : 'Loading your timeline…'}</div>}
          {visible.map((saved, index) => {
            const goal = preview?.id === saved.id ? preview : saved;
            const left = Math.max(start, dayNumber(goal.startDate)), right = Math.min(end, dayNumber(goal.endDate) + 1);
            return <div data-goal key={goal.id} className="timeline-goal" style={{ top: index * 72 + 30, left: pct(left), width: `${Math.max(0, right - left) / days * 100}%`, background: goal.color }}>
              <button className="goal-handle" aria-label={`Resize start of ${goal.title}`} disabled={busy} onPointerDown={e => pointerDown(e, goal, 'start')} onPointerMove={pointerMove} onPointerUp={pointerUp} onPointerCancel={() => { drag.current = null; setPreview(null); }} onClick={() => { if (!suppressClick.current) open(goal); }}>‖</button>
              <button className="goal-body" disabled={busy} title={`${goal.title}: ${goal.startDate} to ${goal.endDate}`} onPointerDown={e => pointerDown(e, goal, 'move')} onPointerMove={pointerMove} onPointerUp={pointerUp} onPointerCancel={() => { drag.current = null; setPreview(null); }} onClick={() => { if (!suppressClick.current) open(goal); suppressClick.current = false; }}>
                <strong>{goal.title}</strong><small>{goal.startDate} → {goal.endDate} · {goal.dailyHours === undefined ? 'Hours not set' : `${goal.dailyHours} h/day`}</small>
              </button>
              <button className="goal-handle" aria-label={`Resize end of ${goal.title}`} disabled={busy} onPointerDown={e => pointerDown(e, goal, 'end')} onPointerMove={pointerMove} onPointerUp={pointerUp} onPointerCancel={() => { drag.current = null; setPreview(null); }} onClick={() => { if (!suppressClick.current) open(goal); }}>‖</button>
            </div>;
          })}
        </div>
      </div>
    </div>
    <details className="timeline-list" open><summary>All goals ({goals.length})</summary>{goals.map(goal => <button key={goal.id} disabled={busy} onClick={() => open(goal)}><span style={{ color: goal.color }}>●</span> {goal.title} · {goal.startDate} → {goal.endDate}</button>)}</details>
    <dialog ref={dialog} onCancel={event => { event.preventDefault(); if (!busy) setDraft(null); }}>
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
        {error && <div role="alert" className="timeline-error">{error}<button type="button" disabled={busy} onClick={() => void load()}>Load latest (keep draft)</button></div>}
        <fieldset disabled={busy}>
          {!creating && <label>Name<input required maxLength={200} value={draft.title} onChange={e => setDraft({ ...draft, title: e.target.value })} /></label>}
          <GoalSchedule draft={draft} goals={goals} onChange={setDraft} />
          {!creating && <>
            <h3>Notes</h3>
            <label>Add a note<textarea maxLength={20000} value={note} onChange={e => setNote(e.target.value)} placeholder="What changed? What did you learn?" /></label>
            {[...draft.notes].sort((a, b) => b.createdAt.localeCompare(a.createdAt)).map(n => <article className="goal-note" key={n.id}><time>{new Date(n.createdAt).toLocaleString()}</time><p>{n.body}</p></article>)}
            {!draft.notes.length && <p className="timeline-help">No notes yet.</p>}
            <h3>Substeps</h3>
            {draft.steps.map(s => <label className="goal-step" key={s.id}><input type="checkbox" checked={s.done} onChange={e => setDraft({ ...draft, steps: draft.steps.map(item => item.id === s.id ? { ...item, done: e.target.checked } : item) })} /><span>{s.title}</span><button type="button" aria-label={`Remove ${s.title}`} onClick={() => setDraft({ ...draft, steps: draft.steps.filter(item => item.id !== s.id) })}>×</button></label>)}
            <label>New substep<input maxLength={500} value={step} onChange={e => setStep(e.target.value)} /></label>
            <button type="button" disabled={!step.trim()} onClick={() => { setDraft({ ...draft, steps: [...draft.steps, { id: crypto.randomUUID(), title: step.trim(), done: false }] }); setStep(''); }}>Add substep</button>
            <h3>Metadata</h3>
            {Object.entries(draft.metadata).map(([key, value]) => <label key={key}>{key}<input maxLength={2000} value={value} onChange={e => setDraft({ ...draft, metadata: { ...draft.metadata, [key]: e.target.value } })} /></label>)}
            <div className="goal-dates"><label>New field<input maxLength={80} placeholder="e.g. Focus area" value={metaKey} onChange={e => setMetaKey(e.target.value)} /></label><label>Value<input maxLength={2000} value={metaValue} onChange={e => setMetaValue(e.target.value)} /></label></div>
            <button type="button" disabled={!metaKey.trim()} onClick={() => { setDraft({ ...draft, metadata: { ...draft.metadata, [metaKey.trim()]: metaValue } }); setMetaKey(''); setMetaValue(''); }}>Add metadata</button>
            <p className="timeline-help">Last updated {new Date(draft.updatedAt).toLocaleString()}</p>
          </>}
        </fieldset>
        <div className="timeline-toolbar">{!creating && <button type="button" disabled={busy} onClick={() => void remove(draft)}>Delete goal</button>}<button type="button" disabled={busy} onClick={() => setDraft(null)}>Cancel</button><button type="submit" disabled={busy}>{busy ? 'Saving…' : creating ? 'Create goal' : 'Save goal'}</button></div>
      </form>}
    </dialog>
  </div>;
}
