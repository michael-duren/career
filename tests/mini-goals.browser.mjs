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
  let onLoad;
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
  const minis = title => evaluate(`Array.from(${card(title)}.querySelectorAll('.graph-minis li span')).map(s => s.textContent)`);
  const stored = async title => (await evaluate(`fetch('/api/goals').then(r => r.json())`)).goals.find(g => g.title === title).steps;
  await send('Runtime.enable');
  await send('Page.enable');
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

  // Reorder with the keyboard: move "three" up.
  await evaluate(`${card('Parent A')}.querySelector('[aria-label^="Reorder three"]').dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }))`);
  await saved('Mini goal reordered');
  assert.deepEqual(await minis('Parent A'), ['one', 'three', 'two renamed']);

  // Native drag and drop: drop "one" onto the lower half of Parent B's "bee".
  await evaluate(`(() => {
    const data = new DataTransfer();
    const row = Array.from(${card('Parent A')}.querySelectorAll('.graph-minis li')).find(li => li.textContent.includes('one'));
    const target = ${card('Parent B')}.querySelector('.graph-minis li');
    const box = target.getBoundingClientRect();
    const at = { bubbles: true, cancelable: true, dataTransfer: data, clientX: box.left + 20, clientY: box.bottom - 2 };
    row.dispatchEvent(new DragEvent('dragstart', { bubbles: true, dataTransfer: data }));
    return new Promise(resolve => setTimeout(() => {
      target.dispatchEvent(new DragEvent('dragover', at));
      setTimeout(() => { target.dispatchEvent(new DragEvent('drop', at)); row.dispatchEvent(new DragEvent('dragend', { bubbles: true })); resolve(); }, 50);
    }, 50));
  })()`);
  await saved('Mini goal moved to “Parent B”');
  assert.deepEqual(await minis('Parent A'), ['three', 'two renamed']);
  assert.deepEqual(await minis('Parent B'), ['bee', 'one']);
  const moved = (await stored('Parent B'))[1];
  assert.equal(moved.title, 'one'); assert.equal(moved.done, true, 'moved mini goals keep their state');

  // Move back through the edit form's goal picker (keyboard/touch alternative to dragging).
  await evaluate(`${card('Parent B')}.querySelector('[aria-label="Edit one"]').click()`);
  await evaluate(`(() => { const select = ${card('Parent B')}.querySelector('li.is-editing select'); select.value = Array.from(select.options).find(o => o.textContent === 'Parent A').value; select.dispatchEvent(new Event('change', { bubbles: true })); })()`);
  await saved('Mini goal moved to “Parent A”');
  assert.deepEqual(await minis('Parent A'), ['three', 'two renamed', 'one']);

  // Delete (confirmation is accepted through CDP).
  await evaluate(`${card('Parent B')}.querySelector('[aria-label="Delete bee"]').click()`);
  await saved('Mini goal deleted');
  assert.deepEqual(await stored('Parent B'), []);

  // Mini goals survive a reload and can be hidden.
  await navigate(base + '/goals/graph');
  await wait(`document.querySelectorAll('.graph-minis').length === 2`);
  assert.deepEqual(await minis('Parent A'), ['three', 'two renamed', 'one']);
  await evaluate(`Array.from(document.querySelectorAll('.graph-toggle')).find(l => l.textContent.includes('Show mini goals')).querySelector('input').click()`);
  await wait(`!document.querySelector('.graph-minis')`);
  console.log('Mini goal browser acceptance passed: create, toggle, rename, keyboard reorder, drag between goals, move via picker, delete, reload and hide.');
} finally { socket?.close(); browser.kill(); }
