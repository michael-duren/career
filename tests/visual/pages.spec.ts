import { test, expect, type Page } from '@playwright/test';
import { THEMES } from '../../src/lib/theme.ts';

// Every page at desktop and phone size in every theme, against tests/visual/workspace.json.
// Run with scripts/visual-tests.sh (see playwright.visual.config.ts and docs/developer.md).
//
// Determinism: the browser clock is frozen at NOW (the fixture's dates are relative to
// it), the time zone is the scheduler's, outside requests are blocked, company logos
// answer 404 (initials), and each page waits for its data, fonts and images before the
// shot. The server keeps its real clock (Leetgrinder's day, the scheduler's capacity and
// missed sessions). That is stable because the fixture's schedule week ends 2026-11-22,
// before the baselines were made, and it has no Leetgrinder attempts (the server dates
// those itself). Keep it that way: add nothing the server dates by today, or mask it
// with toHaveScreenshot's `mask`. Goal dates may be in the future; the browser renders
// them against NOW.
//
// "phone" is a 390px Desktop Chrome viewport (no touch emulation), like ui-check.

// Wednesday of the fixture's week (tests/visual/workspace.json).
const NOW = new Date('2026-11-18T10:30:00-06:00');

const VIEWPORTS = [
  { name: 'desktop', size: { width: 1280, height: 800 } },
  { name: 'phone', size: { width: 390, height: 844 } },
] as const;

interface Shot {
  name: string;
  path: string;
  signedOut?: boolean;
}

const PAGES: Shot[] = [
  { name: 'home', path: '/' },
  { name: 'timeline', path: '/timeline' },
  { name: 'schedule', path: '/weekly-scheduler' },
  { name: 'leetgrinder', path: '/leetgrinder' },
  { name: 'companies', path: '/companies' },
  { name: 'people', path: '/connections' },
  { name: 'books', path: '/books' },
  { name: 'journal', path: '/journal' },
  { name: 'notes', path: '/notes' },
  { name: 'audio-thoughts', path: '/audio-thoughts' },
  { name: 'agents', path: '/agents' },
  { name: 'settings-modules', path: '/settings/modules' },
  { name: 'login', path: '/login', signedOut: true },
];

// Grows the viewport to the page's full height instead of taking a full-page shot, so
// fixed parts (the phone tab bar) sit at the bottom of the page as when scrolled there,
// not over the content where the first screen ended.
async function growToContent(page: Page, size: { width: number; height: number }) {
  let height = size.height;
  for (let i = 0; i < 3; i++) {
    const content = await page.evaluate(() => document.documentElement.scrollHeight);
    if (content <= height) return;
    expect(content, 'page taller than 6000px: shots would crop it').toBeLessThanOrEqual(6000);
    height = content;
    await page.setViewportSize({ width: size.width, height });
  }
  expect(await page.evaluate(() => document.documentElement.scrollHeight), 'page height did not settle').toBeLessThanOrEqual(height);
}

async function settle(page: Page) {
  await page.waitForLoadState('networkidle');
  await expect(page.locator('body')).not.toContainText(/Loading[\w ]*…/);
  await page.evaluate(async () => {
    await document.fonts.ready;
    // Lazy images below the first screen would never start loading.
    for (const img of document.images) img.loading = 'eager';
    await Promise.all([...document.images].map(img => img.complete ? null : new Promise(done => { img.onload = img.onerror = done; })));
  });
}

for (const theme of THEMES) {
  for (const viewport of VIEWPORTS) {
    test.describe(`${theme} ${viewport.name}`, () => {
      test.use({ viewport: viewport.size });

      for (const shot of PAGES) {
        test(shot.name, async ({ page, baseURL, context }) => {
          if (shot.signedOut) await context.clearCookies();
          await context.route(url => url.origin !== new URL(baseURL!).origin, route => route.abort());
          // No logos: they come from an outside favicon service; every company shows its initials.
          await context.route('**/api/companies/logo?**', route => route.fulfill({ status: 404, body: '' }));
          await page.clock.setFixedTime(NOW);
          await page.addInitScript(name => localStorage.setItem('career-theme', name), theme);

          await page.goto(shot.path);
          await settle(page);
          await growToContent(page, viewport.size);
          await settle(page);
          await expect(page).toHaveScreenshot(`${shot.name}-${viewport.name}-${theme}.png`);
        });
      }
    });
  }
}
