import { expect, test, type Page } from '@playwright/test';

// The People page (docs/redesign/pages/people.md) against mocked connections and companies.

const person = (id: string, name: string, role: string, companyName: string, extra: Record<string, unknown> = {}): Record<string, unknown> & { id: string; name: string; stage?: string } =>
  ({ id: `00000000-0000-4000-8000-${id.padStart(12, '0')}`, name, role, companyName, email: '', queued: false, notes: '', tags: [], ...extra });

const people = [
  person('1', 'Paige North', 'Senior Software Engineer', 'Microsoft', { companySlug: 'microsoft', lastContactedOn: '2026-09-11', connectedOn: '2024-02-01', url: 'https://www.linkedin.com/in/paige', notes: 'Met at a meetup' }),
  person('2', 'Brian Holt', 'Principal TPM', 'Microsoft', { companySlug: 'microsoft' }),
  person('3', 'Adam Iverson', 'Software Engineer II', 'Maximus Inc'),
  person('4', 'Alex Stone', 'Software Engineer', 'maximus'),
  person('5', 'Kayla Altepeter', 'Staff Engineer', 'Datasite', { queued: true }),
];
// The catch-up queue's people: three on the board, one snoozed, one candidate (linked, no stage).
const queuePeople = [
  person('11', 'Hayden Kotelman', 'Senior SWE', 'Grafana Labs', { companySlug: 'grafana', stage: 'to_contact' }),
  person('12', 'Paige North', 'Senior SWE', 'Microsoft', { companySlug: 'microsoft', stage: 'messaged', lastContactedOn: '2026-09-11' }),
  person('13', 'Nicolas Perez', 'Senior SWE', 'Google', { companySlug: 'google', stage: 'follow_up', lastContactedOn: '2026-09-25' }),
  person('14', 'Snoozy Sam', 'SWE', 'Stripe', { companySlug: 'stripe', stage: 'to_contact', snoozedUntil: '2026-12-01' }),
  person('15', 'Joey Brown', 'Lead MTS', 'Heroku', { companySlug: 'heroku' }),
  person('16', 'Dana Unlinked', 'Engineer', 'Nowhere Inc'),
];
const goal = (id: string, title: string, extra: Record<string, unknown> = {}) => ({
  id, title, status: 'active', dependsOn: [], startDate: '2026-11-01', endDate: '2027-01-31', color: '#888888', createdAt: '2026-11-01T00:00:00Z', updatedAt: '2026-11-01T00:00:00Z',
  notes: [], metadata: {}, steps: [], ...extra,
});
const goals = [
  goal('g1', 'Warm up connections for Q1 referrals', { metadata: { catchUpQueue: 'linked' }, steps: [
    { id: 's1', title: 'Pick 5–6 contacts', done: true }, { id: 's2', title: 'Send first messages', done: false },
    { id: 's3', title: 'Follow up', done: false }, { id: 's4', title: 'Referral asks in Jan', done: false }] }),
  goal('g2', 'System design prep'),
];

/** Mock the queue API on top of mockPeople; records every queue request. */
async function mockQueue(page: Page) {
  const calls: { method: string; path: string; body: Record<string, unknown> }[] = [];
  const goalSaves: Record<string, unknown>[] = [];
  const week = { start: '2026-11-23', end: '2026-11-29', sent: 0, target: 2 };
  await mockPeople(page, { list: queuePeople });
  await page.route('**/api/connections/queue**', route => {
    const request = route.request();
    const path = new URL(request.url()).pathname.replace('/api/connections/queue', '') || '/';
    if (request.method() === 'GET') return route.fulfill({ json: { today: '2026-11-26', week, connections: queuePeople.filter(p => p.stage), revisions: {} } });
    const body = request.postDataJSON();
    calls.push({ method: request.method(), path, body });
    if (path === '/settings') { week.target = body.weeklyTarget; return route.fulfill({ json: { week } }); }
    const current = queuePeople.find(p => p.id === body.id)!;
    const stages = ['to_contact', 'messaged', 'follow_up', 'referral_ask'];
    let entry: Record<string, unknown> = { ...current };
    if (path === '/sent') { week.sent++; entry = { ...entry, stage: stages[stages.indexOf(String(current.stage)) + 1], lastContactedOn: '2026-11-26' }; }
    else if (path === '/snooze') entry = { ...entry, snoozedUntil: body.until };
    else if (request.method() === 'DELETE') { const { stage: _s, snoozedUntil: _u, ...rest } = entry; entry = rest; }
    else { const { snoozedUntil: _u, ...rest } = entry; entry = { ...rest, stage: body.stage }; }
    return route.fulfill({ json: { entry, revision: 'rev-queue', ...(path === '/sent' ? { week } : {}) } });
  });
  await page.route('**/api/goals', route => {
    if (route.request().method() === 'POST') {
      const { goal: saved } = route.request().postDataJSON();
      goalSaves.push(saved);
      return route.fulfill({ json: { goal: saved, revision: `rev-${saved.id}` } });
    }
    return route.fulfill({ json: { goals, revisions: { g1: 'rev-g1', g2: 'rev-g2' } } });
  });
  return { calls, goalSaves, week };
}
const companies = [{ slug: 'microsoft', title: 'Microsoft', category: 'Big tech', type: 'company', url: 'https://microsoft.com', status: 'not_started', featured: false, priority: 'medium', tags: [], body: '' }];

