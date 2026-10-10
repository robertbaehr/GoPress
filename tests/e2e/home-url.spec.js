// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.36: home_url / site_url helpers surface on front + REST.
 * Fixture home/siteurl point at the PHP oracle host; GoPress should emit
 * that same base in links (not the Go listen address).
 */
test('REST index exposes home and url from options', async ({ request }) => {
  const res = await request.get('/wp-json/');
  expect(res.status()).toBe(200);
  const body = await res.json();
  expect(body.home).toMatch(/^https?:\/\//);
  expect(body.url).toMatch(/^https?:\/\//);
  expect(body.home).toContain('127.0.0.1');
  expect(body.url).toContain('127.0.0.1');
});

test('front home pagination links use home_url base', async ({ request }) => {
  const index = await request.get('/wp-json/');
  const { home } = await index.json();
  const homeBase = String(home).replace(/\/$/, '');

  const res = await request.get('/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  expect(body).not.toContain('front route not implemented');
  // Query pagination next / page numbers should be absolute under home_url.
  expect(body).toMatch(new RegExp(homeBase.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '/page/2/'));
  expect(body).toMatch(/wp-block-query-pagination/);
});

test('trailing-slash canonical Location uses absolute home_url', async ({ request }) => {
  const index = await request.get('/wp-json/');
  const { home } = await index.json();
  const homeBase = String(home).replace(/\/$/, '');

  const res = await request.get('/sample-page', { maxRedirects: 0 });
  expect(res.status()).toBe(301);
  const loc = res.headers()['location'] || '';
  expect(loc).toBe(homeBase + '/sample-page/');
});
