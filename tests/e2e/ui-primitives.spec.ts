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
  const unchecked = page.getByRole('checkbox', { name: 'Unchecked', exact: true }).locator('~ .ui-check__x');
  await expect(unchecked).toHaveCount(1);
  await expect(unchecked).toBeHidden();
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

test('primitive values are pinned to the reference screens (.bx, .dim, .lbl, .btn)', async ({ page }) => {
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

const INK = 'rgb(236, 234, 227)', PAPER = 'rgb(12, 12, 11)', ACCENT = 'rgb(224, 83, 61)';

test('the danger button keeps its accent on hover', async ({ page }) => {
  const remove = page.getByRole('button', { name: 'Delete', exact: true }).first();
  await remove.hover();
  await expect(remove).toHaveCSS('color', ACCENT);
  await expect(remove).toHaveCSS('border-top-color', ACCENT);
  const cancel = page.getByRole('button', { name: 'Cancel' });
  await cancel.hover();
  await expect(cancel).toHaveCSS('border-top-color', INK);
});

test('logo spot shows the ink-masked logo once loaded, else a monogram', async ({ page }) => {
  const logo = page.getByRole('img', { name: 'Delta logo' });
  await expect(logo).toHaveAttribute('data-logo-state', 'logo');
  await expect(logo).toHaveText('');
  const mark = await logo.locator('.ui-logo__mark').evaluate(el => {
    const cs = getComputedStyle(el);
    return { mask: cs.maskImage || cs.webkitMaskImage, color: cs.backgroundColor };
  });
  expect(mark.mask).toMatch(/^url\("data:image\/svg\+xml/);
  expect(mark.color).toBe(INK); // the orange sample logo is painted in ink
  const spots = page.locator('.ui-logo[aria-hidden="true"]');
  await expect(spots).toHaveText(['JS', 'RA', 'BL', 'MU', 'DA']);
  // A logo that fails to load and a malformed URL both keep the monogram.
  await expect(spots.nth(2)).toHaveAttribute('data-logo-state', 'failed');
  await expect(spots.nth(3)).toHaveAttribute('data-logo-state', 'monogram');
  await expect(spots.nth(0)).toHaveAttribute('data-logo-state', 'monogram');
  await expect(page.locator('.ui-logo .ui-logo__mark')).toHaveCount(1);
});

// The app-wide :focus-visible ring is unlayered; the primitives' own focus rules must still win.
test('primitive focus rings replace the app-wide ring', async ({ page }) => {
  const outline = (locator: import('@playwright/test').Locator) => locator.evaluate(el => {
    const cs = getComputedStyle(el);
    return { style: cs.outlineStyle, width: cs.outlineWidth, offset: cs.outlineOffset, color: cs.outlineColor };
  });

  await page.getByRole('switch', { name: 'Timeline' }).focus();
  await page.keyboard.press('Tab');
  const leet = page.getByRole('switch', { name: 'Leetgrinder' });
  await expect(leet).toBeFocused();
  expect((await outline(leet)).style).toBe('none');
  expect(await outline(leet.locator('.ui-switch__track'))).toEqual({ style: 'solid', width: '2px', offset: '3px', color: INK });

  const tabs = page.getByRole('tablist', { name: 'Priority' });
  await tabs.getByRole('tab', { name: 'All 61' }).focus();
  await page.keyboard.press('ArrowRight');
  // The selected tab is solid ink, so its ring is drawn in paper.
  expect(await outline(tabs.getByRole('tab', { name: 'High' }))).toEqual({ style: 'solid', width: '2px', offset: '-5px', color: PAPER });

  const search = page.getByRole('searchbox', { name: 'Search companies' });
  await search.click();
  expect((await outline(search)).style).toBe('none');
  expect(await outline(page.locator('.ui-search').first())).toEqual({ style: 'solid', width: '2px', offset: '2px', color: INK });
});

// Patterns and ticks carry meaning, so they survive forced-colors mode.
test('ticks, the switch track and the selected tab keep their state in forced colors', async ({ page }) => {
  await page.emulateMedia({ forcedColors: 'active' });
  const style = (selector: string, prop: 'backgroundColor' | 'backgroundImage' | 'textDecorationLine') =>
    page.locator(selector).first().evaluate((el, p) => getComputedStyle(el)[p], prop);
  const on = await style('.ui-ticks__tick--on', 'backgroundColor');
  const off = await style('.ui-ticks__tick:not(.ui-ticks__tick--on)', 'backgroundColor');
  expect(on).not.toBe(off);
  expect(on).not.toBe('rgba(0, 0, 0, 0)');
  expect(await style('.ui-switch[aria-checked="true"] .ui-switch__knob', 'backgroundColor'))
    .not.toBe(await style('.ui-switch[aria-checked="false"] .ui-switch__knob', 'backgroundColor'));
  expect(await style('.ui-tabs__tab[aria-selected="true"]', 'textDecorationLine')).toBe('underline');
  for (const selector of ['.ui-hatch--diagonal', '.ui-hatch--dots', '.ui-switch[aria-checked="true"] .ui-switch__track']) {
    expect(await style(selector, 'backgroundImage'), selector).not.toBe('none');
  }
});

test('progress is a progressbar with its value', async ({ page }) => {
  await expect(page.getByRole('progressbar', { name: 'Progress 40%' })).toHaveAttribute('aria-valuenow', '40');
  await expect(page.getByRole('progressbar', { name: 'Progress Done' })).toHaveAttribute('aria-valuenow', '100');
});

test('modified arrow keys are left to the browser', async ({ page }) => {
  const tabs = page.getByRole('tablist', { name: 'Priority' });
  await tabs.getByRole('tab', { name: 'All 61' }).focus();
  await page.keyboard.press('Control+End');
  await expect(tabs.getByRole('tab', { name: 'All 61' })).toHaveAttribute('aria-selected', 'true');
});