async function mockPeople(page: Page, { companiesOn = true, list = people } = {}) {
  const saves: Record<string, unknown>[] = [];
  const created: Record<string, unknown>[] = [];
  await page.route('**/api/settings', async route => {
    if (route.request().method() !== 'GET' || companiesOn) return route.fallback();
    const response = await route.fetch();
    const body = await response.json();
    body.modules = { ...body.modules, companies: false };
    if (body.effective) body.effective = { ...body.effective, companies: false };
    return route.fulfill({ response, json: body });
  });
  await page.route('**/api/connections', route => route.fulfill({ json: { connections: list, revisions: Object.fromEntries(list.map(p => [p.id, `rev-${p.id}`])) } }));
  // Without staged people the queue is empty; mockQueue replaces this with a working queue.
  await page.route('**/api/connections/queue', route => route.fulfill({ json: { today: '2026-11-26', week: { start: '2026-11-23', end: '2026-11-29', sent: 0, target: 2 }, connections: [], revisions: {} } }));
  await page.route('**/api/entries/connection', route => {
    const { entry } = route.request().postDataJSON();
    saves.push(entry);
    return route.fulfill({ json: { entry, revision: 'rev-new' } });
  });
  await page.route('**/api/entries/company**', route => {
    if (route.request().method() === 'POST') {
      const { entry } = route.request().postDataJSON();
      created.push(entry);
      return route.fulfill({ json: { entry, revision: 'rev-company' } });
    }
    return route.fulfill({ json: { entries: companies.map(entry => ({ entry, revision: `rev-${entry.slug}` })), nextOffset: null } });
  });
  return { saves, created };
}

