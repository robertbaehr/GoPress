// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * P10.44: E2E for external redirect_to rejection (P6.20).
 */
test('login POST rejects external redirect_to', async ({ request }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';

  const res = await request.post('/wp-login.php', {
    form: {
      log: user,
      pwd: pass,
      redirect_to: 'https://evil.example/phish',
      'wp-submit': 'Log In',
      testcookie: '1',
    },
    maxRedirects: 0,
  });
  expect([302, 303]).toContain(res.status());
  const loc = res.headers()['location'] || '';
  expect(loc).not.toMatch(/evil\.example/i);
  expect(loc).toMatch(/\/wp-admin\/?/);
});

test('logout rejects external redirect_to', async ({ request }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';

  // Establish a session first.
  const login = await request.post('/wp-login.php', {
    form: {
      log: user,
      pwd: pass,
      redirect_to: '/wp-admin/',
      'wp-submit': 'Log In',
      testcookie: '1',
    },
    maxRedirects: 5,
  });
  expect(login.ok()).toBeTruthy();

  const res = await request.get(
    '/wp-login.php?action=logout&redirect_to=' + encodeURIComponent('https://evil.example/steal'),
    { maxRedirects: 0 },
  );
  expect([302, 303]).toContain(res.status());
  const loc = res.headers()['location'] || '';
  expect(loc).not.toMatch(/evil\.example/i);
  expect(loc).toMatch(/\/wp-admin\/?/);
});

test('logout allows same-host absolute redirect_to', async ({ request, baseURL }) => {
  const user = process.env.GOPRESS_ADMIN_USER || 'admin';
  const pass = process.env.GOPRESS_ADMIN_PASSWORD || 'password';
  const origin = (baseURL || 'http://127.0.0.1:8082').replace(/\/$/, '');

  const login = await request.post('/wp-login.php', {
    form: {
      log: user,
      pwd: pass,
      redirect_to: '/wp-admin/',
      'wp-submit': 'Log In',
      testcookie: '1',
    },
    maxRedirects: 5,
  });
  expect(login.ok()).toBeTruthy();

  const safe = origin + '/sample-page/';
  const res = await request.get(
    '/wp-login.php?action=logout&redirect_to=' + encodeURIComponent(safe),
    { maxRedirects: 0 },
  );
  expect([302, 303]).toContain(res.status());
  const loc = res.headers()['location'] || '';
  expect(loc).toContain('/sample-page/');
  expect(loc).not.toMatch(/evil\.example/i);
});
