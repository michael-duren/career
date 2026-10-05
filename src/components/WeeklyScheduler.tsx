import { useCallback, useEffect, useRef, useState } from 'react';
import { quickAddSlot, addDays, clockLabel, draftMutation, localFields, localInstant, mondayOf, SchedulerError, schedulerRequest, weekSchema, zonedDate, type Assignment, type Mutation, type SchedulerSession, type SchedulerWeek, type SessionDraft } from '../lib/scheduler';
import { type PlacementResult } from '../lib/scheduler-placement';
import { WeeklySchedulerGrid, type SchedulerGridHandle } from './WeeklySchedulerGrid';
import { WeeklySchedulerSettings } from './WeeklySchedulerSettings';
import { WeeklySchedulerGoogle } from './WeeklySchedulerGoogle';
import '../styles/weekly-scheduler.css';
const hours = (value: number) => `${Number(value.toFixed(2))}h`;
const browserZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone;
const dayName = (date: string) => new Intl.DateTimeFormat(undefined, { weekday: 'short', month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${date}T12:00Z`));
// Dragging a goal or subgoal onto the grid places a one-hour block; resize it afterward.
const DRAG_MINUTES = 60;

export function WeeklyScheduler() {
  const [weekDate, setWeekDate] = useState(() => mondayOf(zonedDate(new Date(), browserZone())));
  const [week, setWeek] = useState<SchedulerWeek | null>(null);
  const [selectedDay, setSelectedDay] = useState('');
  const [draft, setDraftState] = useState<SessionDraft | null>(null);
  const [draftWeek, setDraftWeek] = useState<SchedulerWeek | null>(null);
  const [now, setNow] = useState(Date.now);
  const [healthRefresh, setHealthRefresh] = useState(0);
  const viewedWeek = useRef(weekDate);
  const mutationPending = useRef(false);
  const reading = useRef(true);
  function navigate(date: string) { if (date === viewedWeek.current) { void load(); return; } viewedWeek.current = date; loadSequence.current++; setWeekDate(date); setWeek(null); reading.current = true; setLoading(true); }
  function setDraft(value: SessionDraft | null) {
    if (mutationPending.current) return;
    if (value && !draft) setDraftWeek(week);
    setDraftState(value);
    if (!value) { setDraftWeek(null); setConflicts([]); }
  }
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [conflicts, setConflicts] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const grid = useRef<SchedulerGridHandle>(null);
  const initialized = useRef(false);
  const loadSequence = useRef(0);
  const dialog = useRef<HTMLDialogElement>(null);
  const load = useCallback(async (clearError = true) => {
    const sequence = ++loadSequence.current;
    const requestedWeek = viewedWeek.current;
    reading.current = true;
    setLoading(true);
    setHealthRefresh(value => value + 1);
    try {
      const data = weekSchema.parse(await schedulerRequest(`week?week=${requestedWeek}&timeZone=${encodeURIComponent(browserZone())}`));
      if (sequence !== loadSequence.current || requestedWeek !== viewedWeek.current || data.week !== requestedWeek) return;
      setWeek(data);
      if (clearError) setError('');
      setSelectedDay(current => current >= data.week && current <= addDays(data.week, 6) ? current : data.week);
      if (!initialized.current) {
        initialized.current = true;
        const today = zonedDate(new Date(), data.settings.timeZone);
        if (mondayOf(today) !== viewedWeek.current) navigate(mondayOf(today));
        setSelectedDay(today);
      }
    } catch (e) { if (sequence === loadSequence.current) setError(e instanceof Error ? e.message : 'Could not load the schedule.'); }
    finally { if (sequence === loadSequence.current) { reading.current = false; setLoading(false); } }
  }, []);
  useEffect(() => { void load(); return () => { loadSequence.current++; }; }, [load, weekDate]);
  useEffect(() => { if (draft) dialog.current?.showModal(); else dialog.current?.close(); }, [Boolean(draft)]);
  useEffect(() => {
    const refresh = () => { if (document.visibilityState === 'visible') { setNow(Date.now()); void load(); } };
    const timer = window.setInterval(refresh, 60000);
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', refresh);
    return () => { clearInterval(timer); window.removeEventListener('focus', refresh); document.removeEventListener('visibilitychange', refresh); };
  }, [load]);
  const refreshWeek = useCallback(() => { void load(); }, [load]);
  useEffect(() => {
    const boundaries = [week, draftWeek].flatMap(value => value?.sessions.flatMap(session => session.plan ? [Date.parse(session.plan.start), Date.parse(session.plan.end)] : []) ?? []).filter(time => time > now);
    const delay = Math.max(1, Math.min(60000, ...boundaries.map(time => time - Date.now())));
    const timer = window.setTimeout(() => { if (document.visibilityState === 'visible') setNow(Date.now()); }, delay);
    return () => clearTimeout(timer);
  }, [week, draftWeek, now]);
  async function mutate(value: Mutation, close = true): Promise<boolean> {
    if (mutationPending.current || reading.current) return false;
    mutationPending.current = true;
    loadSequence.current++;
    setSaving(true); setError(''); setConflicts([]); setNotice('');
    try {
      const result = weekSchema.parse(await schedulerRequest('mutate', value));
      loadSequence.current++;
      if (result.week === viewedWeek.current) setWeek(result);
      if (close) { setDraftState(null); setDraftWeek(null); }
      setNotice('Saved.');
      return true;
    } catch (e) {
      loadSequence.current++;
      if (e instanceof SchedulerError && e.current) {
        if (e.current.week === viewedWeek.current) setWeek(e.current);
        if (e.current.week === value.week) setDraftWeek(e.current);
        setConflicts(e.conflictIds);
      }
      setError(`${e instanceof Error ? e.message : 'Save failed.'} Your draft is still available. Review and retry.`);
      return false;
    } finally {
      mutationPending.current = false;
      reading.current = false;
      setLoading(false);
      setSaving(false);
      if (value.week !== viewedWeek.current) void load(false);
    }
  }
  function edit(session: SchedulerSession) {
    if (!week || mutationPending.current || reading.current) return;
    if (!session.assignment.goalId && session.plan && Date.parse(session.plan.start) <= Date.now()) { setNotice('This commitment has started. Its original plan is preserved.'); return; }
    const mode = session.actual || session.plan && Date.parse(session.plan.start) <= Date.now() ? 'actual' : 'plan';
    const interval = session.actual?.status !== 'skipped' && session.actual ? session.actual : session.plan;
    if (!interval) { const rule = week.rules.find(rule => rule.id === session.ruleId); setDraft({ ...create(session.assignment, session.date), id: session.id, ruleId: session.ruleId, start: rule?.localStart ?? '09:00' }); return; }
    const start = localFields(interval.start, week.settings.timeZone), end = localFields(interval.end, week.settings.timeZone);
    setDraft({ id: session.id, ruleId: session.ruleId, assignment: session.assignment, date: mode === 'actual' && session.actual ? session.actual.date : session.date, startDate: start.date, start: start.time, endDate: end.date, end: end.time, originalStart: interval.start, originalEnd: interval.end, mode, repeat: false, scope: 'date' });
  }
  function deleteSession(session: SchedulerSession) {
    if (!week || mutationPending.current || reading.current) return;
    if (session.ruleId) {
      if (!window.confirm(`Delete this and all future occurrences of ${session.assignment.title}? To delete only this date, open the session and use "Remove this session" with "This date" selected.`)) return;
      void mutate({ revision: week.revision, week: week.week, action: 'cancel', id: session.id, scope: 'future' });
      return;
    }
    if (!window.confirm(`Delete this session for ${session.assignment.title}?`)) return;
    void mutate({ revision: week.revision, week: week.week, action: 'cancel', id: session.id });
  }
  function create(assignment: Assignment, date = selectedDay || weekDate, minute: number | undefined = undefined, mode?: 'plan' | 'actual', durationMinutes = 60, at = Date.now()): SessionDraft {
    if (minute === undefined && week && mode !== 'actual') {
      const slot = quickAddSlot(week, date, new Date(now), durationMinutes);
      if (slot.reason) return {assignment,date,startDate:date,start:'',endDate:date,end:'',mode:'plan',repeat:false,scope:'date',explanation:slot.reason};
      const a=localFields(slot.start,week.settings.timeZone), b=localFields(slot.end,week.settings.timeZone);
      return {assignment,date,startDate:a.date,start:a.time,endDate:b.date,end:b.time,originalStart:slot.start,originalEnd:slot.end,mode:'plan',repeat:false,scope:'date'};
    }
    minute ??= 540;
    const startDate = addDays(date, Math.floor(minute / 1440));
    const start = clockLabel(minute).slice(0, 5);
    let resolvedMode = mode;
    if (!resolvedMode) {
      resolvedMode = 'plan';
      // A block dropped or quick-added onto an already-past slot can never be
      // a future plan; default it to a recorded actual instead of letting the
      // server reject it outright, matching how editing an existing session
      // already switches to actual mode once it has started.
      if (week) { try { if (Date.parse(localInstant(startDate, start, week.settings.timeZone)) <= at) resolvedMode = 'actual'; } catch { /* leave as plan; the editor surfaces the real error */ } }
    }
    return { assignment, date, startDate, start, endDate: addDays(date, Math.floor((minute + durationMinutes) / 1440)), end: clockLabel(minute + durationMinutes).slice(0, 5), mode: resolvedMode, repeat: false, scope: 'date' };
  }
  function saveDraft(value: SessionDraft) {
    if (!draftWeek || mutationPending.current || reading.current) return;
    try {
      const original = draftWeek.sessions.find(session => session.id === value.id)?.plan;
      if (value.mode === 'plan' && original && Date.parse(original.start) <= Date.now()) throw new Error('This session has started. Its original plan is preserved. Record actual work instead.');
      const mutation = draftMutation(value, draftWeek);
      if (mutation.action === 'actual' && Date.parse(mutation.actual.end) > Date.now()) throw new Error('Actual work cannot end in the future.');
      void mutate(mutation);
    } catch (e) { setError(e instanceof Error ? e.message : 'Check the session times.'); }
  }
  // Drag/resize save immediately without opening the editor dialog - it would
  // otherwise flash open and shut on every successful move. Only surface the
  // dialog if the save actually fails, so the user can see and fix the draft.
  function attemptPlace(result: PlacementResult) {
    if (!week || mutationPending.current || reading.current) return;
    if (result.kind === 'invalid') { setError(result.message); return; }
    const proposal = result.draft;
    const owner = week;
    try { void mutate(draftMutation(proposal, week)).then(ok => { if (!ok) { setDraftState(proposal); setDraftWeek(current => current?.week === owner.week ? current : owner); } }); }
    catch (e) { setError(e instanceof Error ? e.message : 'Check the session times.'); setDraft(proposal); }
  }
  const editorWeek = draftWeek;
  const busy = loading || saving;
  const activeSession = editorWeek?.sessions.find(s => s.id === draft?.id);
  const planDraftStarted = draft?.mode === 'plan' && Boolean(activeSession?.plan && Date.parse(activeSession.plan.start) <= now);
  return <div className="weekly-scheduler" aria-busy={loading || saving}>
    <div className="scheduler-toolbar"><div className="scheduler-actions"><button aria-label="Previous week" onClick={() => navigate(addDays(weekDate, -7))}>←</button><button onClick={() => { const today = zonedDate(new Date(), week?.settings.timeZone ?? browserZone()); if (mondayOf(today) !== viewedWeek.current) navigate(mondayOf(today)); setSelectedDay(today); }}>Today</button><button aria-label="Next week" onClick={() => navigate(addDays(weekDate, 7))}>→</button></div><h2>{dayName(weekDate)} – {dayName(addDays(weekDate, 6))}</h2><button onClick={() => void load()}>Refresh schedule</button><button disabled={busy} onClick={() => setSettingsOpen(v => !v)}>Day settings</button></div>
    {week && <p className="scheduler-muted">{week.settings.timeZone} · {hours(week.remainingCapacityHours)} remaining free time · Drag blocks in 15-minute steps, or use their editor.</p>}
    <div aria-live="polite" className="scheduler-status">{notice}{loading && ' Loading schedule…'}</div>
    {/* When the dialog is open, its own copy below is what's actually visible -
        the native <dialog>'s ::backdrop sits above ordinary page content, so
        this one would otherwise be silently hidden behind it (e.g. a failed
        drag/resize save reopening the dialog via attemptPlace). */}
    {!draft && error && <div role="alert" className="scheduler-error">{error}<button onClick={() => void load()}>Refresh schedule</button></div>}
    <WeeklySchedulerGoogle refreshGeneration={healthRefresh} week={weekDate} refreshWeek={refreshWeek} />
    {week && settingsOpen && <WeeklySchedulerSettings settings={week.settings} saving={busy} close={() => setSettingsOpen(false)} save={settings => { void mutate({ action: 'settings', settings, revision: week.revision, week: weekDate }, false).then(ok => { if (ok) setSettingsOpen(false); }); }} />}
    {week && <>
      {week.warnings.length > 0 && <details className="scheduler-warnings" open><summary>Schedule needs attention ({week.warnings.length})</summary><ul>{week.warnings.map((warning, index) => <li key={index}>{(() => { const occurrence=week.warnings.slice(0,index).filter(message=>message===warning).length; const target=week.warningTargets.filter(target => target.message===warning)[occurrence]; return target ? <button onClick={() => { if (target.date) setSelectedDay(target.date); const session=week.sessions.find(session=>session.id===target.sessionId); if (session) edit(session); else if (target.goalId) document.getElementById(`scheduler-goal-${target.goalId}`)?.scrollIntoView({block:'nearest'}); else setSettingsOpen(true); }}>{warning}</button> : warning; })()}</li>)}</ul></details>}
      <div className="scheduler-workspace"><aside className="scheduler-goals" aria-label="Weekly requirements"><h2>Weekly requirements</h2><div className="scheduler-actions"><button disabled={busy} onClick={() => setDraft(create({ title: '' }))}>Add commitment</button><button disabled={busy} onClick={() => setDraft(create(week.goals[0] ? { goalId: week.goals[0].goal.id, title: week.goals[0].goal.title, color: week.goals[0].goal.color } : { title: '' }, selectedDay, 540, 'actual'))}>Log unplanned work</button></div>{week.goals.length === 0 && <p>No goals apply this week. <a href="/timeline">Add or date a goal on the timeline.</a></p>}{week.goals.map(summary => { const goal = summary.goal; const assignment = { goalId: goal.id, title: goal.title, color: goal.color }; return <article className="scheduler-goal" id={`scheduler-goal-${goal.id}`} key={goal.id} style={{ borderLeftColor: goal.color }}><button disabled={busy} className="scheduler-goal-title" onPointerDown={e => grid.current?.beginAssignment(e, assignment, DRAG_MINUTES)} onClick={() => { if (!grid.current?.consumeClick()) setDraft(create(assignment)); }}>{goal.title}<span aria-hidden="true">＋</span></button><button disabled={busy} className="scheduler-touch-handle scheduler-goal-handle" aria-label={`Drag ${goal.title}`} onPointerDown={e => grid.current?.beginAssignment(e, assignment, DRAG_MINUTES)}>⋮⋮</button><dl><div><dt>Required</dt><dd>{summary.requiredHours === null ? 'Hours not set' : hours(summary.requiredHours)}</dd></div><div><dt>Actual</dt><dd>{hours(summary.actualHours)}</dd></div><div><dt>Still scheduled</dt><dd>{hours(summary.remainingScheduledHours)}</dd></div><div className={summary.uncoveredHours > 0 ? 'scheduler-shortfall' : ''}><dt>Uncovered</dt><dd>{hours(summary.uncoveredHours)}</dd></div></dl>{summary.requiredHours === null && <a href="/timeline">Set a daily estimate</a>}{summary.excessHours > 0 && <p>{hours(summary.excessHours)} extra coverage</p>}{goal.dependsOn.length > 0 && <p className="scheduler-muted">This goal has dependencies. Check their readiness on the timeline.</p>}{goal.steps.some(step => !step.completed) && <details><summary>Subgoals</summary>{goal.steps.filter(step => !step.completed).map(step => { const sub = { ...assignment, stepId: step.id, title: step.title, goalTitle: goal.title }; return <div key={step.id} className="scheduler-step-row"><button disabled={busy} className="scheduler-step" onPointerDown={e => grid.current?.beginAssignment(e, sub, DRAG_MINUTES)} onClick={() => { if (!grid.current?.consumeClick()) setDraft(create(sub)); }}>{step.title}{summary.unscheduledStepIds.includes(step.id) && <small>Unscheduled</small>}</button><button disabled={busy} className="scheduler-touch-handle scheduler-step-handle" aria-label={`Drag ${step.title}`} onPointerDown={e => grid.current?.beginAssignment(e, sub, DRAG_MINUTES)}>⋮⋮</button></div>; })}</details>}</article>; })}</aside>
      <WeeklySchedulerGrid key={weekDate} ref={grid} busy={busy} now={now} week={week} selectedDay={selectedDay} conflicts={conflicts} selectDay={setSelectedDay} addSession={(date, minute, durationMinutes, at) => { const next = create(week.goals[0] ? { goalId: week.goals[0].goal.id, title: week.goals[0].goal.title, color: week.goals[0].goal.color } : { title: '' }, date, minute, undefined, durationMinutes, at); /* Past gaps open as actual work, which needs a goal; with none, there is nothing to record. */ if (next.mode === 'actual' && !next.assignment.goalId) { setNotice('Add a goal to record past work.'); return; } setSelectedDay(date); setDraft(next); }} edit={edit} deleteSession={deleteSession} place={attemptPlace} />
      </div>
    </>}
    <dialog ref={dialog} className="scheduler-editor-dialog" aria-label="Session editor" onCancel={event => { event.preventDefault(); if (!saving) setDraft(null); }}>{draft && editorWeek && <form onSubmit={e => { e.preventDefault(); saveDraft(draft); }}><h2>{draft.mode === 'actual' ? 'Record actual work' : draft.id ? 'Edit planned session' : 'New session'}</h2><p>Draft for week of {editorWeek.week}.</p>{draft.explanation && <p role="status">{draft.explanation}</p>}{planDraftStarted && <p role="alert">This session has started. Its original plan is preserved. Your draft fields are retained.</p>}{error && <div role="alert" className="scheduler-error">{error}<button type="button" onClick={() => void load()}>Refresh schedule</button></div>}{activeSession?.plan && draft.mode === 'actual' && <p>Original plan: {localFields(activeSession.plan.start, editorWeek.settings.timeZone).time}–{localFields(activeSession.plan.end, editorWeek.settings.timeZone).time} ({localFields(activeSession.plan.start, editorWeek.settings.timeZone).offset} → {localFields(activeSession.plan.end, editorWeek.settings.timeZone).offset}). Actuals replace the assumption and preserve this plan.</p>}<label>Assignment<select disabled={busy || Boolean(draft.id)} value={draft.assignment.goalId ?? ''} onChange={e => { const goal = editorWeek.goals.find(s => s.goal.id === e.target.value)?.goal; setDraft({ ...draft, assignment: goal ? { goalId: goal.id, title: goal.title, color: goal.color } : { title: '' } }); }}><option value="">Fixed commitment</option>{editorWeek.goals.map(summary => <option key={summary.goal.id} value={summary.goal.id}>{summary.goal.title}</option>)}{draft.assignment.goalId && !editorWeek.goals.some(s => s.goal.id === draft.assignment.goalId) && <option value={draft.assignment.goalId}>{draft.assignment.goalTitle || draft.assignment.title} (historical)</option>}</select></label>{!draft.assignment.goalId && <label>Commitment name<input disabled={busy} required value={draft.assignment.title} onChange={e => setDraft({ ...draft, assignment: { ...draft.assignment, title: e.target.value } })} /></label>}{draft.assignment.goalId && (draft.mode === 'plan' || !draft.id) && <label>Subgoal<select disabled={busy} value={draft.assignment.stepId ?? ''} onChange={e => { const goal = editorWeek.goals.find(s => s.goal.id === draft.assignment.goalId)?.goal; const step = goal?.steps.find(s => s.id === e.target.value); if (goal) setDraft({ ...draft, assignment: { goalId: goal.id, stepId: step?.id, title: step?.title || goal.title, goalTitle: step ? goal.title : undefined, color: goal.color } }); }}><option value="">Goal itself</option>{editorWeek.goals.find(s => s.goal.id === draft.assignment.goalId)?.goal.steps.filter(s => !s.completed || s.id === draft.assignment.stepId).map(s => <option value={s.id} key={s.id}>{s.title}</option>)}</select></label>}<div className="scheduler-editor-fields"><label>Scheduling date<input disabled={busy} required type="date" value={draft.date} onChange={e => setDraft({ ...draft, date: e.target.value, startDate: e.target.value, endDate: e.target.value })} /></label><label>Start date<input disabled={busy} required type="date" value={draft.startDate ?? draft.date} onChange={e => setDraft({ ...draft, startDate: e.target.value })} /></label><label>Start time<input disabled={busy} required type="time" step="60" value={draft.start} onChange={e => setDraft({ ...draft, start: e.target.value })} /></label><label>End date<input disabled={busy} required type="date" value={draft.endDate} onChange={e => setDraft({ ...draft, endDate: e.target.value })} /></label><label>End time<input disabled={busy} required type="time" step="60" value={draft.end} onChange={e => setDraft({ ...draft, end: e.target.value })} /></label></div><p className="scheduler-muted">Times use {editorWeek.settings.timeZone}. Overnight work belongs to its scheduling date. Repeated clock times use the earlier occurrence.{(() => { try { return ` Start offset: ${localFields(draft.originalStart && localFields(draft.originalStart, editorWeek.settings.timeZone).date === (draft.startDate ?? draft.date) && localFields(draft.originalStart, editorWeek.settings.timeZone).time === draft.start ? draft.originalStart : localInstant(draft.startDate ?? draft.date, draft.start, editorWeek.settings.timeZone), editorWeek.settings.timeZone).offset}.`; } catch { return ''; } })()}</p>{draft.mode === 'plan' && (draft.ruleId ? <label>Apply change to<select disabled={busy} value={draft.scope} onChange={e => setDraft({ ...draft, scope: e.target.value === 'future' ? 'future' : 'date' })}><option value="date">This date</option><option value="future">This and future occurrences</option></select></label> : !draft.id && <label className="scheduler-check"><input disabled={busy} type="checkbox" checked={draft.repeat} onChange={e => setDraft({ ...draft, repeat: e.target.checked })} />Repeat weekly from this date</label>)}{draft.mode === 'actual' && draft.id && <p>Actual work is assigned to {draft.assignment.goalTitle ? `${draft.assignment.goalTitle} / ${draft.assignment.title}` : draft.assignment.title}.</p>}{draft.mode === 'actual' && !draft.assignment.goalId && <p role="alert">Actual work must be assigned to a goal.</p>}<div className="scheduler-actions"><button type="submit" disabled={busy || planDraftStarted || draft.mode === 'actual' && !draft.assignment.goalId}>{saving ? 'Saving…' : 'Save session'}</button><button type="button" disabled={busy} onClick={() => { setDraft(null); setConflicts([]); }}>Discard draft</button>{planDraftStarted && activeSession?.assignment.goalId && <button type="button" disabled={busy} onClick={() => setDraft({ ...draft, assignment: activeSession.assignment, mode: 'actual', repeat: false })}>Record actual work</button>}{draft.id && !planDraftStarted && (draft.mode === 'plan' || draft.mode === 'actual' && !activeSession?.plan) && <button type="button" disabled={busy} onClick={() => { if (draft.id) void mutate({ revision: editorWeek.revision, week: editorWeek.week, action: 'cancel', id: draft.id, scope: draft.ruleId ? draft.scope : undefined }); }}>Remove this session</button>}{draft.id && draft.mode === 'actual' && activeSession?.plan && <><button type="button" disabled={busy} onClick={() => { if (draft.id && activeSession.plan) void mutate({ revision: editorWeek.revision, week: editorWeek.week, action: 'actual', id: draft.id, actual: { ...activeSession.plan, date: activeSession.date, status: 'skipped' } }); }}>Mark skipped</button><button type="button" disabled={busy || Date.parse(activeSession.plan.end) > now} title={Date.parse(activeSession.plan.end) > now ? 'Available once the planned session ends' : undefined} onClick={() => { if (draft.id && activeSession.plan) void mutate({ revision: editorWeek.revision, week: editorWeek.week, action: 'actual', id: draft.id, actual: { ...activeSession.plan, date: activeSession.date, status: 'assumed' } }); }}>Restore as planned</button></>}</div></form>}</dialog>
  </div>;
}
