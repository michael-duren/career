import { useCallback, useEffect, useRef, useState, type DragEvent, type PointerEvent } from 'react';
import { addDays, clockLabel, draftMutation, localFields, localInstant, minuteOf, mondayOf, SchedulerError, schedulerRequest, snapMinutes, weekSchema, zonedDate, type Assignment, type Mutation, type SchedulerSession, type SchedulerWeek, type SessionDraft } from '../lib/scheduler';
import { WeeklySchedulerSettings } from './WeeklySchedulerSettings';
import { WeeklySchedulerGoogle } from './WeeklySchedulerGoogle';
import '../styles/weekly-scheduler.css';
const hours = (value: number) => `${Number(value.toFixed(2))}h`;
const browserZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone;
const dayName = (date: string) => new Intl.DateTimeFormat(undefined, { weekday: 'short', month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${date}T12:00Z`));
type DragItem = { kind: 'assignment'; assignment: Assignment } | { kind: 'session'; session: SchedulerSession };

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
  const drag = useRef<DragItem | null>(null);
  const initialized = useRef(false);
  const loadSequence = useRef(0);
  const editorRef = useRef<HTMLDivElement>(null);
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
  useEffect(() => { if (draft) { editorRef.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' }); editorRef.current?.querySelector<HTMLInputElement>('input')?.focus(); } }, [Boolean(draft)]);
  async function mutate(value: Mutation, close = true) {
    setSaving(true); setError(''); setConflicts([]); setNotice('');
    try { const result = weekSchema.parse(await schedulerRequest('mutate', value)); setWeek(result); if (close) setDraft(null); setNotice('Saved.'); return true; }
    catch (e) { if (e instanceof SchedulerError) { if (e.current) setWeek(e.current); setConflicts(e.conflictIds); } setError(`${e instanceof Error ? e.message : 'Save failed.'} Your draft is still available. Review and retry.`); return false; }
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
  function create(assignment: Assignment, date = selectedDay || weekDate, minute = 540, mode: 'plan' | 'actual' = 'plan'): SessionDraft {
    const startDate = addDays(date, Math.floor(minute / 1440));
    return { assignment, date, startDate, start: clockLabel(minute).slice(0, 5), endDate: addDays(date, Math.floor((minute + 60) / 1440)), end: clockLabel(minute + 60).slice(0, 5), mode, repeat: false, scope: 'date' };
  }
  function saveDraft(value: SessionDraft) {
    if (!week) return;
    try { void mutate(draftMutation(value, week)); } catch (e) { setError(e instanceof Error ? e.message : 'Check the session times.'); }
  }
  function beginDrag(e: DragEvent, item: DragItem) { drag.current = item; e.dataTransfer.setData('text/plain', item.kind === 'session' ? item.session.id : item.assignment.title); e.dataTransfer.effectAllowed = 'move'; }
  const intervalMinutes = (time: string) => Number(time.slice(0, 2)) * 60 + Number(time.slice(3));
  const axisStart = week ? Math.min(...week.days.map(day => intervalMinutes(day.interval.start)), 300) : 300;
  const axisEnd = week ? Math.max(...week.days.map(day => intervalMinutes(day.interval.end) + (day.interval.nextDay ? 1440 : 0)), 1230) : 1230;
  const height = axisEnd - axisStart;
  function drop(e: DragEvent<HTMLDivElement>, date: string) {
    e.preventDefault(); if (!week || !drag.current) return;
    const minute = snapMinutes(e.clientY - e.currentTarget.getBoundingClientRect().top + axisStart);
    const item = drag.current; drag.current = null;
    try {
      let proposal: SessionDraft;
      if (item.kind === 'assignment') proposal = create(item.assignment, date, minute);
      else {
        if (!item.session.plan || Date.parse(item.session.plan.start) <= Date.now()) return;
        const startDate = addDays(date, Math.floor(minute / 1440));
        const start = localInstant(startDate, clockLabel(minute).slice(0, 5), week.settings.timeZone);
        const end = localFields(new Date(Date.parse(start) + Date.parse(item.session.plan.end) - Date.parse(item.session.plan.start)).toISOString(), week.settings.timeZone);
        proposal = { ...create(item.session.assignment, date, minute), id: item.session.id, ruleId: item.session.ruleId, endDate: end.date, end: end.time };
      }
      setDraft(proposal); saveDraft(proposal);
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not place the session.'); }
  }
  function resize(e: PointerEvent<HTMLButtonElement>, session: SchedulerSession) {
    e.preventDefault(); e.stopPropagation(); if (!week || !session.plan || Date.parse(session.plan.start) <= Date.now()) return;
    const target = e.currentTarget; target.setPointerCapture(e.pointerId);
    const y = e.clientY, oldEnd = Date.parse(session.plan.end), start = localFields(session.plan.start, week.settings.timeZone);
    const move = (event: globalThis.PointerEvent) => { target.textContent = `${Math.max(15, (oldEnd - Date.parse(session.plan?.start ?? '') + snapMinutes(event.clientY - y) * 60000) / 60000)}m`; };
    const finish = (event: globalThis.PointerEvent) => {
      target.removeEventListener('pointermove', move); target.removeEventListener('pointerup', finish); target.removeEventListener('pointercancel', cancel); target.textContent = 'Resize';
      const end = localFields(new Date(Math.max(Date.parse(session.plan?.start ?? '') + 15 * 60000, oldEnd + snapMinutes(event.clientY - y) * 60000)).toISOString(), week.settings.timeZone);
      const proposal: SessionDraft = { id: session.id, ruleId: session.ruleId, assignment: session.assignment, date: session.date, startDate: start.date, start: start.time, originalStart: session.plan?.start, endDate: end.date, end: end.time, mode: 'plan', repeat: false, scope: 'date' };
      setDraft(proposal); saveDraft(proposal);
    };
    const cancel = () => { target.removeEventListener('pointermove', move); target.removeEventListener('pointerup', finish); target.removeEventListener('pointercancel', cancel); target.textContent = 'Resize'; };
    target.addEventListener('pointermove', move); target.addEventListener('pointerup', finish); target.addEventListener('pointercancel', cancel);
  }
  const activeSession = week?.sessions.find(s => s.id === draft?.id);
  return <div className="weekly-scheduler" aria-busy={loading || saving}>
    <div className="scheduler-toolbar"><div className="scheduler-actions"><button aria-label="Previous week" onClick={() => setWeekDate(addDays(weekDate, -7))}>←</button><button onClick={() => { const today = zonedDate(new Date(), week?.settings.timeZone ?? browserZone()); setWeekDate(mondayOf(today)); setSelectedDay(today); }}>Today</button><button aria-label="Next week" onClick={() => setWeekDate(addDays(weekDate, 7))}>→</button></div><h2>{dayName(weekDate)} – {dayName(addDays(weekDate, 6))}</h2><button onClick={() => setSettingsOpen(v => !v)}>Day settings</button></div>
    {week && <p className="scheduler-muted">{week.settings.timeZone} · {hours(week.remainingCapacityHours)} remaining free time · Drag blocks in 15-minute steps, or use their editor.</p>}
    <div aria-live="polite" className="scheduler-status">{notice}{loading && ' Loading schedule…'}</div>
    {error && <div role="alert" className="scheduler-error">{error}<button onClick={() => void load()}>Refresh schedule</button></div>}
    <WeeklySchedulerGoogle week={weekDate} refreshWeek={load} />
    {week && settingsOpen && <WeeklySchedulerSettings settings={week.settings} saving={saving} close={() => setSettingsOpen(false)} save={settings => { void mutate({ action: 'settings', settings, revision: week.revision, week: weekDate }, false).then(ok => { if (ok) setSettingsOpen(false); }); }} />}
    {week && <>
      {week.warnings.length > 0 && <details className="scheduler-warnings" open><summary>Schedule needs attention ({week.warnings.length})</summary><ul>{week.warnings.map((warning, index) => <li key={index}>{warning}</li>)}</ul></details>}
      <div className="scheduler-workspace"><aside className="scheduler-goals" aria-label="Weekly requirements"><h2>Weekly requirements</h2><div className="scheduler-actions"><button onClick={() => setDraft(create({ title: '' }))}>Add commitment</button><button onClick={() => setDraft(create(week.goals[0] ? { goalId: week.goals[0].goal.id, title: week.goals[0].goal.title, color: week.goals[0].goal.color } : { title: '' }, selectedDay, 540, 'actual'))}>Log unplanned work</button></div>{week.goals.length === 0 && <p>No goals apply this week. <a href="/timeline">Add or date a goal on the timeline.</a></p>}{week.goals.map(summary => { const goal = summary.goal; const assignment = { goalId: goal.id, title: goal.title, color: goal.color }; return <article className="scheduler-goal" id={`scheduler-goal-${goal.id}`} key={goal.id} style={{ borderLeftColor: goal.color }}><button className="scheduler-goal-title" draggable onDragStart={e => beginDrag(e, { kind: 'assignment', assignment })} onClick={() => setDraft(create(assignment))}>{goal.title}<span aria-hidden="true">＋</span></button><dl><div><dt>Required</dt><dd>{summary.requiredHours === null ? 'Hours not set' : hours(summary.requiredHours)}</dd></div><div><dt>Actual</dt><dd>{hours(summary.actualHours)}</dd></div><div><dt>Still scheduled</dt><dd>{hours(summary.remainingScheduledHours)}</dd></div><div className={summary.uncoveredHours > 0 ? 'scheduler-shortfall' : ''}><dt>Uncovered</dt><dd>{hours(summary.uncoveredHours)}</dd></div></dl>{summary.requiredHours === null && <a href="/timeline">Set a daily estimate</a>}{summary.excessHours > 0 && <p>{hours(summary.excessHours)} extra coverage</p>}{goal.dependsOn.length > 0 && <p className="scheduler-muted">This goal has dependencies. Check their readiness on the timeline.</p>}{goal.steps.some(step => !step.completed) && <details><summary>Subgoals</summary>{goal.steps.filter(step => !step.completed).map(step => { const sub = { ...assignment, stepId: step.id, title: step.title, goalTitle: goal.title }; return <button key={step.id} className="scheduler-step" draggable onDragStart={e => beginDrag(e, { kind: 'assignment', assignment: sub })} onClick={() => setDraft(create(sub))}>{step.title}{summary.unscheduledStepIds.includes(step.id) && <small>Unscheduled</small>}</button>; })}</details>}</article>; })}</aside>
      <div className="scheduler-calendar"><label className="scheduler-day-picker">Selected day<select value={selectedDay} onChange={e => setSelectedDay(e.target.value)}>{week.days.map(day => <option key={day.date} value={day.date}>{dayName(day.date)}</option>)}</select></label><div className="scheduler-grid"><div className="scheduler-axis"><div className="scheduler-day-heading">Time</div><div style={{ height }}>{Array.from({ length: Math.ceil(height / 60) }, (_, index) => <span style={{ top: index * 60 }} key={index}>{clockLabel(axisStart + index * 60)}</span>)}</div></div>{week.days.map(day => { const startMinute = intervalMinutes(day.interval.start), endMinute = intervalMinutes(day.interval.end) + (day.interval.nextDay ? 1440 : 0); return <section className={`scheduler-day ${day.date === selectedDay ? 'is-selected' : ''}`} key={day.date} aria-label={dayName(day.date)}><div className="scheduler-day-heading"><strong>{dayName(day.date)}</strong><button aria-label={`Add session on ${day.date}`} onClick={() => { setSelectedDay(day.date); setDraft(create(week.goals[0] ? { goalId: week.goals[0].goal.id, title: week.goals[0].goal.title, color: week.goals[0].goal.color } : { title: '' }, day.date, startMinute)); }}>＋</button></div><div className="scheduler-day-body" data-scheduler-date={day.date} style={{ height }} onDragOver={e => e.preventDefault()} onDrop={e => drop(e, day.date)}><div className="scheduler-unavailable" style={{ top: 0, height: startMinute - axisStart }} /><div className="scheduler-unavailable" style={{ top: endMinute - axisStart, height: axisEnd - endMinute }} />{week.busy.map(busy => { const start = Math.max(axisStart, minuteOf(busy.start, day.date, week.settings.timeZone)), end = Math.min(axisEnd, minuteOf(busy.end, day.date, week.settings.timeZone)); return end > start ? <div className={`scheduler-block scheduler-busy ${conflicts.includes(busy.id) ? 'scheduler-conflict' : ''}`} key={busy.id} style={{ top: start - axisStart, height: Math.max(20, end - start) }}><strong>{busy.title || 'Google busy'}</strong><small>Google busy · {clockLabel(start)}–{clockLabel(end)}</small></div> : null; })}{week.sessions.filter(session => (session.actual?.date || session.date) === day.date && session.state !== 'canceled').map(session => { const interval = session.actual && session.actual.status !== 'skipped' ? session.actual : session.plan; const top = interval ? minuteOf(interval.start, day.date, week.settings.timeZone) : startMinute; const end = interval ? minuteOf(interval.end, day.date, week.settings.timeZone) : top + 60; const started = session.plan ? Date.parse(session.plan.start) <= Date.now() : true; return <div className={`scheduler-block ${session.assignment.goalId ? 'scheduler-work' : 'scheduler-fixed'} ${session.attention || conflicts.includes(session.id) ? 'scheduler-conflict' : ''} ${session.actual?.status === 'skipped' ? 'scheduler-skipped' : ''}`} key={session.id} data-session-id={session.id} style={{ top: Math.max(0, top - axisStart), minHeight: 30, height: Math.max(30, end - top), borderLeftColor: session.assignment.color }} draggable={!started && Boolean(session.plan)} onDragStart={e => beginDrag(e, { kind: 'session', session })}><button className="scheduler-block-main" onClick={() => edit(session)}><strong>{session.assignment.title}</strong>{session.assignment.goalTitle && <small>{session.assignment.goalTitle}</small>}<small>{clockLabel(top)}–{clockLabel(end)}</small><small>{session.attention || (session.actual ? `${session.actual.status} actual` : session.assignment.goalId ? 'Planned' : 'Commitment')}{session.ruleId ? ' · Weekly' : ''}</small>{session.actual?.outsideTimeline && <small>Outside original timeline</small>}</button>{!started && session.plan && <button className="scheduler-resize" aria-label={`Resize ${session.assignment.title}`} onPointerDown={e => resize(e, session)} onClick={() => edit(session)}>Resize</button>}</div>; })}</div></section>; })}</div></div></div>
    </>}
    {draft && week && <div ref={editorRef} className="scheduler-panel scheduler-editor" aria-label="Session editor"><form onSubmit={e => { e.preventDefault(); saveDraft(draft); }}><h2>{draft.mode === 'actual' ? 'Record actual work' : draft.id ? 'Edit planned session' : 'New session'}</h2>{activeSession?.plan && draft.mode === 'actual' && <p>Original plan: {localFields(activeSession.plan.start, week.settings.timeZone).time}–{localFields(activeSession.plan.end, week.settings.timeZone).time} ({localFields(activeSession.plan.start, week.settings.timeZone).offset} → {localFields(activeSession.plan.end, week.settings.timeZone).offset}). Actuals replace the assumption and preserve this plan.</p>}<label>Assignment<select value={draft.assignment.goalId ?? ''} disabled={Boolean(draft.id)} onChange={e => { const goal = week.goals.find(s => s.goal.id === e.target.value)?.goal; setDraft({ ...draft, assignment: goal ? { goalId: goal.id, title: goal.title, color: goal.color } : { title: '' } }); }}><option value="">Fixed commitment</option>{week.goals.map(summary => <option key={summary.goal.id} value={summary.goal.id}>{summary.goal.title}</option>)}{draft.assignment.goalId && !week.goals.some(s => s.goal.id === draft.assignment.goalId) && <option value={draft.assignment.goalId}>{draft.assignment.goalTitle || draft.assignment.title} (historical)</option>}</select></label>{!draft.assignment.goalId && <label>Commitment name<input required value={draft.assignment.title} onChange={e => setDraft({ ...draft, assignment: { ...draft.assignment, title: e.target.value } })} /></label>}{draft.assignment.goalId && !draft.id && <label>Subgoal<select value={draft.assignment.stepId ?? ''} onChange={e => { const goal = week.goals.find(s => s.goal.id === draft.assignment.goalId)?.goal; const step = goal?.steps.find(s => s.id === e.target.value); if (goal) setDraft({ ...draft, assignment: { goalId: goal.id, stepId: step?.id, title: step?.title || goal.title, goalTitle: step ? goal.title : undefined, color: goal.color } }); }}><option value="">Goal itself</option>{week.goals.find(s => s.goal.id === draft.assignment.goalId)?.goal.steps.filter(s => !s.completed).map(s => <option value={s.id} key={s.id}>{s.title}</option>)}</select></label>}<div className="scheduler-editor-fields"><label>Scheduling date<input required type="date" value={draft.date} onChange={e => setDraft({ ...draft, date: e.target.value, startDate: e.target.value, endDate: e.target.value })} /></label><label>Start date<input required type="date" value={draft.startDate ?? draft.date} onChange={e => setDraft({ ...draft, startDate: e.target.value })} /></label><label>Start time<input required type="time" step="60" value={draft.start} onChange={e => setDraft({ ...draft, start: e.target.value })} /></label><label>End date<input required type="date" value={draft.endDate} onChange={e => setDraft({ ...draft, endDate: e.target.value })} /></label><label>End time<input required type="time" step="60" value={draft.end} onChange={e => setDraft({ ...draft, end: e.target.value })} /></label></div><p className="scheduler-muted">Times use {week.settings.timeZone}. Overnight work belongs to its scheduling date. Repeated clock times use the earlier occurrence.{(() => { try { return ` Start offset: ${localFields(draft.originalStart && localFields(draft.originalStart, week.settings.timeZone).date === (draft.startDate ?? draft.date) && localFields(draft.originalStart, week.settings.timeZone).time === draft.start ? draft.originalStart : localInstant(draft.startDate ?? draft.date, draft.start, week.settings.timeZone), week.settings.timeZone).offset}.`; } catch { return ''; } })()}</p>{draft.mode === 'plan' && (draft.ruleId ? <label>Apply change to<select value={draft.scope} onChange={e => setDraft({ ...draft, scope: e.target.value === 'future' ? 'future' : 'date' })}><option value="date">This date</option><option value="future">This and future occurrences</option></select></label> : !draft.id && <label className="scheduler-check"><input type="checkbox" checked={draft.repeat} onChange={e => setDraft({ ...draft, repeat: e.target.checked })} />Repeat weekly from this date</label>)}{draft.mode === 'actual' && !draft.assignment.goalId && <p role="alert">Actual work must be assigned to a goal.</p>}<div className="scheduler-actions"><button type="submit" disabled={saving || draft.mode === 'actual' && !draft.assignment.goalId}>{saving ? 'Saving…' : 'Save session'}</button><button type="button" onClick={() => { setDraft(null); setConflicts([]); }}>Discard draft</button>{draft.id && (draft.mode === 'plan' || draft.mode === 'actual' && !activeSession?.plan) && <button type="button" disabled={saving} onClick={() => { if (draft.id) void mutate({ revision: week.revision, week: week.week, action: 'cancel', id: draft.id }); }}>Remove this session</button>}{draft.id && draft.mode === 'actual' && activeSession?.plan && <><button type="button" disabled={saving} onClick={() => { if (draft.id && activeSession.plan) void mutate({ revision: week.revision, week: week.week, action: 'actual', id: draft.id, actual: { ...activeSession.plan, date: activeSession.date, status: 'skipped' } }); }}>Mark skipped</button><button type="button" disabled={saving || Date.parse(activeSession.plan.end) > Date.now()} title={Date.parse(activeSession.plan.end) > Date.now() ? 'Available once the planned session ends' : undefined} onClick={() => { if (draft.id && activeSession.plan) void mutate({ revision: week.revision, week: week.week, action: 'actual', id: draft.id, actual: { ...activeSession.plan, date: activeSession.date, status: 'assumed' } }); }}>Restore as planned</button></>}</div></form></div>}
  </div>;
}
