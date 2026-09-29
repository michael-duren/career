const SVG_NS = 'http://www.w3.org/2000/svg';
const LANGUAGES = ['cpp', 'python', 'java', 'go'];
const LANGUAGE_NAMES = { cpp: 'C++', python: 'Python', java: 'Java', go: 'Go' };
const STORAGE_KEY = 'leetgrinder.lessonLanguage';
const SPEEDS = [0.5, 1, 2];
const mounted = new WeakMap();

export function createPlayback(frameCount, state = { index: 0, playing: false, speed: 1 }) {
  const count = Math.max(1, Math.trunc(frameCount) || 1);
  return {
    index: Math.min(count - 1, Math.max(0, Math.trunc(state.index) || 0)),
    playing: Boolean(state.playing) && state.index < count - 1,
    speed: SPEEDS.includes(state.speed) ? state.speed : 1,
  };
}

export function advancePlayback(state, action, frameCount) {
  const current = createPlayback(frameCount, state);
  const last = Math.max(0, Math.trunc(frameCount) - 1);
  switch (action.type) {
    case 'start': return { ...current, index: 0, playing: false };
    case 'stop': return { ...current, playing: false };
    case 'play': return { ...current, playing: current.index < last };
    case 'previous': return { ...current, index: Math.max(0, current.index - 1), playing: false };
    case 'next': return { ...current, index: Math.min(last, current.index + 1), playing: false };
    case 'tick': {
      if (!current.playing) return current;
      const index = Math.min(last, current.index + 1);
      return { ...current, index, playing: index < last };
    }
    case 'speed': return SPEEDS.includes(action.value) ? { ...current, speed: action.value } : current;
    default: return current;
  }
}

function make(doc, tag, className, parent, text) {
  const node = doc.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = String(text);
  parent.append(node);
  return node;
}

function svgNode(doc, tag, attrs, parent, text) {
  const node = doc.createElementNS(SVG_NS, tag);
  for (const [name, value] of Object.entries(attrs)) node.setAttribute(name, String(value));
  if (text !== undefined) node.textContent = String(text);
  parent.append(node);
  return node;
}

function identifier(value) {
  return String(value).replace(/[^a-zA-Z0-9_-]/g, '-');
}

function center(node) {
  return { x: node.x + node.width / 2, y: node.y + node.height / 2 };
}

function boundary(node, toward) {
  const c = center(node);
  const dx = toward.x - c.x;
  const dy = toward.y - c.y;
  if (!dx && !dy) return c;
  const scale = node.shape === 'circle'
    ? (node.width / 2) / Math.hypot(dx, dy)
    : Math.min(dx ? node.width / (2 * Math.abs(dx)) : Infinity,
      dy ? node.height / (2 * Math.abs(dy)) : Infinity);
  return { x: c.x + dx * scale, y: c.y + dy * scale };
}

