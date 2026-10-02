import { forwardRef, useEffect, useImperativeHandle, useRef, useState, type PointerEvent } from 'react';
import { displayClock, localFields, minuteOf, snapMinutes, type Assignment, type SchedulerSession, type SchedulerWeek } from '../lib/scheduler';
import { proposePlacement, type PlacementResult } from '../lib/scheduler-placement';

type DragItem = { kind: 'assignment'; assignment: Assignment; durationMinutes: number } | { kind: 'session'; session: SchedulerSession };
type Gesture = {
  pointerId: number; source: HTMLElement; startX: number; startY: number; moved: boolean;
  item: DragItem | { kind: 'resize'; session: SchedulerSession; edge: 'start' | 'end' };
  grabOffsetMinutes: number; proposal: PlacementResult | null; stop: () => void;
};
type Preview = { date: string; top: number; height: number; start: string; end: string; duration: number; result: PlacementResult };
export type SchedulerGridHandle = { beginAssignment: (event: PointerEvent<HTMLElement>, assignment: Assignment, durationMinutes: number) => void; consumeClick: () => boolean };
type Props = {
  week: SchedulerWeek; selectedDay: string; conflicts: string[];
  selectDay: (date: string) => void; addSession: (date: string, minute: number) => void;
  edit: (session: SchedulerSession) => void; deleteSession: (session: SchedulerSession) => void;
  place: (result: PlacementResult) => void;
};
const intervalMinutes = (time: string) => Number(time.slice(0, 2)) * 60 + Number(time.slice(3));
const dayName = (date: string) => new Intl.DateTimeFormat(undefined, { weekday: 'short', month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${date}T12:00Z`));
const activeInterval = (session: SchedulerSession) => session.actual && session.actual.status !== 'skipped' ? session.actual : session.plan;

export const WeeklySchedulerGrid = forwardRef<SchedulerGridHandle, Props>(function WeeklySchedulerGrid({ week, selectedDay, conflicts, selectDay, addSession, edit, deleteSession, place }, ref) {
  const gesture = useRef<Gesture | null>(null);
  const suppressClick = useRef(false);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [outside, setOutside] = useState<{ x: number; y: number } | null>(null);
  const axisStart = Math.min(...week.days.map(day => intervalMinutes(day.interval.start)), 300);
  const axisEnd = Math.max(...week.days.map(day => intervalMinutes(day.interval.end) + (day.interval.nextDay ? 1440 : 0)), 1230);
  const height = axisEnd - axisStart;

  function clearGesture() {
    const current = gesture.current;
    if (!current) return;
    gesture.current = null;
    current.stop();
    if (current.source.hasPointerCapture(current.pointerId)) current.source.releasePointerCapture(current.pointerId);
    setPreview(null);
    setOutside(null);
  }
  useEffect(() => () => clearGesture(), [week.week]);

  function begin(event: PointerEvent<HTMLElement>, item: Gesture['item']) {
    if (event.button !== 0 || gesture.current || (event.pointerType === 'touch' && !event.currentTarget.classList.contains('scheduler-touch-handle') && !event.currentTarget.classList.contains('scheduler-edge'))) return;
    const source = event.currentTarget;
    const block = source.closest<HTMLElement>('.scheduler-block');
    const grabOffsetMinutes = item.kind === 'session' && block ? snapMinutes(event.clientY - block.getBoundingClientRect().top) : 0;
    const current: Gesture = { item, source, pointerId: event.pointerId, startX: event.clientX, startY: event.clientY, moved: false, grabOffsetMinutes, proposal: null, stop: () => {} };
    gesture.current = current;
    source.setPointerCapture(event.pointerId);
    let frame = 0;
    let lastX = event.clientX, lastY = event.clientY;
    const proposalAt = (x: number, y: number) => {
      const dayEl = document.elementFromPoint(x, y)?.closest<HTMLElement>('[data-scheduler-date]');
      let date = dayEl?.dataset.schedulerDate;
      if (!dayEl || !date) { current.proposal = null; setPreview(null); setOutside({ x, y }); return null; }
      setOutside(null);
      let top: number;
      let duration: number;
      let result: PlacementResult;
      if (item.kind === 'resize') {
        const interval = activeInterval(item.session);
        if (!interval) { current.proposal = null; return null; }
        const deltaMinutes = snapMinutes(y - current.startY);
        result = proposePlacement({ kind: 'resize', session: item.session, edge: item.edge, deltaMinutes }, week, new Date());
        const startMinute = minuteOf(interval.start, item.session.actual?.date ?? item.session.date, week.settings.timeZone);
        const oldDuration = (Date.parse(interval.end) - Date.parse(interval.start)) / 60000;
        top = startMinute + (item.edge === 'start' ? deltaMinutes : 0) - axisStart;
        duration = oldDuration + (item.edge === 'start' ? -deltaMinutes : deltaMinutes);
        date = item.session.actual?.date ?? item.session.date;
      } else {
        const startMinute = snapMinutes(y - dayEl.getBoundingClientRect().top + axisStart - current.grabOffsetMinutes);
        result = item.kind === 'assignment'
          ? proposePlacement({ kind: 'assignment', assignment: item.assignment, targetDate: date, startMinute, durationMinutes: item.durationMinutes }, week, new Date())
          : proposePlacement({ kind: 'move', session: item.session, targetDate: date, startMinute }, week, new Date());
        top = startMinute - axisStart;
        const interval = item.kind === 'session' ? activeInterval(item.session) : null;
        duration = item.kind === 'assignment' ? item.durationMinutes : interval ? (Date.parse(interval.end) - Date.parse(interval.start)) / 60000 : 60;
      }
      let startLabel = displayClock(top + axisStart);
      let endLabel = displayClock(top + axisStart + duration);
      if (result.kind === 'valid') {
        const start = localFields(result.start, week.settings.timeZone);
        const end = localFields(result.end, week.settings.timeZone);
        startLabel = start.time;
        endLabel = end.time;
        top = minuteOf(result.start, date, week.settings.timeZone) - axisStart;
        duration = (Date.parse(result.end) - Date.parse(result.start)) / 60000;
      }
      setPreview({ date, top, height: Math.max(1, duration), start: startLabel, end: endLabel, duration, result });
      current.proposal = result;
      return result;
    };
    const autoScroll = () => {
      if (gesture.current !== current) return;
      const edge = 60;
      const speed = lastY < edge ? -Math.min(22, (edge - lastY) / 3) : lastY > innerHeight - edge ? Math.min(22, (lastY - innerHeight + edge) / 3) : 0;
      if (speed) { window.scrollBy(0, speed); proposalAt(lastX, lastY); }
      frame = requestAnimationFrame(autoScroll);
    };
    const move = (pointer: globalThis.PointerEvent) => {
      if (pointer.pointerId !== current.pointerId || gesture.current !== current) return;
      lastX = pointer.clientX; lastY = pointer.clientY;
      if (!current.moved && Math.hypot(lastX - current.startX, lastY - current.startY) < 4) return;
      current.moved = true;
      proposalAt(lastX, lastY);
    };
    const up = (pointer: globalThis.PointerEvent) => {
      if (pointer.pointerId !== current.pointerId || gesture.current !== current) return;
      const moved = current.moved;
      const result = moved ? pointer.pointerType === 'touch' ? current.proposal : proposalAt(pointer.clientX, pointer.clientY) : null;
      suppressClick.current = moved && !current.source.classList.contains('scheduler-touch-handle');
      clearGesture();
      if (result) place(result);
    };
    const cancel = (pointer: globalThis.PointerEvent) => { if (pointer.pointerId === current.pointerId) clearGesture(); };
    const keydown = (key: KeyboardEvent) => { if (key.key === 'Escape') { suppressClick.current = !current.source.classList.contains('scheduler-touch-handle'); clearGesture(); } };
    const lost = () => { if (gesture.current === current) clearGesture(); };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
    window.addEventListener('pointercancel', cancel);
    window.addEventListener('keydown', keydown);
    source.addEventListener('lostpointercapture', lost);
    current.stop = () => {
      cancelAnimationFrame(frame);
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      window.removeEventListener('pointercancel', cancel);
      window.removeEventListener('keydown', keydown);
      source.removeEventListener('lostpointercapture', lost);
    };
    frame = requestAnimationFrame(autoScroll);
  }
  const handle: SchedulerGridHandle = {
    beginAssignment: (event, assignment, durationMinutes) => begin(event, { kind: 'assignment', assignment, durationMinutes }),
    consumeClick: () => { const value = suppressClick.current; suppressClick.current = false; return value; },
  };
  useImperativeHandle(ref, () => handle);
  return <div className="scheduler-calendar">
    <label className="scheduler-day-picker">Selected day<select value={selectedDay} onChange={event => selectDay(event.target.value)}>{week.days.map(day => <option key={day.date} value={day.date}>{dayName(day.date)}</option>)}</select></label>
    <div className="scheduler-grid"><div className="scheduler-axis"><div className="scheduler-day-heading">Time</div><div style={{ height }}>{Array.from({ length: Math.ceil(height / 60) }, (_, index) => <span style={{ top: index * 60 }} key={index}>{displayClock(axisStart + index * 60)}</span>)}</div></div>
      {week.days.map(day => {
        const startMinute = intervalMinutes(day.interval.start), endMinute = intervalMinutes(day.interval.end) + (day.interval.nextDay ? 1440 : 0);
        const shown = preview?.date === day.date ? preview : null;
        return <section className={`scheduler-day ${day.date === selectedDay ? 'is-selected' : ''}`} key={day.date} aria-label={dayName(day.date)}>
          <div className="scheduler-day-heading"><strong>{dayName(day.date)}</strong><button aria-label={`Add session on ${day.date}`} onClick={() => addSession(day.date, startMinute)}>＋</button></div>
          <div className="scheduler-day-body" data-scheduler-date={day.date} style={{ height }}>
            <div className="scheduler-unavailable" style={{ top: 0, height: startMinute - axisStart }} /><div className="scheduler-unavailable" style={{ top: endMinute - axisStart, height: axisEnd - endMinute }} />
            {week.busy.map(busy => { const start = Math.max(axisStart, minuteOf(busy.start, day.date, week.settings.timeZone)), end = Math.min(axisEnd, minuteOf(busy.end, day.date, week.settings.timeZone)); return end > start ? <div className={`scheduler-block scheduler-busy ${conflicts.includes(busy.id) ? 'scheduler-conflict' : ''}`} key={busy.id} style={{ top: start - axisStart, height: Math.max(20, end - start) }}><strong>{busy.title || 'Google busy'}</strong><small>Google busy · {displayClock(start)}–{displayClock(end)}</small></div> : null; })}
            {week.sessions.filter(session => (session.actual?.date || session.date) === day.date && session.state !== 'canceled').map(session => {
              const interval = activeInterval(session);
              const top = interval ? minuteOf(interval.start, day.date, week.settings.timeZone) : startMinute;
              const end = interval ? minuteOf(interval.end, day.date, week.settings.timeZone) : top + 60;
              const started = session.plan ? Date.parse(session.plan.start) <= Date.now() : true;
              const canDelete = (!session.plan || !started) && (!session.actual || !session.ruleId);
              return <div className={`scheduler-block ${session.assignment.goalId ? 'scheduler-work' : 'scheduler-fixed'} ${session.attention || conflicts.includes(session.id) ? 'scheduler-conflict' : ''} ${session.actual?.status === 'skipped' ? 'scheduler-skipped' : ''}`} key={session.id} data-session-id={session.id} style={{ top: Math.max(0, top - axisStart), minHeight: 30, height: Math.max(30, end - top), borderLeftColor: session.assignment.color }}>
                <button className="scheduler-block-main" onPointerDown={interval ? event => begin(event, { kind: 'session', session }) : undefined} onClick={() => { if (!handle.consumeClick()) edit(session); }}><strong>{session.assignment.title}</strong>{session.assignment.goalTitle && <small>{session.assignment.goalTitle}</small>}<small>{displayClock(top)}–{displayClock(end)}</small><small>{session.attention || (session.actual ? `${session.actual.status} actual` : session.assignment.goalId ? 'Planned' : 'Commitment')}{session.ruleId ? ' · Weekly' : ''}</small>{session.actual?.outsideTimeline && <small>Outside original timeline</small>}</button>
                <div className="scheduler-block-actions"><button className="scheduler-icon-btn" aria-label={`Edit ${session.assignment.title}`} onClick={() => edit(session)}>✎</button>{canDelete && <button className="scheduler-icon-btn" aria-label={`Delete ${session.assignment.title}`} onClick={() => deleteSession(session)}>×</button>}</div>
                {interval && <><button className="scheduler-edge scheduler-edge-top" aria-label={`Change start time of ${session.assignment.title}`} onPointerDown={event => { event.preventDefault(); event.stopPropagation(); begin(event, { kind: 'resize', session, edge: 'start' }); }} onClick={() => { if (!handle.consumeClick()) edit(session); }} /><button className="scheduler-edge scheduler-edge-bottom" aria-label={`Change end time of ${session.assignment.title}`} onPointerDown={event => { event.preventDefault(); event.stopPropagation(); begin(event, { kind: 'resize', session, edge: 'end' }); }} onClick={() => { if (!handle.consumeClick()) edit(session); }} /><button className="scheduler-touch-handle scheduler-block-handle" aria-label={`Drag ${session.assignment.title}`} onPointerDown={event => { event.preventDefault(); event.stopPropagation(); begin(event, { kind: 'session', session }); }}>⋮⋮</button></>}
              </div>;
            })}
            {shown && <div className={`scheduler-drop-preview ${shown.result.kind === 'invalid' ? 'is-invalid' : shown.result.conflictIds.length ? 'is-conflicting' : 'is-valid'}`} data-date={shown.date} data-start={shown.start} data-end={shown.end} data-duration-minutes={shown.duration} style={{ top: shown.top, height: shown.height }} role="status"><div className="scheduler-preview-label"><strong>{shown.date} · {shown.start}–{shown.end}</strong><span>{shown.duration} min · {shown.result.kind === 'invalid' ? shown.result.message : shown.result.conflictIds.length ? `Overlaps ${shown.result.conflictIds.length} known reservation(s). Save will be checked again.` : 'Ready to place. Save will be checked again.'}</span></div></div>}
          </div>
        </section>;
      })}
    </div>
    {outside && <div className="scheduler-preview-outside" style={{ left: outside.x, top: outside.y }} role="status">Place inside a scheduling day.</div>}
  </div>;
});
