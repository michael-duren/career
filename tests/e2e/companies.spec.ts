import { expect, test, type Page } from '@playwright/test';

const common = { category: 'Observability / Infra', type: 'company', url: 'https://example.com/careers', status: 'not_started', featured: false, priority: 'high', tags: [], body: '' };
const companies = [
  { ...common, slug: 'keep-co', title: 'Keep Co' },
  { ...common, slug: 'drop-co', title: 'Drop Co' },
];

async function mockCompanies(page: Page) {
  const deletes: unknown[] = [];
  await page.route('**/api/connections?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ connections: [] }) }));
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
  return deletes;
}

test('a company can be deleted from its card on the companies page', async ({ page }) => {
  const deletes = await mockCompanies(page);
  await page.goto('/companies');
  await expect(page.getByRole('link', { name: 'Edit Drop Co' })).toHaveAttribute('href', '/manage/companies?id=drop-co');

  page.once('dialog', dialog => dialog.accept());
  await page.getByRole('button', { name: 'Delete Drop Co' }).click();

  await expect(page.getByRole('status')).toHaveText('Drop Co deleted.');
  await expect(page.getByRole('link', { name: 'Drop Co', exact: true })).toHaveCount(0);
  await expect(page.getByRole('link', { name: 'Keep Co', exact: true })).toBeVisible();
  expect(deletes).toEqual([{ id: 'drop-co', revision: 'rev-drop-co' }]);
});

test('cancelling the confirmation keeps the company', async ({ page }) => {
  const deletes = await mockCompanies(page);
  await page.goto('/companies');

  page.once('dialog', dialog => dialog.dismiss());
  await page.getByRole('button', { name: 'Delete Drop Co' }).click();

  await expect(page.getByRole('link', { name: 'Drop Co', exact: true })).toBeVisible();
  expect(deletes).toEqual([]);
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
