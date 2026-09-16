// Local browser smoke test using Chromium's DevTools protocol (no test dependency).
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { writeFile } from 'node:fs/promises';
import jwt from 'jsonwebtoken';

const base = 'http://127.0.0.1:4321';
const browser = spawn(process.env.TEST_BROWSER || '/opt/brave-origin-bin/brave', [
  '--headless=new', '--no-sandbox', '--disable-gpu', '--remote-debugging-port=9335',
  '--user-data-dir=/tmp/career-workspace-browser', '--no-first-run', 'about:blank',
], { stdio: 'ignore' });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
let socket;
let noteId;
const createdEntries = [];
const token = jwt.sign({ username: process.env.AUTH_USERNAME }, process.env.JWT_SECRET, { expiresIn: '10m' });
const headers = { cookie: `auth_token=${token}`, origin: base, 'content-type': 'application/json' };
try {
  let tabs;
  for (let i = 0; i < 80; i++) {
    try { tabs = await (await fetch('http://127.0.0.1:9335/json')).json(); break; } catch { await pause(100); }
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
    throw new Error(`Timed out: ${expression}\n${await evaluate('document.body.innerText')}`);
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
  await navigate('Page.navigate', { url: `${base}/notes` });
  await wait(`document.body.innerText.includes('All topics')`);
  await evaluate(`localStorage.setItem('auth_username', 'workspace-test')`);
  await click('+ New note');
  await type('input[maxlength="200"]', 'Discard this draft');
  await click('Cancel');
  await wait(`document.querySelector('[role="status"]')?.textContent === 'Cancelled.'`);
  assert.equal(await evaluate(`localStorage.getItem('career-workspace:v1:workspace-test:note')`), null);
  await click('+ New note');
  const title = `Browser test ${Date.now()}`;
  await type('input[maxlength="200"]', title);
  await type('textarea', '**Preview works**\n\n- [x] Markdown checklist\n\n<script>window.markdownExecuted=true</script>\n\n[bad link](javascript:alert(1))');
  await click('Preview');
  await wait(`document.querySelector('.prose strong')?.textContent === 'Preview works'`);
  assert.equal(await evaluate('window.markdownExecuted === true'), false);
  assert.equal(await evaluate(`!!document.querySelector('.prose script, .prose a[href^="javascript:"]')`), false);
  await click('Save');
  await wait(`document.querySelector('[role="status"]')?.textContent === 'Saved.'`);
  let snapshot = await (await fetch(`${base}/api/workspace`, { headers })).json();
  noteId = snapshot.data.notes.find(n => n.title === title)?.id;
  assert.ok(noteId);
  await navigate('Page.navigate', { url: `${base}/notes/${noteId}` });
  await wait(`document.querySelector('.prose strong')?.textContent === 'Preview works'`);
  await click('Edit');
  await type('textarea', '\n\nRecovered draft');
  await pause(400);
  await navigate('Page.reload');
  await wait(`document.querySelector('textarea')?.value.includes('Recovered draft')`);
  assert.ok(await evaluate(`document.querySelector('input[maxlength="200"]').value === ${JSON.stringify(title)}`));
  await send('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  assert.ok(await evaluate('document.documentElement.scrollWidth <= window.innerWidth'));
  await writeFile('/tmp/career-workspace-mobile.png', Buffer.from((await send('Page.captureScreenshot', { format: 'png' })).data, 'base64'));
  await click('Save');
  await wait(`document.querySelector('[role="status"]')?.textContent === 'Saved.'`);
  await click('Delete');
  await wait(`document.querySelector('[role="status"]')?.textContent === 'Deleted.'`);
  noteId = undefined;
  await navigate('Page.navigate', { url: `${base}/journal` });
  await wait(`document.querySelector('input[type="date"]')?.value.length === 10`);
  await type('textarea', '**Quick draft preview**');
  await click('Markdown preview');
  await wait(`document.querySelector('.prose strong')?.textContent === 'Quick draft preview'`);
  await click('Cancel');
  await wait(`document.querySelector('[role="status"]')?.textContent === 'Cancelled.'`);
  assert.equal(await evaluate(`document.querySelector('textarea')?.value ?? ''`), '');
  await send('Emulation.setDeviceMetricsOverride', { width: 1280, height: 900, deviceScaleFactor: 1, mobile: false });
  for (const [kind, collection, route, button, idKey] of [
    ['book', 'books', '/manage/books', '+ New book or course', 'slug'],
    ['company', 'companies', '/manage/companies', '+ New company', 'slug'],
    ['document', 'documents', '/documents', '+ New page', 'id'],
  ]) {
    await navigate('Page.navigate', { url: `${base}${route}` });
    await wait(`Array.from(document.querySelectorAll('button')).some(b => b.textContent.trim() === ${JSON.stringify(button)} && !b.disabled)`);
    await click(button);
    const title = `Browser ${kind} ${Date.now()}`;
    await type('input[maxlength="200"]', title);
    if (kind === 'company') await type('input[type="url"]', 'https://example.com');
    if (kind === 'book') {
      await type('textarea[aria-label="Authors, one per line"]', 'Browser Author');
      await evaluate(`const select = Array.from(document.querySelectorAll('label')).find(l => l.textContent.startsWith('Status')).querySelector('select'); select.value = 'reading'; select.dispatchEvent(new Event('change', { bubbles: true }));`);
    }
    await type('textarea[maxlength="100000"]', '\n\n**Saved from the browser**');
    if (kind !== 'document') await evaluate(`Array.from(document.querySelectorAll('fieldset')).find(f => f.querySelector(':scope > legend')?.textContent === 'Checklist').querySelector('input[type="checkbox"]').click()`);
    await click('Preview');
    await wait(`document.querySelector('.prose strong')?.textContent === 'Saved from the browser'`);
    await writeFile(`/tmp/career-${kind}-editor.png`, Buffer.from((await send('Page.captureScreenshot', { format: 'png' })).data, 'base64'));
    await click('Save');
    await wait(`document.querySelector('[role="status"]')?.textContent === 'Saved.'`);
    const snapshot = await (await fetch(`${base}/api/workspace`, { headers })).json();
    const entry = snapshot.data[collection].find(item => item.title === title);
    assert.ok(entry, `${kind} persisted`);
    createdEntries.push({ kind, id: entry[idKey] });
    if (kind !== 'document') assert.ok(entry.body.includes('- [x]'));
    if (kind === 'book') { assert.equal(entry.status, 'reading'); assert.deepEqual(entry.authors, ['Browser Author']); }
    await navigate('Page.navigate', { url: `${base}${route}?id=${entry[idKey]}` });
    await wait(`document.querySelector('.prose strong')?.textContent === 'Saved from the browser'`);
    await click('Delete');
    await wait(`document.querySelector('[role="status"]')?.textContent === 'Deleted.'`);
    createdEntries.pop();
  }
  console.log('Browser checks passed: all editors save and reload; checklists, Markdown safety, drafts, and mobile layout work.');
} finally {
  for (const entry of createdEntries) {
    const snapshot = await (await fetch(`${base}/api/workspace`, { headers })).json();
    await fetch(`${base}/api/workspace`, { method: 'POST', headers, body: JSON.stringify({ ...entry, action: 'delete', revision: snapshot.revision }) });
  }
  if (noteId) {
    const snapshot = await (await fetch(`${base}/api/workspace`, { headers })).json();
    await fetch(`${base}/api/workspace`, { method: 'POST', headers, body: JSON.stringify({ kind: 'note', action: 'delete', id: noteId, revision: snapshot.revision }) });
  }
  socket?.close();
  browser.kill();
}