test('people tab lists everyone and opens a person in the detail panel', async ({ page }) => {
  const { saves } = await mockPeople(page);
  await page.goto('/connections');
  const main = page.locator('main');
  await expect(main.getByText('5 people · 3 employers')).toBeVisible();
  await expect(main.getByRole('tab', { name: 'People' })).toHaveAttribute('aria-selected', 'true');
  await expect(main.getByRole('tab', { name: 'Catch-up queue · 0' })).toBeVisible();

  const row = main.getByRole('button', { name: /Paige North/ });
  await expect(row).toContainText('Sep 11');
  await expect(row).toContainText('on your companies board');
  await row.click();
  await expect(row).toHaveAttribute('aria-pressed', 'true');
  const panel = main.getByRole('complementary', { name: 'About Paige North' });
  await expect(panel).toContainText('Senior Software Engineer · Microsoft');
  await expect(panel).toContainText('2At Microsoft');
  await expect(panel.getByRole('link', { name: 'LinkedIn ↗' })).toHaveAttribute('href', 'https://www.linkedin.com/in/paige');

  await panel.getByRole('textbox', { name: 'Notes' }).fill('Ask about the platform team');
  await panel.getByRole('button', { name: 'Save notes' }).click();
  await expect(main.getByRole('status')).toHaveText('Notes saved for Paige North.');
  expect(saves.at(-1)).toMatchObject({ name: 'Paige North', notes: 'Ask about the platform team' });

  // Notes typed but not saved go along with Log catch-up instead of being dropped.
  await panel.getByRole('textbox', { name: 'Notes' }).fill('Follow up in January');
  const before = saves.length;
  await panel.getByRole('button', { name: 'Log catch-up' }).click();
  await expect(main.getByRole('status')).toHaveText('Logged a catch-up with Paige North.');
  expect(saves.length).toBe(before + 1);
  const today = await page.evaluate(() => { const d = new Date(); return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`; });
  expect(saves.at(-1)?.lastContactedOn).toBe(today);
  expect(saves.at(-1)?.notes).toBe('Follow up in January');
  await expect(panel.getByRole('textbox', { name: 'Notes' })).toHaveValue('Follow up in January');

  await main.getByRole('searchbox', { name: 'Search people or roles' }).fill('maximus');
  await expect(main.getByRole('button', { name: /Adam Iverson/ })).toBeVisible();
  await expect(main.getByRole('button', { name: /Brian Holt/ })).toHaveCount(0);
});

test('employers tab groups people per employer and adds one to the companies board', async ({ page }) => {
  const { created } = await mockPeople(page);
  await page.goto('/connections?tab=employers');
  const main = page.locator('main');
  await expect(main.getByRole('tab', { name: 'Employers' })).toHaveAttribute('aria-selected', 'true');
  await expect(main.getByRole('button', { name: 'Maximus Inc: 2 people' })).toBeVisible();
  await expect(main.getByRole('button', { name: 'Microsoft: 2 people, on your companies board' })).toBeVisible();

  await main.getByRole('checkbox', { name: 'Maximus Inc · 2 people' }).check();
  await main.getByRole('button', { name: 'Add to companies' }).click();
  await expect(main.getByRole('status')).toContainText('Added 1 company to your board.');
  expect(created).toEqual([expect.objectContaining({ slug: 'maximus-inc', title: 'Maximus Inc', status: 'not_started' })]);

  // An employer opens the People tab filtered to the people who work there.
  await main.getByRole('button', { name: 'Microsoft: 2 people, on your companies board' }).click();
  await expect(main.getByRole('tab', { name: 'People' })).toHaveAttribute('aria-selected', 'true');
  await expect(main.getByRole('button', { name: /Brian Holt/ })).toBeVisible();
  await expect(main.getByRole('button', { name: /Adam Iverson/ })).toHaveCount(0);

  // An employer known only by LinkedIn name shows every spelling that grouped into it.
  await main.getByRole('button', { name: 'Clear employer filter: Microsoft' }).click();
  await main.getByRole('tab', { name: 'Employers' }).click();
  await main.getByRole('button', { name: 'Maximus Inc: 2 people' }).click();
  await expect(main.getByRole('button', { name: /Adam Iverson/ })).toBeVisible();
  await expect(main.getByRole('button', { name: /Alex Stone/ })).toBeVisible();
  await expect(main.getByRole('button', { name: /Paige North/ })).toHaveCount(0);
});

test('an unlinked person at a company already on the board counts toward it, not as a new employer', async ({ page }) => {
  people.push(person('6', 'Dana Unlinked', 'Engineer', 'Microsoft Corporation'));
  try {
    await mockPeople(page);
    await page.goto('/connections?tab=employers');
    const main = page.locator('main');
    await expect(main.getByRole('button', { name: 'Microsoft: 3 people, on your companies board' })).toBeVisible();
    await expect(main.getByRole('checkbox', { name: /Microsoft/ })).toHaveCount(0);
  } finally {
    people.pop();
  }
});

test('tabs follow the keyboard and the address keeps the open tab', async ({ page }) => {
  await mockPeople(page);
  await page.goto('/connections');
  const main = page.locator('main');
  await main.getByRole('tab', { name: 'People' }).focus();
  await page.keyboard.press('ArrowRight');
  await expect(main.getByRole('tab', { name: /Catch-up queue/ })).toHaveAttribute('aria-selected', 'true');
  await expect(page).toHaveURL(/tab=queue/);
  await expect(main.getByRole('heading', { name: 'To contact' })).toBeVisible();
  // Kayla is flagged the old way, so she shows in the cadence list under the board.
  await expect(main.getByRole('listitem').filter({ hasText: 'Kayla Altepeter' })).toContainText('Flagged');
  await page.reload();
  await expect(main.getByRole('tab', { name: /Catch-up queue/ })).toHaveAttribute('aria-selected', 'true');
});

test('the catch-up tab is hidden with a note when Companies is off', async ({ page }) => {
  await mockPeople(page, { companiesOn: false });
  const queueRequests: string[] = [];
  page.on('request', request => { if (new URL(request.url()).pathname.startsWith('/api/connections/queue')) queueRequests.push(request.url()); });
  await page.goto('/connections?tab=queue');
  const main = page.locator('main');
  await expect(main.getByText('The catch-up queue needs the Companies module')).toBeVisible();
  await expect(main.getByRole('tab')).toHaveText(['People', 'Employers']);
  await expect(main.getByRole('tab', { name: 'People' })).toHaveAttribute('aria-selected', 'true');
  await expect(main.getByRole('button', { name: /Paige North/ })).not.toContainText('on your companies board');

  // Employers group by LinkedIn name only, with no board marks or add list, and the filter clears.
  await main.getByRole('tab', { name: 'Employers' }).click();
  await expect(main.getByText('Not on your board yet')).toHaveCount(0);
  await expect(main.getByRole('button', { name: 'Add to companies' })).toHaveCount(0);
  await expect(main.getByText('On your companies board')).toHaveCount(0);
  await main.getByRole('button', { name: 'Microsoft: 2 people' }).click();
  await expect(main.getByRole('button', { name: /Adam Iverson/ })).toHaveCount(0);
  await main.getByRole('button', { name: 'Clear employer filter: Microsoft' }).click();
  await expect(main.getByRole('button', { name: /Adam Iverson/ })).toBeVisible();
  // Nothing on this page asks for the queue while Companies is off.
  expect(queueRequests).toEqual([]);
});

test('on a phone the detail replaces the list until you go back', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockPeople(page);
  await page.goto('/connections');
  const main = page.locator('main');
  await expect(main.getByRole('heading', { level: 1, name: 'People', exact: true })).toBeVisible();
  await expect(main.getByRole('tab', { name: 'Catch-up · 0' })).toBeVisible();
  await main.getByRole('button', { name: /Paige North/ }).click();
  await expect(main.getByRole('complementary', { name: 'About Paige North' })).toBeVisible();
  await expect(main.getByRole('button', { name: /Brian Holt/ })).toBeHidden();
  await main.getByRole('button', { name: '← ALL PEOPLE' }).click();
  await expect(main.getByRole('button', { name: /Brian Holt/ })).toBeVisible();
});

test('catch-up board shows stages, hides snoozed people and drives the queue API', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  const { calls } = await mockQueue(page);
  await page.goto('/connections?tab=queue');
  const main = page.locator('main');
  // Snoozed people do not count and sit in their own list.
  await expect(main.getByRole('tab', { name: 'Catch-up queue · 3' })).toHaveAttribute('aria-selected', 'true');
  await expect(main.getByText('Warm up connections for Q1 referrals')).toBeVisible();
  await expect(main.getByText('step 2 of 4: Pick 5–6 contacts (done), Send first messages, Follow up, Referral asks in Jan')).toBeAttached();
  await expect(main.getByText('Messages this week: 0 / 2')).toBeVisible();
  const lane = (name: string) => main.getByRole('region', { name });
  await expect(lane('To contact').getByRole('button', { name: /Hayden Kotelman/ })).toContainText('NO CONTACT LOGGED');
  await expect(lane('Messaged').getByRole('button', { name: /Paige North/ })).toContainText('LAST · SEP 11');
  await expect(lane('Referral ask')).toContainText('Nobody here yet');
  await expect(lane('To contact')).not.toContainText('Snoozy Sam');
  await expect(main.getByRole('region', { name: 'Snoozed' })).toContainText('Back Dec 1');

  // The first card is picked on wide screens; pick Paige for the panel.
  await expect(main.getByRole('complementary', { name: 'Catch up with Hayden Kotelman' })).toContainText('never contacted');
  await lane('Messaged').getByRole('button', { name: /Paige North/ }).click();
  await expect(lane('Messaged').getByRole('button', { name: /Paige North/ })).toHaveAttribute('aria-pressed', 'true');
  const panel = main.getByRole('complementary', { name: 'Catch up with Paige North' });
  await expect(panel).toContainText('Follow up on your last message.');
  await expect(panel).toContainText('last contact Sep 11 · 76 days ago');
  await panel.getByRole('textbox', { name: 'Message draft' }).fill('Hi Paige, any news?');

  await panel.getByRole('button', { name: 'Mark sent' }).click();
  await expect(main.getByRole('status')).toHaveText('Sent. Paige North moved to Follow up.');
  expect(calls.at(-1)).toEqual({ method: 'POST', path: '/sent', body: { id: queuePeople[1].id, revision: 'rev-' + queuePeople[1].id } });
  await expect(main.getByText('Messages this week: 1 / 2')).toBeVisible();
  await expect(lane('Follow up').getByRole('button', { name: /Paige North/ })).toBeVisible();
  // A sent message clears its draft.
  await expect(main.getByRole('complementary', { name: 'Catch up with Paige North' }).getByRole('textbox', { name: 'Message draft' })).toHaveValue('');

  const [, paige, nicolas, sam, joey] = queuePeople;
  const tab = (n: number) => main.getByRole('tab', { name: `Catch-up queue · ${n}` });
  await main.getByRole('complementary').getByRole('button', { name: 'Snooze 1 wk' }).click();
  await expect(main.getByRole('status')).toHaveText('Paige North snoozed until Dec 3.');
  expect(calls.at(-1)).toEqual({ method: 'POST', path: '/snooze', body: { id: paige.id, revision: 'rev-queue', until: '2026-12-03' } });
  await expect(lane('Follow up')).not.toContainText('Paige North');
  await expect(main.getByRole('region', { name: 'Snoozed' })).toContainText('Paige North');
  await expect(tab(2)).toBeVisible();

  await lane('Follow up').getByRole('button', { name: /Nicolas Perez/ }).click();
  await main.getByRole('group', { name: 'Move to' }).getByRole('button', { name: 'Referral ask' }).click();
  await expect(main.getByRole('status')).toHaveText('Nicolas Perez moved to Referral ask.');
  expect(calls.at(-1)).toEqual({ method: 'POST', path: '/', body: { id: nicolas.id, revision: `rev-${nicolas.id}`, stage: 'referral_ask' } });
  await expect(lane('Referral ask').getByRole('button', { name: /Nicolas Perez/ })).toBeVisible();
  await expect(lane('Follow up')).toContainText('Nobody here yet');

  // Candidates are linked people without a stage.
  await main.getByRole('button', { name: '+ Add people' }).click();
  const candidates = main.getByRole('region', { name: 'Add people at board companies' });
  await expect(candidates.getByRole('button', { name: /^Add .* to the queue$/ })).toHaveCount(1);
  await candidates.getByRole('button', { name: 'Add Joey Brown to the queue' }).click();
  await expect(main.getByRole('status')).toHaveText('Joey Brown added to the queue.');
  expect(calls.at(-1)).toEqual({ method: 'POST', path: '/', body: { id: joey.id, revision: `rev-${joey.id}`, stage: 'to_contact' } });
  await expect(lane('To contact').getByRole('button', { name: /Joey Brown/ })).toBeVisible();
  await expect(tab(3)).toBeVisible();

  await lane('Referral ask').getByRole('button', { name: /Nicolas Perez/ }).click();
  await main.getByRole('complementary', { name: 'Catch up with Nicolas Perez' }).getByRole('button', { name: 'Remove from queue' }).click();
  await expect(main.getByRole('status')).toHaveText('Nicolas Perez removed from the queue.');
  expect(calls.at(-1)).toEqual({ method: 'DELETE', path: '/', body: { id: nicolas.id, revision: 'rev-queue' } });
  await expect(lane('Referral ask')).toContainText('Nobody here yet');
  await expect(tab(2)).toBeVisible();

  await main.getByRole('button', { name: /Change the weekly target/ }).click();
  await main.getByRole('spinbutton', { name: 'Per week' }).fill('4');
  await main.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(main.getByText('Messages this week: 1 / 4')).toBeVisible();
  expect(calls.at(-1)).toEqual({ method: 'PATCH', path: '/settings', body: { weeklyTarget: 4 } });

  await main.getByRole('region', { name: 'Snoozed' }).getByRole('button', { name: 'Unsnooze Snoozy Sam' }).click();
  await expect(main.getByRole('status')).toHaveText('Snoozy Sam is back in the queue.');
  expect(calls.at(-1)).toEqual({ method: 'POST', path: '/', body: { id: sam.id, revision: `rev-${sam.id}`, stage: 'to_contact' } });
  await expect(main.getByRole('region', { name: 'Snoozed' })).not.toContainText('Snoozy Sam');
  await expect(lane('To contact').getByRole('button', { name: /Snoozy Sam/ })).toBeVisible();
  await expect(tab(3)).toBeVisible();
});

test('a message draft survives a reload and a failed send keeps it', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  const { calls } = await mockQueue(page);
  await page.route('**/api/connections/queue/sent', route => route.fulfill({ status: 500, json: { error: 'Server unavailable.' } }));
  await page.goto('/connections?tab=queue');
  const main = page.locator('main');
  const draft = () => main.getByRole('complementary', { name: 'Catch up with Hayden Kotelman' }).getByRole('textbox', { name: 'Message draft' });
  await draft().fill('Hi Hayden, quick question');
  await page.reload();
  await expect(draft()).toHaveValue('Hi Hayden, quick question');
  await main.getByRole('button', { name: 'Mark sent' }).click();
  await expect(main.getByRole('alert')).toHaveText('Server unavailable.');
  await expect(draft()).toHaveValue('Hi Hayden, quick question');
  await expect(main.getByText('Messages this week: 0 / 2')).toBeVisible();

  // Removing someone from the queue drops their draft.
  await main.getByRole('button', { name: 'Remove from queue' }).click();
  await expect(main.getByRole('status')).toHaveText('Hayden Kotelman removed from the queue.');
  expect(calls.at(-1)?.method).toBe('DELETE');
  expect(await page.evaluate(id => localStorage.getItem(`career-catch-up-draft:${id}`), queuePeople[0].id)).toBeNull();
});

test('the weekly target only accepts 1 to 50', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  const { calls } = await mockQueue(page);
  await page.goto('/connections?tab=queue');
  const main = page.locator('main');
  await main.getByRole('button', { name: /Change the weekly target/ }).click();
  for (const value of ['0', '51']) {
    await main.getByRole('spinbutton', { name: 'Per week' }).fill(value);
    await main.getByRole('button', { name: 'Save', exact: true }).click();
  }
  await expect(main.getByRole('spinbutton', { name: 'Per week' })).toBeVisible();
  expect(calls.filter(c => c.path === '/settings')).toEqual([]);
});

test('a failed queue load shows an alert and keeps the rest of the page', async ({ page }) => {
  await mockPeople(page);
  await page.route('**/api/connections/queue', route => route.fulfill({ status: 500, json: { error: 'boom' } }));
  await page.goto('/connections');
  const main = page.locator('main');
  await expect(main.getByRole('alert')).toHaveText('The catch-up queue could not be loaded. Reload to retry.');
  await expect(main.getByRole('button', { name: /Paige North/ })).toBeVisible();
});

test('people due by cadence or flagged are listed under the board with their actions', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  const { saves } = await mockPeople(page);
  await page.goto('/connections?tab=queue');
  const main = page.locator('main');
  const due = main.getByRole('region', { name: 'Due for a catch-up' });
  const kayla = due.getByRole('listitem').filter({ hasText: 'Kayla Altepeter' });
  await expect(kayla).toContainText('Flagged · last talked never');

  await kayla.getByRole('button', { name: 'Caught up today' }).click();
  await expect(main.getByRole('status')).toHaveText('Logged a conversation with Kayla Altepeter.');
  const today = await page.evaluate(() => { const d = new Date(); return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`; });
  expect(saves.at(-1)).toMatchObject({ id: people[4].id, queued: false, lastContactedOn: today });
  // Caught up clears the flag, so she leaves the list.
  await expect(due).toHaveCount(0);
});

