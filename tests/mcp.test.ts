import test from 'node:test';
import assert from 'node:assert/strict';
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import { handleCareerMcp } from '../src/lib/mcp-http.ts';
import type { Snapshot } from '../src/lib/workspace.ts';

const token = 'test-only-private-career-token-1234567890';
const url = 'https://career.example/api/mcp';
const options = { config: { mode: 'legacy' as const, token, allowedOrigins: [] }, audit: () => {} };
function fixture(): Snapshot {
  return { revision: 'r1', data: { version: 2, notes: [{ id: 'database', title: 'Database notes', topic: 'database', description: '', tags: [], body: 'Raft and storage engines' }], documents: [], books: [], companies: [], weeks: [], personalJournal: [{ id: 'reflection', title: 'Reflection', date: '2026-09-13', description: '', tags: [], body: 'I enjoy building systems' }], goals: [{ id: 'goal-1', title: 'Build a database', startDate: '2026-09-01', endDate: '2026-12-01', color: '#67e8f9', dailyHours: 1, createdAt: '2026-09-01T00:00:00Z', updatedAt: '2026-09-01T00:00:00Z', steps: [], metadata: {}, notes: [] }] } };
}
const request = (body: unknown, headers = {}, method = 'POST') => new Request(url, { method, headers: { authorization: `Bearer ${token}`, 'content-type': 'application/json', accept: 'application/json, text/event-stream', ...headers }, ...(method === 'POST' ? { body: JSON.stringify(body) } : {}) });
const initialize = { jsonrpc: '2.0', id: 1, method: 'initialize', params: { protocolVersion: '2025-11-25', capabilities: {}, clientInfo: { name: 'test', version: '1' } } };

test('private endpoint fails closed without reading data, independent of browser cookies', async () => {
  let reads = 0;
  const read = async () => { reads++; return fixture(); };
  for (const [headers, status] of [
    [{ authorization: '' }, 401],
    [{ authorization: 'Bearer wrong', cookie: 'auth_token=website-session' }, 401],
    [{ origin: 'https://untrusted.example' }, 403],
    [{ origin: 'null' }, 403],
    [{ 'content-type': 'text/plain' }, 415],
  ] as const) {
    const response = await handleCareerMcp(request(initialize, headers), read, options);
    assert.equal(response.status, status);
    assert.equal(response.headers.get('cache-control'), 'private, no-store');
  }
  assert.equal((await handleCareerMcp(request(initialize), read, { config: { mode: 'disabled' }, audit: () => {} })).status, 503);
  assert.equal((await handleCareerMcp(request(initialize), read, { config: { mode: 'disabled' }, audit: () => {} })).status, 503);
  assert.equal((await handleCareerMcp(request(null, {}, 'GET'), read, options)).status, 405);
  assert.equal((await handleCareerMcp(request('x'.repeat(70000)), read, options)).status, 413);
  assert.equal((await handleCareerMcp(new Request(url, { method: 'POST', headers: request({}).headers, body: '{' }), read, options)).status, 400);
  assert.equal(reads, 0);
});

test('SDK client exercises initialization, read-only discovery, live reads, search, chunking, resources and prompts', async () => {
  let snapshot = fixture();
  const client = new Client({ name: 'test', version: '1' });
  const transport = new StreamableHTTPClientTransport(new URL(url), {
    requestInit: { headers: { authorization: `Bearer ${token}` } },
    fetch: async (input, init) => handleCareerMcp(new Request(input, init), async () => snapshot, options),
  });
  await client.connect(transport);
  try {
    const { tools } = await client.listTools();
    assert.equal(tools.length, 4);
    assert.ok(tools.every(t => t.annotations?.readOnlyHint && t.annotations?.destructiveHint === false));
    assert.match(client.getInstructions()!, /historical context/);
    const call = async (name: string, args = {}) => {
      const response = await client.callTool({ name, arguments: args });
      const content = response.content as { type: string; text: string }[];
      return { response, data: JSON.parse(content[0].text) };
    };
    const overview = await call('get_career_overview');
    assert.equal(overview.data.goals[0].startDate, '2026-09-01');
    assert.equal(overview.data.counts.personal_journal, 1);
    snapshot = { ...snapshot, revision: 'r2', data: { ...snapshot.data, goals: [] } };
    assert.deepEqual((await call('get_career_overview')).data.goals, []);
    const list = (await call('list_career_entries', { kind: 'personal_journal', limit: 1 })).data;
    assert.equal(list.entries[0].id, 'reflection');
    assert.equal(list.revision, 'r2');
    const search = (await call('search_career_context', { query: 'storage engines' })).data;
    assert.equal(search.matches[0].id, 'database');
    const first = (await call('read_career_entry', { kind: 'note', id: 'database', length: 10 })).data;
    assert.equal(first.text.length, 10);
    assert.equal(first.nextOffset, 10);
    const rest = (await call('read_career_entry', { kind: 'note', id: 'database', offset: 10, revision: 'r2' })).data;
    assert.equal(JSON.parse(first.text + rest.text).body, 'Raft and storage engines');
    assert.equal((await call('read_career_entry', { kind: 'note', id: 'database', revision: 'r1' })).response.isError, true);
    assert.equal((await call('read_career_entry', { kind: 'goal', id: 'goal-1' })).response.isError, true);
    assert.equal((await client.callTool({ name: 'list_career_entries', arguments: { limit: 500 } })).isError, true);
    const resources = await client.listResources();
    assert.equal(resources.resources[0].uri, 'career://overview');
    const resource = await client.readResource({ uri: 'career://overview' });
    assert.ok('text' in resource.contents[0]);
    assert.match(resource.contents[0].text, /r2/);
    assert.equal((await client.listPrompts()).prompts[0].name, 'career_conversation');
    const prompt = await client.getPrompt({ name: 'career_conversation', arguments: { topic: 'applications' } });
    assert.match(JSON.stringify(prompt), /applications/);
  } finally { await client.close(); }
});

test('storage failures produce an error instead of an empty career plan or leaked error text', async () => {
  const body = { jsonrpc: '2.0', id: 2, method: 'tools/call', params: { name: 'get_career_overview', arguments: {} } };
  const response = await handleCareerMcp(request(body), async () => { throw new Error('private storage credential'); }, options);
  const result = await response.json();
  assert.equal(result.result.isError, true);
  assert.doesNotMatch(JSON.stringify(result), /private storage credential/);
  const resource = await handleCareerMcp(request({ jsonrpc: '2.0', id: 3, method: 'resources/read', params: { uri: 'career://overview' } }), async () => { throw new Error('private storage credential'); }, options);
  const failed = await resource.json();
  assert.ok(failed.error);
  assert.doesNotMatch(JSON.stringify(failed), /private storage credential/);
});
