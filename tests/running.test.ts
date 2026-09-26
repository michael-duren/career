import test from 'node:test';
import assert from 'node:assert/strict';
import { indexedDB } from 'fake-indexeddb';
import { enqueueClip, queuedClips, removeClip, pauseSync, syncClips, takesForThought, type QueuedClip } from '../src/lib/running-queue.ts';
import { agentContext, journalMarkdown } from '../src/lib/agent-context.ts';
import { careerEntries } from '../src/lib/mcp-server.ts';
import { searchEntries } from '../src/lib/search.ts';
import type { Workspace } from '../src/lib/workspace.ts';

test('three offline clips survive failed and lost-response uploads, replay once by UUID', async () => {
  Object.defineProperty(globalThis, 'indexedDB', { value: indexedDB, configurable: true });
  Object.defineProperty(globalThis, 'window', { value: new EventTarget(), configurable: true });
  const original = globalThis.fetch;
  try {
    for (let i = 0; i < 3; i++) await enqueueClip({ clientId: `clip-${i}`, recordedAt: `2026-09-19T10:0${i}:00Z`, durationMs: 1000, audio: new Blob(['audio'], { type: 'audio/webm' }) });
    globalThis.fetch = async () => { throw new Error('offline'); };
    await assert.rejects(syncClips(() => {}), /offline/);
    assert.equal((await queuedClips()).length, 3);
    globalThis.fetch = async () => new Response('<html>login</html>', { status: 200 });
    await assert.rejects(syncClips(() => {}));
    assert.equal((await queuedClips()).length, 3, 'unverified successful response must retain audio');
    const stored = new Map<string, Blob>(); let lostResponse = true;
    globalThis.fetch = async (_url, init) => {
      const form = init?.body as FormData, id = String(form.get('clientId'));
      stored.set(id, form.get('audio') as Blob);
      if (lostResponse) { lostResponse = false; throw new Error('response lost'); }
      return new Response(JSON.stringify({ clipId: id, noteId: 'run-one' }), { status: 200 });
    };
    await assert.rejects(syncClips(() => {}), /response lost/);
    assert.equal((await queuedClips()).length, 3);
    await Promise.all([syncClips(() => {}), syncClips(() => {})]);
    assert.equal((await queuedClips()).length, 0);
    assert.equal(stored.size, 3);
    assert.equal(await stored.get('clip-0')!.text(), 'audio');
  } finally { globalThis.fetch = original; }
});

test('deleting a thought finds unsent takes the server would group into it', () => {
  const take = (clientId: string, recordedAt: string, noteId?: string): QueuedClip => ({ clientId, recordedAt, noteId, durationMs: 1000, audio: new Blob(['a']) });
  const window = 90 * 60 * 1000;
  const queued = [
    take('near', '2026-09-19T11:30:00Z'),
    take('edge', '2026-09-19T08:30:00Z'),
    take('far', '2026-09-19T13:00:01Z'),
    take('addressed', '2026-09-20T10:00:00Z', 'run-one'),
    take('other-note', '2026-09-19T10:05:00Z', 'run-two'),
    take('bad-time', 'not a date'),
  ];
  const ids = (clips: QueuedClip[]) => clips.map(c => c.clientId).sort();
  assert.deepEqual(ids(takesForThought(queued, 'run-one', ['2026-09-19T10:00:00Z', '2026-09-19T11:30:00Z'], window)), ['addressed', 'edge', 'near']);
  assert.deepEqual(ids(takesForThought(queued, 'run-one', [], window)), ['addressed'], 'a typed thought only owns addressed takes');
  const chained = [take('a', '2026-09-19T11:20:00Z'), take('b', '2026-09-19T12:30:00Z'), take('c', '2026-09-19T14:01:00Z')];
  assert.deepEqual(ids(takesForThought(chained, 'run-one', ['2026-09-19T10:00:00Z'], window)), ['a', 'b'], 'takes chain through earlier backlog takes, like server grouping');
});

