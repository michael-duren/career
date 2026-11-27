import { expect, test, type Page } from '@playwright/test';

// The Audio thoughts page (docs/redesign/pages/audio-thoughts.md) against mocked thoughts.
// Times are UTC; the browser runs in America/Chicago (playwright.config.ts), six hours behind in November.
const thought = (id: string, startedAt: string, excerpt: string, extra: Record<string, unknown> = {}) =>
  ({ id, title: excerpt ? `${excerpt.split(' ').slice(0, 6).join(' ')}...` : 'New audio thought', runDate: startedAt.slice(0, 10), startedAt, tags: [], body: '', excerpt, ...extra });
const thoughts = [
  thought('t-graphs', '2026-11-25T11:36:00Z', 'Spend the next two weeks on graph problems, then union find.', { title: 'Practice plan for graphs', tags: ['study'] }),
  thought('t-load', '2026-11-26T16:24:00Z', 'Note to self about the load test and sustained throughput at the plateau.'),
  thought('t-pending', '2026-11-26T09:12:00Z', ''),
  ...Array.from({ length: 8 }, (_, i) => thought(`t-old-${i}`, `2026-11-${String(20 - i).padStart(2, '0')}T15:00:00Z`, `Older idea number ${i + 1} about the portfolio project.`)),
  thought('t-last-year', '2025-12-30T23:30:00Z', 'An idea from last year.'),
];

async function mockThoughts(page: Page) {
  const statusRequests: string[] = [];
  await page.route('**/api/entries/run**', async route => {
    const id = new URL(route.request().url()).searchParams.get('id');
    if (id) {
      const found = thoughts.find(t => t.id === id);
      return route.fulfill({ status: found ? 200 : 404, contentType: 'application/json', body: JSON.stringify(found ? { entry: { ...found, excerpt: undefined, body: found.excerpt }, revision: 'r1' } : { error: 'not found' }) });
    }
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: thoughts.map(entry => ({ entry, revision: 'r1' })), nextOffset: null }) });
  });
  await page.route('**/api/running/*/status', async route => {
    const id = decodeURIComponent(new URL(route.request().url()).pathname.split('/')[3]);
    statusRequests.push(id);
    if (id === 't-graphs') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ groupWindowMs: 5400000, clips: [
      { clipId: '11111111-1111-4111-8111-111111111111', status: 'done', transcript: 'Spend the next two weeks on graph problems.', error: '', recordedAt: '2026-11-25T11:36:00Z' },
      { clipId: '22222222-2222-4222-8222-222222222222', status: 'failed', transcript: '', error: 'whisper: connection refused', recordedAt: '2026-11-25T11:40:00Z' },
    ] }) });
    if (id === 't-load') return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ groupWindowMs: 5400000, clips: [
      { clipId: '33333333-3333-4333-8333-333333333333', status: 'done', transcript: 'Note to self.', error: '', recordedAt: '2026-11-26T16:24:00Z' },
    ] }) });
    if (id === 't-old-0') return route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'database unavailable' }) });
    // A thought typed in the editor: the server answers 200 with no clips.
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ groupWindowMs: 5400000, clips: [] }) });
  });
  return statusRequests;
}

test.beforeEach(async ({ page }) => { await page.clock.setFixedTime(new Date('2026-11-27T12:00:00')); });

/** 0.1 s of 8 kHz mono 16-bit silence. */
function silentWav() {
  const samples = 800, data = samples * 2, wav = Buffer.alloc(44 + data);
  wav.write('RIFF', 0); wav.writeUInt32LE(36 + data, 4); wav.write('WAVEfmt ', 8); wav.writeUInt32LE(16, 16);
  wav.writeUInt16LE(1, 20); wav.writeUInt16LE(1, 22); wav.writeUInt32LE(8000, 24); wav.writeUInt32LE(16000, 28);
  wav.writeUInt16LE(2, 32); wav.writeUInt16LE(16, 34); wav.write('data', 36); wav.writeUInt32LE(data, 40);
  return wav;
}

const rows = (page: Page) => page.locator('main [data-thought-open]');

