// @ts-check
const { test, expect } = require('@playwright/test');

test('front category archive /category/fixture-leaf/ lists posts', async ({ page, request }) => {
  const res = await request.get('/category/fixture-leaf/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/archive/i);
  expect(body).toMatch(/category/i);
  expect(body).toMatch(/Category:/i);
  expect(body).toMatch(/Fixture/i);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/category/fixture-leaf/');
  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/archive/);
  await expect(page.locator('body')).toHaveClass(/category/);
  await expect(page.locator('body')).toContainText(/Category:/i);
  await expect(page.locator('body')).toContainText(/Fixture/i);
});

test('front missing category archive returns 404', async ({ request }) => {
  const res = await request.get('/category/nobody-here-cat-e2e/');
  expect(res.status()).toBe(404);
  const body = await res.text();
  expect(body).not.toContain('front route not implemented');
});
