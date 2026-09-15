import { defineConfig } from '@playwright/test';

if (!process.env.BASESTACK_APP_DIR) throw new Error('Set BASESTACK_APP_DIR to the generated smoke-app.');

export default defineConfig({
  testDir: '.',
  testMatch: '*.spec.mjs',
  workers: 1,
  retries: 0,
  reporter: 'list',
  use: { baseURL: 'http://127.0.0.1:4173', screenshot: 'only-on-failure', trace: 'retain-on-failure' },
  projects: [
    { name: 'desktop', use: { viewport: { width: 1440, height: 1000 } } },
    { name: 'mobile', use: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true } },
  ],
  webServer: {
    command: 'npm run preview -- --port 4173 --strictPort',
    cwd: process.env.BASESTACK_APP_DIR,
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: false,
  },
});
