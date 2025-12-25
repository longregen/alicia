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

  // Global timeout for entire test run (10 minutes for NixOS VM tests)
  globalTimeout: isNixosTest ? 600000 : 0,

  use: {
    baseURL: SERVER_URL,

    // Use headless mode in NixOS VM tests (no GPU, more reliable)
    headless: isNixosTest ? true : false,

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
      // Disable HAR recording in NixOS tests (can cause hangs)
      ...(isNixosTest ? {} : {
        recordHar: {
          path: path.join(outputDir, 'traces/har'),
          mode: 'full',
          content: 'embed',
        },
      }),
    },

    locale: 'en-US',
    timezoneId: 'UTC',
  },

  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // Don't use 'channel' in NixOS tests - it overrides PLAYWRIGHT_BROWSERS_PATH
        ...(isNixosTest ? {} : { channel: 'chromium' }),
        launchOptions: {
          // In NixOS tests, use explicit path from environment variable
          executablePath: isNixosTest ? process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH : undefined,
          args: [
            '--disable-gpu',
            '--no-sandbox',
            '--disable-dev-shm-usage',
            '--disable-setuid-sandbox',
            '--use-fake-ui-for-media-stream',
            '--use-fake-device-for-media-stream',
            // Additional flags for NixOS VM environment
            ...(isNixosTest ? [
              '--headless=new',
              '--disable-software-rasterizer',
              '--disable-extensions',
              '--disable-background-networking',
              '--disable-sync',
              '--disable-translate',
              '--no-first-run',
              '--disable-features=VizDisplayCompositor',
            ] : []),
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
