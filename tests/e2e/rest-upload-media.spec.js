// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const fs = require('fs');
const os = require('os');

test('admin uploads media via REST', async ({ page, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = baseURL || 'http://127.0.0.1:8082';

  await page.goto('/wp-login.php');
  await expect(page.locator('#loginform')).toBeVisible();
  await page.evaluate((o) => {
    const form = document.querySelector('#loginform');
    if (form) form.setAttribute('action', o + '/wp-login.php');
  }, origin);
  await page.fill('#user_login', user);
  await page.fill('#user_pass', pass);
  await Promise.all([
    page.waitForURL(/\/wp-admin(\/|$|\?)/),
    page.click('#wp-submit'),
  ]);

  const nonceRes = await page.request.get('/wp-admin/admin-ajax.php?action=rest-nonce');
  expect(nonceRes.status()).toBe(200);
  const nonce = (await nonceRes.text()).trim();
  expect(nonce.length).toBeGreaterThan(5);

  // Minimal PNG header bytes written to a temp file for multipart upload.
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'gopress-e2e-'));
  const filePath = path.join(tmpDir, `e2e-upload-${Date.now()}.png`);
  fs.writeFileSync(filePath, Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]));

  const uploadRes = await page.request.post('/wp-json/wp/v2/media', {
    headers: {
      'X-WP-Nonce': nonce,
    },
    multipart: {
      file: {
        name: path.basename(filePath),
        mimeType: 'image/png',
        buffer: fs.readFileSync(filePath),
      },
    },
  });
  expect(uploadRes.status()).toBe(201);
  const created = await uploadRes.json();
  expect(created.id).toBeTruthy();
  expect(created.type || created.mime_type).toBeTruthy();
  const sourceURL = created.source_url || '';
  expect(sourceURL).toContain('/wp-content/uploads/');

  const getRes = await page.request.get(`/wp-json/wp/v2/media/${created.id}`, {
    headers: { 'X-WP-Nonce': nonce },
  });
  expect(getRes.status()).toBe(200);
  const fetched = await getRes.json();
  expect(fetched.id).toBe(created.id);

  try {
    fs.unlinkSync(filePath);
    fs.rmdirSync(tmpDir);
  } catch {
    // best-effort cleanup
  }
});
