# E2E Test Artifact Collection System

## Overview

This document describes the artifact collection system for Alicia's end-to-end testing infrastructure. The system captures screenshots, logs, traces, and other diagnostic data from two NixOS VMs:

- **server**: Runs the Alicia backend, frontend (nginx), and PostgreSQL
- **client**: Runs XFCE desktop with Playwright for browser-based testing

**Design Decisions:**

- Playwright's native tracing and screenshot capabilities for in-browser captures
- VNC-based screenshots for full desktop context (beyond browser viewport)
- Structured JSON logging via Go's slog for machine-parseable backend logs
- journalctl-based log collection post-test for simplicity
- 9P virtfs for VM-to-host artifact sharing (native to QEMU/NixOS tests)
- Compression for large artifacts (traces, videos)

---

## Output Folder Structure

```
test-results/
├── {test-run-timestamp}/
│   ├── screenshots/
│   │   ├── browser/
│   │   │   ├── 001-app-loaded.png
│   │   │   ├── 002-conversation-created.png
│   │   │   ├── 003-message-sent.png
│   │   │   └── failure-{test-name}-{timestamp}.png
│   │   └── desktop/
│   │       ├── vnc-before-test.png
│   │       ├── vnc-on-failure.png
│   │       └── vnc-final-state.png
│   ├── logs/
│   │   ├── backend.jsonl              # Structured JSON lines
│   │   ├── backend-stderr.log         # Raw stderr capture
│   │   ├── frontend-console.jsonl     # Browser console events
│   │   ├── network-requests.jsonl     # HTTP request/response log
│   │   ├── postgres.log               # PostgreSQL logs
│   │   └── nginx-access.log           # Nginx access logs
│   ├── traces/
│   │   ├── trace-{test-name}.zip      # Playwright trace archives
│   │   └── har/
│   │       └── {test-name}.har        # Network HAR files
│   ├── dom/
│   │   └── failure-{test-name}.html   # DOM snapshot on failure
│   ├── videos/                        # Optional: full test recordings
│   │   └── {test-name}.webm
│   ├── report.html                    # Playwright HTML report
│   └── summary.json                   # Test run metadata
```

---

## Phase 1: Backend Structured Logging

**End state:** Backend produces structured JSON logs via slog, captured to file alongside journalctl.

### Log Configuration

The backend should transition from standard `log` package to `log/slog` for structured output.

**Integration contract:**

```go
// internal/config/logging.go
type LogConfig struct {
    Level      string // debug, info, warn, error
    Format     string // json, text
    Output     string // stdout, stderr, file path
    AddSource  bool   // include file:line in logs
}

// Log entry structure for JSON output
type LogEntry struct {
    Time    string         `json:"time"`
    Level   string         `json:"level"`
    Msg     string         `json:"msg"`
    Source  *LogSource     `json:"source,omitempty"`
    Attrs   map[string]any `json:"-"` // flattened into root
}

type LogSource struct {
    Function string `json:"function"`
    File     string `json:"file"`
    Line     int    `json:"line"`
}
```

**Environment variables:**

```bash
ALICIA_LOG_LEVEL=debug     # Capture everything in e2e
ALICIA_LOG_FORMAT=json     # Machine-parseable
ALICIA_LOG_OUTPUT=stdout   # journald captures this
```

**Key requirements:**

- Request correlation via trace IDs (X-Request-ID header)
- Structured error logging with stack traces
- HTTP request/response logging at debug level
- Database query logging at debug level

---

## Phase 2: NixOS Test VM Configuration

**End state:** Both VMs configured with logging, screenshot capabilities, and shared artifact directory.

### Server VM Configuration

