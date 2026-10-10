// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.58: E2E for edit.php Quick Edit stub (P7.66 / P7.67).
 */
test('edit.php exposes Quick Edit stub and sticky toggle', async ({ page, baseURL }) => {
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
  await expect(page.locator('#the-list')).toBeVisible();

  // Row Quick Edit control (P7.66). CSS hides .row-actions until hover.
  const quickEdit = page.locator('button.editinline').first();
  await expect(quickEdit).toBeAttached();
  await expect(quickEdit).toContainText(/Quick/);

  // Hidden inline-edit template + sticky checkbox (P7.67).
  await expect(page.locator('#inlineedit')).toBeAttached();
  await expect(page.locator('#inline-edit')).toBeAttached();
  await expect(page.locator('#quick-edit-legend')).toContainText('Quick Edit');
  await expect(page.locator('input[name="post_title"]')).toBeAttached();
  await expect(page.locator('select[name="_status"]')).toBeAttached();
  await expect(page.locator('input[name="sticky"][value="sticky"]')).toBeAttached();
  await expect(page.locator('.checkbox-title')).toContainText(/Make this post sticky/i);

  // Per-row inline data stubs exist.
  const inlineData = page.locator('[id^="inline_"]');
  await expect(inlineData.first()).toBeAttached();
  await expect(inlineData.first().locator('.post_title')).toBeAttached();
  await expect(inlineData.first().locator('.sticky')).toBeAttached();
});
