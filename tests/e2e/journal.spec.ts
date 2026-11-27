import { expect, test, type Page } from '@playwright/test';

// The Journal page (docs/redesign/pages/journal.md): Work weeks, the private Personal journal and
// the Logs explorer, against mocked entries, goals and goal hours.
const recent = { slug: '2026/week-48', week: 48, year: 2026, dates: '2026-11-23 to 2026-11-29', hours: { LeetCode: 6, Kubernetes: 2 }, tags: [], targets: {},
  body: '## What I did\n\n### Monday — Nov 23\n- Fixed the Alloy PR\n- Read DDIA chapter 5\n\n### Wednesday\n- CKA lab: networking\n\n## Blockers\n\n## Notes\n- Ask about on-call\n' };
const older = { slug: '2026/week-26', week: 26, year: 2026, dates: '2026-10-12 to 2026-10-18', hours: {}, tags: [], targets: {},
  body: '## What I did\n\n### Friday — Oct 16\n- Did some stuff on Alloy in the morning.\n' };
const personal = [
  { id: '2026/2026-11-24', title: 'Good run, clear head', date: '2026-11-24', description: 'Morning run', tags: ['health'], body: 'Ran **8k** before work.' },
  { id: 'bio', title: 'About me', date: '', description: '', tags: [], body: 'Background.' },
];
const goals = [{ id: 'g1', title: 'Kubernetes cert', startDate: '2026-11-01', endDate: '2026-12-31', color: '#fff', steps: [], status: 'active', dependsOn: [] }];

async function mockJournal(page: Page, { weeks: initialWeeks = [recent, older], scheduleOn = true } = {}) {
  // Saves change what later loads return, as the API would (the page reloads after each save).
  let weeks = [...initialWeeks];
  const saves: unknown[] = [], personalSaves: unknown[] = [], deletes: unknown[] = [];
  const calls: string[] = [];
  const list = [...personal];
  await page.route('**/api/settings', async route => {
    if (route.request().method() !== 'GET' || scheduleOn) return route.fallback();
    const response = await route.fetch();
    const body = await response.json();
    body.modules = { ...body.modules, schedule: false };
    if (body.effective) body.effective = { ...body.effective, schedule: false };
    return route.fulfill({ response, json: body });
  });
  await page.route('**/api/goals', route => { calls.push('goals'); return route.fulfill({ json: { goals, revisions: { g1: 'rev-g1' } } }); });
  await page.route('**/api/scheduler/goal-hours', route => { calls.push('goal-hours'); return route.fulfill({ json: { goals: { g1: { totalHours: 9, weekHours: { '2026-11-23': 4.5 } } } } }); });
  await page.route('**/api/entries/week**', route => {
    const request = route.request();
    if (request.method() === 'POST') {
      const input = request.postDataJSON();
      saves.push(input);
      weeks = weeks.map(week => week.slug === input.entry.slug ? input.entry : week);
      return route.fulfill({ json: { entry: input.entry, revision: 'rev-saved' } });
    }
    return route.fulfill({ json: { entries: weeks.map(entry => ({ entry, revision: entry === recent || entry === older ? `rev-${entry.week}` : 'rev-saved' })), nextOffset: null } });
  });
  await page.route('**/api/entries/personal**', route => {
    const request = route.request();
    if (request.method() === 'POST') {
      const input = request.postDataJSON();
      personalSaves.push(input);
      list.push(input.entry);
      return route.fulfill({ json: { entry: input.entry, revision: 'rev-new' } });
    }
    if (request.method() === 'DELETE') {
      const input = request.postDataJSON();
      deletes.push(input);
      list.splice(list.findIndex(item => item.id === input.id), 1);
      return route.fulfill({ json: { deleted: true } });
    }
    const id = new URL(request.url()).searchParams.get('id');
    if (id) {
      calls.push(`personal:${id}`);
      const entry = list.find(item => item.id === id);
      return entry ? route.fulfill({ json: { entry, revision: `rev-${id}` } }) : route.fulfill({ status: 404, json: { error: 'Entry not found.' } });
    }
    // Lists carry no body.
    return route.fulfill({ json: { entries: list.map(({ body: _body, ...entry }) => ({ entry, revision: `rev-${entry.id}` })), nextOffset: null } });
  });
  return { saves, personalSaves, deletes, calls };
}

test.beforeEach(async ({ page }) => {
  await page.clock.setFixedTime(new Date('2026-11-27T12:00:00'));
  // No journal drafts left over from another test.
  await page.addInitScript(() => {
    for (const key of Object.keys(localStorage)) if (key.startsWith('career-workspace:')) localStorage.removeItem(key);
  });
});

