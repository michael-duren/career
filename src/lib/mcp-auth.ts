import { createHash, timingSafeEqual } from 'node:crypto';
import { createRemoteJWKSet, jwtVerify, errors, type JWTVerifyGetKey } from 'jose';

export const MCP_SCOPE = 'career:read';
export const MCP_METADATA_PATH = '/.well-known/oauth-protected-resource/api/mcp';
export const MCP_METADATA_PATHS = [MCP_METADATA_PATH, '/.well-known/oauth-protected-resource'];
export type OAuthConfig = {
  mode: 'oauth'; resource: string; issuer: string; audience: string; jwksUri: string;
  ownerSubject: string; serviceIdentities: { subject: string; clientId: string }[];
  allowedOrigins: string[]; tokenProfile: 'at+jwt' | 'token_use';
};
export type McpAuthConfig = OAuthConfig | { mode: 'legacy'; token: string; allowedOrigins: string[] } | { mode: 'disabled' };
export class McpAuthError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string) { super(code); this.status = status; this.code = code; }
}

function endpoint(value: string | undefined, local: boolean): URL {
  const url = new URL(value ?? '');
  if (url.username || url.password || url.search || url.hash ||
      !(url.protocol === 'https:' || (local && url.protocol === 'http:' && ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname)))) {
    throw new Error('Invalid OAuth endpoint');
  }
  return url;
}

// Read at request time: Netlify Functions configuration may differ from build-time values.
export function readMcpAuthConfig(env: NodeJS.ProcessEnv = process.env): McpAuthConfig {
  const mode = env.CAREER_MCP_AUTH_MODE ?? 'disabled';
  if (mode === 'disabled') return { mode };
  try {
    const local = env.NODE_ENV !== 'production';
    const allowedOrigins = (env.CAREER_MCP_ALLOWED_ORIGINS ?? '').split(',').map(s => s.trim()).filter(Boolean);
    for (const origin of allowedOrigins) if (endpoint(origin, local).origin !== origin) throw new Error('Invalid origin');
    if (mode === 'legacy') {
      const token = env.CAREER_MCP_TOKEN;
      if (!token || token.length < 32) throw new Error('Missing legacy token');
      return { mode, token, allowedOrigins };
    }
    if (mode !== 'oauth') throw new Error('Invalid auth mode');
    const resource = endpoint(env.CAREER_MCP_RESOURCE, local).href;
    if (new URL(resource).pathname !== '/api/mcp') throw new Error('Invalid resource path');
    const issuer = env.CAREER_MCP_OAUTH_ISSUER!;
    endpoint(issuer, local);
    const jwksUri = endpoint(env.CAREER_MCP_OAUTH_JWKS_URI, local).href;
    const audience = env.CAREER_MCP_OAUTH_AUDIENCE || resource;
    const ownerSubject = env.CAREER_MCP_OWNER_SUBJECT;
    if (!ownerSubject?.trim()) throw new Error('Missing owner');
    const serviceIdentities = JSON.parse(env.CAREER_MCP_SERVICE_IDENTITIES || '[]');
    if (!Array.isArray(serviceIdentities) || serviceIdentities.length > 100 || serviceIdentities.some(identity =>
      !identity || typeof identity.subject !== 'string' || !identity.subject || identity.subject === ownerSubject || typeof identity.clientId !== 'string' || !identity.clientId)) {
      throw new Error('Invalid service identities');
    }
    const tokenProfile = env.CAREER_MCP_TOKEN_PROFILE || 'at+jwt';
    if (tokenProfile !== 'at+jwt' && tokenProfile !== 'token_use') throw new Error('Invalid token profile');
    return { mode, resource, issuer, jwksUri, audience, ownerSubject, serviceIdentities, allowedOrigins, tokenProfile };
  } catch { throw new McpAuthError(503, 'oauth_not_configured'); }
}

