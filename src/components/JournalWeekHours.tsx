import { useEffect, useState } from 'react';
import { formatHours, type GoalHours } from '../lib/goal-progress';
import { TRACK_META } from '../lib/progress';
import type { Goal } from '../lib/timeline';

/**
 * Hours for one journal week. Goal hours come from work recorded in the weekly
 * scheduler; the older per-track hours are kept read-only as history.
 */
export function JournalWeekHours({ weekStart, tracks }: { weekStart: string; tracks: Record<string, number> }) {
  const [rows, setRows] = useState<{ id: string; title: string; color: string; hours: number }[]>();
  const [error, setError] = useState('');
  useEffect(() => {
    let live = true;
    const load = () => Promise.all([fetch('/api/goals', { cache: 'no-store' }), fetch('/api/scheduler/goal-hours', { cache: 'no-store' })])
      .then(async ([goalsResponse, hoursResponse]) => {
        if (!goalsResponse.ok || !hoursResponse.ok) throw new Error('Could not load goal hours.');
        const goals: Goal[] = (await goalsResponse.json()).goals;
        const hours: Record<string, GoalHours> = (await hoursResponse.json()).goals;
        if (!live) return;
        setRows(goals.flatMap(goal => hours[goal.id]?.weekHours[weekStart] ? [{ id: goal.id, title: goal.title, color: goal.color, hours: hours[goal.id].weekHours[weekStart] }] : []));
        setError('');
      })
      .catch(e => { if (live) setError(e instanceof Error ? e.message : 'Could not load goal hours.'); });
    void load();
    window.addEventListener('workspace-saved', load);
    return () => { live = false; window.removeEventListener('workspace-saved', load); };
  }, [weekStart]);
  const legacy = Object.entries(tracks).filter(([, hours]) => hours > 0);
  return <div className="space-y-2 text-sm">
    <p className="text-zinc-400">Goal hours this week, from the weekly scheduler. <a className="text-blue-400 hover:underline" href="/">Log work →</a></p>
    {error ? <p role="alert" className="text-red-300">{error}</p> : !rows ? <p role="status" className="text-zinc-500">Loading…</p> : rows.length === 0 ? <p className="text-zinc-500">No goal work recorded this week.</p>
      : <ul className="grid grid-cols-2 gap-2">{rows.map(row => <li key={row.id} className="rounded bg-zinc-800/60 px-3 py-2"><span style={{ color: row.color }}>{row.title}</span> · {formatHours(row.hours)}h</li>)}</ul>}
    {legacy.length > 0 && <p className="text-xs text-zinc-500">Earlier track hours: {legacy.map(([key, hours]) => `${TRACK_META[key]?.name ?? key} ${formatHours(hours)}h`).join(' · ')}</p>}
  </div>;
}
