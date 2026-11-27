import fs from 'node:fs';
import type { FullConfig } from '@playwright/test';

// Logs in (as tests/e2e/global-setup.ts does), marks first-run setup as done (a freshly
// seeded database has not, and every page would redirect to /setup), and adds what the
// workspace import cannot hold: a Leetgrinder todo set. Its problems are in the built-in
// catalog, so the metadata worker never fetches them from LeetCode.
export default async function globalSetup(config: FullConfig) {
  const baseURL = config.projects[0]?.use.baseURL as string;
  const login = await fetch(`${baseURL}/api/auth/login`, {
    method: 'POST',
    headers: { origin: baseURL, 'content-type': 'application/json' },
    body: JSON.stringify({ username: process.env.AUTH_USERNAME || 'visual', password: process.env.TEST_AUTH_PASSWORD || 'playwright-ci-password' }),
  });
  if (!login.ok) throw new Error(`Login failed with status ${login.status}. Run the suite with scripts/visual-tests.sh.`);
  const session = login.headers.get('set-cookie')?.match(/^session=([^;]+)/)?.[1];
  if (!session) throw new Error('Login response had no session cookie.');

  const settings = await fetch(`${baseURL}/api/settings`, {
    method: 'PATCH',
    headers: { origin: baseURL, 'content-type': 'application/json', cookie: `session=${session}` },
    body: JSON.stringify({ setupCompletedAt: '2026-11-01T12:00:00Z' }),
  });
  if (!settings.ok) throw new Error(`PATCH /api/settings failed with status ${settings.status}: ${await settings.text()}`);

  const todos = await fetch(`${baseURL}/leetgrinder/todos/sets`, {
    method: 'POST',
    redirect: 'manual',
    headers: { origin: baseURL, 'content-type': 'application/x-www-form-urlencoded', cookie: `session=${session}` },
    body: new URLSearchParams({ title: 'Graphs', problems: ['number-of-islands', 'clone-graph', 'pacific-atlantic-water-flow', 'course-schedule', 'rotting-oranges', 'two-sum'].join('\n') }),
  });
  if (todos.status !== 303) throw new Error(`Creating the todo set failed with status ${todos.status}: ${await todos.text()}`);

  fs.mkdirSync('playwright/.auth', { recursive: true });
  fs.writeFileSync('playwright/.auth/visual-state.json', JSON.stringify({
    cookies: [{ name: 'session', value: session, domain: new URL(baseURL).hostname, path: '/', expires: -1, httpOnly: true, secure: false, sameSite: 'Lax' as const }],
    origins: [],
  }));
}