const keySets = new Map<string, ReturnType<typeof createRemoteJWKSet>>();
function remoteKeys(uri: string) {
  let keys = keySets.get(uri);
  if (!keys) {
    // jose refuses redirects, bounds retrieval, and caches/refreshes keys, including unknown kids.
    keys = createRemoteJWKSet(new URL(uri), { timeoutDuration: 5000, cooldownDuration: 30000, cacheMaxAge: 300000 });
    if (keySets.size >= 8) keySets.delete(keySets.keys().next().value!);
    keySets.set(uri, keys);
  }
  return keys;
}

export type McpPrincipal = { clientId?: string; kind: 'owner' | 'service' | 'legacy' };
export type McpTokenVerifier = (token: string, config: OAuthConfig) => Promise<McpPrincipal>;

export async function verifyMcpToken(token: string, config: OAuthConfig, keys: JWTVerifyGetKey = remoteKeys(config.jwksUri)): Promise<McpPrincipal> {
  let payload;
  try {
    const result = await jwtVerify(token, keys, {
      issuer: config.issuer, audience: config.audience, algorithms: ['RS256', 'ES256', 'PS256', 'EdDSA'],
      requiredClaims: ['exp', 'iat', 'sub', 'aud', 'iss'], clockTolerance: 5,
      ...(config.tokenProfile === 'at+jwt' ? { typ: 'at+jwt' } : {}),
    });
    payload = result.payload;
    if (config.tokenProfile === 'token_use' && payload.token_use !== 'access') throw new Error('Not an access token');
    if (payload.token_use !== undefined && payload.token_use !== 'access') throw new Error('Not an access token');
    if (payload.iat! > Date.now() / 1000 + 5 || payload.exp! <= payload.iat! || payload.exp! - payload.iat! > 600) throw new Error('Invalid lifetime');
  } catch (error) {
    if (error instanceof errors.JWKSTimeout || !(error instanceof errors.JOSEError) && error instanceof TypeError ||
        error instanceof errors.JOSEError && ['ERR_JWKS_INVALID', 'ERR_JOSE_GENERIC'].includes(error.code)) {
      throw new McpAuthError(503, 'oauth_unavailable');
    }
    throw new McpAuthError(401, 'invalid_token');
  }
  const clientId = typeof payload.client_id === 'string' ? payload.client_id : undefined;
  const owner = payload.sub === config.ownerSubject && payload.gty !== 'client-credentials' && payload.grant_type !== 'client_credentials';
  const service = config.serviceIdentities.some(identity => identity.subject === payload.sub && identity.clientId === clientId);
  if (!owner && !service) throw new McpAuthError(403, 'access_denied');
  if (typeof payload.scope !== 'string' || !payload.scope.split(/\s+/).includes(MCP_SCOPE)) throw new McpAuthError(403, 'insufficient_scope');
  return { kind: owner ? 'owner' : 'service', clientId };
}

export async function authenticateMcp(request: Request, config: McpAuthConfig, verify: McpTokenVerifier = verifyMcpToken): Promise<McpPrincipal> {
  if (config.mode === 'disabled') throw new McpAuthError(503, 'mcp_disabled');
  const token = /^Bearer ([^\s]+)$/i.exec(request.headers.get('authorization') ?? '')?.[1];
  if (!token || token.length > 16384) throw new McpAuthError(401, token ? 'invalid_token' : 'authorization_required');
  if (config.mode === 'oauth') return verify(token, config);
  const digest = (value: string) => createHash('sha256').update(value).digest();
  if (!timingSafeEqual(digest(token), digest(config.token))) throw new McpAuthError(401, 'invalid_token');
  return { kind: 'legacy' };
}

export function mcpChallenge(config: McpAuthConfig, code: string): string {
  const metadata = config.mode === 'oauth' ? `resource_metadata="${new URL(MCP_METADATA_PATH, config.resource).href}", scope="${MCP_SCOPE}"` : 'realm="career-mcp"';
  return `Bearer ${metadata}${['invalid_token', 'insufficient_scope'].includes(code) ? `, error="${code}"` : ''}`;
}
