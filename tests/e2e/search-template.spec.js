// @ts-check
const { test, expect } = require('@playwright/test');

test('search.html template renders query-title, search form, and results', async ({ page, request }) => {
  const res = await request.get('/?s=Hello');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/wp-block-query-title/i);
  expect(body).toMatch(/Search results for/i);
  // Curly quotes around the term (P3.36 / TT5).
  expect(body).toMatch(/\u201CHello\u201D|&#8220;Hello&#8221;/);
  expect(body).toContain('role="search"');
  expect(body).toMatch(/wp-block-search/);
  expect(body).toMatch(/Hello world|Hello World/i);
  expect(body).toMatch(/wp-theme-twentytwentyfive/);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/?s=Hello');
  await expect(page.locator('body')).toHaveClass(/search/);
  await expect(page.locator('body')).toHaveClass(/search-results/);
  await expect(page.locator('.wp-block-query-title')).toBeVisible();
  await expect(page.locator('form.wp-block-search, form[role="search"]')).toBeVisible();
  await expect(page.locator('input[name="s"]')).toHaveValue('Hello');
  await expect(page.locator('body')).toContainText(/Hello world|Hello World/i);
});

test('empty search still uses search template chrome', async ({ request }) => {
  const res = await request.get('/?s=');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).toMatch(/search/i);
  expect(body).toMatch(/wp-block-search|role="search"|Search results/i);
  expect(body).not.toContain('front route not implemented');
});
