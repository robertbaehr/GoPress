// @ts-check
const { test, expect } = require('@playwright/test');

test('front search /?s= returns results for Hello', async ({ page, request }) => {
  const res = await request.get('/?s=Hello');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/search/i);
  expect(body).toMatch(/Search results/i);
  expect(body).toMatch(/Hello world|Hello World/i);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/?s=Hello');
  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('body')).toHaveClass(/search/);
  await expect(page.locator('body')).toContainText(/Hello world|Hello World/i);
});

test('front search with no matches stays 200', async ({ request }) => {
  const res = await request.get('/?s=zzznomatchterm999');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/search/i);
  expect(body).not.toContain('front route not implemented');
});
