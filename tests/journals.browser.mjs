// Local browser smoke test using Chromium's DevTools protocol (no test dependency).
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';



const base = 'http://127.0.0.1:4322';
const browser = spawn(process.env.TEST_BROWSER || '/opt/brave-origin-bin/brave', [
  '--headless=new', '--no-sandbox', '--disable-gpu', '--remote-debugging-port=9336',
  '--user-data-dir=/tmp/career-journals-browser', '--no-first-run', 'about:blank',
], { stdio: 'ignore' });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
let socket;
let noteId;

const login = await fetch(`${base}/.netlify/functions/auth`, { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ action: 'login', username: 'admin', password: 'local-dev-password' }) });
assert.equal(login.status, 200);
const { token } = await login.json();
const headers = { cookie: `auth_token=${token}`, origin: base, 'content-type': 'application/json' };
try {
  let tabs;
  for (let i = 0; i < 80; i++) {
    try { tabs = await (await fetch('http://127.0.0.1:9336/json')).json(); break; } catch { await pause(100); }
  }
  assert.ok(tabs?.length, 'Browser started');
  socket = new WebSocket(tabs[0].webSocketDebuggerUrl);
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
  });
  const navigate = async (method, params = {}) => {
    const loaded = new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error('Navigation timed out')), 15000);
      onLoad = () => { clearTimeout(timer); resolve(); };
    });
    await send(method, params);
    await loaded;
  };
  const evaluate = async expression => {
    const result = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
    if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description || result.exceptionDetails.text);
    return result.result.value;
  };
  const wait = async expression => {
    for (let i = 0; i < 100; i++) { try { if (await evaluate(expression)) return; } catch { /* Navigation may replace the execution context. */ } await pause(100); }
    throw new Error(`Timed out: ${expression}`);
  };
  const click = async text => {
    await wait(`Array.from(document.querySelectorAll('button')).some(b => b.textContent.trim() === ${JSON.stringify(text)} && !b.disabled)`);
    await evaluate(`Array.from(document.querySelectorAll('button')).find(b => b.textContent.trim() === ${JSON.stringify(text)}).click()`);
    await pause(150);
  };
  async function type(selector, text) {
    await evaluate(`document.querySelector(${JSON.stringify(selector)}).focus()`);
    await send('Input.insertText', { text });
    await pause(100);
  }
  await send('Page.enable');
  await send('Network.enable');
  await send('Network.setCookie', { name: 'auth_token', value: token, url: base });

  await navigate('Page.navigate', { url: `${base}/personal-journal` });
  await click('+ New personal journal entry');
  const title = `Personal browser test ${Date.now()}`;
  await type('input[maxlength="200"]', title);
  await type('textarea', '**Life reflection**');
  await click('Save');
  await wait(`document.body.innerText.includes('Saved.')`);
  let snapshot = await (await fetch(`${base}/api/workspace`, { headers })).json();
  noteId = snapshot.data.personalJournal.find(n => n.title === title)?.id;
  assert.ok(noteId);
  await navigate('Page.navigate', { url: `${base}/personal-journal?id=${noteId}` });
  await wait(`document.querySelector('.prose strong')?.textContent === 'Life reflection'`);
  await navigate('Page.navigate', { url: `${base}/journal` });
  await wait(`document.body.innerText.includes('entries')`);
  assert.ok(!(await evaluate('document.body.innerText')).includes(title));
  await navigate('Page.navigate', { url: `${base}/agents` });
  await wait(`document.body.innerText.includes('Complete agent briefing')`);
  assert.ok((await evaluate('document.body.innerText')).includes(title));
  await send('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  assert.ok(await evaluate('document.documentElement.scrollWidth <= window.innerWidth'));
  await evaluate(`document.querySelector('#mobile-menu-button').click()`);
  await wait(`document.querySelector('#mobile-menu-button').getAttribute('aria-expanded') === 'true'`);
  const labels = await evaluate(`Array.from(document.querySelectorAll('#mobile-sidebar nav > div:first-child a')).map(a => a.textContent.trim())`);
  assert.ok(labels.includes('Personal journal'));
  assert.ok(labels.includes('Work journal'));
  assert.ok(labels.includes('Agent context'));
  assert.deepEqual(labels, [...labels].sort((a, b) => a.localeCompare(b, 'en', { sensitivity: 'base' })));
  console.log('Personal journal creation/readback, work separation, agent briefing, and mobile navigation passed.');
} finally {
  if (noteId) {
    const current = await (await fetch(`${base}/api/workspace`, { headers })).json();
    await fetch(`${base}/api/workspace`, { method: 'POST', headers, body: JSON.stringify({ action: 'delete', kind: 'personal', id: noteId, revision: current.revision }) });
  }
  socket?.close();
  browser.kill();
}
