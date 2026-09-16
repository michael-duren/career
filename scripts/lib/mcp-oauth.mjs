import { createHash, randomBytes, timingSafeEqual } from 'node:crypto';
import { createServer } from 'node:http';
import { mkdir, open, readFile, rename, rm, lstat } from 'node:fs/promises';
import { join } from 'node:path';
import { homedir } from 'node:os';
import { auth, discoverOAuthProtectedResourceMetadata, discoverAuthorizationServerMetadata, selectClientAuthMethod } from '@modelcontextprotocol/sdk/client/auth.js';

const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
const hash = value => createHash('sha256').update(value).digest('hex');
const loopback = url => ['127.0.0.1', 'localhost', '[::1]'].includes(url.hostname);
export function mcpUrl(value) {
  const url = new URL(value || '');
  if (url.username || url.password || url.search || url.hash || url.pathname !== '/api/mcp' ||
      !(url.protocol === 'https:' || url.protocol === 'http:' && loopback(url))) throw new Error('Invalid MCP URL');
  return url;
}

function safeUrl(value, local) {
  const url = new URL(value);
  if (url.username || url.password || url.hash || !(url.protocol === 'https:' && (local || !loopback(url)) || local && url.protocol === 'http:' && loopback(url))) throw new Error('Unsafe OAuth URL');
  return url;
}

export async function discoverCareerOAuth(resource, expectedIssuer, fetchImpl = fetch) {
  const local = loopback(resource) && resource.protocol === 'http:';
  const safeFetch = (input, init = {}) => fetchImpl(safeUrl(String(input), local), { ...init, redirect: 'error', signal: init.signal ?? AbortSignal.timeout(10000) });
  const resourceMetadata = await discoverOAuthProtectedResourceMetadata(resource, undefined, safeFetch);
  if (resourceMetadata.resource !== resource.href || resourceMetadata.authorization_servers?.length !== 1) throw new Error('Invalid MCP discovery');
  const issuer = resourceMetadata.authorization_servers[0];
  safeUrl(issuer, local);
  if (expectedIssuer && issuer !== expectedIssuer) throw new Error('OAuth issuer changed');
  const metadata = await discoverAuthorizationServerMetadata(issuer, { fetchFn: safeFetch });
  if (!metadata || metadata.issuer !== issuer || !metadata.code_challenge_methods_supported?.includes('S256')) throw new Error('Invalid OAuth metadata');
  for (const key of ['authorization_endpoint', 'token_endpoint', 'registration_endpoint', 'revocation_endpoint']) if (metadata[key]) safeUrl(metadata[key], local);
  if (!metadata.authorization_endpoint || !metadata.token_endpoint) throw new Error('Missing OAuth endpoints');
  return { state: { authorizationServerUrl: issuer, resourceMetadata, authorizationServerMetadata: metadata }, safeFetch };
}

async function privatePath(path, directory) {
  const info = await lstat(path);
  if (info.isSymbolicLink() || (directory ? !info.isDirectory() : !info.isFile()) ||
      process.getuid && info.uid !== process.getuid() || (info.mode & 0o077) !== 0) throw new Error('Unsafe OAuth credential permissions');
}

// A protected file store is the portable fallback; no secrets enter the repo or stdout.
export async function credentialStore(directory, key) {
  await mkdir(directory, { recursive: true, mode: 0o700 });
  await privatePath(directory, true);
  const file = join(directory, `${hash(key)}.json`);
  const lock = `${file}.lock`;
  return {
    async locked(callback) {
      const deadline = Date.now() + 15000;
      while (true) {
        try { await mkdir(lock, { mode: 0o700 }); break; }
        catch (error) {
          if (error.code !== 'EEXIST') throw error;
          await privatePath(lock, true);
          // Never steal a lock by age: login can be waiting on a human, and refresh may be in flight.
          if (Date.now() >= deadline) throw new Error('OAuth credential store is busy');
          await pause(50);
        }
      }
      try {
        await open(join(lock, 'pid'), 'wx', 0o600).then(async handle => { try { await handle.writeFile(String(process.pid)); } finally { await handle.close(); } });
        let data = {};
        try { await privatePath(file, false); data = JSON.parse(await readFile(file, 'utf8')); }
        catch (error) { if (error.code !== 'ENOENT') throw error; }
        const save = async () => {
          const temp = `${file}.${randomBytes(12).toString('hex')}.tmp`;
          const handle = await open(temp, 'wx', 0o600);
          try { await handle.writeFile(JSON.stringify(data)); await handle.sync(); } finally { await handle.close(); }
          try { await rename(temp, file); } finally { await rm(temp, { force: true }); }
        };
        return await callback(data, save);
      } finally { await rm(lock, { recursive: true, force: true }); }
    },
  };
}

