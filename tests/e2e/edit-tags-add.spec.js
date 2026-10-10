// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.50: E2E for edit-tags.php Add New Category (P7.58).
 */
test('edit-tags Add New Category creates a term', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');
  const stamp = Date.now();
  const name = `E2E Cat ${stamp}`;
  const slug = `e2e-cat-${stamp}`;

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

  await page.goto('/wp-admin/edit-tags.php?taxonomy=category');
  await expect(page.locator('h1.wp-heading-inline')).toContainText(/Categories/i);
  await expect(page.locator('#addtag')).toBeVisible();
  await expect(page.locator('input[name="action"]')).toHaveAttribute('value', 'add-tag');

  await page.fill('#tag-name', name);
  await page.fill('#tag-slug', slug);
  await page.fill('#tag-description', `E2E description ${stamp}`);
  await Promise.all([
    page.waitForURL(/edit-tags\.php.*message=1/),
    page.click('#submit'),
  ]);

  await expect(page.locator('#message')).toContainText(/Category added/i);
  await expect(page.locator('#the-list')).toContainText(name);
  await expect(page.locator('#the-list')).toContainText(slug);
});

test('edit-tags Add New Tag creates a term', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');
  const stamp = Date.now();
  const name = `E2E Tag ${stamp}`;

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

  await page.goto('/wp-admin/edit-tags.php?taxonomy=post_tag');
  await expect(page.locator('h1.wp-heading-inline')).toContainText(/Tags/i);
  await page.fill('#tag-name', name);
  await Promise.all([
    page.waitForURL(/edit-tags\.php.*message=1/),
    page.click('#submit'),
  ]);
  await expect(page.locator('#message')).toContainText(/Tag added/i);
  await expect(page.locator('#the-list')).toContainText(name);
});
