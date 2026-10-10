// @ts-check
const { test, expect } = require('@playwright/test');

test('admin updates blogname via REST settings', async ({ page, baseURL }) => {
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

  const nonceRes = await page.request.get('/wp-admin/admin-ajax.php?action=rest-nonce');
  expect(nonceRes.status()).toBe(200);
  const nonce = (await nonceRes.text()).trim();
  expect(nonce.length).toBeGreaterThan(5);

  const beforeRes = await page.request.get('/wp-json/wp/v2/settings', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(beforeRes.status()).toBe(200);
  const before = await beforeRes.json();
  const original = before.blogname || before.title || '';
  expect(original.length).toBeGreaterThan(0);

  const nextName = `E2E Settings ${Date.now()}`;
  const updateRes = await page.request.post('/wp-json/wp/v2/settings', {
    headers: {
      'Content-Type': 'application/json',
      'X-WP-Nonce': nonce,
    },
    data: { title: nextName },
  });
  expect(updateRes.status()).toBe(200);
  const updated = await updateRes.json();
  expect(updated.blogname || updated.title).toBe(nextName);
  expect(updated.title).toBe(nextName);

  const verifyRes = await page.request.get('/wp-json/wp/v2/settings', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(verifyRes.status()).toBe(200);
  const verified = await verifyRes.json();
  expect(verified.blogname).toBe(nextName);

  // Restore fixture title so other tests / corpus stay stable.
  const restoreRes = await page.request.post('/wp-json/wp/v2/settings', {
    headers: {
      'Content-Type': 'application/json',
      'X-WP-Nonce': nonce,
    },
    data: { title: original },
  });
  expect(restoreRes.status()).toBe(200);
  const restored = await restoreRes.json();
  expect(restored.blogname || restored.title).toBe(original);
});
