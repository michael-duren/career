import { test, expect, type Page } from '@playwright/test';

// Login page, logged out. The failed attempt is followed by a good one, which resets the
// server's per-IP attempt counter (10 per 15 minutes).

test.use({ storageState: { cookies: [], origins: [] } });

const username = process.env.AUTH_USERNAME || 'pwa-test';
const password = process.env.TEST_AUTH_PASSWORD || 'local-dev-password';

const tokenColor = (page: Page, token: string) => page.evaluate(name => {
  const probe = document.createElement('span');
  probe.style.color = `var(${name})`;
  document.body.append(probe);
  const color = getComputedStyle(probe).color;
  probe.remove();
  return color;
}, token);

test('a failed login shows the server error, Cancel clears it, and a good login redirects', async ({ page }) => {
  await page.goto('/login?redirect=/books');
  const user = page.getByLabel('Username');
  const pass = page.getByLabel('Password');
  const error = page.getByRole('alert');

  await expect(user).toHaveCSS('height', '44px');
  await expect(error).toHaveText('');

  // The server's 401 body, not the client fallback (which would be the same text with no body).
  const response = page.waitForResponse('**/api/auth/login');
  await user.fill('no-such-user');
  await pass.fill('definitely-not-the-password');
  await page.getByRole('button', { name: 'Sign in' }).click();
  expect((await response).status()).toBe(401);
  await expect(error).toHaveText('Invalid credentials');
  await expect(error).toBeInViewport();
  await expect(error).toHaveCSS('color', await tokenColor(page, '--accent'));

  await page.getByRole('button', { name: 'Cancel' }).click();
  await expect(error).toHaveText('');
  await expect(user).toHaveValue('');
  await expect(pass).toHaveValue('');

  await user.fill(username);
  await pass.fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page).toHaveURL(/\/books$/);
});

test('the phone layout keeps the 40px title and 16px fields', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/login');
  const title = page.getByRole('heading', { level: 1, name: 'Sign in' });
  await expect(title).toHaveCSS('font-size', '40px');
  await expect(title).toHaveCSS('line-height', '36px');
  await expect(page.getByLabel('Username')).toHaveCSS('font-size', '16px');
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