```nix
# nix/tests/e2e.nix
{ pkgs, lib, ... }:

{
  name = "alicia-e2e";

  # Artifact directory shared between VMs and host
  hostPkgs = pkgs;

  nodes = {
    server = { config, pkgs, ... }: {
      imports = [ ../modules/alicia.nix ];

      # Enable the Alicia service with test configuration
      services.alicia = {
        enable = true;
        database.url = "postgres://alicia:alicia@localhost/alicia";
      };

      # PostgreSQL with logging
      services.postgresql = {
        enable = true;
        settings = {
          log_statement = "all";
          log_duration = true;
          log_min_duration_statement = 0;
          logging_collector = true;
          log_directory = "/var/log/postgresql";
          log_filename = "postgresql.log";
        };
        initialScript = pkgs.writeText "init.sql" ''
          CREATE USER alicia WITH PASSWORD 'alicia';
          CREATE DATABASE alicia OWNER alicia;
        '';
      };

      # Nginx logging
      services.nginx.appendHttpConfig = ''
        access_log /var/log/nginx/access.log combined;
        error_log /var/log/nginx/error.log info;
      '';

      # Systemd journal configuration for full capture
      services.journald.extraConfig = ''
        Storage=persistent
        MaxRetentionSec=1day
        MaxFileSec=1hour
        ForwardToConsole=no
        RateLimitBurst=10000
        RateLimitIntervalSec=1s
      '';

      # Memory and resources for testing
      virtualisation = {
        memorySize = 2048;
        cores = 2;
        # Share artifact directory with host
        sharedDirectories = {
          artifacts = {
            source = "/tmp/alicia-e2e-artifacts";
            target = "/artifacts";
          };
        };
      };

      # Firewall rules for inter-VM communication
      networking.firewall.allowedTCPPorts = [ 80 443 8080 5432 ];
    };

    client = { config, pkgs, ... }: {
      # XFCE desktop for visual testing
      services.xserver = {
        enable = true;
        desktopManager.xfce.enable = true;
        displayManager.lightdm.enable = true;
        displayManager.autoLogin = {
          enable = true;
          user = "test";
        };
      };

      # VNC server for remote screenshots
      services.x11vnc = {
        enable = true;
        auth = null;  # No auth for testing
        shared = true;
        viewOnly = false;
      };

      # Test user
      users.users.test = {
        isNormalUser = true;
        home = "/home/test";
        extraGroups = [ "video" "audio" ];
      };

      # Browser and test tools
      environment.systemPackages = with pkgs; [
        chromium
        firefox
        nodejs
        # Screenshot utilities
        scrot
        imagemagick
        # Network debugging
        tcpdump
        wireshark-cli
      ];

      virtualisation = {
        memorySize = 4096;  # Browser needs memory
        cores = 4;
        # Resolution for screenshots
        resolution = { x = 1920; y = 1080; };
        sharedDirectories = {
          artifacts = {
            source = "/tmp/alicia-e2e-artifacts";
            target = "/artifacts";
          };
          tests = {
            source = toString ../../frontend/e2e;
            target = "/tests";
          };
        };
      };

      networking.firewall.enable = false;
    };
  };
}
```

---

## Phase 3: Playwright Configuration for Artifact Collection

**End state:** Playwright configured to capture screenshots, traces, console logs, and network activity.

### Enhanced Playwright Configuration

```typescript
// frontend/e2e/playwright.e2e.config.ts
import { defineConfig, devices } from '@playwright/test';
import path from 'path';

const outputDir = process.env.ARTIFACT_DIR || './test-results';
const isCI = !!process.env.CI;
const isNixosTest = !!process.env.NIXOS_TEST;

export default defineConfig({
  testDir: './e2e',
  outputDir: path.join(outputDir, 'playwright-output'),

  // Longer timeouts for VM environment
  timeout: isNixosTest ? 60000 : 30000,
  expect: { timeout: isNixosTest ? 10000 : 5000 },

  // Sequential execution for deterministic artifact ordering
  fullyParallel: false,
  workers: 1,

  // Retry configuration
  retries: isCI ? 2 : 0,

  // Rich reporting
  reporter: [
    ['html', { outputFolder: path.join(outputDir, 'report'), open: 'never' }],
    ['json', { outputFile: path.join(outputDir, 'results.json') }],
    ['list'],
  ],

  use: {
    // Target the server VM
    baseURL: process.env.BASE_URL || 'http://server',

    // Screenshot configuration
    screenshot: {
      mode: 'on',  // Capture on every test
      fullPage: true,
    },

    // Trace configuration - always capture for debugging
    trace: 'on',

    // Video recording
    video: isNixosTest ? 'on' : 'retain-on-failure',

    // Viewport for consistent screenshots
    viewport: { width: 1920, height: 1080 },

    // Locale and timezone for reproducibility
    locale: 'en-US',
    timezoneId: 'UTC',

    // Permissions
    permissions: ['clipboard-read', 'clipboard-write'],

    // Browser context options
    contextOptions: {
      recordHar: {
        path: path.join(outputDir, 'traces/har'),
        mode: 'full',
        content: 'embed',
      },
    },
  },

  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // Chromium-specific settings for stability in VM
        launchOptions: {
          args: [
            '--disable-gpu',
            '--disable-dev-shm-usage',
            '--no-sandbox',
            '--disable-setuid-sandbox',
          ],
        },
      },
    },
  ],

  // No webServer config - server VM handles this
  webServer: isNixosTest ? undefined : {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: true,
    timeout: 120000,
  },
});
```

---

