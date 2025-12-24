# E2E Screenshot and Snapshot System Architecture

## Executive Summary

This document provides a comprehensive analysis and design for the screenshot and snapshot capture system in Alicia's e2e testing infrastructure. The system uses a dual-capture approach: Playwright-based browser screenshots for UI-level verification and VNC-based desktop snapshots for VM-level debugging, all running in a QEMU/microVM with XFCE desktop.

**Key Design Decisions:**
- **Dual-layer capture**: Browser screenshots (Playwright) + Desktop snapshots (VNC/scrot)
- **Numbered sequencing**: `001-step-name.png`, `002-next-step.png` for chronological clarity
- **Automatic failure enrichment**: Multiple artifacts captured on test failure
- **Layered organization**: Separate directories for browser vs desktop captures
- **Integration with test flow**: Automatic capture via fixtures and hooks

---

## 1. Playwright Screenshot Capabilities

### 1.1 Built-in Screenshot Methods

Playwright provides several screenshot capabilities:

```typescript
// Method 1: Page-level screenshots
await page.screenshot({ path: 'screenshot.png', fullPage: true });

// Method 2: Element-level screenshots
await element.screenshot({ path: 'element.png' });

// Method 3: Automatic screenshots via config
use: {
  screenshot: {
    mode: 'on' | 'off' | 'only-on-failure',
    fullPage: boolean
  }
}
```

**Current Configuration:**
```typescript
// playwright.config.ts (line 36-39)
screenshot: {
  mode: 'on',          // Screenshots for EVERY test
  fullPage: false,     // Viewport only by default
}
```

### 1.2 Screenshot Options Analysis

| Option | Purpose | Trade-offs |
|--------|---------|------------|
| `fullPage: true` | Captures entire scrollable content | Larger files, better context |
| `fullPage: false` | Captures visible viewport only | Faster, shows actual user view |
| `mode: 'on'` | Screenshot every test | Complete visual record, more storage |
| `mode: 'only-on-failure'` | Minimal storage | Less debugging context for passing tests |

**Recommendation:** Keep `mode: 'on'` for comprehensive documentation. Use `fullPage: true` for failure captures, `fullPage: false` for in-test progress screenshots.

### 1.3 Trace Recording

```typescript
// playwright.config.ts (line 46)
trace: 'retain-on-failure'
```

Playwright traces provide:
- **Timeline of actions** with screenshots at each step
- **Network activity** logs
- **Console logs** synchronized with actions
- **DOM snapshots** at interaction points
- **Source code** correlation

**Value:** Traces are superior to standalone screenshots for debugging because they provide temporal context and interactivity. However, they require the Playwright Trace Viewer to inspect.

### 1.4 Video Recording

```typescript
// playwright.config.ts (line 41-44)
video: {
  mode: 'retain-on-failure',
  size: { width: 1920, height: 1080 },
}
```

**Analysis:**
- **Pros:** Complete visual record of test execution, shows timing and animations
- **Cons:** Large file sizes (10-50MB per test), harder to scan than screenshots
- **Current strategy:** Only on failure (good balance)

**When to use:**
- Complex UI interactions (drag-drop, animations)
- Debugging flaky tests (timing issues visible in video)
- Race conditions and async behavior

---

## 2. VNC-Based Desktop Snapshots

### 2.1 VNC vs Scrot (Current Implementation)

**Current Implementation (NixOS test):**
```python
# e2e-test/nix/default.nix (line 29-35)
def vnc_screenshot(name: str):
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    filename = f"{name}-{timestamp}"
    client.succeed(f"DISPLAY=:0 scrot /artifacts/screenshots/desktop/{filename}.png")
    return f"/artifacts/screenshots/desktop/{filename}.png"
```

Uses **scrot** (not VNC) for desktop capture. This is actually a better approach for NixOS testing.

### 2.2 Desktop Snapshot Methods Comparison

| Method | Implementation | Pros | Cons |
|--------|---------------|------|------|
| **scrot** | Direct X11 screenshot | Fast, reliable, no server needed | Requires X11 access |
| **VNC screenshot** | VNC server + client tool | Works remotely, cross-platform | Requires VNC server, compression artifacts |
| **Playwright screenshot** | Browser viewport only | Precise, no VM dependencies | Missing desktop chrome, dialogs |

**Current choice: scrot** is optimal for the NixOS test environment.

### 2.3 When Desktop Snapshots Matter

Desktop snapshots capture what Playwright screenshots miss:

