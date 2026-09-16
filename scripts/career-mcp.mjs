#!/usr/bin/env node
// Local stdio facade with OAuth on its remote HTTP connection.
import { Client } from '@modelcontextprotocol/sdk/client/index.js';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js';
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { ListToolsRequestSchema, CallToolRequestSchema, ListResourcesRequestSchema, ReadResourceRequestSchema, ListPromptsRequestSchema, GetPromptRequestSchema } from '@modelcontextprotocol/sdk/types.js';
import { createCareerOAuth, mcpUrl } from './lib/mcp-oauth.mjs';

async function main() {
  const url = mcpUrl(process.env.CAREER_MCP_URL);
  const command = process.argv[2];
  if (command && !['login', 'logout'].includes(command)) throw new Error('Unknown command');
  const oauth = await createCareerOAuth(url);
  if (command) { await oauth[command](); console.error(command === 'login' ? 'Career MCP authorized.' : 'Career MCP grant revoked and local credentials cleared.'); return; }
  const remote = new Client({ name: 'career-local-bridge', version: '1.0.0' });
  await remote.connect(new StreamableHTTPClientTransport(url, {
    fetch: oauth.fetch, requestInit: { redirect: 'error' },
  }));
  const server = new Server({ name: 'career-strategy', version: '1.0.0' }, {
    capabilities: { tools: {}, resources: {}, prompts: {} },
    instructions: remote.getInstructions(),
  });
  server.setRequestHandler(ListToolsRequestSchema, request => remote.listTools(request.params));
  server.setRequestHandler(CallToolRequestSchema, request => remote.callTool(request.params));
  server.setRequestHandler(ListResourcesRequestSchema, request => remote.listResources(request.params));
  server.setRequestHandler(ReadResourceRequestSchema, request => remote.readResource(request.params));
  server.setRequestHandler(ListPromptsRequestSchema, request => remote.listPrompts(request.params));
  server.setRequestHandler(GetPromptRequestSchema, request => remote.getPrompt(request.params));
  let closing = false;
  const close = async () => {
    if (closing) return;
    closing = true;
    await server.close();
    await remote.close();
  };
  server.onclose = () => { void close(); };
  for (const signal of ['SIGINT', 'SIGTERM']) process.once(signal, () => { void close(); });
  await server.connect(new StdioServerTransport());
}
main().catch(error => {
  // Never print tokens, response bodies, or credential-bearing URLs to client logs.
  const actionable = [
    'Run career-mcp.mjs login to authorize this client',
    'Provider did not issue a refresh token; enable offline access and authorize again',
    'OAuth credential store is busy',
    'Unsafe OAuth credential permissions',
    'OAuth issuer changed',
    'OAuth callback rejected',
    'OAuth login timed out',
  ];
  if (actionable.includes(error?.message)) console.error(error.message);
  console.error('Career MCP OAuth failed. Check the server URL, provider configuration, and credential directory permissions. Run node scripts/career-mcp.mjs login to authorize again. For logout failures, revoke the grant in the provider dashboard.');
  process.exitCode = 1;
});