test('work tab: week list beside the selected week, with goal hours from Schedule', async ({ page }) => {
  const { calls } = await mockJournal(page);
  await page.goto('/journal');
  const main = page.locator('main');
  await expect(main.getByText('2 work weeks · 2 personal')).toBeVisible();
  const weeks = main.getByRole('navigation', { name: 'Weeks' });
  await expect(weeks.getByRole('button')).toHaveText(['W48Nov 23 – 29', 'W26Oct 12 – 18']);
  await expect(weeks.getByRole('button', { name: /W48/ })).toHaveAttribute('aria-current', 'true');

  const article = main.locator('article');
  await expect(article.getByRole('heading', { name: 'Week 48' })).toBeVisible();
  await expect(article.getByText('Nov 23 – 29, 2026')).toBeVisible();
  await expect(article.getByText('4.5 h logged this week')).toBeVisible();
  await expect(article.getByText('Kubernetes cert 4.5 h')).toBeVisible();
  await expect(article.getByText('Earlier tracks: LeetCode 6 h · Kubernetes 2 h')).toBeVisible();
  const what = article.getByRole('region', { name: 'What I did' });
  await expect(what.getByText('Monday · Nov 23')).toBeVisible();
  await expect(what.getByText('Wednesday · Nov 25')).toBeVisible();
  await expect(what.getByRole('listitem')).toHaveText(['—Fixed the Alloy PR', '—Read DDIA chapter 5', '—CKA lab: networking']);
  await expect(article.getByRole('region', { name: 'Blockers' })).toContainText('Nothing noted.');
  await expect(article.getByRole('region', { name: 'Notes' })).toContainText('Ask about on-call');
  // The latest entry (Nov 25) is two days old: no catch-up note.
  await expect(main.getByText(/catch up\?/)).toHaveCount(0);
  expect(calls).toEqual(expect.arrayContaining(['goals', 'goal-hours']));

  await weeks.getByRole('button', { name: /W26/ }).click();
  await expect(article.getByRole('heading', { name: 'Week 26' })).toBeVisible();
  await expect(article.getByText('No hours logged this week')).toBeVisible();
  await expect(page).toHaveURL(/id=2026%2Fweek-26/);

  await main.getByRole('searchbox', { name: 'Search journal' }).fill('ddia');
  await expect(weeks.getByRole('button')).toHaveText(['W48Nov 23 – 29']);
  await main.getByRole('searchbox', { name: 'Search journal' }).fill('nothing like this');
  await expect(weeks.getByText('No weeks match your search.')).toBeVisible();
});

test('a stale journal shows the catch-up note, and ?id= opens that week', async ({ page }) => {
  await mockJournal(page, { weeks: [older] });
  await page.goto('/journal?id=2026%2Fweek-26');
  await expect(page.locator('main').getByText('last entry Oct 16 — catch up?')).toBeVisible();
  await expect(page.locator('main article').getByRole('heading', { name: 'Week 26' })).toBeVisible();
});

test('with Schedule off the week shows only its own hours and asks for no goal hours', async ({ page }) => {
  const { calls } = await mockJournal(page, { scheduleOn: false });
  await page.goto('/journal');
  const article = page.locator('main article');
  await expect(article.getByText('8 h in earlier tracks')).toBeVisible();
  await expect(article.getByRole('link', { name: 'Log work →' })).toHaveCount(0);
  await expect(article.getByText(/Kubernetes cert/)).toHaveCount(0);
  expect(calls.filter(call => call === 'goals' || call === 'goal-hours')).toEqual([]);
});

test('editing a week saves its Markdown with the revision it was read at', async ({ page }) => {
  const { saves } = await mockJournal(page);
  await page.goto('/journal');
  const article = page.locator('main article');
  await article.getByRole('button', { name: 'Edit week 48' }).click();
  const body = article.getByRole('textbox', { name: 'Markdown' });
  await expect(body).toHaveValue(recent.body);
  await body.fill('## What I did\n\n### Friday — Nov 27\n- Shipped the journal\n\n## Blockers\n- None\n');
  await article.getByRole('textbox', { name: 'Tags, comma separated' }).fill('infra, infra , redesign');
  await article.getByRole('button', { name: 'Save' }).click();

  await expect(article.getByRole('heading', { name: 'Week 48' })).toBeVisible();
  await expect(article.getByRole('region', { name: 'What I did' }).getByRole('listitem')).toHaveText(['—Shipped the journal']);
  await expect(article.getByRole('region', { name: 'Blockers' }).getByRole('listitem')).toHaveText(['—None']);
  expect(saves).toEqual([{ revision: 'rev-48', entry: { ...recent, tags: ['infra', 'redesign'], body: '## What I did\n\n### Friday — Nov 27\n- Shipped the journal\n\n## Blockers\n- None\n' } }]);
});

