# Client VM and E2E Testing: Comprehensive Design

## Executive Summary

This document provides a complete architectural design for the Alicia e2e testing infrastructure, focusing on the client VM that runs Playwright tests in a graphical NixOS environment. The design addresses VM architecture, test story design, implementation patterns, and artifact collection.

**Status:** Design document for implementation
**Target:** NixOS VM with XFCE running headed Playwright tests against Alicia server VM

---

## 1. VM Architecture

### 1.1 Core Design Principles

**Display Server Strategy:**
- Use X11 with XFCE (not Xvfb) for real graphical environment
- Enables actual screenshot capture (not just headless buffers)
- Supports VNC for desktop-level visibility during debugging
- Required for Playwright's headed mode with visual verification

**Network Topology:**
```
Host Machine
  └── QEMU Virtual Network (192.168.1.0/24)
      ├── server-vm (192.168.1.2)
      │   ├── PostgreSQL :5432
      │   ├── Backend :8888
      │   └── Nginx :80 (frontend + API proxy)
      └── client-vm (192.168.1.3)
          ├── XFCE Desktop
          ├── Chromium Browser
          └── Playwright Test Runner
```

**Resource Allocation:**
- Server VM: 4GB RAM, 2 cores (needs Postgres + Go backend + LLM connections)
- Client VM: 4GB RAM, 4 cores (needs X11 + Chromium + Playwright)
- Resolution: 1920x1080 (fixed for reproducible screenshots)

### 1.2 Nix Configuration Structure

```
e2e-test/nix/
├── default.nix          # Test orchestration (nixosTest)
├── client.nix           # Client VM module (XFCE + Playwright)
├── server.nix           # Server VM configuration
└── log-collector.nix    # Log collection script
```

### 1.3 Client VM Architecture

**System Layers:**
```
┌─────────────────────────────────────┐
│  Playwright Test Process            │
│  (Node.js, TypeScript)               │
├─────────────────────────────────────┤
│  Chromium Browser                    │
│  (headed mode, real window)          │
├─────────────────────────────────────┤
│  XFCE Desktop Environment            │
│  (xfce4-panel, xfwm4, thunar)        │
├─────────────────────────────────────┤
│  X11 Server (:0)                     │
│  (1920x1080, no blanking)            │
├─────────────────────────────────────┤
│  LightDM Display Manager             │
│  (autologin as 'test' user)          │
├─────────────────────────────────────┤
│  NixOS Base System                   │
│  (minimal, documentation disabled)   │
└─────────────────────────────────────┘
```

**Key Components:**

1. **Display Manager (LightDM):**
   - Autologin enabled for unattended execution
   - User: `test`
   - Session: `xfce`
   - No password required (test VM only)

2. **Desktop Environment (XFCE):**
   - Minimal installation (no unnecessary packages)
   - Window manager: xfwm4 (needed for Chromium windows)
   - No desktop icons or panel configuration needed
   - Purpose: Provide X11 environment for browser

3. **Browser (Chromium):**
   - Controlled by Playwright
   - Runs in headed mode (visible windows)
   - Fake audio devices for voice testing (`--use-fake-device-for-media-stream`)
   - No GPU (`--disable-gpu` for VM compatibility)

4. **Artifact Sharing:**
   - 9P virtfs mount: `/artifacts` (bidirectional)
   - Host path: `/tmp/alicia-e2e-artifacts`
   - Playwright writes: screenshots, traces, HAR files
   - Desktop screenshots: VNC captures via `scrot`

### 1.4 Critical Configuration Details

**X11 Display Configuration:**
```nix
services.xserver = {
  enable = true;
  resolutions = [{ x = 1920; y = 1080; }];

  # Prevent screen blanking during tests
  serverFlagsSection = ''
    Option "BlankTime" "0"
    Option "StandbyTime" "0"
    Option "SuspendTime" "0"
    Option "OffTime" "0"
  '';
};

virtualisation = {
  resolution = { x = 1920; y = 1080; };
  graphics = true;
  qemu.options = [ "-vga virtio" ];
};
```

**Playwright System Dependencies:**

