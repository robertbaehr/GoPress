// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.37: E2E for GET /wp/v2/plugins (+ plugin item) from P4.60.
 */
test('admin lists plugins via REST and fetches Hello Dolly', async ({ page, baseURL }) => {
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

  const anon = await page.request.get('/wp-json/wp/v2/plugins');
  expect(anon.status()).toBe(401);

  const listRes = await page.request.get('/wp-json/wp/v2/plugins', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(listRes.status()).toBe(200);
  const list = await listRes.json();
  expect(Array.isArray(list)).toBe(true);
  expect(list.length).toBeGreaterThan(0);
  const hello = list.find((p) => p.plugin === 'hello' || p.name === 'Hello Dolly');
  expect(hello).toBeTruthy();
  expect(hello.name).toBe('Hello Dolly');
  expect(hello.version).toBeTruthy();

  const itemRes = await page.request.get('/wp-json/wp/v2/plugins/hello', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(itemRes.status()).toBe(200);
  const item = await itemRes.json();
  expect(item.plugin).toBe('hello');
  expect(item.name).toBe('Hello Dolly');
  expect(item.status === 'active' || item.status === 'inactive').toBe(true);
});
