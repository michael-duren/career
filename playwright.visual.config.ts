import { defineConfig, devices } from '@playwright/test';

// Visual regression suite: screenshots of every page at 1280x800 and 390x844 in all
// four themes, compared against the PNGs in tests/visual/__screenshots__.
//
// Run it with scripts/visual-tests.sh, not directly: the baselines are rendered by
// the Chromium in the official Playwright Docker image against a server seeded from
// tests/visual/workspace.json, and the script recreates exactly that (CI runs the
// same script). Rendering on another OS or font setup would not match.
// `npm run test:e2e` uses playwright.config.ts and never runs these.
const baseURL = process.env.TEST_BASE_URL || 'http://127.0.0.1:18999';

export default defineConfig({
  testDir: './tests/visual',
  // One baseline per screenshot name, no per-platform suffix: the suite only ever
  // runs in the Linux Playwright image.
  snapshotPathTemplate: '{testDir}/__screenshots__/{arg}{ext}',
  outputDir: './test-results-visual',
  fullyParallel: true,
  workers: process.env.CI ? 2 : 4,
  retries: 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never', outputFolder: 'playwright-report-visual' }]] : 'list',
  globalSetup: './tests/visual/global-setup.ts',
  expect: {
    // No threshold loosening: the data, clock, fonts and renderer are all pinned, so
    // the pages must match the baselines pixel for pixel.
    toHaveScreenshot: { animations: 'disabled', caret: 'hide', scale: 'css' },
  },
  use: {
    baseURL,
    storageState: 'playwright/.auth/visual-state.json',
    timezoneId: 'America/Chicago',
    locale: 'en-US',
    reducedMotion: 'reduce',
    colorScheme: 'dark',
    // The service worker would answer requests from its cache and around page.route.
    serviceWorkers: 'block',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'visual', use: { ...devices['Desktop Chrome'], deviceScaleFactor: 1 } }],
});
