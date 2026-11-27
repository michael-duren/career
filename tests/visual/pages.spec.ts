import { test, expect, type Page } from '@playwright/test';
import { THEMES } from '../../src/lib/theme.ts';

// Every page at desktop and phone size in every theme, against tests/visual/workspace.json.
// Run with scripts/visual-tests.sh (see playwright.visual.config.ts and docs/developer.md).
//
// Determinism: the browser clock is frozen at NOW (the fixture's dates are relative to
// it), the time zone is the scheduler's, outside requests are blocked (company logos
// come from a favicon service), and each page waits for its data, fonts and images
// before the shot. The server keeps its real clock (Leetgrinder's day, the scheduler's
// capacity and missed sessions), but every fixture date is before the day the baselines
// were made, so it sees the same past whenever the suite runs and nothing needs masking.
// If a page ever shows the server's own date, mask it with toHaveScreenshot's `mask`.

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
    const content = Math.min(await page.evaluate(() => document.documentElement.scrollHeight), 6000);
    if (content <= height) return;
    height = content;
    await page.setViewportSize({ width: size.width, height });
  }
}

async function settle(page: Page) {
  await page.waitForLoadState('networkidle');
  await expect(page.locator('body')).not.toContainText(/Loading[\w ]*…/);
  await page.evaluate(async () => {
    await document.fonts.ready;
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
          await context.route(url => !url.href.startsWith(baseURL!), route => route.abort());
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
