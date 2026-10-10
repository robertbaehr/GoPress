// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.53: E2E for PUT /wp/v2/plugins/{plugin} status activate/deactivate (P4.77).
 */
test('admin can activate and deactivate Hello Dolly via REST', async ({ page, request, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  const anon = await request.put('/wp-json/wp/v2/plugins/hello', {
    data: { status: 'active' },
  });
  expect(anon.status()).toBe(401);

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

  const deactivate = await page.request.put('/wp-json/wp/v2/plugins/hello', {
    headers: { 'X-WP-Nonce': nonce, 'Content-Type': 'application/json' },
    data: { status: 'inactive' },
  });
  expect(deactivate.status()).toBe(200);
  const inactive = await deactivate.json();
  expect(inactive.plugin).toBe('hello');
  expect(inactive.status).toBe('inactive');

  const activate = await page.request.put('/wp-json/wp/v2/plugins/hello', {
    headers: { 'X-WP-Nonce': nonce, 'Content-Type': 'application/json' },
    data: { status: 'active' },
  });
  expect(activate.status()).toBe(200);
  const active = await activate.json();
  expect(active.status).toBe('active');

  // Leave inactive for a clean fixture-ish state.
  await page.request.put('/wp-json/wp/v2/plugins/hello', {
    headers: { 'X-WP-Nonce': nonce, 'Content-Type': 'application/json' },
    data: { status: 'inactive' },
  });
});