Must be present for Chromium to launch:
- `nss`, `nspr` (Mozilla network security)
- `at-spi2-atk` (accessibility)
- `cups` (printing subsystem)
- `libdrm`, `mesa` (graphics)
- `libxkbcommon` (keyboard)
- `alsa-lib`, `pulseaudio` (audio)
- `pango`, `cairo`, `gtk3` (rendering)

**Test User Environment:**
```nix
users.users.test = {
  isNormalUser = true;
  home = "/home/test";
  createHome = true;
  extraGroups = [ "audio" "video" ];
  initialPassword = "";  # No password for test VM
};

environment.variables = {
  ALICIA_SERVER_URL = "http://server:80";
  DISPLAY = ":0";
  PLAYWRIGHT_BROWSERS_PATH = "/home/test/.cache/ms-playwright";
};
```

### 1.5 Test Execution Flow

```
┌──────────────────────────────────────────────────────────┐
│ 1. Boot Phase                                             │
│    - Start server VM (Postgres → Backend → Nginx)        │
│    - Start client VM (LightDM → XFCE → X11)              │
│    - Wait for graphical.target on both VMs               │
└──────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────┐
│ 2. Health Check Phase                                     │
│    - Server: curl http://server/health                   │
│    - Client: wait_for_x, wait_for_unit("display-manager")│
│    - Network: client pings server                        │
└──────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────┐
│ 3. Playwright Setup Phase                                │
│    - npm ci (install dependencies)                       │
│    - npx playwright install chromium --with-deps         │
│    - Create /artifacts directory structure               │
└──────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────┐
│ 4. Test Execution Phase                                  │
│    - Run: npx playwright test --headed                   │
│    - DISPLAY=:0 (X11 on client VM)                       │
│    - BASE_URL=http://server                              │
│    - ARTIFACT_DIR=/artifacts                             │
│    - Tests run sequentially (12 smoke tests)             │
└──────────────────────────────────────────────────────────┘
                          ↓
┌──────────────────────────────────────────────────────────┐
│ 5. Artifact Collection Phase                             │
│    - Browser screenshots: /artifacts/screenshots/browser/│
│    - Desktop screenshots: /artifacts/screenshots/desktop/│
│    - Console logs: /artifacts/logs/frontend-console.jsonl│
│    - Network logs: /artifacts/logs/network-requests.jsonl│
│    - Server logs: collect-logs (journalctl)              │
│    - Summary JSON with metadata                          │
└──────────────────────────────────────────────────────────┘
```

---

## 2. E2E Test Story Design

### 2.1 Test Story Philosophy

**Goals:**
1. **Smoke test coverage** - Verify core workflows work end-to-end
2. **Visual verification** - Headed mode allows seeing what users see
3. **Real-world simulation** - Test as close to production as possible
4. **Comprehensive logging** - Capture everything for post-mortem debugging

