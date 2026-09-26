import { useEffect, useMemo, useRef, useState, type DragEvent } from 'react';
import dagre from '@dagrejs/dagre';
import type { Goal } from '../lib/timeline';
import { criticalPath, dependencyCycle, dependencyState, finished } from '../lib/goal-dependencies';
import { applyStepMove, newMiniGoal, stepMove, type MiniGoal, type StepMove } from '../lib/mini-goals';
import '../styles/goal-graph.css';

type Point = { x: number; y: number };
const WIDTH = 260, HEIGHT = 150;
// Mini goal rows have fixed heights so card sizes are known before layout.
// Longer lists scroll inside the card (see .graph-minis ul max-height).
const MINI_BASE = 64, MINI_ROW = 28, MINI_EDIT = 66, MINI_LIST_MAX = 6 * MINI_ROW;
const MINIS_KEY = 'goal-graph:mini-goals';
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
  const [showMinis, setShowMinis] = useState(true), [adding, setAdding] = useState<Record<string, string>>({});
  const [editing, setEditing] = useState<{ step: string; title: string; to: string } | null>(null);
  const [dragging, setDragging] = useState(''), [drop, setDrop] = useState<{ goal: string; slot: number } | null>(null);
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
  // Read after hydration so the server-rendered markup matches the first client render.
  useEffect(() => { try { if (localStorage.getItem(MINIS_KEY) === 'hidden') setShowMinis(false); } catch { /* Storage is optional. */ } }, []);
  const path = useMemo(() => criticalPath(goals, target), [goals, target]);
  const visible = useMemo(() => goals.filter(g => showFinished || !finished(g)), [goals, showFinished]);
  const editingStep = editing?.step;
  const height = (g: Goal) => showMinis ? HEIGHT + MINI_BASE + Math.min(MINI_LIST_MAX, g.steps.length * MINI_ROW + (g.steps.some(s => s.id === editingStep) ? MINI_EDIT : 0)) : HEIGHT;
  const layout = useMemo(() => {
    const graph = new dagre.graphlib.Graph().setGraph({ rankdir: 'LR', nodesep: 40, ranksep: 100, marginx: 24, marginy: 24 }).setDefaultEdgeLabel(() => ({}));
    for (const g of visible) graph.setNode(g.id, { width: WIDTH, height: height(g) });
    for (const g of visible) for (const id of g.dependsOn) if (graph.hasNode(id)) graph.setEdge(id, g.id);
    dagre.layout(graph);
    return Object.fromEntries(visible.map(g => [g.id, { x: graph.node(g.id).x - WIDTH / 2, y: graph.node(g.id).y - height(g) / 2 }]));
  }, [visible, showMinis, editingStep]);
  const points = { ...layout, ...positions };
  const edges = visible.flatMap(g => g.dependsOn.filter(id => layout[id]).map(id => ({ from: id, to: g.id })));
  const byId = new Map(goals.map(g => [g.id, g]));
  function local(clientX: number, clientY: number): Point {
    const rect = canvas.current!.getBoundingClientRect();
    return { x: (clientX - rect.left - view.x) / view.zoom, y: (clientY - rect.top - view.y) / view.zoom };
  }
  function fit(reset = false) {
    const nodes = visible.map(g => ({ ...(reset ? layout : points)[g.id], h: height(g) }));
    if (!nodes.length || !canvas.current) return;
    const left = Math.min(...nodes.map(p => p.x)), top = Math.min(...nodes.map(p => p.y));
    const width = Math.max(...nodes.map(p => p.x + WIDTH)) - left;
    const bottom = Math.max(...nodes.map(p => p.y + p.h)) - top;
    const { clientWidth: w, clientHeight: h } = canvas.current;
    const zoom = Math.min(1.2, Math.max(.15, Math.min((w - 64) / width, (h - 64) / bottom)));
    setView({ x: (w - width * zoom) / 2 - left * zoom, y: (h - bottom * zoom) / 2 - top * zoom, zoom });
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
  }, [loaded, showFinished, showMinis]);
  /**
   * Runs one save at a time and merges the changed goals the server returns.
   * `pin` keeps every card where it is (connections); otherwise only manually
   * dragged cards stay put and the rest re-layout around changed card heights.
   * `preview` shows the change immediately and is reverted if the save fails.
   */
  async function mutate(pending: string, done: string, conflict: string, request: () => Promise<Response>, options: { pin?: boolean; preview?: (goals: Goal[]) => Goal[] } = {}) {
    if (busy.current) return false;
    busy.current = true; setSaving(true); setMessage(pending); setError('');
    const before = goals;
    if (options.preview) setGoals(options.preview);
    try {
      const response = await request();
      const data = await response.json();
      if (!response.ok) throw new Error(response.status === 409 ? conflict : data.error || 'Could not save. Try again.');
      const changed: Goal[] = data.goals ?? [data.goal];
      if (options.pin) setPositions(points);
      setGoals(current => current.map(g => changed.find(c => c.id === g.id) ?? g));
      setRevisions(current => ({ ...current, ...(data.revisions ?? { [data.goal.id]: data.revision }) }));
      setMessage(done); return true;
    } catch (e) {
      if (options.preview) setGoals(before);
      setError(e instanceof Error ? e.message : 'Could not save.'); setMessage('Changes were not saved'); return false;
    }
    finally { busy.current = false; setSaving(false); }
  }
  const postGoal = (goal: Goal) => () => fetch('/api/goals', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal, revision: revisions[goal.id] ?? null }) });
  async function saveConnection(from: string, to: string, remove = false) {
    if (busy.current) return;
    setSource(''); setPointer(null); setError('');
    const goal = byId.get(to)!;
    if (!remove && (from === to || goal.dependsOn.includes(from))) { setError(from === to ? 'Choose a different goal to connect.' : 'These goals are already connected.'); return; }
    const draft = { ...goal, dependsOn: remove ? goal.dependsOn.filter(id => id !== from) : [...goal.dependsOn, from] };
    if (dependencyCycle(goals, draft)) { setError('That connection would create a loop. A prerequisite cannot depend on its own goal.'); return; }
    if (await mutate('Saving connection…', remove ? 'Connection removed' : 'Connection saved', 'This goal changed elsewhere. Reload the page before connecting it.', postGoal(draft), { pin: true })) setSelected(null);
  }
  const saveSteps = (goal: Goal, steps: MiniGoal[], pending: string, done: string) =>
    mutate(pending, done, 'This goal changed elsewhere. Reload the page before editing its mini goals.', postGoal({ ...goal, steps }));
  async function addMini(goal: Goal) {
    const title = adding[goal.id]?.trim();
    if (!title) return;
    if (await saveSteps(goal, [...goal.steps, newMiniGoal(title)], 'Adding mini goal…', 'Mini goal added')) setAdding(current => ({ ...current, [goal.id]: '' }));
  }
  async function renameMini(goal: Goal, step: MiniGoal, title: string) {
    if (!title.trim()) { setError('A mini goal needs a name.'); return; }
    if (title.trim() === step.title) { setEditing(null); return; }
    if (await saveSteps(goal, goal.steps.map(s => s.id === step.id ? { ...s, title: title.trim() } : s), 'Renaming mini goal…', 'Mini goal renamed')) setEditing(null);
  }
  function deleteMini(goal: Goal, step: MiniGoal) {
    if (busy.current || !window.confirm(`Delete the mini goal “${step.title}”?`)) return;
    void saveSteps(goal, goal.steps.filter(s => s.id !== step.id), 'Deleting mini goal…', 'Mini goal deleted');
  }
  async function moveMini(move: StepMove | null) {
    if (!move) return false;
    return mutate('Moving mini goal…', move.from === move.to ? 'Mini goal reordered' : `Mini goal moved to “${byId.get(move.to)?.title}”`, 'A goal changed elsewhere. Reload the page before moving mini goals.',
      () => fetch('/api/goals/steps/move', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ stepId: move.stepId, from: move.from, fromRevision: revisions[move.from], to: move.to, toRevision: revisions[move.to], index: move.index }) }),
      { preview: current => applyStepMove(current, move) });
  }
  /** Keyboard and button reorders keep focus on the moved mini goal's control. */
  async function shiftMini(goal: Goal, step: MiniGoal, by: -1 | 1, focus: string) {
    const index = goal.steps.findIndex(s => s.id === step.id);
    const move = stepMove(goals, step.id, goal.id, by < 0 ? index - 1 : index + 2);
    const refocus = () => requestAnimationFrame(() => document.querySelector<HTMLElement>(`[data-mini-focus="${focus}"]`)?.focus());
    if (!move) return;
    refocus();
    await moveMini(move);
    refocus();
  }
  function endDrag() { setDragging(''); setDrop(null); }
  /** Whole cards accept dropped mini goals: before or after a hovered row, otherwise at the end. */
  const dropHandlers = (goal: Goal) => ({
    onDragOver: (e: DragEvent) => {
      if (!dragging) return;
      e.preventDefault(); e.dataTransfer.dropEffect = 'move';
      const row = (e.target as Element).closest<HTMLElement>('[data-mini-index]');
      const box = row?.getBoundingClientRect();
      const slot = row && box ? Number(row.dataset.miniIndex) + (e.clientY > box.top + box.height / 2 ? 1 : 0) : goal.steps.length;
      if (drop?.goal !== goal.id || drop.slot !== slot) setDrop({ goal: goal.id, slot });
    },
    onDragLeave: (e: DragEvent) => {
      if (dragging && !e.currentTarget.contains(e.relatedTarget as Node | null)) setDrop(current => current?.goal === goal.id ? null : current);
    },
    onDrop: (e: DragEvent) => {
      if (!dragging) return;
      e.preventDefault();
      const move = stepMove(goals, dragging, goal.id, drop?.goal === goal.id ? drop.slot : goal.steps.length);
      endDrag(); void moveMini(move);
    },
  });
  function miniGoals(goal: Goal) {
    const done = goal.steps.filter(s => s.done).length;
    const target = drop?.goal === goal.id ? drop.slot : -1;
    return <div className="graph-minis">
      <div className="graph-minis-head"><span>Mini goals</span><span>{done} / {goal.steps.length}</span></div>
      <ul className={target === goal.steps.length && goal.steps.length ? 'drop-end' : ''}>
        {goal.steps.map((step, index) => {
          const edit = editing?.step === step.id ? editing : null;
          const open = () => setEditing({ step: step.id, title: step.title, to: goal.id });
          return <li key={step.id} data-mini-index={index} className={`${step.done ? 'is-done' : ''} ${dragging === step.id ? 'is-dragging' : ''} ${target === index ? 'drop-before' : ''} ${edit ? 'is-editing' : ''}`}>
            {edit ? <form onSubmit={e => { e.preventDefault(); void renameMini(goal, step, edit.title); }}>
              <input autoFocus aria-label="Mini goal name" maxLength={500} value={edit.title} onChange={e => setEditing({ ...edit, title: e.target.value })} onKeyDown={e => { if (e.key === 'Escape') { e.stopPropagation(); setEditing(null); } }} />
              <div>
                <button type="button" data-mini-focus={`up:${step.id}`} aria-label={`Move ${step.title} up`} disabled={saving || index === 0} onClick={() => void shiftMini(goal, step, -1, `up:${step.id}`)}>↑</button>
                <button type="button" data-mini-focus={`down:${step.id}`} aria-label={`Move ${step.title} down`} disabled={saving || index === goal.steps.length - 1} onClick={() => void shiftMini(goal, step, 1, `down:${step.id}`)}>↓</button>
                <span />
                <button type="submit" disabled={saving}>Save</button><button type="button" onClick={() => setEditing(null)}>Cancel</button>
              </div>
              <div>
                <select aria-label={`Goal for ${step.title}`} value={edit.to} disabled={saving} onChange={e => setEditing({ ...edit, to: e.target.value })}>
                  {goals.map(g => <option key={g.id} value={g.id}>{g.id === goal.id ? 'Move to another goal…' : g.title}</option>)}
                </select>
                <button type="button" disabled={saving || edit.to === goal.id} onClick={() => { void moveMini(stepMove(goals, step.id, edit.to, Infinity)).then(ok => { if (ok) setEditing(null); }); }}>Move</button>
              </div>
            </form> : <>
              <span className="graph-mini-handle" role="button" tabIndex={0} draggable data-mini-focus={`handle:${step.id}`}
                aria-label={`Reorder ${step.title}. Drag this handle to reorder or move to another goal, or press the up and down arrow keys.`} title="Drag to reorder or move to another goal"
                onDragStart={e => {
                  if (busy.current) { e.preventDefault(); return; }
                  e.dataTransfer.setData('text/plain', step.id); e.dataTransfer.effectAllowed = 'move';
                  const row = e.currentTarget.closest('li');
                  if (row) e.dataTransfer.setDragImage(row, 12, 14);
                  setDragging(step.id);
                }}
                onDragEnd={endDrag}
                onKeyDown={e => { if (e.key === 'ArrowUp' || e.key === 'ArrowDown') { e.preventDefault(); void shiftMini(goal, step, e.key === 'ArrowUp' ? -1 : 1, `handle:${step.id}`); } }}>⋮⋮</span>
              <input type="checkbox" aria-label={`Mark ${step.title} ${step.done ? 'not done' : 'done'}`} checked={step.done} disabled={saving}
                onChange={e => void saveSteps(goal, goal.steps.map(s => s.id === step.id ? { ...s, done: e.target.checked } : s), 'Saving mini goal…', e.target.checked ? 'Mini goal done' : 'Mini goal reopened')} />
              <span title={step.title} onDoubleClick={open}>{step.title}</span>
              <button type="button" aria-label={`Edit ${step.title}`} title="Rename, reorder or move to another goal" onClick={open}>✎</button>
              <button type="button" aria-label={`Delete ${step.title}`} title="Delete mini goal" disabled={saving} onClick={() => deleteMini(goal, step)}>×</button>
            </>}
          </li>;
        })}
      </ul>
      <form className="graph-mini-add" onSubmit={e => { e.preventDefault(); void addMini(goal); }}>
        <input aria-label={`New mini goal for ${goal.title}`} placeholder="Add a mini goal" maxLength={500} value={adding[goal.id] ?? ''} onChange={e => setAdding(current => ({ ...current, [goal.id]: e.target.value }))} />
        <button type="submit" aria-label={`Add mini goal to ${goal.title}`} disabled={saving || !adding[goal.id]?.trim()}>+</button>
      </form>
    </div>;
  }
  return <section className="goal-graph" aria-label="Goal graph">
    <div className="graph-toolbar">
      <div className="graph-count"><span className="graph-live-dot" />{visible.length} goals <span>· {edges.length} connections</span></div>
      <div className="graph-options">
        <label className="graph-toggle"><input type="checkbox" checked={showFinished} onChange={e => setShowFinished(e.target.checked)} /> Include finished</label>
        <label className="graph-toggle"><input type="checkbox" checked={showMinis} onChange={e => { const on = e.target.checked; setShowMinis(on); setPositions({}); try { localStorage.setItem(MINIS_KEY, on ? 'shown' : 'hidden'); } catch { /* Storage is optional. */ } }} /> Show mini goals</label>
        <select aria-label="Critical path target" value={target} onChange={e => setTarget(e.target.value)}><option value="">Highlight critical path</option>{goals.map(g => <option key={g.id} value={g.id}>{g.title}</option>)}</select>
        <button onClick={() => fit(true)}>Auto arrange</button>
      </div>
    </div>
    <div className="graph-instructions"><span>{source ? `Connecting from “${byId.get(source)?.title}” — choose another goal’s left handle.` : `Drag cards to arrange. Drag from + to another card, or click + then its left handle.${showMinis ? ' Drag a mini goal’s ⋮⋮ handle to reorder it or move it to another goal.' : ''}`}</span>{source && <button onClick={() => { setSource(''); setPointer(null); }}>Cancel</button>}<span className="graph-save" role="status">{message}</span></div>
    {error && <div className="graph-error" role="alert">{error}<button onClick={() => setError('')} aria-label="Dismiss error">×</button></div>}
    {target && <p className="graph-path" role="status">Critical path: {path.days} days · {path.ids.map(id => byId.get(id)?.title).join(' → ')}</p>}
    <div ref={canvas} className={`graph-canvas ${source ? 'is-connecting' : ''}`} tabIndex={0} aria-label="Dependency canvas. Drag empty space to pan. Use the zoom controls to resize. Escape cancels a connection."
      onKeyDown={e => { if (e.key === 'Escape') { setSource(''); setPointer(null); setSelected(null); } }}
      onPointerDown={e => {
        if (e.button !== 0 || (e.target as Element).closest('button,a,input,select,.graph-card,.graph-edge')) return;
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
          return <article key={goal.id} data-goal-node={goal.id} className={`graph-card ${source === goal.id ? 'is-source' : ''} ${path.ids.includes(goal.id) ? 'on-path' : ''} ${drop?.goal === goal.id ? 'is-drop-target' : ''}`} style={{ left: node.x, top: node.y, width: WIDTH, height: height(goal), borderTopColor: goal.color }} {...(showMinis ? dropHandlers(goal) : {})}
            onPointerDown={e => {
              if (e.button !== 0 || (e.target as Element).closest('button,a,input,select,.graph-minis')) return;
              gesture.current = { kind: 'node', id: goal.id, start: { x: e.clientX, y: e.clientY }, origin: node, moved: false };
              canvas.current!.setPointerCapture(e.pointerId);
            }}>
            <button className="graph-port graph-port-in" aria-label={`Connect to ${goal.title}`} title="Connect a prerequisite to this goal" disabled={!source || saving} onClick={() => source && void saveConnection(source, goal.id)}>→</button>
            <div className="graph-card-meta"><span className={`graph-badge ${state.blocked ? 'blocked' : state.ready ? 'ready' : ''}`}>{state.blocked ? 'Blocked' : state.ready ? 'Ready' : 'Finished'}</span><span>{goal.status}</span></div>
            <a href={`/timeline?goal=${encodeURIComponent(goal.id)}`} title={goal.title}><strong>{goal.title}</strong><span aria-hidden="true">↗︎</span></a>
            <div className="graph-dates">{goal.startDate} <span>→</span> {goal.endDate}</div>
            <div className="graph-card-detail" title={state.blockers.map(g => g.title).join(', ')}>{state.blocked ? `Blocked by: ${[...state.blockers.map(g => g.title), ...state.missing].join(', ')}` : state.dropped.length ? `Dropped prerequisite: ${state.dropped.map(g => g.title).join(', ')}` : showMinis ? '' : `${goal.steps.filter(s => s.done).length} / ${goal.steps.length} mini goals complete`}</div>
            {showMinis && miniGoals(goal)}
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
