// Run against an authenticated server with a disposable, initially empty workspace.
// Uses Chromium DevTools protocol; TEST_BASE_URL and TEST_BROWSER override local defaults.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';



const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:4337';
const browser = spawn(process.env.TEST_BROWSER || '/opt/brave-origin-bin/brave', [
  '--headless=new', '--no-sandbox', '--disable-gpu', '--remote-debugging-port=9343',
  '--user-data-dir=/tmp/career-goal-dependencies-browser', '--no-first-run', 'about:blank',
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
    try { tabs = await (await fetch('http://127.0.0.1:9343/json')).json(); break; } catch { await pause(100); }
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
    for (let i = 0; i < 100; i++) { try { if (await evaluate(`Boolean(${expression})`)) return; } catch { /* Navigation may replace the execution context. */ } await pause(100); }
    throw new Error(`Timed out: ${expression}`);
  };
  const click = async text => {
    await wait(`Array.from(document.querySelectorAll('button')).some(b => b.textContent.trim() === ${JSON.stringify(text)} && !b.disabled)`);
    await evaluate(`Array.from(document.querySelectorAll('button')).find(b => b.textContent.trim() === ${JSON.stringify(text)}).click()`);
    await pause(150);
  };
  await send('Runtime.enable');
  socket.addEventListener('message', event => { const m=JSON.parse(event.data); if(m.method==='Runtime.exceptionThrown') console.error(JSON.stringify(m.params)); });
  await send('Page.enable');
  await send('Network.enable');
  await send('Network.setCookie', { name: 'session', value: session, url: base, httpOnly: true, sameSite: 'Lax' });


  await navigate('Page.navigate', { url: base + '/timeline' });
  await wait(`document.body.textContent.includes('All changes saved')`);
  const setValue = async (selector, value) => evaluate(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(el, ${JSON.stringify(value)}); el.dispatchEvent(new Event('input', { bubbles: true })); })()`);
  async function create(title, deps = []) {
    await click('+ New goal');
    await wait(`document.querySelector('dialog[open] input[maxlength="200"]')`);
    await setValue('dialog input[maxlength="200"]', title);
    if (deps.length) await evaluate(`(() => { const el=document.querySelector('dialog select[multiple]'); for(const o of el.options) o.selected=${JSON.stringify(deps)}.includes(o.textContent); el.dispatchEvent(new Event('change',{bubbles:true})); })()`);
    await click('Create goal'); await wait(`!document.querySelector('dialog[open]')`);
  }
  await create('Prerequisite A');
  await create('Dependent B', ['Prerequisite A']);
  const cycleResponses = await evaluate(`(async () => {
    const data=await fetch('/api/goals').then(r=>r.json());
    const a=data.goals.find(g=>g.title==='Prerequisite A'), b=data.goals.find(g=>g.title==='Dependent B');
    const statuses=[];
    for (const id of [a.id,b.id]) { const r=await fetch('/api/goals',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({goal:{...a,dependsOn:[id]},revision:data.revisions[a.id]})}); statuses.push({status:r.status,error:(await r.json()).error}); }
    return statuses;
  })()`);
  for (const response of cycleResponses) { assert.equal(response.status,400);assert.match(response.error,/cycle/); }
  await evaluate(`Array.from(document.querySelectorAll('.timeline-list button')).find(b=>b.textContent.includes('Prerequisite A')).click()`);
  await wait(`document.querySelector('dialog[open]')`);
  assert.equal(await evaluate(`Array.from(document.querySelector('dialog select[multiple]').options).find(o=>o.textContent==='Dependent B').disabled`), true, 'cycle-causing option disabled');
  await click('Cancel');
  await navigate('Page.navigate', { url: base + '/goals/graph' });
  await wait(`document.querySelectorAll('[data-goal-node]').length===2`);
  assert.match(await evaluate(`document.querySelector('[aria-label="Goal graph"]').textContent`), /Blocked by: Prerequisite A/);
  assert.ok(await evaluate(`Array.from(document.querySelectorAll('.graph-edge path')).some(e=>e.getAttribute('stroke-dasharray'))`), 'conflict edge dashed');
  await evaluate(`(() => {const el=document.querySelector('[aria-label="Critical path target"]');el.value=Array.from(el.options).find(o=>o.textContent==='Dependent B').value;el.dispatchEvent(new Event('change',{bubbles:true}));})()`);
  await wait(`document.body.textContent.includes('Critical path:')`);
  assert.match(await evaluate(`document.body.textContent`), /Prerequisite A → Dependent B/);
  await evaluate(`Array.from(document.querySelectorAll('[data-goal-node]')).find(e=>e.querySelector('strong').textContent==='Prerequisite A').querySelector('a').click()`);
  await wait(`document.querySelector('dialog[open]')`);
  await evaluate(`(() => {const el=document.querySelector('dialog select:not([multiple])');el.value='dropped';el.dispatchEvent(new Event('change',{bubbles:true}));})()`);
  await click('Save goal'); await wait(`!document.querySelector('dialog[open]')`);
  await navigate('Page.navigate', { url: base + '/goals/graph' });
  await wait(`document.querySelectorAll('[data-goal-node]').length===1`);
  assert.match(await evaluate(`document.querySelector('[data-goal-node]').textContent`), /Ready/);
  assert.match(await evaluate(`document.querySelector('[data-goal-node]').textContent`), /Dropped prerequisite: Prerequisite A/);
  await evaluate(`document.querySelector('input[type="checkbox"]').click()`);
  await wait(`document.querySelectorAll('[data-goal-node]').length===2`);
  await evaluate(`Array.from(document.querySelectorAll('[data-goal-node]')).find(e=>e.querySelector('strong').textContent==='Prerequisite A').querySelector('a').click()`);
  await wait(`document.querySelector('dialog[open]')`);
  // Capture the confirmation text while accepting through CDP.
  let confirmation = '';
  socket.addEventListener('message', event => { const m=JSON.parse(event.data); if(m.method==='Page.javascriptDialogOpening') confirmation=m.params.message; });
  await click('Delete goal'); await wait(`!document.querySelector('dialog[open]')`);
  assert.match(confirmation, /Dependent B/);
  const saved = await evaluate(`fetch('/api/goals').then(r=>r.json())`);
  assert.equal(saved.goals.length,1);assert.deepEqual(saved.goals[0].dependsOn,[]);
  console.log('Goal dependency browser acceptance passed: create edges, cycle prevention, graph, conflicts, critical target, hide-finished, dropped warning, editor links and confirmed cascade deletion.');
} finally { socket?.close(); browser.kill(); }
