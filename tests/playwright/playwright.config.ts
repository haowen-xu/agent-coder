import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './specs',
  timeout: 5 * 60 * 1000,
  expect: {
    timeout: 20_000,
  },
  fullyParallel: false,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:18080',
    headless: true,
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
  },
})
