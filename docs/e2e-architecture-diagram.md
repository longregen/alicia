# E2E Testing Architecture: Visual Diagrams

## System Architecture Overview

```
╔══════════════════════════════════════════════════════════════════════════╗
║                           HOST MACHINE (NixOS/Linux)                      ║
║                                                                           ║
║  ┌─────────────────────────────────────────────────────────────────────┐ ║
║  │                    QEMU Virtual Network                              │ ║
║  │                    (192.168.1.0/24)                                  │ ║
║  │                                                                       │ ║
║  │  ┌───────────────────────────┐      ┌───────────────────────────┐  │ ║
║  │  │   SERVER VM               │      │   CLIENT VM               │  │ ║
║  │  │   192.168.1.2             │      │   192.168.1.3             │  │ ║
║  │  │   Hostname: "server"      │      │   Hostname: "client"      │  │ ║
║  │  │                           │      │                           │  │ ║
║  │  │  ┌─────────────────────┐  │      │  ┌─────────────────────┐  │  │ ║
║  │  │  │ PostgreSQL :5432    │  │      │  │ XFCE Desktop        │  │  │ ║
║  │  │  │ (with pgvector)     │  │      │  │                     │  │  │ ║
║  │  │  └─────────────────────┘  │      │  │  ┌───────────────┐  │  │  │ ║
║  │  │           ↓                │      │  │  │  Chromium     │  │  │  │ ║
║  │  │  ┌─────────────────────┐  │      │  │  │  (headed)     │  │  │  ║
║  │  │  │ Alicia Backend      │  │      │  │  └───────────────┘  │  │  │ ║
║  │  │  │ (Go) :8888          │  │      │  │         ↑           │  │  │ ║
║  │  │  │                     │  │      │  │  ┌───────────────┐  │  │  │ ║
║  │  │  │ • HTTP API          │←─┼──────┼──┼──│  Playwright   │  │  │  │ ║
║  │  │  │ • WebSocket         │  │      │  │  │  (Node.js 22) │  │  │  │ ║
║  │  │  │ • LiveKit Agent     │  │      │  │  └───────────────┘  │  │  │ ║
║  │  │  └─────────────────────┘  │      │  │                     │  │  │ ║
║  │  │           ↓                │      │  └─────────────────────┘  │  │ ║
║  │  │  ┌─────────────────────┐  │      │           ↓                │  │ ║
║  │  │  │ Nginx :80           │  │      │  ┌─────────────────────┐  │  │ ║
║  │  │  │                     │  │      │  │ X11 Server :0       │  │  │ ║
║  │  │  │ • Static frontend   │  │      │  │ (1920x1080)         │  │  │ ║
║  │  │  │ • API proxy         │  │      │  └─────────────────────┘  │  │ ║
║  │  │  │ • WebSocket proxy   │  │      │                           │  │ ║
║  │  │  └─────────────────────┘  │      └───────────────────────────┘  │ ║
║  │  └───────────────────────────┘                                     │ ║
║  │                    │                              │                 │ ║
║  │                    └──────────────────────────────┘                 │ ║
║  │                                ↓                                    │ ║
║  │                    ┌──────────────────────────┐                    │ ║
║  │                    │   Shared Directory       │                    │ ║
║  │                    │   /artifacts             │                    │ ║
║  │                    │   (9P virtfs mount)      │                    │ ║
║  │                    └──────────────────────────┘                    │ ║
║  └─────────────────────────────────────────────────────────────────────┘ ║
║                                ↓                                          ║
║                    ┌──────────────────────────┐                          ║
║                    │  /tmp/alicia-e2e-artifacts/                        ║
║                    │  (host filesystem)        │                        ║
║                    │                           │                        ║
║                    │  • screenshots/           │                        ║
║                    │  • logs/                  │                        ║
║                    │  • traces/                │                        ║
║                    │  • report/                │                        ║
║                    │  • summary.json           │                        ║
║                    └──────────────────────────┘                          ║
╚══════════════════════════════════════════════════════════════════════════╝
```

