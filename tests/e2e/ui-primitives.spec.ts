import { test, expect } from '@playwright/test';

// Drawing primitives on the dev-only gallery (/dev/ui): keyboard and screen-reader behavior
// from docs/redesign/design-system.md (Accessibility), and values from the reference .bx/.dim/.lbl/.btn.

test.beforeEach(async ({ page }) => {
  await page.goto('/dev/ui');
  await expect(page.getByRole('heading', { name: 'Drawing primitives' })).toBeVisible();
});

test('switch is a real role=switch button toggled by Space, Enter and click', async ({ page }) => {
  const leet = page.getByRole('switch', { name: 'Leetgrinder' });
  await expect(leet).toHaveAttribute('aria-checked', 'false');
  await leet.focus();
  await page.keyboard.press('Space');
  await expect(leet).toHaveAttribute('aria-checked', 'true');
  await page.keyboard.press('Enter');
  await expect(leet).toHaveAttribute('aria-checked', 'false');
  await leet.click();
  await expect(leet).toHaveAttribute('aria-checked', 'true');
  await expect(page.getByRole('switch', { name: 'Locked on' })).toBeDisabled();

  const hit = await leet.boundingBox();
  const track = await leet.locator('.ui-switch__track').boundingBox();
  expect(hit!.height).toBeGreaterThanOrEqual(44);
  expect([track!.width, track!.height]).toEqual([42, 22]);
  await expect(page.getByRole('switch', { name: 'Locked off' }).locator('.ui-switch__track')).toHaveCSS('border-top-style', 'dashed');
});

test('checkbox is native: Space and its label toggle it, the ring marks the next item', async ({ page }) => {
  const ddia = page.getByRole('checkbox', { name: 'DDIA chapter 5' });
  const ring = page.locator('label .ui-check__ring');
  await expect(ring.locator('xpath=ancestor::label')).toHaveText('DDIA chapter 5');
  await ddia.focus();
  await page.keyboard.press('Space');
  await expect(ddia).toBeChecked();
  await expect(ring.locator('xpath=ancestor::label')).toHaveText('Alloy model check');
  await page.getByText('DDIA chapter 5').click();
  await expect(ddia).not.toBeChecked();
  const hit = await ddia.boundingBox();
  expect([hit!.width, hit!.height]).toEqual([44, 44]);
  await expect(page.getByRole('checkbox', { name: 'Checked', exact: true }).locator('~ .ui-check__x')).toBeVisible();
  await expect(page.getByRole('checkbox', { name: 'Unchecked', exact: true }).locator('~ .ui-check__x')).toBeHidden();
});

test('segmented tabs follow the tablist keyboard pattern', async ({ page }) => {
  const tabs = page.getByRole('tablist', { name: 'Priority' });
  const tab = (name: string) => tabs.getByRole('tab', { name });
  await expect(tabs.locator('[tabindex="0"]')).toHaveCount(1);
  await tab('All 61').focus();
  await page.keyboard.press('ArrowRight');
  await expect(tab('High')).toHaveAttribute('aria-selected', 'true');
  await expect(tab('High')).toBeFocused();
  await expect(page.getByRole('tabpanel')).toHaveText('Showing high priority companies.');
  await page.keyboard.press('End');
  await expect(tab('Low')).toHaveAttribute('aria-selected', 'true');
  await page.keyboard.press('ArrowRight');
  await expect(tab('All 61')).toHaveAttribute('aria-selected', 'true');
  await page.keyboard.press('ArrowLeft');
  await expect(tab('Low')).toBeFocused();
  await page.keyboard.press('Home');
  await expect(tab('All 61')).toHaveAttribute('aria-selected', 'true');
  await expect(tab('All 61')).toHaveAttribute('aria-controls', 'ui-tab-panel');
});

test('search field is a labelled searchbox; icon buttons have names', async ({ page }) => {
  const search = page.getByRole('searchbox', { name: 'Search companies' });
  await page.locator('.ui-search').first().click({ position: { x: 4, y: 4 } });
  await expect(search).toBeFocused();
  await page.keyboard.type('ramp');
  await expect(search).toHaveValue('ramp');
  const icons = page.locator('.ui-btn--icon');
  await expect(icons).toHaveCount(4);
  for (const [i, name] of ['Add', 'Settings', 'Edit', 'Delete'].entries()) await expect(icons.nth(i)).toHaveAccessibleName(name);
});

test('primitive values match the reference screens', async ({ page }) => {
  const style = (selector: string, pseudo?: string) => page.evaluate(([s, p]) => {
    const cs = getComputedStyle(document.querySelector(s!)!, p ?? null);
    return { left: cs.left, right: cs.right, top: cs.top, bottom: cs.bottom, height: cs.height, fontSize: cs.fontSize, fontWeight: cs.fontWeight, letterSpacing: cs.letterSpacing, textTransform: cs.textTransform, transform: cs.transform };
  }, [selector, pseudo]);
  expect(await style('.ui-box:not(.ui-box--sm)', '::before')).toMatchObject({ left: '-3px', right: '-3px' });
  expect(await style('.ui-box:not(.ui-box--sm)', '::after')).toMatchObject({ top: '-3px', bottom: '-3px' });
  expect(await style('.ui-box--sm', '::before')).toMatchObject({ left: '-2px', right: '-2px' });
  expect(await style('.ui-dim', '::before')).toMatchObject({ top: '-5px', height: '10px', transform: 'matrix(1, 0, -1, 1, 0, 0)' });
  expect(await style('.ui-label')).toMatchObject({ fontSize: '10px', fontWeight: '500', letterSpacing: '1.6px', textTransform: 'uppercase' });
  expect(await style('.ui-btn')).toMatchObject({ height: '44px', fontSize: '11px', fontWeight: '500', letterSpacing: '1.76px', textTransform: 'uppercase' });
});

test('the theme picker previews a theme without saving it', async ({ page }) => {
  await page.getByRole('tablist', { name: 'Theme' }).getByRole('tab', { name: 'cobalt' }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'cobalt');
  expect(await page.evaluate(() => localStorage.getItem('career-theme'))).toBeNull();
  // Labels keep the per-theme contrast alpha.
  await expect(page.locator('.ui-label').first()).toHaveCSS('color', 'rgba(0, 71, 171, 0.78)');
});