1. **Browser chrome**: Address bar, browser UI, extensions
2. **OS-level dialogs**: Permission prompts, file pickers
3. **Desktop notifications**: System alerts outside browser
4. **Window management**: Multiple windows, overlays
5. **Screen resolution context**: Full desktop layout

**Use cases in Alicia testing:**
- Debugging why Chromium didn't launch
- Verifying desktop environment setup
- Capturing OS-level permission dialogs (microphone access)
- Understanding VM state at failure

### 2.4 VNC Integration Script

The failure handler already has VNC screenshot integration:

```typescript
// lib/failure-handler.ts (line 158-169)
private async triggerVncScreenshot(testName: string, ts: string): Promise<void> {
  if (!process.env.NIXOS_TEST) return;

  try {
    await execAsync(
      `vnc-screenshot.sh failure-${testName}-${ts}`,
      { env: { ...process.env } }
    );
  } catch {
    // VNC screenshot is best-effort
  }
}
```

**Note:** The script name is `vnc-screenshot.sh` but it likely wraps `scrot` or similar. This is a naming inconsistency but functionally correct.

---

## 3. Screenshot Strategy

### 3.1 When to Capture Screenshots

**Implemented capture points:**

1. **Test step completion** (via `step` fixture):
```typescript
// lib/fixtures.ts (line 62-68)
step: async ({ artifacts }, use) => {
  const stepFn = async (name: string, fn: () => Promise<void>): Promise<void> => {
    await fn();
    await artifacts.screenshot(name);  // After every step
  };
  await use(stepFn);
}
```

2. **Manual captures** in tests:
```typescript
await artifacts.screenshot('01-app-loaded');
await artifacts.screenshot('02-before-new-conversation');
```

3. **Automatic on failure**:
```typescript
// lib/fixtures.ts (line 56-59)
if (testInfo.status !== testInfo.expectedStatus) {
  const failureHandler = new FailureHandler(page, testInfo, collector, artifactDir);
  await failureHandler.captureAll();
}
```

### 3.2 Recommended Screenshot Triggers

| Trigger | Priority | Rationale |
|---------|----------|-----------|
| **Before each test** | Medium | Establishes baseline state |
| **After each major action** | High | Documents test flow |
| **On each assertion** | Low | Too verbose, trace is better |
| **On test failure** | Critical | Essential for debugging |
| **At test completion** | Medium | Shows final state |

**Current implementation:** Excellent balance with manual captures at key points and automatic failure enrichment.

### 3.3 Screenshot Naming Convention

**Current implementation** (ArtifactCollector):
```typescript
// lib/artifact-collector.ts (line 131-139)
async screenshot(name: string, options?: { fullPage?: boolean }): Promise<string> {
  this.screenshotCounter++;
  const paddedNum = String(this.screenshotCounter).padStart(3, '0');
  const filename = `${paddedNum}-${this.sanitizeFilename(name)}.png`;
  const filepath = path.join(
    this.artifactDir,
    'screenshots/browser',
    filename
  );
  // ...
}
```

**Naming pattern:** `001-app-loaded.png`, `002-before-new-conversation.png`

**Strengths:**
- ✅ Chronological ordering (numbered prefix)
- ✅ Self-documenting (descriptive names)
- ✅ Sanitized for filesystem safety
- ✅ Scoped to test (counter resets per test)

**Alternative patterns considered:**
- `timestamp-name.png` → Harder to sort chronologically in list view
- `step-001.png` → Less descriptive, requires log correlation
- `test-name-001.png` → Redundant (already in directory structure)

**Recommendation:** Current naming is optimal.

---

## 4. Video Recording Strategy

### 4.1 Current Configuration

```typescript
// playwright.config.ts (line 41-44)
video: {
  mode: 'retain-on-failure',
  size: { width: 1920, height: 1080 },
}
```

### 4.2 Video Recording Analysis

**Playwright video recording:**
- Records browser viewport only (not desktop)
- Captures at every Playwright action
- Saved as `.webm` format
- 1-2 FPS typically (action-based, not real-time)

**File size estimates:**
- 30-second test: ~5-10MB
- 2-minute test: ~20-30MB
- 12-test suite: ~100-200MB if all fail

### 4.3 Desktop Video Recording

**Not currently implemented**, but could be added:

```bash
# Using ffmpeg to record X11 display
ffmpeg -f x11grab -s 1920x1080 -i :0 \
  -c:v libx264 -preset ultrafast \
  /artifacts/videos/desktop-recording.mp4
```

