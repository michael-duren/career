// Run against an authenticated server with a disposable, initially empty workspace.
// Uses Chromium DevTools protocol; TEST_BASE_URL and TEST_BROWSER override local defaults.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';


const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:4338';
const profile = await mkdtemp(join(tmpdir(), 'career-entry-reader-'));
const browser = spawn(process.env.TEST_BROWSER || '/opt/brave-origin-bin/brave', [
  '--headless=new', '--no-sandbox', '--disable-gpu', '--remote-debugging-port=9344',
  `--user-data-dir=${profile}`, '--no-first-run', 'about:blank',
], { stdio: 'ignore' });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
let socket;
let acceptDialog = true;
const runtimeErrors = [];

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
  socket = new WebSocket(tabs[0].webSocketDebuggerUrl);
  await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }));
  let id = 0;
  let onLoad;
  const pending = new Map();
  function send(method, params = {}, sessionId) {
    return new Promise((resolve, reject) => {
      const next = ++id; pending.set(next, { resolve, reject });
      socket.send(JSON.stringify({ id: next, method, params, sessionId }));
    });
  }
  socket.addEventListener('message', event => {
    const message = JSON.parse(event.data);
    if (message.id) { const task = pending.get(message.id); pending.delete(message.id); if (message.error) task?.reject(message.error); else task?.resolve(message.result); }
    if (message.method === 'Page.loadEventFired') onLoad?.();
    if (message.method === 'Page.javascriptDialogOpening') void send('Page.handleJavaScriptDialog', { accept: acceptDialog });
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
    for (let i = 0; i < 100; i++) { try { if (await evaluate(`Boolean(${expression})`)) return; } catch { /* Navigation may replace the execution context. */ } await pause(100); }
    throw new Error(`Timed out: ${expression}`);
  };
  const click = async text => {
    await wait(`Array.from(document.querySelectorAll('button')).some(b => b.checkVisibility() && b.textContent.trim() === ${JSON.stringify(text)} && !b.disabled)`);
    await evaluate(`Array.from(document.querySelectorAll('button')).find(b => b.checkVisibility() && b.textContent.trim() === ${JSON.stringify(text)}).click()`);
    await pause(150);
  };
  await send('Runtime.enable');
  socket.addEventListener('message', event => { const m=JSON.parse(event.data); if(m.method==='Runtime.exceptionThrown') runtimeErrors.push(m.params); });
  await send('Page.enable');
  await send('Network.enable');
  await send('Network.setCookie', { name: 'session', value: session, url: base, httpOnly: true, sameSite: 'Lax' });

  const setValue = async (selector, value) => evaluate(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); Object.getOwnPropertyDescriptor(el.tagName === 'TEXTAREA' ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype, 'value').set.call(el, ${JSON.stringify(value)}); el.dispatchEvent(new Event('input', { bubbles: true })); })()`);
  const open = async path => { await navigate('Page.navigate', { url: base + path }); await wait(`document.querySelector('input[type="search"]')`); };
  const visibleList = `Array.from(document.querySelectorAll('nav[aria-label$=" entries"]')).some(el => el.checkVisibility())`;
  const reader = `Array.from(document.querySelectorAll('button')).some(el => el.checkVisibility() && el.textContent.includes('Back to'))`;
  await open('/notes');
  await wait(`document.body.textContent.includes('0 entries')`);
  assert.equal(await evaluate(visibleList), true);
  assert.equal(await evaluate(`document.body.textContent.includes('A little reflection')`), false);
  const fixtures = [
    ['note', { id:'reader-a', title:'Reader Alpha', topic:'Alpha', description:'', tags:[], body:'# Full reader\nSaved body' }],
    ['note', { id:'reader-b', title:'Reader Beta', topic:'Beta', description:'', tags:[], body:'Second body' }],
    ['personal', { id:'reader-personal', title:'Personal reflection', date:'2026-09-19', description:'', tags:[], body:'Personal body' }],
    ['week', { slug:'2026/week-22', week:22, year:2026, dates:'2026-09-14 to 2026-09-20', hours:{}, tags:[], body:'Work body' }],
  ];
  for (const [kind, entry] of fixtures) {
    const response = await evaluate(`fetch('/api/entries/${kind}', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(${JSON.stringify({ entry, revision:null })})}).then(async r=>({status:r.status,body:await r.json()}))`);
    assert.equal(response.status,200,JSON.stringify(response));
  }
  await click('Reload latest'); await wait(`document.body.textContent.includes('Reader Alpha')`);
  await click('Beta'); await setValue('input[type="search"]','Reader');
  await evaluate(`document.querySelector('nav[aria-label="note entries"] button').focus(); document.querySelector('nav[aria-label="note entries"] button').click()`);
  await wait(reader);
  assert.equal(await evaluate(visibleList), false);
  assert.equal(await evaluate(`document.activeElement.textContent.includes('Back to')`), true);
  const width = await evaluate(`(() => {const b=Array.from(document.querySelectorAll('button')).find(b=>b.textContent==='Edit');return {reader:b.closest('.rounded-xl').getBoundingClientRect().width,section:b.closest('section').getBoundingClientRect().width};})()`);
  assert.ok(Math.abs(width.reader-width.section)<2,'Reader spans the available section width');
  await click('← Back to notes');
  assert.equal(await evaluate(`document.querySelector('input[type="search"]').value`),'Reader');
  assert.equal(await evaluate(`Array.from(document.querySelectorAll('button')).find(b=>b.textContent==='Beta').getAttribute('aria-pressed')`),'false');
  assert.equal(await evaluate(`document.activeElement.textContent.includes('Reader Alpha')`),true);
  await click('Reader AlphaAlpha'); await wait(reader);
  await click('Edit'); await setValue('textarea','Cancelled body'); await click('Cancel'); await wait(`document.body.textContent.includes('Saved body') && !document.querySelector('textarea')`);
  await click('Edit'); await setValue('textarea','Saved after cancelling'); await click('Save');
  await wait(`document.body.textContent.includes('Saved after cancelling') && !document.querySelector('textarea')`);
  await click('Edit'); await setValue('textarea','Protected draft');
  acceptDialog=false; await click('← Back to notes'); assert.equal(await evaluate(reader),true);
  assert.equal(await evaluate(`document.querySelector('textarea').value`),'Protected draft');
  acceptDialog=true; await click('← Back to notes'); assert.equal(await evaluate(visibleList),true);
  await click('+ New note'); await click('Cancel'); assert.equal(await evaluate(visibleList),true);
  await click('+ New note'); await setValue('input[maxlength="200"]','Created in reader'); await setValue('textarea','Created body'); await click('Save'); await wait(`document.body.textContent.includes('Created body') && !document.querySelector('textarea')`); await click('← Back to notes');
  await open('/notes/reader-a'); await wait(reader);
  await click('← Back to notes'); await click('Reload latest'); await pause(250);
  assert.equal(await evaluate(reader),false); assert.equal(await evaluate('location.pathname'),'/notes');
  await open('/notes?id=reader-a'); await wait(reader);
  await click('Edit'); await setValue('textarea','Recovered browser draft');
  await navigate('Page.reload'); await wait(`document.body.textContent.includes('Recovered your unsaved draft.')`);
  assert.equal(await evaluate(`document.querySelector('textarea').value`),'Recovered browser draft');
  await click('Cancel'); await click('Delete'); await wait(visibleList);
  assert.equal(await evaluate(`document.querySelector('nav[aria-label="note entries"]').textContent.includes('Reader Alpha')`),false);
  for (const [path,label,title] of [['/personal-journal','personal journal entry entries','Personal reflection'],['/journal','week entries','Week 22']]) {
    await open(path); await wait(`document.querySelector('nav[aria-label=${JSON.stringify(label)}] button')`);
    assert.equal(await evaluate(visibleList),true);
    if(path==='/journal') await setValue('textarea[aria-label="Journal entry in Markdown"]','Quick draft survives reading');
    await evaluate(`document.querySelector('nav[aria-label=${JSON.stringify(label)}] button').click()`); await wait(reader);
    assert.equal(await evaluate(visibleList),false);
    assert.equal(await evaluate(`document.querySelector('textarea[aria-label="Journal entry in Markdown"]')?.checkVisibility() ?? false`),false);
    await click('Edit'); await click('Cancel'); await wait(`Array.from(document.querySelectorAll('button')).some(b=>b.checkVisibility() && b.textContent==='Edit')`);
    assert.equal(await evaluate(`document.body.textContent.includes(${JSON.stringify(path==='/journal' ? 'Work body' : 'Personal body')})`),true,'Cancel restores full journal detail');
    await click('← Back to journal'); assert.equal(await evaluate(visibleList),true);
    if(path==='/journal') assert.equal(await evaluate(`document.querySelector('textarea[aria-label="Journal entry in Markdown"]').value`),'Quick draft survives reading');
  }
  await send('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:true});
  await open('/personal-journal'); await wait(`document.querySelector('nav[aria-label="personal journal entry entries"] button')`);
  await evaluate(`document.querySelector('nav[aria-label="personal journal entry entries"] button').click()`); await wait(reader);
  assert.equal(await evaluate(`document.documentElement.scrollWidth<=window.innerWidth`),true,'Mobile reader does not overflow');
  await click('← Back to journal'); assert.equal(await evaluate(visibleList),true);
  await open('/manage/books'); await wait(`document.body.textContent.includes('A little reflection')`);
  assert.equal(await evaluate(reader),false,'Other workspace kinds retain original layout');
  assert.deepEqual(runtimeErrors,[], 'No uncaught browser errors');
  console.log('Entry reader acceptance passed: full-width lists/readers, filters, focus, edit/cancel/save, guarded Back, deep links, draft recovery, deletion, journal quick drafts, mobile, and unchanged book layout.');
} finally { socket?.close(); const exited = new Promise(resolve => browser.once('exit', resolve)); browser.kill(); await exited; await rm(profile, { recursive:true, force:true, maxRetries:5, retryDelay:200 }); }
