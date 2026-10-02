import { useCallback, useEffect, useRef, useState, type PointerEvent } from 'react';
import { addDays, clockLabel, displayClock, draftMutation, localFields, localInstant, minuteOf, mondayOf, SchedulerError, schedulerRequest, snapMinutes, weekSchema, zonedDate, type Assignment, type Mutation, type SchedulerSession, type SchedulerWeek, type SessionDraft } from '../lib/scheduler';
import { proposePlacement, type PlacementResult } from '../lib/scheduler-placement';
import { WeeklySchedulerSettings } from './WeeklySchedulerSettings';
import { WeeklySchedulerGoogle } from './WeeklySchedulerGoogle';
import '../styles/weekly-scheduler.css';
const hours = (value: number) => `${Number(value.toFixed(2))}h`;
const browserZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone;
const dayName = (date: string) => new Intl.DateTimeFormat(undefined, { weekday: 'short', month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${date}T12:00Z`));
type DragItem = { kind: 'assignment'; assignment: Assignment } | { kind: 'session'; session: SchedulerSession };
type PointerDragState = { item: DragItem; pointerId: number; startX: number; startY: number; grabOffsetMinutes: number; moved: boolean };

export function WeeklyScheduler() {
  const [weekDate, setWeekDate] = useState(() => mondayOf(zonedDate(new Date(), browserZone())));
  const [week, setWeek] = useState<SchedulerWeek | null>(null);
  const [selectedDay, setSelectedDay] = useState('');
  const [draft, setDraft] = useState<SessionDraft | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [conflicts, setConflicts] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const drag = useRef<PointerDragState | null>(null);
  const suppressClick = useRef(false);
  const [pointerGhost, setPointerGhost] = useState<{ x: number; y: number; label: string } | null>(null);
  const initialized = useRef(false);
  const loadSequence = useRef(0);
  const dialog = useRef<HTMLDialogElement>(null);
  const load = useCallback(async () => {
    const sequence = ++loadSequence.current;
    setLoading(true);
    try {
      const data = weekSchema.parse(await schedulerRequest(`week?week=${weekDate}&timeZone=${encodeURIComponent(browserZone())}`));
      if (sequence !== loadSequence.current) return;
      setWeek(data);
      setSelectedDay(current => current >= data.week && current <= addDays(data.week, 6) ? current : data.week);
      if (!initialized.current) {
        initialized.current = true;
        const today = zonedDate(new Date(), data.settings.timeZone);
        setWeekDate(mondayOf(today));
        setSelectedDay(today);
      }
    } catch (e) { if (sequence === loadSequence.current) setError(e instanceof Error ? e.message : 'Could not load the schedule.'); }
    finally { if (sequence === loadSequence.current) setLoading(false); }
  }, [weekDate]);
  useEffect(() => { void load(); return () => { loadSequence.current++; }; }, [load]);
  useEffect(() => { if (draft) dialog.current?.showModal(); else dialog.current?.close(); }, [Boolean(draft)]);
  async function mutate(value: Mutation, close = true): Promise<boolean> {
    setSaving(true); setError(''); setConflicts([]); setNotice('');
    try { const result = weekSchema.parse(await schedulerRequest('mutate', value)); setWeek(result); if (close) setDraft(null); setNotice('Saved.'); return true; }
    catch (e) {
      // A plain week fetch can silently advance the revision (newly generated
      // recurring occurrences, reconciliation), so a stale-revision rejection
      // here doesn't necessarily mean a real conflicting edit - but the server
      // has no way to tell the two apart either (empty conflictIds either
      // way), so this can't be auto-retried without risking silently
      // overwriting someone else's real edit (another tab, the MCP write
      // path). setWeek(e.current) below already refreshes the revision, so a
      // manual retry (the user saving again) just works.
      if (e instanceof SchedulerError && e.current) { setWeek(e.current); setConflicts(e.conflictIds); }
      setError(`${e instanceof Error ? e.message : 'Save failed.'} Your draft is still available. Review and retry.`);
      return false;
    }
    finally { setSaving(false); }
  }
  function edit(session: SchedulerSession) {
    if (!week) return;
    if (!session.assignment.goalId && session.plan && Date.parse(session.plan.start) <= Date.now()) { setNotice('This commitment has started. Its original plan is preserved.'); return; }
    const mode = session.actual || session.plan && Date.parse(session.plan.start) <= Date.now() ? 'actual' : 'plan';
    const interval = session.actual?.status !== 'skipped' && session.actual ? session.actual : session.plan;
    if (!interval) { const rule = week.rules.find(rule => rule.id === session.ruleId); setDraft({ ...create(session.assignment, session.date), id: session.id, ruleId: session.ruleId, start: rule?.localStart ?? '09:00' }); return; }
    const start = localFields(interval.start, week.settings.timeZone), end = localFields(interval.end, week.settings.timeZone);
    setDraft({ id: session.id, ruleId: session.ruleId, assignment: session.assignment, date: mode === 'actual' && session.actual ? session.actual.date : session.date, startDate: start.date, start: start.time, endDate: end.date, end: end.time, originalStart: interval.start, originalEnd: interval.end, mode, repeat: false, scope: 'date' });
  }
  function deleteSession(session: SchedulerSession) {
    if (!week) return;
    if (session.ruleId) {
      if (!window.confirm(`Delete this and all future occurrences of ${session.assignment.title}? To delete only this date, open the session and use "Remove this session" with "This date" selected.`)) return;
      void mutate({ revision: week.revision, week: week.week, action: 'cancel', id: session.id, scope: 'future' });
      return;
    }
    if (!window.confirm(`Delete this session for ${session.assignment.title}?`)) return;
    void mutate({ revision: week.revision, week: week.week, action: 'cancel', id: session.id });
  }
  function create(assignment: Assignment, date = selectedDay || weekDate, minute = 540, mode?: 'plan' | 'actual', durationMinutes = 60): SessionDraft {
    const startDate = addDays(date, Math.floor(minute / 1440));
    const start = clockLabel(minute).slice(0, 5);
    let resolvedMode = mode;
    if (!resolvedMode) {
      resolvedMode = 'plan';
      // A block dropped or quick-added onto an already-past slot can never be
      // a future plan; default it to a recorded actual instead of letting the
      // server reject it outright, matching how editing an existing session
      // already switches to actual mode once it has started.
      if (week) { try { if (Date.parse(localInstant(startDate, start, week.settings.timeZone)) <= Date.now()) resolvedMode = 'actual'; } catch { /* leave as plan; the editor surfaces the real error */ } }
    }
    return { assignment, date, startDate, start, endDate: addDays(date, Math.floor((minute + durationMinutes) / 1440)), end: clockLabel(minute + durationMinutes).slice(0, 5), mode: resolvedMode, repeat: false, scope: 'date' };
  }
  function suggestedMinutes(assignment: Assignment): number {
    const summary = assignment.goalId ? week?.goals.find(s => s.goal.id === assignment.goalId) : undefined;
    if (!summary || summary.uncoveredHours <= 0) return 60;
    return Math.min(240, Math.max(15, Math.round(summary.uncoveredHours * 4) * 15));
  }
  function saveDraft(value: SessionDraft) {
    if (!week) return;
    try { void mutate(draftMutation(value, week)); } catch (e) { setError(e instanceof Error ? e.message : 'Check the session times.'); }
  }
  // Drag/resize save immediately without opening the editor dialog - it would
  // otherwise flash open and shut on every successful move. Only surface the
  // dialog if the save actually fails, so the user can see and fix the draft.
  function attemptPlace(result: PlacementResult) {
    if (!week) return;
    if (result.kind === 'invalid') { setError(result.message); return; }
    const proposal = result.draft;
    try { void mutate(draftMutation(proposal, week)).then(ok => { if (!ok) setDraft(proposal); }); }
    catch (e) { setError(e instanceof Error ? e.message : 'Check the session times.'); setDraft(proposal); }
  }
  // Native HTML5 drag-and-drop is unreliable across browsers/window managers
  // (the rest of the app avoids it too - see GoalTimeline's pointer-based
  // move/resize), so dragging here is implemented on pointer events instead:
  // capture the pointer on the drag source, track its position by hand, and
  // hit-test the element under the cursor on move/release.
  function beginPointerDrag(e: PointerEvent, item: DragItem) {
    if (e.button !== 0) return;
    e.currentTarget.setPointerCapture(e.pointerId);
    const block = e.currentTarget.closest<HTMLElement>('.scheduler-block');
    const grabOffsetMinutes = item.kind === 'session' && block ? snapMinutes(e.clientY - block.getBoundingClientRect().top) : 0;
    drag.current = { item, pointerId: e.pointerId, startX: e.clientX, startY: e.clientY, grabOffsetMinutes, moved: false };
  }
  function pointerDragMove(e: PointerEvent) {
    const current = drag.current;
    if (!current || current.pointerId !== e.pointerId) return;
    if (!current.moved && Math.hypot(e.clientX - current.startX, e.clientY - current.startY) < 4) return;
    current.moved = true;
    setPointerGhost({ x: e.clientX, y: e.clientY, label: current.item.kind === 'assignment' ? current.item.assignment.title : current.item.session.assignment.title });
  }
  function pointerDragEnd(e: PointerEvent) {
    const current = drag.current; drag.current = null;
    setPointerGhost(null);
    if (!current || current.pointerId !== e.pointerId || !week) return;
    suppressClick.current = current.moved;
    if (!current.moved) return;
    const dayEl = document.elementFromPoint(e.clientX, e.clientY)?.closest<HTMLElement>('[data-scheduler-date]');
    const targetDate = dayEl?.dataset.schedulerDate;
    if (!dayEl || !targetDate) return;
    const startMinute = snapMinutes(e.clientY - dayEl.getBoundingClientRect().top + axisStart - current.grabOffsetMinutes);
    attemptPlace(current.item.kind === 'assignment'
      ? proposePlacement({ kind: 'assignment', assignment: current.item.assignment, targetDate, startMinute, durationMinutes: suggestedMinutes(current.item.assignment) }, week, new Date())
      : proposePlacement({ kind: 'move', session: current.item.session, targetDate, startMinute }, week, new Date()));
  }
  function pointerDragCancel() { drag.current = null; setPointerGhost(null); }
  const intervalMinutes = (time: string) => Number(time.slice(0, 2)) * 60 + Number(time.slice(3));
  const axisStart = week ? Math.min(...week.days.map(day => intervalMinutes(day.interval.start)), 300) : 300;
  const axisEnd = week ? Math.max(...week.days.map(day => intervalMinutes(day.interval.end) + (day.interval.nextDay ? 1440 : 0)), 1230) : 1230;
  const height = axisEnd - axisStart;
  // The active interval to drag/resize: a recorded actual takes precedence
  // over the plan, matching edit()'s own mode choice. A session with a
  // started (or already-past) plan and no actual yet still edits via the
  // plan's own times, but the result always lands as 'actual' - dragging or
  // resizing never rewrites a plan once its start has passed, so the
  // original planned time stays intact as a historical record; only what
  // actually happened is editable that way.
  function activeInterval(session: SchedulerSession) {
    return session.actual && session.actual.status !== 'skipped' ? session.actual : session.plan;
  }
  function resizeEdge(e: PointerEvent<HTMLButtonElement>, session: SchedulerSession, edge: 'start' | 'end') {
    e.preventDefault(); e.stopPropagation();
    const interval = activeInterval(session);
    if (!week || !interval) return;
    const target = e.currentTarget; target.setPointerCapture(e.pointerId);
    const y = e.clientY, startMs = Date.parse(interval.start), endMs = Date.parse(interval.end);
    const move = (event: globalThis.PointerEvent) => {
      const deltaMs = snapMinutes(event.clientY - y) * 60000;
      const minutes = edge === 'end' ? Math.max(15, (endMs - startMs + deltaMs) / 60000) : Math.max(15, (endMs - startMs - deltaMs) / 60000);
      target.textContent = `${minutes}m`;
    };
    const cleanup = () => { target.removeEventListener('pointermove', move); target.removeEventListener('pointerup', finish); target.removeEventListener('pointercancel', cleanup); target.textContent = ''; };
    const finish = (event: globalThis.PointerEvent) => {
      cleanup();
      const deltaMinutes = snapMinutes(event.clientY - y);
      attemptPlace(proposePlacement({ kind: 'resize', session, edge, deltaMinutes }, week, new Date()));
    };
    target.addEventListener('pointermove', move); target.addEventListener('pointerup', finish); target.addEventListener('pointercancel', cleanup);
  }
  const activeSession = week?.sessions.find(s => s.id === draft?.id);
  return <div className="weekly-scheduler" aria-busy={loading || saving}>
    <div className="scheduler-toolbar"><div className="scheduler-actions"><button aria-label="Previous week" onClick={() => setWeekDate(addDays(weekDate, -7))}>←</button><button onClick={() => { const today = zonedDate(new Date(), week?.settings.timeZone ?? browserZone()); setWeekDate(mondayOf(today)); setSelectedDay(today); }}>Today</button><button aria-label="Next week" onClick={() => setWeekDate(addDays(weekDate, 7))}>→</button></div><h2>{dayName(weekDate)} – {dayName(addDays(weekDate, 6))}</h2><button onClick={() => setSettingsOpen(v => !v)}>Day settings</button></div>
    {week && <p className="scheduler-muted">{week.settings.timeZone} · {hours(week.remainingCapacityHours)} remaining free time · Drag blocks in 15-minute steps, or use their editor.</p>}
    <div aria-live="polite" className="scheduler-status">{notice}{loading && ' Loading schedule…'}</div>
    {/* When the dialog is open, its own copy below is what's actually visible -
        the native <dialog>'s ::backdrop sits above ordinary page content, so
        this one would otherwise be silently hidden behind it (e.g. a failed
        drag/resize save reopening the dialog via attemptPlace). */}
    {!draft && error && <div role="alert" className="scheduler-error">{error}<button onClick={() => void load()}>Refresh schedule</button></div>}
    <WeeklySchedulerGoogle week={weekDate} refreshWeek={load} />
    {week && settingsOpen && <WeeklySchedulerSettings settings={week.settings} saving={saving} close={() => setSettingsOpen(false)} save={settings => { void mutate({ action: 'settings', settings, revision: week.revision, week: weekDate }, false).then(ok => { if (ok) setSettingsOpen(false); }); }} />}
    {week && <>
      {week.warnings.length > 0 && <details className="scheduler-warnings" open><summary>Schedule needs attention ({week.warnings.length})</summary><ul>{week.warnings.map((warning, index) => <li key={index}>{warning}</li>)}</ul></details>}
      <div className="scheduler-workspace"><aside className="scheduler-goals" aria-label="Weekly requirements"><h2>Weekly requirements</h2><div className="scheduler-actions"><button onClick={() => setDraft(create({ title: '' }))}>Add commitment</button><button onClick={() => setDraft(create(week.goals[0] ? { goalId: week.goals[0].goal.id, title: week.goals[0].goal.title, color: week.goals[0].goal.color } : { title: '' }, selectedDay, 540, 'actual'))}>Log unplanned work</button></div>{week.goals.length === 0 && <p>No goals apply this week. <a href="/timeline">Add or date a goal on the timeline.</a></p>}{week.goals.map(summary => { const goal = summary.goal; const assignment = { goalId: goal.id, title: goal.title, color: goal.color }; return <article className="scheduler-goal" id={`scheduler-goal-${goal.id}`} key={goal.id} style={{ borderLeftColor: goal.color }}><button className="scheduler-goal-title" onPointerDown={e => beginPointerDrag(e, { kind: 'assignment', assignment })} onPointerMove={pointerDragMove} onPointerUp={pointerDragEnd} onPointerCancel={pointerDragCancel} onClick={() => { if (!suppressClick.current) setDraft(create(assignment)); suppressClick.current = false; }}>{goal.title}<span aria-hidden="true">＋</span></button><dl><div><dt>Required</dt><dd>{summary.requiredHours === null ? 'Hours not set' : hours(summary.requiredHours)}</dd></div><div><dt>Actual</dt><dd>{hours(summary.actualHours)}</dd></div><div><dt>Still scheduled</dt><dd>{hours(summary.remainingScheduledHours)}</dd></div><div className={summary.uncoveredHours > 0 ? 'scheduler-shortfall' : ''}><dt>Uncovered</dt><dd>{hours(summary.uncoveredHours)}</dd></div></dl>{summary.requiredHours === null && <a href="/timeline">Set a daily estimate</a>}{summary.excessHours > 0 && <p>{hours(summary.excessHours)} extra coverage</p>}{goal.dependsOn.length > 0 && <p className="scheduler-muted">This goal has dependencies. Check their readiness on the timeline.</p>}{goal.steps.some(step => !step.completed) && <details><summary>Subgoals</summary>{goal.steps.filter(step => !step.completed).map(step => { const sub = { ...assignment, stepId: step.id, title: step.title, goalTitle: goal.title }; return <button key={step.id} className="scheduler-step" onPointerDown={e => beginPointerDrag(e, { kind: 'assignment', assignment: sub })} onPointerMove={pointerDragMove} onPointerUp={pointerDragEnd} onPointerCancel={pointerDragCancel} onClick={() => { if (!suppressClick.current) setDraft(create(sub)); suppressClick.current = false; }}>{step.title}{summary.unscheduledStepIds.includes(step.id) && <small>Unscheduled</small>}</button>; })}</details>}</article>; })}</aside>
      <div className="scheduler-calendar"><label className="scheduler-day-picker">Selected day<select value={selectedDay} onChange={e => setSelectedDay(e.target.value)}>{week.days.map(day => <option key={day.date} value={day.date}>{dayName(day.date)}</option>)}</select></label><div className="scheduler-grid"><div className="scheduler-axis"><div className="scheduler-day-heading">Time</div><div style={{ height }}>{Array.from({ length: Math.ceil(height / 60) }, (_, index) => <span style={{ top: index * 60 }} key={index}>{displayClock(axisStart + index * 60)}</span>)}</div></div>{week.days.map(day => { const startMinute = intervalMinutes(day.interval.start), endMinute = intervalMinutes(day.interval.end) + (day.interval.nextDay ? 1440 : 0); return <section className={`scheduler-day ${day.date === selectedDay ? 'is-selected' : ''}`} key={day.date} aria-label={dayName(day.date)}><div className="scheduler-day-heading"><strong>{dayName(day.date)}</strong><button aria-label={`Add session on ${day.date}`} onClick={() => { setSelectedDay(day.date); setDraft(create(week.goals[0] ? { goalId: week.goals[0].goal.id, title: week.goals[0].goal.title, color: week.goals[0].goal.color } : { title: '' }, day.date, startMinute)); }}>＋</button></div><div className="scheduler-day-body" data-scheduler-date={day.date} style={{ height }}><div className="scheduler-unavailable" style={{ top: 0, height: startMinute - axisStart }} /><div className="scheduler-unavailable" style={{ top: endMinute - axisStart, height: axisEnd - endMinute }} />{week.busy.map(busy => { const start = Math.max(axisStart, minuteOf(busy.start, day.date, week.settings.timeZone)), end = Math.min(axisEnd, minuteOf(busy.end, day.date, week.settings.timeZone)); return end > start ? <div className={`scheduler-block scheduler-busy ${conflicts.includes(busy.id) ? 'scheduler-conflict' : ''}`} key={busy.id} style={{ top: start - axisStart, height: Math.max(20, end - start) }}><strong>{busy.title || 'Google busy'}</strong><small>Google busy · {displayClock(start)}–{displayClock(end)}</small></div> : null; })}{week.sessions.filter(session => (session.actual?.date || session.date) === day.date && session.state !== 'canceled').map(session => { const interval = session.actual && session.actual.status !== 'skipped' ? session.actual : session.plan; const top = interval ? minuteOf(interval.start, day.date, week.settings.timeZone) : startMinute; const end = interval ? minuteOf(interval.end, day.date, week.settings.timeZone) : top + 60; const started = session.plan ? Date.parse(session.plan.start) <= Date.now() : true; const canDelete = (!session.plan || !started) && (!session.actual || !session.ruleId); return <div className={`scheduler-block ${session.assignment.goalId ? 'scheduler-work' : 'scheduler-fixed'} ${session.attention || conflicts.includes(session.id) ? 'scheduler-conflict' : ''} ${session.actual?.status === 'skipped' ? 'scheduler-skipped' : ''}`} key={session.id} data-session-id={session.id} style={{ top: Math.max(0, top - axisStart), minHeight: 30, height: Math.max(30, end - top), borderLeftColor: session.assignment.color }}><button className="scheduler-block-main" onPointerDown={interval ? e => beginPointerDrag(e, { kind: 'session', session }) : undefined} onPointerMove={pointerDragMove} onPointerUp={pointerDragEnd} onPointerCancel={pointerDragCancel} onClick={() => { if (!suppressClick.current) edit(session); suppressClick.current = false; }}><strong>{session.assignment.title}</strong>{session.assignment.goalTitle && <small>{session.assignment.goalTitle}</small>}<small>{displayClock(top)}–{displayClock(end)}</small><small>{session.attention || (session.actual ? `${session.actual.status} actual` : session.assignment.goalId ? 'Planned' : 'Commitment')}{session.ruleId ? ' · Weekly' : ''}</small>{session.actual?.outsideTimeline && <small>Outside original timeline</small>}</button><div className="scheduler-block-actions"><button className="scheduler-icon-btn" aria-label={`Edit ${session.assignment.title}`} onClick={() => edit(session)}>✎</button>{canDelete && <button className="scheduler-icon-btn" aria-label={`Delete ${session.assignment.title}`} onClick={() => deleteSession(session)}>×</button>}</div>{interval && <><button className="scheduler-edge scheduler-edge-top" aria-label={`Change start time of ${session.assignment.title}`} onPointerDown={e => resizeEdge(e, session, 'start')} onClick={() => edit(session)} /><button className="scheduler-edge scheduler-edge-bottom" aria-label={`Change end time of ${session.assignment.title}`} onPointerDown={e => resizeEdge(e, session, 'end')} onClick={() => edit(session)} /></>}</div>; })}</div></section>; })}</div></div></div>
    </>}
    <dialog ref={dialog} className="scheduler-editor-dialog" aria-label="Session editor" onCancel={event => { event.preventDefault(); if (!saving) setDraft(null); }}>{draft && week && <form onSubmit={e => { e.preventDefault(); saveDraft(draft); }}><h2>{draft.mode === 'actual' ? 'Record actual work' : draft.id ? 'Edit planned session' : 'New session'}</h2>{error && <div role="alert" className="scheduler-error">{error}<button type="button" onClick={() => void load()}>Refresh schedule</button></div>}{activeSession?.plan && draft.mode === 'actual' && <p>Original plan: {localFields(activeSession.plan.start, week.settings.timeZone).time}–{localFields(activeSession.plan.end, week.settings.timeZone).time} ({localFields(activeSession.plan.start, week.settings.timeZone).offset} → {localFields(activeSession.plan.end, week.settings.timeZone).offset}). Actuals replace the assumption and preserve this plan.</p>}<label>Assignment<select value={draft.assignment.goalId ?? ''} disabled={Boolean(draft.id)} onChange={e => { const goal = week.goals.find(s => s.goal.id === e.target.value)?.goal; setDraft({ ...draft, assignment: goal ? { goalId: goal.id, title: goal.title, color: goal.color } : { title: '' } }); }}><option value="">Fixed commitment</option>{week.goals.map(summary => <option key={summary.goal.id} value={summary.goal.id}>{summary.goal.title}</option>)}{draft.assignment.goalId && !week.goals.some(s => s.goal.id === draft.assignment.goalId) && <option value={draft.assignment.goalId}>{draft.assignment.goalTitle || draft.assignment.title} (historical)</option>}</select></label>{!draft.assignment.goalId && <label>Commitment name<input required value={draft.assignment.title} onChange={e => setDraft({ ...draft, assignment: { ...draft.assignment, title: e.target.value } })} /></label>}{draft.assignment.goalId && (draft.mode === 'plan' || !draft.id) && <label>Subgoal<select value={draft.assignment.stepId ?? ''} onChange={e => { const goal = week.goals.find(s => s.goal.id === draft.assignment.goalId)?.goal; const step = goal?.steps.find(s => s.id === e.target.value); if (goal) setDraft({ ...draft, assignment: { goalId: goal.id, stepId: step?.id, title: step?.title || goal.title, goalTitle: step ? goal.title : undefined, color: goal.color } }); }}><option value="">Goal itself</option>{week.goals.find(s => s.goal.id === draft.assignment.goalId)?.goal.steps.filter(s => !s.completed || s.id === draft.assignment.stepId).map(s => <option value={s.id} key={s.id}>{s.title}</option>)}</select></label>}<div className="scheduler-editor-fields"><label>Scheduling date<input required type="date" value={draft.date} onChange={e => setDraft({ ...draft, date: e.target.value, startDate: e.target.value, endDate: e.target.value })} /></label><label>Start date<input required type="date" value={draft.startDate ?? draft.date} onChange={e => setDraft({ ...draft, startDate: e.target.value })} /></label><label>Start time<input required type="time" step="60" value={draft.start} onChange={e => setDraft({ ...draft, start: e.target.value })} /></label><label>End date<input required type="date" value={draft.endDate} onChange={e => setDraft({ ...draft, endDate: e.target.value })} /></label><label>End time<input required type="time" step="60" value={draft.end} onChange={e => setDraft({ ...draft, end: e.target.value })} /></label></div><p className="scheduler-muted">Times use {week.settings.timeZone}. Overnight work belongs to its scheduling date. Repeated clock times use the earlier occurrence.{(() => { try { return ` Start offset: ${localFields(draft.originalStart && localFields(draft.originalStart, week.settings.timeZone).date === (draft.startDate ?? draft.date) && localFields(draft.originalStart, week.settings.timeZone).time === draft.start ? draft.originalStart : localInstant(draft.startDate ?? draft.date, draft.start, week.settings.timeZone), week.settings.timeZone).offset}.`; } catch { return ''; } })()}</p>{draft.mode === 'plan' && (draft.ruleId ? <label>Apply change to<select value={draft.scope} onChange={e => setDraft({ ...draft, scope: e.target.value === 'future' ? 'future' : 'date' })}><option value="date">This date</option><option value="future">This and future occurrences</option></select></label> : !draft.id && <label className="scheduler-check"><input type="checkbox" checked={draft.repeat} onChange={e => setDraft({ ...draft, repeat: e.target.checked })} />Repeat weekly from this date</label>)}{draft.mode === 'actual' && !draft.assignment.goalId && <p role="alert">Actual work must be assigned to a goal.</p>}<div className="scheduler-actions"><button type="submit" disabled={saving || draft.mode === 'actual' && !draft.assignment.goalId}>{saving ? 'Saving…' : 'Save session'}</button><button type="button" onClick={() => { setDraft(null); setConflicts([]); }}>Discard draft</button>{draft.id && (draft.mode === 'plan' || draft.mode === 'actual' && !activeSession?.plan) && <button type="button" disabled={saving} onClick={() => { if (draft.id) void mutate({ revision: week.revision, week: week.week, action: 'cancel', id: draft.id, scope: draft.ruleId ? draft.scope : undefined }); }}>Remove this session</button>}{draft.id && draft.mode === 'actual' && activeSession?.plan && <><button type="button" disabled={saving} onClick={() => { if (draft.id && activeSession.plan) void mutate({ revision: week.revision, week: week.week, action: 'actual', id: draft.id, actual: { ...activeSession.plan, date: activeSession.date, status: 'skipped' } }); }}>Mark skipped</button><button type="button" disabled={saving || Date.parse(activeSession.plan.end) > Date.now()} title={Date.parse(activeSession.plan.end) > Date.now() ? 'Available once the planned session ends' : undefined} onClick={() => { if (draft.id && activeSession.plan) void mutate({ revision: week.revision, week: week.week, action: 'actual', id: draft.id, actual: { ...activeSession.plan, date: activeSession.date, status: 'assumed' } }); }}>Restore as planned</button></>}</div></form>}</dialog>
    {pointerGhost && <div className="scheduler-drag-ghost" style={{ left: pointerGhost.x, top: pointerGhost.y }} aria-hidden="true">{pointerGhost.label}</div>}
  </div>;
}
