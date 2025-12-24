# Server VM E2E Testing Architecture

## Overview

This document provides a comprehensive architectural design for the **server VM** that runs the Alicia application stack for end-to-end testing. The server VM is a minimal, self-contained NixOS system built using QEMU/microVM with Nix-generated images, designed to work in concert with a separate client VM that executes Playwright tests.

**Design Goals:**
- Reproducible builds via Nix expressions
- Minimal boot time and resource footprint
- Complete application stack isolation (backend, database, frontend)
- Network accessibility from client VM
- Comprehensive logging and health monitoring
- External service integration (LLM, LiveKit)

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         SERVER VM (NixOS)                       │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │                   Nginx (Port 80)                      │   │
│  │  - Serve frontend static files                         │   │
│  │  - Reverse proxy /api/* → backend:8888                 │   │
│  │  - WebSocket proxy /ws → backend:8888                  │   │
│  │  - Health endpoint → backend:8888/health               │   │
│  │  - Dynamic /config.json (LiveKit URL)                  │   │
│  └──────────────────┬─────────────────────────────────────┘   │
│                     │                                           │
│  ┌─────────────────▼────────────────────────────────────┐     │
│  │          Alicia Backend (Port 8888)                  │     │
│  │  - Go HTTP server                                    │     │
│  │  - REST API (/api/*)                                 │     │
│  │  - WebSocket real-time communication                 │     │
│  │  - Health check endpoint (/health)                   │     │
│  │  - Connects to external LLM/LiveKit                  │     │
│  └──────────────────┬───────────────────────────────────┘     │
│                     │                                           │
│  ┌─────────────────▼────────────────────────────────────┐     │
│  │         PostgreSQL 17 + pgvector                     │     │
│  │  - Unix socket only (no TCP)                         │     │
│  │  - Schema initialized via migrations                 │     │
│  │  - Test credentials (non-secret)                     │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                                 │
│  External Connections (via VM network):                        │
│  ├─ https://llm.decent.town/v1 (OpenAI-compatible LLM)        │
│  └─ wss://livekit.decent.town (LiveKit voice server)          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
         │
         │ Virtual Network Bridge
         │
         ▼
┌─────────────────────────────────────────────────────────────────┐
│                       CLIENT VM (NixOS)                         │
│  - XFCE Desktop                                                 │
│  - Chromium + Playwright                                        │
│  - Accesses server at http://server (hostname resolution)      │
└─────────────────────────────────────────────────────────────────┘
```

## Nix Image Generation

### Image Structure

The server VM image is generated using NixOS's `nixosTest` framework with a custom NixOS configuration. The current implementation at `/e2e-test/nix/server.nix` provides a solid foundation.

**Key Nix Patterns:**

```nix
# In e2e-test/nix/server.nix
{ config, pkgs, lib, ... }:

let
  # Reference packages built by the flake
  alicia = pkgs.alicia;  # Combined backend + frontend + migrations

  # PostgreSQL with pgvector extension for vector similarity search
  postgresWithVector = pkgs.postgresql_17.withPackages (p: [ p.pgvector ]);

  # Test environment configuration (non-secret)
  dbUser = "alicia";
  dbPassword = "alicia-test-password";
  dbName = "alicia";
  postgresUrl = "postgres://${dbUser}:${dbPassword}@localhost:5432/${dbName}?sslmode=disable";
in
{
  # Minimal system configuration for fast boot
  system.stateVersion = "24.11";
  documentation.enable = false;
  nix.enable = false;  # No nix commands needed inside VM

  # ... (services configuration)
}
```

**Package Requirements:**

The VM requires the following packages, all provided by Nix:
- `alicia` - Combined package with backend binary + frontend static files + SQL migrations
- `postgresql_17` with `pgvector` extension
- `nginx` - Web server and reverse proxy
- `curl`, `jq`, `netcat`, `htop` - Debugging utilities

### Migration Handling

**Current State:** The backend package copies migrations to `$out/share/alicia/migrations/`:

```nix
# In nix/packages/backend.nix
postInstall = ''
  mkdir -p $out/share/alicia/migrations
  cp -r ${src}/migrations/*.sql $out/share/alicia/migrations/
'';
```

**Migration Execution:** A systemd oneshot service runs before the backend starts:

```nix
systemd.services.alicia-migrate = {
  description = "Alicia Database Migrations";
  after = [ "postgresql.service" ];
  before = [ "alicia.service" ];
  wantedBy = [ "multi-user.target" ];

  serviceConfig = {
    Type = "oneshot";
    RemainAfterExit = true;
  };

  script = ''
    for migration in ${alicia}/share/alicia/migrations/*_*.up.sql; do
      if [ -f "$migration" ]; then
        psql -v ON_ERROR_STOP=1 -f "$migration"
      fi
    done
  '';
};
```

**Migration Files:** Currently 6 migrations (389 lines total):
- `001_init.up.sql` - Core schema (conversations, messages, audio, memory, tools)
- `002_add_message_sync_tracking.up.sql` - Sync tracking
- `003_add_stanza_id_tracking.up.sql` - XMPP stanza IDs
- `004_add_completion_status.up.sql` - Message completion status
- `005_add_mcp_servers.up.sql` - MCP server configuration
- `006_add_user_id.up.sql` - User ID tracking

## Network Architecture

### Port Allocation

| Service    | Internal Port | External Port | Purpose                          |
|------------|---------------|---------------|----------------------------------|
| Nginx      | 80            | 80            | Frontend + API proxy (main)      |
| Backend    | 8888          | 8888          | Direct backend access (debug)    |
| PostgreSQL | 5432          | -             | Unix socket only (no TCP)        |

### Hostname Resolution

The `nixosTest` framework provides automatic virtual networking:
- Server VM hostname: `server`
- Client can access: `http://server` (resolves to nginx on port 80)
- Client can access: `http://server:8888` (direct backend for debugging)

No manual DNS or hosts file configuration needed.

### Firewall Configuration

```nix
networking.firewall = {
  enable = true;
  allowedTCPPorts = [
    80    # Nginx (frontend + API proxy)
    8888  # Backend direct (for debugging)
  ];
};
```

### External Service Access

The VM must reach external services via its network interface:
- **LLM API:** `https://llm.decent.town/v1` (OpenAI-compatible endpoint)
- **LiveKit:** `wss://livekit.decent.town` (WebSocket for voice)

DNS resolution is ensured with:
```nix
networking.nameservers = [ "8.8.8.8" "1.1.1.1" ];
```

## Service Configuration

### PostgreSQL Database

**Configuration Highlights:**

```nix
services.postgresql = {
  enable = true;
  package = postgresWithVector;
  enableTCPIP = false;  # Unix socket only for security

  # Trust local connections (test environment)
  authentication = lib.mkForce ''
    local   all       all                   trust
    host    all       all       127.0.0.1/32  md5
    host    all       all       ::1/128       md5
  '';

  # Initialize database with user and extensions
  initialScript = pkgs.writeText "init-alicia-db.sql" ''
    CREATE USER ${dbUser} WITH PASSWORD '${dbPassword}';
    CREATE DATABASE ${dbName} OWNER ${dbUser};
    GRANT ALL PRIVILEGES ON DATABASE ${dbName} TO ${dbUser};

    \c ${dbName}
    CREATE EXTENSION IF NOT EXISTS pgcrypto;
    CREATE EXTENSION IF NOT EXISTS vector;
  '';

  # Performance tuning for test VM (minimal resources)
  settings = {
    shared_buffers = "128MB";
    effective_cache_size = "256MB";
    work_mem = "4MB";
    maintenance_work_mem = "64MB";
    log_statement = "all";      # Full query logging
    log_duration = true;        # Log query durations
  };
};
```

**Database Schema:** Initialized via migrations service (see Migration Handling section).

**Socket Location:** `/run/postgresql/.s.PGSQL.5432`

### Alicia Backend

**Service User:**

```nix
users.users.alicia = {
  isSystemUser = true;
  group = "alicia";
  home = "/var/lib/alicia";
  createHome = true;
};
```

**Service Definition:**

```nix
systemd.services.alicia = {
  description = "Alicia Voice Assistant Backend";
  after = [ "network-online.target" "postgresql.service" "alicia-migrate.service" ];
  requires = [ "postgresql.service" "alicia-migrate.service" ];
  wants = [ "network-online.target" ];
  wantedBy = [ "multi-user.target" ];

  environment = {
    # Server configuration
    ALICIA_SERVER_HOST = "127.0.0.1";
    ALICIA_SERVER_PORT = "8888";

    # Database
    ALICIA_POSTGRES_URL = postgresUrl;

    # LLM configuration (external service)
    ALICIA_LLM_URL = "https://llm.decent.town/v1";
    ALICIA_LLM_MODEL = "qwen3-8b";
    ALICIA_LLM_MAX_TOKENS = "32368";
    ALICIA_LLM_TEMPERATURE = "0.1";

    # ASR (Whisper via speaches)
    ALICIA_ASR_URL = "https://llm.decent.town/v1";
    ALICIA_ASR_MODEL = "whisper";

    # TTS (Kokoro via speaches)
    ALICIA_TTS_URL = "https://llm.decent.town/v1";
    ALICIA_TTS_MODEL = "kokoro";
    ALICIA_TTS_VOICE = "af_heart";

    # Embedding (e5-large)
    ALICIA_EMBEDDING_URL = "https://llm.decent.town/v1";
    ALICIA_EMBEDDING_MODEL = "e5-large";

    # LiveKit configuration
    ALICIA_LIVEKIT_URL = "wss://livekit.decent.town";

    # Static files (served by nginx, but backend has fallback)
    ALICIA_STATIC_DIR = "${alicia}/share/alicia/frontend";
  };

  serviceConfig = {
    Type = "simple";
    User = "alicia";
    Group = "alicia";
    WorkingDirectory = "/var/lib/alicia";

    # Environment file with API keys (loaded at runtime)
    EnvironmentFile = "/etc/alicia/env";

    # Main service command
    ExecStart = "${alicia}/bin/alicia server";

    # Health check: wait for /health to respond
    ExecStartPost = pkgs.writeShellScript "alicia-healthcheck" ''
      for i in $(seq 1 30); do
        if ${pkgs.curl}/bin/curl -sf http://127.0.0.1:8888/health >/dev/null 2>&1; then
          echo "Backend is healthy"
          exit 0
        fi
        echo "Waiting for backend... ($i/30)"
        sleep 1
      done
      echo "Backend failed to become healthy"
      exit 1
    '';

    Restart = "on-failure";
    RestartSec = "5s";

    # Security hardening
    NoNewPrivileges = true;
    PrivateTmp = true;
    ProtectSystem = "strict";
    ProtectHome = true;
    ReadWritePaths = [ "/var/lib/alicia" ];
  };
};
```

**Backend Command:** The backend is invoked via `alicia server`, which:
1. Connects to PostgreSQL using `ALICIA_POSTGRES_URL`
2. Starts HTTP server on `ALICIA_SERVER_HOST:ALICIA_SERVER_PORT`
3. Serves REST API at `/api/*`
4. Serves WebSocket at `/ws`
5. Provides health check at `/health` (returns `{"status": "ok"}`)

### Nginx (Frontend + Reverse Proxy)

**Service Configuration:**

```nix
services.nginx = {
  enable = true;

  recommendedGzipSettings = true;
  recommendedOptimisation = true;
  recommendedProxySettings = true;

  virtualHosts."_" = {
    default = true;
    root = "${alicia}/share/alicia/frontend";

    locations = {
      # API proxy to backend
      "/api/" = {
        proxyPass = "http://127.0.0.1:8888/api/";
        proxyWebsockets = true;
        extraConfig = ''
          proxy_set_header X-Real-IP $remote_addr;
          proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
          proxy_set_header X-Forwarded-Proto $scheme;
          proxy_set_header Host $host;

          proxy_connect_timeout 300s;
          proxy_send_timeout 300s;
          proxy_read_timeout 300s;

          client_max_body_size 10m;
        '';
      };

      # Health endpoint
      "/health" = {
        proxyPass = "http://127.0.0.1:8888/health";
      };

      # WebSocket proxy
      "/ws" = {
        proxyPass = "http://127.0.0.1:8888/ws";
        proxyWebsockets = true;
        extraConfig = ''
          proxy_http_version 1.1;
          proxy_set_header Upgrade $http_upgrade;
          proxy_set_header Connection "upgrade";
          proxy_read_timeout 86400;
        '';
      };

      # Dynamic config.json for frontend
      "/config.json" = {
        extraConfig = ''
          default_type application/json;
          return 200 '{"livekitUrl": "wss://livekit.decent.town"}';
          add_header Cache-Control "no-store, no-cache, must-revalidate";
        '';
      };

      # SPA fallback
      "/" = {
        tryFiles = "$uri $uri/ /index.html";
        extraConfig = ''
          location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
          }
        '';
      };
    };

    extraConfig = ''
      add_header X-Frame-Options "SAMEORIGIN" always;
      add_header X-Content-Type-Options "nosniff" always;
      add_header X-XSS-Protection "1; mode=block" always;
    '';
  };
};

# Ensure nginx waits for backend
systemd.services.nginx = {
  after = [ "alicia.service" ];
  wants = [ "alicia.service" ];
};
```

**Frontend Files:** The `alicia` package includes the built Vite/React frontend at `share/alicia/frontend/`, containing:
- `index.html` - Main SPA entry point
- `assets/` - JS, CSS, fonts, images

**Why Nginx?** While the Go backend can serve static files, nginx provides:
- Superior static file serving performance
- Simpler reverse proxy configuration
- Centralized HTTP handling
- Better caching control

## Environment Variables

### Core Variables (Injected via `/etc/alicia/env`)

These contain sensitive data (API keys) and are loaded via systemd's `EnvironmentFile`:

```bash
# /etc/alicia/env (created by environment.etc."alicia/env".text)
ALICIA_LLM_API_KEY=test-api-key
ALICIA_LIVEKIT_API_KEY=APIkey1234
ALICIA_LIVEKIT_API_SECRET=Fm2gvy8tLtzjDagyRe45Z/XLhC8BaDk5NF+bWdgvHy4=
```

**Injection Method:**

```nix
environment.etc."alicia/env".text = ''
  ALICIA_LLM_API_KEY=test-api-key
  ALICIA_LIVEKIT_API_KEY=APIkey1234
  ALICIA_LIVEKIT_API_SECRET=Fm2gvy8tLtzjDagyRe45Z/XLhC8BaDk5NF+bWdgvHy4=
'';
```

### Complete Environment Variable Matrix

| Variable                      | Source              | Value (Test)                                  | Purpose                          |
|-------------------------------|---------------------|-----------------------------------------------|----------------------------------|
| `ALICIA_SERVER_HOST`          | systemd.environment | `127.0.0.1`                                   | Backend bind address             |
| `ALICIA_SERVER_PORT`          | systemd.environment | `8888`                                        | Backend port                     |
| `ALICIA_POSTGRES_URL`         | systemd.environment | `postgres://alicia:alicia-test-password@...` | PostgreSQL connection            |
| `ALICIA_LLM_URL`              | systemd.environment | `https://llm.decent.town/v1`                  | LLM API endpoint                 |
| `ALICIA_LLM_API_KEY`          | EnvironmentFile     | `test-api-key`                                | LLM authentication               |
| `ALICIA_LLM_MODEL`            | systemd.environment | `qwen3-8b`                                    | LLM model selection              |
| `ALICIA_LLM_MAX_TOKENS`       | systemd.environment | `32368`                                       | LLM max tokens                   |
| `ALICIA_LLM_TEMPERATURE`      | systemd.environment | `0.1`                                         | LLM temperature                  |
| `ALICIA_ASR_URL`              | systemd.environment | `https://llm.decent.town/v1`                  | Whisper ASR endpoint             |
| `ALICIA_ASR_MODEL`            | systemd.environment | `whisper`                                     | ASR model                        |
| `ALICIA_TTS_URL`              | systemd.environment | `https://llm.decent.town/v1`                  | Kokoro TTS endpoint              |
| `ALICIA_TTS_MODEL`            | systemd.environment | `kokoro`                                      | TTS model                        |
| `ALICIA_TTS_VOICE`            | systemd.environment | `af_heart`                                    | TTS voice selection              |
| `ALICIA_EMBEDDING_URL`        | systemd.environment | `https://llm.decent.town/v1`                  | Embedding API endpoint           |
| `ALICIA_EMBEDDING_MODEL`      | systemd.environment | `e5-large`                                    | Embedding model                  |
| `ALICIA_LIVEKIT_URL`          | systemd.environment | `wss://livekit.decent.town`                   | LiveKit WebSocket URL            |
| `ALICIA_LIVEKIT_API_KEY`      | EnvironmentFile     | `APIkey1234`                                  | LiveKit API key                  |
| `ALICIA_LIVEKIT_API_SECRET`   | EnvironmentFile     | `Fm2gvy8tLtzjDagyRe45Z/XLhC8BaDk5NF+...`     | LiveKit API secret               |
| `ALICIA_STATIC_DIR`           | systemd.environment | `${alicia}/share/alicia/frontend`             | Frontend static files path       |

**Note:** In production, API keys would be injected via secrets management. For e2e testing, hardcoded test values are acceptable.

### Alternative Parameterization Strategies

**Option 1: Environment Variable Overlay (Current)**
- Pros: Simple, explicit, easy to debug
- Cons: Verbose Nix code

**Option 2: Nix Function Parameters**
```nix
# e2e-test/nix/server.nix
{ config, pkgs, lib, testConfig ? {} }:

let
  llmUrl = testConfig.llmUrl or "https://llm.decent.town/v1";
  livekitUrl = testConfig.livekitUrl or "wss://livekit.decent.town";
  apiKey = testConfig.apiKey or "test-api-key";
in
# ... use in environment
```

**Option 3: External Config File**
```nix
# Pass config.json via ALICIA_CONFIG
environment.etc."alicia/config.json".text = builtins.toJSON {
  llm = { url = llmUrl; model = "qwen3-8b"; };
  # ...
};
```

**Recommendation:** Stick with Option 1 (current approach) for clarity and explicitness in test environments.

## Health Checks

### Backend Health Check

The backend provides a `/health` endpoint that returns:
```json
{"status": "ok"}
```

**systemd Integration:** The `ExecStartPost` script polls this endpoint:

```bash
for i in $(seq 1 30); do
  if curl -sf http://127.0.0.1:8888/health >/dev/null 2>&1; then
    echo "Backend is healthy"
    exit 0
  fi
  echo "Waiting for backend... ($i/30)"
  sleep 1
done
echo "Backend failed to become healthy"
exit 1
```

**Failure Handling:** If health check fails after 30 seconds, systemd marks the service as failed, and the entire VM startup is considered broken.

### Service Dependency Chain

```
postgresql.service (PostgreSQL starts)
    ↓
alicia-migrate.service (Migrations run)
    ↓
alicia.service (Backend starts)
    ↓ (health check succeeds)
nginx.service (Nginx starts)
```

### Test Script Health Verification

The `e2e-test/nix/default.nix` test script includes comprehensive health checks:

```python
# Wait for server services
server.wait_for_unit("postgresql.service")
server.wait_for_unit("alicia.service")
server.wait_for_unit("nginx.service")
server.wait_for_open_port(80)

# Verify backend health endpoint
server.succeed("curl -sf http://localhost/health | grep -q ok")
```

### Multi-Level Health Strategy

1. **systemd Level:** Services must start successfully (ExecStart exits 0)
2. **Health Endpoint Level:** Backend must respond to HTTP health check
3. **Test Script Level:** Client VM verifies connectivity before running tests
4. **Playwright Level:** Tests can query `/health` before each test suite

## Log Collection

### Log Sources

| Service    | Log Location                  | Format        | Collection Method              |
|------------|-------------------------------|---------------|--------------------------------|
| Backend    | journald (`alicia.service`)   | JSON + Text   | `journalctl -u alicia`         |
| PostgreSQL | `/var/log/postgresql/*.log`   | Text          | Copy file                      |
| Nginx      | `/var/log/nginx/access.log`   | Text (NCSA)   | Copy file                      |
| Nginx      | `/var/log/nginx/error.log`    | Text          | Copy file                      |
| System     | journald                      | JSON          | `journalctl --priority=err`    |

### Log Collector Script

The `e2e-test/nix/log-collector.nix` provides a `collect-logs` script:

```bash
#!/bin/bash
set -euo pipefail

ARTIFACT_DIR="${ARTIFACT_DIR:-/artifacts}"
LOGS_DIR="$ARTIFACT_DIR/logs"
mkdir -p "$LOGS_DIR"

# Alicia service logs (JSON format)
journalctl -u alicia --no-pager --output=json > "$LOGS_DIR/backend.jsonl"

# Alicia service logs (text format for stderr)
journalctl -u alicia --no-pager > "$LOGS_DIR/backend-stderr.log"

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

# System errors (last hour)
journalctl --no-pager --since="1 hour ago" --priority=err > "$LOGS_DIR/system-errors.log"

# Compress large files (>10MB)
for file in "$LOGS_DIR"/*.{log,jsonl}; do
  if [ -f "$file" ] && [ $(stat -c%s "$file") -gt 10485760 ]; then
    gzip "$file"
  fi
done

echo "Logs collected to $LOGS_DIR"
```

**Installation:**

```nix
# In server.nix
environment.systemPackages = [
  (import ./log-collector.nix { inherit pkgs; })
];
```

**Usage in Test Script:**

```python
# Collect logs at test end or on failure
server.succeed("collect-logs")
```

### Log Artifact Structure

After collection, artifacts are organized:

```
/tmp/alicia-e2e-artifacts/
├── logs/
│   ├── backend.jsonl              # Structured backend logs
│   ├── backend-stderr.log         # Backend console output
│   ├── postgresql.log             # Database logs (queries, errors)
│   ├── nginx-access.log           # HTTP access logs
│   ├── nginx-error.log            # Nginx errors
│   └── system-errors.log          # System-level errors
├── screenshots/
│   ├── browser/                   # Playwright screenshots
│   └── desktop/                   # VNC desktop screenshots
└── summary.json                   # Test execution summary
```

### Structured Log Parsing

Backend logs in JSON format (`backend.jsonl`) can be parsed:

```bash
# Extract error messages
jq -r 'select(.PRIORITY == "3") | .MESSAGE' logs/backend.jsonl

# Filter by time range
jq -r 'select(.TIMESTAMP > "2025-01-01T12:00:00") | .MESSAGE' logs/backend.jsonl
```

PostgreSQL logs with `log_statement = "all"` provide complete query history for debugging test failures.

## VM Resource Configuration

### Recommended Allocations

```nix
virtualisation = {
  memorySize = 4096;    # 4GB RAM
  cores = 4;            # 4 CPU cores
  diskSize = 8192;      # 8GB disk
  graphics = false;     # No display needed (server)
};
```

**Rationale:**
- **Memory:** PostgreSQL (128MB shared_buffers) + Backend (Go runtime ~100MB) + Nginx (~10MB) + OS (~500MB) = ~1GB baseline, 4GB provides headroom
- **CPU:** Parallel request handling + LLM API calls benefit from multi-core
- **Disk:** System (~2GB) + Postgres data (~1GB) + logs (~500MB) + frontend (~50MB) = ~4GB, 8GB provides margin

### Performance Tuning

**PostgreSQL:** Settings tuned for minimal resources:
```nix
settings = {
  shared_buffers = "128MB";         # Small shared memory
  effective_cache_size = "256MB";   # Conservative cache estimate
  work_mem = "4MB";                 # Per-operation memory
  maintenance_work_mem = "64MB";    # Vacuum, index builds
};
```

**Backend:** Go runtime auto-tunes based on available CPU.

**Nginx:** Default settings adequate for e2e testing (low traffic).

## Complete Nix Expression Structure

### File Organization

```
e2e-test/nix/
├── default.nix           # Main test orchestrator (nixosTest)
├── server.nix            # Server VM configuration
├── client.nix            # Client VM configuration (separate doc)
└── log-collector.nix     # Log collection script
```

### Integration with Flake

```nix
# In flake.nix
checks = {
  e2e = import ./e2e-test/nix {
    inherit pkgs lib;
    self = self;  # Pass flake self for package references
  };
};
```

**Build Command:**
```bash
nix build .#checks.x86_64-linux.e2e
```

**Run Test:**
```bash
nix build .#checks.x86_64-linux.e2e --show-trace
```

### Server VM Nix Expression Template

```nix
# e2e-test/nix/server.nix
{ config, pkgs, lib, ... }:

let
  postgresWithVector = pkgs.postgresql_17.withPackages (p: [ p.pgvector ]);
  alicia = pkgs.alicia;

  # Configuration
  dbUser = "alicia";
  dbPassword = "alicia-test-password";
  dbName = "alicia";
  dbPort = 5432;
  postgresUrl = "postgres://${dbUser}:${dbPassword}@localhost:${toString dbPort}/${dbName}?sslmode=disable";

  llmUrl = "https://llm.decent.town/v1";
  livekitUrl = "wss://livekit.decent.town";
in
{
  system.stateVersion = "24.11";
  documentation.enable = false;
  nix.enable = false;

  networking = {
    hostName = "alicia-server";
    firewall.enable = true;
    firewall.allowedTCPPorts = [ 80 8888 ];
    nameservers = [ "8.8.8.8" "1.1.1.1" ];
  };

  # PostgreSQL
  services.postgresql = { /* ... */ };

  # Migration service
  systemd.services.alicia-migrate = { /* ... */ };

  # Backend service
  users.users.alicia = { /* ... */ };
  systemd.services.alicia = { /* ... */ };

  # Nginx
  services.nginx = { /* ... */ };

  # Environment files
  environment.etc."alicia/env".text = ''
    ALICIA_LLM_API_KEY=test-api-key
    ALICIA_LIVEKIT_API_KEY=APIkey1234
    ALICIA_LIVEKIT_API_SECRET=Fm2gvy8tLtzjDagyRe45Z/XLhC8BaDk5NF+bWdgvHy4=
  '';

  # Debugging tools
  environment.systemPackages = with pkgs; [
    curl jq postgresql_17 netcat htop
    (import ./log-collector.nix { inherit pkgs; })
  ];
}
```

## Testing Strategy

### Startup Sequence

1. **Build Phase:** `nix build .#checks.x86_64-linux.e2e`
   - Builds backend, frontend, migrations
   - Generates server VM image
   - Generates client VM image

