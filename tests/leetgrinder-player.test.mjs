import test from 'node:test';
import assert from 'node:assert/strict';
import { createPlayback, advancePlayback, mountExample, mountLessonExamples } from '../internal/leetgrinder/lesson_player.js';

test('playback starts paused and Start resets without playing', () => {
  assert.deepEqual(createPlayback(3), { index: 0, playing: false, speed: 1 });
  assert.deepEqual(advancePlayback({ index: 2, playing: true, speed: 2 }, { type: 'start' }, 3),
    { index: 0, playing: false, speed: 2 });
});

test('Stop pauses in place and Play continues from that frame', () => {
  const stopped = advancePlayback({ index: 1, playing: true, speed: 1 }, { type: 'stop' }, 3);
  assert.deepEqual(stopped, { index: 1, playing: false, speed: 1 });
  assert.deepEqual(advancePlayback(stopped, { type: 'play' }, 3),
    { index: 1, playing: true, speed: 1 });
});

test('manual steps pause and clamp at endpoints', () => {
  assert.deepEqual(advancePlayback({ index: 0, playing: true, speed: 1 }, { type: 'previous' }, 3),
    { index: 0, playing: false, speed: 1 });
  assert.deepEqual(advancePlayback({ index: 2, playing: true, speed: 1 }, { type: 'next' }, 3),
    { index: 2, playing: false, speed: 1 });
});

test('ticks stop at final frame and Play there does nothing', () => {
  const last = advancePlayback({ index: 1, playing: true, speed: 1 }, { type: 'tick' }, 3);
  assert.deepEqual(last, { index: 2, playing: false, speed: 1 });
  assert.deepEqual(advancePlayback(last, { type: 'play' }, 3), last);
  assert.deepEqual(advancePlayback(last, { type: 'tick' }, 3), last);
});

test('speed changes preserve frame and playback state', () => {
  assert.deepEqual(advancePlayback({ index: 1, playing: true, speed: 1 }, { type: 'speed', value: 0.5 }, 3),
    { index: 1, playing: true, speed: 0.5 });
  assert.deepEqual(advancePlayback({ index: 1, playing: true, speed: 1 }, { type: 'speed', value: 9 }, 3),
    { index: 1, playing: true, speed: 1 });
});

class FakeElement {
  constructor(tagName, ownerDocument) {
    this.tagName = tagName.toUpperCase();
    this.ownerDocument = ownerDocument;
    this.children = [];
    this.attributes = new Map();
    this.listeners = new Map();
    this.parentElement = null;
    this._text = '';
    this.hidden = false;
    this.disabled = false;
  }
  append(...nodes) {
    for (const node of nodes) {
      node.parentElement = this;
      this.children.push(node);
    }
  }
  replaceChildren(...nodes) { this.children = []; this._text = ''; this.append(...nodes); }
  remove() { this.parentElement.children = this.parentElement.children.filter(node => node !== this); }
  set textContent(value) { this.children = []; this._text = String(value); }
  get textContent() { return this._text + this.children.map(node => node.textContent).join(''); }
  setAttribute(name, value) { this.attributes.set(name, String(value)); }
  getAttribute(name) { return this.attributes.get(name) ?? null; }
  addEventListener(name, fn) {
    if (!this.listeners.has(name)) this.listeners.set(name, new Set());
    this.listeners.get(name).add(fn);
  }
  removeEventListener(name, fn) { this.listeners.get(name)?.delete(fn); }
  fire(name, properties = {}) {
    const event = { key: undefined, preventDefault() {}, ...properties };
    for (const fn of this.listeners.get(name) || []) fn(event);
  }
  click() { this.fire('click'); }
  focus() { this.focused = true; }
  querySelectorAll(selector) {
    const match = node => selector.startsWith('[data-')
      ? node.attributes.has(selector.slice(1, -1))
      : selector.startsWith('.')
        ? (node.className || '').split(' ').includes(selector.slice(1))
        : node.tagName.toLowerCase() === selector.toLowerCase();
    const result = [];
    const visit = node => {
      for (const child of node.children) {
        if (match(child)) result.push(child);
        visit(child);
      }
    };
    visit(this);
    return result;
  }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
}

