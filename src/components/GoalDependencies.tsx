import type { Goal } from '../lib/timeline';
import { dependencyCycle, dependencyState, statusSuggestions } from '../lib/goal-dependencies';
import { localDate } from '../lib/workspace';
export function GoalDependencies({ draft, goals, onChange }: { draft: Goal; goals: Goal[]; onChange: (g: Goal) => void }) {
  const state = dependencyState(draft, goals);
  return <section aria-label="Goal dependencies">
    <label>Status<select value={draft.status} onChange={e => onChange({ ...draft, status: e.target.value as Goal['status'] })}>{['planned', 'active', 'done', 'dropped'].map(s => <option key={s}>{s}</option>)}</select></label>
    <label>Depends on<select multiple value={draft.dependsOn} onChange={e => onChange({ ...draft, dependsOn: Array.from(e.target.selectedOptions, o => o.value) })}>
      {goals.filter(g => g.id !== draft.id).map(g => <option key={g.id} value={g.id} disabled={!!dependencyCycle(goals, { ...draft, dependsOn: [...draft.dependsOn.filter(id => id !== g.id), g.id] })}>{g.title}</option>)}
    </select></label>
    <p>Choose prerequisite goals. Disabled choices would create a cycle.</p>
    {state.conflicts.map(g => <p key={g.id} role="status">Date conflict: {g.title} ends {g.endDate}, on or after this goal starts.</p>)}
    {state.dropped.map(g => <p key={g.id} role="status">Dropped prerequisite: {g.title}. Review whether this plan still works.</p>)}
    {statusSuggestions(draft, localDate()).map(s => <button key={s.status} type="button" onClick={() => onChange({ ...draft, status: s.status })}>{s.label}</button>)}
  </section>;
}
