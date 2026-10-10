// @ts-check
const { test, expect } = require('@playwright/test');

test('resetpass without key redirects to lostpassword invalidkey', async ({ request }) => {
  const resp = await request.get('/wp-login.php?action=rp', { maxRedirects: 0 });
  expect(resp.status()).toBe(302);
  const loc = resp.headers()['location'] || '';
  expect(loc).toMatch(/action=lostpassword/);
  expect(loc).toMatch(/error=invalidkey/);
});

test('lostpassword shows invalidkey and expiredkey errors', async ({ page }) => {
  await page.goto('/wp-login.php?action=lostpassword&error=invalidkey');
  await expect(page.locator('#login_error')).toBeVisible();
  await expect(page.locator('#login_error')).toContainText(/invalid/i);
  await expect(page.locator('#lostpasswordform')).toBeVisible();

  await page.goto('/wp-login.php?action=lostpassword&error=expiredkey');
  await expect(page.locator('#login_error')).toBeVisible();
  await expect(page.locator('#login_error')).toContainText(/expired/i);
  await expect(page.locator('#lostpasswordform')).toBeVisible();
});
