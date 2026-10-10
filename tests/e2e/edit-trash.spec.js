// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.66: E2E for edit.php Trash + Restore (P7.74).
 */
test('edit.php trash and restore row actions', async ({ page, baseURL }) => {
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
  await expect(page.locator('option[value="trash"]')).toBeAttached();
  await expect(page.locator('#the-list .row-actions .trash a').first()).toBeAttached();

  // Search finds Fixture Post 01 (not always on page 1 among block fixtures).
  await page.fill('#post-search-input', 'Fixture Post 01');
  await Promise.all([
    page.waitForURL(/[?&]s=/),
    page.click('#search-submit'),
  ]);
  await expect(page.locator('#the-list')).toContainText('Fixture Post 01');

  const trashLink = page.locator('#the-list tr').filter({ hasText: 'Fixture Post 01' }).locator('.row-actions .trash a');
  await expect(trashLink).toBeAttached();
  await Promise.all([
    page.waitForURL(/trashed=1/),
    trashLink.click(),
  ]);
  await expect(page.locator('body')).toContainText(/moved to the Trash|trashed/i);

  await page.goto('/wp-admin/edit.php?post_status=trash&s=Fixture+Post+01');
  await expect(page.locator('#the-list')).toContainText('Fixture Post 01');
  await expect(page.locator('option[value="untrash"]')).toBeAttached();
  await expect(page.locator('#the-list .row-actions .untrash a').first()).toBeAttached();

  const restoreLink = page.locator('#the-list tr').filter({ hasText: 'Fixture Post 01' }).locator('.row-actions .untrash a');
  await Promise.all([
    page.waitForURL(/untrashed=1/),
    restoreLink.click(),
  ]);
  await expect(page.locator('body')).toContainText(/restored from the Trash|untrashed/i);

  await page.goto('/wp-admin/edit.php?s=Fixture+Post+01');
  await expect(page.locator('#the-list')).toContainText('Fixture Post 01');
});