## Client VM Stack Detail

```
╔════════════════════════════════════════════════════════════════════════╗
║                        CLIENT VM (192.168.1.3)                          ║
╠════════════════════════════════════════════════════════════════════════╣
║                                                                         ║
║  Layer 7: Test Execution                                               ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  Playwright Test Runner (Node.js 22)                              │ ║
║  │                                                                    │ ║
║  │  • tests/smoke.spec.ts (12 tests)                                 │ ║
║  │  • lib/fixtures.ts (sidebar, chat, voice, settings)               │ ║
║  │  • lib/artifact-collector.ts (screenshots, logs)                  │ ║
║  │  • lib/failure-handler.ts (enhanced diagnostics)                  │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 6: Browser Automation                                           ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  Chromium Browser (headed mode, version 120.x)                    │ ║
║  │                                                                    │ ║
║  │  Launch flags:                                                     │ ║
║  │  • --disable-gpu (VM compatibility)                               │ ║
║  │  • --no-sandbox (test environment)                                │ ║
║  │  • --use-fake-device-for-media-stream (mocked audio)              │ ║
║  │                                                                    │ ║
║  │  Viewport: 1920x1080                                              │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 5: Desktop Environment                                          ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  XFCE Desktop (minimal installation)                              │ ║
║  │                                                                    │ ║
║  │  Components:                                                       │ ║
║  │  • xfwm4 (window manager) - manages Chromium window               │ ║
║  │  • xfce4-session - session management                             │ ║
║  │  • No panel, no desktop icons (not needed for tests)              │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 4: Display Server                                               ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  X11 Server (X.org)                                                │ ║
║  │                                                                    │ ║
║  │  Display: :0                                                       │ ║
║  │  Resolution: 1920x1080                                            │ ║
║  │  Depth: 24-bit                                                     │ ║
║  │  Screen blanking: DISABLED                                        │ ║
║  │                                                                    │ ║
║  │  VGA: virtio (QEMU accelerated graphics)                          │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 3: Display Manager                                              ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  LightDM                                                           │ ║
║  │                                                                    │ ║
║  │  Autologin: ENABLED                                               │ ║
║  │  User: test                                                        │ ║
║  │  Session: xfce                                                     │ ║
║  │  Password: (none - test VM only)                                  │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 2: System Services                                              ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  • PulseAudio (audio system - for fake media streams)             │ ║
║  │  • D-Bus (inter-process communication)                            │ ║
║  │  • NetworkManager (network configuration)                         │ ║
║  │  • systemd (service management)                                   │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 1: Operating System                                             ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  NixOS (minimal configuration)                                     │ ║
║  │                                                                    │ ║
║  │  • Linux kernel 6.12                                              │ ║
║  │  • systemd init                                                    │ ║
║  │  • Nix package manager (disabled in VM)                           │ ║
║  │  • Documentation: DISABLED (reduce size)                          │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                                                                         ║
║  Resources:                                                             ║
║  • Memory: 4GB RAM                                                      ║
║  • CPUs: 4 cores                                                        ║
║  • Disk: ephemeral (no persistence needed)                             ║
╚════════════════════════════════════════════════════════════════════════╝
```

## Server VM Stack Detail

