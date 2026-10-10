// @ts-check
const { test, expect } = require('@playwright/test');

test('/comments/feed/atom/ returns Atom comments feed', async ({ request }) => {
  const resp = await request.get('/comments/feed/atom/');
  expect(resp.status()).toBe(200);
  const ct = resp.headers()['content-type'] || '';
  expect(ct).toMatch(/atom|xml/i);
  const body = await resp.text();
  expect(body).toContain('<feed');
  expect(body).toContain('xmlns="http://www.w3.org/2005/Atom"');
  expect(body).toContain('xmlns:thr="http://purl.org/syndication/thread/1.0"');
  expect(body).toMatch(/Comments for/i);
  expect(body).toContain('<entry>');
  expect(body).toContain('<thr:in-reply-to');
  expect(body).toMatch(/Hello world/i);
  expect(body).not.toMatch(/not implemented/i);
});

test('?feed=atom comments via comments path', async ({ request }) => {
  const resp = await request.get('/comments/feed/atom/');
  expect(resp.status()).toBe(200);
  const body = await resp.text();
  expect(body).toContain('<link rel="self"');
  expect(body).toContain('/comments/feed/atom/');
  expect(body).toContain('</feed>');
});