**When desktop video is valuable:**
- Browser fails to launch
- Window management issues
- Desktop environment crashes
- Multi-window workflows

**Recommendation:** Not needed for current scope. Screenshots + Playwright traces + Playwright video cover debugging needs. Desktop video would be 500MB+ per test run.

### 4.4 Video vs Screenshots vs Traces

| Artifact | Best For | File Size | Scan Speed |
|----------|----------|-----------|------------|
| **Screenshots** | Quick visual verification | Small (~100KB each) | Fast (image viewer) |
| **Playwright Trace** | Detailed debugging with timeline | Medium (~5MB) | Slow (needs viewer) |
| **Playwright Video** | Timing and animation issues | Large (~20MB) | Medium (video player) |
| **Desktop Video** | VM-level issues | Very Large (~500MB) | Medium (video player) |

**Current strategy is optimal:** Screenshots for quick review, traces for detailed debugging, video on failure.

---

## 5. Output Folder Structure

### 5.1 Current Structure

```
test-results/                           # $ARTIFACT_DIR
├── screenshots/
│   ├── browser/                        # Playwright screenshots
│   │   ├── 001-01-app-loaded.png
│   │   ├── 002-02-before-new-conversation.png
│   │   ├── failure-test-name-fullpage-TIMESTAMP.png
│   │   └── failure-test-name-viewport-TIMESTAMP.png
│   └── desktop/                        # VNC/scrot desktop snapshots
│       ├── vnc-before-test-TIMESTAMP.png
│       └── vnc-on-failure-TIMESTAMP.png
├── logs/
│   ├── frontend-console.jsonl          # Browser console
│   ├── network-requests.jsonl          # HTTP requests/responses
│   └── playwright-output.log           # Test runner output
├── traces/
│   ├── har/                            # HAR network captures
│   └── trace.zip                       # Playwright trace (on failure)
├── dom/
│   ├── failure-test-name-TIMESTAMP.html           # DOM snapshot
│   ├── failure-test-name-a11y-TIMESTAMP.json      # Accessibility tree
│   ├── failure-test-name-storage-TIMESTAMP.json   # localStorage/sessionStorage
│   └── failure-test-name-pending-requests-TIMESTAMP.json
├── playwright-output/                  # Playwright internal artifacts
├── report/                             # HTML test report
│   └── index.html
├── results.json                        # JSON test results
└── summary.json                        # Test run summary (NixOS tests)
```

### 5.2 Directory Purpose Analysis

| Directory | Purpose | Retention | Size |
|-----------|---------|-----------|------|
| `screenshots/browser/` | Test progress documentation | Always | 5-10MB |
| `screenshots/desktop/` | VM-level debugging | On failure or checkpoints | 1-2MB |
| `logs/` | Debugging logs | Always | 1-5MB |
| `traces/` | Detailed debugging | On failure | 5-20MB |
| `dom/` | Failure state inspection | On failure | 1-5MB |
| `report/` | Human-readable results | Always | <1MB |

**Total size per test run:**
- **Success:** ~10-15MB (screenshots + logs + report)
- **Failure:** ~30-50MB (adds traces, videos, DOM snapshots, desktop screenshots)

### 5.3 Naming Improvements

**Current issues:**
- Desktop screenshots use timestamp suffixes, browser screenshots use counters
- Inconsistent naming between browser and desktop captures

**Recommended structure:**
```
screenshots/
├── browser/
│   ├── test-name/
│   │   ├── 001-step-name.png
│   │   ├── 002-next-step.png
│   │   └── failure-fullpage.png
│   └── ...
└── desktop/
    ├── test-name/
    │   ├── before-test.png
    │   ├── failure.png
    │   └── after-test.png
    └── ...
```

**Benefits:**
- Test isolation (each test in own folder)
- Easier to navigate (no long filename prefixes)
- Clearer correlation between browser and desktop screenshots

**Implementation complexity:** Would require changes to ArtifactCollector to track test name and create subdirectories.

---

## 6. Implementation Patterns

### 6.1 Playwright Test Hooks

**Current implementation** uses fixtures for automatic behavior:

```typescript
// lib/fixtures.ts
export const test = base.extend<TestFixtures>({
  artifacts: async ({ page }, use, testInfo) => {
    const artifactDir = process.env.ARTIFACT_DIR || './test-results';
    const collector = new ArtifactCollector(page, testInfo, artifactDir);

    await use(collector);  // Test runs here

    // Cleanup and failure handling
    await collector.saveAllLogs();

    if (testInfo.status !== testInfo.expectedStatus) {
      const failureHandler = new FailureHandler(page, testInfo, collector, artifactDir);
      await failureHandler.captureAll();
    }
  },
});
```

