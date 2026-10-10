// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.68: E2E for edit.php status views (P7.73).
 */
test('edit.php All / Published / Drafts / Trash status views', async ({ page, baseURL }) => {
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
  await expect(page.locator('ul.subsubsub')).toBeVisible();
  await expect(page.locator('ul.subsubsub li.all a.current')).toBeVisible();
  await expect(page.locator('ul.subsubsub li.publish a')).toContainText('Published');
  await expect(page.locator('ul.subsubsub li.draft a')).toContainText('Drafts');
  await expect(page.locator('ul.subsubsub li.trash a')).toContainText('Trash');
  await expect(page.locator('#the-list')).toContainText('Hello world!');

  await Promise.all([
    page.waitForURL(/post_status=publish/),
    page.locator('ul.subsubsub li.publish a').click(),
  ]);
  await expect(page.locator('ul.subsubsub li.publish a.current')).toBeVisible();
  await expect(page.locator('#the-list')).toContainText('Hello world!');

  await Promise.all([
    page.waitForURL(/post_status=draft/),
    page.locator('ul.subsubsub li.draft a').click(),
  ]);
  await expect(page.locator('ul.subsubsub li.draft a.current')).toBeVisible();
  await expect(page.locator('#the-list')).toContainText(/Fixture Post/i);
  await expect(page.locator('#the-list')).not.toContainText('Hello world!');

  await Promise.all([
    page.waitForURL(/post_status=trash/),
    page.locator('ul.subsubsub li.trash a').click(),
  ]);
  await expect(page.locator('ul.subsubsub li.trash a.current')).toBeVisible();
  await expect(page.locator('#the-list')).toContainText(/Fixture Post/i);
  await expect(page.locator('#the-list')).not.toContainText('Hello world!');

  await Promise.all([
    page.waitForURL(/\/wp-admin\/edit\.php(?!.*post_status)/),
    page.locator('ul.subsubsub li.all a').click(),
  ]);
  await expect(page.locator('ul.subsubsub li.all a.current')).toBeVisible();
  await expect(page.locator('#the-list')).toContainText('Hello world!');
});
