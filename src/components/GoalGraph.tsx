import { useEffect, useMemo, useState } from 'react';
import dagre from '@dagrejs/dagre';
import type { Goal } from '../lib/timeline';
import { criticalPath, dependencyState, finished } from '../lib/goal-dependencies';
export function GoalGraph() {
  const [goals, setGoals] = useState<Goal[]>([]), [error, setError] = useState(''), [loaded, setLoaded] = useState(false);
  const [showFinished, setShowFinished] = useState(false), [target, setTarget] = useState('');
  useEffect(() => { fetch('/api/goals', { cache: 'no-store' }).then(async response => { const data = await response.json(); if (!response.ok) throw new Error(data.error); setGoals(data.goals); setLoaded(true); }).catch(e => setError(e.message)); }, []);
  const path = useMemo(() => criticalPath(goals, target), [goals, target]);
  const visible = goals.filter(g => showFinished || !finished(g));
  const layout = useMemo(() => {
    const graph = new dagre.graphlib.Graph().setGraph({ rankdir: 'LR', nodesep: 32, ranksep: 70, marginx: 20, marginy: 20 }).setDefaultEdgeLabel(() => ({}));
    const shown = goals.filter(g => showFinished || !finished(g));
    for (const g of shown) graph.setNode(g.id, { width: 280, height: 200 });
    for (const g of shown) for (const id of g.dependsOn) if (graph.hasNode(id)) graph.setEdge(id, g.id);
    dagre.layout(graph); return graph;
  }, [goals, showFinished]);
  return <section aria-label="Goal graph">
    <p>Arrows run from prerequisites to dependents. Click a goal to edit it. Red dashed edges indicate date conflicts.</p>
    <label><input type="checkbox" checked={showFinished} onChange={e => setShowFinished(e.target.checked)} /> Show done and dropped goals</label>{' '}
    <label>Critical path target <select value={target} onChange={e => setTarget(e.target.value)}><option value="">Choose a target</option>{goals.map(g => <option key={g.id} value={g.id}>{g.title}</option>)}</select></label>
    {target && <p role="status">Critical path: {path.days} calendar days · {path.ids.map(id => goals.find(g => g.id === id)?.title).join(' → ')}. Includes finished goals even when hidden.</p>}
    {error && <p role="alert">{error}</p>}
    {!loaded && !error && <p>Loading goals…</p>}
    {loaded && !visible.length && <p>No goals to show.</p>}
    <div style={{ overflow: 'auto', marginTop: 20 }}>
      <svg aria-label="Dependency graph" width={Math.max(300, layout.graph().width ?? 300)} height={Math.max(220, layout.graph().height ?? 220)}>
        <defs><marker id="goal-arrow" markerWidth="8" markerHeight="8" refX="8" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8" fill="context-stroke" /></marker></defs>
        {layout.edges().map(edge => {
          const prerequisite = goals.find(g => g.id === edge.v)!, dependent = goals.find(g => g.id === edge.w)!;
          const conflict = dependent.startDate <= prerequisite.endDate;
          const highlighted = path.ids.indexOf(edge.w) === path.ids.indexOf(edge.v) + 1 && path.ids.includes(edge.v);
          return <polyline key={`${edge.v}-${edge.w}`} points={layout.edge(edge).points.map((p: { x: number; y: number }) => `${p.x},${p.y}`).join(' ')} fill="none" stroke={conflict ? '#f87171' : highlighted ? '#facc15' : '#94a3b8'} strokeWidth={highlighted ? 4 : 2} strokeDasharray={conflict ? '7 4' : undefined} markerEnd="url(#goal-arrow)"><title>{prerequisite.title} → {dependent.title}{conflict ? ': date conflict' : ''}</title></polyline>;
        })}
        {visible.map(goal => {
          const node = layout.node(goal.id), state = dependencyState(goal, goals);
          return <foreignObject key={goal.id} x={node.x - 140} y={node.y - 100} width={280} height={200}>
            <a href={`/timeline?goal=${encodeURIComponent(goal.id)}`} data-goal-node={goal.id} style={{ display: 'block', boxSizing: 'border-box', height: '100%', overflow: 'auto', border: `3px solid ${path.ids.includes(goal.id) ? '#facc15' : goal.color}`, borderRadius: 8, background: '#18181b', color: '#fafafa', padding: 12, textDecoration: 'none' }}>
              <strong style={{ color: goal.color }}>{goal.title}</strong><div>{goal.status} · <span style={{ borderRadius: 4, padding: '2px 5px', background: state.blocked ? '#78350f' : state.ready ? '#14532d' : '#3f3f46' }}>{state.ready ? 'Ready' : state.blocked ? 'Blocked' : 'Finished'}</span></div>
              <div>{goal.startDate} → {goal.endDate}</div>
              {state.blocked && <div>Blocked by: {state.blockers.map(g => g.title).join(', ')}{state.missing.join(', ')}</div>}
              {!!state.dropped.length && <div>Dropped prerequisite: {state.dropped.map(g => g.title).join(', ')}</div>}
              {!!state.conflicts.length && <div style={{ color: '#f87171' }}>Date conflict: {state.conflicts.map(g => g.title).join(', ')}</div>}
            </a>
          </foreignObject>;
        })}
      </svg>
    </div>
  </section>;
}
