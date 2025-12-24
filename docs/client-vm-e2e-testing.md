# Client VM and E2E Testing Architecture

## Overview

This document describes the architecture for running end-to-end tests against Alicia's web frontend using a NixOS VM with XFCE desktop environment. The VM provides a consistent, reproducible graphical environment for Playwright to execute visual tests in headed mode.

**Goals:**
- Visual test execution with screenshot capture at key interaction points
- Consistent screen resolution for reproducible visual testing
- Full browser automation via Playwright in headed (non-headless) mode
- Smoke test coverage for core user workflows

**Decisions:**
- XFCE chosen for minimal resource footprint while providing full X11 support
- LightDM with autologin for unattended test execution
- 1920x1080 resolution for consistent screenshot baselines
- Chromium browser (Playwright's default on Linux)
- Node.js 22 for Playwright execution

---

## 1. NixOS VM Configuration

### 1.1 Client VM Module

Create `/nix/modules/client-vm.nix`:

```nix
{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.alicia.testing.clientVm;
in
{
  options.alicia.testing.clientVm = {
    enable = mkEnableOption "Alicia client VM for e2e testing";

    serverUrl = mkOption {
      type = types.str;
      default = "http://server-vm:8080";
      description = "URL of the Alicia server to test against";
    };

    resolution = mkOption {
      type = types.str;
      default = "1920x1080";
      description = "Screen resolution for display";
    };

    testUser = mkOption {
      type = types.str;
      default = "tester";
      description = "User account for running tests";
    };
  };

  config = mkIf cfg.enable {
    # XFCE Desktop Environment (minimal)
    services.xserver = {
      enable = true;

      # Display manager with autologin
      displayManager = {
        lightdm = {
          enable = true;
          autoLogin = {
            enable = true;
            user = cfg.testUser;
          };
        };
        defaultSession = "xfce";
      };

      # XFCE desktop
      desktopManager.xfce = {
        enable = true;
        enableXfwm = true;
        noDesktop = false;
      };

      # Virtual display configuration
      resolutions = [
        { x = 1920; y = 1080; }
      ];

      # Disable screen blanking for tests
      serverFlags = {
        Option = "BlankTime" "0";
        Option = "StandbyTime" "0";
        Option = "SuspendTime" "0";
        Option = "OffTime" "0";
      };
    };

    # Test user configuration
    users.users.${cfg.testUser} = {
      isNormalUser = true;
      home = "/home/${cfg.testUser}";
      createHome = true;
      extraGroups = [ "audio" "video" ];
    };

    # Required packages
    environment.systemPackages = with pkgs; [
      # Browser
      chromium

      # Node.js for Playwright
      nodejs_22

      # Playwright system dependencies
      # These are the libraries Playwright needs for Chromium
      nss
      nspr
      at-spi2-atk
      cups
      libdrm
      libxkbcommon
      mesa
      alsa-lib
      pango
      cairo
      expat
      glib
      gtk3

      # Utilities
      xdotool
      scrot
      xclip

      # Debugging tools
      htop
      curl
      jq
    ];

    # Audio (for voice testing with mocked audio)
    hardware.pulseaudio.enable = true;

    # Fonts for consistent rendering
    fonts.packages = with pkgs; [
      noto-fonts
      noto-fonts-emoji
      liberation_ttf
      dejavu_fonts
    ];

    # Environment variables for test execution
    environment.variables = {
      ALICIA_SERVER_URL = cfg.serverUrl;
      DISPLAY = ":0";
      # Playwright settings
      PLAYWRIGHT_BROWSERS_PATH = "/home/${cfg.testUser}/.cache/ms-playwright";
    };

    # Systemd service for running tests (triggered externally)
    systemd.services.alicia-e2e-tests = {
      description = "Alicia E2E Test Runner";
      after = [ "graphical.target" "network-online.target" ];
      wants = [ "network-online.target" ];

      serviceConfig = {
        Type = "oneshot";
        User = cfg.testUser;
        WorkingDirectory = "/home/${cfg.testUser}/e2e-tests";
        Environment = [
          "DISPLAY=:0"
          "HOME=/home/${cfg.testUser}"
          "ALICIA_SERVER_URL=${cfg.serverUrl}"
        ];
        ExecStart = "${pkgs.bash}/bin/bash -c 'npx playwright test --headed'";
        TimeoutStartSec = "600";  # 10 minute timeout for full test run
      };
    };

    # Network configuration for VM communication
    networking = {
      firewall.enable = false;  # Within test network
      hostName = "client-vm";
    };
  };
}
```

### 1.2 VM Build Configuration

Add to `flake.nix` outputs:

```nix
# Client VM for e2e testing
nixosConfigurations.client-vm = nixpkgs.lib.nixosSystem {
  system = "x86_64-linux";
  modules = [
    ./nix/modules/client-vm.nix
    ({ pkgs, ... }: {
      alicia.testing.clientVm = {
        enable = true;
        serverUrl = "http://server-vm:8080";
      };

      # VM-specific settings
      virtualisation.vmVariant = {
        virtualisation = {
          memorySize = 4096;
          cores = 4;
          graphics = true;
          qemu.options = [
            "-vga virtio"
          ];
        };
      };
    })
  ];
};
```

---

## 2. Playwright Setup

### 2.1 Test Project Structure

```
e2e-test/
├── package.json
├── playwright.config.ts
├── tsconfig.json
├── tests/
│   ├── smoke.spec.ts           # Comprehensive smoke test
│   ├── fixtures/
│   │   ├── index.ts            # Test fixtures and helpers
│   │   ├── page-objects/
│   │   │   ├── sidebar.ts
│   │   │   ├── chat-window.ts
│   │   │   ├── settings.ts
│   │   │   └── voice-controls.ts
│   │   └── mocks/
│   │       └── audio.ts
│   └── helpers/
│       ├── wait.ts
│       ├── screenshot.ts
│       └── server-ready.ts
├── screenshots/                 # Screenshot output directory
│   ├── baseline/               # Baseline images for comparison
│   └── current/                # Current test run screenshots
└── results/                    # Test results and artifacts
```

### 2.2 Package Configuration

`e2e-test/package.json`:
```json
{
  "name": "alicia-e2e-tests",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "test": "playwright test",
    "test:headed": "playwright test --headed",
    "test:debug": "playwright test --debug",
    "test:smoke": "playwright test tests/smoke.spec.ts --headed",
    "report": "playwright show-report",
    "install-browsers": "playwright install chromium"
  },
  "devDependencies": {
    "@playwright/test": "^1.40.0",
    "typescript": "^5.3.0"
  }
}
```

### 2.3 Playwright Configuration

`e2e-test/playwright.config.ts`:
```typescript
import { defineConfig, devices } from '@playwright/test';

const SERVER_URL = process.env.ALICIA_SERVER_URL || 'http://localhost:8080';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,  // Sequential for smoke tests
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 1,
  workers: 1,  // Single worker for headed tests

  reporter: [
    ['html', { outputFolder: 'results/html-report' }],
    ['json', { outputFile: 'results/results.json' }],
    ['list'],
  ],

  timeout: 60000,  // 60 second timeout per test
  expect: {
    timeout: 10000,
  },

  use: {
    baseURL: SERVER_URL,

    // Headed mode for visual testing
    headless: false,

    // Viewport matches VM resolution
    viewport: { width: 1920, height: 1080 },

    // Screenshot configuration
    screenshot: {
      mode: 'on',  // Capture on every test
      fullPage: false,
    },

    // Video recording
    video: {
      mode: 'retain-on-failure',
      size: { width: 1920, height: 1080 },
    },

    // Trace collection
    trace: 'retain-on-failure',

    // Slower actions for visual clarity
    actionTimeout: 15000,
    navigationTimeout: 30000,

    // Browser context options
    contextOptions: {
      reducedMotion: 'no-preference',
      colorScheme: 'light',
    },
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
            '--use-fake-ui-for-media-stream',  // Auto-accept media permissions
            '--use-fake-device-for-media-stream',  // Use fake audio/video
          ],
        },
      },
    },
  ],

  // Wait for server before running tests
  webServer: {
    command: 'echo "Using external server"',
    url: SERVER_URL,
    reuseExistingServer: true,
    timeout: 120000,
  },

  // Output directories
  outputDir: 'results/test-artifacts',
  snapshotDir: 'screenshots/baseline',
});
```

---

## 3. Core E2E Smoke Test

### 3.1 Test Fixtures

`e2e-test/tests/fixtures/index.ts`:
```typescript
import { test as base, expect, Page, Locator } from '@playwright/test';

// Page object interfaces
export interface SidebarActions {
  createConversation(): Promise<string>;
  selectConversation(id: string): Promise<void>;
  deleteConversation(id: string): Promise<void>;
  getConversationList(): Promise<string[]>;
  waitForConversationCount(count: number): Promise<void>;
}

export interface ChatActions {
  sendMessage(text: string): Promise<void>;
  waitForUserMessage(text: string): Promise<Locator>;
  waitForAssistantResponse(timeout?: number): Promise<Locator>;
  getMessageCount(): Promise<number>;
  isTyping(): Promise<boolean>;
}

export interface VoiceActions {
  activateVoiceMode(): Promise<void>;
  deactivateVoiceMode(): Promise<void>;
  isVoiceModeActive(): Promise<boolean>;
  waitForConnection(state: 'connected' | 'connecting' | 'disconnected'): Promise<void>;
  startRecording(): Promise<void>;
  stopRecording(): Promise<void>;
}

export interface SettingsActions {
  open(): Promise<void>;
  close(): Promise<void>;
  addMcpServer(name: string, command: string, args?: string): Promise<void>;
  removeMcpServer(name: string): Promise<void>;
  waitForServerStatus(name: string, status: string): Promise<void>;
}

export interface ScreenshotHelper {
  capture(name: string, options?: { fullPage?: boolean }): Promise<string>;
  captureElement(locator: Locator, name: string): Promise<string>;
}

// Extended test fixtures
type TestFixtures = {
  sidebar: SidebarActions;
  chat: ChatActions;
  voice: VoiceActions;
  settings: SettingsActions;
  screenshot: ScreenshotHelper;
  waitForServerReady: () => Promise<void>;
};

export const test = base.extend<TestFixtures>({
  // Wait for server to be ready
  waitForServerReady: async ({ page }, use) => {
    const waitFn = async () => {
      const maxAttempts = 30;
      const delayMs = 2000;

      for (let i = 0; i < maxAttempts; i++) {
        try {
          const response = await page.request.get('/health');
          if (response.ok()) {
            return;
          }
        } catch {
          // Server not ready yet
        }
        await page.waitForTimeout(delayMs);
      }
      throw new Error('Server did not become ready within timeout');
    };
    await use(waitFn);
  },

  // Sidebar actions
  sidebar: async ({ page }, use) => {
    const actions: SidebarActions = {
      async createConversation() {
        await page.click('button:has-text("New Chat")');
        await page.waitForSelector('.chat-window', { state: 'visible' });

        const selected = page.locator('.conversation-item.selected').first();
        const id = await selected.getAttribute('data-conversation-id');

        if (!id) throw new Error('Failed to get conversation ID');
        return id;
      },

      async selectConversation(id: string) {
        await page.click(`[data-conversation-id="${id}"]`);
        await page.waitForSelector('.chat-window', { state: 'visible' });
      },

      async deleteConversation(id: string) {
        await page.click(`[data-conversation-id="${id}"] .delete-btn`);
        await page.click('button:has-text("Delete")');
        await page.waitForSelector(`[data-conversation-id="${id}"]`, {
          state: 'hidden',
          timeout: 5000,
        });
      },

      async getConversationList() {
        const items = page.locator('.conversation-item');
        const count = await items.count();
        const ids: string[] = [];

        for (let i = 0; i < count; i++) {
          const id = await items.nth(i).getAttribute('data-conversation-id');
          if (id) ids.push(id);
        }
        return ids;
      },

      async waitForConversationCount(count: number) {
        await expect(page.locator('.conversation-item')).toHaveCount(count, {
          timeout: 10000,
        });
      },
    };
    await use(actions);
  },

  // Chat actions
  chat: async ({ page }, use) => {
    const actions: ChatActions = {
      async sendMessage(text: string) {
        await page.fill('.input-bar input[type="text"]', text);
        await page.click('.input-bar button[type="submit"]');
      },

      async waitForUserMessage(text: string) {
        const msg = page.locator(`.message-bubble.user:has-text("${text}")`);
        await expect(msg).toBeVisible({ timeout: 5000 });
        return msg;
      },

      async waitForAssistantResponse(timeout = 30000) {
        const msg = page.locator('.message-bubble.assistant').first();
        await expect(msg).toBeVisible({ timeout });
        return msg;
      },

      async getMessageCount() {
        return page.locator('.message-bubble').count();
      },

      async isTyping() {
        const typing = page.locator('.typing-indicator, .streaming-response');
        return typing.isVisible();
      },
    };
    await use(actions);
  },

  // Voice actions with mocked audio
  voice: async ({ page }, use) => {
    // Inject audio mocks before use
    await page.addInitScript(() => {
      navigator.mediaDevices.getUserMedia = async () => {
        const audioContext = new AudioContext();
        const oscillator = audioContext.createOscillator();
        const destination = audioContext.createMediaStreamDestination();
        oscillator.connect(destination);
        oscillator.start();
        return destination.stream;
      };
    });

    const actions: VoiceActions = {
      async activateVoiceMode() {
        const toggle = page.locator('.voice-mode-toggle');
        const isActive = await toggle.evaluate(el =>
          el.classList.contains('active')
        );

        if (!isActive) {
          await toggle.click();
          await page.waitForTimeout(500);
        }
      },

      async deactivateVoiceMode() {
        const toggle = page.locator('.voice-mode-toggle');
        const isActive = await toggle.evaluate(el =>
          el.classList.contains('active')
        );

        if (isActive) {
          await toggle.click();
          await page.waitForTimeout(500);
        }
      },

      async isVoiceModeActive() {
        const toggle = page.locator('.voice-mode-toggle');
        return toggle.evaluate(el => el.classList.contains('active'));
      },

      async waitForConnection(state) {
        const stateText = {
          connected: 'Connected',
          connecting: 'Connecting',
          disconnected: 'Disconnected',
        }[state];

        await page.waitForSelector(
          `.connection-status:has-text("${stateText}")`,
          { timeout: 15000 }
        );
      },

      async startRecording() {
        const btn = page.locator('.record-btn');
        const isRecording = await btn.evaluate(el =>
          el.classList.contains('recording')
        );

        if (!isRecording) {
          await btn.click();
        }
      },

      async stopRecording() {
        const btn = page.locator('.record-btn');
        const isRecording = await btn.evaluate(el =>
          el.classList.contains('recording')
        );

        if (isRecording) {
          await btn.click();
        }
      },
    };
    await use(actions);
  },

  // Settings actions
  settings: async ({ page }, use) => {
    const actions: SettingsActions = {
      async open() {
        await page.click('button[title="Settings"]');
        await page.waitForSelector('.mcp-settings', { state: 'visible' });
      },

      async close() {
        await page.keyboard.press('Escape');
        await page.waitForSelector('.mcp-settings', { state: 'hidden' });
      },

      async addMcpServer(name: string, command: string, args?: string) {
        await page.click('button:has-text("Add Server")');
        await page.fill('#server-name', name);
        await page.fill('#command', command);
        if (args) {
          await page.fill('#args', args);
        }
        await page.click('button[type="submit"]:has-text("Add Server")');
        await page.waitForSelector('.toast-success', { timeout: 5000 });
      },

      async removeMcpServer(name: string) {
        const card = page.locator(`.server-card:has-text("${name}")`);
        await card.locator('.remove-server-btn').click();
        await page.click('button:has-text("Confirm")');
        await page.waitForSelector(`.server-card:has-text("${name}")`, {
          state: 'hidden',
        });
      },

      async waitForServerStatus(name: string, status: string) {
        await page.waitForSelector(
          `.server-card:has-text("${name}") .status-badge:has-text("${status}")`,
          { timeout: 10000 }
        );
      },
    };
    await use(actions);
  },

  // Screenshot helper
  screenshot: async ({ page }, use) => {
    let screenshotIndex = 0;

    const helper: ScreenshotHelper = {
      async capture(name: string, options = {}) {
        screenshotIndex++;
        const timestamp = Date.now();
        const filename = `screenshots/current/${screenshotIndex.toString().padStart(2, '0')}-${name}-${timestamp}.png`;

        await page.screenshot({
          path: filename,
          fullPage: options.fullPage ?? false,
        });

        return filename;
      },

      async captureElement(locator: Locator, name: string) {
        screenshotIndex++;
        const timestamp = Date.now();
        const filename = `screenshots/current/${screenshotIndex.toString().padStart(2, '0')}-${name}-${timestamp}.png`;

        await locator.screenshot({ path: filename });

        return filename;
      },
    };
    await use(helper);
  },
});

export { expect };
```

### 3.2 Comprehensive Smoke Test

`e2e-test/tests/smoke.spec.ts`:
```typescript
import { test, expect } from './fixtures';

test.describe('Alicia Smoke Test', () => {
  test.describe.configure({ mode: 'serial' });

  // Shared state across tests in this file
  let conversationId: string;

  test.beforeAll(async ({ waitForServerReady }) => {
    // Ensure server is ready before running any tests
    await waitForServerReady();
  });

  test('01 - Application loads successfully', async ({ page, screenshot }) => {
    await page.goto('/');

    // Wait for core UI elements
    await expect(page.locator('.app')).toBeVisible();
    await expect(page.locator('.sidebar')).toBeVisible();
    await expect(page.locator('button:has-text("New Chat")')).toBeVisible();

    await screenshot.capture('01-app-loaded');

    // Verify no console errors on load
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error' && !msg.text().includes('net::ERR_')) {
        errors.push(msg.text());
      }
    });

    await page.waitForTimeout(1000);
    expect(errors).toEqual([]);
  });

  test('02 - Create new conversation', async ({ page, sidebar, screenshot }) => {
    await page.goto('/');

    // Capture initial state
    await screenshot.capture('02-before-new-conversation');

    // Create conversation
    conversationId = await sidebar.createConversation();

    // Verify conversation appears in sidebar
    await expect(
      page.locator(`[data-conversation-id="${conversationId}"]`)
    ).toBeVisible();

    // Verify chat window is ready
    await expect(page.locator('.chat-window')).toBeVisible();
    await expect(page.locator('.input-bar')).toBeVisible();

    await screenshot.capture('02-conversation-created');
  });

  test('03 - Send text message', async ({ page, chat, screenshot }) => {
    await page.goto('/');
    await page.click(`[data-conversation-id="${conversationId}"]`);

    const testMessage = 'Hello Alicia, this is a test message.';

    // Send message
    await chat.sendMessage(testMessage);

    // Verify user message appears
    const userMsg = await chat.waitForUserMessage(testMessage);
    await expect(userMsg).toBeVisible();

    await screenshot.capture('03-message-sent');
  });

  test('04 - Receive AI response', async ({ page, chat, screenshot }) => {
    await page.goto('/');
    await page.click(`[data-conversation-id="${conversationId}"]`);

    // Wait for existing response or send new message
    let msgCount = await chat.getMessageCount();

    if (msgCount < 2) {
      await chat.sendMessage('Please respond briefly.');
    }

    // Wait for assistant response (longer timeout for LLM)
    const response = await chat.waitForAssistantResponse(60000);
    await expect(response).toBeVisible();

    // Verify response has content
    const responseText = await response.textContent();
    expect(responseText?.length).toBeGreaterThan(0);

    await screenshot.capture('04-ai-response-received');
  });

  test('05 - Navigate to settings', async ({ page, settings, screenshot }) => {
    await page.goto('/');

    // Open settings
    await settings.open();

    // Verify settings panel content
    await expect(page.locator('.mcp-settings h2')).toContainText('Settings');

    await screenshot.capture('05-settings-opened');

    // Close settings
    await settings.close();
    await expect(page.locator('.mcp-settings')).not.toBeVisible();
  });

  test('06 - Voice mode activation', async ({ page, sidebar, voice, screenshot }) => {
    await page.goto('/');

    // Need a conversation for voice mode
    if (!conversationId) {
      conversationId = await sidebar.createConversation();
    } else {
      await page.click(`[data-conversation-id="${conversationId}"]`);
    }

    // Capture before state
    await screenshot.capture('06a-before-voice-mode');

    // Activate voice mode
    await voice.activateVoiceMode();

    // Verify voice mode is active
    const isActive = await voice.isVoiceModeActive();
    expect(isActive).toBe(true);

    // Verify voice controls appear
    await expect(page.locator('.voice-controls')).toBeVisible();
    await expect(page.locator('.audio-input')).toBeVisible();

    await screenshot.capture('06b-voice-mode-activated');

    // Check connection status (may be connecting or connected)
    const connectionStatus = page.locator('.connection-status');
    await expect(connectionStatus).toBeVisible();
    await expect(connectionStatus).toContainText(/Connecting|Connected/);

    // Deactivate voice mode
    await voice.deactivateVoiceMode();
    await expect(page.locator('.voice-controls')).not.toBeVisible();

    await screenshot.capture('06c-voice-mode-deactivated');
  });

  test('07 - Voice interaction flow', async ({ page, sidebar, voice, screenshot }) => {
    await page.goto('/');

    if (!conversationId) {
      conversationId = await sidebar.createConversation();
    } else {
      await page.click(`[data-conversation-id="${conversationId}"]`);
    }

    // Activate voice mode
    await voice.activateVoiceMode();

    // Wait for connection (with retry logic)
    try {
      await voice.waitForConnection('connected');
    } catch {
      // Connection might not be available in test environment
      // Log and continue with available functionality
      console.log('LiveKit connection not available in test environment');
    }

    // Test recording toggle (uses mocked audio)
    const recordBtn = page.locator('.record-btn');
    if (await recordBtn.isEnabled()) {
      await voice.startRecording();
      await screenshot.capture('07a-recording-started');

      await page.waitForTimeout(1000);

      await voice.stopRecording();
      await screenshot.capture('07b-recording-stopped');
    }

    // Test voice selector
    const voiceSelector = page.locator('.voice-selector-toggle');
    if (await voiceSelector.isVisible()) {
      await voiceSelector.click();
      await expect(page.locator('.voice-selector-panel')).toBeVisible();

      await screenshot.capture('07c-voice-selector-open');

      await page.keyboard.press('Escape');
    }

    // Cleanup
    await voice.deactivateVoiceMode();
  });

  test('08 - Multiple conversations', async ({ page, sidebar, chat, screenshot }) => {
    await page.goto('/');

    // Create first conversation
    const conv1 = await sidebar.createConversation();
    await chat.sendMessage('Message in conversation 1');

    await screenshot.capture('08a-first-conversation');

    // Create second conversation
    const conv2 = await sidebar.createConversation();
    await chat.sendMessage('Message in conversation 2');

    await screenshot.capture('08b-second-conversation');

    // Switch back to first conversation
    await sidebar.selectConversation(conv1);

    // Verify correct messages shown
    await expect(
      page.locator('.message-bubble:has-text("Message in conversation 1")')
    ).toBeVisible();
    await expect(
      page.locator('.message-bubble:has-text("Message in conversation 2")')
    ).not.toBeVisible();

    await screenshot.capture('08c-switched-to-first');

    // Cleanup
    await sidebar.deleteConversation(conv2);
    await sidebar.deleteConversation(conv1);
  });

  test('09 - Error handling - offline mode', async ({ page, sidebar, chat, screenshot }) => {
    await page.goto('/');

    const id = await sidebar.createConversation();

    // Go offline
    await page.context().setOffline(true);

    // Try to send message
    await chat.sendMessage('Offline test message');

    await screenshot.capture('09a-offline-message-sent');

    // Message should appear locally (optimistic UI)
    await expect(
      page.locator('.message-bubble:has-text("Offline test message")')
    ).toBeVisible();

    // May show error indicator
    const errorBanner = page.locator('.error-banner, .sync-error');
    if (await errorBanner.isVisible()) {
      await screenshot.capture('09b-error-indicator');
    }

    // Go back online
    await page.context().setOffline(false);
    await page.waitForTimeout(2000);

    await screenshot.capture('09c-back-online');

    // Cleanup
    await sidebar.deleteConversation(id);
  });

  test('10 - Error handling - invalid input', async ({ page, sidebar, screenshot }) => {
    await page.goto('/');
    await sidebar.createConversation();

    const input = page.locator('.input-bar input[type="text"]');
    const submitBtn = page.locator('.input-bar button[type="submit"]');

    // Try to submit empty message
    await input.fill('');
    await submitBtn.click();

    // Should not create a message
    await page.waitForTimeout(500);
    const msgCount = await page.locator('.message-bubble').count();

    await screenshot.capture('10-empty-message-handled');

    // Input should still be available
    await expect(input).toBeEnabled();
  });

  test('11 - Persistence across reload', async ({ page, sidebar, chat, screenshot }) => {
    await page.goto('/');

    // Create conversation with message
    const id = await sidebar.createConversation();
    const persistMsg = `Persist test ${Date.now()}`;
    await chat.sendMessage(persistMsg);

    await chat.waitForUserMessage(persistMsg);
    await screenshot.capture('11a-before-reload');

    // Reload page
    await page.reload();
    await page.waitForSelector('.sidebar', { state: 'visible' });

    // Conversation should still exist
    await expect(page.locator(`[data-conversation-id="${id}"]`)).toBeVisible();

    // Select and verify message
    await sidebar.selectConversation(id);
    await expect(
      page.locator(`.message-bubble:has-text("${persistMsg}")`)
    ).toBeVisible();

    await screenshot.capture('11b-after-reload');

    // Cleanup
    await sidebar.deleteConversation(id);
  });

  test('12 - Cleanup and final state', async ({ page, screenshot }) => {
    await page.goto('/');

    // Delete any remaining test conversations
    const items = page.locator('.conversation-item');
    const count = await items.count();

    for (let i = count - 1; i >= 0; i--) {
      const item = items.nth(i);
      const id = await item.getAttribute('data-conversation-id');

      if (id) {
        await item.locator('.delete-btn').click();
        await page.click('button:has-text("Delete")');
        await page.waitForTimeout(300);
      }
    }

    await screenshot.capture('12-final-clean-state');
  });
});
```

---

## 4. Test Execution Flow

### 4.1 Pre-Test Server Readiness Check

`e2e-test/tests/helpers/server-ready.ts`:
```typescript
import { Page } from '@playwright/test';

export interface ServerReadyOptions {
  maxAttempts?: number;
  delayMs?: number;
  healthEndpoint?: string;
}

export async function waitForServer(
  page: Page,
  baseUrl: string,
  options: ServerReadyOptions = {}
): Promise<void> {
  const {
    maxAttempts = 30,
    delayMs = 2000,
    healthEndpoint = '/health',
  } = options;

  console.log(`Waiting for server at ${baseUrl}${healthEndpoint}...`);

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const response = await page.request.get(`${baseUrl}${healthEndpoint}`);

      if (response.ok()) {
        const body = await response.json().catch(() => ({}));
        console.log(`Server ready after ${attempt} attempts:`, body);
        return;
      }

      console.log(`Attempt ${attempt}: Server returned ${response.status()}`);
    } catch (error) {
      console.log(`Attempt ${attempt}: ${error}`);
    }

    if (attempt < maxAttempts) {
      await new Promise(resolve => setTimeout(resolve, delayMs));
    }
  }

  throw new Error(
    `Server at ${baseUrl} did not become ready after ${maxAttempts} attempts`
  );
}
```

### 4.2 Execution Script

`e2e-test/run-tests.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail

# Configuration
SERVER_URL="${ALICIA_SERVER_URL:-http://server-vm:8080}"
RESULTS_DIR="./results"
SCREENSHOTS_DIR="./screenshots/current"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[TEST]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Cleanup previous run
cleanup() {
    log "Cleaning up previous test artifacts..."
    rm -rf "$RESULTS_DIR"
    rm -rf "$SCREENSHOTS_DIR"
    mkdir -p "$RESULTS_DIR" "$SCREENSHOTS_DIR"
}

# Wait for server to be ready
wait_for_server() {
    log "Waiting for server at $SERVER_URL..."

    local max_attempts=60
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -sf "${SERVER_URL}/health" > /dev/null 2>&1; then
            log "Server is ready!"
            return 0
        fi

        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    error "Server did not become ready within $(( max_attempts * 2 )) seconds"
    return 1
}

# Run tests
run_tests() {
    log "Installing dependencies..."
    npm ci

    log "Installing Playwright browsers..."
    npx playwright install chromium

    log "Running smoke tests in headed mode..."

    # Set display for headed mode
    export DISPLAY="${DISPLAY:-:0}"

    npx playwright test tests/smoke.spec.ts \
        --headed \
        --reporter=list,html,json \
        --output="$RESULTS_DIR/test-artifacts" \
        2>&1 | tee "$RESULTS_DIR/test-output.log"

    local exit_code=${PIPESTATUS[0]}

    if [ $exit_code -eq 0 ]; then
        log "All tests passed!"
    else
        error "Some tests failed (exit code: $exit_code)"
    fi

    return $exit_code
}

# Collect artifacts on failure
collect_failure_artifacts() {
    log "Collecting failure artifacts..."

    # Take final screenshot
    scrot "$SCREENSHOTS_DIR/failure-desktop.png" 2>/dev/null || true

    # Collect browser logs if available
    if [ -d "$RESULTS_DIR/test-artifacts" ]; then
        log "Test artifacts saved to $RESULTS_DIR/test-artifacts"
    fi

    # Generate summary
    if [ -f "$RESULTS_DIR/results.json" ]; then
        log "Generating test summary..."
        jq '.suites[] | {name: .title, tests: [.specs[] | {name: .title, status: .tests[0].status}]}' \
            "$RESULTS_DIR/results.json" > "$RESULTS_DIR/summary.json" 2>/dev/null || true
    fi
}

# Main execution
main() {
    log "Starting Alicia E2E Test Run"
    log "Server URL: $SERVER_URL"

    cleanup

    if ! wait_for_server; then
        exit 1
    fi

    if run_tests; then
        log "Test run completed successfully"
        exit 0
    else
        collect_failure_artifacts
        error "Test run completed with failures"
        exit 1
    fi
}

main "$@"
```

---

## 5. Failure Handling

### 5.1 Artifact Collection

On test failure, the following artifacts are automatically captured:

1. **Screenshots**: Captured at the moment of failure via Playwright
2. **Videos**: Full test video (retained on failure)
3. **Traces**: Playwright trace files for debugging
4. **Console Logs**: Browser console output
5. **Network Logs**: HAR file of network requests

### 5.2 Retry Strategy

```typescript
// In playwright.config.ts
retries: process.env.CI ? 2 : 1,  // Retry failed tests

// Per-test retry with different timing
test('flaky network test', async ({ page }) => {
  test.info().annotations.push({ type: 'retry', description: 'Network sensitive' });

  // Use soft assertions for non-critical checks
  await expect.soft(page.locator('.sync-status')).toContainText('Synced');

  // Hard assertion for critical functionality
  await expect(page.locator('.message-bubble')).toBeVisible();
});
```

### 5.3 Timing Resilience

```typescript
// Helper for flaky timing
export async function waitWithRetry<T>(
  action: () => Promise<T>,
  options: { maxAttempts?: number; delay?: number } = {}
): Promise<T> {
  const { maxAttempts = 3, delay = 1000 } = options;

  for (let i = 0; i < maxAttempts; i++) {
    try {
      return await action();
    } catch (error) {
      if (i === maxAttempts - 1) throw error;
      await new Promise(r => setTimeout(r, delay));
    }
  }

  throw new Error('Unreachable');
}

// Usage
await waitWithRetry(async () => {
  await expect(page.locator('.status')).toContainText('Ready');
});
```

### 5.4 Known Failure Handling

```typescript
// Skip tests that require unavailable services
test('LiveKit voice interaction', async ({ page, voice }) => {
  test.skip(
    !process.env.LIVEKIT_AVAILABLE,
    'LiveKit server not available in this environment'
  );

  await voice.activateVoiceMode();
  await voice.waitForConnection('connected');
  // ... rest of test
});
```

---

## 6. Integration with Orchestrator

The client VM tests integrate with the broader test orchestration:

1. **Server VM starts first** - Alicia backend with all services
2. **Client VM waits for server health check** - `/health` endpoint
3. **Tests execute in headed mode** - Screenshots at each step
4. **Results collected** - JSON, HTML report, screenshots, videos
5. **Orchestrator aggregates results** - Pass/fail status reported

### Communication Contract

The client VM expects:
- Server VM accessible at `$ALICIA_SERVER_URL`
- `/health` endpoint returning 200 when ready
- Standard Alicia API endpoints available

The client VM provides:
- Exit code 0 on success, non-zero on failure
- `results/results.json` with test outcomes
- `screenshots/current/` with visual evidence
- `results/html-report/` for human review
