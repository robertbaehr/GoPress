// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.62: E2E for edit.php search box (P7.69) + author/category filters.
 */
test('edit.php search box filters posts list', async ({ page, baseURL }) => {
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
  await expect(page.locator('#post-search-input')).toBeAttached();
  await expect(page.locator('#search-submit')).toBeAttached();
  await expect(page.locator('#filter-by-author')).toBeAttached();
  await expect(page.locator('#cat')).toBeAttached();

  await page.fill('#post-search-input', 'Hello world');
  await Promise.all([
    page.waitForURL(/[?&]s=/),
    page.click('#search-submit'),
  ]);
  await expect(page.locator('#post-search-input')).toHaveValue('Hello world');
  await expect(page.locator('#the-list')).toContainText('Hello world');

  await page.goto('/wp-admin/edit.php?s=zzznomatch-edit-search');
  await expect(page.locator('#post-search-input')).toHaveValue('zzznomatch-edit-search');
  await expect(page.locator('#the-list')).not.toContainText('Hello world');
});
