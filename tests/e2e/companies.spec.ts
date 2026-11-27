import { expect, test, type Page } from '@playwright/test';

// The Companies page (docs/redesign/pages/companies.md) against mocked companies, plus the
// edit and detail flows against the real API.
const common = { category: 'Observability / Infra', type: 'company', url: 'https://example.com/careers', status: 'not_started', featured: false, priority: 'high', tags: [], body: '' };
const companies = [
  { ...common, slug: 'keep-co', title: 'Keep Co', url: 'https://www.keep.example/jobs', featured: true, body: '## Why\n\nStrong tracing team.\n\n## Steps\n\n- [x] Research\n- [ ] Apply\n' },
  { ...common, slug: 'drop-co', title: 'Drop Co', url: 'https://drop.example', priority: 'low', category: 'Databases', tags: ['remote'] },
];
// A 16×16 opaque black PNG standing in for GET /api/companies/logo (black pixels, alpha = logo).
const LOGO_PNG = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAAFElEQVR4nGJiIBGMahjVMIw1AAYANyAAITTSrPQAAAAASUVORK5CYII=', 'base64');

async function mockCompanies(page: Page) {
  const deletes: unknown[] = [];
  const logos: string[] = [];
  await page.route('**/api/companies/logo?**', route => {
    const domain = new URL(route.request().url()).searchParams.get('domain')!;
    logos.push(domain);
    return domain === 'keep.example'
      ? route.fulfill({ status: 200, contentType: 'image/png', body: LOGO_PNG })
      : route.fulfill({ status: 404, contentType: 'application/json', body: '{"error":"No logo for this domain"}' });
  });
  await page.route('**/api/entries/company**', route => {
    const request = route.request();
    if (request.method() === 'DELETE') {
      deletes.push(request.postDataJSON());
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ deleted: true }) });
    }
    const id = new URL(request.url()).searchParams.get('id');
    const entry = companies.find(company => company.slug === id);
    const body = entry ? { entry, revision: `rev-${entry.slug}` } : { entries: companies.map(company => ({ entry: company, revision: `rev-${company.slug}` })), nextOffset: null };
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
  });
  return { deletes, logos };
}

test('a company can be deleted from its card on the companies page', async ({ page }) => {
  const { deletes } = await mockCompanies(page);
  await page.goto('/companies');
  const main = page.locator('main');
  await expect(main.getByRole('link', { name: 'Edit Drop Co' })).toHaveAttribute('href', '/manage/companies?id=drop-co');

  page.once('dialog', dialog => dialog.accept());
  await main.getByRole('button', { name: 'Delete Drop Co' }).click();

  await expect(main.getByRole('status')).toHaveText('Drop Co deleted.');
  await expect(main.getByRole('link', { name: 'Drop Co', exact: true })).toHaveCount(0);
  await expect(main.getByRole('link', { name: 'Keep Co', exact: true })).toBeVisible();
  await expect(main.getByText('1 tracked')).toBeVisible();
  expect(deletes).toEqual([{ id: 'drop-co', revision: 'rev-drop-co' }]);
});

test('cancelling the confirmation keeps the company', async ({ page }) => {
  const { deletes } = await mockCompanies(page);
  await page.goto('/companies');
  const main = page.locator('main');

  page.once('dialog', dialog => dialog.dismiss());
  await main.getByRole('button', { name: 'Delete Drop Co' }).click();

  await expect(main.getByRole('link', { name: 'Drop Co', exact: true })).toBeVisible();
  expect(deletes).toEqual([]);
});

test('cards show the header counts, priority, featured mark and checklist progress', async ({ page }) => {
  await mockCompanies(page);
  await page.goto('/companies');
  const main = page.locator('main');
  await expect(main.getByText('2 tracked · 1 featured')).toBeVisible();
  await expect(main.getByRole('heading', { level: 1, name: 'Companies' })).toBeVisible();
  const keep = main.getByRole('listitem').filter({ hasText: 'Keep Co' });
  await expect(keep).toContainText('Strong tracing team.');
  await expect(keep.getByTitle('Featured')).toBeVisible();
  await expect(keep.getByTitle('High priority')).toBeVisible();
  await expect(keep.getByRole('img', { name: '1 of 2 steps done' })).toBeVisible();
  await expect(main.getByRole('listitem').filter({ hasText: 'Drop Co' }).getByTitle('Low priority')).toBeVisible();
  await expect(main.getByRole('link', { name: '+ Add company' })).toHaveAttribute('href', '/manage/companies?new=1');
  await expect(main.getByRole('link', { name: 'Manage' })).toHaveAttribute('href', '/manage/companies');
  await expect(main.getByRole('link', { name: 'Keep Co', exact: true })).toHaveAttribute('href', '/companies/keep-co');
});