2. **VM Boot:** Server VM starts
   - PostgreSQL initializes
   - Migrations run
   - Backend starts and passes health check
   - Nginx starts

3. **Client Connection:** Client VM boots
   - X server starts (XFCE desktop)
   - Tests verify `http://server/health` is reachable

4. **Test Execution:** Playwright runs
   - 12 smoke tests execute sequentially
   - Artifacts collected at each step

5. **Cleanup:** Logs collected from server

### Debugging Failed Boots

If server VM fails to boot or services don't start:

```bash
# Build with verbose output
nix build .#checks.x86_64-linux.e2e --show-trace

# Enter server VM interactively (modify default.nix testScript)
# Add: server.shell_interact()

# Check service status
systemctl status postgresql
systemctl status alicia
systemctl status nginx

# Check logs
journalctl -u alicia -f
journalctl -u postgresql -f

# Test connectivity
curl http://localhost:8888/health
curl http://localhost/health
```

### Network Debugging

From client VM:

```bash
# Verify DNS resolution
ping server

# Verify port reachability
nc -zv server 80
nc -zv server 8888

# Test health endpoint
curl -v http://server/health
```

## Security Considerations

### Test Environment Only

**Important:** This design is for **test environments only**. Production deployment requires:
- TLS/HTTPS termination
- Real secrets management (Vault, sops-nix, etc.)
- Firewall restrictions
- Authentication/authorization
- Database backup strategy
- Monitoring and alerting