test('desktop: header, recorder target, recordings newest first with 12-hour stamps, + OLDER and transcript search', async ({ page }) => {
  await mockThoughts(page);
  await page.goto('/audio-thoughts');
  const main = page.locator('main');
  await expect(main.getByRole('heading', { level: 1, name: 'Audio thoughts' })).toBeVisible();
  await expect(main.getByText('12 recordings · transcribed on your homelab')).toBeVisible();
  await expect(main.getByRole('button', { name: 'Record', exact: true })).toHaveAttribute('aria-pressed', 'false');
  await expect(main.getByText('TAP TO RECORD')).toBeVisible();
  await expect(main.getByText('Waiting to upload', { exact: true })).toBeVisible();
  await expect(main.getByText('Transcriber', { exact: true })).toBeVisible();
  await expect(main.getByRole('meter', { name: 'Microphone level' })).toBeVisible();

  // Newest first, eight rows, then "+ N OLDER".
  await expect(rows(page)).toHaveCount(8);
  const first = rows(page).first();
  await expect(first).toContainText('NOV 26');
  await expect(first).toContainText('10:24 AM');
  await expect(first).toContainText('Note to self about the load test');
  // A pending thought says so; a chosen title is shown before the transcript.
  await expect(rows(page).nth(1)).toContainText('Waiting for transcription…');
  await expect(rows(page).nth(2).locator('span.line-clamp-2')).toHaveText('Practice plan for graphs — Spend the next two weeks on graph problems, then union find.');
  // Searching shows matches past the first eight rows without expanding.
  const search = main.getByRole('searchbox', { name: 'Search transcripts' }).filter({ visible: true });
  await search.fill('number 8 about');
  await expect(rows(page)).toHaveCount(1);
  await expect(rows(page)).toContainText('NOV 13');
  await expect(main.getByRole('button', { name: /OLDER/ })).toHaveCount(0);
  await search.fill('');
  await expect(rows(page)).toHaveCount(8);
  await main.getByRole('button', { name: '+ 4 OLDER' }).click();
  await expect(rows(page)).toHaveCount(12);
  await expect(rows(page).last()).toContainText('DEC 30 2025');
  await expect(rows(page).last()).toContainText('5:30 PM');
  await expect(main.getByRole('button', { name: /OLDER/ })).toHaveCount(0);

  // Search reaches the transcript, not just the title.
  await search.fill('union find');
  await expect(rows(page)).toHaveCount(1);
  await expect(rows(page)).toContainText('Practice plan for graphs');
  await search.fill('kafka');
  await expect(rows(page)).toHaveCount(0);
  await expect(main.getByText('No recordings match.')).toBeVisible();
});

test('desktop: opening a thought shows its takes with 12-hour times and retranscribe', async ({ page }) => {
  await mockThoughts(page);
  await page.goto('/audio-thoughts');
  const main = page.locator('main');
  await main.getByRole('button', { name: 'Open Practice plan for graphs' }).click();
  await expect(main.getByRole('heading', { name: 'Practice plan for graphs' })).toBeVisible();
  await expect(main.getByText('Wed 25 Nov · 5:36 AM')).toBeVisible();
  await expect(main.getByRole('button', { name: 'Record', exact: true })).toBeHidden();
  const takes = main.getByRole('region', { name: 'Audio takes' });
  await expect(takes.getByText('2 takes')).toBeVisible();
  await expect(takes.getByText('NOV 25 · 5:40 AM')).toBeVisible();
  await expect(takes.getByText('whisper: connection refused')).toBeVisible();
  await expect(takes.getByRole('button', { name: 'Retranscribe' })).toHaveCount(2);

  let retranscribed = '';
  await page.route('**/api/running/clips/**/retranscribe', route => { retranscribed = route.request().url(); return route.fulfill({ status: 202, contentType: 'application/json', body: '{}' }); });
  await takes.getByRole('button', { name: 'Retranscribe' }).nth(1).click();
  await expect.poll(() => retranscribed).toContain('/api/running/clips/t-graphs/22222222-2222-4222-8222-222222222222/retranscribe');

  await main.getByRole('button', { name: '← Back to audio thoughts' }).click();
  await expect(main.getByRole('button', { name: 'Record', exact: true })).toBeVisible();
  await expect(rows(page)).toHaveCount(8);
});

test('phone: smaller target, queue line, search under the recorder, stacked rows', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockThoughts(page);
  await page.goto('/audio-thoughts');
  const main = page.locator('main');
  await expect(main.getByText(/^12 recordings/)).toBeVisible();
  await expect(main.getByText('0 waiting to upload · works offline')).toBeVisible();
  await expect(main.getByText('Transcriber', { exact: true })).toBeHidden();
  const target = await main.getByRole('button', { name: 'Record', exact: true }).boundingBox();
  expect(Math.round(target!.width)).toBe(88);
  const search = main.getByRole('searchbox', { name: 'Search transcripts' }).filter({ visible: true });
  await expect(search).toHaveCount(1);
  expect((await search.boundingBox())!.y).toBeGreaterThan(target!.y);
  await expect(rows(page).first()).toContainText('NOV 26· 10:24 AM');
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
});

