import test from 'node:test';
import assert from 'node:assert/strict';
import { spawn, execFileSync } from 'node:child_process';
import { chromium } from 'playwright';

const port = 18097;
const origin = `http://127.0.0.1:${port}`;

async function waitForServer(server, getLogs) {
  for (let attempt = 0; attempt < 200; attempt++) {
    try { const response = await fetch(`${origin}/`); if (response.ok) return; } catch {}
    await new Promise(resolve => setTimeout(resolve, 150));
  }
  throw new Error(`Leetgrinder preview did not start (exit ${server.exitCode}): ${getLogs()}`);
}

test('authored lessons render and the SVG player controls real page content', { timeout: 120000 }, async t => {
  const binary = '/tmp/career-leetgrinder-preview-test';
  execFileSync('go', ['build', '-o', binary, './cmd/leetgrinder-preview'], { env: { ...process.env, GOCACHE: '/tmp/career-leetgrinder-go-cache' } });
  const server = spawn(binary, ['-addr', `127.0.0.1:${port}`], { stdio: 'pipe' });
  let serverLogs = '';
  server.stderr.on('data', chunk => { serverLogs += chunk.toString(); });
  t.after(() => server.kill('SIGTERM'));
  await waitForServer(server, () => serverLogs);
  const browser = await chromium.launch({ headless: true });
  t.after(() => browser.close());
  const page = await browser.newPage({ viewport: { width: 360, height: 800 } });
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto(`${origin}/day/1`);
  await page.waitForTimeout(500);
  assert.equal(await page.locator('[data-lesson-example="day-01-complement-lookup"] .lesson-player').count(), 1, `player missing; page errors: ${errors.join('; ')}; examples: ${await page.locator('[data-lesson-example]').count()}`);
  const player = page.locator('[data-lesson-example="day-01-complement-lookup"]');
  assert.equal(await player.locator('[data-player-fallback]').isVisible(), false);
  assert.match(await player.locator('.lesson-player-explanation').textContent(), /map|index|start|first/i);
  await player.getByRole('button', { name: 'Next' }).click();
  assert.equal(await player.locator('.lesson-player-svg').count(), 1);
  await player.getByRole('tab', { name: 'Go' }).click();
  assert.equal(await player.getByRole('tab', { name: 'Go' }).getAttribute('aria-selected'), 'true');
  await player.getByRole('button', { name: 'Start' }).click();
  assert.equal(await player.getByRole('button', { name: 'Previous' }).isDisabled(), true);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, JSON.stringify(await page.evaluate(() => [...document.querySelectorAll('*')].filter(el => el.getBoundingClientRect().right > innerWidth + 2).slice(0, 8).map(el => ({tag:el.tagName, class:el.className?.baseVal || el.className, right:el.getBoundingClientRect().right})))));
  assert.deepEqual(errors, []);

  await page.goto(`${origin}/day/84`);
  const guidance = page.locator('details.review-guidance');
  assert.equal(await guidance.getAttribute('open'), null);
  assert.equal(await guidance.locator('.lesson-references').count(), 1);
  assert.equal(await page.getByText('Reading path').count(), 0);

  for (const day of [1, 32, 66]) {
    await page.goto(`${origin}/day/${day}`);
    assert.equal(await page.locator('[data-lesson-example]').count() > 0, true);
    for (const example of await page.locator('[data-lesson-example]').all()) {
      assert.equal(await example.locator('.lesson-player').count(), 1);
      assert.equal(await example.locator('.lesson-player-svg').count(), 1);
      await example.getByRole('button', { name: 'Next' }).click();
      assert.equal(await example.locator('.lesson-player-code .is-active').count() > 0, true);
      await example.getByRole('tab', { name: 'C++' }).click();
      assert.equal(await example.getByRole('tab', { name: 'C++' }).getAttribute('aria-selected'), 'true');
      await example.getByRole('button', { name: 'Start' }).click();
    }
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `day ${day} overflows at 360px: ${JSON.stringify(await page.evaluate(() => [...document.querySelectorAll('*')].filter(el => el.getBoundingClientRect().right > innerWidth + 2).slice(0, 10).map(el => ({tag:el.tagName, cls:el.className?.baseVal || el.className, right:el.getBoundingClientRect().right}))))}`);
    assert.deepEqual(errors, []);
  }
  const noScript = await browser.newContext({ javaScriptEnabled: false });
  const fallbackPage = await noScript.newPage();
  await fallbackPage.goto(`${origin}/day/1`);
  assert.equal(await fallbackPage.locator('[data-player-fallback] pre code').count(), 4 * await fallbackPage.locator('[data-lesson-example]').count());
  await noScript.close();
});
