import { defineConfig, devices } from '@playwright/test';
import path from 'path';

const SERVER_URL = process.env.BASE_URL || process.env.ALICIA_SERVER_URL || 'http://localhost:8080';
const outputDir = process.env.ARTIFACT_DIR || './test-results';
const isCI = !!process.env.CI;
const isNixosTest = !!process.env.NIXOS_TEST;

export default defineConfig({
  testDir: './tests',
  outputDir: path.join(outputDir, 'playwright-output'),

  fullyParallel: false,
  forbidOnly: !!isCI,
  retries: isCI ? 2 : 1,
  workers: 1,

  reporter: [
    ['html', { outputFolder: path.join(outputDir, 'report'), open: 'never' }],
    ['json', { outputFile: path.join(outputDir, 'results.json') }],
    ['list'],
  ],

  timeout: isNixosTest ? 60000 : 30000,
  expect: {
    timeout: isNixosTest ? 10000 : 5000,
  },

  use: {
    baseURL: SERVER_URL,

    headless: false,

    viewport: { width: 1920, height: 1080 },

    screenshot: {
      mode: 'on',
      fullPage: false,
    },

    video: {
      mode: 'retain-on-failure',
      size: { width: 1920, height: 1080 },
    },

    trace: 'retain-on-failure',

    actionTimeout: 15000,
    navigationTimeout: 30000,

    contextOptions: {
      reducedMotion: 'no-preference',
      colorScheme: 'light',
      recordHar: {
        path: path.join(outputDir, 'traces/har'),
        mode: 'full',
        content: 'embed',
      },
    },

    locale: 'en-US',
    timezoneId: 'UTC',
  },

  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        channel: 'chromium',
        launchOptions: {
          args: [
            '--disable-gpu',
            '--no-sandbox',
            '--disable-dev-shm-usage',
            '--disable-setuid-sandbox',
            '--use-fake-ui-for-media-stream',
            '--use-fake-device-for-media-stream',
          ],
        },
      },
    },
  ],

  webServer: isNixosTest ? undefined : {
    command: 'echo "Using external server"',
    url: SERVER_URL,
    reuseExistingServer: true,
    timeout: 120000,
  },
});
