// @ts-check
const { test, expect } = require('@playwright/test');

test('front year archive /2024/ lists posts', async ({ page, request }) => {
  const res = await request.get('/2024/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/archive/i);
  expect(body).toMatch(/Year:/i);
  expect(body).toMatch(/Hello world|Hello World/i);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/2024/');
  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/archive/);
  await expect(page.locator('body')).toHaveClass(/date/);
  await expect(page.locator('body')).toContainText(/Year:/i);
  await expect(page.locator('body')).toContainText(/Hello world|Hello World/i);
});

test('front month archive /2024/06/ lists June posts', async ({ page, request }) => {
  const res = await request.get('/2024/06/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/Month:/i);
  expect(body).toMatch(/June/i);
  expect(body).toMatch(/Hello world|Hello World/i);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/2024/06/');
  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/archive/);
  await expect(page.locator('body')).toContainText(/Month:/i);
});

test('front empty year archive returns 404', async ({ request }) => {
  const res = await request.get('/2019/');
  expect(res.status()).toBe(404);
  const body = await res.text();
  expect(body).not.toContain('front route not implemented');
});
