// Run against an authenticated server with a disposable, initially empty workspace.
// Uses Chromium DevTools protocol; TEST_BASE_URL and TEST_BROWSER override local defaults.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';

const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:4337';
const browser = spawn(process.env.TEST_BROWSER || '/opt/brave-origin-bin/brave', [
  '--headless=new', '--no-sandbox', '--disable-gpu', '--remote-debugging-port=9344',
  '--user-data-dir=/tmp/career-mini-goals-browser', '--no-first-run', 'about:blank',
], { stdio: 'ignore' });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
let socket;

const login = await fetch(`${base}/api/auth/login`, { method: 'POST', headers: { origin: base, 'content-type': 'application/json' }, body: JSON.stringify({ username: process.env.AUTH_USERNAME || 'pwa-test', password: process.env.TEST_AUTH_PASSWORD || 'local-dev-password' }) });
assert.equal(login.status, 200);
const session = login.headers.get('set-cookie').match(/^session=([^;]+)/)?.[1];
assert.ok(session);

try {
  let tabs;
  for (let i = 0; i < 80; i++) {
    try { tabs = await (await fetch('http://127.0.0.1:9344/json')).json(); break; } catch { await pause(100); }
  }
  assert.ok(tabs?.length, 'Browser started');
  socket = new WebSocket(tabs.find(t => t.type === 'page').webSocketDebuggerUrl);
  await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }));
  let id = 0;
  let onLoad, onDrag;
  const pending = new Map();
  function send(method, params = {}) {
    return new Promise((resolve, reject) => {
      const next = ++id; pending.set(next, { resolve, reject });
      socket.send(JSON.stringify({ id: next, method, params }));
    });
  }
  socket.addEventListener('message', event => {
    const message = JSON.parse(event.data);
    if (message.id) { const task = pending.get(message.id); pending.delete(message.id); if (message.error) task?.reject(message.error); else task?.resolve(message.result); }
    if (message.method === 'Page.loadEventFired') onLoad?.();
    if (message.method === 'Input.dragIntercepted') onDrag?.(message.params.data);
    if (message.method === 'Page.javascriptDialogOpening') void send('Page.handleJavaScriptDialog', { accept: true });
    if (message.method === 'Runtime.exceptionThrown') console.error(JSON.stringify(message.params));
  });
  const navigate = async url => {
    const loaded = new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error('Navigation timed out')), 15000);
      onLoad = () => { clearTimeout(timer); resolve(); };
    });
    await send('Page.navigate', { url });
    await loaded;
  };
  const evaluate = async expression => {
    const result = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
    if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description || result.exceptionDetails.text);
    return result.result.value;
  };
  const wait = async expression => {
    for (let i = 0; i < 100; i++) { try { if (await evaluate(`Boolean(${expression})`)) return; } catch { /* Navigation may replace the execution context. */ } await pause(100); }
    throw new Error(`Timed out: ${expression}`);
  };
  const saved = message => wait(`document.querySelector('.graph-save').textContent === ${JSON.stringify(message)}`);
  const card = title => `Array.from(document.querySelectorAll('[data-goal-node]')).find(e => e.querySelector('strong').textContent === ${JSON.stringify(title)})`;
  const minis = title => evaluate(`Array.from(${card(title)}.querySelectorAll('.graph-minis li > span:not(.graph-mini-handle)')).map(s => s.textContent)`);
  const row = (title, text) => `Array.from(${card(title)}.querySelectorAll('.graph-minis li')).find(li => li.textContent.includes(${JSON.stringify(text)}))`;
  const handle = (title, text) => `${row(title, text)}.querySelector('.graph-mini-handle')`;
  const zoom = () => evaluate(`document.querySelector('.graph-navigation span').textContent`);
  // A real browser drag: mouse down on the handle, then CDP-dispatched drag events at the target's viewport position.
  async function realDrag(from, to, half) {
    const at = async (expr, where) => evaluate(`(() => { const b = (${expr}).getBoundingClientRect(), c = document.querySelector('.graph-canvas').getBoundingClientRect();
      if (b.top < c.top || b.bottom > c.bottom || b.left < c.left || b.right > c.right) throw new Error('Element outside the canvas at this zoom');
      return { x: b.left + b.width / 2, y: ${JSON.stringify(where)} === 'top' ? b.top + 4 : ${JSON.stringify(where)} === 'bottom' ? b.bottom - 4 : b.top + b.height / 2 }; })()`, where);
    const a = await at(from, 'middle'), b = await at(to, half);
    const intercepted = new Promise(resolve => { onDrag = resolve; });
    await send('Input.setInterceptDrags', { enabled: true });
    await send('Input.dispatchMouseEvent', { type: 'mousePressed', x: a.x, y: a.y, button: 'left', clickCount: 1 });
    await send('Input.dispatchMouseEvent', { type: 'mouseMoved', x: a.x + 6, y: a.y + 6, button: 'left' });
    await send('Input.dispatchMouseEvent', { type: 'mouseMoved', x: b.x, y: b.y, button: 'left' });
    const data = await intercepted;
    for (const type of ['dragEnter', 'dragOver', 'drop']) await send('Input.dispatchDragEvent', { type, x: b.x, y: b.y, data });
    await send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: b.x, y: b.y, button: 'left', clickCount: 1 });
    await send('Input.setInterceptDrags', { enabled: false });
  }
  const clickZoom = async (label, times) => { for (let i = 0; i < times; i++) { await evaluate(`document.querySelector('[aria-label="${label}"]').click()`); await pause(50); } };
  const stored = async title => (await evaluate(`fetch('/api/goals').then(r => r.json())`)).goals.find(g => g.title === title).steps;
  await send('Runtime.enable');
  await send('Page.enable');
  await send('Emulation.setDeviceMetricsOverride', { width: 1400, height: 1100, deviceScaleFactor: 1, mobile: false });
  await send('Network.setCookie', { name: 'session', value: session, url: base, httpOnly: true, sameSite: 'Lax' });

  await navigate(base + '/goals/graph');
  // Seed two goals through the same API the timeline uses.
  await evaluate(`(async () => {
    const now = new Date().toISOString();
    for (const title of ['Parent A', 'Parent B']) {
      const goal = { id: crypto.randomUUID(), title, status: 'planned', dependsOn: [], startDate: '2026-10-01', endDate: '2026-10-31', color: '#67e8f9', createdAt: now, updatedAt: now, notes: [], steps: [], metadata: {} };
      const r = await fetch('/api/goals', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal, revision: null }) });
      if (!r.ok) throw new Error(await r.text());
    }
  })()`);
  await navigate(base + '/goals/graph');
  await wait(`document.querySelectorAll('[data-goal-node]').length === 2`);

  // Create mini goals through each card's add form.
  const add = async (title, text) => {
    await evaluate(`(() => { const input = ${card(title)}.querySelector('.graph-mini-add input'); Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(input, ${JSON.stringify(text)}); input.dispatchEvent(new Event('input', { bubbles: true })); })()`);
    await wait(`!${card(title)}.querySelector('.graph-mini-add button').disabled`);
    await evaluate(`${card(title)}.querySelector('.graph-mini-add button').click()`);
    await wait(`${card(title)}.querySelectorAll('.graph-minis li').length && Array.from(${card(title)}.querySelectorAll('.graph-minis li span')).some(s => s.textContent === ${JSON.stringify(text)})`);
  };
  for (const text of ['one', 'two', 'three']) await add('Parent A', text);
  await add('Parent B', 'bee');
  // Cards that were not dragged by hand re-layout around their taller heights.
  assert.equal(await evaluate(`(() => { const [a, b] = Array.from(document.querySelectorAll('[data-goal-node]')).map(e => e.getBoundingClientRect()).sort((x, y) => x.top - y.top); return a.bottom <= b.top || a.right <= b.left; })()`), true, 'cards do not overlap');
  assert.deepEqual((await stored('Parent A')).map(s => s.title), ['one', 'two', 'three']);

  // Toggle done.
  await evaluate(`${card('Parent A')}.querySelector('.graph-minis li input[type=checkbox]').click()`);
  await saved('Mini goal done');
  assert.equal((await stored('Parent A'))[0].done, true);
  assert.match(await evaluate(`${card('Parent A')}.querySelector('.graph-minis-head').textContent`), /1 \/ 3/);

  // Rename through the edit form.
  await evaluate(`${card('Parent A')}.querySelector('[aria-label="Edit two"]').click()`);
  await wait(`${card('Parent A')}.querySelector('li.is-editing input')`);
  await evaluate(`(() => { const input = ${card('Parent A')}.querySelector('li.is-editing input'); Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(input, 'two renamed'); input.dispatchEvent(new Event('input', { bubbles: true })); input.form.requestSubmit(); })()`);
  await saved('Mini goal renamed');
  assert.deepEqual(await minis('Parent A'), ['one', 'two renamed', 'three']);

  // Reorder with the keyboard; focus stays on the moved mini goal's handle.
  await evaluate(`${handle('Parent A', 'three')}.focus()`);
  await send('Input.dispatchKeyEvent', { type: 'rawKeyDown', key: 'ArrowUp', code: 'ArrowUp', windowsVirtualKeyCode: 38 });
  await saved('Mini goal reordered');
  assert.deepEqual(await minis('Parent A'), ['one', 'three', 'two renamed']);
  await send('Input.dispatchKeyEvent', { type: 'rawKeyDown', key: 'ArrowDown', code: 'ArrowDown', windowsVirtualKeyCode: 40 });
  await saved('Mini goal reordered');
  assert.deepEqual(await minis('Parent A'), ['one', 'two renamed', 'three']);
  await wait(`document.activeElement === ${handle('Parent A', 'three')}`);

  // Touch-friendly reorder: Up / Down buttons in the edit form.
  await evaluate(`${card('Parent A')}.querySelector('[aria-label="Edit three"]').click()`);
  await evaluate(`${card('Parent A')}.querySelector('[aria-label="Move three up"]').click()`);
  await saved('Mini goal reordered');
  assert.deepEqual((await stored('Parent A')).map(s => s.title), ['one', 'three', 'two renamed']);
  await wait(`document.activeElement === ${card('Parent A')}.querySelector('[aria-label="Move three up"]')`);
  await evaluate(`Array.from(${card('Parent A')}.querySelectorAll('li.is-editing button')).find(b => b.textContent === 'Cancel').click()`);

  // Real drag from the handle at a zoomed-out view: drop "one" on the lower half of Parent B's "bee".
  await evaluate(`Array.from(document.querySelectorAll('.graph-navigation button')).find(b => b.textContent === 'Fit view').click()`);
  await clickZoom('Zoom out', 2);
  const zoomedOut = await zoom();
  await realDrag(handle('Parent A', 'one'), row('Parent B', 'bee'), 'bottom');
  await saved('Mini goal moved to “Parent B”');
  assert.deepEqual(await minis('Parent A'), ['three', 'two renamed']);
  assert.deepEqual(await minis('Parent B'), ['bee', 'one']);
  const moved = (await stored('Parent B'))[1];
  assert.equal(moved.title, 'one'); assert.equal(moved.done, true, 'moved mini goals keep their state');

  // And again zoomed in: drop "one" on the upper half of Parent A's first row.
  await evaluate(`Array.from(document.querySelectorAll('.graph-navigation button')).find(b => b.textContent === 'Fit view').click()`);
  await clickZoom('Zoom in', 1);
  const zoomedIn = await zoom();
  await realDrag(handle('Parent B', 'one'), row('Parent A', 'three'), 'top');
  await saved('Mini goal moved to “Parent A”');
  assert.deepEqual(await minis('Parent A'), ['one', 'three', 'two renamed']);
  assert.deepEqual(await minis('Parent B'), ['bee']);
  // Dragging the title text (not the handle) does not start a move.
  assert.equal(await evaluate(`${row('Parent A', 'one')}.draggable`), false);

  // Move through the goal picker: choosing a goal does nothing until Move is pressed.
  await evaluate(`${card('Parent A')}.querySelector('[aria-label="Edit one"]').click()`);
  await evaluate(`(() => { const select = ${card('Parent A')}.querySelector('li.is-editing select'); select.value = Array.from(select.options).find(o => o.textContent === 'Parent B').value; select.dispatchEvent(new Event('change', { bubbles: true })); })()`);
  await pause(300);
  assert.deepEqual(await minis('Parent B'), ['bee'], 'picker change alone does not move');
  await evaluate(`Array.from(${card('Parent A')}.querySelectorAll('li.is-editing button')).find(b => b.textContent === 'Move').click()`);
  await saved('Mini goal moved to “Parent B”');
  assert.deepEqual(await minis('Parent B'), ['bee', 'one']);
  assert.deepEqual(await minis('Parent A'), ['three', 'two renamed']);

  // Delete (confirmation is accepted through CDP).
  await evaluate(`${card('Parent B')}.querySelector('[aria-label="Delete bee"]').click()`);
  await saved('Mini goal deleted');
  assert.deepEqual((await stored('Parent B')).map(s => s.title), ['one']);

  // A failed move is shown immediately, then reverted: change Parent A behind the page's back.
  await evaluate(`(async () => {
    const data = await fetch('/api/goals').then(r => r.json());
    const goal = data.goals.find(g => g.title === 'Parent A');
    await fetch('/api/goals', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal: { ...goal, metadata: { edited: 'elsewhere' } }, revision: data.revisions[goal.id] }) });
  })()`);
  await evaluate(`${handle('Parent A', 'three')}.focus()`);
  await send('Input.dispatchKeyEvent', { type: 'rawKeyDown', key: 'ArrowDown', code: 'ArrowDown', windowsVirtualKeyCode: 40 });
  await saved('Changes were not saved');
  assert.match(await evaluate(`document.querySelector('.graph-error').textContent`), /changed elsewhere/);
  assert.deepEqual(await minis('Parent A'), ['three', 'two renamed'], 'optimistic reorder reverted');

  // Mini goals survive a reload; hiding them is remembered.
  await navigate(base + '/goals/graph');
  await wait(`document.querySelectorAll('.graph-minis').length === 2`);
  assert.deepEqual(await minis('Parent A'), ['three', 'two renamed']);
  assert.equal(await evaluate(`${card('Parent A')}.querySelector('.graph-card-detail').textContent`), '', 'no duplicate progress line while mini goals show');
  const toggle = `Array.from(document.querySelectorAll('.graph-toggle')).find(l => l.textContent.includes('Show mini goals')).querySelector('input')`;
  await evaluate(`${toggle}.click()`);
  await wait(`!document.querySelector('.graph-minis')`);
  await navigate(base + '/goals/graph');
  await wait(`document.querySelectorAll('[data-goal-node]').length === 2`);
  await wait(`!document.querySelector('.graph-minis') && !${toggle}.checked`);
  assert.match(await evaluate(`${card('Parent A')}.querySelector('.graph-card-detail').textContent`), /0 \/ 2 mini goals complete/);
  await evaluate(`${toggle}.click()`);
  await wait(`document.querySelectorAll('.graph-minis').length === 2`);
  console.log(`Mini goal browser acceptance passed: create, toggle, rename, keyboard reorder with focus, Up/Down buttons, real handle drags at ${zoomedOut} and ${zoomedIn}, explicit picker move, delete, optimistic revert, reload and remembered hide.`);
} finally { socket?.close(); browser.kill(); }
