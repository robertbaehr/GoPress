// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.52: E2E for DELETE /wp/v2/menu-locations/{location} (P4.74).
 */
test('admin can assign then DELETE unset a menu location', async ({ page, request, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  const anon = await request.delete('/wp-json/wp/v2/menu-locations/primary');
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

  const menusRes = await page.request.get('/wp-json/wp/v2/menus?per_page=5', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(menusRes.status()).toBe(200);
  const menus = await menusRes.json();
  expect(Array.isArray(menus)).toBe(true);
  expect(menus.length).toBeGreaterThan(0);
  const menuId = menus[0].id;
  expect(menuId).toBeTruthy();

  const assign = await page.request.post('/wp-json/wp/v2/menu-locations/primary', {
    headers: {
      'X-WP-Nonce': nonce,
      'Content-Type': 'application/json',
    },
    data: { menu: menuId },
  });
  expect(assign.status()).toBe(200);
  const assigned = await assign.json();
  expect(assigned.name).toBe('primary');
  expect(assigned.menu).toBe(menuId);

  const del = await page.request.delete('/wp-json/wp/v2/menu-locations/primary', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(del.status()).toBe(200);
  const unset = await del.json();
  expect(unset.name).toBe('primary');
  expect(unset.menu).toBe(0);

  const get = await page.request.get('/wp-json/wp/v2/menu-locations/primary', {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(get.status()).toBe(200);
  const after = await get.json();
  expect(after.menu).toBe(0);
});
