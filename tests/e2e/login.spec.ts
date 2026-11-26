import { test, expect } from '@playwright/test';

// Login page: a failed sign-in shows the accent error, Cancel clears it, and a good one
// follows ?redirect=. Starts logged out; the failure is followed by a success, which
// resets the server's per-IP attempt counter.

test.use({ storageState: { cookies: [], origins: [] } });

test('a failed login shows the error, Cancel clears it, and a good login redirects', async ({ page }) => {
  await page.goto('/login?redirect=/books');
  const username = page.getByLabel('Username');
  const password = page.getByLabel('Password');
  const error = page.getByRole('alert');

  await expect(username).toHaveCSS('height', '44px');
  await expect(error).toBeHidden();

  await username.fill(process.env.AUTH_USERNAME || 'pwa-test');
  await password.fill('definitely-not-the-password');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(error).toHaveText('Invalid credentials');
  const accent = await page.evaluate(() => {
    const probe = document.createElement('span');
    probe.style.color = 'var(--accent)';
    document.body.append(probe);
    const color = getComputedStyle(probe).color;
    probe.remove();
    return color;
  });
  await expect(error).toHaveCSS('color', accent);

  await page.getByRole('button', { name: 'Cancel' }).click();
  await expect(error).toBeHidden();
  await expect(username).toHaveValue('');

  await username.fill(process.env.AUTH_USERNAME || 'pwa-test');
  await password.fill(process.env.TEST_AUTH_PASSWORD || 'local-dev-password');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page).toHaveURL(/\/books$/);
});
