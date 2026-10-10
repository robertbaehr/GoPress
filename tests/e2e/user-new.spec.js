// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.56: E2E for user-new.php createuser (P7.64).
 */
test('admin creates a user via user-new.php', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');
  const stamp = Date.now();
  const username = `e2e_new_${stamp}`;
  const email = `${username}@example.com`;

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

  await page.goto('/wp-admin/user-new.php');
  await expect(page.locator('#createuser')).toBeVisible();
  await expect(page.locator('input[name="action"]')).toHaveAttribute('value', 'createuser');

  await page.fill('#user_login', username);
  await page.fill('#email', email);
  await page.fill('#first_name', 'E2E');
  await page.fill('#last_name', 'NewUser');
  const passField = page.locator('#pass1');
  if (await passField.count()) {
    await passField.fill('E2eNewUserPass!234');
  } else {
    await page.fill('#password', 'E2eNewUserPass!234');
  }
  const role = page.locator('#role');
  if (await role.count()) {
    await role.selectOption('author');
  }

  await Promise.all([
    page.waitForURL(/\/wp-admin\/users\.php/),
    page.click('#createusersub'),
  ]);

  expect(page.url()).toMatch(/users\.php/);
  expect(page.url()).toMatch(/update=add/);
  await expect(page.locator('#message')).toContainText(/New user created/i);
  await expect(page.locator('body')).toContainText(username);
});
