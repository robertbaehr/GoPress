// @ts-check
const { test, expect } = require('@playwright/test');

test('/feed/rdf/ returns RDF/RSS 1.0 stub', async ({ request }) => {
  const resp = await request.get('/feed/rdf/');
  expect(resp.status()).toBe(200);
  const ct = resp.headers()['content-type'] || '';
  expect(ct).toMatch(/rdf|xml/i);
  const body = await resp.text();
  expect(body).toContain('<rdf:RDF');
  expect(body).toContain('xmlns="http://purl.org/rss/1.0/"');
  expect(body).toContain('<rdf:Seq>');
  expect(body).toContain('<item rdf:about=');
  expect(body).not.toMatch(/not implemented/i);
});

test('?feed=rdf returns RDF channel', async ({ request }) => {
  const resp = await request.get('/?feed=rdf');
  expect(resp.status()).toBe(200);
  const body = await resp.text();
  expect(body).toContain('<rdf:RDF');
  expect(body).toContain('<channel rdf:about=');
});
