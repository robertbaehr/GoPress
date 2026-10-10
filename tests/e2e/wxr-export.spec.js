// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.45: E2E for Tools → Export minimal WXR download (P7.53).
 */
test('export.php screen offers WXR download', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  await page.goto('/wp-login.php');
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

  await page.goto('/wp-admin/export.php');
  await expect(page.locator('h1')).toContainText(/Export/i);
  await expect(page.locator('#export-filters')).toBeVisible();
  await expect(page.locator('input[name="download"]')).toHaveAttribute('value', 'true');
  await expect(page.locator('#submit')).toBeEnabled();
});

test('export.php?download=true returns WXR XML', async ({ page, request, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  await page.goto('/wp-login.php');
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

  // Reuse browser storage cookies for authenticated download.
  const cookies = await page.context().cookies();
  const cookieHeader = cookies.map((c) => `${c.name}=${c.value}`).join('; ');

  const res = await request.get('/wp-admin/export.php?download=true&content=all', {
    headers: { Cookie: cookieHeader },
  });
  expect(res.status()).toBe(200);
  const ct = res.headers()['content-type'] || '';
  expect(ct).toMatch(/xml/i);
  const cd = res.headers()['content-disposition'] || '';
  expect(cd).toMatch(/attachment/i);
  expect(cd).toMatch(/\.xml/i);
  const body = await res.text();
  expect(body).toContain('<rss version="2.0"');
  expect(body).toContain('<wp:wxr_version>1.2</wp:wxr_version>');
  expect(body).toContain('<wp:base_blog_url>');
  expect(body).toContain('<item>');
  expect(body).toContain('<wp:post_type>');
  expect(body).toContain('</channel>');
});