test('a take discarded while a sync is running is not uploaded', async () => {
  Object.defineProperty(globalThis, 'indexedDB', { value: indexedDB, configurable: true });
  Object.defineProperty(globalThis, 'window', { value: new EventTarget(), configurable: true });
  const original = globalThis.fetch;
  try {
    for (const clip of await queuedClips()) await removeClip(clip.clientId);
    await enqueueClip({ clientId: 'kept', recordedAt: '2026-09-19T10:00:00Z', durationMs: 1000, audio: new Blob(['a'], { type: 'audio/webm' }) });
    await enqueueClip({ clientId: 'discarded', recordedAt: '2026-09-19T10:05:00Z', durationMs: 1000, audio: new Blob(['a'], { type: 'audio/webm' }) });
    const uploaded: string[] = [];
    globalThis.fetch = async (_url, init) => {
      const id = String((init?.body as FormData).get('clientId'));
      uploaded.push(id);
      // The thought is deleted while the first take uploads.
      await removeClip('discarded');
      return new Response(JSON.stringify({ clipId: id, noteId: 'run-one' }), { status: 200 });
    };
    await syncClips(() => {});
    assert.deepEqual(uploaded, ['kept']);
    assert.equal((await queuedClips()).length, 0);
  } finally { globalThis.fetch = original; }
});

test('pausing sync waits only for the take in flight, holds the rest until resumed', async () => {
  Object.defineProperty(globalThis, 'indexedDB', { value: indexedDB, configurable: true });
  Object.defineProperty(globalThis, 'window', { value: new EventTarget(), configurable: true });
  const original = globalThis.fetch;
  try {
    for (const clip of await queuedClips()) await removeClip(clip.clientId);
    for (let i = 0; i < 3; i++) await enqueueClip({ clientId: `take-${i}`, recordedAt: `2026-09-19T10:0${i}:00Z`, durationMs: 1000, audio: new Blob(['a'], { type: 'audio/webm' }) });
    const uploaded: string[] = [];
    let release!: () => void, started!: () => void;
    const inFlight = new Promise<void>(resolve => { started = resolve; });
    globalThis.fetch = async (_url, init) => {
      const id = String((init?.body as FormData).get('clientId'));
      uploaded.push(id);
      if (id === 'take-0') { started(); await new Promise<void>(resolve => { release = resolve; }); }
      return new Response(JSON.stringify({ clipId: id, noteId: 'run-one' }), { status: 200 });
    };
    const running = syncClips(() => {});
    await inFlight;
    const pause = pauseSync();
    assert.equal(pause.uploading, true);
    let settled = false;
    void pause.settled.then(() => { settled = true; });
    await new Promise(resolve => setTimeout(resolve, 10));
    assert.equal(settled, false, 'waits for the take being uploaded');
    release();
    await pause.settled; await running;
    assert.deepEqual(uploaded, ['take-0'], 'stops before the next take');
    await syncClips(() => {});
    assert.deepEqual(uploaded, ['take-0'], 'a retry while paused uploads nothing');
    assert.equal((await queuedClips()).length, 2);
    const idle = pauseSync();
    assert.equal(idle.uploading, false);
    pause.resume(); pause.resume();
    await syncClips(() => {});
    assert.deepEqual(uploaded, ['take-0'], 'still paused by the second holder; double resume counts once');
    idle.resume();
    await syncClips(() => {});
    assert.deepEqual(uploaded, ['take-0', 'take-1', 'take-2']);
    assert.equal((await queuedClips()).length, 0);
  } finally { globalThis.fetch = original; }
});

test('running notes are searchable and explicitly opt in to agent and MCP context', () => {
  const data: Workspace = { version: 2, notes: [], weeks: [], books: [], companies: [], documents: [], runningNotes: [{ id: 'run-one', title: 'Morning run', runDate: '2026-09-19', startedAt: '2026-09-19T10:00:00Z', tags: [], body: 'Private running thoughts' }] };
  assert.equal(searchEntries(data).find(e => e.kind === 'Audio thought')?.href, '/audio-thoughts?id=run-one');
  assert.equal(data.notes.length, 0);
  assert.equal(JSON.stringify(agentContext({ data, revision: null })).includes('Private running'), false);
  assert.equal(careerEntries({ data, revision: null }).some(e => e.kind === 'running'), false);
  assert.equal(careerEntries({ data, revision: null }, 'running').length, 1);
  assert.match(journalMarkdown(data, 'running'), /Private running thoughts/);
});