function renderScene(doc, svg, scene, prefix) {
  svg.replaceChildren();
  svg.setAttribute('viewBox', `0 0 ${scene.width} ${scene.height}`);
  svg.setAttribute('aria-labelledby', `${prefix}-title ${prefix}-description`);
  svgNode(doc, 'title', { id: `${prefix}-title` }, svg, 'Algorithm diagram');
  svgNode(doc, 'desc', { id: `${prefix}-description` }, svg, scene.description);
  const defs = svgNode(doc, 'defs', {}, svg);
  const marker = svgNode(doc, 'marker', {
    id: `${prefix}-arrow`, viewBox: '0 0 10 10', refX: 9, refY: 5,
    markerWidth: 7, markerHeight: 7, orient: 'auto-start-reverse',
  }, defs);
  svgNode(doc, 'path', { d: 'M 0 0 L 10 5 L 0 10 z' }, marker);
  const byID = new Map((scene.nodes || []).map(node => [node.id, node]));
  for (const edge of scene.edges || []) {
    const from = byID.get(edge.from);
    const to = byID.get(edge.to);
    if (!from || !to) continue;
    const start = boundary(from, center(to));
    const end = boundary(to, center(from));
    const attrs = {
      x1: start.x, y1: start.y, x2: end.x, y2: end.y,
      class: `scene-edge scene-${edge.role || 'neutral'}`,
    };
    if (edge.directed) attrs['marker-end'] = `url(#${prefix}-arrow)`;
    svgNode(doc, 'line', attrs, svg);
    if (edge.label) svgNode(doc, 'text', {
      x: (start.x + end.x) / 2, y: (start.y + end.y) / 2 - 6,
      class: 'scene-edge-label', 'text-anchor': 'middle',
    }, svg, edge.label);
  }
  for (const node of scene.nodes || []) {
    const group = svgNode(doc, 'g', { class: `scene-node scene-${node.role || 'neutral'}` }, svg);
    if (node.shape === 'circle') {
      svgNode(doc, 'circle', {
        cx: node.x + node.width / 2, cy: node.y + node.height / 2, r: node.width / 2,
      }, group);
    } else {
      svgNode(doc, 'rect', { x: node.x, y: node.y, width: node.width, height: node.height }, group);
    }
    svgNode(doc, 'text', {
      x: node.x + node.width / 2, y: node.y + node.height / 2,
      'text-anchor': 'middle', 'dominant-baseline': 'middle',
    }, group, node.text);
  }
  for (const label of scene.labels || []) {
    svgNode(doc, 'text', { x: label.x, y: label.y, class: 'scene-label' }, svg, label.text);
  }
}

function storedLanguage(win) {
  try {
    const value = win.localStorage.getItem(STORAGE_KEY);
    return LANGUAGES.includes(value) ? value : 'python';
  } catch { return 'python'; }
}

function rememberLanguage(win, language) {
  try { win.localStorage.setItem(STORAGE_KEY, language); } catch { /* Storage is optional. */ }
}

