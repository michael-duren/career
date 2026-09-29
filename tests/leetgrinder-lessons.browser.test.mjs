import test from 'node:test';
import assert from 'node:assert/strict';
import { spawn, execFileSync } from 'node:child_process';
import { chromium } from 'playwright';
import { readdirSync } from 'node:fs';

const port = 18097;
const origin = `http://127.0.0.1:${port}`;

async function waitForServer(server, getLogs) {
  for (let attempt = 0; attempt < 200; attempt++) {
    try { const response = await fetch(`${origin}/`); if (response.ok) return; } catch {}
    await new Promise(resolve => setTimeout(resolve, 150));
  }
  throw new Error(`Leetgrinder preview did not start (exit ${server.exitCode}): ${getLogs()}`);
}

test('authored lessons render and the SVG player controls real page content', { timeout: 20 * 60 * 1000 }, async t => {
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

  const authoredDays = process.env.LEETGRINDER_DAYS
    ? process.env.LEETGRINDER_DAYS.split(',').map(Number)
    : readdirSync('internal/leetgrinder/lessons').filter(name => /^day-\d\d\.json$/.test(name)).map(name => Number(name.slice(4, 6))).sort((a, b) => a - b);
  for (const day of authoredDays) {
    await page.goto(`${origin}/day/${day}`);
    const guidance = page.locator('details.review-guidance');
    if (day >= 78) {
      assert.equal(await guidance.getAttribute('open'), null, `day ${day} guidance should start closed`);
      await guidance.locator('summary').click();
    }
    const examples = await page.locator('[data-lesson-example]').all();
    assert.ok(examples.length > 0, `day ${day} needs linked examples`);
    for (const example of examples) {
      const payload = await example.evaluate(element => JSON.parse(element.querySelector('[data-player-data]').content.textContent));
      const player = example.locator('.lesson-player');
      assert.equal(await player.count(), 1, `${payload.id} player missing`);
      assert.equal(await example.locator('.lesson-player-svg title').count(), 1);
      for (const traceCase of payload.cases) {
        await example.locator('.lesson-player-case').selectOption(traceCase.id);
        for (let frameIndex = 0; frameIndex < traceCase.frames.length; frameIndex++) {
          if (frameIndex) await example.getByRole('button', { name: 'Next' }).click();
          const frame = traceCase.frames[frameIndex];
          assert.equal(await example.locator('.lesson-player-explanation').textContent(), frame.explanation, `${payload.id}/${traceCase.id} frame ${frameIndex}`);
          assert.equal(await example.locator('.lesson-player-svg desc').textContent(), frame.scene.description, `${payload.id}/${traceCase.id} scene ${frameIndex}`);
          const variables = await example.locator('.lesson-player-variables tbody tr').evaluateAll(rows => rows.map(row => ({ name: row.querySelector('th').textContent, value: row.querySelector('td').textContent })));
          assert.deepEqual(variables, frame.variables, `${payload.id}/${traceCase.id} variables ${frameIndex}`);
          for (const [language, label] of [['cpp', 'C++'], ['python', 'Python'], ['java', 'Java'], ['go', 'Go']]) {
            const tab = example.getByRole('tab', { name: label });
            await tab.click();
            assert.equal(await tab.getAttribute('aria-selected'), 'true');
            const activeLines = await example.locator('.lesson-player-code .is-active').evaluateAll(lines => lines.map(line => Number(line.dataset.line)));
            assert.deepEqual(activeLines, frame.lines[language], `${payload.id}/${traceCase.id}/${language} lines at frame ${frameIndex}`);
          }
        }
        assert.equal(await example.getByRole('button', { name: 'Next' }).isDisabled(), true);
      }
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