### Test-Specific Relaxations

1. **Passwords in Nix:** Database credentials are in plaintext Nix code (acceptable for ephemeral test VMs)
2. **Trust Authentication:** PostgreSQL uses `trust` for local connections
3. **No HTTPS:** HTTP only (TLS not needed for VM-to-VM communication)
4. **Firewall Permissive:** Port 8888 exposed for debugging
5. **Static API Keys:** Hardcoded test API keys

## Future Enhancements

### Potential Improvements

1. **Parallelization:** Run multiple isolated server VMs in parallel for test sharding
2. **Snapshot Testing:** Capture database state snapshots for faster test restarts
3. **Mock External Services:** Replace llm.decent.town/livekit.decent.town with local mocks for offline testing
4. **Custom Qemu Options:** Tune Qemu for faster I/O (virtio-scsi, aio=native)
5. **Prebuilt Databases:** Ship pre-migrated database dumps to skip migration time
6. **Metrics Collection:** Prometheus exporters for performance analysis

### Integration with CI/CD

```yaml
# Example GitHub Actions workflow
- name: Build Server VM
  run: nix build .#checks.x86_64-linux.e2e

- name: Upload Artifacts
  uses: actions/upload-artifact@v3
  with:
    name: e2e-test-artifacts
    path: /tmp/alicia-e2e-artifacts/
```

## Conclusion

This architecture provides a **reproducible, isolated, and comprehensive** e2e testing environment for Alicia. The Nix-based approach ensures:
- **Deterministic builds:** Same inputs → same outputs
- **Dependency isolation:** No host system contamination
- **Version pinning:** Exact package versions specified
- **Declarative configuration:** Infrastructure as code

The server VM successfully runs the full Alicia stack (PostgreSQL, backend, frontend) with external service integration, providing a realistic test environment for Playwright-based e2e tests executed from the client VM.

**Next Steps:**
1. Review and refine this architecture
2. Implement any missing configuration elements
3. Test end-to-end VM communication
4. Document client VM architecture (separate document)
5. Create implementation plan for any required changes