export function mountExample(element) {
  if (mounted.has(element)) return mounted.get(element);
  const doc = element.ownerDocument;
  const win = doc.defaultView;
  const payload = element.querySelector('[data-player-data]');
  if (!payload) throw new Error('Lesson example is missing [data-player-data]');
  const example = JSON.parse(payload.content?.textContent || payload.textContent);
  const fallback = element.querySelector('[data-player-fallback]');
  if (!example.cases?.length || !example.variants?.length) throw new Error('Lesson example has no cases or variants');
  const prefix = `lesson-${identifier(example.id)}`;
  const listeners = [];
  let timer = null;
  let caseIndex = 0;
  let language = storedLanguage(win);
  if (!example.variants.some(v => v.language === language)) language = example.variants[0].language;
  let playback = createPlayback(example.cases[0].frames.length);
  const ui = make(doc, 'div', 'lesson-player', element);
  const caseLabel = make(doc, 'label', 'lesson-player-case-label', ui, 'Case');
  const caseSelect = make(doc, 'select', 'lesson-player-case', caseLabel);
  for (const traceCase of example.cases) {
    const option = make(doc, 'option', '', caseSelect, traceCase.label);
    option.value = traceCase.id;
  }
  const region = make(doc, 'div', 'lesson-player-diagram-region', ui);
  region.setAttribute('role', 'region');
  region.setAttribute('aria-label', 'Algorithm diagram, scrollable when needed');
  region.tabIndex = 0;
  const svg = svgNode(doc, 'svg', { role: 'img', class: 'lesson-player-svg' }, region);
  make(doc, 'p', 'lesson-player-diagram-hint', ui, 'Swipe diagram to see the rest.');
  const legend = make(doc, 'div', 'lesson-player-legend', ui);
  legend.setAttribute('aria-label', 'Diagram color and outline key');
  for (const role of ['neutral', 'active', 'visited', 'discarded', 'result']) {
    make(doc, 'span', `scene-${role}`, legend, role[0].toUpperCase() + role.slice(1));
  }
  const explanation = make(doc, 'p', 'lesson-player-explanation', ui);
  explanation.setAttribute('aria-live', 'off');
  const variables = make(doc, 'table', 'lesson-player-variables', ui);
  const caption = make(doc, 'caption', '', variables, 'Current variables');
  const tbody = make(doc, 'tbody', '', variables);
  const status = make(doc, 'span', 'lesson-player-status', ui);
  status.setAttribute('aria-live', 'polite');
  status.setAttribute('aria-atomic', 'true');
  const controls = make(doc, 'div', 'lesson-player-controls', ui);
  const buttons = {};
  for (const [action, label] of Object.entries({ start: 'Start', previous: 'Previous', play: 'Play', stop: 'Stop', next: 'Next' })) {
    buttons[action] = make(doc, 'button', `lesson-player-${action}`, controls, label);
    buttons[action].type = 'button';
  }
  const speedLabel = make(doc, 'label', 'lesson-player-speed-label', controls, 'Speed');
  const speedSelect = make(doc, 'select', 'lesson-player-speed', speedLabel);
  for (const value of SPEEDS) {
    const option = make(doc, 'option', '', speedSelect, `${value}×`);
    option.value = String(value);
  }
  const tablist = make(doc, 'div', 'lesson-player-tabs', ui);
  tablist.setAttribute('role', 'tablist');
  tablist.setAttribute('aria-label', 'Programming language');
  const codePanel = make(doc, 'div', 'lesson-player-code-panel', ui);
  codePanel.id = `${prefix}-panel`;
  codePanel.setAttribute('role', 'tabpanel');
  codePanel.tabIndex = 0;
  const code = make(doc, 'code', 'lesson-player-code', make(doc, 'pre', '', codePanel));
  const fullSource = make(doc, 'details', 'lesson-player-full-source', codePanel);
  make(doc, 'summary', '', fullSource, 'Full runnable program');
  const fullCode = make(doc, 'code', '', make(doc, 'pre', '', fullSource));
  const copy = make(doc, 'button', 'lesson-player-copy', codePanel, 'Copy code');
  copy.type = 'button';
  const tabs = new Map();
  for (const variant of example.variants) {
    const tab = make(doc, 'button', 'lesson-player-tab', tablist, LANGUAGE_NAMES[variant.language] || variant.language);
    tab.type = 'button';
    tab.id = `${prefix}-tab-${variant.language}`;
    tab.setAttribute('role', 'tab');
    tab.setAttribute('aria-controls', codePanel.id);
    tabs.set(variant.language, tab);
  }
  const transcript = make(doc, 'details', 'lesson-player-transcript', ui);
  make(doc, 'summary', '', transcript, 'Full trace transcript');
  const transcriptBody = make(doc, 'ol', '', transcript);

  function on(target, event, handler) {
    target.addEventListener(event, handler);
    listeners.push(() => target.removeEventListener(event, handler));
  }
  function currentCase() { return example.cases[caseIndex]; }
  function syncTimer() {
    if (timer !== null) { win.clearTimeout(timer); timer = null; }
    if (playback.playing) {
      timer = win.setTimeout(() => {
        timer = null;
        dispatch({ type: 'tick' });
      }, 1000 / playback.speed);
    }
  }
  function renderCode(frame) {
    code.replaceChildren();
    const highlighted = new Set(frame.lines[language] || []);
    const source = example.sources?.[language] || '';
    const lines = source.replace(/\n$/, '').split('\n');
    const variant = example.variants.find(item => item.language === language);
    const first = variant?.algorithmStart || 1;
    const last = variant?.algorithmEnd || lines.length;
    fullCode.textContent = source;
    for (let i = first - 1; i < Math.min(last, lines.length); i++) {
      const line = make(doc, 'span', highlighted.has(i + 1) ? 'lesson-code-line is-active' : 'lesson-code-line', code);
      line.setAttribute('data-line', String(i + 1));
      line.textContent = `${i + 1}  ${lines[i]}`;
    }
  }
  function renderTranscript() {
    transcriptBody.replaceChildren();
    for (const frame of currentCase().frames) {
      const item = make(doc, 'li', '', transcriptBody, frame.explanation);
      for (const variable of frame.variables || []) {
        make(doc, 'span', 'lesson-player-transcript-variable', item, ` ${variable.name} = ${variable.value}`);
      }
    }
  }
  function render(manual = false) {
    const traceCase = currentCase();
    const frame = traceCase.frames[playback.index];
    const scenePrefix = `${prefix}-${identifier(traceCase.id)}`;
    renderScene(doc, svg, frame.scene, scenePrefix);
    explanation.textContent = frame.explanation;
    tbody.replaceChildren();
    for (const variable of frame.variables || []) {
      const row = make(doc, 'tr', '', tbody);
      make(doc, 'th', '', row, variable.name);
      make(doc, 'td', '', row, variable.value);
    }
    renderCode(frame);
    for (const [name, tab] of tabs) {
      const active = name === language;
      tab.setAttribute('aria-selected', String(active));
      tab.tabIndex = active ? 0 : -1;
    }
    codePanel.setAttribute('aria-labelledby', tabs.get(language).id);
    buttons.previous.disabled = playback.index === 0;
    buttons.next.disabled = playback.index === traceCase.frames.length - 1;
    buttons.play.disabled = playback.playing || buttons.next.disabled;
    buttons.stop.disabled = !playback.playing;
    speedSelect.value = String(playback.speed);
    status.textContent = manual ? `Step ${playback.index + 1} of ${traceCase.frames.length}: ${frame.explanation}` : '';
  }
  function dispatch(action, manual = false) {
    playback = advancePlayback(playback, action, currentCase().frames.length);
    render(manual);
    syncTimer();
  }
  for (const action of Object.keys(buttons)) on(buttons[action], 'click', () => dispatch({ type: action }, action === 'previous' || action === 'next' || action === 'start'));
  on(speedSelect, 'change', () => dispatch({ type: 'speed', value: Number(speedSelect.value) }));
  on(caseSelect, 'change', () => {
    const next = example.cases.findIndex(c => c.id === caseSelect.value);
    if (next < 0) return;
    caseIndex = next;
    playback = createPlayback(currentCase().frames.length, { speed: playback.speed });
    renderTranscript();
    render(true);
    syncTimer();
  });
  for (const [name, tab] of tabs) {
    on(tab, 'click', () => {
      language = name;
      rememberLanguage(win, language);
      playback = advancePlayback(playback, { type: 'stop' }, currentCase().frames.length);
      render();
      syncTimer();
    });
    on(tab, 'keydown', event => {
      if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
      event.preventDefault();
      const names = [...tabs.keys()];
      const at = names.indexOf(name);
      const next = event.key === 'Home' ? 0 : event.key === 'End' ? names.length - 1
        : (at + (event.key === 'ArrowRight' ? 1 : -1) + names.length) % names.length;
      tabs.get(names[next]).click();
      tabs.get(names[next]).focus();
    });
  }
  on(copy, 'click', async () => {
    try {
      await win.navigator.clipboard.writeText(example.sources?.[language] || '');
      status.textContent = `${LANGUAGE_NAMES[language] || language} code copied`;
    } catch {
      status.textContent = 'Copy unavailable; select the code to copy it.';
    }
  });
  on(doc, 'visibilitychange', () => {
    if (doc.hidden) dispatch({ type: 'stop' });
  });
  let ancestor = element.parentElement;
  while (ancestor) {
    if (ancestor.tagName === 'DETAILS') {
      const disclosure = ancestor;
      on(disclosure, 'toggle', () => { if (!disclosure.open) dispatch({ type: 'stop' }); });
    }
    ancestor = ancestor.parentElement;
  }
  on(win, 'pagehide', () => dispatch({ type: 'stop' }));
  if (fallback) fallback.hidden = true;
  renderTranscript();
  render();
  const controller = { destroy() {
    if (timer !== null) win.clearTimeout(timer);
    for (const remove of listeners) remove();
    ui.remove();
    if (fallback) fallback.hidden = false;
    mounted.delete(element);
  } };
  mounted.set(element, controller);
  return controller;
}

export function mountLessonExamples(root = document) {
  const elements = [...root.querySelectorAll('[data-lesson-example]')];
  if (root.getAttribute?.('data-lesson-example') != null) {
    elements.unshift(root);
  }
  const controllers = elements.map(mountExample);
  return () => { for (const controller of controllers) controller.destroy(); };
}

if (typeof document !== 'undefined') {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => mountLessonExamples(), { once: true });
  } else {
    mountLessonExamples();
  }
}
