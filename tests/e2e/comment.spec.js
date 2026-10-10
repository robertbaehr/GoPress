// @ts-check
const { test, expect } = require('@playwright/test');

test('visitor posts a comment on Hello world', async ({ page, baseURL }) => {
  const origin = baseURL || 'http://127.0.0.1:8082';
  const body = `E2E comment ${Date.now()}`;

  await page.goto('/?p=1');
  await expect(page.locator('#commentform')).toBeVisible();

  // Fixture DBs often store home/siteurl as the PHP oracle (:8081). Force the
  // comment form to post back to this GoPress origin.
  await page.evaluate((o) => {
    const form = document.querySelector('#commentform');
    if (form) form.setAttribute('action', o + '/wp-comments-post.php');
  }, origin);

  await page.fill('#author', 'E2E Commenter');
  await page.fill('#email', 'e2e-commenter@example.com');
  await page.fill('#comment', body);

  await Promise.all([
    page.waitForURL(/#comment-\d+/),
    page.click('#submit'),
  ]);

  await expect(page.locator('body')).toContainText(body);
});