```
╔════════════════════════════════════════════════════════════════════════╗
║                        SERVER VM (192.168.1.2)                          ║
╠════════════════════════════════════════════════════════════════════════╣
║                                                                         ║
║  Layer 5: Reverse Proxy                                                ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  Nginx :80                                                         │ ║
║  │                                                                    │ ║
║  │  Routes:                                                           │ ║
║  │  • / → /share/alicia/frontend (static files)                      │ ║
║  │  • /api/* → http://127.0.0.1:8888/api/* (backend proxy)           │ ║
║  │  • /ws → http://127.0.0.1:8888/ws (WebSocket upgrade)             │ ║
║  │  • /health → http://127.0.0.1:8888/health (health check)          │ ║
║  │  • /config.json → {"livekitUrl": "wss://livekit.decent.town"}     │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 4: Application Server                                           ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  Alicia Backend (Go) :8888                                         │ ║
║  │                                                                    │ ║
║  │  Endpoints:                                                        │ ║
║  │  • GET  /api/conversations                                        │ ║
║  │  • POST /api/conversations                                        │ ║
║  │  • GET  /api/conversations/:id/messages                           │ ║
║  │  • POST /api/messages                                             │ ║
║  │  • WS   /ws (real-time updates)                                   │ ║
║  │  • GET  /health (health check)                                    │ ║
║  │                                                                    │ ║
║  │  External Dependencies:                                            │ ║
║  │  • LLM: https://llm.decent.town/v1 (Qwen3-8B)                     │ ║
║  │  • ASR: https://llm.decent.town/v1 (Whisper)                      │ ║
║  │  • TTS: https://llm.decent.town/v1 (Kokoro)                       │ ║
║  │  • LiveKit: wss://livekit.decent.town                             │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 3: Database                                                     ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  PostgreSQL 17 :5432 (with pgvector)                              │ ║
║  │                                                                    │ ║
║  │  Database: alicia                                                  │ ║
║  │  User: alicia                                                      │ ║
║  │                                                                    │ ║
║  │  Tables:                                                           │ ║
║  │  • conversations                                                   │ ║
║  │  • messages                                                        │ ║
║  │  • embeddings (with vector similarity search)                     │ ║
║  │                                                                    │ ║
║  │  Logging: ALL statements logged                                   │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 2: System Services                                              ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  • systemd-journald (log collection)                              │ ║
║  │  • NetworkManager (network configuration)                         │ ║
║  │  • systemd (service management)                                   │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                              ↓                                          ║
║  Layer 1: Operating System                                             ║
║  ┌───────────────────────────────────────────────────────────────────┐ ║
║  │  NixOS (minimal configuration)                                     │ ║
║  │                                                                    │ ║
║  │  • Linux kernel 6.12                                              │ ║
║  │  • systemd init                                                    │ ║
║  │  • Nix package manager (disabled in VM)                           │ ║
║  └───────────────────────────────────────────────────────────────────┘ ║
║                                                                         ║
║  Resources:                                                             ║
║  • Memory: 4GB RAM                                                      ║
║  • CPUs: 2 cores                                                        ║
║  • Disk: ephemeral (database in memory, migrations on boot)            ║
╚════════════════════════════════════════════════════════════════════════╝
```

