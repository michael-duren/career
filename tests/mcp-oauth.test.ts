import test from 'node:test';
import assert from 'node:assert/strict';
import { generateKeyPair, exportJWK, createLocalJWKSet, createRemoteJWKSet, SignJWT, errors } from 'jose';
import { verifyMcpToken, readMcpAuthConfig, type OAuthConfig } from '../src/lib/mcp-auth.ts';
import { handleCareerMcp } from '../src/lib/mcp-http.ts';
import { handleMcpDiscovery } from '../src/lib/mcp-discovery.ts';
import { startOAuthServer } from './fixtures/oauth-server.mjs';

const config: OAuthConfig = { mode: 'oauth', resource: 'https://career.example/api/mcp', issuer: 'https://auth.example', audience: 'https://career.example/api/mcp',
  jwksUri: 'https://auth.example/jwks', ownerSubject: 'owner', allowedOrigins: ['https://chat.example'], serviceIdentities: [{ subject: 'job', clientId: 'scheduler' }], tokenProfile: 'at+jwt' };
const pair = await generateKeyPair('RS256');
const keys = createLocalJWKSet({ keys: [{ ...await exportJWK(pair.publicKey), alg: 'RS256', kid: 'test' }] });
async function mint(claims = {}, header = {}) {
  const now = Math.floor(Date.now() / 1000);
  return new SignJWT({ iss: config.issuer, aud: config.audience, sub: config.ownerSubject, iat: now, exp: now + 600, scope: 'career:read', ...claims })
    .setProtectedHeader({ alg: 'RS256', kid: 'test', typ: 'at+jwt', ...header }).sign(pair.privateKey);
}
const req = (token = '', headers = {}, method = 'POST') => new Request(config.resource, { method, headers: { authorization: `Bearer ${token}`, ...headers } });

test('configuration is explicitly enabled, validates endpoints, and has no token fallback', () => {
  assert.deepEqual(readMcpAuthConfig({ CAREER_MCP_TOKEN: 'x'.repeat(32) }), { mode: 'disabled' });
  const env = { CAREER_MCP_AUTH_MODE: 'oauth', CAREER_MCP_RESOURCE: config.resource, CAREER_MCP_OAUTH_ISSUER: config.issuer,
    CAREER_MCP_OAUTH_JWKS_URI: config.jwksUri, CAREER_MCP_OWNER_SUBJECT: config.ownerSubject, NODE_ENV: 'production' };
  assert.equal((readMcpAuthConfig(env) as OAuthConfig).audience, config.resource);
  for (const changed of [ { CAREER_MCP_OWNER_SUBJECT: '' }, { CAREER_MCP_RESOURCE: 'http://127.0.0.1/api/mcp' },
    { CAREER_MCP_OAUTH_ISSUER: 'https://user:password@auth.example' }, { CAREER_MCP_ALLOWED_ORIGINS: '*' },
    { CAREER_MCP_SERVICE_IDENTITIES: '[{"subject":"any"}]' }, { CAREER_MCP_AUTH_MODE: 'automatic' }, { CAREER_MCP_TOKEN_PROFILE: 'JWT' } ]) {
    assert.throws(() => readMcpAuthConfig({ ...env, ...changed }));
  }
});

