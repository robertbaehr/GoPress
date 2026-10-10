// @ts-check
const { test, expect } = require('@playwright/test');

test('front home pagination /page/2/ lists later posts', async ({ page, request }) => {
  const res = await request.get('/page/2/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/Page 2/i);
  expect(body).toMatch(/paged/i);
  expect(body).toMatch(/home/i);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/page/2/');
  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/home/);
  await expect(page.locator('body')).toHaveClass(/paged/);
  await expect(page.locator('body')).toHaveClass(/paged-2/);
  await expect(page).toHaveTitle(/Page 2/i);
});

test('front /page/1/ is home-shaped', async ({ request }) => {
  const res = await request.get('/page/1/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).not.toContain('front route not implemented');
  expect(body).toMatch(/home/i);
});
