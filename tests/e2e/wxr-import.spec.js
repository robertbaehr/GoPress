// @ts-check
const { test, expect } = require('@playwright/test');
const path = require('path');
const fs = require('fs');
const os = require('os');

/**
 * P10.46: E2E for Tools → Import WXR upload stub (P7.54).
 */
async function loginAdmin(page, baseURL) {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  await page.goto('/wp-login.php');
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
}

test('import.php screen offers WXR uploader', async ({ page, baseURL }) => {
  await loginAdmin(page, baseURL);

  await page.goto('/wp-admin/import.php');
  await expect(page.locator('h1')).toContainText(/Import/i);
  await expect(page.locator('#wordpress-importer')).toBeVisible();
  await expect(page.locator('input[name="import_file"]')).toBeVisible();
  await expect(page.locator('#submit')).toBeEnabled();
  await expect(page.locator('text=WordPress')).toBeVisible();
});

test('import.php rejects non-WXR upload', async ({ page, baseURL }) => {
  await loginAdmin(page, baseURL);
  await page.goto('/wp-admin/import.php');

  const tmp = path.join(os.tmpdir(), `gopress-not-wxr-${Date.now()}.txt`);
  fs.writeFileSync(tmp, 'this is not a wordpress export');
  try {
    await page.setInputFiles('input[name="import_file"]', tmp);
    await page.click('#submit');
    await expect(page.locator('.notice-error')).toBeVisible();
    await expect(page.locator('.notice-error')).toContainText(/Import rejected|not a WXR/i);
  } finally {
    fs.unlinkSync(tmp);
  }
});

test('import.php parses minimal WXR', async ({ page, baseURL }) => {
  await loginAdmin(page, baseURL);
  await page.goto('/wp-admin/import.php');

  const wxr = `<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0"
	xmlns:excerpt="http://wordpress.org/export/1.2/excerpt/"
	xmlns:content="http://purl.org/rss/1.0/modules/content/"
	xmlns:dc="http://purl.org/dc/elements/1.1/"
	xmlns:wp="http://wordpress.org/export/1.2/">
<channel>
	<title>E2E Fixture</title>
	<link>http://example.com</link>
	<wp:wxr_version>1.2</wp:wxr_version>
	<item>
		<title>Hello Import</title>
		<wp:post_type>post</wp:post_type>
	</item>
	<item>
		<title>About Import</title>
		<wp:post_type>page</wp:post_type>
	</item>
</channel>
</rss>
`;
  const tmp = path.join(os.tmpdir(), `gopress-wxr-${Date.now()}.xml`);
  fs.writeFileSync(tmp, wxr);
  try {
    await page.setInputFiles('input[name="import_file"]', tmp);
    await page.click('#submit');
    await expect(page.locator('.notice-success')).toBeVisible();
    await expect(page.locator('.notice-success')).toContainText(/Parsed WXR/i);
    await expect(page.locator('.notice-success')).toContainText(/version 1\.2/i);
    await expect(page.locator('.notice-success')).toContainText(/2 item/);
  } finally {
    fs.unlinkSync(tmp);
  }
});