**Hook execution flow:**
1. **Before test:** Create ArtifactCollector, attach listeners
2. **During test:** Manual `artifacts.screenshot()` calls
3. **After test (always):** Save console/network logs
4. **After test (on failure):** Comprehensive failure capture

### 6.2 Custom Screenshot Helper

**Current ArtifactCollector implementation:**

```typescript
// lib/artifact-collector.ts (line 131-153)
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
    fullPage: options?.fullPage ?? true,  // Default to full page
  });

  // Attach to Playwright report
  await this.testInfo.attach(name, {
    path: filepath,
    contentType: 'image/png',
  });

  return filepath;
}
```

**Key features:**
- Auto-incrementing counter for ordering
- Sanitized filenames
- Full-page screenshots by default
- Attached to Playwright HTML report
- Returns filepath for further processing

### 6.3 Failure Enrichment Pattern

**FailureHandler captures multiple artifacts in parallel:**

```typescript
// lib/failure-handler.ts (line 18-33)
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

  await this.collector.saveAllLogs();
}
```

**Why Promise.allSettled:**
- Failures in one capture don't block others
- Parallel execution for speed
- Best-effort approach (some may fail, that's OK)

### 6.4 VNC Screenshot Integration

**Trigger from test environment:**

```typescript
// lib/failure-handler.ts (line 158-169)
private async triggerVncScreenshot(testName: string, ts: string): Promise<void> {
  if (!process.env.NIXOS_TEST) return;  // Only in NixOS VM

  try {
    await execAsync(
      `vnc-screenshot.sh failure-${testName}-${ts}`,
      { env: { ...process.env } }
    );
  } catch {
    // VNC screenshot is best-effort
  }
}
```

**Shell script pattern:**
```bash
#!/usr/bin/env bash
# vnc-screenshot.sh
NAME="${1:-screenshot}"
DISPLAY=:0 scrot "/artifacts/screenshots/desktop/${NAME}.png"
```

**Integration with NixOS test:**
```python
# e2e-test/nix/default.nix (line 29-35)
def vnc_screenshot(name: str):
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    filename = f"{name}-{timestamp}"
    client.succeed(f"DISPLAY=:0 scrot /artifacts/screenshots/desktop/{filename}.png")
    return f"/artifacts/screenshots/desktop/{filename}.png"
```

---

## 7. Detailed Implementation Specifications

### 7.1 Screenshot Helper API

**Current API:**
```typescript
class ArtifactCollector {
  async screenshot(name: string, options?: { fullPage?: boolean }): Promise<string>
  async captureDomSnapshot(name: string): Promise<string>
  async captureFailure(testName: string): Promise<void>
  async saveAllLogs(): Promise<void>
  getConsoleErrors(): ConsoleLogEntry[]
  getNetworkFailures(): NetworkLogEntry[]
}
```

**Recommended additions:**
```typescript
class ArtifactCollector {
  // Existing methods...

  // New: Compare screenshots for visual regression
  async compareScreenshot(
    name: string,
    baseline: string,
    threshold?: number
  ): Promise<{ matches: boolean; diff?: Buffer }>

  // New: Screenshot with annotations
  async annotatedScreenshot(
    name: string,
    annotations: Array<{ x: number; y: number; text: string }>
  ): Promise<string>

  // New: Bulk screenshot export
  async generateScreenshotSummary(): Promise<void>
}
```

### 7.2 VNC Screenshot Script Specification

**Shell script location:** `/e2e-test/scripts/vnc-screenshot.sh`

```bash
#!/usr/bin/env bash
# vnc-screenshot.sh - Capture desktop screenshot in NixOS test environment
set -euo pipefail

NAME="${1:-screenshot}"
ARTIFACT_DIR="${ARTIFACT_DIR:-./test-results}"
OUTPUT_DIR="${ARTIFACT_DIR}/screenshots/desktop"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
FILENAME="${NAME}-${TIMESTAMP}.png"

mkdir -p "$OUTPUT_DIR"

# Use scrot for X11 screenshot
DISPLAY="${DISPLAY:-:0}" scrot "$OUTPUT_DIR/$FILENAME"

echo "Desktop screenshot saved: $OUTPUT_DIR/$FILENAME"
```

**Integration points:**
1. Called from FailureHandler on test failure
2. Called from NixOS test script at checkpoints
3. Available for manual debugging

### 7.3 Playwright Configuration for Artifacts

**Current optimal configuration:**
```typescript
// playwright.config.ts
export default defineConfig({
  outputDir: path.join(outputDir, 'playwright-output'),

  use: {
    // Screenshots: capture all but viewport only
    screenshot: {
      mode: 'on',
      fullPage: false,
    },

    // Video: only on failure to save space
    video: {
      mode: 'retain-on-failure',
      size: { width: 1920, height: 1080 },
    },

    // Trace: detailed debugging on failure
    trace: 'retain-on-failure',

    // HAR: full network capture for API debugging
    contextOptions: {
      recordHar: {
        path: path.join(outputDir, 'traces/har'),
        mode: 'full',
        content: 'embed',
      },
    },
  },
});
```

**Rationale:**
- **`screenshot: 'on'`** → Visual documentation of every test
- **`fullPage: false`** → Faster, shows actual viewport (full page on failure)
- **`video: 'retain-on-failure'`** → Detailed debugging without excessive storage
- **`trace: 'retain-on-failure'`** → Best debugging tool, but large files
- **HAR recording** → Essential for API debugging

### 7.4 Screenshot Comparison Pattern

**For visual regression testing** (future enhancement):

```typescript
import { test, expect } from '@playwright/test';
import { compareImages } from 'pixelmatch';

test('visual regression', async ({ page, artifacts }) => {
  await page.goto('/');

  const screenshot = await artifacts.screenshot('homepage');
  const baseline = './baselines/homepage.png';

  const diff = await compareImages(
    await fs.readFile(baseline),
    await fs.readFile(screenshot),
    { threshold: 0.1 }
  );

  expect(diff.matches).toBe(true);
});
```

**Not currently needed** for smoke tests, but valuable for regression testing.

---

## 8. Integration with Test Reporting

### 8.1 Playwright HTML Report

**Current configuration:**
```typescript
// playwright.config.ts (line 18-22)
reporter: [
  ['html', { outputFolder: path.join(outputDir, 'report'), open: 'never' }],
  ['json', { outputFile: path.join(outputDir, 'results.json') }],
  ['list'],
],
```

**Screenshot attachment to report:**
```typescript
// lib/artifact-collector.ts (line 147-150)
await this.testInfo.attach(name, {
  path: filepath,
  contentType: 'image/png',
});
```

**Result:** Screenshots appear inline in Playwright HTML report, attached to the test step that captured them.

### 8.2 Custom Summary Generation

**NixOS test generates summary.json:**
```python
# e2e-test/nix/default.nix (line 129-151)
summary = {
    "timestamp": datetime.now().isoformat(),
    "status": "passed" if test_passed else "failed",
    "artifacts": {
        "logs": os.listdir(f"${artifactDir}/logs"),
        "screenshots_browser": os.listdir(f"${artifactDir}/screenshots/browser"),
        "screenshots_desktop": os.listdir(f"${artifactDir}/screenshots/desktop"),
    }
}
```

**Purpose:** Quick overview of test results and available artifacts without opening full HTML report.

### 8.3 Screenshot Gallery Generator

**Future enhancement:** Generate an HTML gallery of all screenshots:

```typescript
// lib/screenshot-gallery.ts
export async function generateGallery(artifactDir: string): Promise<void> {
  const screenshotDir = path.join(artifactDir, 'screenshots/browser');
  const screenshots = await fs.readdir(screenshotDir);

  const html = `
    <!DOCTYPE html>
    <html>
    <head><title>Test Screenshots</title></head>
    <body>
      <h1>Test Screenshot Gallery</h1>
      ${screenshots.map(file => `
        <div class="screenshot">
          <h3>${file}</h3>
          <img src="screenshots/browser/${file}" />
        </div>
      `).join('\n')}
    </body>
    </html>
  `;

  await fs.writeFile(
    path.join(artifactDir, 'screenshot-gallery.html'),
    html
  );
}
```

---

## 9. Comparison: VNC vs Playwright Screenshots

### 9.1 Technical Comparison

| Aspect | Playwright Screenshot | VNC/Scrot Desktop Snapshot |
|--------|----------------------|---------------------------|
| **Capture Target** | Browser viewport content | Full desktop including window chrome |
| **Precision** | Pixel-perfect, deterministic | Depends on window manager state |
| **Speed** | Fast (~100ms) | Very fast (~50ms) |
| **File Size** | Small (50-200KB) | Medium (200-500KB) |
| **Context** | Application state only | Full desktop context |
| **Availability** | Always (Playwright built-in) | Only in headed mode with X11/VNC |
| **Use Case** | Test verification | Environment debugging |

### 9.2 When to Use Each

**Use Playwright screenshots when:**
- ✅ Verifying UI state within application
- ✅ Testing responsive layout
- ✅ Documenting test flow
- ✅ Visual regression testing
- ✅ Running headless tests

**Use VNC/desktop snapshots when:**
- ✅ Browser won't launch
- ✅ OS-level dialogs (file pickers, permissions)
- ✅ Desktop notifications
- ✅ Debugging VM environment
- ✅ Multi-window scenarios

### 9.3 Dual-Capture Strategy (Current Implementation)

**Best of both worlds:**
1. **Playwright screenshots** for all test steps (detailed UI state)
2. **Desktop snapshots** at VM checkpoints (environment verification)
3. **Both on failure** (complete debugging context)

**Example timeline:**
```
00:00 - Desktop: vnc-before-test.png (environment check)
00:05 - Browser: 001-app-loaded.png (test step)
00:08 - Browser: 002-conversation-created.png (test step)
00:12 - Browser: 003-message-sent.png (test step)
00:15 - [FAILURE]
00:15 - Desktop: vnc-on-failure.png (environment state)
00:15 - Browser: failure-fullpage.png (UI state)
00:15 - Browser: failure-viewport.png (viewport state)
```

---

## 10. Performance and Storage Considerations

### 10.1 File Size Analysis

**Per-test artifact sizes:**
- Browser screenshot (viewport): ~100KB
- Browser screenshot (full page): ~200KB
- Desktop snapshot: ~300KB
- Playwright trace: ~5MB
- Playwright video: ~20MB
- HAR file: ~2MB

**12-test smoke suite estimates:**
- **All passing:** ~15MB (screenshots + logs + report)
- **1 test failing:** ~40MB (adds trace + video + failure artifacts)
- **All failing:** ~300MB (trace + video for each test)

### 10.2 Retention Strategy

**Recommended retention:**
```typescript
interface ArtifactRetentionPolicy {
  screenshots: 'always';          // Small, always valuable
  traces: 'failure-only';         // Large, debugging only
  videos: 'failure-only';         // Very large, debugging only
  logs: 'always';                 // Small, always valuable
  har: 'failure-only';            // Medium, debugging only
  desktopSnapshots: 'checkpoints-and-failures';  // Medium, selective
}
```

**Implementation:**
```typescript
// Clean up artifacts based on test status
if (testInfo.status === 'passed') {
  await cleanupLargeArtifacts(['traces', 'videos', 'har']);
}
```

### 10.3 Storage Optimization

**Compression strategies:**
```typescript
// Compress screenshots to JPEG for long-term storage
async function compressScreenshots(dir: string): Promise<void> {
  const pngs = await glob('**/*.png', { cwd: dir });

  for (const png of pngs) {
    // Convert PNG to optimized JPEG (70% quality)
    await sharp(png)
      .jpeg({ quality: 70 })
      .toFile(png.replace('.png', '.jpg'));

    await fs.unlink(png);
  }
}
```

**Not recommended for current scope** (15MB per run is acceptable).

---

## 11. Error Handling and Resilience

### 11.1 Graceful Degradation

**Current implementation handles failures gracefully:**

```typescript
// lib/failure-handler.ts (line 22-30)
await Promise.allSettled([
  this.captureFullPageScreenshot(testName, timestamp),
  this.captureViewportScreenshot(testName, timestamp),
  this.captureDomSnapshot(testName, timestamp),
  // ... more captures
]);
```

**Benefits of Promise.allSettled:**
- One capture failing doesn't block others
- Get as many artifacts as possible
- Particularly important for VNC screenshot (may not be available)

### 11.2 Artifact Directory Permissions

**Current safeguards:**
```typescript
// lib/artifact-collector.ts (line 141)
await fs.promises.mkdir(path.dirname(filepath), { recursive: true });
```

**Potential issues:**
- Directory not writable (permissions)
- Disk full
- Path too long (Windows)

**Recommended enhancement:**
```typescript
async screenshot(name: string): Promise<string> {
  try {
    // Existing implementation...
  } catch (error) {
    console.error(`Failed to capture screenshot '${name}':`, error);

    // Fallback: try to save to temp directory
    const fallbackPath = path.join('/tmp', 'test-screenshots', filename);
    await fs.promises.mkdir(path.dirname(fallbackPath), { recursive: true });
    await this.page.screenshot({ path: fallbackPath });

    console.warn(`Screenshot saved to fallback location: ${fallbackPath}`);
    return fallbackPath;
  }
}
```

### 11.3 VNC Screenshot Availability

**Current handling:**
```typescript
// lib/failure-handler.ts (line 158-169)
private async triggerVncScreenshot(testName: string, ts: string): Promise<void> {
  if (!process.env.NIXOS_TEST) return;  // Skip if not in VM

  try {
    await execAsync(`vnc-screenshot.sh failure-${testName}-${ts}`);
  } catch {
    // VNC screenshot is best-effort
  }
}
```

**Good approach:** VNC screenshots are optional, don't fail the test if unavailable.

---

## 12. Advanced Features (Future Enhancements)

### 12.1 Visual Regression Testing

**Pattern:**
```typescript
import pixelmatch from 'pixelmatch';

async function visualRegression(
  current: Buffer,
  baseline: Buffer,
  threshold: number = 0.1
): Promise<boolean> {
  const diff = pixelmatch(current, baseline, null, 1920, 1080, { threshold });
  return diff < (1920 * 1080 * threshold);
}
```

**Use cases:**
- Detect unintended UI changes
- Verify CSS consistency
- Catch layout regressions

### 12.2 Screenshot Annotations

**Pattern:**
```typescript
import sharp from 'sharp';

async function annotateScreenshot(
  imagePath: string,
  annotations: Array<{ x: number; y: number; text: string }>
): Promise<Buffer> {
  const svg = `
    <svg width="1920" height="1080">
      ${annotations.map(a => `
        <circle cx="${a.x}" cy="${a.y}" r="10" fill="red" />
        <text x="${a.x + 15}" y="${a.y}" fill="red">${a.text}</text>
      `).join('\n')}
    </svg>
  `;

  return sharp(imagePath)
    .composite([{ input: Buffer.from(svg), top: 0, left: 0 }])
    .toBuffer();
}
```

**Use cases:**
- Highlight failed assertion locations
- Mark areas of interest in debugging
- Generate annotated failure reports

### 12.3 Screenshot Diffing

**Pattern:**
```typescript
import { PNG } from 'pngjs';

async function screenshotDiff(
  before: string,
  after: string,
  outputPath: string
): Promise<void> {
  const img1 = PNG.sync.read(await fs.readFile(before));
  const img2 = PNG.sync.read(await fs.readFile(after));
  const diff = new PNG({ width: img1.width, height: img1.height });

  pixelmatch(img1.data, img2.data, diff.data, img1.width, img1.height);

  await fs.writeFile(outputPath, PNG.sync.write(diff));
}
```

**Use cases:**
- Visualize changes between test steps
- Debug state transitions
- Verify animations completed

### 12.4 Adaptive Screenshot Frequency

**Pattern:**
```typescript
class AdaptiveScreenshotCollector extends ArtifactCollector {
  private actionsSinceLastScreenshot = 0;
  private readonly actionThreshold = 3;

  async maybeScreenshot(action: string): Promise<void> {
    this.actionsSinceLastScreenshot++;

    if (this.actionsSinceLastScreenshot >= this.actionThreshold) {
      await this.screenshot(action);
      this.actionsSinceLastScreenshot = 0;
    }
  }
}
```

**Use cases:**
- Reduce screenshot count for long tests
- Focus on important actions
- Balance detail vs storage

---

## 13. Recommendations Summary

### 13.1 Current Implementation Assessment

**Strengths:**
- ✅ Comprehensive dual-layer capture (browser + desktop)
- ✅ Numbered screenshot sequencing for clarity
- ✅ Automatic failure enrichment with multiple artifact types
- ✅ Graceful degradation (Promise.allSettled)
- ✅ Integration with Playwright HTML report
- ✅ Appropriate video/trace retention (only on failure)

**Areas for improvement:**
- ⚠️ Screenshot folder organization (consider per-test subdirectories)
- ⚠️ VNC screenshot script naming (currently called "vnc-screenshot.sh" but uses scrot)
- ⚠️ No visual regression testing (OK for smoke tests, needed for regression suite)

### 13.2 Immediate Recommendations

**No changes needed.** The current implementation is well-architected and meets all requirements for comprehensive visual test documentation.

**Optional enhancements (low priority):**

1. **Per-test screenshot subdirectories:**
   ```
   screenshots/browser/
   ├── 01-application-loads/
   │   ├── 001-app-loaded.png
   │   └── 002-no-console-errors.png
   └── 02-create-conversation/
       ├── 001-before-new-conversation.png
       └── 002-conversation-created.png
   ```

2. **Rename vnc-screenshot.sh to desktop-screenshot.sh** for clarity

3. **Add screenshot summary generator** for quick overview without opening HTML report

### 13.3 Future Enhancements (when needed)

1. **Visual regression testing** (when adding regression test suite)
2. **Screenshot annotations** (for better failure reports)
3. **Adaptive screenshot frequency** (for longer test suites)
4. **Desktop video recording** (only if VM-level issues become common)

---

## 14. Complete Configuration Reference

### 14.1 Playwright Config

```typescript
// e2e-test/playwright.config.ts
import { defineConfig } from '@playwright/test';
import path from 'path';

const SERVER_URL = process.env.ALICIA_SERVER_URL || 'http://localhost:8080';
const outputDir = process.env.ARTIFACT_DIR || './test-results';
const isCI = !!process.env.CI;
const isNixosTest = !!process.env.NIXOS_TEST;

export default defineConfig({
  testDir: './tests',
  outputDir: path.join(outputDir, 'playwright-output'),

  fullyParallel: false,  // Sequential for deterministic screenshots
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
      mode: 'on',           // Screenshot every test
      fullPage: false,      // Viewport only (full page on failure)
    },

    video: {
      mode: 'retain-on-failure',  // Videos only when tests fail
      size: { width: 1920, height: 1080 },
    },

    trace: 'retain-on-failure',  // Detailed traces on failure

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
        channel: 'chromium',
        launchOptions: {
          args: [
            '--disable-gpu',
            '--no-sandbox',
            '--disable-dev-shm-usage',
            '--use-fake-ui-for-media-stream',
            '--use-fake-device-for-media-stream',
          ],
        },
      },
    },
  ],
});
```

### 14.2 Fixture Configuration

```typescript
// e2e-test/lib/fixtures.ts
import { test as base } from '@playwright/test';
import { ArtifactCollector } from './artifact-collector';
import { FailureHandler } from './failure-handler';

export const test = base.extend({
  artifacts: async ({ page }, use, testInfo) => {
    const artifactDir = process.env.ARTIFACT_DIR || './test-results';
    const collector = new ArtifactCollector(page, testInfo, artifactDir);

    await use(collector);

    await collector.saveAllLogs();

    if (testInfo.status !== testInfo.expectedStatus) {
      const failureHandler = new FailureHandler(page, testInfo, collector, artifactDir);
      await failureHandler.captureAll();
    }
  },

  step: async ({ artifacts }, use) => {
    const stepFn = async (name: string, fn: () => Promise<void>): Promise<void> => {
      await fn();
      await artifacts.screenshot(name);
    };
    await use(stepFn);
  },
});
```

### 14.3 NixOS Test Configuration

```python
# e2e-test/nix/default.nix (excerpt)

def vnc_screenshot(name: str):
    """Capture desktop screenshot from client VM"""
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    filename = f"{name}-{timestamp}"
    client.succeed(f"DISPLAY=:0 scrot /artifacts/screenshots/desktop/{filename}.png")
    print(f"Screenshot saved: {filename}.png")
    return f"/artifacts/screenshots/desktop/{filename}.png"

# Checkpoints
vnc_screenshot("vnc-before-test")
vnc_screenshot("vnc-playwright-ready")

# On failure
except Exception as e:
    vnc_screenshot("vnc-on-failure")
    collect_server_logs()
    raise

# Final state
vnc_screenshot("vnc-final-state")
```

---

## 15. Conclusion

The Alicia e2e screenshot and snapshot system is **well-architected and production-ready**. It implements a comprehensive dual-layer capture strategy that balances:

- **Documentation** (every test step screenshotted)
- **Debugging** (failure enrichment with multiple artifact types)
- **Storage efficiency** (traces/videos only on failure)
- **Resilience** (graceful degradation, best-effort VNC)

**Key architectural strengths:**

1. **Dual capture layers** - Browser (Playwright) for UI, Desktop (scrot) for environment
2. **Sequential numbering** - Clear chronological ordering of screenshots
3. **Automatic failure handling** - Comprehensive artifact collection without test code changes
4. **Layered organization** - Separate directories for different artifact types
5. **Integration with reporting** - Screenshots attached to Playwright HTML report

**No immediate changes required.** The system is ready for production use.

**Future evolution:** When the test suite expands beyond smoke tests, consider adding visual regression testing capabilities and per-test screenshot subdirectories for better organization.
