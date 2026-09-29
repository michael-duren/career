import { defineConfig, devices } from '@playwright/test';

// Mirrors the existing tests/*.browser.mjs convention: point at an already-
// running, disposable authenticated workspace via env vars. This config does
// not spin up the app itself - see scripts/start-local.sh (local) or the
// "e2e" CI job (throwaway Postgres + built binary) for that.
const baseURL = process.env.TEST_BASE_URL || 'http://127.0.0.1:4337';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  globalSetup: './tests/e2e/global-setup.ts',
  use: {
    baseURL,
    storageState: 'playwright/.auth/state.json',
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
    // WeeklyScheduler guesses "today" from the browser's local timezone on
    // first mount, then re-syncs to the workspace's configured timezone once
    // the initial fetch resolves - a second background reload. Matching the
    // two avoids that reload (and the "date belongs to a different week"
    // race it can otherwise cause if a dialog is opened in between).
    timezoneId: 'America/Chicago',
    // Tall enough that a day column's full 05:00-20:30 range fits on screen -
    // a drag/resize target computed from a day's bounding box otherwise can
    // land just past the viewport edge, where elementFromPoint returns null.
    viewport: { width: 1280, height: 1400 },
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
