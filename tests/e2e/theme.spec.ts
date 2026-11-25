import { test, expect, type Page } from '@playwright/test';

// Theme tokens: data-theme on <html> picks the palette, the saved choice is applied
// before first paint, and fonts come from the app itself.

const PALETTE = {
  dark: { paper: 'rgb(12, 12, 11)', ink: 'rgb(236, 234, 227)', label: 'rgba(236, 234, 227, 0.6)', accent: 'rgb(224, 83, 61)' },
  vellum: { paper: 'rgb(243, 241, 234)', ink: 'rgb(26, 26, 23)', label: 'rgba(26, 26, 23, 0.64)', accent: 'rgb(194, 54, 30)' },
  cobalt: { paper: 'rgb(247, 247, 244)', ink: 'rgb(0, 71, 171)', label: 'rgba(0, 71, 171, 0.78)', accent: 'rgb(214, 64, 42)' },
  concrete: { paper: 'rgb(44, 45, 47)', ink: 'rgb(239, 237, 230)', label: 'rgba(239, 237, 230, 0.6)', accent: 'rgb(242, 194, 48)' },
};

// Computed colors on <body> plus a probe painted with the label and accent tokens.
const paint = (page: Page) => page.evaluate(() => {
  let probe = document.getElementById('theme-probe');
  if (!probe) {
    probe = document.createElement('span');
    probe.id = 'theme-probe';
    probe.style.cssText = 'color: var(--ink-60); border: 1px solid var(--accent)';
    document.body.append(probe);
  }
  const body = getComputedStyle(document.body), style = getComputedStyle(probe);
  return { theme: document.documentElement.dataset.theme, paper: body.backgroundColor, ink: body.color, label: style.color, accent: style.borderTopColor };
});

test('defaults to dark and re-themes every token when data-theme changes', async ({ page }) => {
  await page.goto('/books');
  expect(await paint(page)).toEqual({ theme: 'dark', ...PALETTE.dark });

  for (const [theme, colors] of Object.entries(PALETTE)) {
    await page.evaluate(name => { document.documentElement.dataset.theme = name; }, theme);
    expect(await paint(page)).toEqual({ theme, ...colors });
  }
  // An unknown or missing attribute falls back to the dark palette in CSS too.
  await page.evaluate(() => { document.documentElement.dataset.theme = 'neon'; });
  expect(await paint(page)).toMatchObject(PALETTE.dark);
  await page.evaluate(() => document.documentElement.removeAttribute('data-theme'));
  expect(await paint(page)).toMatchObject(PALETTE.dark);
});

for (const path of ['/books', '/login']) test(`the theme script runs inline before any stylesheet on ${path}`, async ({ page }) => {
  const html = await (await page.request.get(path)).text();
  const head = html.slice(0, html.indexOf('</head>'));
  const script = head.indexOf('document.documentElement.dataset.theme');
  const stylesheet = head.search(/<link[^>]+rel="?stylesheet|<style/);
  expect(head).toMatch(/<html[^>]*data-theme="dark"/);
  expect(script).toBeGreaterThan(-1);
  expect(stylesheet).toBeGreaterThan(script);
  const tag = head.slice(head.lastIndexOf('<script', script), script);
  expect(tag).not.toMatch(/\b(defer|async|src=|type="module")/);
});

test('a saved theme is applied before <body> exists', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('career-theme', 'cobalt'));
  // Record whether <body> had been parsed when data-theme first became cobalt. Set from <head>,
  // nothing in the body has been parsed yet, so it cannot have painted in the default theme.
  await page.addInitScript(() => {
    const observer = new MutationObserver(() => {
      if (document.documentElement?.dataset.theme !== 'cobalt') return;
      (window as unknown as { bodyAtSwitch?: boolean }).bodyAtSwitch = !!document.body;
      observer.disconnect();
    });
    observer.observe(document, { subtree: true, childList: true, attributes: true, attributeFilter: ['data-theme'] });
  });
  await page.goto('/books');
  expect(await page.evaluate(() => (window as unknown as { bodyAtSwitch?: boolean }).bodyAtSwitch)).toBe(false);
  expect((await paint(page)).paper).toBe(PALETTE.cobalt.paper);
});

test('a nested data-theme re-resolves the tokens', async ({ page }) => {
  await page.goto('/books');
  const nested = await page.evaluate(() => {
    const box = document.createElement('div');
    box.dataset.theme = 'cobalt';
    box.style.cssText = 'background: var(--paper); color: var(--ink-60)';
    document.body.append(box);
    const style = getComputedStyle(box);
    return { paper: style.backgroundColor, label: style.color };
  });
  expect(nested).toEqual({ paper: PALETTE.cobalt.paper, label: PALETTE.cobalt.label });
});

test('a saved theme is honored', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('career-theme', 'vellum'));
  await page.goto('/books');
  expect(await paint(page)).toMatchObject({ theme: 'vellum', paper: PALETTE.vellum.paper });
});

test('an unknown saved theme falls back to dark', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('career-theme', 'neon'));
  await page.goto('/books');
  expect(await paint(page)).toMatchObject({ theme: 'dark', paper: PALETTE.dark.paper });
});

test('blocked storage keeps the dark default', async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(window, 'localStorage', { get() { throw new DOMException('blocked', 'SecurityError'); } });
  });
  await page.goto('/books');
  expect(await paint(page)).toMatchObject({ theme: 'dark', paper: PALETTE.dark.paper });
});

test('fonts are self-hosted', async ({ page, baseURL }) => {
  const fontHosts = new Set<string>();
  page.on('request', request => { if (request.resourceType() === 'font') fontHosts.add(new URL(request.url()).origin); });
  await page.goto('/books');
  const faces = ['300 16px "Inter Tight Variable"', '500 10px "IBM Plex Mono"', '16px "Architects Daughter"'];
  // load() resolves to the matching faces, so an empty list means the @font-face is missing.
  const loaded = await page.evaluate(fonts => Promise.all(fonts.map(async font => (await document.fonts.load(font)).filter(face => face.status === 'loaded').length)), faces);
  for (const count of loaded) expect(count).toBeGreaterThan(0);
  expect(await page.evaluate(() => getComputedStyle(document.body).fontFamily)).toMatch(/^"Inter Tight Variable"/);
  expect([...fontHosts]).toEqual([new URL(baseURL!).origin]);
});