## Phase 4: Console and Network Log Collection

**End state:** All browser console output and network requests captured to structured log files.

### Artifact Collector Module

```typescript
// frontend/e2e/lib/artifact-collector.ts
import { Page, BrowserContext, TestInfo } from '@playwright/test';
import fs from 'fs';
import path from 'path';

export interface ConsoleLogEntry {
  timestamp: string;
  type: 'log' | 'info' | 'warn' | 'error' | 'debug';
  text: string;
  location?: {
    url: string;
    lineNumber: number;
    columnNumber: number;
  };
  args?: string[];
}

export interface NetworkLogEntry {
  timestamp: string;
  type: 'request' | 'response' | 'failure';
  method: string;
  url: string;
  status?: number;
  headers?: Record<string, string>;
  timing?: {
    startTime: number;
    responseTime?: number;
  };
  body?: string;
  error?: string;
}

export class ArtifactCollector {
  private consoleLogs: ConsoleLogEntry[] = [];
  private networkLogs: NetworkLogEntry[] = [];
  private screenshotCounter = 0;
  private artifactDir: string;

  constructor(
    private page: Page,
    private testInfo: TestInfo,
    baseDir?: string
  ) {
    this.artifactDir = baseDir || process.env.ARTIFACT_DIR || './test-results';
    this.setupConsoleCapture();
    this.setupNetworkCapture();
  }

  private setupConsoleCapture(): void {
    this.page.on('console', (msg) => {
      const entry: ConsoleLogEntry = {
        timestamp: new Date().toISOString(),
        type: msg.type() as ConsoleLogEntry['type'],
        text: msg.text(),
        location: msg.location() ? {
          url: msg.location().url,
          lineNumber: msg.location().lineNumber,
          columnNumber: msg.location().columnNumber,
        } : undefined,
      };

      // Capture argument values for complex logs
      if (msg.args().length > 0) {
        entry.args = msg.args().map(arg => {
          try {
            return arg.toString();
          } catch {
            return '[unserializable]';
          }
        });
      }

      this.consoleLogs.push(entry);
    });

    // Capture uncaught exceptions
    this.page.on('pageerror', (error) => {
      this.consoleLogs.push({
        timestamp: new Date().toISOString(),
        type: 'error',
        text: `Uncaught exception: ${error.message}`,
        args: [error.stack || ''],
      });
    });
  }

  private setupNetworkCapture(): void {
    this.page.on('request', (request) => {
      this.networkLogs.push({
        timestamp: new Date().toISOString(),
        type: 'request',
        method: request.method(),
        url: request.url(),
        headers: request.headers(),
        timing: { startTime: Date.now() },
      });
    });

    this.page.on('response', async (response) => {
      const entry: NetworkLogEntry = {
        timestamp: new Date().toISOString(),
        type: 'response',
        method: response.request().method(),
        url: response.url(),
        status: response.status(),
        headers: response.headers(),
      };

      // Capture response body for API calls (not static assets)
      if (response.url().includes('/api/') && response.status() !== 304) {
        try {
          const contentType = response.headers()['content-type'] || '';
          if (contentType.includes('application/json')) {
            entry.body = await response.text();
          }
        } catch {
          // Response body not available
        }
      }

      this.networkLogs.push(entry);
    });

    this.page.on('requestfailed', (request) => {
      this.networkLogs.push({
        timestamp: new Date().toISOString(),
        type: 'failure',
        method: request.method(),
        url: request.url(),
        error: request.failure()?.errorText,
      });
    });
  }

  /**
   * Take a numbered screenshot with descriptive name
   */
  async screenshot(name: string, options?: { fullPage?: boolean }): Promise<string> {
    this.screenshotCounter++;
    const paddedNum = String(this.screenshotCounter).padStart(3, '0');
    const filename = `${paddedNum}-${this.sanitizeFilename(name)}.png`;
    const filepath = path.join(
      this.artifactDir,
      'screenshots/browser',
      filename
    );

    await fs.promises.mkdir(path.dirname(filepath), { recursive: true });
    await this.page.screenshot({
      path: filepath,
      fullPage: options?.fullPage ?? true,
    });

    // Also attach to Playwright report
    await this.testInfo.attach(name, {
      path: filepath,
      contentType: 'image/png',
    });

    return filepath;
  }

  /**
   * Capture DOM snapshot (useful for debugging failures)
   */
  async captureDomSnapshot(name: string): Promise<string> {
    const filename = `${this.sanitizeFilename(name)}.html`;
    const filepath = path.join(this.artifactDir, 'dom', filename);

    await fs.promises.mkdir(path.dirname(filepath), { recursive: true });

    const html = await this.page.content();
    await fs.promises.writeFile(filepath, html, 'utf-8');

    await this.testInfo.attach(`${name}-dom`, {
      path: filepath,
      contentType: 'text/html',
    });

    return filepath;
  }

  /**
   * Save all collected logs to files
   */
  async saveAllLogs(): Promise<void> {
    const logsDir = path.join(this.artifactDir, 'logs');
    await fs.promises.mkdir(logsDir, { recursive: true });

    // Console logs
    const consolePath = path.join(logsDir, 'frontend-console.jsonl');
    await fs.promises.writeFile(
      consolePath,
      this.consoleLogs.map(log => JSON.stringify(log)).join('\n'),
      'utf-8'
    );

    // Network logs
    const networkPath = path.join(logsDir, 'network-requests.jsonl');
    await fs.promises.writeFile(
      networkPath,
      this.networkLogs.map(log => JSON.stringify(log)).join('\n'),
      'utf-8'
    );
  }

  /**
   * Capture failure artifacts (screenshot, DOM, enhanced logs)
   */
  async captureFailure(testName: string): Promise<void> {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const failureName = `failure-${this.sanitizeFilename(testName)}-${timestamp}`;

    // Full page screenshot
    await this.screenshot(failureName, { fullPage: true });

    // DOM snapshot
    await this.captureDomSnapshot(failureName);

    // Save logs immediately
    await this.saveAllLogs();
  }

  /**
   * Get filtered console errors for assertions
   */
  getConsoleErrors(): ConsoleLogEntry[] {
    return this.consoleLogs.filter(log => log.type === 'error');
  }

  /**
   * Get failed network requests
   */
  getNetworkFailures(): NetworkLogEntry[] {
    return this.networkLogs.filter(log => log.type === 'failure');
  }

  private sanitizeFilename(name: string): string {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9-]/g, '-')
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, '');
  }
}
```