test('only resource-bound access tokens with an allowed identity and scope can read', async () => {
  assert.deepEqual(await verifyMcpToken(await mint(), config, keys), { kind: 'owner', clientId: undefined });
  assert.equal((await verifyMcpToken(await mint({ sub: 'job', client_id: 'scheduler', grant_type: 'client_credentials' }), config, keys)).kind, 'service');
  for (const [claims, header, status] of [
    [{ iss: 'https://other.example' }, {}, 401], [{ aud: 'website-client' }, {}, 401], [{ sub: 'stranger' }, {}, 403],
    [{ sub: 'job', client_id: 'wrong' }, {}, 403], [{ scope: 'career:write' }, {}, 403], [{ scope: undefined }, {}, 403],
    [{ exp: 1 }, {}, 401], [{ exp: undefined }, {}, 401], [{ exp: Math.floor(Date.now() / 1000) + 3600 }, {}, 401], [{ nbf: 9999999999 }, {}, 401], [{ iat: 9999999999 }, {}, 401],
    [{}, { typ: 'JWT' }, 401], [{ token_use: 'id' }, {}, 401], [{ grant_type: 'client_credentials' }, {}, 403],
  ] as const) {
    let reads = 0;
    const response = await handleCareerMcp(req(await mint(claims, header)), async () => { reads++; throw new Error('Must not read'); }, {
      config, verify: (token, c) => verifyMcpToken(token, c, keys), audit: () => {},
    });
    assert.equal(response.status, status, JSON.stringify({ claims, header })); assert.equal(reads, 0);
    assert.match(response.headers.get('www-authenticate')!, /resource_metadata=/);
    assert.doesNotMatch(await response.text(), /stranger|website-client/);
  }
  assert.equal((await verifyMcpToken(await mint({ token_use: 'access' }, { typ: 'JWT' }), { ...config, tokenProfile: 'token_use' }, keys)).kind, 'owner');
  await assert.rejects(verifyMcpToken(await mint({}, { typ: 'JWT' }), { ...config, tokenProfile: 'token_use' }, keys), { status: 401 });
  await assert.rejects(verifyMcpToken('not.a.token', config, keys), { status: 401 });
  const valid = await mint();
  const parts = valid.split('.');
  parts[1] = Buffer.from(JSON.stringify({ ...JSON.parse(Buffer.from(parts[1], 'base64url').toString()), scope: 'career:read stolen' })).toString('base64url');
  await assert.rejects(verifyMcpToken(parts.join('.'), config, keys), { status: 401 });
  await assert.rejects(verifyMcpToken(await mint(), config, async () => { throw new errors.JWKSTimeout(); }), { status: 503 });
});

test('public discovery, unauthenticated preflight, origin handling and OAuth error challenges', async () => {
  const metadata = handleMcpDiscovery(new Request('https://career.example/.well-known/oauth-protected-resource'), config);
  assert.equal(metadata.status, 200);
  assert.deepEqual(await metadata.json(), { resource: config.resource, authorization_servers: [config.issuer], scopes_supported: ['career:read'], bearer_methods_supported: ['header'] });
  assert.equal(handleMcpDiscovery(new Request(config.resource), { mode: 'disabled' }).status, 503);
  const options = { config, audit: () => {} };
  const never = async () => { throw new Error('Must not read'); };
  const response = await handleCareerMcp(req('', { origin: 'https://chat.example', cookie: 'auth_token=website-jwt' }), never, options);
  assert.equal(response.status, 401); assert.equal(response.headers.get('access-control-allow-origin'), 'https://chat.example');
  assert.match(response.headers.get('access-control-expose-headers')!, /WWW-Authenticate/);
  assert.equal(response.headers.get('access-control-allow-credentials'), null);
  assert.equal(response.headers.get('vary'), 'Origin');
  assert.match(response.headers.get('www-authenticate')!, /https:\/\/career.example\/\.well-known\/oauth-protected-resource\/api\/mcp/);
  assert.equal((await handleCareerMcp(req('', { origin: 'null' }), never, options)).status, 403);
  const preflight = { origin: 'https://chat.example', 'access-control-request-method': 'POST', 'access-control-request-headers': 'Authorization, Content-Type, MCP-Protocol-Version' };
  assert.equal((await handleCareerMcp(req('', preflight, 'OPTIONS'), never, options)).status, 204);
  assert.equal((await handleCareerMcp(req('', { ...preflight, 'access-control-request-headers': 'cookie' }, 'OPTIONS'), never, options)).status, 403);
  const audit: unknown[] = [];
  const failed = await handleCareerMcp(req('secret'), never, { config, verify: async () => { throw new Error('private-provider-error'); }, audit: e => { audit.push(e); } });
  assert.equal(failed.status, 503); assert.doesNotMatch(JSON.stringify(audit) + await failed.text(), /secret|private-provider-error/);
});

test('remote JWKS is cached, rotated, and never fetched from token headers', async () => {
  const fixture = await startOAuthServer();
  try {
    const c = { ...config, issuer: fixture.issuer, resource: fixture.resource, audience: fixture.resource, jwksUri: `${fixture.issuer}/jwks` };
    const remote = createRemoteJWKSet(new URL(c.jwksUri), { cooldownDuration: 0, cacheMaxAge: 808000 });
    await verifyMcpToken(await fixture.mint({}, { header: { jku: 'https://untrusted.example/keys' } }), c, remote);
    await verifyMcpToken(await fixture.mint(), c, remote);
    assert.equal(fixture.counts.jwks, 1);
    const old = await fixture.mint();
    await fixture.rotate(); await verifyMcpToken(await fixture.mint(), c, remote);
    assert.equal(fixture.counts.jwks, 2);
    await assert.rejects(verifyMcpToken(old, c, remote), { status: 401 });
  } finally { await fixture.close(); }
});
