// @ts-check
const { test, expect } = require('@playwright/test');

test('admin creates a post via REST', async ({ page, request, baseURL }) => {
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

  const title = `E2E REST Post ${Date.now()}`;
  const createRes = await page.request.post('/wp-json/wp/v2/posts', {
    headers: {
      'Content-Type': 'application/json',
      'X-WP-Nonce': nonce,
    },
    data: {
      title,
      content: 'Created by Playwright e2e',
      status: 'draft',
    },
  });
  expect(createRes.status()).toBe(201);
  const created = await createRes.json();
  expect(created.id).toBeTruthy();
  expect(created.status).toBe('draft');
  const renderedTitle =
    (created.title && (created.title.raw || created.title.rendered)) || '';
  expect(String(renderedTitle)).toContain('E2E REST Post');

  // Sanity: fetch the draft back with the same auth.
  const getRes = await page.request.get(`/wp-json/wp/v2/posts/${created.id}?context=edit`, {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(getRes.status()).toBe(200);
  const fetched = await getRes.json();
  expect(fetched.id).toBe(created.id);
});
