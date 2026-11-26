import { expect, test, type Page } from '@playwright/test';

// The People page (docs/redesign/pages/people.md) against mocked connections and companies.

const person = (id: string, name: string, role: string, companyName: string, extra: Record<string, unknown> = {}) =>
  ({ id: `00000000-0000-4000-8000-${id.padStart(12, '0')}`, name, role, companyName, email: '', queued: false, notes: '', tags: [], ...extra });

const people = [
  person('1', 'Paige North', 'Senior Software Engineer', 'Microsoft', { companySlug: 'microsoft', lastContactedOn: '2026-09-11', connectedOn: '2024-02-01', url: 'https://www.linkedin.com/in/paige', notes: 'Met at a meetup' }),
  person('2', 'Brian Holt', 'Principal TPM', 'Microsoft', { companySlug: 'microsoft' }),
  person('3', 'Adam Iverson', 'Software Engineer II', 'Maximus Inc'),
  person('4', 'Alex Stone', 'Software Engineer', 'maximus'),
  person('5', 'Kayla Altepeter', 'Staff Engineer', 'Datasite', { queued: true }),
];
const companies = [{ slug: 'microsoft', title: 'Microsoft', category: 'Big tech', type: 'company', url: 'https://microsoft.com', status: 'not_started', featured: false, priority: 'medium', tags: [], body: '' }];

async function mockPeople(page: Page, { companiesOn = true } = {}) {
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
  await page.route('**/api/connections', route => route.fulfill({ json: { connections: people, revisions: Object.fromEntries(people.map(p => [p.id, `rev-${p.id}`])) } }));
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
  await expect(main.getByRole('tab', { name: 'Catch-up queue · 1' })).toBeVisible();

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
  await expect(main.getByRole('heading', { name: 'Catch-up queue' })).toBeVisible();
  await expect(main.getByRole('listitem').filter({ hasText: 'Kayla Altepeter' })).toBeVisible();
  await page.reload();
  await expect(main.getByRole('tab', { name: /Catch-up queue/ })).toHaveAttribute('aria-selected', 'true');
});

test('the catch-up tab is hidden with a note when Companies is off', async ({ page }) => {
  await mockPeople(page, { companiesOn: false });
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
});

test('on a phone the detail replaces the list until you go back', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockPeople(page);
  await page.goto('/connections');
  const main = page.locator('main');
  await expect(main.getByRole('heading', { level: 1, name: 'People', exact: true })).toBeVisible();
  await expect(main.getByRole('tab', { name: 'Catch-up · 1' })).toBeVisible();
  await main.getByRole('button', { name: /Paige North/ }).click();
  await expect(main.getByRole('complementary', { name: 'About Paige North' })).toBeVisible();
  await expect(main.getByRole('button', { name: /Brian Holt/ })).toBeHidden();
  await main.getByRole('button', { name: '← ALL PEOPLE' }).click();
  await expect(main.getByRole('button', { name: /Brian Holt/ })).toBeVisible();
});
