// Run against an authenticated disposable workspace: TEST_BASE_URL, AUTH_USERNAME,
// TEST_AUTH_PASSWORD and TEST_BROWSER configure the real server/browser.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:4337';
const browser = spawn(process.env.TEST_BROWSER || '/opt/brave-origin-bin/brave', ['--headless=new', '--no-sandbox', '--disable-gpu', '--remote-debugging-port=9348', '--user-data-dir=/tmp/career-scheduler-browser', '--no-first-run', 'about:blank'], { stdio: 'ignore' });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
let socket;
const login = await fetch(`${base}/api/auth/login`, { method: 'POST', headers: { origin: base, 'content-type': 'application/json' }, body: JSON.stringify({ username: process.env.AUTH_USERNAME || 'pwa-test', password: process.env.TEST_AUTH_PASSWORD || 'local-dev-password' }) });
assert.equal(login.status, 200);
const session = login.headers.get('set-cookie').match(/^session=([^;]+)/)?.[1];
assert.ok(session);
try {
  let tabs;
  for (let i = 0; i < 80; i++) { try { tabs = await (await fetch('http://127.0.0.1:9348/json')).json(); if (tabs.length) break; await pause(100); } catch { await pause(100); } }
  assert.ok(tabs?.length);
  socket = new WebSocket(tabs.find(tab => tab.type === 'page').webSocketDebuggerUrl);
  await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }));
  let id = 0;
  const pending = new Map();
  function send(method, params = {}) { return new Promise((resolve, reject) => { const next = ++id; pending.set(next, { resolve, reject }); socket.send(JSON.stringify({ id: next, method, params })); }); }
  socket.addEventListener('message', event => { const message = JSON.parse(event.data); if (message.id) { const task = pending.get(message.id); pending.delete(message.id); if (message.error) task?.reject(message.error); else task?.resolve(message.result); } });
  async function evaluate(expression) { const result = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true }); if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description || result.exceptionDetails.text); return result.result.value; }
  async function wait(expression) { for (let i = 0; i < 100; i++) { try { if (await evaluate(`Boolean(${expression})`)) return; } catch {} await pause(100); } throw new Error(`Timed out: ${expression}\n${await evaluate("document.body.innerText")}`); }
  const click = text => evaluate(`Array.from(document.querySelectorAll('.weekly-scheduler button')).find(b => b.textContent.trim() === ${JSON.stringify(text)}).click()`);
  const input = (label, value) => evaluate(`(() => { const e = Array.from(document.querySelectorAll('.scheduler-editor label')).find(l => l.firstChild.textContent === ${JSON.stringify(label)}).querySelector('input,select'); const setter = Object.getOwnPropertyDescriptor(e.tagName === 'SELECT' ? HTMLSelectElement.prototype : HTMLInputElement.prototype, 'value').set; setter.call(e, ${JSON.stringify(value)}); e.dispatchEvent(new Event(e.tagName === 'SELECT' ? 'change' : 'input', { bubbles: true })); })()`);
  const api = (path, body) => evaluate(`fetch(${JSON.stringify(path)}, ${JSON.stringify(body === undefined ? {} : { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })}).then(async r => ({ status: r.status, body: await r.json() }))`);
  await send('Runtime.enable'); await send('Page.enable');
  await send('Emulation.setDeviceMetricsOverride', { width: 1500, height: 1100, deviceScaleFactor: 1, mobile: false });
  await send('Network.setCookie', { name: 'session', value: session, url: base, httpOnly: true, sameSite: 'Lax' });
  await send('Page.navigate', { url: `${base}/weekly-scheduler` });
  await wait("document.querySelector('.scheduler-grid') && document.querySelector('.weekly-scheduler').getAttribute('aria-busy') === 'false'");
  assert.equal(await evaluate("Array.from(document.querySelectorAll('.scheduler-day')).filter(e => getComputedStyle(e).display !== 'none').length"), 7);
  // A future week makes this deterministic regardless of the wall clock.
  await evaluate("document.querySelector('[aria-label=\"Next week\"]').click()");
  await wait("document.querySelector('.weekly-scheduler').getAttribute('aria-busy') === 'false'");
  const date = await evaluate("document.querySelector('[data-scheduler-date]').dataset.schedulerDate");
  const goalId = `scheduler-browser-${Date.now()}`;
  const created = await api('/api/goals', { revision: null, goal: { id: goalId, title: 'Scheduler browser goal', status: 'planned', startDate: date, endDate: date.slice(0,4) + '-12-31', dailyHours: 1, selectedWeekdays: [1,3,5], color: '#67e8f9', dependsOn: [], steps: [], notes: [], metadata: {}, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() } });
  assert.equal(created.status, 200, JSON.stringify(created));
  await click('Day settings');
  await evaluate("(() => { const e = document.querySelector('[aria-label=\"Scheduler settings\"] input'); Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(e,'UTC'); e.dispatchEvent(new Event('input',{bubbles:true})); })()");
  await click('Save settings');
  await wait("!document.querySelector('[aria-label=\"Scheduler settings\"]')");
  await wait("document.querySelector('.scheduler-goal-title')?.textContent.includes('Scheduler browser goal')");
  await evaluate("document.querySelector('.scheduler-goal-title').click()");
  await input('Scheduling date', date); await input('Start time', '09:07'); await input('End time', '10:22');
  await click('Save session');
  await wait("!document.querySelector('.scheduler-editor') && document.querySelector('[data-session-id]')");
  let state = (await api(`/api/scheduler/week?week=${date}`)).body;
  assert.equal(state.sessions.filter(s => s.assignment.goalId === goalId).length, 1);
  assert.equal(state.goals.find(s => s.goal.id === goalId).requiredHours, 3);
  assert.equal(state.goals.find(s => s.goal.id === goalId).remainingScheduledHours, 1.25);
  // Overlap retains the editor draft and highlights the conflicting reservation.
  await evaluate("document.querySelector('.scheduler-goal-title').click()");
  await input('Scheduling date', date); await input('Start time', '09:30'); await input('End time', '10:30'); await click('Save session');
  await wait("document.querySelector('[role=\"alert\"]')?.textContent.includes('draft')");
  assert.equal(await evaluate("document.querySelector('.scheduler-editor input[type=time]').value"), '09:30');
  assert.equal(await evaluate("document.querySelectorAll('.scheduler-conflict').length"), 1);
  await click('Discard draft');
  // Real drag moves an accepted future block and snaps to fifteen minutes.
  const block = await evaluate("(() => { const e=document.querySelector('[data-session-id]'); e.scrollIntoView({block:'center'}); const r=e.getBoundingClientRect(); return {x:r.x+r.width/2,y:r.y+15}; })()");
  const target = await evaluate("(() => { const r=document.querySelectorAll('[data-scheduler-date]')[1].getBoundingClientRect(); return {x:r.x+r.width/2,y:r.y+300}; })()");
  await send('Input.dispatchDragEvent', { type: 'dragEnter', x: target.x, y: target.y, data: { items: [{ mimeType: 'text/plain', data: 'session' }], dragOperationsMask: 1 } });
  // Native DragEvent initializes the same drag state as a mouse drag; CDP supplies the drop location.
  await evaluate("document.querySelector('[data-session-id]').dispatchEvent(new DragEvent('dragstart',{bubbles:true,dataTransfer:new DataTransfer()}))");
  await send('Input.dispatchDragEvent', { type: 'drop', x: target.x, y: target.y, data: { items: [{ mimeType: 'text/plain', data: 'session' }], dragOperationsMask: 1 } });
  await wait("!document.querySelector('.scheduler-editor') && document.querySelector('.scheduler-status').textContent.includes('Saved')");
  state = (await api(`/api/scheduler/week?week=${date}`)).body;
  assert.equal(state.sessions.find(s => s.assignment.goalId === goalId).date, state.days[1].date);
  assert.equal(new Date(state.sessions.find(s => s.assignment.goalId === goalId).plan.start).getUTCMinutes() % 15, 0);
  // Narrow screen retains editor equivalents and displays only the selected day.
  await send('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  assert.equal(await evaluate("Array.from(document.querySelectorAll('.scheduler-day')).filter(e => getComputedStyle(e).display !== 'none').length"), 1);
  assert.equal(await evaluate("document.documentElement.scrollWidth <= innerWidth"), true);
  await click('Add commitment'); await input('Commitment name', 'Browser commute'); await input('Scheduling date', date); await input('Start time', '07:00'); await input('End time', '08:00');
  await evaluate("document.querySelector('.scheduler-editor input[type=checkbox]').click()");
  await click('Save session'); await wait("!document.querySelector('.scheduler-editor')");
  state = (await api(`/api/scheduler/week?week=${date}`)).body;
  assert.equal(state.rules.some(r => r.assignment.title === 'Browser commute'), true);
  assert.equal(state.sessions.some(s => s.assignment.title === 'Browser commute'), true);
  console.log('Scheduler browser checks passed: desktop create, accounting, conflict draft, drag move, mobile recurrence and width.');
} finally { socket?.close(); browser.kill(); }