test('clearing the flag or queueing someone from the cadence list', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  const { saves } = await mockPeople(page);
  const queueCalls: { method: string; body: Record<string, unknown> }[] = [];
  await page.route('**/api/connections/queue', route => {
    if (route.request().method() === 'GET') return route.fallback();
    const body = route.request().postDataJSON();
    queueCalls.push({ method: route.request().method(), body });
    return route.fulfill({ json: { entry: { ...people.find(p => p.id === body.id), stage: body.stage }, revision: 'rev-queue' } });
  });
  await page.goto('/connections?tab=queue');
  const main = page.locator('main');
  const due = main.getByRole('region', { name: 'Due for a catch-up' });
  await due.getByRole('button', { name: 'Clear the flag for Kayla Altepeter' }).click();
  await expect(main.getByRole('status')).toHaveText('Kayla Altepeter is no longer flagged.');
  expect(saves.at(-1)).toMatchObject({ id: people[4].id, queued: false });
  await expect(due).toHaveCount(0);

  await page.reload();
  await main.getByRole('region', { name: 'Due for a catch-up' }).getByRole('button', { name: 'Add Kayla Altepeter to the queue' }).click();
  await expect(main.getByRole('status')).toHaveText('Kayla Altepeter added to the queue.');
  expect(queueCalls).toEqual([{ method: 'POST', body: { id: people[4].id, revision: `rev-${people[4].id}`, stage: 'to_contact' } }]);
  // On the board now, so the cadence list leaves her out.
  await expect(main.getByRole('region', { name: 'To contact' }).getByRole('button', { name: /Kayla Altepeter/ })).toBeVisible();
  await expect(main.getByRole('region', { name: 'Due for a catch-up' })).toHaveCount(0);
});