---

## Phase 5: Test Fixtures with Artifact Collection

**End state:** Test fixtures automatically collect artifacts for every test.

```typescript
// frontend/e2e/fixtures/with-artifacts.ts
import { test as base, expect } from '@playwright/test';
import { ArtifactCollector } from '../lib/artifact-collector';

// Re-export existing fixtures
export * from '../fixtures';

interface ArtifactFixtures {
  artifacts: ArtifactCollector;
  step: (name: string, fn: () => Promise<void>) => Promise<void>;
}

export const test = base.extend<ArtifactFixtures>({
  artifacts: async ({ page }, use, testInfo) => {
    const collector = new ArtifactCollector(page, testInfo);

    // Initial screenshot
    await page.waitForLoadState('domcontentloaded');
    await collector.screenshot('initial-state');

    await use(collector);

    // Save logs after test
    await collector.saveAllLogs();

    // Capture failure artifacts if test failed
    if (testInfo.status !== testInfo.expectedStatus) {
      await collector.captureFailure(testInfo.title);
    }
  },

  // Step helper that automatically takes screenshots
  step: async ({ artifacts }, use) => {
    const stepFn = async (name: string, fn: () => Promise<void>): Promise<void> => {
      await fn();
      await artifacts.screenshot(name);
    };
    await use(stepFn);
  },
});

export { expect };
```

### Example Test Using Artifact Collection

```typescript
// frontend/e2e/conversation-with-artifacts.spec.ts
import { test, expect } from './fixtures/with-artifacts';

test.describe('Conversation with artifacts', () => {
  test('create and send message', async ({ page, artifacts, step, conversationHelpers }) => {
    await page.goto('/');
    await artifacts.screenshot('app-loaded');

    await step('create-conversation', async () => {
      await conversationHelpers.createConversation();
    });

    await step('send-message', async () => {
      const convId = await page.locator('.conversation-item.selected')
        .getAttribute('data-conversation-id');
      await conversationHelpers.sendMessage(convId!, 'Hello, Alicia!');
    });

    await step('verify-response', async () => {
      await expect(page.locator('.message-bubble.assistant')).toBeVisible({
        timeout: 30000,
      });
    });

    // Verify no console errors
    const errors = artifacts.getConsoleErrors();
    expect(errors.filter(e => !e.text.includes('favicon'))).toHaveLength(0);
  });
});
```

---

## Phase 6: VNC Desktop Screenshot Capture

**End state:** Full desktop screenshots captured via VNC for context beyond browser.

### VNC Screenshot Script

