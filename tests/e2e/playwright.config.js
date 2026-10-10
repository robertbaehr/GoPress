// @ts-check
const { defineConfig } = require('@playwright/test');

const baseURL = process.env.GOPRESS_URL || 'http://127.0.0.1:8082';

module.exports = defineConfig({
  testDir: '.',
  timeout: 30_000,
  retries: 0,
  use: {
    baseURL,
    trace: 'off',
  },
  reporter: [['list']],
});
