import test from 'node:test';
import assert from 'node:assert/strict';
import { indexedDB } from 'fake-indexeddb';
import { enqueueClip, queuedClips, syncClips } from '../src/lib/running-queue.ts';
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