## Test Execution Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│ PHASE 1: VM BOOT                                                      │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  Server VM                          Client VM                        │
│  ────────────                        ──────────                      │
│  1. Start PostgreSQL                 1. Start LightDM                │
│  2. Run migrations                   2. Autologin (test user)        │
│  3. Start Alicia backend             3. Start XFCE                   │
│  4. Health check (wait)              4. Start X11 server             │
│  5. Start Nginx                      5. Wait for graphical.target    │
│  6. Listen on :80                    6. Ready                        │
│                                                                       │
│  Duration: ~30-60 seconds            Duration: ~20-40 seconds        │
└──────────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────────┐
│ PHASE 2: HEALTH CHECKS                                                │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  1. Server health check:    curl http://server/health                │
│     Expected: {"status": "ok"}                                       │
│                                                                       │
│  2. Client X11 check:       wait_for_x()                             │
│     Expected: DISPLAY=:0 active                                      │
│                                                                       │
│  3. Network connectivity:   ping server from client                  │
│     Expected: success                                                │
│                                                                       │
│  Duration: ~5-10 seconds                                             │
└──────────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────────┐
│ PHASE 3: PLAYWRIGHT SETUP                                             │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  1. Install dependencies:    npm ci                                  │
│     Location: /home/test/e2e-test/                                   │
│     Duration: ~15-30 seconds (first run)                             │
│                                                                       │
│  2. Install Chromium:        npx playwright install chromium         │
│     With deps: --with-deps                                           │
│     Duration: ~10-20 seconds                                         │
│                                                                       │
│  3. Create artifact dirs:    mkdir -p /artifacts/{screenshots,logs}  │
│                                                                       │
│  Duration: ~30-60 seconds                                            │
└──────────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────────┐
│ PHASE 4: TEST EXECUTION                                               │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  Command: DISPLAY=:0 NIXOS_TEST=1 ALICIA_SERVER_URL=http://server \  │
│           ARTIFACT_DIR=/artifacts npx playwright test                │
│                                                                       │
│  Test Sequence (serial):                                             │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │ 01. App loads              →  5s  → ✅ Screenshot taken         │  │
│  │ 02. Create conversation    →  2s  → ✅ Screenshot taken         │  │
│  │ 03. Send message           →  1s  → ✅ Screenshot taken         │  │
│  │ 04. AI response            → 60s  → ⚠️  Slow (LLM inference)    │  │
│  │ 05. Settings               →  2s  → ✅ Screenshot taken         │  │
│  │ 06. Voice mode             →  5s  → ⚠️  LiveKit may fail        │  │
│  │ 07. Voice interaction      → 10s  → ⚠️  LiveKit dependent       │  │
│  │ 08. Multi-conversation     → 10s  → ✅ Screenshots taken        │  │
│  │ 09. Offline error          →  5s  → ✅ Error handling           │  │
│  │ 10. Invalid input          →  3s  → ✅ Validation               │  │
│  │ 11. Persistence            → 15s  → ✅ Reload test              │  │
│  │ 12. Cleanup                →  5s  → ✅ Delete conversations     │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                       │
│  Duration: ~3-5 minutes (depends on AI response time)                │
└──────────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────────┐
│ PHASE 5: ARTIFACT COLLECTION                                          │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  Browser Artifacts (Playwright):                                     │
│  • Screenshots: ~20-25 PNG files                                     │
│  • Console logs: frontend-console.jsonl                              │
│  • Network logs: network-requests.jsonl                              │
│  • HAR files: traces/har/*.har                                       │
│  • On failure: DOM snapshots, a11y tree, storage                     │
│                                                                       │
│  Desktop Artifacts (VNC):                                            │
│  • vnc-before-test.png                                               │
│  • vnc-on-failure.png (if test fails)                                │
│  • vnc-final-state.png                                               │
│                                                                       │
│  Server Artifacts (journalctl):                                      │
│  • backend.jsonl (structured logs)                                   │
│  • backend-stderr.log (text logs)                                    │
│  • postgresql.log (database logs)                                    │
│  • nginx-access.log, nginx-error.log                                 │
│  • system-errors.log                                                 │
│                                                                       │
│  Summary:                                                             │
│  • summary.json (test results, metadata)                             │
│  • report/index.html (Playwright HTML report)                        │
│                                                                       │
│  Duration: ~10-15 seconds                                            │
└──────────────────────────────────────────────────────────────────────┘
                              ↓
                        ✅ COMPLETE
                  Artifacts in /tmp/alicia-e2e-artifacts/
```

## Artifact Directory Structure

```
/tmp/alicia-e2e-artifacts/
│
├── screenshots/
│   ├── browser/                     # Playwright screenshots
│   │   ├── 001-01-app-loaded.png
│   │   ├── 002-02-conversation-created.png
│   │   ├── 003-03-message-sent.png
│   │   ├── ...
│   │   ├── 024-12-cleanup-complete.png
│   │   └── failure-*.png            # On test failure
│   │
│   └── desktop/                     # VNC desktop screenshots
│       ├── vnc-before-test-20251224-200000.png
│       ├── vnc-on-failure-20251224-200130.png  # If test fails
│       └── vnc-final-state-20251224-200230.png
│
├── logs/
│   ├── frontend-console.jsonl       # Browser console logs (JSONL)
│   ├── network-requests.jsonl       # HTTP requests/responses (JSONL)
│   ├── backend.jsonl                # Go backend logs (JSONL)
│   ├── backend-stderr.log           # Backend text logs
│   ├── postgresql.log               # Database logs
│   ├── nginx-access.log             # Nginx access logs
│   ├── nginx-error.log              # Nginx error logs
│   └── system-errors.log            # System-level errors
│
├── traces/
│   ├── har/                         # HAR network captures
│   │   └── *.har
│   └── *.zip                        # Playwright traces (on failure)
│
├── dom/                             # DOM snapshots (on failure)
│   ├── failure-test-name.html       # HTML with computed styles
│   ├── failure-test-name-a11y.json  # Accessibility tree
│   └── failure-test-name-storage.json  # localStorage + sessionStorage
│
├── report/                          # Playwright HTML report
│   ├── index.html                   # Main report page
│   └── data/                        # Report assets
│
├── summary.json                     # Test run metadata
└── results.json                     # Playwright results JSON

Total size: ~50-100 MB (depends on number of screenshots and logs)
```

## Data Flow Diagram

```
┌─────────────┐
│   User      │
│  (Simulated │
│   by Test)  │
└──────┬──────┘
       │
       │ 1. Interact (click, type, navigate)
       ↓
┌──────────────────────────────────────────┐
│        Chromium Browser                  │
│        (in Client VM)                    │
│                                          │
│  ┌────────────────────────────────────┐  │
│  │  React Frontend                    │  │
│  │  (served by Nginx from Server VM)  │  │
│  │                                    │  │
│  │  • Sidebar, ChatWindow, Settings   │  │
│  │  • Voice controls (LiveKit UI)     │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
       │
       │ 2. HTTP/WebSocket requests
       ↓
┌──────────────────────────────────────────┐
│        Nginx (Server VM :80)             │
│                                          │
│  Routes:                                 │
│  • / → static files                     │
│  • /api/* → backend :8888               │
│  • /ws → backend WebSocket              │
└──────────────────────────────────────────┘
       │
       │ 3. Proxied requests
       ↓
┌──────────────────────────────────────────┐
│    Alicia Backend (Server VM :8888)      │
│                                          │
│  Handlers:                               │
│  • CreateConversation                    │
│  • ListMessages                          │
│  • SendMessage                           │
│  • WebSocket (real-time)                 │
└──────────────────────────────────────────┘
       │
       │ 4. Database queries
       ↓
┌──────────────────────────────────────────┐
│   PostgreSQL (Server VM :5432)           │
│                                          │
│  Tables:                                 │
│  • conversations                         │
│  • messages                              │
│  • embeddings (vector search)            │
└──────────────────────────────────────────┘
       │
       │ 5. AI requests (external)
       ↓
┌──────────────────────────────────────────┐
│   External Services                      │
│                                          │
│  • https://llm.decent.town               │
│    - Qwen3-8B (LLM)                     │
│    - Whisper (ASR)                      │
│    - Kokoro (TTS)                       │
│                                          │
│  • wss://livekit.decent.town             │
│    - LiveKit (WebRTC audio)             │
└──────────────────────────────────────────┘

Meanwhile, Playwright captures:
┌──────────────────────────────────────────┐
│   Artifact Collector                     │
│                                          │
│  Listening to:                           │
│  • page.on('console')                   │
│  • page.on('request')                   │
│  • page.on('response')                  │
│  • page.on('pageerror')                 │
│                                          │
│  Writing to:                             │
│  • /artifacts/screenshots/browser/      │
│  • /artifacts/logs/*.jsonl              │
│  • /artifacts/traces/har/               │
└──────────────────────────────────────────┘
```
