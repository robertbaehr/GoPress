// @ts-check
const { test, expect } = require('@playwright/test');

test('login form posts and reaches wp-admin', async ({ page }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';

  await page.goto('/wp-login.php');
  await expect(page.locator('#loginform')).toBeVisible();
  await page.fill('#user_login', user);
  await page.fill('#user_pass', pass);
  await Promise.all([
    page.waitForURL(/\/wp-admin/),
    page.click('#wp-submit'),
  ]);
  await expect(page.locator('body')).toBeVisible();
  // Dashboard stub (or full chrome) should mention Dashboard.
  await expect(page.locator('html')).toContainText(/Dashboard/i);
});
