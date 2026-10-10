// @ts-check
const { test, expect } = require('@playwright/test');

test('front author archive /author/admin/ lists posts', async ({ page, request }) => {
  const res = await request.get('/author/admin/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/archive/i);
  expect(body).toMatch(/author/i);
  expect(body).toMatch(/Author:/i);
  expect(body).toMatch(/Hello world|Hello World|Fixture Post/i);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/author/admin/');
  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/archive/);
  await expect(page.locator('body')).toHaveClass(/author/);
  await expect(page.locator('body')).toContainText(/Author:/i);
});

test('front missing author archive returns 404', async ({ request }) => {
  const res = await request.get('/author/nobody-here-e2e/');
  expect(res.status()).toBe(404);
  const body = await res.text();
  expect(body).not.toContain('front route not implemented');
});
