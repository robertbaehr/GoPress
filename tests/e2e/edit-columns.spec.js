// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.72: E2E for edit.php Screen Options Columns (P7.80).
 */
test('edit.php Screen Options columns hide Author column', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  await page.goto('/wp-login.php');
  await expect(page.locator('#loginform')).toBeVisible();
  await page.evaluate((o) => {
    const form = document.querySelector('#loginform');
    if (form) form.setAttribute('action', o + '/wp-login.php');
  }, origin);
  await page.fill('#user_login', user);
  await page.fill('#user_pass', pass);
  await Promise.all([
    page.waitForURL(/\/wp-admin(\/|$|\?)/),
    page.click('#wp-submit'),
  ]);

  await page.goto('/wp-admin/edit.php');
  await expect(page.locator('h1')).toContainText('Posts');
  await expect(page.locator('#screen-options-wrap')).toBeAttached();
  await expect(page.locator('#author-hide')).toBeAttached();
  await expect(page.locator('#date-hide')).toBeAttached();
  await expect(page.locator('legend', { hasText: 'Columns' })).toBeAttached();

  // Uncheck Author (force: Screen Options panel may be CSS-hidden).
  await page.locator('#author-hide').uncheck({ force: true });
  await expect(page.locator('#date-hide')).toBeChecked({ checked: true });

  await Promise.all([
    page.waitForNavigation({ waitUntil: 'load' }),
    page.locator('#adv-settings').evaluate((form) => {
      /** @type {HTMLFormElement} */ (form).requestSubmit();
    }),
  ]);

  await expect(page.locator('#author-hide')).not.toBeChecked();
  // Author column header should carry hidden class after prefs apply.
  const authorTh = page.locator('th.column-author');
  await expect(authorTh).toBeAttached();
  await expect(authorTh).toHaveClass(/hidden/);
  // Date column remains visible (no hidden class).
  await expect(page.locator('th.column-date')).not.toHaveClass(/hidden/);
});
