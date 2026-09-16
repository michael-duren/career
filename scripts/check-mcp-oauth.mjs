#!/usr/bin/env node
// Read-only deployment qualification; never prints tokens, subjects, or provider response bodies.
import { readMcpAuthConfig, verifyMcpToken } from '../src/lib/mcp-auth.ts';
import { discoverCareerOAuth } from './lib/mcp-oauth.mjs';

try {
  const config = readMcpAuthConfig();
  if (config.mode !== 'oauth') throw new Error('OAuth mode is required');
  const { state, safeFetch } = await discoverCareerOAuth(new URL(config.resource), config.issuer);
  const metadata = state.authorizationServerMetadata;
  const checks = {
    discovery: true,
    jwksMatchesConfiguration: metadata.jwks_uri === config.jwksUri,
    pkceS256: metadata.code_challenge_methods_supported?.includes('S256') === true,
    dynamicRegistration: Boolean(metadata.registration_endpoint),
    clientMetadataDocuments: metadata.client_id_metadata_document_supported === true,
    authorizationCode: metadata.grant_types_supported?.includes('authorization_code') === true,
    refreshTokens: metadata.grant_types_supported?.includes('refresh_token') === true,
    scope: metadata.scopes_supported?.includes('career:read') === true,
    revocation: Boolean(metadata.revocation_endpoint),
    machineGrant: metadata.grant_types_supported?.includes('client_credentials') === true,
  };
  for (const [name, passes] of Object.entries(checks)) console.log(`${name}: ${passes ? 'advertised' : 'needs provider configuration or verification'}`);
  const jwks = await safeFetch(config.jwksUri);
  if (!jwks.ok || !Array.isArray((await jwks.json()).keys)) throw new Error('Signing keys unavailable');
  if (process.env.CAREER_MCP_CHECK_ACCESS_TOKEN) {
    await verifyMcpToken(process.env.CAREER_MCP_CHECK_ACCESS_TOKEN, config);
    console.log('accessToken: verified signature, token type, issuer, audience, owner/service identity, expiry and scope');
  } else console.log('accessToken: not checked; provide CAREER_MCP_CHECK_ACCESS_TOKEN through a secret environment to verify a real grant');
  console.log('Metadata checks do not prove client compatibility. Complete login, refresh and revocation in each target client.');
  if (Object.entries(checks).some(([name, passes]) => name !== 'machineGrant' && !passes)) process.exitCode = 1;
} catch {
  console.error('OAuth qualification failed. Check server configuration, public discovery, trusted issuer/JWKS, provider capabilities and the supplied access token.');
  process.exitCode = 1;
}
