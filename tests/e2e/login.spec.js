// @ts-check
const { test, expect } = require('@playwright/test');

test('login form posts and reaches wp-admin', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';

  await page.goto('/wp-login.php');
  await expect(page.locator('#loginform')).toBeVisible();

  // Fixture DBs often store siteurl as the PHP oracle (:8081). Force the form
  // to post back to this GoPress origin so the smoke stays host-local.
  await page.evaluate((origin) => {
    const form = document.querySelector('#loginform');
    if (form) {
      form.setAttribute('action', origin + '/wp-login.php');
    }
  }, baseURL || 'http://127.0.0.1:8082');

  await page.fill('#user_login', user);
  await page.fill('#user_pass', pass);
  await Promise.all([
    page.waitForURL(/\/wp-admin(\/|$|\?)/),
    page.click('#wp-submit'),
  ]);
  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('html')).toContainText(/Dashboard/i);
});