test('service worker caches only data-free audio thoughts shell and public assets; private APIs stay network-only', async () => {
  const { readFile } = await import('node:fs/promises');
  const { runInNewContext } = await import('node:vm');
  const listeners: Record<string, (event: any) => void> = {};
  const stored = new Map<string, Response>();
  const key = (request: string | { url: string }) => typeof request === 'string' ? request : new URL(request.url).pathname;
  let offline = false, loginRedirect = false;
  runInNewContext(await readFile(new URL('../public/sw.js', import.meta.url), 'utf8'), {
    URL, Response,
    self: { location: { origin: 'https://career.example' }, addEventListener: (name: string, fn: (event: any) => void) => { listeners[name] = fn; } },
    caches: { open: async () => ({ put: async (request: any, response: Response) => stored.set(key(request), response) }), match: async (request: any) => stored.get(key(request)) },
    fetch: async (request: any) => { if (offline) throw new Error('offline'); const response = new Response('data-free shell'); Object.defineProperty(response, 'url', { value: loginRedirect ? 'https://career.example/login' : request.url }); Object.defineProperty(response, 'redirected', { value: loginRedirect }); return response; },
  });
  const fetchEvent = (path: string, mode = 'navigate') => { let result: Promise<Response> | undefined; listeners.fetch({ request: { url: `https://career.example${path}`, method: 'GET', mode }, respondWith: (promise: Promise<Response>) => { result = promise; } }); return result; };
  assert.equal(fetchEvent('/api/running/run/status', 'cors'), undefined);
  assert.equal(fetchEvent('/api/running/clips/run/clip/audio', 'cors'), undefined);
  await fetchEvent('/personal-journal');
  assert.equal(stored.has('/personal-journal'), false);
  loginRedirect = true; await fetchEvent('/audio-thoughts'); assert.equal(stored.has('/audio-thoughts'), false);
  loginRedirect = false; await fetchEvent('/audio-thoughts?id=run'); assert.equal(stored.has('/audio-thoughts'), true);
  offline = true; assert.equal(await (await fetchEvent('/audio-thoughts'))!.clone().text(), 'data-free shell');
  assert.equal(await (await fetchEvent('/running'))!.text(), 'data-free shell');
});

test('service worker upgrade carries the pre-rename recorder shell and its assets into the new cache', async () => {
  const { readFile } = await import('node:fs/promises');
  const { runInNewContext } = await import('node:vm');
  const origin = 'https://career.example', stores = new Map<string, Map<string, string>>(), listeners: Record<string, (event: any) => void> = {};
  const key = (request: string | { url: string }) => new URL(typeof request === 'string' ? request : request.url, origin).pathname;
  const open = async (name: string) => {
    if (!stores.has(name)) stores.set(name, new Map());
    const entries = stores.get(name)!;
    return { put: async (request: any, value: string) => void entries.set(key(request), value), match: async (request: any) => entries.get(key(request)), keys: async () => [...entries.keys()].map(path => ({ url: origin + path })) };
  };
  runInNewContext(await readFile(new URL('../public/sw.js', import.meta.url), 'utf8'), {
    URL,
    self: { location: { origin }, clients: { claim: async () => {} }, addEventListener: (name: string, fn: (event: any) => void) => { listeners[name] = fn; } },
    caches: { open, keys: async () => [...stores.keys()], delete: async (name: string) => stores.delete(name) },
  });
  const old = await open('career-public-v2');
  await old.put('/running', 'old shell'); await old.put('/_astro/recorder.abc.js', 'old js'); await old.put('/offline.html', 'offline');
  let done: Promise<void> | undefined;
  listeners.activate({ waitUntil: (promise: Promise<void>) => { done = promise; } });
  await done;
  assert.deepEqual([...stores.keys()], ['career-public-v3']);
  assert.deepEqual(Object.fromEntries(stores.get('career-public-v3')!), { '/audio-thoughts': 'old shell', '/_astro/recorder.abc.js': 'old js' });
});

test('audio thought titles match the server: first six words, "..." when cut, generated titles replaceable', async () => {
  const { thoughtTitle, isAutoTitle, thoughtExcerpt } = await import('../src/lib/audio-thoughts.ts');
  const cases: Record<string, string> = {
    '': '',
    '  short idea  ': 'short idea',
    'one two three four five six': 'one two three four five six',
    'one two three four five six, seven': 'one two three four five six...',
    '\n\nso I was thinking about\nthe queue design today': 'so I was thinking about the...',
    ['x'.repeat(80)]: `${'x'.repeat(60)}...`,
  };
  for (const [body, want] of Object.entries(cases)) assert.equal(thoughtTitle(body), want);
  for (const title of ['New audio thought', 'Run 2026-09-20 07:15', 'Audio thought 2026-09-23']) assert.equal(isAutoTitle(title), true);
  assert.equal(isAutoTitle('Tempo notes'), false);
  assert.equal(thoughtExcerpt({ excerpt: 'from list' }), 'from list');
  assert.equal(thoughtExcerpt({ body: 'full\n\nbody  text' }), 'full body text');
});