function fakeDocument() {
  const timers = new Map();
  let nextTimer = 0;
  const document = new FakeElement('document');
  document.ownerDocument = document;
  document.hidden = false;
  document.createElement = tag => new FakeElement(tag, document);
  document.createElementNS = (_namespace, tag) => document.createElement(tag);
  document.createTextNode = value => {
    const node = document.createElement('#text');
    node.textContent = value;
    return node;
  };
  const store = new Map();
  document.defaultView = {
    localStorage: { getItem: key => store.get(key), setItem: (key, value) => store.set(key, value) },
    setTimeout(fn, delay) { const id = ++nextTimer; timers.set(id, { fn, delay }); return id; },
    clearTimeout(id) { timers.delete(id); },
    listeners: new Map(),
    addEventListener: FakeElement.prototype.addEventListener,
    removeEventListener: FakeElement.prototype.removeEventListener,
    fire: FakeElement.prototype.fire,
  };
  document.runTimer = () => {
    const [id, timer] = timers.entries().next().value;
    timers.delete(id);
    timer.fn();
  };
  document.timers = timers;
  document.store = store;
  return document;
}

function example(id = 'day-01-lookup') {
  const scene = description => ({
    width: 200, height: 80, description,
    nodes: [
      { id: 'a', shape: 'rect', x: 10, y: 10, width: 40, height: 30, text: '</script>', role: 'active' },
      { id: 'b', shape: 'circle', x: 120, y: 10, width: 30, height: 30, text: 'B', role: 'result' },
    ],
    edges: [{ id: 'e', from: 'a', to: 'b', label: 'next', directed: true, role: 'active' }],
    labels: [{ id: 'l', x: 10, y: 70, text: 'index 0' }],
  });
  const frames = [0, 1, 2].map(index => ({
    event: `step-${index}`, explanation: `Frame ${index} </script>`,
    variables: [{ name: 'i', value: String(index) }],
    lines: { cpp: [index + 1], python: [index + 1], java: [index + 1], go: [index + 1] },
    scene: scene(`Scene ${index}`),
  }));
  return {
    id,
    variants: ['cpp', 'python', 'java', 'go'].map(language => ({ language, file: `main.${language}` })),
    sources: Object.fromEntries(['cpp', 'python', 'java', 'go'].map(language => [language, 'one\ntwo\nthree\n'])),
    cases: [
      { id: 'normal', label: 'Normal', frames },
      { id: 'edge', label: 'Edge', frames: frames.slice(0, 2) },
    ],
  };
}

function host(document, data) {
  const root = document.createElement('section');
  root.setAttribute('data-lesson-example', '');
  const payload = document.createElement('script');
  payload.setAttribute('data-player-data', '');
  payload.textContent = JSON.stringify(data);
  root.append(payload);
  const fallback = document.createElement('div');
  fallback.setAttribute('data-player-fallback', '');
  root.append(fallback);
  document.append(root);
  return { root, fallback };
}

test('player updates frame, code lines and SVG text without parsing hostile strings', () => {
  const document = fakeDocument();
  const { root, fallback } = host(document, example());
  const controller = mountExample(root);
  assert.equal(fallback.hidden, true);
  assert.equal(root.querySelector('.lesson-player-explanation').textContent, 'Frame 0 </script>');
  assert.equal(root.querySelectorAll('script').length, 1);
  assert.equal(root.querySelector('svg').querySelector('desc').textContent, 'Scene 0');
  root.querySelector('.lesson-player-next').click();
  assert.equal(root.querySelector('.lesson-player-explanation').textContent, 'Frame 1 </script>');
  assert.equal(root.querySelector('.is-active').getAttribute('data-line'), '2');
  assert.equal(root.querySelector('.lesson-player-variables').querySelector('td').textContent, '1');
  controller.destroy();
  assert.equal(fallback.hidden, false);
  assert.equal(root.querySelector('.lesson-player'), null);
});

test('one timer survives repeated Play and speed changes; language preserves frame then pauses', () => {
  const document = fakeDocument();
  const { root } = host(document, example());
  mountExample(root);
  const play = root.querySelector('.lesson-player-play');
  play.click();
  play.click();
  assert.equal(document.timers.size, 1);
  const speed = root.querySelector('.lesson-player-speed');
  speed.value = '2';
  speed.fire('change');
  assert.equal(document.timers.size, 1);
  assert.equal([...document.timers.values()][0].delay, 500);
  document.runTimer();
  assert.equal(root.querySelector('.lesson-player-variables').querySelector('td').textContent, '1');
  const cpp = root.querySelectorAll('.lesson-player-tab')[0];
  cpp.click();
  assert.equal(document.timers.size, 0);
  assert.equal(document.store.get('leetgrinder.lessonLanguage'), 'cpp');
  assert.equal(root.querySelector('.is-active').getAttribute('data-line'), '2');
  assert.equal(cpp.getAttribute('aria-selected'), 'true');
});

