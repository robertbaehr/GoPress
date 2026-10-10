// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.64: E2E for edit.php author filter dropdown (P7.70).
 */
test('edit.php author filter dropdown filters posts list', async ({ page, baseURL }) => {
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
  await expect(page.locator('#filter-by-author')).toBeAttached();
  await expect(page.locator('#post-query-submit')).toBeAttached();
  await expect(page.locator('#filter-by-author')).toContainText('All authors');
  await expect(page.locator('#filter-by-author')).toContainText('Admin User');

  await page.selectOption('#filter-by-author', '1');
  await Promise.all([
    page.waitForURL(/[?&]author=1/),
    page.click('#post-query-submit'),
  ]);
  await expect(page.locator('#filter-by-author')).toHaveValue('1');
  await expect(page.locator('#the-list')).toContainText('Hello world');

  await page.goto('/wp-admin/edit.php?author=99999');
  await expect(page.locator('#the-list')).not.toContainText('Hello world');
});
