import fs from 'node:fs';
import type { FullConfig } from '@playwright/test';

// Logs in the same way tests/*.browser.mjs do (POST /api/auth/login, lift the
// session cookie from Set-Cookie) and saves it as Playwright storageState so
// every spec starts already authenticated.
export default async function globalSetup(config: FullConfig) {
  const baseURL = config.projects[0]?.use.baseURL as string;
  const response = await fetch(`${baseURL}/api/auth/login`, {
    method: 'POST',
    headers: { origin: baseURL, 'content-type': 'application/json' },
    body: JSON.stringify({
      username: process.env.AUTH_USERNAME || 'pwa-test',
      password: process.env.TEST_AUTH_PASSWORD || 'local-dev-password',
    }),
  });
  if (!response.ok) throw new Error(`Login failed with status ${response.status}. Set AUTH_USERNAME/TEST_AUTH_PASSWORD to match the target workspace.`);
  const session = response.headers.get('set-cookie')?.match(/^session=([^;]+)/)?.[1];
  if (!session) throw new Error('Login response had no session cookie.');
  fs.mkdirSync('playwright/.auth', { recursive: true });
  fs.writeFileSync('playwright/.auth/state.json', JSON.stringify({
    cookies: [{ name: 'session', value: session, domain: new URL(baseURL).hostname, path: '/', expires: -1, httpOnly: true, secure: false, sameSite: 'Lax' as const }],
    origins: [],
  }));
}
