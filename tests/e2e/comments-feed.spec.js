// @ts-check
const { test, expect } = require('@playwright/test');

test('comments RSS feed /comments/feed/ lists approved comments', async ({ page, request }) => {
  const res = await request.get('/comments/feed/');
  expect(res.status()).toBe(200);
  const ct = res.headers()['content-type'] || '';
  expect(ct).toMatch(/xml|rss|text\/xml|application\/rss/i);
  const body = await res.text();
  expect(body).toMatch(/<rss\b/i);
  expect(body).toMatch(/Comments for/i);
  expect(body).toContain('/comments/feed/');
  expect(body).toMatch(/<item>/i);
  expect(body).toMatch(/Hello world/i);
  expect(body).not.toContain('front route not implemented');

  await page.goto('/comments/feed/');
  await expect(page.locator('rss, channel, body')).toBeVisible();
});

test('comments feed is not a 501 stub', async ({ request }) => {
  const res = await request.get('/comments/feed/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).not.toMatch(/not implemented/i);
  expect(body).toMatch(/<\/channel>/i);
});
