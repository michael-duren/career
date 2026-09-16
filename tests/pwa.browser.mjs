// Local browser smoke test using Chromium's DevTools protocol (no test dependency).
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';



const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:4335';
const browser = spawn(process.env.TEST_BROWSER || '/opt/brave-origin-bin/brave', [
  '--headless=new', '--no-sandbox', '--disable-gpu', '--remote-debugging-port=9338',
  '--user-data-dir=/tmp/career-pwa-browser', '--no-first-run', 'about:blank',
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
    try { tabs = await (await fetch('http://127.0.0.1:9338/json')).json(); break; } catch { await pause(100); }
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
  await send('Network.setCookie', { name: 'session', value: session, url: base, httpOnly: true, sameSite: 'Lax' });

  const routes = new Set(['/', '/login', '/agents', '/personal-journal', '/journal', '/timeline', '/documents', '/notes', '/companies', '/progress', '/logs', '/books', '/manage/books', '/manage/companies']);
  const { readdir } = await import('node:fs/promises');
  for (const route of ['/2026/career-study-plan', '/2026/languages', '/2026/os-oss', '/2026/os-oss/ostep', '/2026/os-oss/build-runtime', '/2026/os-oss/conference-talk', '/2026/os-oss/container-internals', '/2026/system-design', '/2026/system-design/alex-xu-vol1', '/2026/system-design/ddia-read', '/2026/system-design/hello-interview', '/2026/random/vintage-computers']) routes.add(route);
  for (const kind of ['notes', 'docs']) {
    for (const file of await readdir('src/content/' + kind, { recursive: true })) {
      if (/\.mdx?$/.test(file) && !file.endsWith('README.md') && !(kind === 'notes' && file === 'bio.md')) routes.add('/' + (kind === 'docs' ? 'documents' : 'notes') + '/' + file.replace(/\.mdx?$/, ''));
    }
  }
  const failures = [];
  const { writeFile, mkdir } = await import('node:fs/promises');
  await mkdir('/tmp/career-pwa-screenshots', { recursive: true });
  for (const width of (process.env.TEST_INTERACTIONS_ONLY ? [] : [375, 390, 430, 844])) {
    await send('Emulation.setDeviceMetricsOverride', { width, height: width === 844 ? 390 : 844, deviceScaleFactor: 1, mobile: true });
    for (const route of routes) {
      await navigate('Page.navigate', { url: base + route });
      await wait(`[...document.querySelectorAll('astro-island[component-url]')].every(el => !el.hasAttribute('ssr'))`);
      await pause(150);
      if (width === 390 && ['/', '/books', '/timeline', '/journal', '/companies', '/agents'].includes(route)) {
        const shot = await send('Page.captureScreenshot', { format: 'png' });
        await writeFile('/tmp/career-pwa-screenshots/' + (route.slice(1) || 'home') + '.png', Buffer.from(shot.data, 'base64'));
      }
      const result = await evaluate(`({width:innerWidth,scroll:document.documentElement.scrollWidth,title:document.title,overflow:[...document.querySelectorAll('main *')].filter(e=>e.getBoundingClientRect().right>innerWidth+1 && getComputedStyle(e).position!=='absolute').slice(0,8).map(e=>e.tagName+'.'+e.className)})`);
      if (result.scroll > width) failures.push({route,width,...result});
      if (width === 375 && ['/', '/notes', '/companies', '/documents'].includes(route)) {
        for (const href of await evaluate(`[...document.querySelectorAll('main a[href]')].map(a=>a.getAttribute('href')).filter(h=>['/notes/','/companies/','/documents/'].some(prefix=>h.startsWith(prefix)))`)) routes.add(href);
      }
    }
  }
  if (!process.env.TEST_INTERACTIONS_ONLY) console.log('Layout audit:', routes.size, 'routes at four phone sizes', JSON.stringify(failures, null, 2));
  assert.equal(failures.length, 0);
  await send('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  await navigate('Page.navigate', { url: base + '/' });
  await wait(`document.querySelector('#mobile-menu-button') !== null`);
  await evaluate(`document.querySelector('#mobile-menu-button').click()`);
  assert.equal(await evaluate(`document.activeElement.id`), 'mobile-menu-close');
  assert.equal(await evaluate(`document.querySelector('main').inert`), true);
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
  await send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
  assert.equal(await evaluate(`document.querySelector('#mobile-sidebar').inert`), true);
  assert.equal(await evaluate(`document.activeElement.id`), 'mobile-menu-button');
  await evaluate(`document.querySelector('#site-search').focus()`);
  const search = await evaluate(`document.querySelector('#search-panel').getBoundingClientRect().toJSON()`);
  assert.ok(search.x >= 0 && search.right <= 390);
  await navigate('Page.navigate', { url: base + '/books' });
  await wait(`document.querySelector('button.group') !== null`);
  await evaluate(`document.querySelector('button.group').click()`);
  await wait(`document.querySelector('.book-modal')?.open`);
  assert.ok(await evaluate(`document.querySelector('.book-modal').getBoundingClientRect().right <= innerWidth`));
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
  await send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
  await wait(`document.querySelector('.book-modal') === null`);
  await navigate('Page.navigate', { url: base + '/timeline' });
  await wait(`[...document.querySelectorAll('button')].some(b => b.textContent.includes('New goal') && !b.disabled)`);
  await evaluate(`[...document.querySelectorAll('button')].find(b => b.textContent.includes('New goal')).click()`);
  await wait(`document.querySelector('dialog').open`);
  assert.ok(await evaluate(`document.querySelector('dialog').open`));
  assert.ok(await evaluate(`document.querySelector('dialog').getBoundingClientRect().right <= innerWidth`));
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
  await send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
  await send('Network.clearBrowserCookies');
  await navigate('Page.navigate', { url: base + '/login' });
  assert.ok(await evaluate(`document.querySelector('#login-form') && document.documentElement.scrollWidth <= 390`));
  await wait(`navigator.serviceWorker.controller !== null`);
  const cached = await evaluate(`(async()=>{const c=await caches.open('career-public-v1');return (await c.keys()).map(r=>new URL(r.url).pathname)})()`);
  assert.deepEqual(cached.sort(), ['/offline.html','/manifest.webmanifest','/apple-touch-icon.png','/icons/icon-192.png','/icons/icon-512.png'].sort());
  const targets = await send('Target.getTargets');
  const worker = targets.targetInfos.find(t => t.type === 'service_worker' && t.url === base + '/sw.js');
  assert.ok(worker);
  const { sessionId: workerSession } = await send('Target.attachToTarget', { targetId: worker.targetId, flatten: true });
  await send('Network.enable', {}, workerSession);
  await send('Network.emulateNetworkConditions', { offline: true, latency: 0, downloadThroughput: 0, uploadThroughput: 0 }, workerSession);
  await send('Network.emulateNetworkConditions', { offline: true, latency: 0, downloadThroughput: 0, uploadThroughput: 0 });
  await navigate('Page.navigate', { url: base + '/notes/ostep' });
  assert.ok((await evaluate('document.body.innerText')).includes('You’re offline'));
  await send('Network.emulateNetworkConditions', { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 }, workerSession);
  await send('Network.emulateNetworkConditions', { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 });
  await navigate('Page.reload');
  assert.ok(!(await evaluate('document.body.innerText')).includes('You’re offline'));
  console.log('Navigation focus, search bounds, public-only caching, offline fallback, and reconnection passed.');

} finally {
  socket?.close();
  browser.kill();
}
