// @ts-check
const { test, expect } = require('@playwright/test');

test('front tag archive /tag/fixture-tag-01/ lists posts', async ({ page, request }) => {
  const res = await request.get('/tag/fixture-tag-01/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/archive/i);
  expect(body).toMatch(/tag/i);
  expect(body).toMatch(/Tag:/i);
  expect(body).toMatch(/Fixture Post/i);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/tag/fixture-tag-01/');
  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/archive/);
  await expect(page.locator('body')).toHaveClass(/tag/);
  await expect(page.locator('body')).toContainText(/Tag:/i);
  await expect(page.locator('body')).toContainText(/Fixture Post/i);
});

test('front missing tag archive returns 404', async ({ request }) => {
  const res = await request.get('/tag/nobody-here-tag-e2e/');
  expect(res.status()).toBe(404);
  const body = await res.text();
  expect(body).not.toContain('front route not implemented');
});
