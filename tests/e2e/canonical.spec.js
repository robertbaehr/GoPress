// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.42: E2E for rel=canonical / trailing-slash redirect (P3.42 / P3.38).
 */
test('singular page emits absolute rel=canonical', async ({ request }) => {
  const index = await request.get('/wp-json/');
  expect(index.status()).toBe(200);
  const { home } = await index.json();
  const homeBase = String(home).replace(/\/$/, '');

  const res = await request.get('/sample-page/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  const m = body.match(/<link\s+rel=["']canonical["']\s+href=["']([^"']+)["']/i);
  expect(m).toBeTruthy();
  expect(m[1]).toBe(homeBase + '/sample-page/');
});

test('singular post emits absolute rel=canonical', async ({ request }) => {
  const index = await request.get('/wp-json/');
  const { home } = await index.json();
  const homeBase = String(home).replace(/\/$/, '');

  const res = await request.get('/blog/uncategorized/hello-world/');
  expect(res.status()).toBe(200);
  const body = await res.text();
  const m = body.match(/<link\s+rel=["']canonical["']\s+href=["']([^"']+)["']/i);
  expect(m).toBeTruthy();
  expect(m[1]).toMatch(new RegExp('^' + homeBase.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '/'));
  expect(m[1]).toContain('hello-world');
});

test('missing trailing slash redirects to absolute canonical Location', async ({ request }) => {
  const index = await request.get('/wp-json/');
  const { home } = await index.json();
  const homeBase = String(home).replace(/\/$/, '');

  const res = await request.get('/sample-page', { maxRedirects: 0 });
  expect(res.status()).toBe(301);
  const loc = res.headers()['location'] || '';
  expect(loc).toBe(homeBase + '/sample-page/');
});
