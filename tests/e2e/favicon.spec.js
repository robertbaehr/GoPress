// @ts-check
const { test, expect } = require('@playwright/test');

test('favicon.ico redirects to site icon or default logo', async ({ request }) => {
  const resp = await request.get('/favicon.ico', { maxRedirects: 0 });
  expect(resp.status()).toBe(302);
  const loc = resp.headers()['location'] || '';
  expect(loc).toMatch(/w-logo-gray-white-bg\.png|\/wp-content\/uploads\//);
});

test('favicon.ico redirect resolves to an image', async ({ request }) => {
  const resp = await request.get('/favicon.ico');
  expect(resp.status()).toBe(200);
  const ct = resp.headers()['content-type'] || '';
  expect(ct).toMatch(/^image\//);
});
