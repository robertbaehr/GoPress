// @ts-check
const { test, expect } = require('@playwright/test');

test('home returns 200 and contains blogname', async ({ page, request }) => {
  const res = await request.get('/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  // Fixture / default sites expose blogname in the home document.
  expect(body.length).toBeGreaterThan(0);
  // Prefer title/h1 content; fall back to any non-empty HTML from a live server.
  const blogname = process.env.GOPRESS_BLOGNAME || 'GoPress';
  expect(body).toContain(blogname);
  // Also exercise a real navigation for Playwright browser path.
  await page.goto('/');
  await expect(page.locator('body')).toBeVisible();
  // Blogname lives in <title> for block-theme home (not always in body text).
  await expect(page).toHaveTitle(new RegExp(blogname.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
});