test('logos come from the company domain in ink, with a monogram when there is none', async ({ page }) => {
  const { logos } = await mockCompanies(page);
  await page.goto('/companies');
  const main = page.locator('main');
  const keep = main.getByRole('listitem').filter({ hasText: 'Keep Co' }).locator('.ui-logo');
  const drop = main.getByRole('listitem').filter({ hasText: 'Drop Co' }).locator('.ui-logo');
  await expect(keep).toHaveAttribute('data-logo-state', 'logo');
  const mark = await keep.locator('.ui-logo__mark').evaluate(el => ({ color: getComputedStyle(el).backgroundColor, mask: getComputedStyle(el).maskImage || getComputedStyle(el).webkitMaskImage }));
  const ink = await page.evaluate(() => { const probe = document.createElement('span'); probe.style.color = 'var(--ink)'; document.body.append(probe); const color = getComputedStyle(probe).color; probe.remove(); return color; });
  expect(mark.color).toBe(ink);
  expect(mark.mask).toContain('/api/companies/logo?domain=keep.example');
  await expect(drop).toHaveAttribute('data-logo-state', 'failed');
  await expect(drop).toHaveText('DC');
  // www. is dropped so a company's logo is cached once per domain.
  expect(logos.sort()).toEqual(['drop.example', 'keep.example']);
});

test('search matches name, category, tags and why text, and combines with the priority filter', async ({ page }) => {
  await mockCompanies(page);
  await page.goto('/companies');
  const main = page.locator('main');
  const search = main.getByRole('searchbox', { name: 'Search companies' });
  const names = () => main.locator('#company-list > li').locator('a[href^="/companies/"]').allTextContents();
  await expect.poll(names).toEqual(['Keep Co', 'Drop Co']);
  for (const [query, expected] of [['drop', ['Drop Co']], ['DATABASES', ['Drop Co']], ['remote', ['Drop Co']], ['tracing team', ['Keep Co']], ['co', ['Keep Co', 'Drop Co']]] as const) {
    await search.fill(query);
    await expect.poll(names, { message: query }).toEqual(expected);
  }
  await search.fill('nothing like this');
  await expect(main.getByText('No companies match "nothing like this".')).toBeVisible();
  await expect(main.getByText('0 shown')).toBeVisible();

  await search.fill('co');
  await main.getByRole('tab', { name: 'Low' }).click();
  await expect.poll(names).toEqual(['Drop Co']);
  await expect(main.getByText('1 shown')).toBeVisible();
  await main.getByRole('tab', { name: 'Low' }).press('Home');
  await expect(main.getByRole('tab', { name: 'All 2' })).toHaveAttribute('aria-selected', 'true');
  await expect.poll(names).toEqual(['Keep Co', 'Drop Co']);
});

test('phones get rows with a + button and 44px edit and delete buttons', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockCompanies(page);
  await page.goto('/companies');
  const main = page.locator('main');
  await expect(main.getByRole('link', { name: 'Add company' })).toHaveAttribute('href', '/manage/companies?new=1');
  await expect(main.getByRole('link', { name: '+ Add company' })).toBeHidden();
  await expect(main.getByRole('tab', { name: 'Med' })).toBeVisible();
  for (const control of [main.getByRole('link', { name: 'Edit Keep Co' }), main.getByRole('button', { name: 'Delete Keep Co' })]) {
    const box = await control.boundingBox();
    expect(box?.height).toBe(44);
    expect(box?.width).toBe(44);
  }
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
});

test('+ Add company opens a blank company in the editor', async ({ page }) => {
  await page.goto('/manage/companies?new=1');
  await expect(page.getByLabel('Title')).toHaveValue('');
  await expect(page.getByLabel('Title')).toBeFocused();
  await expect(page).toHaveURL('/manage/companies');
});

async function createCompany(page: Page, baseURL: string, name: string) {
  const slug = `e2e-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  // Unique titles keep retries and reruns against a reused workspace from colliding.
  const title = `${name} ${slug}`;
  const entry = { ...common, slug, title, body: '## Why\n\nE2E fixture.\n' };
  const response = await page.request.post('/api/entries/company', { headers: { Origin: baseURL }, data: { entry, revision: null } });
  expect(response.ok()).toBe(true);
  return { slug, title, revision: (await response.json()).revision as string };
}

test('the edit link opens that company in the editor', async ({ page, baseURL }) => {
  const { slug, title, revision } = await createCompany(page, baseURL!, 'Editable Co');
  try {
    await page.goto('/companies');
    await page.getByRole('link', { name: `Edit ${title}` }).click();

    await expect(page).toHaveURL(`/manage/companies?id=${slug}`);
    await expect(page.getByRole('heading', { name: title })).toBeVisible();
  } finally {
    await page.request.delete('/api/entries/company', { headers: { Origin: baseURL! }, data: { id: slug, revision } });
  }
});

test('deleting from the company detail page returns to the companies list', async ({ page, baseURL }) => {
  const { slug, title } = await createCompany(page, baseURL!, 'Detail Co');
  await page.goto(`/companies/${slug}`);

  page.once('dialog', dialog => dialog.accept());
  await page.getByRole('button', { name: `Delete ${title}` }).click();

  await expect(page).toHaveURL('/companies');
  expect((await page.request.get(`/api/entries/company?id=${slug}`)).status()).toBe(404);
});
