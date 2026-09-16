import { useState } from 'react';
import { dayNumber, dayString, monthOffset, shiftGoal, workloadFor, type Goal } from '../lib/timeline';

const labelDate = (date: string) => new Date(`${date}T00:00:00Z`).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' });
function Calendar({ label, value, startDate, endDate, onChange }: { label: string; value: string; startDate: string; endDate: string; onChange: (date: string) => void }) {
  const [month, setMonth] = useState(value.slice(0, 7) + '-01');
  const first = dayNumber(month);
  const offset = new Date(`${month}T00:00:00Z`).getUTCDay();
  const length = dayNumber(monthOffset(month, 1)) - first;
  return <section className="goal-calendar" aria-label={`${label} calendar`}>
    <strong>{label}: {labelDate(value)}</strong>
    <div className="calendar-navigation">
      <button type="button" aria-label={`Previous month for ${label}`} disabled={month <= '1900-01-01'} onClick={() => setMonth(monthOffset(month, -1))}>←</button>
      <label><span className="sr-only">Month for {label}</span><select value={Number(month.slice(5, 7)) - 1} onChange={e => setMonth(`${month.slice(0, 4)}-${String(Number(e.target.value) + 1).padStart(2, '0')}-01`)}>{Array.from({ length: 12 }, (_, i) => <option key={i} value={i}>{new Date(Date.UTC(2026, i, 1)).toLocaleDateString('en-US', { month: 'long', timeZone: 'UTC' })}</option>)}</select></label>
      <label><span className="sr-only">Year for {label}</span><select value={month.slice(0, 4)} onChange={e => setMonth(`${e.target.value}${month.slice(4)}`)}>{Array.from({ length: 301 }, (_, i) => <option key={i} value={1900 + i}>{1900 + i}</option>)}</select></label>
      <button type="button" aria-label={`Next month for ${label}`} disabled={month >= '2200-12-01'} onClick={() => setMonth(monthOffset(month, 1))}>→</button>
    </div>
    <div className="calendar-days">
      {['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'].map(day => <span key={day}>{day}</span>)}
      {Array.from({ length: offset }, (_, i) => <span key={`blank-${i}`} />)}
      {Array.from({ length }, (_, i) => {
        const date = dayString(first + i);
        return <button key={date} type="button" aria-label={`${label}: ${labelDate(date)}`} aria-pressed={date === value} className={date >= startDate && date <= endDate ? 'in-range' : ''} onClick={() => onChange(date)}>{i + 1}</button>;
      })}
    </div>
  </section>;
}
export function GoalSchedule({ draft, goals, onChange }: { draft: Goal; goals: Goal[]; onChange: (goal: Goal) => void }) {
  const { others, segments, peakHours, peakCount } = workloadFor(goals, draft);
  const start = dayNumber(draft.startDate), end = dayNumber(draft.endDate);
  const padding = Math.max(7, Math.ceil((end - start + 1) * .1));
  const viewStart = start - padding, viewEnd = end + padding + 1;
  const nearby = goals.filter(g => g.id !== draft.id && dayNumber(g.startDate) < viewEnd && dayNumber(g.endDate) >= viewStart).sort((a, b) => a.startDate.localeCompare(b.startDate));
  const position = (day: number) => `${(day - viewStart) / (viewEnd - viewStart) * 100}%`;
  const width = (from: number, to: number) => `${(to - from) / (viewEnd - viewStart) * 100}%`;
  return <section className="goal-schedule">
    <div className="goal-calendars">
      <Calendar key={`start-${draft.startDate.slice(0, 7)}`} label="Start date" value={draft.startDate} startDate={draft.startDate} endDate={draft.endDate} onChange={date => onChange(shiftGoal(draft, dayNumber(date) - start, 'move'))} />
      <Calendar key={`end-${draft.endDate.slice(0, 7)}`} label="End date" value={draft.endDate} startDate={draft.startDate} endDate={draft.endDate} onChange={date => onChange({ ...draft, endDate: date, startDate: date < draft.startDate ? date : draft.startDate })} />
    </div>
    <div className="timeline-ranges">{[1, 2, 3].map(months => <button key={months} type="button" onClick={() => onChange({ ...draft, endDate: monthOffset(draft.startDate, months) > '2200-12-31' ? '2200-12-31' : monthOffset(draft.startDate, months) })}>{months} {months === 1 ? 'month' : 'months'}</button>)}<span className="timeline-help">{end - start + 1} days · Changing the start keeps the duration.</span></div>
    <div className="goal-dates">
      <label>Expected hours per day<input type="number" required min="0" max="24" step="any" value={draft.dailyHours ?? ''} placeholder="Set an estimate" onChange={e => onChange({ ...draft, dailyHours: e.target.value === '' ? undefined : Number(e.target.value) })} /></label>
    </div>
    <p className="timeline-help">Estimates apply every calendar day. The preview below shows combined expected hours across overlapping goals, including this goal.</p>
    <h3>Schedule preview</h3>
    <div className="schedule-summary" role="status"><strong>{others.length} existing {others.length === 1 ? 'goal overlaps' : 'goals overlap'} these dates</strong><span>Including this goal: up to {peakCount} at once · {Number(peakHours.toFixed(2))} h/day estimated</span></div>
    {segments.some(s => s.unknown > 0) && <p className="timeline-help">Some goals have no hour estimate. Workload totals are incomplete.</p>}
    <div className="schedule-preview" aria-label="Nearby goals timeline">
      <div className="schedule-scale"><span>{labelDate(dayString(viewStart))}</span><span>{labelDate(dayString(viewEnd - 1))}</span></div>
      {[draft, ...nearby].map(goal => {
        const from = Math.max(viewStart, dayNumber(goal.startDate)), to = Math.min(viewEnd, dayNumber(goal.endDate) + 1);
        return <div className="schedule-row" key={goal.id}>
          <div className="schedule-row-label"><strong>{goal.id === draft.id ? 'This goal' : goal.title}</strong><span>{goal.dailyHours === undefined ? 'Hours not set' : `${goal.dailyHours} h/day`} · {labelDate(goal.startDate)} – {labelDate(goal.endDate)}</span></div>
          <div className="schedule-row-track"><div className="schedule-selection" style={{ left: position(start), width: width(start, end + 1) }} /><div className="schedule-bar" style={{ left: position(from), width: width(from, to), background: goal.color }} /></div>
          {goal.id !== draft.id && <button type="button" disabled={goal.endDate >= '2200-12-31' || dayNumber(goal.endDate) + 1 + end - start > dayNumber('2200-12-31')} onClick={() => onChange(shiftGoal(draft, dayNumber(goal.endDate) + 1 - start, 'move'))}>Start after this goal</button>}
        </div>;
      })}
    </div>
    <details className="workload-details" open><summary>Expected hours by date</summary>
      <div className="workload-table"><table><thead><tr><th>Dates</th><th>Goals</th><th>Expected hours/day</th></tr></thead><tbody>{segments.map(segment => <tr key={segment.start}><td>{labelDate(dayString(segment.start))} – {labelDate(dayString(segment.end - 1))}</td><td>{segment.count}</td><td>{Number(segment.hours.toFixed(2))}{segment.unknown ? ' + unestimated' : ''}</td></tr>)}</tbody></table></div>
    </details>
  </section>;
}