```bash
#!/usr/bin/env bash
# scripts/vnc-screenshot.sh
# Captures screenshot from VNC server

set -euo pipefail

VNC_HOST="${VNC_HOST:-client}"
VNC_PORT="${VNC_PORT:-5900}"
OUTPUT_DIR="${ARTIFACT_DIR:-/artifacts}/screenshots/desktop"
FILENAME="${1:-vnc-screenshot}"

mkdir -p "$OUTPUT_DIR"

# Use vncsnapshot or fbgrab depending on availability
if command -v vncsnapshot &> /dev/null; then
    vncsnapshot -passwd /dev/null "${VNC_HOST}:${VNC_PORT}" \
        "${OUTPUT_DIR}/${FILENAME}.png"
elif command -v scrot &> /dev/null; then
    # If running on the client VM directly
    DISPLAY=:0 scrot "${OUTPUT_DIR}/${FILENAME}.png"
else
    echo "No screenshot tool available" >&2
    exit 1
fi

echo "${OUTPUT_DIR}/${FILENAME}.png"
```

### NixOS Test Driver Integration

```nix
# nix/tests/e2e.nix (continued)
{
  testScript = { nodes, ... }: ''
    import json
    import os
    from datetime import datetime

    # Artifact directory setup
    artifact_dir = "/tmp/alicia-e2e-artifacts"
    os.makedirs(f"{artifact_dir}/screenshots/browser", exist_ok=True)
    os.makedirs(f"{artifact_dir}/screenshots/desktop", exist_ok=True)
    os.makedirs(f"{artifact_dir}/logs", exist_ok=True)
    os.makedirs(f"{artifact_dir}/traces", exist_ok=True)

    def screenshot_desktop(name: str):
        """Capture VNC screenshot from client desktop"""
        timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
        filename = f"{name}-{timestamp}"
        client.succeed(f"DISPLAY=:0 scrot /artifacts/screenshots/desktop/{filename}.png")
        return f"/artifacts/screenshots/desktop/{filename}.png"

    def collect_server_logs():
        """Collect all server-side logs"""
        # Backend logs from journald
        server.succeed(
            "journalctl -u alicia --no-pager --output=json > /artifacts/logs/backend.jsonl"
        )
        server.succeed(
            "journalctl -u alicia --no-pager > /artifacts/logs/backend-stderr.log"
        )

        # PostgreSQL logs
        server.succeed(
            "cp /var/log/postgresql/postgresql.log /artifacts/logs/ || true"
        )

        # Nginx logs
        server.succeed(
            "cp /var/log/nginx/access.log /artifacts/logs/nginx-access.log || true"
        )

    def run_playwright_tests():
        """Execute Playwright tests on client VM"""
        client.succeed(
            "cd /home/test && "
            "NIXOS_TEST=1 "
            "BASE_URL=http://server "
            "ARTIFACT_DIR=/artifacts "
            "npx playwright test --config=/tests/playwright.e2e.config.ts"
        )

    # Start VMs
    start_all()

    # Wait for services
    server.wait_for_unit("postgresql.service")
    server.wait_for_unit("alicia.service")
    server.wait_for_open_port(8080)
    server.wait_for_open_port(80)

    client.wait_for_x()
    client.wait_for_unit("display-manager.service")

    # Initial desktop screenshot
    screenshot_desktop("vnc-before-test")

    # Run tests
    try:
        run_playwright_tests()
    except Exception as e:
        # Capture failure state
        screenshot_desktop("vnc-on-failure")
        collect_server_logs()
        raise

    # Capture final state
    screenshot_desktop("vnc-final-state")
    collect_server_logs()

    # Generate summary
    with open(f"{artifact_dir}/summary.json", "w") as f:
        json.dump({
            "timestamp": datetime.now().isoformat(),
            "status": "completed",
            "artifacts": os.listdir(artifact_dir),
        }, f, indent=2)
  '';
}
```

---

## Phase 7: Backend Log Collection Script

**End state:** Backend logs collected in structured format with metadata.

### Log Collection Module

