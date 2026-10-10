// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.40: E2E for wp-login.php?reauth=1 from P6.18.
 * Logged-in users hitting reauth=1 must see the login form (cookies cleared).
 */
test('reauth=1 forces login form for authenticated user', async ({ page, context, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = baseURL || 'http://127.0.0.1:8082';

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
  await expect(page.locator('body')).toBeVisible();

  // Without reauth, visiting login while authenticated redirects to admin.
  await page.goto('/wp-login.php?redirect_to=%2Fwp-admin%2F');
  await expect(page).toHaveURL(/\/wp-admin(\/|$|\?)/);

  // reauth=1 forces the login form again.
  await page.goto('/wp-login.php?reauth=1&redirect_to=%2Fwp-admin%2F');
  await expect(page.locator('#loginform')).toBeVisible();
  await expect(page).toHaveURL(/reauth=1/);

  const cookies = await context.cookies();
  const loggedIn = cookies.filter((c) => c.name.startsWith('wordpress_logged_in_'));
  for (const c of loggedIn) {
    expect(c.value === '' || c.value === ' ' || c.expires === 0 || c.expires < Date.now() / 1000).toBeTruthy();
  }
});