test('a failed list load says so and offers a retry', async ({ page }) => {
  let fail = true;
  await page.route('**/api/entries/run**', route => fail
    ? route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'database unavailable' }) })
    : route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ entries: [], nextOffset: null }) }));
  await page.goto('/audio-thoughts');
  const main = page.locator('main');
  await expect(main.getByText('Recordings unavailable')).toBeVisible();
  fail = false;
  await main.getByRole('button', { name: 'Reload latest' }).click();
  await expect(main.getByText('0 recordings · transcribed on your homelab')).toBeVisible();
  await expect(main.getByText(/No audio thoughts yet/)).toBeVisible();
});

test('play: takes play in order, pressing again stops, one row at a time, no-audio and error states', async ({ page }) => {
  // Stub playback so the test controls when each take ends.
  await page.addInitScript(() => {
    const w = window as unknown as { played: string[]; paused: string[]; media: HTMLMediaElement[] };
    w.played = []; w.paused = []; w.media = [];
    HTMLMediaElement.prototype.play = function () { w.media.push(this); w.played.push(this.src); return Promise.resolve(); };
    HTMLMediaElement.prototype.pause = function () { if (this.src) w.paused.push(this.src); };
  });
  await mockThoughts(page);
  // Setting src still loads the file, so serve a valid (silent) WAV instead of the real clip.
  await page.route('**/api/running/clips/*/*/audio', route => route.fulfill({ status: 200, contentType: 'audio/wav', body: silentWav() }));
  await page.goto('/audio-thoughts');
  const main = page.locator('main');
  const played = () => page.evaluate(() => (window as unknown as { played: string[] }).played.map(src => new URL(src).pathname));
  const endCurrent = () => page.evaluate(() => { const media = (window as unknown as { media: HTMLMediaElement[] }).media; media[media.length - 1].dispatchEvent(new Event('ended')); });
  const clip = (note: string, id: string) => `/api/running/clips/${note}/${id}/audio`;

  // Two takes play in order, then the button returns to Play.
  await main.getByRole('button', { name: 'Play Practice plan for graphs' }).click();
  const stopGraphs = main.getByRole('button', { name: 'Stop Practice plan for graphs' });
  await expect(stopGraphs).toHaveAttribute('aria-pressed', 'true');
  await expect.poll(played).toEqual([clip('t-graphs', '11111111-1111-4111-8111-111111111111')]);
  await endCurrent();
  await expect.poll(played).toEqual([clip('t-graphs', '11111111-1111-4111-8111-111111111111'), clip('t-graphs', '22222222-2222-4222-8222-222222222222')]);
  await expect(stopGraphs).toBeVisible();
  await endCurrent();
  await expect(main.getByRole('button', { name: 'Play Practice plan for graphs' })).toHaveAttribute('aria-pressed', 'false');

  // Pressing again stops.
  await main.getByRole('button', { name: 'Play Practice plan for graphs' }).click();
  await stopGraphs.click();
  await expect(main.getByRole('button', { name: 'Play Practice plan for graphs' })).toHaveAttribute('aria-pressed', 'false');

  // Starting another row stops the first.
  await main.getByRole('button', { name: 'Play Practice plan for graphs' }).click();
  await expect(stopGraphs).toBeVisible();
  await main.getByRole('button', { name: 'Play thought from NOV 26 at 10:24 AM' }).click();
  await expect(main.getByRole('button', { name: 'Stop thought from NOV 26 at 10:24 AM' })).toBeVisible();
  await expect(main.getByRole('button', { name: 'Play Practice plan for graphs' })).toBeVisible();
  await expect.poll(played).toContain(clip('t-load', '33333333-3333-4333-8333-333333333333'));

  // A thought with no takes says so (and can be retried); a failed status request shows the error label.
  await main.getByRole('button', { name: 'Play thought from NOV 26 at 3:12 AM' }).click();
  await expect(main.getByRole('button', { name: 'No audio yet for thought from NOV 26 at 3:12 AM' })).toBeEnabled();
  await expect(main.getByRole('button', { name: 'Play thought from NOV 26 at 10:24 AM' })).toBeVisible();
  await main.getByRole('button', { name: 'Play thought from NOV 20 at 9:00 AM' }).click();
  await expect(main.getByRole('button', { name: "Couldn't play thought from NOV 20 at 9:00 AM. Try again" })).toBeVisible();
});