```nix
# nix/tests/lib/log-collector.nix
{ pkgs }:

pkgs.writeShellScriptBin "collect-logs" ''
  set -euo pipefail

  ARTIFACT_DIR="''${ARTIFACT_DIR:-/artifacts}"
  LOGS_DIR="$ARTIFACT_DIR/logs"

  mkdir -p "$LOGS_DIR"

  echo "Collecting backend logs..."

  # Alicia service logs (JSON format if configured)
  journalctl -u alicia \
    --no-pager \
    --output=json \
    > "$LOGS_DIR/backend.jsonl" 2>/dev/null || true

  # Alicia agent logs (if running)
  journalctl -u alicia-agent \
    --no-pager \
    --output=json \
    > "$LOGS_DIR/agent.jsonl" 2>/dev/null || true

  # PostgreSQL logs
  if [ -f /var/log/postgresql/postgresql.log ]; then
    cp /var/log/postgresql/postgresql.log "$LOGS_DIR/"
  fi

  # Nginx logs
  if [ -f /var/log/nginx/access.log ]; then
    cp /var/log/nginx/access.log "$LOGS_DIR/nginx-access.log"
  fi
  if [ -f /var/log/nginx/error.log ]; then
    cp /var/log/nginx/error.log "$LOGS_DIR/nginx-error.log"
  fi

  # System journal summary
  journalctl --no-pager --since="1 hour ago" \
    --priority=err \
    > "$LOGS_DIR/system-errors.log" 2>/dev/null || true

  # Compress if large
  for file in "$LOGS_DIR"/*.log "$LOGS_DIR"/*.jsonl; do
    if [ -f "$file" ] && [ $(stat -c%s "$file") -gt 10485760 ]; then
      gzip "$file"
    fi
  done

  echo "Logs collected to $LOGS_DIR"
  ls -la "$LOGS_DIR"
''
```

---

## Phase 8: Error Artifact Enrichment

**End state:** Additional diagnostic data captured on test failure.

### Failure Handler

```typescript
// frontend/e2e/lib/failure-handler.ts
import { Page, TestInfo } from '@playwright/test';
import { ArtifactCollector } from './artifact-collector';
import { exec } from 'child_process';
import { promisify } from 'util';
import path from 'path';
import fs from 'fs';

const execAsync = promisify(exec);

export class FailureHandler {
  constructor(
    private page: Page,
    private testInfo: TestInfo,
    private collector: ArtifactCollector,
    private artifactDir: string
  ) {}

  async captureAll(): Promise<void> {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const testName = this.sanitize(this.testInfo.title);

    await Promise.allSettled([
      this.captureFullPageScreenshot(testName, timestamp),
      this.captureViewportScreenshot(testName, timestamp),
      this.captureDomSnapshot(testName, timestamp),
      this.captureAccessibilityTree(testName, timestamp),
      this.captureLocalStorage(testName, timestamp),
      this.captureNetworkState(testName, timestamp),
      this.triggerVncScreenshot(testName, timestamp),
    ]);

    // Save all collected logs
    await this.collector.saveAllLogs();
  }

  private async captureFullPageScreenshot(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'screenshots/browser',
      `failure-${testName}-fullpage-${ts}.png`
    );
    await fs.promises.mkdir(path.dirname(filepath), { recursive: true });
    await this.page.screenshot({ path: filepath, fullPage: true });
    await this.testInfo.attach('failure-fullpage', {
      path: filepath,
      contentType: 'image/png',
    });
  }

  private async captureViewportScreenshot(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'screenshots/browser',
      `failure-${testName}-viewport-${ts}.png`
    );
    await this.page.screenshot({ path: filepath, fullPage: false });
    await this.testInfo.attach('failure-viewport', {
      path: filepath,
      contentType: 'image/png',
    });
  }

  private async captureDomSnapshot(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'dom',
      `failure-${testName}-${ts}.html`
    );
    await fs.promises.mkdir(path.dirname(filepath), { recursive: true });

    // Get complete HTML with inline styles
    const html = await this.page.evaluate(() => {
      // Clone document to avoid modifying the page
      const clone = document.documentElement.cloneNode(true) as HTMLElement;

      // Inline computed styles for key elements
      const importantElements = clone.querySelectorAll('*');
      importantElements.forEach(el => {
        const computed = window.getComputedStyle(el as Element);
        const important = ['display', 'visibility', 'opacity', 'position'];
        const styleStr = important
          .map(prop => `${prop}: ${computed.getPropertyValue(prop)}`)
          .join('; ');
        (el as HTMLElement).setAttribute('data-computed-style', styleStr);
      });

      return clone.outerHTML;
    });

    await fs.promises.writeFile(filepath, html, 'utf-8');
    await this.testInfo.attach('failure-dom', {
      path: filepath,
      contentType: 'text/html',
    });
  }

  private async captureAccessibilityTree(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'dom',
      `failure-${testName}-a11y-${ts}.json`
    );

    try {
      const snapshot = await this.page.accessibility.snapshot();
      await fs.promises.writeFile(
        filepath,
        JSON.stringify(snapshot, null, 2),
        'utf-8'
      );
      await this.testInfo.attach('failure-accessibility', {
        path: filepath,
        contentType: 'application/json',
      });
    } catch (e) {
      // Accessibility tree not always available
    }
  }

  private async captureLocalStorage(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'dom',
      `failure-${testName}-storage-${ts}.json`
    );

    const storage = await this.page.evaluate(() => ({
      localStorage: { ...localStorage },
      sessionStorage: { ...sessionStorage },
    }));

    await fs.promises.writeFile(
      filepath,
      JSON.stringify(storage, null, 2),
      'utf-8'
    );
  }

  private async captureNetworkState(testName: string, ts: string): Promise<void> {
    // Network state is captured via HAR in Playwright config
    // This captures pending requests at failure time
    const filepath = path.join(
      this.artifactDir,
      'dom',
      `failure-${testName}-pending-requests-${ts}.json`
    );

    const pending = await this.page.evaluate(() => {
      // @ts-ignore - accessing performance entries
      const entries = performance.getEntriesByType('resource');
      return entries.map(e => ({
        name: e.name,
        duration: e.duration,
        transferSize: (e as PerformanceResourceTiming).transferSize,
      }));
    });

    await fs.promises.writeFile(
      filepath,
      JSON.stringify(pending, null, 2),
      'utf-8'
    );
  }

  private async triggerVncScreenshot(testName: string, ts: string): Promise<void> {
    if (!process.env.NIXOS_TEST) return;

    try {
      await execAsync(
        `vnc-screenshot.sh failure-${testName}-${ts}`,
        { env: { ...process.env } }
      );
    } catch (e) {
      // VNC screenshot is best-effort
    }
  }

  private sanitize(name: string): string {
    return name.toLowerCase().replace(/[^a-z0-9]/g, '-').substring(0, 50);
  }
}
```

