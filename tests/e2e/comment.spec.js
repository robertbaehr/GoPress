// @ts-check
const { test, expect } = require('@playwright/test');

test('visitor posts a comment on Hello world', async ({ page, baseURL }) => {
  const origin = baseURL || 'http://127.0.0.1:8082';
  const body = `E2E comment ${Date.now()}`;
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';

  // Log in first so the comment is auto-approved (comment_previously_approved).
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

  await page.goto('/?p=1');
  await expect(page.locator('#commentform')).toBeVisible();

  // Fixture DBs often store home/siteurl as the PHP oracle (:8081). Force the
  // comment form onto this GoPress origin and keep the redirect local.
  await page.evaluate((o) => {
    const form = document.querySelector('#commentform');
    if (!form) return;
    form.setAttribute('action', o + '/wp-comments-post.php');
    let input = form.querySelector('input[name="redirect_to"]');
    if (!input) {
      input = document.createElement('input');
      input.type = 'hidden';
      input.name = 'redirect_to';
      form.appendChild(input);
    }
    input.value = o + '/?p=1';
  }, origin);

  await page.fill('#comment', body);

  await Promise.all([
    page.waitForURL(/[?&]p=1.*#comment-\d+|\/?\?p=1#comment-\d+/),
    page.click('#submit'),
  ]);

  await expect(page.locator('body')).toContainText(body);
});
