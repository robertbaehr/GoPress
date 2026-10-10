// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.60: E2E for edit.php Bulk Edit stub (P7.68) + search box (P7.69).
 */
test('edit.php exposes Bulk Edit stub and search box', async ({ page, baseURL }) => {
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

  // Search box (P7.69).
  await expect(page.locator('#post-search-input')).toBeAttached();
  await expect(page.locator('#search-submit')).toBeAttached();
  await expect(page.locator('.search-box')).toContainText(/Search Posts/i);

  // Bulk actions + row checkboxes (P7.68).
  await expect(page.locator('#posts-filter')).toBeAttached();
  await expect(page.locator('#bulk-action-selector-top')).toBeAttached();
  await expect(page.locator('option[value="edit"]')).toBeAttached();
  await expect(page.locator('input[name="post[]"]').first()).toBeAttached();

  // Hidden bulk-edit template.
  await expect(page.locator('#bulk-edit')).toBeAttached();
  await expect(page.locator('#bulk-edit-legend')).toContainText('Bulk Edit');
  await expect(page.locator('#bulk-titles')).toBeAttached();
  await expect(page.locator('input[name="bulk_edit"]')).toBeAttached();
  await expect(page.locator('#bulk-edit select[name="_status"]')).toBeAttached();
  await expect(page.locator('#bulk-edit select[name="_status"] option[value="-1"]')).toContainText(/No Change/i);
});