export function validateCallback(url, redirectUrl, state, metadata) {
  if (url.origin !== new URL(redirectUrl).origin || url.pathname !== new URL(redirectUrl).pathname) throw new Error('Invalid callback');
  for (const key of ['state', 'iss', 'code', 'error']) if (url.searchParams.getAll(key).length > 1) throw new Error('Duplicate callback parameter');
  const received = url.searchParams.get('state') ?? '';
  if (received.length !== state.length || !timingSafeEqual(Buffer.from(received), Buffer.from(state))) throw new Error('Invalid OAuth state');
  const issuer = url.searchParams.get('iss');
  if (issuer !== null && issuer !== metadata.issuer || metadata.authorization_response_iss_parameter_supported && issuer !== metadata.issuer) throw new Error('Invalid OAuth issuer');
  if (url.searchParams.has('error')) throw new Error('Authorization declined');
  const code = url.searchParams.get('code');
  if (!code || code.length > 8192) throw new Error('Missing authorization code');
  return code;
}

export async function createCareerOAuth(resource, env = process.env, fetchImpl = fetch) {
  const { state: discovery, safeFetch } = await discoverCareerOAuth(resource, env.CAREER_MCP_OAUTH_ISSUER, fetchImpl);
  const metadata = discovery.authorizationServerMetadata;
  const machine = env.CAREER_MCP_GRANT_TYPE === 'client_credentials';
  if (env.CAREER_MCP_GRANT_TYPE && !['authorization_code', 'client_credentials'].includes(env.CAREER_MCP_GRANT_TYPE)) throw new Error('Invalid grant type');
  const clientId = env.CAREER_MCP_CLIENT_ID;
  const clientSecret = env.CAREER_MCP_CLIENT_SECRET;
  if (machine && (!clientId || !clientSecret)) throw new Error('Machine OAuth requires registered client credentials');
  const scopes = env.CAREER_MCP_SCOPES || 'career:read';
  if (!scopes.split(/\s+/).includes('career:read')) throw new Error('Missing career:read scope');
  const store = await credentialStore(env.CAREER_MCP_CREDENTIAL_DIR || join(homedir(), '.career-mcp'), JSON.stringify([resource.href, metadata.issuer, clientId || 'dynamic', machine]));

  function provider(data, save, redirectUrl, onRedirect) {
    let verifier;
    const state = randomBytes(32).toString('base64url');
    return {
      redirectUrl: machine ? undefined : redirectUrl || data.redirectUrl || 'http://127.0.0.1/callback',
      clientMetadata: { client_name: 'Career MCP local bridge', redirect_uris: machine ? [] : [redirectUrl || data.redirectUrl || 'http://127.0.0.1/callback'],
        grant_types: machine ? ['client_credentials'] : ['authorization_code', 'refresh_token'], response_types: machine ? [] : ['code'],
        token_endpoint_auth_method: clientSecret ? 'client_secret_basic' : 'none', scope: scopes },
      state: () => state,
      discoveryState: () => discovery,
      validateResourceURL: async (_server, value) => { if (value !== resource.href) throw new Error('Resource mismatch'); return resource; },
      clientInformation: () => clientId ? { client_id: clientId, ...(clientSecret ? { client_secret: clientSecret } : {}) } : data.client,
      saveClientInformation: async info => {
        if (data.client?.client_id !== info.client_id) { delete data.tokens; delete data.expiresAt; }
        data.client = info; data.issuer = metadata.issuer; data.resource = resource.href; await save();
      },
      tokens: () => data.tokens,
      saveTokens: async tokens => {
        if (tokens.token_type.toLowerCase() !== 'bearer' || !Number.isFinite(tokens.expires_in) || tokens.expires_in <= 0) throw new Error('Invalid OAuth token response');
        if (redirectUrl && !tokens.refresh_token) throw new Error('Provider did not issue a refresh token; enable offline access and authorize again');
        data.tokens = { ...tokens, ...(tokens.refresh_token || redirectUrl ? {} : data.tokens?.refresh_token ? { refresh_token: data.tokens.refresh_token } : {}) };
        data.expiresAt = Date.now() + tokens.expires_in * 1000;
        data.issuer = metadata.issuer; data.resource = resource.href;
        if (redirectUrl) data.redirectUrl = redirectUrl;
        await save();
      },
      saveCodeVerifier: value => { verifier = value; },
      codeVerifier: () => { if (!verifier) throw new Error('Missing PKCE verifier'); return verifier; },
      redirectToAuthorization: async url => {
        if (!onRedirect) throw new Error('Run career-mcp.mjs login to authorize this client');
        await onRedirect(url, state);
      },
      invalidateCredentials: async scope => {
        if (['all', 'tokens'].includes(scope)) { delete data.tokens; delete data.expiresAt; }
        if (['all', 'client'].includes(scope)) delete data.client;
        if (['all', 'verifier'].includes(scope)) verifier = undefined;
        await save();
      },
      ...(machine ? { prepareTokenRequest: () => new URLSearchParams({ grant_type: 'client_credentials', scope: scopes }) } : {}),
    };
  }

  async function token(rejectedToken) {
    return store.locked(async (data, save) => {
      if (data.issuer && (data.issuer !== metadata.issuer || data.resource !== resource.href)) throw new Error('Stored OAuth issuer/resource mismatch');
      if (data.tokens && data.expiresAt > Date.now() + 30000 && data.tokens.access_token !== rejectedToken) return data.tokens.access_token;
      if (!machine && !data.tokens?.refresh_token) throw new Error('Run career-mcp.mjs login to authorize this client');
      const result = await auth(provider(data, save), { serverUrl: resource, scope: scopes, fetchFn: safeFetch });
      if (result !== 'AUTHORIZED') throw new Error('OAuth login required');
      return data.tokens.access_token;
    });
  }

  return {
    async login(onAuthorization = url => console.error(`Open this URL in your browser to authorize Career MCP:\n${url.href}`)) {
      if (machine) { await token(); return; }
      // Keep login separate from stdio startup; browser interaction cannot corrupt protocol output.
      const callbackPath = `/callback/${randomBytes(16).toString('hex')}`;
      let resolveCode, rejectCode, pending;
      const codePromise = new Promise((resolve, reject) => { resolveCode = resolve; rejectCode = reject; });
      codePromise.catch(() => {});
      const server = createServer((req, res) => {
        res.setHeader('Cache-Control', 'no-store');
        res.setHeader('Content-Type', 'text/plain; charset=utf-8');
        res.setHeader('Referrer-Policy', 'no-referrer');
        try {
          if (req.method !== 'GET' || !pending) throw new Error('No pending login');
          const url = new URL(req.url, pending.redirectUrl);
          if (req.headers.host !== new URL(pending.redirectUrl).host) throw new Error('Invalid callback host');
          const code = validateCallback(url, pending.redirectUrl, pending.state, metadata);
          pending = undefined;
          res.end('Authorization received. You can close this tab and return to the terminal.');
          resolveCode(code);
        } catch {
          res.statusCode = 400; res.end('Authorization was not accepted. Return to the terminal and try login again.');
          rejectCode(new Error('OAuth callback rejected'));
        }
      });
      await new Promise((resolve, reject) => { server.once('error', reject); server.listen(Number(env.CAREER_MCP_CALLBACK_PORT || 0), '127.0.0.1', resolve); });
      const redirectUrl = `http://127.0.0.1:${server.address().port}${clientId ? '/callback' : callbackPath}`;
      const timer = setTimeout(() => rejectCode(new Error('OAuth login timed out')), 180000);
      try {
        await store.locked(async (data, save) => {
          // Preserve a successful grant unless the new login succeeds. SDK changes are persisted as needed.
          const p = provider(data, save, redirectUrl, async (url, state) => { pending = { redirectUrl, state }; await onAuthorization(url); });
          // Explicit login always requests consent rather than silently refreshing an existing grant.
          p.tokens = () => undefined;
          // DCR registrations include exact callback URIs; register anew for this loopback listener.
          if (!clientId) { delete data.client; delete data.tokens; delete data.expiresAt; }
          if (await auth(p, { serverUrl: resource, scope: scopes, fetchFn: safeFetch }) !== 'REDIRECT') throw new Error('Expected OAuth redirect');
          const code = await codePromise;
          if (await auth(p, { serverUrl: resource, authorizationCode: code, scope: scopes, fetchFn: safeFetch }) !== 'AUTHORIZED') throw new Error('OAuth exchange failed');
        });
      } finally { clearTimeout(timer); server.closeAllConnections(); await new Promise(resolve => server.close(resolve)); }
    },
    async logout() {
      await store.locked(async (data, save) => {
        if (data.tokens?.refresh_token || data.tokens?.access_token) {
          if (!metadata.revocation_endpoint) throw new Error('Provider has no revocation endpoint; revoke in its dashboard before clearing credentials');
          const info = clientId ? { client_id: clientId, client_secret: clientSecret } : data.client;
          const body = new URLSearchParams({ token: data.tokens.refresh_token || data.tokens.access_token,
            token_type_hint: data.tokens.refresh_token ? 'refresh_token' : 'access_token', client_id: info.client_id });
          const headers = { 'Content-Type': 'application/x-www-form-urlencoded' };
          const method = selectClientAuthMethod(info, metadata.revocation_endpoint_auth_methods_supported || metadata.token_endpoint_auth_methods_supported || ['client_secret_basic']);
          if (method === 'client_secret_basic') headers.Authorization = `Basic ${Buffer.from(`${encodeURIComponent(info.client_id)}:${encodeURIComponent(info.client_secret)}`).toString('base64')}`;
          else if (method === 'client_secret_post') body.set('client_secret', info.client_secret);
          const response = await safeFetch(metadata.revocation_endpoint, { method: 'POST', headers, body });
          if (!response.ok) throw new Error('OAuth revocation failed');
        }
        for (const key of Object.keys(data)) delete data[key];
        await save();
      });
    },
    async fetch(input, init = {}) {
      const request = new Request(input, init);
      if (request.url !== resource.href) throw new Error('Refusing to send MCP token to another URL');
      const accessToken = await token();
      const send = async value => {
        const copy = request.clone();
        const headers = new Headers(copy.headers); headers.set('Authorization', `Bearer ${value}`);
        return fetchImpl(new Request(copy, { headers, redirect: 'error', signal: init.signal ?? AbortSignal.timeout(30000) }));
      };
      let response = await send(accessToken);
      if (response.status === 401) { await response.body?.cancel(); response = await send(await token(accessToken)); }
      return response;
    },
  };
}