test('case switching resets, disclosure closing pauses, and examples remain independent', () => {
  const document = fakeDocument();
  const disclosure = document.createElement('details');
  disclosure.open = true;
  document.append(disclosure);
  const first = host(document, example('first')).root;
  first.remove();
  disclosure.append(first);
  const second = host(document, example('second')).root;
  const cleanup = mountLessonExamples(document);
  first.querySelector('.lesson-player-play').click();
  second.querySelector('.lesson-player-play').click();
  assert.equal(document.timers.size, 2);
  disclosure.open = false;
  disclosure.fire('toggle');
  assert.equal(document.timers.size, 1);
  const cases = second.querySelector('.lesson-player-case');
  cases.value = 'edge';
  cases.fire('change');
  assert.equal(document.timers.size, 0);
  assert.equal(second.querySelector('.lesson-player-explanation').textContent, 'Frame 0 </script>');
  assert.equal(first.querySelector('svg').querySelector('marker').getAttribute('id'), 'lesson-first-normal-arrow');
  assert.equal(second.querySelector('svg').querySelector('marker').getAttribute('id'), 'lesson-second-edge-arrow');
  cleanup();
  assert.equal(first.querySelector('.lesson-player'), null);
  assert.equal(second.querySelector('.lesson-player'), null);
});

test('repeated mounts reuse one controller and destruction clears timers', () => {
  const document = fakeDocument();
  const { root } = host(document, example());
  const first = mountExample(root);
  assert.equal(mountExample(root), first);
  assert.equal(root.querySelectorAll('.lesson-player').length, 1);
  root.querySelector('.lesson-player-play').click();
  first.destroy();
  assert.equal(document.timers.size, 0);
});

test('keyboard tabs select the next language and blocked storage falls back to Python', () => {
  const document = fakeDocument();
  Object.defineProperty(document.defaultView, 'localStorage', { get() { throw new Error('blocked'); } });
  const { root } = host(document, example());
  mountExample(root);
  const tabs = root.querySelectorAll('.lesson-player-tab');
  assert.equal(tabs[1].getAttribute('aria-selected'), 'true');
  tabs[1].fire('keydown', { key: 'ArrowRight' });
  assert.equal(tabs[2].getAttribute('aria-selected'), 'true');
  assert.equal(tabs[2].focused, true);
  assert.equal(root.querySelector('.lesson-player-code-panel').getAttribute('aria-labelledby'), tabs[2].id);
});

test('visibility pauses and the final timer leaves playback stopped', () => {
  const document = fakeDocument();
  const { root } = host(document, example());
  mountExample(root);
  root.querySelector('.lesson-player-play').click();
  document.hidden = true;
  document.fire('visibilitychange');
  assert.equal(document.timers.size, 0);
  assert.equal(root.querySelector('.lesson-player-stop').disabled, true);
  document.hidden = false;
  root.querySelector('.lesson-player-play').click();
  document.runTimer();
  document.runTimer();
  assert.equal(document.timers.size, 0);
  assert.equal(root.querySelector('.lesson-player-play').disabled, true);
  assert.equal(root.querySelector('.lesson-player-next').disabled, true);
  root.querySelector('.lesson-player-start').click();
  assert.equal(root.querySelector('.lesson-player-previous').disabled, true);
  assert.equal(root.querySelector('.lesson-player-explanation').textContent, 'Frame 0 </script>');
});

test('player exposes source copy and a visible scene-role legend', () => {
  const document = fakeDocument();
  const { root } = host(document, example());
  mountExample(root);
  assert.equal(root.querySelector('.lesson-player-copy').textContent, 'Copy code');
  assert.match(root.querySelector('.lesson-player-legend').textContent, /Active/);
  assert.match(root.querySelector('.lesson-player-legend').textContent, /Result/);
  const edge = root.querySelector('svg').querySelector('line');
  assert.equal(edge.getAttribute('x1'), '50');
  assert.equal(edge.getAttribute('x2'), '120');
  assert.equal(edge.getAttribute('marker-end'), 'url(#lesson-day-01-lookup-normal-arrow)');
});

