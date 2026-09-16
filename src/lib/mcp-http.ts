import { randomUUID } from 'node:crypto';
import { WebStandardStreamableHTTPServerTransport } from '@modelcontextprotocol/sdk/server/webStandardStreamableHttp.js';
import { createCareerMcpServer } from './mcp-server.ts';
import type { Snapshot } from './workspace.ts';
import { authenticateMcp, readMcpAuthConfig, mcpChallenge, McpAuthError, type McpAuthConfig, type McpTokenVerifier } from './mcp-auth.ts';

const headers = { 'Cache-Control': 'private, no-store', 'Content-Type': 'application/json; charset=utf-8', 'X-Content-Type-Options': 'nosniff' };
const failure = (status: number, error: string, extra = {}) => new Response(JSON.stringify({ error }), { status, headers: { ...headers, ...extra } });
type Audit = { requestId: string; status: number; clientId?: string; tool?: string };
type Options = { config?: McpAuthConfig; verify?: McpTokenVerifier; audit?: (event: Audit) => void };
const allowedHeaders = ['authorization', 'content-type', 'accept', 'mcp-protocol-version', 'mcp-session-id'];

export async function handleCareerMcp(request: Request, read: () => Promise<Snapshot>, options: Options = {}): Promise<Response> {
  let config: McpAuthConfig = { mode: 'disabled' };
  const event: Audit = { requestId: randomUUID(), status: 500 };
  const cors: Record<string, string> = { Vary: 'Origin', 'X-Request-Id': event.requestId };
  const origin = request.headers.get('origin');
  let response: Response;
  try {
    config = options.config ?? readMcpAuthConfig();
    const expectedOrigin = config.mode === 'oauth' ? new URL(config.resource).origin : new URL(request.url).origin;
    if (origin && origin !== expectedOrigin && !(config.mode !== 'disabled' && config.allowedOrigins.includes(origin))) throw new McpAuthError(403, 'origin_not_allowed');
    if (origin) Object.assign(cors, { 'Access-Control-Allow-Origin': origin, 'Access-Control-Expose-Headers': 'WWW-Authenticate, X-Request-Id' });
    if (config.mode === 'disabled') throw new McpAuthError(503, 'mcp_disabled');
    if (request.method === 'OPTIONS') {
      const requestedHeaders = (request.headers.get('access-control-request-headers') ?? '').toLowerCase().split(',').map(s => s.trim()).filter(Boolean);
      if (request.headers.get('access-control-request-method') !== 'POST' || requestedHeaders.some(h => !allowedHeaders.includes(h))) throw new McpAuthError(403, 'preflight_not_allowed');
      response = new Response(null, { status: 204, headers: { ...headers, 'Access-Control-Allow-Methods': 'POST', 'Access-Control-Allow-Headers': allowedHeaders.join(', ') } });
    } else {
      const principal = await authenticateMcp(request, config, options.verify);
      // Client identifiers are provider-validated; bound length even for a misconfigured issuer.
      event.clientId = principal.clientId?.slice(0, 200);
      response = await handleAuthorizedMcp(request, read, event);
    }
  } catch (error) {
    const denied = error instanceof McpAuthError ? error : new McpAuthError(503, 'oauth_unavailable');
    response = failure(denied.status, denied.code, [401, 403].includes(denied.status) && !denied.code.includes('origin') && !denied.code.includes('preflight')
      ? { 'WWW-Authenticate': mcpChallenge(config, denied.code) } : {});
  }
  for (const [key, value] of Object.entries(cors)) response.headers.set(key, value);
  event.status = response.status;
  // Never log request bodies, URLs, subjects, credentials, or provider errors.
  try { (options.audit ?? (record => console.info(JSON.stringify({ event: 'career_mcp', ...record }))))(event); } catch { /* Logging must not change authorization. */ }
  return response;
}

async function handleAuthorizedMcp(request: Request, read: () => Promise<Snapshot>, event: Audit): Promise<Response> {
  if (request.method !== 'POST') return failure(405, 'Use POST for this stateless MCP endpoint.', { Allow: 'POST' });
  if (!request.headers.get('content-type')?.toLowerCase().startsWith('application/json')) return failure(415, 'Use application/json.');
  if (Number(request.headers.get('content-length')) > 65536) return failure(413, 'Request is too large.');
  // Bound streaming bodies as well as requests declaring Content-Length.
  const reader = request.body?.getReader();
  let size = 0;
  const chunks: Uint8Array[] = [];
  if (reader) {
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        size += value.byteLength;
        if (size > 65536) { await reader.cancel(); return failure(413, 'Request is too large.'); }
        chunks.push(value);
      }
    } catch { return failure(400, 'Could not read request.'); }
    finally { reader.releaseLock(); }
  }
  let body: unknown;
  try { body = JSON.parse(Buffer.concat(chunks).toString('utf8')); }
  catch { return failure(400, 'Invalid JSON.'); }
  const call = body as { method?: unknown; params?: { name?: unknown } } | null;
  if (call?.method === 'tools/call' && typeof call.params?.name === 'string' &&
      ['get_career_overview', 'list_career_entries', 'search_career_context', 'read_career_entry'].includes(call.params.name)) event.tool = call.params.name;
  const server = createCareerMcpServer(read);
  const transport = new WebStandardStreamableHTTPServerTransport({ sessionIdGenerator: undefined, enableJsonResponse: true });
  try {
    await server.connect(transport);
    const response = await transport.handleRequest(request, { parsedBody: body });
    // Materialize JSON before closing per-request transport (no long-lived Netlify sessions).
    const bytes = response.body ? await response.arrayBuffer() : null;
    const responseHeaders = new Headers(response.headers);
    responseHeaders.set('Cache-Control', 'private, no-store');
    responseHeaders.set('X-Content-Type-Options', 'nosniff');
    return new Response(bytes, { status: response.status, headers: responseHeaders });
  } catch { return failure(500, 'MCP request failed. Retry.'); }
  finally { await server.close(); }
}