---

## Phase 9: Complete NixOS Test Integration

**End state:** Full NixOS VM test with artifact collection working end-to-end.

### Complete Test Definition

```nix
# nix/tests/e2e.nix
{ pkgs, lib, self, ... }:

let
  artifactDir = "/tmp/alicia-e2e-artifacts";

  logCollector = import ./lib/log-collector.nix { inherit pkgs; };
in
pkgs.nixosTest {
  name = "alicia-e2e";

  nodes = {
    server = { config, pkgs, ... }: {
      imports = [ self.nixosModules.default ];

      services.alicia = {
        enable = true;
        mode = "both";
        host = "0.0.0.0";
        port = 8080;

        database.url = "postgres://alicia:alicia@localhost/alicia";
        database.autoMigrate = true;

        llm = {
          url = "http://localhost:8000/v1";  # Mock LLM for testing
          model = "test-model";
        };
      };

      services.postgresql = {
        enable = true;
        settings = {
          log_statement = "all";
          log_duration = true;
          log_min_duration_statement = 0;
        };
        ensureUsers = [{
          name = "alicia";
          ensureDBOwnership = true;
        }];
        ensureDatabases = [ "alicia" ];
        authentication = ''
          local all all trust
          host all all 127.0.0.1/32 trust
        '';
      };

      # Mock LLM server for testing
      systemd.services.mock-llm = {
        wantedBy = [ "multi-user.target" ];
        before = [ "alicia.service" ];
        script = ''
          ${pkgs.python3}/bin/python3 << 'EOF'
          from http.server import HTTPServer, BaseHTTPRequestHandler
          import json

          class MockLLM(BaseHTTPRequestHandler):
              def do_POST(self):
                  self.send_response(200)
                  self.send_header('Content-Type', 'application/json')
                  self.end_headers()
                  response = {
                      "choices": [{
                          "message": {
                              "role": "assistant",
                              "content": "This is a mock response for testing."
                          }
                      }]
                  }
                  self.wfile.write(json.dumps(response).encode())

          HTTPServer(('', 8000), MockLLM).serve_forever()
          EOF
        '';
      };

      environment.systemPackages = [ logCollector ];

      virtualisation = {
        memorySize = 2048;
        cores = 2;
        sharedDirectories.artifacts = {
          source = artifactDir;
          target = "/artifacts";
        };
      };

      networking.firewall.allowedTCPPorts = [ 80 8080 5432 8000 ];
    };

    client = { config, pkgs, ... }: {
      services.xserver = {
        enable = true;
        desktopManager.xfce.enable = true;
        displayManager.lightdm.enable = true;
        displayManager.autoLogin = {
          enable = true;
          user = "test";
        };
      };

      users.users.test = {
        isNormalUser = true;
        home = "/home/test";
      };

      environment.systemPackages = with pkgs; [
        chromium
        nodejs_22
        scrot
      ];

      virtualisation = {
        memorySize = 4096;
        cores = 4;
        resolution = { x = 1920; y = 1080; };
        sharedDirectories = {
          artifacts = {
            source = artifactDir;
            target = "/artifacts";
          };
          frontend = {
            source = toString ../../frontend;
            target = "/frontend";
          };
        };
      };

      networking.firewall.enable = false;
    };
  };

  testScript = ''
    import json
    import os
    from datetime import datetime

    # Ensure artifact directory exists on host
    os.makedirs("${artifactDir}", exist_ok=True)
    os.makedirs("${artifactDir}/screenshots/browser", exist_ok=True)
    os.makedirs("${artifactDir}/screenshots/desktop", exist_ok=True)
    os.makedirs("${artifactDir}/logs", exist_ok=True)
    os.makedirs("${artifactDir}/traces", exist_ok=True)
    os.makedirs("${artifactDir}/dom", exist_ok=True)

    def vnc_screenshot(name):
        """Capture desktop screenshot"""
        ts = datetime.now().strftime("%Y%m%d-%H%M%S")
        path = f"/artifacts/screenshots/desktop/{name}-{ts}.png"
        client.succeed(f"DISPLAY=:0 scrot {path}")
        print(f"Screenshot saved: {path}")

    def collect_all_logs():
        """Collect logs from server"""
        server.succeed("collect-logs")

    start_all()

    # Wait for server services
    with subtest("Server services start"):
        server.wait_for_unit("postgresql.service")
        server.wait_for_unit("mock-llm.service")
        server.wait_for_unit("alicia.service")
        server.wait_for_open_port(8080)

    # Wait for client desktop
    with subtest("Client desktop starts"):
        client.wait_for_x()
        client.wait_for_unit("display-manager.service")

    vnc_screenshot("initial-state")

    # Verify backend health
    with subtest("Backend health check"):
        server.succeed("curl -sf http://localhost:8080/health")

    # Install Playwright on client
    with subtest("Install Playwright"):
        client.succeed(
            "cd /frontend && npm ci && npx playwright install chromium --with-deps"
        )

    vnc_screenshot("playwright-installed")

    # Run e2e tests
    with subtest("Run Playwright tests"):
        try:
            client.succeed(
                "cd /frontend && "
                "NIXOS_TEST=1 "
                "BASE_URL=http://server:8080 "
                "ARTIFACT_DIR=/artifacts "
                "npx playwright test e2e/conversation.spec.ts "
                "--config=e2e/playwright.e2e.config.ts "
                "--reporter=html,json "
                "2>&1 | tee /artifacts/logs/playwright-output.log"
            )
        except Exception as e:
            vnc_screenshot("test-failure")
            collect_all_logs()
            raise

    vnc_screenshot("tests-complete")
    collect_all_logs()

    # Generate summary
    summary = {
        "timestamp": datetime.now().isoformat(),
        "status": "passed",
    }
    with open("${artifactDir}/summary.json", "w") as f:
        json.dump(summary, f, indent=2)
  '';
}
```