test('copy uses the selected language source program', async () => {
  const document = fakeDocument();
  let copied = '';
  document.defaultView.navigator = { clipboard: { async writeText(value) { copied = value; } } };
  const data = example();
  data.sources.cpp = '#include <iostream>\nint main() {}\n';
  const { root } = host(document, data);
  mountExample(root);
  root.querySelectorAll('.lesson-player-tab')[0].click();
  root.querySelector('.lesson-player-copy').click();
  await Promise.resolve();
  assert.equal(copied, '#include <iostream>\nint main() {}\n');
});

test('mountLessonExamples accepts an example root and cleanup restores its fallback', () => {
  const document = fakeDocument();
  const { root, fallback } = host(document, example());
  const cleanup = mountLessonExamples(root);
  assert.equal(root.querySelectorAll('.lesson-player').length, 1);
  cleanup();
  assert.equal(fallback.hidden, false);
});

test('player shows algorithm lines and folds the complete runnable source', () => {
  const document = fakeDocument();
  const data = example();
  for (const variant of data.variants) { variant.algorithmStart = 2; variant.algorithmEnd = 2; }
  for (const traceCase of data.cases) for (const frame of traceCase.frames) {
    frame.lines = { cpp: [2], python: [2], java: [2], go: [2] };
  }
  const { root } = host(document, data);
  mountExample(root);
  assert.equal(root.querySelector('.lesson-player-code').querySelectorAll('.lesson-code-line').length, 1);
  assert.match(root.querySelector('.lesson-player-code').textContent, /1  two/);
  assert.equal(Boolean(root.querySelector('.lesson-player-full-source').open), false);
  assert.match(root.querySelector('.lesson-player-full-source').textContent, /one/);
});

test('codeView hides emit calls and moves their highlight to the traced statement', async () => {
  const { codeView } = await import('../internal/leetgrinder/lesson_player.js');
  const lines = [
    'def solve(values):',
    '    total = 0',
    '    emit("start", total=total)',
    '    for value in values:',
    '        total += value',
    '        emit("add",',
    '             total=total)',
    '',
    '    emit("done", total=total)',
    '    return total',
  ];
  const view = codeView(lines, 1, lines.length, [3, 6, 9]);
  assert.deepEqual(view.map(entry => entry.number), [1, 2, 4, 5, 8, 10]);
  assert.deepEqual(view.filter(entry => entry.active).map(entry => entry.number), [2, 4, 5]);
});

test('codeView maps each hidden emit to the statement it reports on', async () => {
  const { codeView } = await import('../internal/leetgrinder/lesson_player.js');
  const lines = [
    'def solve(values):',
    '    emit("start")',
    '    total = 0',
    '    for value in values:',
    '        if value:',
    '            total += value',
    '        emit("step")',
    '',
    '    emit("done")',
  ];
  const active = highlighted => codeView(lines, 1, lines.length, highlighted).filter(entry => entry.active).map(entry => entry.number);
  assert.deepEqual(active([2]), [1], 'emit at the start of a block maps to its header');
  assert.deepEqual(active([7]), [5], 'emit after a nested block maps to the statement at its own level, not inside the block');
  assert.deepEqual(active([9]), [4], 'emit after the loop maps to the loop, not into its body');
  assert.deepEqual(codeView(lines, 2, 3, [2]).filter(entry => entry.active).map(entry => entry.number), [3], 'emit at the start of the shown range maps forward');
  assert.deepEqual(codeView(lines, 3, 9, [9]).filter(entry => entry.active).map(entry => entry.number), [4], 'emit at the end of the shown range maps back');
});

test('codeView ignores parentheses inside strings and never hides past an unbalanced emit', async () => {
  const { codeView } = await import('../internal/leetgrinder/lesson_player.js');
  const lines = ['x = 1', 'emit("open (", x=x)', 'y = 2', 'emit(', 'z = 3'];
  const numbers = codeView(lines, 1, lines.length, []).map(entry => entry.number);
  assert.deepEqual(numbers, [1, 3, 5]);
});