test('personal journal stays locked until opened, then reads, creates and deletes entries', async ({ page }) => {
  const { personalSaves, deletes, calls } = await mockJournal(page);
  await page.goto('/journal?tab=personal');
  const main = page.locator('main');
  await expect(main.getByRole('navigation', { name: 'Personal entries' })).toHaveCount(0);
  expect(calls.filter(call => call.startsWith('personal:'))).toEqual([]);
  await main.getByRole('button', { name: /2 personal entries.*Private · open to read/ }).click();

  const entries = main.getByRole('navigation', { name: 'Personal entries' });
  await expect(entries.getByRole('button')).toHaveText(['Nov 24Good run, clear head', 'UndatedAbout me']);
  const article = main.locator('article');
  await expect(article.getByRole('heading', { name: 'Good run, clear head' })).toBeVisible();
  await expect(article.getByText('Ran 8k before work.')).toBeVisible();
  await expect(article.getByText('Morning run')).toBeVisible();

  await main.getByRole('button', { name: '+ New entry' }).click();
  await article.getByRole('textbox', { name: 'Title' }).fill('Mock interview');
  await article.getByRole('textbox', { name: 'Markdown' }).fill('It went well.');
  await article.getByRole('button', { name: 'Save' }).click();
  await expect(article.getByRole('heading', { name: 'Mock interview' })).toBeVisible();
  await expect(article.getByText('It went well.')).toBeVisible();
  expect(personalSaves).toHaveLength(1);
  const saved = personalSaves[0] as { revision: unknown; entry: Record<string, unknown> };
  expect(saved.revision).toBeNull();
  expect(saved.entry).toEqual({ id: expect.any(String), title: 'Mock interview', date: '2026-11-27', description: '', tags: [], body: 'It went well.' });

  await entries.getByRole('button', { name: /About me/ }).click();
  await article.getByRole('button', { name: 'Edit About me' }).click();
  page.once('dialog', dialog => dialog.accept());
  await article.getByRole('button', { name: 'Delete' }).click();
  await expect(entries.getByRole('button', { name: /About me/ })).toHaveCount(0);
  expect(deletes).toEqual([{ id: 'bio', revision: 'rev-bio' }]);
});

test('logs search underlines matches and filters by section', async ({ page }) => {
  await mockJournal(page);
  await page.goto('/journal?tab=logs');
  const main = page.locator('main');
  // Logs has its own large search; the header search is not repeated.
  await expect(main.getByRole('searchbox', { name: 'Search journal' })).toHaveCount(0);
  const results = main.getByRole('region', { name: 'Log entries' });
  await expect(results.getByRole('status')).toHaveText('4 matches');
  await results.getByRole('searchbox', { name: 'Search logs' }).fill('alloy');
  await expect(results.getByRole('status')).toHaveText('2 matches (filtered)');
  await expect(results.locator('mark').first()).toHaveText(/alloy/i);

  await results.getByRole('searchbox', { name: 'Search logs' }).fill('');
  await main.getByRole('complementary', { name: 'Filters' }).getByRole('button', { name: /Notes/ }).click();
  await expect(results.getByRole('status')).toHaveText('1 match (filtered)');
  await results.getByRole('button', { name: /Ask about on-call/ }).click();
  await expect(results.getByRole('link', { name: 'Open in journal →' })).toHaveAttribute('href', '/journal?id=2026%2Fweek-48');
  await results.getByRole('button', { name: 'Remove filter Section: Notes' }).click();
  await expect(results.getByRole('status')).toHaveText('4 matches');
});

test.describe('phone', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('weeks expand inline one at a time', async ({ page }) => {
    await mockJournal(page);
    await page.goto('/journal');
    const main = page.locator('main');
    await expect(main.locator('article')).toHaveCount(0);
    const w48 = main.getByRole('button', { name: /W48/ }), w26 = main.getByRole('button', { name: /W26/ });
    await expect(w48).toHaveAttribute('aria-expanded', 'true');
    await expect(main.getByText('— Fixed the Alloy PR')).toBeVisible();
    await expect(main.getByText('Blockers · none')).toBeVisible();
    await w26.click();
    await expect(w26).toHaveAttribute('aria-expanded', 'true');
    await expect(w48).toHaveAttribute('aria-expanded', 'false');
    await expect(main.getByText('— Did some stuff on Alloy in the morning.')).toBeVisible();
    await w26.click();
    await expect(w26).toHaveAttribute('aria-expanded', 'false');
    await expect(main.getByRole('link', { name: 'New work entry' })).toHaveAttribute('href', '/journal/new');
  });
});

test('old journal routes redirect to their tabs', async ({ page }) => {
  await page.goto('/logs');
  await expect(page).toHaveURL(/\/journal\?tab=logs$/);
  await expect(page.locator('main').getByRole('tab', { name: 'Logs' })).toHaveAttribute('aria-selected', 'true');
  await page.goto('/personal-journal');
  await expect(page).toHaveURL(/\/journal\?tab=personal$/);
  await expect(page.locator('main').getByRole('tab', { name: 'Personal' })).toHaveAttribute('aria-selected', 'true');
});
