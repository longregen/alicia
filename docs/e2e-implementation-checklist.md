# E2E Testing: Implementation Checklist

This checklist provides step-by-step instructions to fix the e2e testing infrastructure and get tests running successfully.

---

## Prerequisites

- [ ] NixOS or Nix with flakes enabled
- [ ] Git repository cloned at `/home/usr/projects/alicia`
- [ ] Sufficient disk space (5GB for VMs + artifacts)
- [ ] 8GB RAM available for running two VMs simultaneously

---

## Phase 1: Configuration Fixes (Critical)

### Task 1.1: Fix Test Script Path

**File:** `/home/usr/projects/alicia/e2e-test/nix/default.nix`
**Line:** 43-44

**Current (incorrect):**
```python
client.succeed(
    "cd /home/test/frontend && "
```

**Change to:**
```python
client.succeed(
    "cd /home/test/e2e-test && "
```

**Rationale:** Tests are in `/e2e-test/` directory, not `/frontend/`.

**Test:** Search for all occurrences of `/home/test/frontend` in the file and replace with `/home/test/e2e-test`.

---

### Task 1.2: Copy E2E Tests into Client VM

**File:** `/home/usr/projects/alicia/e2e-test/nix/client.nix`
**Location:** Add after line 131 (before `systemd.services`)

**Add this configuration:**
```nix
# Copy e2e test files into VM
environment.etc."e2e-test" = {
  source = ../../e2e-test;
  target = "e2e-test";
};

# Create symlink in test user's home
system.activationScripts.linkE2eTests = ''
  mkdir -p /home/test
  ln -sfn /etc/e2e-test /home/test/e2e-test
  chown -R test:users /home/test/e2e-test
'';
```

**Rationale:** Client VM needs access to test files, package.json, playwright.config.ts, etc.

