import assert from 'node:assert/strict';
import test from 'node:test';
import bcrypt from 'bcryptjs';
import { configureDevAuth } from '../scripts/dev-auth.mjs';

test('local defaults allow login and authorize protected server requests', async () => {
  const env = {};
  configureDevAuth(env);
  const previous = { ...process.env };
  Object.assign(process.env, env);
  try {
    const { default: handler } = await import('../netlify/functions/auth.js?local');
    const { authenticated } = await import('../src/lib/server-auth.ts');
    const login = (password) => handler(new Request('http://localhost/.netlify/functions/auth', { method: 'POST', body: JSON.stringify({ action: 'login', username: 'admin', password }) }));
    assert.equal((await login('wrong-password')).status, 401);
    const response = await login('local-dev-password');
    assert.equal(response.status, 200);
    const { token } = await response.json();
    assert.equal(authenticated(new Request('http://localhost/api/workspace', { headers: { cookie: `auth_token=${token}` } })), true);
  } finally {
    for (const key of Object.keys(env)) {
      if (previous[key] === undefined) delete process.env[key];
      else process.env[key] = previous[key];
    }
  }
});

test('configured values are preserved and secrets vary across dev sessions', () => {
  const configured = { JWT_SECRET: 'custom-secret', AUTH_USERNAME: 'custom-user', AUTH_PASSWORD_HASH: bcrypt.hashSync('custom-password', 4) };
  const env = { ...configured };
  assert.deepEqual(configureDevAuth(env), []);
  assert.deepEqual(env, configured);
  const first = {};
  const second = {};
  configureDevAuth(first);
  configureDevAuth(second);
  assert.notEqual(first.JWT_SECRET, second.JWT_SECRET);
});

test('example password placeholder receives a usable local fallback', () => {
  const env = { AUTH_USERNAME: 'custom-user', AUTH_PASSWORD_HASH: '$2a$10$YourHashedPasswordHere' };
  configureDevAuth(env);
  assert.equal(env.AUTH_USERNAME, 'custom-user');
  assert.equal(bcrypt.compareSync('local-dev-password', env.AUTH_PASSWORD_HASH), true);
});

test('auth function fails closed without the dev launcher or configured credentials', async () => {
  const keys = ['JWT_SECRET', 'AUTH_USERNAME', 'AUTH_PASSWORD_HASH'];
  const previous = { ...process.env };
  for (const key of keys) delete process.env[key];
  try {
    const { default: handler } = await import('../netlify/functions/auth.js?unconfigured');
    const response = await handler(new Request('http://localhost/.netlify/functions/auth', { method: 'POST', body: JSON.stringify({ action: 'login', username: 'admin', password: 'local-dev-password' }) }));
    assert.equal(response.status, 503);
  } finally {
    for (const key of keys) {
      if (previous[key] !== undefined) process.env[key] = previous[key];
    }
  }
});
