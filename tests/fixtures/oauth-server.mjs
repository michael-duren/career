// Test-only authorization server. /authorize deliberately simulates an already authenticated owner.
// Never deploy this fixture or use it with real career data.
import { createServer } from 'node:http';
import { createHash, randomUUID } from 'node:crypto';
import { generateKeyPair, exportJWK, SignJWT } from 'jose';

export async function startOAuthServer({ resource, tokenLifetime = 120 } = {}) {
  let pair = await generateKeyPair('RS256');
  let kid = randomUUID();
  let publicKeys = [{ ...await exportJWK(pair.publicKey), kid, alg: 'RS256', use: 'sig' }];
  const clients = new Map();
  const codes = new Map();
  const refresh = new Map();
  const families = new Map();
  const access = new Map();
  const counts = { registrations: 0, exchanges: 0, refreshes: 0, revocations: 0, jwks: 0, machine: 0 };
  let issuer;
  const mint = async (claims = {}, options = {}) => {
    const now = Math.floor(Date.now() / 1000);
    return new SignJWT({ scope: 'career:read', client_id: 'test-client', ...claims })
      .setProtectedHeader({ alg: 'RS256', kid, typ: 'at+jwt', ...options.header })
      .setIssuer(options.issuer ?? issuer).setAudience(options.audience ?? resource)
      .setSubject(options.subject ?? 'owner').setJti(randomUUID()).setIssuedAt(options.iat ?? now)
      .setExpirationTime(options.exp ?? now + tokenLifetime).sign(pair.privateKey);
  };
  const issue = async (clientId, family = randomUUID(), machine = false) => {
    families.set(family, true);
    const accessToken = await mint({ client_id: clientId, ...(machine ? { grant_type: 'client_credentials' } : {}) }, { subject: machine ? 'service' : 'owner' });
    const refreshToken = randomUUID();
    access.set(accessToken, family);
    if (!machine) refresh.set(refreshToken, { clientId, family, used: false });
    return { token_type: 'Bearer', access_token: accessToken, expires_in: tokenLifetime, scope: 'career:read', ...(!machine ? { refresh_token: refreshToken } : {}) };
  };
  const server = createServer(async (req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.setHeader('Cache-Control', 'no-store');
    const json = (status, body) => { res.statusCode = status; res.end(JSON.stringify(body)); };
    try {
      const url = new URL(req.url, issuer);
      let body = '';
      for await (const chunk of req) { body += chunk; if (body.length > 65536) return json(413, {}); }
      const params = new URLSearchParams(body);
      if (url.pathname === '/.well-known/oauth-authorization-server') return json(200, {
        issuer, authorization_endpoint: `${issuer}/authorize`, token_endpoint: `${issuer}/token`, jwks_uri: `${issuer}/jwks`,
        registration_endpoint: `${issuer}/register`, revocation_endpoint: `${issuer}/revoke`, response_types_supported: ['code'],
        grant_types_supported: ['authorization_code', 'refresh_token', 'client_credentials'], token_endpoint_auth_methods_supported: ['none', 'client_secret_basic'],
        code_challenge_methods_supported: ['S256'], scopes_supported: ['career:read', 'offline_access'], authorization_response_iss_parameter_supported: true,
      });
      if (url.pathname.startsWith('/.well-known/oauth-protected-resource')) return json(200, { resource, authorization_servers: [issuer], scopes_supported: ['career:read'] });
      if (url.pathname === '/jwks') { counts.jwks++; return json(200, { keys: publicKeys }); }
      if (url.pathname === '/register' && req.method === 'POST') {
        const metadata = JSON.parse(body);
        if (!metadata.redirect_uris?.length || metadata.redirect_uris.some(uri => { const u = new URL(uri); return u.hostname !== '127.0.0.1' || u.protocol !== 'http:'; })) return json(400, { error: 'invalid_redirect_uri' });
        const info = { ...metadata, client_id: randomUUID() };
        clients.set(info.client_id, info); counts.registrations++; return json(201, info);
      }
      if (url.pathname === '/authorize') {
        const p = url.searchParams;
        const client = clients.get(p.get('client_id'));
        if (!client || !client.redirect_uris.includes(p.get('redirect_uri'))) return json(400, { error: 'invalid_redirect_uri' });
        if (p.get('resource') !== resource || p.get('code_challenge_method') !== 'S256' || !p.get('code_challenge') || p.get('response_type') !== 'code') return json(400, { error: 'invalid_request' });
        const callback = new URL(p.get('redirect_uri'));
        callback.searchParams.set('state', p.get('state') || ''); callback.searchParams.set('iss', issuer);
        if (p.get('test_deny') === 'true') callback.searchParams.set('error', 'access_denied');
        else {
          const code = randomUUID();
          codes.set(code, { clientId: p.get('client_id'), redirect: p.get('redirect_uri'), resource, challenge: p.get('code_challenge'), expires: Date.now() + 60000 });
          callback.searchParams.set('code', code);
        }
        res.writeHead(302, { Location: callback.href }); return res.end();
      }
      if (['/token', '/revoke'].includes(url.pathname) && req.method === 'POST') {
        let id = params.get('client_id'); let secret;
        if (req.headers.authorization?.startsWith('Basic ')) {
          const parts = Buffer.from(req.headers.authorization.slice(6), 'base64').toString().split(':');
          id = decodeURIComponent(parts[0]); secret = decodeURIComponent(parts[1]);
        }
        const client = clients.get(id);
        if (!client || client.client_secret && secret !== client.client_secret) return json(401, { error: 'invalid_client' });
        if (url.pathname === '/revoke') {
          const value = params.get('token'); const grant = refresh.get(value);
          if (grant?.clientId === id) families.set(grant.family, false);
          counts.revocations++; return json(200, {});
        }
        if (params.get('resource') !== resource) return json(400, { error: 'invalid_target' });
        if (params.get('grant_type') === 'client_credentials') {
          if (!client.machine || !secret) return json(400, { error: 'unauthorized_client' });
          counts.machine++; return json(200, await issue(id, undefined, true));
        }
        if (params.get('grant_type') === 'authorization_code') {
          const code = codes.get(params.get('code')); codes.delete(params.get('code'));
          const challenge = createHash('sha256').update(params.get('code_verifier') || '').digest('base64url');
          if (!code || code.expires < Date.now() || code.clientId !== id || code.redirect !== params.get('redirect_uri') || code.challenge !== challenge) return json(400, { error: 'invalid_grant' });
          counts.exchanges++; return json(200, await issue(id));
        }
        if (params.get('grant_type') === 'refresh_token') {
          const grant = refresh.get(params.get('refresh_token'));
          if (grant?.used) families.set(grant.family, false);
          if (!grant || grant.used || !families.get(grant.family) || grant.clientId !== id) return json(400, { error: 'invalid_grant' });
          grant.used = true; counts.refreshes++; return json(200, await issue(id, grant.family));
        }
        return json(400, { error: 'unsupported_grant_type' });
      }
      if (url.pathname === '/api/mcp') {
        const token = req.headers.authorization?.replace(/^Bearer /, '');
        return json(access.has(token) ? 200 : 401, { ok: access.has(token) });
      }
      return json(404, {});
    } catch { return json(500, { error: 'test_fixture_error' }); }
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  issuer = `http://127.0.0.1:${server.address().port}`;
  resource ??= `${issuer}/api/mcp`;
  return {
    issuer, resource, counts, mint, clients, issue,
    async rotate() { pair = await generateKeyPair('RS256'); kid = randomUUID(); publicKeys = [{ ...await exportJWK(pair.publicKey), kid, alg: 'RS256', use: 'sig' }]; },
    async authorize(url) { const result = await fetch(url, { redirect: 'manual' }); if (result.status !== 302) throw new Error('Fixture authorization failed'); return new URL(result.headers.get('location')); },
    async close() { server.closeAllConnections(); await new Promise(resolve => server.close(resolve)); },
  };
}
