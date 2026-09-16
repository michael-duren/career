import { readMcpAuthConfig, MCP_SCOPE, type McpAuthConfig } from './mcp-auth.ts';

export function handleMcpDiscovery(request: Request, config?: McpAuthConfig): Response {
  const headers = { 'Content-Type': 'application/json', 'Cache-Control': 'no-store', 'Access-Control-Allow-Origin': '*', 'X-Content-Type-Options': 'nosniff' };
  // This document is intentionally public and contains no credentials or user identifiers.
  if (!['GET', 'HEAD', 'OPTIONS'].includes(request.method)) return new Response(null, { status: 405, headers: { ...headers, Allow: 'GET, HEAD, OPTIONS' } });
  if (request.method === 'OPTIONS') return new Response(null, { status: 204, headers: { ...headers, 'Access-Control-Allow-Methods': 'GET, HEAD, OPTIONS', 'Access-Control-Allow-Headers': 'MCP-Protocol-Version' } });
  try {
    config ??= readMcpAuthConfig();
    if (config.mode !== 'oauth') throw new Error('OAuth disabled');
    const body = JSON.stringify({ resource: config.resource, authorization_servers: [config.issuer], scopes_supported: [MCP_SCOPE], bearer_methods_supported: ['header'] });
    return new Response(request.method === 'HEAD' ? null : body, { headers });
  } catch { return new Response(request.method === 'HEAD' ? null : JSON.stringify({ error: 'oauth_not_configured' }), { status: 503, headers }); }
}
