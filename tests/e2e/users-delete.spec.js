// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.55: E2E for DELETE /wp/v2/users/{id}?force=true&reassign= (P4.79).
 */
test('admin can delete a user via REST with reassign', async ({ page, request, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  const anon = await request.delete('/wp-json/wp/v2/users/2?force=true');
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

  const stamp = Date.now();
  const username = `e2e_del_${stamp}`;
  const createRes = await page.request.post('/wp-json/wp/v2/users', {
    headers: { 'Content-Type': 'application/json', 'X-WP-Nonce': nonce },
    data: {
      username,
      email: `${username}@example.com`,
      password: 'E2eDelPass!234',
      roles: ['author'],
    },
  });
  expect(createRes.status()).toBe(201);
  const created = await createRes.json();
  expect(created.id).toBeTruthy();

  const noForce = await page.request.delete(`/wp-json/wp/v2/users/${created.id}`, {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(noForce.status()).toBe(501);

  const meRes = await page.request.get('/wp-json/wp/v2/users/me', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(meRes.status()).toBe(200);
  const me = await meRes.json();
  expect(me.id).toBeTruthy();

  const delRes = await page.request.delete(
    `/wp-json/wp/v2/users/${created.id}?force=true&reassign=${me.id}`,
    { headers: { 'X-WP-Nonce': nonce } },
  );
  expect(delRes.status()).toBe(200);
  const deleted = await delRes.json();
  expect(deleted.deleted).toBe(true);
  expect(deleted.previous).toBeTruthy();
  expect(deleted.previous.id).toBe(created.id);

  const gone = await page.request.get(`/wp-json/wp/v2/users/${created.id}`, {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(gone.status()).toBe(404);
});