---

## Phase 10: Flake Integration

**End state:** E2E test runnable via `nix flake check` or `nix build .#checks.x86_64-linux.e2e`.

```nix
# In flake.nix, add to checks:
checks = {
  # ... existing checks ...

  e2e = import ./nix/tests/e2e.nix {
    inherit pkgs lib self;
  };
};
```

### Running the Tests

```bash
# Run e2e tests
nix build .#checks.x86_64-linux.e2e

# View artifacts
ls /tmp/alicia-e2e-artifacts/

# Or with explicit artifact directory
ARTIFACT_DIR=/path/to/output nix build .#checks.x86_64-linux.e2e
```

---

## Summary

| Component | Location | Format |
|-----------|----------|--------|
| Browser screenshots | `screenshots/browser/` | PNG |
| Desktop screenshots | `screenshots/desktop/` | PNG |
| Backend logs | `logs/backend.jsonl` | JSON Lines |
| Frontend console | `logs/frontend-console.jsonl` | JSON Lines |
| Network requests | `logs/network-requests.jsonl` | JSON Lines |
| Playwright traces | `traces/*.zip` | ZIP archive |
| HAR files | `traces/har/*.har` | HAR JSON |
| DOM snapshots | `dom/*.html` | HTML |
| Test report | `report.html` | HTML |
| Run summary | `summary.json` | JSON |

**Key integration points:**

1. `ArtifactCollector` class: instantiated per-test, captures console/network
2. `FailureHandler` class: captures enriched diagnostics on failure
3. `collect-logs` script: run on server VM to gather system logs
4. NixOS `sharedDirectories`: provides `/artifacts` mount point
5. Playwright config: enables tracing, screenshots, HAR recording
