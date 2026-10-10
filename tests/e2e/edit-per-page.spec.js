// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.70: E2E for edit.php Screen Options posts_per_page (P7.77).
 */
test('edit.php Screen Options per page limits the posts list', async ({ page, baseURL }) => {
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
  await expect(page.locator('#edit_post_per_page')).toBeAttached();
  await expect(page.locator('#screen-options-apply')).toBeAttached();

  await page.fill('#edit_post_per_page', '3');
  await Promise.all([
    page.waitForURL(/\/wp-admin\/edit\.php/),
    page.click('#screen-options-apply'),
  ]);

  await expect(page.locator('#edit_post_per_page')).toHaveValue('3');
  const rows = page.locator('#the-list > tr');
  await expect(rows.first()).toBeVisible();
  const count = await rows.count();
  expect(count).toBeLessThanOrEqual(3);
  expect(count).toBeGreaterThan(0);
});
