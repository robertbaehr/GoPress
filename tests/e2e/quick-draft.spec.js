// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.48: E2E for Dashboard Quick Draft (P7.55).
 */
test('dashboard Quick Draft creates a draft post', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');
  const stamp = Date.now();
  const title = `E2E Quick Draft ${stamp}`;
  const content = `Quick draft body ${stamp}`;

  await page.goto('/wp-login.php');
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

  await page.goto('/wp-admin/');
  await expect(page.locator('#dashboard_quick_press')).toBeVisible();
  await expect(page.locator('#quick-press')).toBeVisible();
  await expect(page.locator('input[name="action"]')).toHaveAttribute(
    'value',
    'post-quickdraft-save',
  );

  await page.fill('#title', title);
  await page.fill('#content', content);
  await Promise.all([
    page.waitForURL(/\/wp-admin(\/|$|\?)/),
    page.click('#save-post'),
  ]);

  await expect(page.locator('.notice-success')).toContainText(/Draft created successfully/i);
  await expect(page.locator('.notice-success a')).toHaveAttribute('href', /post\.php\?post=\d+/);

  // Open the edit link and confirm the draft title is present.
  await page.locator('.notice-success a').first().click();
  await expect(page.locator('#title')).toHaveValue(title);
});

test('dashboard Quick Draft rejects empty title and content', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  await page.goto('/wp-login.php');
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

  await page.goto('/wp-admin/');
  await page.fill('#title', '');
  await page.fill('#content', '');
  await Promise.all([
    page.waitForURL(/\/wp-admin(\/|$|\?)/),
    page.click('#save-post'),
  ]);
  await expect(page.locator('.notice-error')).toContainText(/empty title and content/i);
});
