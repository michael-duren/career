import { expect, test } from '@playwright/test';

// The Audio thoughts recorder with Chromium's fake microphone (a tone), so the live waveform has input.
// launchOptions starts its own browser, so this lives apart from audio-thoughts.spec.ts.
test.use({ permissions: ['microphone'], launchOptions: { args: ['--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream'] } });

test('the waveform follows the live input, and the take is queued and uploaded', async ({ page }) => {
  await page.route('**/api/entries/run**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: [], nextOffset: null }) }));
  const uploads: string[] = [];
  await page.route('**/api/running/clips', async route => {
    const clientId = /name="clientId"\r\n\r\n([^\r]+)/.exec(route.request().postDataBuffer()?.toString('latin1') ?? '')?.[1] ?? '';
    uploads.push(clientId);
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ clipId: clientId, noteId: 'run-new' }) });
  });
  await page.goto('/audio-thoughts');
  const main = page.locator('main');
  const record = main.getByRole('button', { name: 'Record', exact: true });
  const wave = main.getByRole('meter', { name: 'Microphone level' }).locator('polyline');
  const heights = async () => new Set((await wave.getAttribute('points'))!.split(' ').map(point => point.split(',')[1])).size;
  expect(await heights()).toBe(1);
  await record.click();
  await expect(record).toHaveAttribute('aria-pressed', 'true');
  await expect(main.getByText('TAP TO STOP')).toBeVisible();
  await expect.poll(heights, { timeout: 5000 }).toBeGreaterThan(1);
  await expect(main.getByRole('timer', { name: 'Elapsed recording time' })).toHaveText(/^00:0[1-9]\.\d/);
  await record.click();
  await expect(main.getByText('Take saved on this device.')).toBeVisible();
  await expect.poll(() => uploads.length).toBe(1);
  expect(uploads[0]).toMatch(/^[0-9a-f-]{36}$/);
  await expect(main.getByText('TAP TO RECORD')).toBeVisible();
});