**Non-Goals:**
- Unit test coverage (handled by Jest/Vitest)
- Performance benchmarking (not e2e's purpose)
- Edge case exhaustion (focus on happy paths + critical errors)

### 2.2 Core User Journey

The test story follows a realistic user workflow with Alicia:

```
User Story: "Alex uses Alicia for the first time"

1. Discovery
   ├── Opens browser to http://localhost:8080
   ├── Sees empty state: "Welcome to Alicia"
   └── Observes UI elements: sidebar, new chat button

2. First Conversation (Text)
   ├── Clicks "New Chat"
   ├── Conversation appears in sidebar with timestamp
   ├── Enters text: "Hello Alicia, this is a test"
   ├── Clicks send
   ├── Sees message appear as user bubble
   ├── Waits for AI response (streaming or complete)
   └── Reads assistant's response

3. Settings Exploration
   ├── Opens settings panel
   ├── Views available voice options
   ├── Checks MCP server configuration
   └── Closes settings

4. Voice Mode Activation
   ├── Enables voice mode toggle
   ├── Observes LiveKit connection state
   ├── Sees microphone permission request (auto-granted in test)
   ├── Voice controls become visible
   └── Can activate/deactivate recording

5. Voice Interaction Flow
   ├── Starts recording (button press)
   ├── "Speaks" (via fake audio device)
   ├── Stops recording
   ├── Transcription appears
   ├── Assistant responds with voice + text
   └── Audio plays (mocked in test environment)

6. Multi-Conversation Management
   ├── Creates second conversation
   ├── Sends different message to each
   ├── Switches between conversations
   ├── Verifies message isolation
   └── Deletes one conversation

7. Error Handling
   ├── Tests offline behavior (disconnect network)
   ├── Tests invalid input (empty message)
   ├── Verifies error UI appears
   └── Confirms graceful degradation

8. Persistence Verification
   ├── Reloads page
   ├── Verifies conversations persist
   ├── Confirms messages retained
   └── Checks selected conversation restored
```

### 2.3 Comprehensive Test Coverage Matrix

| Test Category | Test Case | Assertions | Async Handling | Edge Cases |
|---------------|-----------|------------|----------------|------------|
| **App Bootstrap** | Load homepage | UI elements visible | Wait for React mount | Console errors |
| **Conversation CRUD** | Create conversation | ID returned, sidebar updated | Wait for API response | Rapid clicking |
| | Select conversation | Chat window updates | Wait for message load | Empty conversation |
| | Delete conversation | Removed from sidebar | Wait for API deletion | Currently selected |
| **Text Messaging** | Send user message | Bubble appears | Immediate (optimistic) | Empty input |
| | Receive AI response | Assistant bubble appears | 60s timeout for LLM | Network failure |
| | Multiple messages | Order preserved | Sequential sends | Rapid fire |
| **Voice Mode** | Activate voice | Toggle state changes | LiveKit connection | No LiveKit server |
| | Recording flow | Button states update | Audio processing | Permission denied |
| | Voice selector | Voice changes | API update | Invalid voice |
| **Settings** | Open/close panel | Panel visibility | Animation complete | Click outside |
| | MCP server config | Server added/removed | Status polling | Invalid command |
| **Persistence** | Reload page | State restored | IndexedDB load | Cleared storage |
| **Error Handling** | Offline mode | Error notification | Network timeout | Retry logic |
| | Invalid input | Validation error | Immediate feedback | XSS attempt |

### 2.4 Async Operation Patterns

**Challenge:** AI responses take 5-60 seconds, LiveKit connections take 2-5 seconds.

**Solutions:**

1. **Dynamic Timeouts:**
```typescript
// Short timeout for UI updates
await expect(page.locator('.message-bubble.user')).toBeVisible({ timeout: 5000 });

// Long timeout for LLM responses
await expect(page.locator('.message-bubble.assistant')).toBeVisible({ timeout: 60000 });

// Variable timeout for LiveKit
const isNixosTest = !!process.env.NIXOS_TEST;
const connectionTimeout = isNixosTest ? 10000 : 5000;
```

2. **Polling Patterns:**
```typescript
// Wait for connection state
async waitForConnection(state: 'connected' | 'connecting' | 'disconnected') {
  await page.waitForFunction(
    (expectedState) => {
      const indicator = document.querySelector('[data-connection-state]');
      return indicator?.getAttribute('data-connection-state') === expectedState;
    },
    state,
    { timeout: 15000 }
  );
}
```

3. **Optimistic UI Handling:**
```typescript
// User message appears immediately (optimistic)
await chat.sendMessage('Hello');
await chat.waitForUserMessage('Hello'); // Fast

// Assistant response requires backend
await chat.waitForAssistantResponse(60000); // Slow
```

### 2.5 Voice/LiveKit Testing Strategy

**Reality Check:**
- Full LiveKit testing requires real audio streams
- External LiveKit server (livekit.decent.town) may be unavailable
- Test VMs have no real microphone

**Pragmatic Approach:**

1. **Connection Testing:**
   - Verify LiveKit client connects (or fails gracefully)
   - Check connection state indicators
   - Validate UI updates based on connection

2. **UI Testing:**
   - Voice mode toggle works
   - Recording button states (idle → recording → processing)
   - Voice selector displays and updates
   - Microphone permission flow

3. **Mocked Audio:**
   - Use `--use-fake-device-for-media-stream`
   - Chromium provides fake microphone
   - Tests voice UI without real audio

4. **Graceful Degradation:**
   - Test behavior when LiveKit unavailable
   - Error states displayed correctly
   - Fallback to text mode works

**Test Implementation:**
```typescript
test('Voice mode activation', async ({ page, voice }) => {
  await voice.activateVoiceMode();

  // Check UI state regardless of LiveKit availability
  const isActive = await voice.isVoiceModeActive();
  expect(isActive).toBe(true);

  // Try to wait for connection (may fail gracefully)
  try {
    await voice.waitForConnection('connected', { timeout: 10000 });
  } catch {
    // LiveKit may be unavailable in test environment
    console.log('LiveKit connection failed (expected in test VM)');
  }

  // UI should still be functional
  await expect(page.locator('[data-recording-button]')).toBeVisible();
});
```

---

## 3. Playwright Test Implementation Patterns

### 3.1 Fixture Architecture

**Design Pattern:** Page Object Model via Fixtures

```typescript
// lib/fixtures.ts
type TestFixtures = {
  sidebar: SidebarActions;      // Conversation list management
  chat: ChatActions;            // Message sending/receiving
  voice: VoiceActions;          // Voice mode control
  settings: SettingsActions;    // Settings panel
  artifacts: ArtifactCollector; // Screenshot/log capture
  step: StepHelper;             // Named step with auto-screenshot
};
```

**Benefits:**
- **Encapsulation:** Each fixture handles one UI domain
- **Reusability:** Shared across all tests
- **Maintainability:** UI changes localized to fixtures
- **Readability:** Tests read like user stories

### 3.2 Core Fixture Patterns

**Sidebar Actions:**
```typescript
interface SidebarActions {
  // Returns conversation ID for subsequent operations
  createConversation(): Promise<string>;

  // Switch to different conversation
  selectConversation(id: string): Promise<void>;

  // Delete with confirmation
  deleteConversation(id: string): Promise<void>;

  // Query conversation list
  getConversationList(): Promise<string[]>;

  // Wait for specific count (useful after create/delete)
  waitForConversationCount(count: number): Promise<void>;
}

// Implementation detail: Uses data-conversation-id attributes
```

**Chat Actions:**
```typescript
interface ChatActions {
  // Send text message
  sendMessage(text: string): Promise<void>;

  // Wait for user's own message (fast, optimistic UI)
  waitForUserMessage(text: string): Promise<Locator>;

  // Wait for AI response (slow, 60s timeout)
  waitForAssistantResponse(timeout?: number): Promise<Locator>;

  // Count messages in current conversation
  getMessageCount(): Promise<number>;

  // Check if assistant is typing/streaming
  isTyping(): Promise<boolean>;
}
```

**Voice Actions:**
```typescript
interface VoiceActions {
  // Toggle voice mode on
  activateVoiceMode(): Promise<void>;

  // Toggle voice mode off
  deactivateVoiceMode(): Promise<void>;

  // Query current state
  isVoiceModeActive(): Promise<boolean>;

  // Wait for LiveKit connection state
  waitForConnection(
    state: 'connected' | 'connecting' | 'disconnected',
    options?: { timeout?: number }
  ): Promise<void>;

  // Start voice recording
  startRecording(): Promise<void>;

  // Stop voice recording
  stopRecording(): Promise<void>;
}

// Implementation: May fail gracefully if LiveKit unavailable
```

**Settings Actions:**
```typescript
interface SettingsActions {
  // Open settings panel
  open(): Promise<void>;

  // Close settings panel
  close(): Promise<void>;

  // Add MCP server
  addMcpServer(
    name: string,
    command: string,
    args?: string
  ): Promise<void>;

  // Remove MCP server
  removeMcpServer(name: string): Promise<void>;

  // Wait for server to reach status
  waitForServerStatus(
    name: string,
    status: 'connected' | 'disconnected' | 'error'
  ): Promise<void>;
}
```

### 3.3 Artifact Collection Pattern

**Design:** Automatic capture throughout test lifecycle

```typescript
class ArtifactCollector {
  private screenshotCounter = 0;
  private consoleLogs: ConsoleLogEntry[] = [];
  private networkLogs: NetworkLogEntry[] = [];

  constructor(page: Page, testInfo: TestInfo, artifactDir: string) {
    this.setupConsoleCapture(); // Listen to console.* events
    this.setupNetworkCapture(); // Listen to request/response
  }

  // Numbered screenshots with descriptive names
  async screenshot(name: string): Promise<string> {
    const num = String(++this.screenshotCounter).padStart(3, '0');
    const filename = `${num}-${name}.png`;
    const path = `${artifactDir}/screenshots/browser/${filename}`;
    await page.screenshot({ path, fullPage: false });
    return path;
  }

  // DOM snapshot for debugging
  async captureDomSnapshot(name: string): Promise<void> {
    const html = await page.content();
    fs.writeFileSync(`${artifactDir}/dom/${name}.html`, html);
  }

  // Write logs to disk (JSONL format)
  async saveAllLogs(): Promise<void> {
    const consolePath = `${artifactDir}/logs/frontend-console.jsonl`;
    const networkPath = `${artifactDir}/logs/network-requests.jsonl`;

    // Each line is a JSON object
    fs.writeFileSync(
      consolePath,
      this.consoleLogs.map(log => JSON.stringify(log)).join('\n')
    );

    fs.writeFileSync(
      networkPath,
      this.networkLogs.map(log => JSON.stringify(log)).join('\n')
    );
  }

  // Get errors for assertions
  getConsoleErrors(): ConsoleLogEntry[] {
    return this.consoleLogs.filter(log => log.type === 'error');
  }

  getNetworkFailures(): NetworkLogEntry[] {
    return this.networkLogs.filter(log => log.type === 'failure');
  }
}
```

**Usage in Tests:**
```typescript
test('example test', async ({ page, chat, artifacts }) => {
  await page.goto('/');
  await artifacts.screenshot('initial-load');

  await chat.sendMessage('Hello');
  await artifacts.screenshot('message-sent');

  await chat.waitForAssistantResponse();
  await artifacts.screenshot('response-received');

  // Logs automatically saved by fixture cleanup
  // On failure, FailureHandler captures enhanced diagnostics
});
```

### 3.4 Failure Handling Pattern

**Design:** Enhanced diagnostics on test failure

```typescript
class FailureHandler {
  async captureAll(): Promise<void> {
    // 1. Full-page screenshot
    await page.screenshot({
      path: `${artifactDir}/screenshots/browser/failure-${testName}-full.png`,
      fullPage: true
    });

    // 2. Viewport screenshot
    await page.screenshot({
      path: `${artifactDir}/screenshots/browser/failure-${testName}-viewport.png`,
      fullPage: false
    });

    // 3. DOM snapshot with computed styles
    const html = await page.evaluate(() => {
      // Include all computed styles inline
      document.querySelectorAll('*').forEach(el => {
        const styles = window.getComputedStyle(el);
        el.setAttribute('data-computed-style', JSON.stringify(styles));
      });
      return document.documentElement.outerHTML;
    });
    fs.writeFileSync(`${artifactDir}/dom/failure-${testName}.html`, html);

    // 4. Accessibility tree
    const a11yTree = await page.accessibility.snapshot();
    fs.writeFileSync(
      `${artifactDir}/dom/failure-${testName}-a11y.json`,
      JSON.stringify(a11yTree, null, 2)
    );

    // 5. localStorage + sessionStorage
    const storage = await page.evaluate(() => ({
      local: Object.fromEntries(Object.entries(localStorage)),
      session: Object.fromEntries(Object.entries(sessionStorage))
    }));
    fs.writeFileSync(
      `${artifactDir}/dom/failure-${testName}-storage.json`,
      JSON.stringify(storage, null, 2)
    );

    // 6. VNC desktop screenshot (in NixOS VM)
    if (process.env.NIXOS_TEST) {
      // Trigger VNC screenshot via Python test script
      // Implementation in default.nix
    }
  }
}
```

### 3.5 Test Organization Pattern

**Serial Execution for State:**
```typescript
test.describe('Alicia Smoke Test', () => {
  // Run tests in order, share state
  test.describe.configure({ mode: 'serial' });

  let conversationId: string;

  test('01 - Create conversation', async ({ sidebar }) => {
    conversationId = await sidebar.createConversation();
  });

  test('02 - Send message', async ({ chat }) => {
    await chat.sendMessage('Hello');
    // Uses conversationId from previous test
  });

  test('03 - Receive response', async ({ chat }) => {
    await chat.waitForAssistantResponse();
  });

  // Tests run in sequence, building on previous state
});
```

**Numbered Test Names:**
- `01 - Application loads`
- `02 - Create new conversation`
- `03 - Send text message`
- Benefits: Clear order, easy to identify which test failed

### 3.6 Timeout Strategy

**Configuration:**
```typescript
// playwright.config.ts
export default defineConfig({
  timeout: isNixosTest ? 60000 : 30000,        // Per-test timeout
  expect: { timeout: isNixosTest ? 10000 : 5000 }, // Per-assertion timeout

  use: {
    actionTimeout: 15000,       // Per-action timeout (click, fill, etc.)
    navigationTimeout: 30000,   // Page navigation timeout
  },
});
```

**Rationale:**
- NixOS VM is slower (QEMU virtualization)
- LLM responses take 5-60 seconds (Qwen3-8B inference)
- LiveKit connection takes 2-10 seconds
- Network latency in VM environment

**In Tests:**
```typescript
// Fast operations (UI updates)
await expect(page.locator('.button')).toBeVisible({ timeout: 5000 });

// Slow operations (AI inference)
await chat.waitForAssistantResponse(60000);

// Variable timeout based on environment
const timeout = process.env.NIXOS_TEST ? 10000 : 5000;
await voice.waitForConnection('connected', { timeout });
```

---

## 4. Log Collection Strategy

### 4.1 Multi-Layer Logging Architecture

```
┌─────────────────────────────────────────────────────────┐
│ Browser Layer (Client VM)                               │
│ ─────────────────────────────────────────────────────── │
│ • Console logs (console.*, pageerror)                   │
│ • Network requests/responses (page.on('request/response'))│
│ • Playwright traces (--trace retain-on-failure)         │
│ • HAR files (recordHar in contextOptions)               │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ Server Layer (Server VM)                                │
│ ─────────────────────────────────────────────────────── │
│ • Backend logs (Go slog → journald)                     │
│ • PostgreSQL logs (log_statement = 'all')               │
│ • Nginx access/error logs                               │
│ • System errors (journalctl --priority=err)             │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ Artifact Storage (Shared /artifacts)                    │
│ ─────────────────────────────────────────────────────── │
│ • /artifacts/logs/                                      │
│ • /artifacts/screenshots/                               │
│ • /artifacts/traces/                                    │
│ • /artifacts/dom/                                       │
│ • /artifacts/summary.json                               │
└─────────────────────────────────────────────────────────┘
```

### 4.2 Frontend Log Collection

**Console Log Capture:**
```typescript
interface ConsoleLogEntry {
  timestamp: string;        // ISO 8601
  type: 'log' | 'info' | 'warn' | 'error' | 'debug';
  text: string;             // Message text
  location?: {              // Source location
    url: string;
    lineNumber: number;
    columnNumber: number;
  };
  args?: string[];          // Additional arguments
}

// Output format: JSONL (one JSON object per line)
// File: /artifacts/logs/frontend-console.jsonl
```

**Network Log Capture:**
```typescript
interface NetworkLogEntry {
  timestamp: string;
  type: 'request' | 'response' | 'failure';
  method: string;           // GET, POST, etc.
  url: string;              // Full URL
  status?: number;          // HTTP status code
  headers?: Record<string, string>;
  timing?: {
    startTime: number;
    responseTime?: number;  // Duration in ms
  };
  body?: string;            // For API calls (JSON)
  error?: string;           // For failures
}

// Output format: JSONL
// File: /artifacts/logs/network-requests.jsonl
```

**HAR Files:**
- Playwright's built-in HAR recording
- Full network traffic capture
- Includes request/response bodies
- Useful for debugging API issues
- File: `/artifacts/traces/har/test-name.har`

### 4.3 Backend Log Collection

**Go Structured Logging (slog):**
```json
{
  "time": "2025-12-24T20:00:00.000Z",
  "level": "INFO",
  "msg": "HTTP request",
  "method": "POST",
  "path": "/api/conversations",
  "duration_ms": 45,
  "status": 201,
  "request_id": "req-abc123"
}
```

**Collection Script (log-collector.nix):**
```bash
#!/bin/bash
# Runs on server VM to collect all logs

# 1. Backend logs (JSON from journald)
journalctl -u alicia --no-pager --output=json \
  > /artifacts/logs/backend.jsonl

# 2. Backend logs (text for grep)
journalctl -u alicia --no-pager \
  > /artifacts/logs/backend-stderr.log

# 3. PostgreSQL logs
cp /var/log/postgresql/postgresql.log \
   /artifacts/logs/postgresql.log

# 4. Nginx access logs
cp /var/log/nginx/access.log \
   /artifacts/logs/nginx-access.log

# 5. Nginx error logs
cp /var/log/nginx/error.log \
   /artifacts/logs/nginx-error.log

# 6. System errors (last hour)
journalctl --no-pager --since="1 hour ago" --priority=err \
  > /artifacts/logs/system-errors.log

# 7. Compress large files (>10MB)
for file in /artifacts/logs/*.log; do
  if [ $(stat -c%s "$file") -gt 10485760 ]; then
    gzip "$file"
  fi
done
```

### 4.4 Screenshot Collection

**Browser Screenshots (Playwright):**
```
/artifacts/screenshots/browser/
├── 001-01-app-loaded.png
├── 002-02-conversation-created.png
├── 003-03-message-sent.png
├── 004-04-response-received.png
├── ...
├── failure-test-name-full.png      (on failure)
└── failure-test-name-viewport.png  (on failure)
```

**Naming Convention:**
- Format: `{counter:03d}-{test-number}-{description}.png`
- Counter: Increments globally across all tests
- Test number: Matches test name (01, 02, etc.)
- Description: Short kebab-case description

**Desktop Screenshots (VNC via scrot):**
```
/artifacts/screenshots/desktop/
├── vnc-before-test-20251224-200000.png
├── vnc-on-failure-20251224-200130.png
└── vnc-final-state-20251224-200230.png
```

**Purpose:**
- Browser screenshots: In-page content (what Playwright sees)
- Desktop screenshots: Full desktop (XFCE, window decorations, etc.)
- Useful for debugging display issues, window manager problems

### 4.5 Trace and DOM Collection

**Playwright Traces:**
- Enabled: `trace: 'retain-on-failure'`
- Location: `/artifacts/traces/`
- Format: ZIP archive
- Contents: Screenshots, DOM snapshots, network, console, actions
- Viewing: `npx playwright show-trace trace.zip`

**DOM Snapshots:**
```
/artifacts/dom/
├── failure-test-name.html          # Full HTML with computed styles
├── failure-test-name-a11y.json     # Accessibility tree
└── failure-test-name-storage.json  # localStorage + sessionStorage
```

**Purpose:**
- Post-mortem debugging without browser
- Verify accessibility
- Inspect client-side storage state

### 4.6 Summary Metadata

**File:** `/artifacts/summary.json`

```json
{
  "timestamp": "2025-12-24T20:00:00.000Z",
  "status": "passed" | "failed",
  "duration_seconds": 180,
  "test_count": 12,
  "passed": 11,
  "failed": 1,
  "server": {
    "services": {
      "postgresql": "running",
      "alicia": "running",
      "nginx": "running"
    },
    "uptime_seconds": 200
  },
  "client": {
    "desktop": "xfce",
    "playwright_version": "1.40.0",
    "chromium_version": "120.0.6099.28"
  },
  "artifacts": {
    "screenshots_browser": 24,
    "screenshots_desktop": 3,
    "log_files": 7,
    "traces": 1,
    "dom_snapshots": 1
  },
  "failures": [
    {
      "test": "04 - Receive AI response",
      "error": "Timeout waiting for assistant response",
      "artifacts": [
        "failure-04-ai-response-full.png",
        "failure-04-ai-response-viewport.png",
        "failure-04-ai-response.html"
      ]
    }
  ]
}
```

**Purpose:**
- Quick overview of test run
- Links to failure artifacts
- System health snapshot
- Used by CI/CD for reporting

---

## 5. Implementation Checklist

### Phase 1: Fix Client VM Configuration
- [ ] Correct `serverFlagsSection` syntax in client.nix
- [ ] Verify all Playwright system dependencies present
- [ ] Test X11 display at 1920x1080
- [ ] Validate autologin works for test user
- [ ] Confirm /artifacts mount bidirectional

### Phase 2: Align Test Scripts
- [ ] Update default.nix paths to match actual test location
- [ ] Point to `/home/test/e2e-test/` not `/home/test/frontend/`
- [ ] Copy e2e-test directory into VM during build
- [ ] Set correct environment variables (ARTIFACT_DIR, BASE_URL)

### Phase 3: Enhance Test Story
- [ ] Review existing 12 smoke tests
- [ ] Add voice mode tests (with graceful LiveKit failure)
- [ ] Improve error handling tests
- [ ] Add accessibility checks
- [ ] Document expected test duration

### Phase 4: Improve Artifact Collection
- [ ] Verify screenshot numbering works
- [ ] Test VNC screenshot integration
- [ ] Validate log compression for large files
- [ ] Generate summary.json after test run
- [ ] Create artifact viewer script (HTML report)

### Phase 5: Documentation
- [ ] Update README with architecture details
- [ ] Document known limitations (LiveKit, voice)
- [ ] Add troubleshooting guide
- [ ] Create debugging runbook

---

## 6. Known Limitations and Mitigations

### Limitation 1: External LiveKit Dependency

**Issue:** Tests depend on `livekit.decent.town` which may be unavailable.

**Mitigation:**
- Voice tests check connection but don't fail if LiveKit unavailable
- Log warning: "LiveKit connection failed (expected in test VM)"
- UI tests verify controls work regardless of connection state
- Future: Run local LiveKit server in third VM

### Limitation 2: AI Response Latency

**Issue:** Qwen3-8B inference takes 5-60 seconds depending on load.

**Mitigation:**
- 60-second timeout for assistant responses
- Log warning if response takes >30 seconds
- Tests can skip AI response if timeout (test creation/deletion separately)
- Future: Use smaller model (Qwen3-1.5B) for faster test runs

### Limitation 3: VM Performance

**Issue:** QEMU VMs slower than bare metal, especially graphical.

**Mitigation:**
- Generous timeouts (2x bare metal)
- Sequential test execution (avoid parallelism)
- Disable unnecessary services (documentation, etc.)
- Use virtio for graphics (faster than emulated GPU)

### Limitation 4: Screenshot Reproducibility

**Issue:** Timing-dependent UI (animations, loading states) causes flaky screenshots.

**Mitigation:**
- Wait for specific elements (`toBeVisible()`) before screenshot
- Disable CSS animations in test builds
- Use viewport screenshots (not full-page) for consistency
- Visual regression testing is future work (not current goal)

### Limitation 5: Voice Audio Testing

**Issue:** Can't test actual audio playback/recording in VM.

**Mitigation:**
- Fake audio devices (`--use-fake-device-for-media-stream`)
- Test UI state, not audio quality
- Mock audio data for deterministic tests
- Real audio testing requires different approach (Android tests)

---

## 7. Future Enhancements

### 7.1 Local LiveKit Server
- Run LiveKit in third VM
- Eliminates external dependency
- Faster connection times
- Full voice flow testing

### 7.2 Visual Regression Testing
- Baseline screenshot comparison
- Detect unintended UI changes
- Use Percy, Chromatic, or custom tool
- Requires stable test environment first

### 7.3 Performance Benchmarking
- Measure page load times
- Track API response durations
- Monitor memory usage
- Lighthouse CI integration

### 7.4 Cross-Browser Testing
- Firefox support (via Playwright)
- Safari (via Playwright WebKit)
- Requires additional browser setup in VM

### 7.5 Parallel Test Execution
- Once tests are deterministic
- Reduce total run time
- Requires isolated test data (separate conversations)

### 7.6 Mobile Viewport Testing
- Test responsive design
- Simulate touch interactions
- Requires different viewport configs

---

## 8. Conclusion

This design provides a comprehensive architecture for e2e testing Alicia in a reproducible NixOS VM environment. The key innovations:

1. **Graphical VM** - Real desktop environment for headed browser testing
2. **Realistic Test Story** - Follows actual user workflows
3. **Comprehensive Logging** - Multi-layer artifact collection
4. **Graceful Degradation** - Tests work even with external service failures
5. **Maintainable Fixtures** - Page Object Model via Playwright fixtures

The implementation should prioritize fixing the client VM configuration, aligning the test script paths, and validating the core smoke tests run successfully. Enhanced voice testing and visual regression can be added incrementally.

**Next Steps:**
1. Fix client.nix and default.nix configurations
2. Run `nix build .#checks.x86_64-linux.e2e`
3. Debug any failures
4. Iterate on test coverage
5. Document learnings
