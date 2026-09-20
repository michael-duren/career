import { useEffect, useMemo, useRef, useState } from 'react';
import dagre from '@dagrejs/dagre';
import type { Goal } from '../lib/timeline';
import { criticalPath, dependencyCycle, dependencyState, finished } from '../lib/goal-dependencies';
import '../styles/goal-graph.css';

type Point = { x: number; y: number };
const WIDTH = 260, HEIGHT = 150;
const curve = (a: Point, b: Point) => {
  const bend = Math.max(70, Math.abs(b.x - a.x) / 2);
  return `M ${a.x} ${a.y} C ${a.x + bend} ${a.y}, ${b.x - bend} ${b.y}, ${b.x} ${b.y}`;
};
export function GoalGraph() {
  const [goals, setGoals] = useState<Goal[]>([]);
  const [revisions, setRevisions] = useState<Record<string, string>>({});
  const [error, setError] = useState(''), [loaded, setLoaded] = useState(false);
  const [showFinished, setShowFinished] = useState(false), [target, setTarget] = useState('');
  const [positions, setPositions] = useState<Record<string, Point>>({});
  const [view, setView] = useState({ x: 36, y: 36, zoom: 1 });
  const [source, setSource] = useState(''), [pointer, setPointer] = useState<Point | null>(null);
  const [selected, setSelected] = useState<{ from: string; to: string } | null>(null);
  const [saving, setSaving] = useState(false), [message, setMessage] = useState('');
  const canvas = useRef<HTMLDivElement>(null);
  const busy = useRef(false);
  const gesture = useRef<{ kind: 'node' | 'pan' | 'connect'; id: string; start: Point; origin: Point; moved: boolean } | null>(null);
  useEffect(() => {
    const controller = new AbortController();
    fetch('/api/goals', { cache: 'no-store', signal: controller.signal }).then(async response => {
      const data = await response.json(); if (!response.ok) throw new Error(data.error || 'Could not load goals.');
      setGoals(data.goals); setRevisions(data.revisions); setLoaded(true);
    }).catch(e => { if (e.name !== 'AbortError') setError(e.message); });
    return () => controller.abort();
  }, []);
  const path = useMemo(() => criticalPath(goals, target), [goals, target]);
  const visible = useMemo(() => goals.filter(g => showFinished || !finished(g)), [goals, showFinished]);
  const layout = useMemo(() => {
    const graph = new dagre.graphlib.Graph().setGraph({ rankdir: 'LR', nodesep: 40, ranksep: 100, marginx: 24, marginy: 24 }).setDefaultEdgeLabel(() => ({}));
    for (const g of visible) graph.setNode(g.id, { width: WIDTH, height: HEIGHT });
    for (const g of visible) for (const id of g.dependsOn) if (graph.hasNode(id)) graph.setEdge(id, g.id);
    dagre.layout(graph);
    return Object.fromEntries(visible.map(g => [g.id, { x: graph.node(g.id).x - WIDTH / 2, y: graph.node(g.id).y - HEIGHT / 2 }]));
  }, [visible]);
  const points = { ...layout, ...positions };
  const edges = visible.flatMap(g => g.dependsOn.filter(id => layout[id]).map(id => ({ from: id, to: g.id })));
  const byId = new Map(goals.map(g => [g.id, g]));
  function local(clientX: number, clientY: number): Point {
    const rect = canvas.current!.getBoundingClientRect();
    return { x: (clientX - rect.left - view.x) / view.zoom, y: (clientY - rect.top - view.y) / view.zoom };
  }
  function fit(reset = false) {
    const nodes = visible.map(g => (reset ? layout : points)[g.id]);
    if (!nodes.length || !canvas.current) return;
    const left = Math.min(...nodes.map(p => p.x)), top = Math.min(...nodes.map(p => p.y));
    const width = Math.max(...nodes.map(p => p.x + WIDTH)) - left;
    const height = Math.max(...nodes.map(p => p.y + HEIGHT)) - top;
    const { clientWidth: w, clientHeight: h } = canvas.current;
    const zoom = Math.min(1.2, Math.max(.15, Math.min((w - 64) / width, (h - 64) / height)));
    setView({ x: (w - width * zoom) / 2 - left * zoom, y: (h - height * zoom) / 2 - top * zoom, zoom });
    if (reset) setPositions({});
  }
  function zoomBy(delta: number) {
    const w = canvas.current!.clientWidth / 2, h = canvas.current!.clientHeight / 2;
    setView(v => { const zoom = Math.max(.15, Math.min(2, v.zoom + delta)); return { x: w - (w - v.x) * zoom / v.zoom, y: h - (h - v.y) * zoom / v.zoom, zoom }; });
  }
  useEffect(() => {
    if (!loaded || !canvas.current) return;
    const resize = () => {
      if (canvas.current!.clientWidth < 640) setView({ x: 24, y: 24, zoom: .85 });
      else fit();
    };
    const observer = new ResizeObserver(resize);
    observer.observe(canvas.current);
    return () => observer.disconnect();
  }, [loaded, showFinished]);
  async function saveConnection(from: string, to: string, remove = false) {
    if (busy.current) return;
    setSource(''); setPointer(null); setError('');
    const goal = byId.get(to)!;
    if (!remove && (from === to || goal.dependsOn.includes(from))) { setError(from === to ? 'Choose a different goal to connect.' : 'These goals are already connected.'); return; }
    const draft = { ...goal, dependsOn: remove ? goal.dependsOn.filter(id => id !== from) : [...goal.dependsOn, from] };
    if (dependencyCycle(goals, draft)) { setError('That connection would create a loop. A prerequisite cannot depend on its own goal.'); return; }
    busy.current = true; setSaving(true); setMessage('Saving connection…');
    try {
      const response = await fetch('/api/goals', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal: draft, revision: revisions[to] ?? null }) });
      const data = await response.json();
      if (!response.ok) throw new Error(response.status === 409 ? 'This goal changed elsewhere. Reload the page before connecting it.' : data.error || 'Could not save connection. Try again.');
      // Keep cards in place while their dependency layout changes.
      setPositions(points);
      setGoals(current => current.map(g => g.id === to ? data.goal : g));
      setRevisions(current => ({ ...current, [to]: data.revision }));
      setSelected(null); setMessage(remove ? 'Connection removed' : 'Connection saved');
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not save connection.'); setMessage('Changes were not saved'); }
    finally { busy.current = false; setSaving(false); }
  }
  return <section className="goal-graph" aria-label="Goal graph">
    <div className="graph-toolbar">
      <div className="graph-count"><span className="graph-live-dot" />{visible.length} goals <span>· {edges.length} connections</span></div>
      <div className="graph-options">
        <label className="graph-toggle"><input type="checkbox" checked={showFinished} onChange={e => setShowFinished(e.target.checked)} /> Include finished</label>
        <select aria-label="Critical path target" value={target} onChange={e => setTarget(e.target.value)}><option value="">Highlight critical path</option>{goals.map(g => <option key={g.id} value={g.id}>{g.title}</option>)}</select>
        <button onClick={() => fit(true)}>Auto arrange</button>
      </div>
    </div>
    <div className="graph-instructions"><span>{source ? `Connecting from “${byId.get(source)?.title}” — choose another goal’s left handle.` : 'Drag cards to arrange. Drag from + to another card, or click + then its left handle.'}</span>{source && <button onClick={() => { setSource(''); setPointer(null); }}>Cancel</button>}<span className="graph-save" role="status">{message}</span></div>
    {error && <div className="graph-error" role="alert">{error}<button onClick={() => setError('')} aria-label="Dismiss error">×</button></div>}
    {target && <p className="graph-path" role="status">Critical path: {path.days} days · {path.ids.map(id => byId.get(id)?.title).join(' → ')}</p>}
    <div ref={canvas} className={`graph-canvas ${source ? 'is-connecting' : ''}`} tabIndex={0} aria-label="Dependency canvas. Drag empty space to pan. Use the zoom controls to resize. Escape cancels a connection."
      onKeyDown={e => { if (e.key === 'Escape') { setSource(''); setPointer(null); setSelected(null); } }}
      onPointerDown={e => {
        if (e.button !== 0 || (e.target as Element).closest('button,a,.graph-card,.graph-edge')) return;
        setSelected(null); gesture.current = { kind: 'pan', id: '', start: { x: e.clientX, y: e.clientY }, origin: view, moved: false };
        e.currentTarget.setPointerCapture(e.pointerId);
      }}
      onPointerMove={e => {
        const g = gesture.current;
        if (source) setPointer(local(e.clientX, e.clientY));
        if (!g) return;
        const dx = e.clientX - g.start.x, dy = e.clientY - g.start.y;
        if (Math.abs(dx) + Math.abs(dy) > 5) g.moved = true;
        if (g.kind === 'pan') setView(v => ({ ...v, x: g.origin.x + dx, y: g.origin.y + dy }));
        if (g.kind === 'node' && g.moved) setPositions(p => ({ ...p, [g.id]: { x: g.origin.x + dx / view.zoom, y: g.origin.y + dy / view.zoom } }));
      }}
      onPointerUp={e => {
        const g = gesture.current; gesture.current = null;
        if (g?.kind === 'connect' && g.moved) {
          const card = document.elementFromPoint(e.clientX, e.clientY)?.closest<HTMLElement>('[data-goal-node]');
          if (card) void saveConnection(g.id, card.dataset.goalNode!);
        }
        if (e.currentTarget.hasPointerCapture(e.pointerId)) e.currentTarget.releasePointerCapture(e.pointerId);
      }}
      onPointerCancel={() => { gesture.current = null; setSource(''); setPointer(null); }}>
      <div className="graph-world" style={{ transform: `translate(${view.x}px, ${view.y}px) scale(${view.zoom})` }}>
        <svg className="graph-edges" width="1" height="1" aria-label="Connections">
          <defs><marker id="goal-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8" fill="context-stroke" /></marker></defs>
          {edges.map(edge => {
            const a = points[edge.from], b = points[edge.to];
            const conflict = byId.get(edge.to)!.startDate <= byId.get(edge.from)!.endDate;
            const highlighted = path.ids.includes(edge.from) && path.ids.indexOf(edge.to) === path.ids.indexOf(edge.from) + 1;
            const chosen = selected?.from === edge.from && selected.to === edge.to;
            const d = curve({ x: a.x + WIDTH + 8, y: a.y + HEIGHT / 2 }, { x: b.x - 8, y: b.y + HEIGHT / 2 });
            return <g key={`${edge.from}-${edge.to}`} className="graph-edge" role="button" tabIndex={0} aria-label={`Connection: ${byId.get(edge.from)?.title} to ${byId.get(edge.to)?.title}`} onClick={() => setSelected(edge)} onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setSelected(edge); } }}>
              <path d={d} fill="none" stroke="transparent" strokeWidth="20" className="graph-edge-hit" />
              <path d={d} fill="none" stroke={chosen ? '#f4f4f5' : highlighted ? '#c4b5fd' : conflict ? '#fb923c' : '#71717a'} strokeWidth={chosen || highlighted ? 3 : 2} strokeDasharray={conflict ? '6 5' : undefined} markerEnd="url(#goal-arrow)" />
              <title>{byId.get(edge.from)?.title} → {byId.get(edge.to)?.title}{conflict ? ' · Dates overlap' : ''}. Click to remove connection.</title>
            </g>;
          })}
          {source && pointer && points[source] && <path d={curve({ x: points[source].x + WIDTH + 8, y: points[source].y + HEIGHT / 2 }, pointer)} fill="none" stroke="#a78bfa" strokeWidth="2" strokeDasharray="5 5" markerEnd="url(#goal-arrow)" />}
        </svg>
        {visible.map(goal => {
          const state = dependencyState(goal, goals), node = points[goal.id];
          return <article key={goal.id} data-goal-node={goal.id} className={`graph-card ${source === goal.id ? 'is-source' : ''} ${path.ids.includes(goal.id) ? 'on-path' : ''}`} style={{ left: node.x, top: node.y, width: WIDTH, height: HEIGHT, borderTopColor: goal.color }}
            onPointerDown={e => {
              if (e.button !== 0 || (e.target as Element).closest('button,a')) return;
              gesture.current = { kind: 'node', id: goal.id, start: { x: e.clientX, y: e.clientY }, origin: node, moved: false };
              canvas.current!.setPointerCapture(e.pointerId);
            }}>
            <button className="graph-port graph-port-in" aria-label={`Connect to ${goal.title}`} title="Connect a prerequisite to this goal" disabled={!source || saving} onClick={() => source && void saveConnection(source, goal.id)}>→</button>
            <div className="graph-card-meta"><span className={`graph-badge ${state.blocked ? 'blocked' : state.ready ? 'ready' : ''}`}>{state.blocked ? 'Blocked' : state.ready ? 'Ready' : 'Finished'}</span><span>{goal.status}</span></div>
            <a href={`/timeline?goal=${encodeURIComponent(goal.id)}`} title={goal.title}><strong>{goal.title}</strong><span aria-hidden="true">↗︎</span></a>
            <div className="graph-dates">{goal.startDate} <span>→</span> {goal.endDate}</div>
            <div className="graph-card-detail" title={state.blockers.map(g => g.title).join(', ')}>{state.blocked ? `Blocked by: ${[...state.blockers.map(g => g.title), ...state.missing].join(', ')}` : state.dropped.length ? `Dropped prerequisite: ${state.dropped.map(g => g.title).join(', ')}` : `${goal.steps.filter(s => s.done).length} / ${goal.steps.length} steps complete`}</div>
            <button className="graph-port graph-port-out" aria-label={`Connect from ${goal.title}`} title="Drag to another goal to create a dependency" disabled={saving}
              onPointerDown={e => {
                if (e.button !== 0) return;
                e.stopPropagation(); setSource(goal.id); setPointer(local(e.clientX, e.clientY));
                gesture.current = { kind: 'connect', id: goal.id, start: { x: e.clientX, y: e.clientY }, origin: node, moved: false };
                canvas.current!.setPointerCapture(e.pointerId);
              }} onClick={() => { if (!busy.current) { setSource(goal.id); setSelected(null); } }}>+</button>
          </article>;
        })}
      </div>
      {!loaded && <div className="graph-empty">{error ? 'Goals could not load. Reload to try again.' : 'Loading your goals…'}</div>}
      {loaded && !visible.length && <div className="graph-empty"><strong>{goals.length ? 'All goals are finished' : 'Start with a goal'}</strong><p>{goals.length ? 'Include finished goals to see their connections.' : 'Create goals in your timeline, then connect them here.'}</p><a href="/timeline">Open timeline ↗︎</a></div>}
      {selected && <div className="graph-selection"><span>{byId.get(selected.from)?.title} → {byId.get(selected.to)?.title}</span><button disabled={saving} onClick={() => void saveConnection(selected.from, selected.to, true)}>Remove connection</button><button onClick={() => setSelected(null)} aria-label="Close connection selection">×</button></div>}
      <div className="graph-navigation"><button onClick={() => zoomBy(-.15)} aria-label="Zoom out">−</button><span>{Math.round(view.zoom * 100)}%</span><button onClick={() => zoomBy(.15)} aria-label="Zoom in">+</button><button onClick={() => fit()}>Fit view</button></div>
    </div>
    <div className="graph-footer"><span>Prerequisite <span aria-hidden="true">⟶</span> Dependent</span><span><i /> Dates overlap</span><span>Drag the background to pan · Click an arrow to remove it</span></div>
  </section>;
}
