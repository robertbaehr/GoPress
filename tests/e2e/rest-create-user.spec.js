// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.39: E2E for POST /wp/v2/users from P4.62.
 */
test('admin creates a user via REST', async ({ page, request, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = baseURL || 'http://127.0.0.1:8082';

  const anon = await request.post('/wp-json/wp/v2/users', {
    data: {
      username: 'anon-denied',
      email: 'anon-denied@example.com',
      password: 'password123',
    },
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

  const stamp = Date.now();
  const username = `e2e_user_${stamp}`;
  const email = `e2e_user_${stamp}@example.com`;
  const createRes = await page.request.post('/wp-json/wp/v2/users', {
    headers: {
      'Content-Type': 'application/json',
      'X-WP-Nonce': nonce,
    },
    data: {
      username,
      email,
      password: 'E2eUserPass!234',
      name: `E2E User ${stamp}`,
      roles: ['subscriber'],
    },
  });
  expect(createRes.status()).toBe(201);
  const created = await createRes.json();
  expect(created.id).toBeTruthy();
  expect(created.username || created.slug).toBeTruthy();
  expect(String(created.email || '')).toContain('@example.com');

  const getRes = await page.request.get(`/wp-json/wp/v2/users/${created.id}?context=edit`, {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(getRes.status()).toBe(200);
  const fetched = await getRes.json();
  expect(fetched.id).toBe(created.id);
});
