// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.54: E2E for POST /wp/v2/themes/{stylesheet} status activate (P4.78).
 */
test('admin can activate a theme via REST and restore previous', async ({ page, request, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  const anon = await request.post('/wp-json/wp/v2/themes/twentytwentyfour', {
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

  const activeList = await page.request.get('/wp-json/wp/v2/themes?status=active', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(activeList.status()).toBe(200);
  const activeThemes = await activeList.json();
  expect(Array.isArray(activeThemes)).toBeTruthy();
  expect(activeThemes.length).toBe(1);
  const previous = activeThemes[0].stylesheet;
  expect(previous).toBeTruthy();

  let target = 'twentytwentyfour';
  if (target === previous) {
    target = 'twentytwentythree';
  }

  const activate = await page.request.post(`/wp-json/wp/v2/themes/${target}`, {
    headers: { 'X-WP-Nonce': nonce, 'Content-Type': 'application/json' },
    data: { status: 'active' },
  });
  expect(activate.status()).toBe(200);
  const activated = await activate.json();
  expect(activated.stylesheet).toBe(target);
  expect(activated.status).toBe('active');

  // Restore fixture theme.
  const restore = await page.request.post(`/wp-json/wp/v2/themes/${previous}`, {
    headers: { 'X-WP-Nonce': nonce, 'Content-Type': 'application/json' },
    data: { status: 'active' },
  });
  expect(restore.status()).toBe(200);
  const restored = await restore.json();
  expect(restored.stylesheet).toBe(previous);
  expect(restored.status).toBe('active');
});