**Alternative approach (if above doesn't work):**
```nix
virtualisation.sharedDirectories = {
  e2e-test = {
    source = ../../e2e-test;
    target = "/home/test/e2e-test";
  };
};
```

---

### Task 1.3: Verify Environment Variables

**File:** `/home/usr/projects/alicia/e2e-test/nix/default.nix`
**Lines:** 45-47

**Current:**
```python
"NIXOS_TEST=1 "
"BASE_URL=http://server "
"ARTIFACT_DIR=/artifacts "
```

**Verify these match:**
- `BASE_URL=http://server` (server VM hostname in virtual network)
- `ARTIFACT_DIR=/artifacts` (9P mount point)
- Add `ALICIA_SERVER_URL=http://server` for consistency

**Change to:**
```python
"NIXOS_TEST=1 "
"ALICIA_SERVER_URL=http://server "
"ARTIFACT_DIR=/artifacts "
```

---

### Task 1.4: Fix Playwright Command

**File:** `/home/usr/projects/alicia/e2e-test/nix/default.nix`
**Line:** 48

**Current:**
```python
"npx playwright test e2e/conversation.spec.ts "
"--config=e2e/playwright.e2e.config.ts "
```

**Issues:**
1. References `e2e/conversation.spec.ts` (wrong path)
2. References `e2e/playwright.e2e.config.ts` (doesn't exist)

**Change to:**
```python
"npx playwright test tests/smoke.spec.ts "
"--config=playwright.config.ts "
```

**Or run all tests:**
```python
"npx playwright test "
"--config=playwright.config.ts "
```

---

## Phase 2: Build and Test

### Task 2.1: Build E2E Test

**Command:**
```bash
cd /home/usr/projects/alicia
nix build .#checks.x86_64-linux.e2e --show-trace
```

**Expected output:**
- VM images build successfully
- No Nix evaluation errors
- Build completes (may take 10-30 minutes first time)

**If errors occur:**
- Check syntax errors in `.nix` files
- Verify all file paths exist
- Check flake.nix includes the e2e check

---

### Task 2.2: Run E2E Tests

**After successful build:**

The test should run automatically. If it completes:

```bash
# Check artifacts
ls -la /tmp/alicia-e2e-artifacts/

# View structure
tree /tmp/alicia-e2e-artifacts/
```

**Expected artifacts:**
```
/tmp/alicia-e2e-artifacts/
├── screenshots/
│   ├── browser/
│   │   └── *.png
│   └── desktop/
│       └── *.png
├── logs/
│   ├── frontend-console.jsonl
│   ├── network-requests.jsonl
│   ├── backend.jsonl
│   └── *.log
├── traces/
│   └── har/
├── report/
│   └── index.html
└── summary.json
```

---

### Task 2.3: View Test Results

**HTML Report:**
```bash
# Open Playwright HTML report
firefox /tmp/alicia-e2e-artifacts/report/index.html

# Or use Python HTTP server
cd /tmp/alicia-e2e-artifacts/report
python3 -m http.server 8000
# Then open http://localhost:8000
```

**Summary JSON:**
```bash
# Quick status check
cat /tmp/alicia-e2e-artifacts/summary.json | jq '.status'

# Test counts
cat /tmp/alicia-e2e-artifacts/summary.json | jq '{passed, failed, total: .test_count}'

# Failures
cat /tmp/alicia-e2e-artifacts/summary.json | jq '.failures'
```

---

## Phase 3: Debugging Common Issues

### Issue 3.1: "Server not ready"

**Symptom:**
```
Error: Server did not become ready within timeout
```

**Checks:**
1. Server VM started? `journalctl -u alicia`
2. PostgreSQL running? `systemctl status postgresql`
3. Backend listening? `curl http://localhost:8888/health`
4. Nginx proxying? `curl http://localhost/health`

**Fix:** Increase health check timeout in `default.nix`:
```python
for i in $(seq 1 60):  # Increase from 30 to 60
```

---

### Issue 3.2: "Playwright install failed"

**Symptom:**
```
Error: Chromium browser not found
```

**Checks:**
1. Node.js version correct? `node --version` (should be v22.x)
2. npm install succeeded? Check for `node_modules/`
3. System dependencies present? Check client.nix packages list

**Fix:** Ensure Playwright system dependencies in client.nix:
```nix
environment.systemPackages = with pkgs; [
  chromium
  nodejs_22
  # All the dependencies from lines 88-101
];
```

---

### Issue 3.3: "X11 display not found"

**Symptom:**
```
Error: DISPLAY not set
Error: Cannot open display :0
```

**Checks:**
1. X server running? `ps aux | grep X`
2. Display manager started? `systemctl status display-manager`
3. DISPLAY set? `echo $DISPLAY`

**Fix:**
```python
# In default.nix test script
client.wait_for_x()
client.wait_for_unit("display-manager.service")

# Then run tests with explicit DISPLAY
"DISPLAY=:0 npx playwright test ..."
```

---

### Issue 3.4: "Tests timeout waiting for AI response"

**Symptom:**
```
Test: 04 - Receive AI response
Error: Timeout 60000ms exceeded waiting for assistant response
```

**Checks:**
1. Backend connected to LLM? Check backend logs
2. LLM API key set? Check `/etc/alicia/env`
3. External LLM reachable? `curl https://llm.decent.town/v1/models`

**Fix (temporary):** Skip AI response test
```typescript
// In smoke.spec.ts
test.skip('04 - Receive AI response', async ({ ... }) => {
  // Skip for now
});
```

**Fix (permanent):**
- Use local LLM model (faster)
- Increase timeout to 120s
- Add retry logic

---

### Issue 3.5: "VNC screenshot failed"

**Symptom:**
```
Error: scrot: No such file or directory
```

**Checks:**
1. `scrot` installed? `which scrot`
2. DISPLAY set when running scrot? `DISPLAY=:0 scrot test.png`

**Fix:** Add to client.nix:
```nix
environment.systemPackages = with pkgs; [
  # ... existing packages ...
  scrot  # Ensure scrot is present
  xdotool
  imagemagick  # For screenshot processing
];
```

---

## Phase 4: Validation

### Task 4.1: Verify All 12 Tests Pass

**Expected tests:**
1. ✅ Application loads successfully
2. ✅ Create new conversation
3. ✅ Send text message
4. ⚠️  Receive AI response (may be slow)
5. ✅ Navigate to settings
6. ⚠️  Voice mode activation (LiveKit may fail)
7. ⚠️  Voice interaction flow (LiveKit dependent)
8. ✅ Multiple conversations
9. ✅ Error handling - offline mode
10. ✅ Error handling - invalid input
11. ✅ Persistence across reload
12. ✅ Cleanup and final state

**Acceptable failures:**
- Test 6-7 (voice) if LiveKit server unavailable
- Test 4 (AI response) if LLM server slow/unavailable

---

### Task 4.2: Validate Artifact Collection

**Check screenshots:**
```bash
# Should have ~20-25 browser screenshots
ls /tmp/alicia-e2e-artifacts/screenshots/browser/ | wc -l

# Should have 3 desktop screenshots (before, on-failure, final)
ls /tmp/alicia-e2e-artifacts/screenshots/desktop/ | wc -l
```

**Check logs:**
```bash
# Console logs
cat /tmp/alicia-e2e-artifacts/logs/frontend-console.jsonl | jq '.'

# Network logs
cat /tmp/alicia-e2e-artifacts/logs/network-requests.jsonl | jq '.'

# Backend logs
cat /tmp/alicia-e2e-artifacts/logs/backend.jsonl | jq '.'
```

**Check traces:**
```bash
# HAR files
ls /tmp/alicia-e2e-artifacts/traces/har/

# Playwright traces (on failure)
ls /tmp/alicia-e2e-artifacts/traces/*.zip
```

---

### Task 4.3: Document Test Duration

**Measure total runtime:**

```bash
# From summary.json
cat /tmp/alicia-e2e-artifacts/summary.json | jq '.duration_seconds'
```

**Expected duration:**
- Minimum: 2-3 minutes (if AI responses cached)
- Typical: 4-6 minutes (normal AI inference)
- Maximum: 8-10 minutes (slow VM, cold start)

**Document in README:**
```markdown
## Test Duration

Full smoke test suite (12 tests): ~5 minutes

Breakdown:
- VM boot: 30-60s
- Playwright setup: 30s
- Test execution: 3-4 minutes
- Artifact collection: 10s
```

---

## Phase 5: Finalization

### Task 5.1: Update Documentation

**Files to update:**
- [ ] `/e2e-test/README.md` - Add NixOS VM instructions
- [ ] `/docs/e2e-client-vm-design.md` - Mark as implemented
- [ ] `/docs/e2e-analysis-summary.md` - Update with actual results
- [ ] `/README.md` - Add e2e testing section

---

### Task 5.2: Add Troubleshooting Guide

**Create:** `/e2e-test/TROUBLESHOOTING.md`

**Sections:**
1. Common errors and fixes
2. How to access VM logs
3. How to run tests locally (without Nix)
4. Performance tuning
5. Known limitations

---

### Task 5.3: Create CI Integration

**GitHub Actions example:**

```yaml
# .github/workflows/e2e-tests.yml
name: E2E Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cachix/install-nix-action@v24
        with:
          extra_nix_config: |
            experimental-features = nix-command flakes
      - name: Run E2E Tests
        run: nix build .#checks.x86_64-linux.e2e
      - name: Upload Artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: e2e-artifacts
          path: /tmp/alicia-e2e-artifacts/
```

---

## Phase 6: Future Enhancements

### Enhancement 6.1: Local LiveKit Server

**Goal:** Eliminate external dependency on `livekit.decent.town`

**Approach:**
1. Add third VM to test setup (livekit-vm)
2. Run LiveKit server in container
3. Update client/server to use `http://livekit-vm:7880`

**Benefits:**
- Faster voice tests
- No external dependency
- Full voice flow coverage

---

### Enhancement 6.2: Visual Regression Testing

**Goal:** Catch unintended UI changes

**Approach:**
1. Baseline screenshot collection
2. Comparison on each test run
3. Diff highlighting

**Tools:**
- Percy (SaaS)
- Chromatic (SaaS)
- reg-suit (self-hosted)
- Custom script with ImageMagick

---

### Enhancement 6.3: Performance Benchmarking

**Goal:** Track performance over time

**Metrics:**
- Page load time
- Time to first render
- API response times
- Memory usage

**Tools:**
- Lighthouse CI
- Playwright performance API
- Custom timing markers

---

## Completion Criteria

The e2e testing infrastructure is considered complete when:

- [x] All configuration files correct
- [ ] `nix build .#checks.x86_64-linux.e2e` succeeds
- [ ] At least 10/12 tests pass (voice tests may fail without LiveKit)
- [ ] All artifacts collected successfully
- [ ] HTML report generated and viewable
- [ ] Documentation updated
- [ ] Known limitations documented

---

## Quick Reference Commands

```bash
# Build and run tests
nix build .#checks.x86_64-linux.e2e --show-trace

# View artifacts
ls -la /tmp/alicia-e2e-artifacts/

# View HTML report
firefox /tmp/alicia-e2e-artifacts/report/index.html

# Check test status
cat /tmp/alicia-e2e-artifacts/summary.json | jq '.status'

# View console errors
cat /tmp/alicia-e2e-artifacts/logs/frontend-console.jsonl | \
  jq 'select(.type == "error")'

# View network failures
cat /tmp/alicia-e2e-artifacts/logs/network-requests.jsonl | \
  jq 'select(.type == "failure")'

# View backend logs
cat /tmp/alicia-e2e-artifacts/logs/backend.jsonl | jq '.'

# Check screenshot count
find /tmp/alicia-e2e-artifacts/screenshots -name "*.png" | wc -l

# Clean artifacts
rm -rf /tmp/alicia-e2e-artifacts/
```

---

## Support

If you encounter issues not covered in this checklist:

1. Check `/e2e-test/TROUBLESHOOTING.md`
2. Review `/docs/e2e-client-vm-design.md` for architecture details
3. Examine VM logs: `journalctl -u alicia`, `journalctl -u display-manager`
4. Review artifact logs in `/tmp/alicia-e2e-artifacts/logs/`
5. File an issue with:
   - Error message
   - Relevant logs
   - Screenshots
   - Steps to reproduce

---

**Estimated total time:** 2-4 hours for initial setup, 1-2 hours for debugging and validation.

Good luck! 🚀
