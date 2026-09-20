// Browser interaction coverage with isolated API fixtures; no database changes.
// Start Astro first. TEST_BASE_URL and TEST_BROWSER can override local defaults.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:4346';
const profile = mkdtempSync(join(tmpdir(), 'career-graph-browser-'));
const browser = spawn(process.env.TEST_BROWSER || '/opt/brave-origin-bin/brave', ['--headless=new', '--no-sandbox', '--disable-gpu', '--remote-debugging-port=9354', `--user-data-dir=${profile}`, '--no-first-run', 'about:blank'], { stdio: 'ignore' });
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
let socket;
try {
  let tabs;
  for (let i = 0; i < 100; i++) { try { tabs = await (await fetch('http://127.0.0.1:9354/json')).json(); if(tabs?.length) break; await pause(100); } catch { await pause(100); } }
  assert.ok(tabs?.length, 'browser started');
  socket = new WebSocket(tabs[0].webSocketDebuggerUrl);
  await new Promise(resolve => socket.addEventListener('open', resolve, { once: true }));
  let id = 0;
  const pending = new Map();
  const errors = [];
  socket.addEventListener('message', event => {
    const m = JSON.parse(event.data);
    if (m.id) { const p = pending.get(m.id); pending.delete(m.id); if (m.error) p.reject(m.error); else p.resolve(m.result); }
    if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') errors.push(m.params.args.map(a=>a.value || a.description).join(' '));
    if (m.method === 'Network.responseReceived' && m.params.response.status >= 400) errors.push(m.params.response.status+' '+m.params.response.url);
    if (m.method === 'Runtime.exceptionThrown') errors.push(m.params.exceptionDetails.exception?.description || m.params.exceptionDetails.text);
  });
  const send = (method, params = {}) => new Promise((resolve, reject) => { const next = ++id; pending.set(next, { resolve, reject }); socket.send(JSON.stringify({ id: next, method, params })); });
  const evaluate = async expression => {
    const r = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
    if (r.exceptionDetails) throw Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text);
    return r.result.value;
  };
  const wait = async expression => { for (let i = 0; i < 100; i++) { if (await evaluate(`Boolean(${expression})`)) return; await pause(100); } throw Error(`Timed out: ${expression}\n${JSON.stringify(await evaluate("({url:location.href,text:document.body.innerText.slice(-2000)})"))}\n${JSON.stringify(errors)}`); };
  const click = selector => evaluate(`document.querySelector(${JSON.stringify(selector)}).click()`);
  const rect = selector => evaluate(`(() => {const r=document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})()`);
  const drag = async (from, to) => {
    await send('Input.dispatchMouseEvent', { type: 'mouseMoved', ...from });
    await send('Input.dispatchMouseEvent', { type: 'mousePressed', button: 'left', clickCount: 1, ...from });
    for (let i = 1; i <= 8; i++) await send('Input.dispatchMouseEvent', { type: 'mouseMoved', button: 'left', buttons: 1, x: from.x + (to.x - from.x) * i / 8, y: from.y + (to.y - from.y) * i / 8 });
    await send('Input.dispatchMouseEvent', { type: 'mouseReleased', button: 'left', clickCount: 1, ...to });
    await pause(200);
  };
  await send('Page.enable'); await send('Runtime.enable'); await send('Network.enable');
  await send('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false });
  await send('Page.addScriptToEvaluateOnNewDocument', { source: `
    const goal = (id,title,startDate,endDate,dependsOn=[]) => ({id,title,startDate,endDate,dependsOn,status:'planned',color:'#a78bfa',createdAt:'',updatedAt:'',notes:[],steps:[],metadata:{}});
    window.fixtureGoals=JSON.parse(sessionStorage.getItem('graph-fixture') || 'null') || [goal('a','Build the foundation','2026-09-01','2026-09-10'),goal('b','Ship a working prototype','2026-09-12','2026-09-20'),goal('c','Launch and gather feedback','2026-09-18','2026-09-30',['b'])];
    window.saves=[];
    const realFetch=window.fetch;
    window.fetch=async (url,options={}) => {
      if(String(url)==='/api/goals') {
        if(options.method==='POST') { const data=JSON.parse(options.body); window.saves.push(data); if(window.failSave) return Response.json({error:'Save unavailable'},{status:503}); window.fixtureGoals=window.fixtureGoals.map(g=>g.id===data.goal.id?data.goal:g); sessionStorage.setItem('graph-fixture',JSON.stringify(window.fixtureGoals)); return Response.json({goal:data.goal,revision:'r2'}); }
        return Response.json({goals:window.fixtureGoals,revisions:{a:'r1',b:'r1',c:'r1'}});
      }
      if(String(url).startsWith('/api/search?')) {
        const u=new URL(url,location.origin); const kind=u.searchParams.get('kind'); const q=u.searchParams.get('q');
        await new Promise(r=>setTimeout(r,kind==='note'?300:30));
        return Response.json({results:q==='nothing'?[]:[{id:'design-notes',kind:kind||'note',title:kind==='book'?'Designing Data-Intensive Applications':'Prototype design notes',body:'Research and practical ideas for building a thoughtful prototype.'}]});
      }
      if(String(url)==='/api/auth/verify') return Response.json({authenticated:true,username:'Demo'});
      return realFetch(url,options);
    };` });
  await send('Page.navigate', { url: base + '/goals/graph' });
  await wait(`document.querySelectorAll('[data-goal-node]').length===3`);
  await drag(await rect('[aria-label="Connect from Build the foundation"]'), await rect('[data-goal-node="b"]'));
  await wait(`window.saves.length===1 && document.body.textContent.includes('Connection saved')`);
  assert.deepEqual(await evaluate('window.fixtureGoals[1].dependsOn'), ['a']);
  assert.equal(await evaluate('window.saves[0].revision'), 'r1');
  assert.equal(await evaluate(`document.querySelectorAll('.graph-edge').length`), 2);
  assert.equal(await evaluate(`document.querySelectorAll('.graph-edge path[stroke-dasharray]').length`), 1);
  // Click-to-connect also works, and cycles never reach the server.
  await click('[aria-label="Connect from Launch and gather feedback"]');
  await click('[aria-label="Connect to Build the foundation"]');
  await wait(`document.querySelector('[role="alert"]')?.textContent.includes('loop')`);
  assert.equal(await evaluate('window.saves.length'), 1);
  await click('[aria-label="Dismiss error"]');
  const before = await rect('[data-goal-node="a"]');
  await drag({ x: before.x, y: before.y + 50 }, { x: before.x + 40, y: before.y + 70 });
  const after = await rect('[data-goal-node="a"]');
  assert.ok(after.x > before.x + 30, 'card dragging');
  const zoomBefore = await evaluate(`parseInt(document.querySelector('.graph-navigation span').textContent)`);
  await click('[aria-label="Zoom out"]');
  assert.equal(await evaluate(`parseInt(document.querySelector('.graph-navigation span').textContent)`), zoomBefore - 15);
  await evaluate(`Array.from(document.querySelectorAll('button')).find(b=>b.textContent==='Fit view').click()`);
  const edgePoint = await evaluate(`(() => { const p=document.querySelector('.graph-edge-hit'); const point=p.getPointAtLength(p.getTotalLength()/2); const screen=new DOMPoint(point.x,point.y).matrixTransform(p.getScreenCTM()); return {x:screen.x,y:screen.y}; })()`);
  await send('Input.dispatchMouseEvent', { type: 'mousePressed', button: 'left', clickCount: 1, ...edgePoint });
  await send('Input.dispatchMouseEvent', { type: 'mouseReleased', button: 'left', clickCount: 1, ...edgePoint });
  await wait(`document.querySelector('.graph-selection')`);
  await evaluate(`Array.from(document.querySelectorAll('button')).find(b=>b.textContent==='Remove connection').click()`);
  await wait(`document.body.textContent.includes('Connection removed')`);
  assert.deepEqual(await evaluate('window.fixtureGoals[1].dependsOn'), []);
  await evaluate('window.failSave=true');
  await click('[aria-label="Connect from Build the foundation"]'); await click('[aria-label="Connect to Ship a working prototype"]');
  await wait(`document.querySelector('[role="alert"]')?.textContent.includes('Save unavailable')`);
  assert.equal(await evaluate(`document.querySelectorAll('.graph-edge').length`), 1, 'failed saves do not add an edge');
  await evaluate('window.failSave=false'); await click('[aria-label="Dismiss error"]');
  await click('[aria-label="Connect from Build the foundation"]'); await click('[aria-label="Connect to Ship a working prototype"]');
  await wait(`document.querySelectorAll('.graph-edge').length===2`);
  await send('Page.reload'); await pause(500); await wait(`document.querySelectorAll('.graph-edge').length===2`);
  // Keyboard search and collection filters, including out-of-order responses.
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'k', code: 'KeyK', modifiers: 2 });
  await wait(`document.activeElement?.id==='site-search'`);
  await send('Input.insertText', { text: 'prototype' });
  await wait(`document.querySelectorAll('.search-result').length===1`);
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'ArrowDown', code: 'ArrowDown' });
  assert.ok(await evaluate(`document.activeElement.classList.contains('search-result')`));
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'ArrowUp', code: 'ArrowUp' });
  assert.equal(await evaluate(`document.activeElement.id`), 'site-search');
  await evaluate(`(() => {const s=document.querySelector('#search-kind');s.value='note';s.dispatchEvent(new Event('change'));s.value='book';s.dispatchEvent(new Event('change'));})()`);
  await pause(450);
  assert.equal(await evaluate(`document.querySelector('.search-result-kind').textContent`), 'Books');
  const screenshot = await send('Page.captureScreenshot', { format: 'png' });
  writeFileSync('/tmp/career-graph-desktop.png', Buffer.from(screenshot.data, 'base64'));
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape' });
  assert.ok(await evaluate(`document.querySelector('#search-panel').hidden`));
  await send('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  await evaluate(`document.querySelector('#site-search').blur();document.querySelector('#site-search').focus()`);
  await wait(`!document.querySelector('#search-panel').hidden`);
  assert.ok(await evaluate(`document.documentElement.scrollWidth<=innerWidth`), 'mobile page fits viewport');
  assert.ok(await evaluate(`document.querySelector('#search-panel').getBoundingClientRect().right<=innerWidth`), 'mobile results fit viewport');
  await wait(`document.querySelectorAll('.search-result').length===1`);
  const mobile = await send('Page.captureScreenshot', { format: 'png' });
  writeFileSync('/tmp/career-graph-mobile.png', Buffer.from(mobile.data, 'base64'));
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape' });
  await evaluate(`document.querySelector('.graph-canvas').scrollIntoView({block:'center'})`);
  const mobileGraph = await send('Page.captureScreenshot', { format: 'png' });
  writeFileSync('/tmp/career-graph-mobile-canvas.png', Buffer.from(mobileGraph.data, 'base64'));
  assert.deepEqual(errors, []);
  console.log('PASS: drag and click connections, revision writes, cycle prevention, failed saves, removal, reload, card dragging, zoom, keyboard search, stale results, mobile layout.');
} finally { socket?.close(); browser.kill(); }