test('linking another goal to the catch-up strip unlinks the old one', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  const { goalSaves } = await mockQueue(page);
  await page.goto('/connections?tab=queue');
  const main = page.locator('main');
  await main.getByRole('button', { name: 'Change', exact: true }).click();
  await main.getByRole('combobox', { name: 'Linked goal' }).selectOption({ label: 'System design prep' });
  await expect(main.getByRole('status')).toHaveText('Linked System design prep to the catch-up queue.');
  expect(goalSaves.map(g => [g.id, g.metadata])).toEqual([['g1', {}], ['g2', { catchUpQueue: 'linked' }]]);
  await expect(main.getByText('This goal has no steps yet. Add them on the timeline.')).toBeVisible();
});

test('on a phone the catch-up queue groups people under stage headers', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockQueue(page);
  await page.goto('/connections?tab=queue');
  const main = page.locator('main');
  await expect(main.getByText('Warm up connections for Q1 referrals · step 2 of 4')).toBeVisible();
  await expect(main.getByText('0 / 2 this week')).toBeVisible();
  await expect(main.getByRole('heading', { name: 'Messaged' })).toBeVisible();
  // Nothing is picked until you tap someone; the panel then replaces the list.
  await expect(main.getByRole('complementary')).toHaveCount(0);
  await main.getByRole('button', { name: /Paige North/ }).click();
  await expect(main.getByRole('complementary', { name: 'Catch up with Paige North' })).toBeVisible();
  await expect(main.getByRole('heading', { name: 'Paige North' })).toBeFocused();
  await expect(main.getByRole('heading', { name: 'Messaged' })).toHaveCount(0);
  await main.getByRole('button', { name: '← CATCH-UP QUEUE' }).click();
  await expect(main.getByRole('button', { name: /Paige North/ })).toBeFocused();
});

test('people tab queue button adds someone to the stage queue', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  const { calls } = await mockQueue(page);
  await page.goto('/connections');
  const main = page.locator('main');
  await main.getByRole('button', { name: /Joey Brown/ }).click();
  await main.getByRole('complementary', { name: 'About Joey Brown' }).getByRole('button', { name: 'Queue' }).click();
  await expect(main.getByRole('status')).toHaveText('Joey Brown added to the queue.');
  expect(calls.at(-1)).toMatchObject({ method: 'POST', path: '/', body: { id: queuePeople[4].id, stage: 'to_contact' } });
  await expect(main.getByRole('complementary', { name: 'About Joey Brown' })).toContainText('In catch-up queue · To contact');
});
