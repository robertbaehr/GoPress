// @ts-check
const { test, expect } = require('@playwright/test');

test('/rsd.xml returns RSD stub XML', async ({ request }) => {
  const resp = await request.get('/rsd.xml');
  expect(resp.status()).toBe(200);
  const ct = resp.headers()['content-type'] || '';
  expect(ct).toMatch(/xml/i);
  const body = await resp.text();
  expect(body).toContain('xmlns="http://archipelago.phrasewise.com/rsd"');
  expect(body).toContain('<engineName>GoPress</engineName>');
  expect(body).toContain('<apis>');
  expect(body).not.toMatch(/not implemented/i);
});

test('xmlrpc.php?rsd stays forbidden', async ({ request }) => {
  const resp = await request.get('/xmlrpc.php?rsd');
  expect(resp.status()).toBe(403);
});
