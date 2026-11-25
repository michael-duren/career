import { test, expect } from '@playwright/test';

// Theme tokens: data-theme on <html> picks the palette, the saved choice is applied
// before first paint, and fonts come from the app itself.

const paperOf = (page: import('@playwright/test').Page) =>
  page.evaluate(() => ({ theme: document.documentElement.dataset.theme, body: getComputedStyle(document.body).backgroundColor, color: getComputedStyle(document.body).color }));

test('defaults to dark and re-themes when data-theme changes', async ({ page }) => {
  await page.goto('/books');
  expect(await paperOf(page)).toEqual({ theme: 'dark', body: 'rgb(12, 12, 11)', color: 'rgb(236, 234, 227)' });

  const expected = { vellum: 'rgb(243, 241, 234)', cobalt: 'rgb(247, 247, 244)', concrete: 'rgb(44, 45, 47)', dark: 'rgb(12, 12, 11)' };
  for (const [theme, body] of Object.entries(expected)) {
    await page.evaluate(name => { document.documentElement.dataset.theme = name; }, theme);
    expect((await paperOf(page)).body).toBe(body);
  }
});

test('a saved theme is applied before the first paint', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('career-theme', 'cobalt'));
  // Record the theme at the moment <body> is parsed, i.e. before anything in it can paint.
  await page.addInitScript(() => {
    const observer = new MutationObserver(() => {
      if (!document.body) return;
      (window as unknown as { firstTheme?: string }).firstTheme = document.documentElement.dataset.theme;
      observer.disconnect();
    });
    observer.observe(document, { childList: true, subtree: true });
  });
  await page.goto('/books');
  expect(await page.evaluate(() => (window as unknown as { firstTheme?: string }).firstTheme)).toBe('cobalt');
  expect((await paperOf(page)).body).toBe('rgb(247, 247, 244)');
});

test('an unknown saved theme falls back to dark', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('career-theme', 'neon'));
  await page.goto('/books');
  expect((await paperOf(page)).theme).toBe('dark');
});

test('fonts are self-hosted', async ({ page }) => {
  const external: string[] = [];
  page.on('request', request => { if (/fonts\.(googleapis|gstatic)\.com/.test(request.url())) external.push(request.url()); });
  await page.goto('/books');
  await page.evaluate(() => document.fonts.ready);
  const loaded = await page.evaluate(async () => {
    await Promise.all(['300 16px "Inter Tight Variable"', '500 10px "IBM Plex Mono"', '16px "Architects Daughter"'].map(font => document.fonts.load(font)));
    return ['300 16px "Inter Tight Variable"', '500 10px "IBM Plex Mono"', '16px "Architects Daughter"'].map(font => document.fonts.check(font));
  });
  expect(loaded).toEqual([true, true, true]);
  expect(external).toEqual([]);
});
