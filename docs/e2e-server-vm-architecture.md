# E2E Testing: Server VM Architecture

## Overview

This document specifies the NixOS VM configuration for the Alicia server component in end-to-end testing. The server VM hosts PostgreSQL, the Go backend, and nginx serving the frontend. It connects to external services (llm.decent.town, livekit.decent.town) and exposes HTTP endpoints to the client VM.

### Context

- **Backend**: Go application using chi router, serves HTTP API on port 8888
- **Frontend**: React/Vite app, built to static files, served via nginx
- **Database**: PostgreSQL 17 with pgvector extension
- **External services**: llm.decent.town/v1 (LLM, ASR, TTS, Embedding), livekit.decent.town (voice)
- **Existing infrastructure**: NixOS module at `nix/modules/alicia.nix`, package definitions in `nix/packages/`

### Decisions

1. **VM Framework**: Use `nixosTest` (not nixos-generators) for integrated test orchestration
2. **Frontend serving**: nginx in production mode (not vite dev server) for realistic testing
3. **Database**: Local PostgreSQL with pgvector, migrations run via ExecStartPre
4. **Networking**: Use nixosTest's built-in virtual network (10.0.2.x range)
5. **Health checks**: Use existing `/health` endpoint for readiness verification
6. **Service ordering**: PostgreSQL -> Backend (with migrations) -> nginx (frontend)

---

## 1. NixOS Test Framework Structure

The e2e test uses `nixosTest` which provides:
- Automatic QEMU VM provisioning
- Built-in virtual networking between nodes
- Python test driver for orchestration
- Screenshot and log capture

### File Location

```
nix/
  tests/
    e2e/
      default.nix       # Main test entry point
      server.nix        # Server VM configuration (this document)
      client.nix        # Client VM configuration (separate document)
```

### Test Entry Point

```nix
# nix/tests/e2e/default.nix
{ pkgs, ... }:

pkgs.nixosTest {
  name = "alicia-e2e";

  nodes = {
    server = import ./server.nix;
    client = import ./client.nix;
  };

  testScript = ''
    # Startup sequence
    server.start()
    server.wait_for_unit("postgresql.service")
    server.wait_for_unit("alicia.service")
    server.wait_for_unit("nginx.service")
    server.wait_for_open_port(80)

    # Verify health endpoint
    server.succeed("curl -sf http://localhost/health | grep -q ok")

    client.start()
    client.wait_for_unit("multi-user.target")

    # Client can reach server
    client.succeed("curl -sf http://server/health | grep -q ok")

    # Run Playwright tests (defined in client.nix)
    client.succeed("run-e2e-tests")
  '';
}
```

---

## 2. Server VM Configuration

### Complete Module

```nix
# nix/tests/e2e/server.nix
{ config, pkgs, lib, ... }:

let
  # PostgreSQL with pgvector extension
  postgresWithVector = pkgs.postgresql_17.withPackages (p: [ p.pgvector ]);

  # Reference to alicia packages from flake
  alicia = pkgs.alicia;

  # Database credentials (test-only, not secrets)
  dbUser = "alicia";
  dbPassword = "alicia-test-password";
  dbName = "alicia";
  dbPort = 5432;

  # Construct connection URL
  postgresUrl = "postgres://${dbUser}:${dbPassword}@localhost:${toString dbPort}/${dbName}?sslmode=disable";

  # External service URLs (real services, accessed via VM's network)
  llmUrl = "https://llm.decent.town/v1";
  livekitUrl = "wss://livekit.decent.town";
in
{
  # ============================================================================
  # System Configuration
  # ============================================================================

  system.stateVersion = "24.11";

  # Minimal system for faster boot
  documentation.enable = false;
  nix.enable = false;

  # ============================================================================
  # Networking
  # ============================================================================

  networking = {
    hostName = "alicia-server";

    # nixosTest provides virtual network automatically
    # Server will be reachable at hostname "server" from client VM

    firewall = {
      enable = true;
      allowedTCPPorts = [
        80    # nginx (frontend + API proxy)
        8888  # backend direct (for debugging)
      ];
    };

    # Ensure external DNS resolution works
    nameservers = [ "8.8.8.8" "1.1.1.1" ];
  };

  # ============================================================================
  # PostgreSQL with pgvector
  # ============================================================================

  services.postgresql = {
    enable = true;
    package = postgresWithVector;
    enableTCPIP = false;  # Unix socket only for security

    # Authentication configuration
    authentication = lib.mkForce ''
      # TYPE  DATABASE  USER      ADDRESS     METHOD
      local   all       all                   trust
      host    all       all       127.0.0.1/32  md5
      host    all       all       ::1/128       md5
    '';

    # Initial database setup
    initialScript = pkgs.writeText "init-alicia-db.sql" ''
      CREATE USER ${dbUser} WITH PASSWORD '${dbPassword}';
      CREATE DATABASE ${dbName} OWNER ${dbUser};
      GRANT ALL PRIVILEGES ON DATABASE ${dbName} TO ${dbUser};

      \c ${dbName}
      CREATE EXTENSION IF NOT EXISTS pgcrypto;
      CREATE EXTENSION IF NOT EXISTS vector;
    '';

    settings = {
      # Performance tuning for test VM (minimal resources)
      shared_buffers = "128MB";
      effective_cache_size = "256MB";
      work_mem = "4MB";
      maintenance_work_mem = "64MB";

      # Logging for debugging
      log_statement = "all";
      log_duration = true;
    };
  };

  # ============================================================================
  # Database Migration Service
  # ============================================================================

  systemd.services.alicia-migrate = {
    description = "Alicia Database Migrations";
    after = [ "postgresql.service" ];
    requires = [ "postgresql.service" ];
    before = [ "alicia.service" ];
    wantedBy = [ "multi-user.target" ];

    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
      User = "alicia";
      Group = "alicia";
    };

    environment = {
      PGHOST = "/run/postgresql";
      PGUSER = dbUser;
      PGDATABASE = dbName;
    };

    script = ''
      set -euo pipefail

      echo "Running database migrations..."

      # Run each migration in order
      for migration in ${alicia}/share/alicia/migrations/*_*.up.sql; do
        if [ -f "$migration" ]; then
          migration_name=$(basename "$migration")
          echo "Applying: $migration_name"

          # Use psql with error handling
          if ! ${postgresWithVector}/bin/psql -v ON_ERROR_STOP=1 -f "$migration" 2>&1; then
            echo "ERROR: Migration failed: $migration_name" >&2
            exit 1
          fi
        fi
      done

      echo "All migrations completed successfully"
    '';
  };

  # ============================================================================
  # Alicia Backend Service
  # ============================================================================

  # Create service user
  users.users.alicia = {
    isSystemUser = true;
    group = "alicia";
    home = "/var/lib/alicia";
    createHome = true;
    description = "Alicia service user";
  };
  users.groups.alicia = {};

  # Data directory
  systemd.tmpfiles.rules = [
    "d /var/lib/alicia 0750 alicia alicia - -"
  ];

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
      ALICIA_LLM_URL = llmUrl;
      ALICIA_LLM_MODEL = "qwen3-8b";
      ALICIA_LLM_MAX_TOKENS = "32368";
      ALICIA_LLM_TEMPERATURE = "0.1";

      # ASR configuration (external service)
      ALICIA_ASR_URL = llmUrl;
      ALICIA_ASR_MODEL = "whisper";

      # TTS configuration (external service)
      ALICIA_TTS_URL = llmUrl;
      ALICIA_TTS_MODEL = "kokoro";
      ALICIA_TTS_VOICE = "af_heart";

      # Embedding configuration (external service)
      ALICIA_EMBEDDING_URL = llmUrl;
      ALICIA_EMBEDDING_MODEL = "e5-large";

      # LiveKit configuration (external service)
      ALICIA_LIVEKIT_URL = livekitUrl;

      # Static files location (for fallback, nginx handles this)
      ALICIA_STATIC_DIR = "${alicia}/share/alicia/frontend";
    };

    # API keys loaded from environment file (test keys)
    serviceConfig = {
      Type = "simple";
      User = "alicia";
      Group = "alicia";
      WorkingDirectory = "/var/lib/alicia";
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

      # Resource limits
      LimitNOFILE = "65536";
    };
  };

  # ============================================================================
  # Nginx (Frontend + API Proxy)
  # ============================================================================

  services.nginx = {
    enable = true;

    recommendedGzipSettings = true;
    recommendedOptimisation = true;
    recommendedProxySettings = true;

    virtualHosts."_" = {
      default = true;

      # Serve frontend static files
      root = "${alicia}/share/alicia/frontend";
      index = "index.html";

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

        # Health endpoint proxy
        "/health" = {
          proxyPass = "http://127.0.0.1:8888/health";
        };

        # WebSocket proxy for real-time features
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

        # Frontend configuration endpoint (LiveKit URL for client)
        "/config.json" = {
          extraConfig = ''
            default_type application/json;
            return 200 '{"livekitUrl": "${livekitUrl}"}';
            add_header Cache-Control "no-store, no-cache, must-revalidate";
          '';
        };

        # SPA fallback: serve index.html for all unmatched routes
        "/" = {
          tryFiles = "$uri $uri/ /index.html";
          extraConfig = ''
            # Cache static assets
            location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
              expires 1y;
              add_header Cache-Control "public, immutable";
            }
          '';
        };
      };

      extraConfig = ''
        # Security headers
        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-XSS-Protection "1; mode=block" always;
      '';
    };
  };

  # Ensure nginx starts after backend is ready
  systemd.services.nginx = {
    after = [ "alicia.service" ];
    wants = [ "alicia.service" ];
  };

  # ============================================================================
  # Test Environment Files
  # ============================================================================

  # Environment file with API keys (test values)
  environment.etc."alicia/env".text = ''
    ALICIA_LLM_API_KEY=test-api-key
    ALICIA_LIVEKIT_API_KEY=APIkey1234
    ALICIA_LIVEKIT_API_SECRET=Fm2gvy8tLtzjDagyRe45Z/XLhC8BaDk5NF+bWdgvHy4=
  '';

  # Make env file available to service
  systemd.services.alicia.serviceConfig.EnvironmentFile = "/etc/alicia/env";

  # ============================================================================
  # Debugging Tools (available in test VM)
  # ============================================================================

  environment.systemPackages = with pkgs; [
    curl
    jq
    postgresql_17  # psql client
    netcat
    htop
  ];
}
```

---

## 3. PostgreSQL Setup Details

### Extension Installation

pgvector is installed via the `withPackages` wrapper:

```nix
postgresWithVector = pkgs.postgresql_17.withPackages (p: [ p.pgvector ]);
```

This ensures the extension is available when `CREATE EXTENSION vector` is called.

### Database Initialization

The `initialScript` runs once when the database cluster is first created:

1. Creates the `alicia` user with password
2. Creates the `alicia` database
3. Enables `pgcrypto` (for `gen_random_bytes` used in ID generation)
4. Enables `vector` (for embedding similarity search)

### Migration Execution

Migrations run as a separate oneshot service:

1. **Ordering**: After PostgreSQL, before backend
2. **Idempotency**: Migrations should be idempotent (use `IF NOT EXISTS`)
3. **Error handling**: Fails loudly with `ON_ERROR_STOP=1`
4. **File ordering**: Sorted by filename prefix (001, 002, etc.)

### Connection Configuration

```
postgres://alicia:alicia-test-password@localhost:5432/alicia?sslmode=disable
```

- Unix socket preferred for local connections (faster, more secure)
- Password authentication for TCP connections (backup)
- `sslmode=disable` acceptable for localhost in test environment

---

## 4. Backend Service Details

### Environment Variables

| Variable | Value | Description |
|----------|-------|-------------|
| `ALICIA_SERVER_HOST` | `127.0.0.1` | Bind to localhost only (nginx proxies) |
| `ALICIA_SERVER_PORT` | `8888` | HTTP API port |
| `ALICIA_POSTGRES_URL` | connection string | Database connection |
| `ALICIA_LLM_URL` | `https://llm.decent.town/v1` | External LLM service |
| `ALICIA_LLM_MODEL` | `qwen3-8b` | Model identifier |
| `ALICIA_LIVEKIT_URL` | `wss://livekit.decent.town` | External voice service |
| `ALICIA_STATIC_DIR` | package path | Frontend files (fallback) |

### Health Check Endpoint

The backend exposes two health endpoints:

- `GET /health` - Simple health check, returns `{"status": "ok"}`
- `GET /health/detailed` - Checks all dependencies (database, LLM, etc.)

The ExecStartPost script polls `/health` for up to 30 seconds before the service is considered started.

### Service Dependencies

```
postgresql.service
       |
       v
alicia-migrate.service (oneshot)
       |
       v
alicia.service
       |
       v
nginx.service
```

---

## 5. Frontend Serving

### Why nginx (not vite dev server)

1. **Production parity**: E2E tests should mirror production deployment
2. **Performance**: Nginx serves static files efficiently
3. **API proxy**: Unified port 80 for both frontend and API
4. **WebSocket handling**: Proper upgrade handling for real-time features

### Static File Locations

```
${alicia}/share/alicia/frontend/
  index.html
  assets/
    index-*.js
    index-*.css
  ...
```

### SPA Routing

The `tryFiles` directive ensures client-side routing works:

```nginx
tryFiles = "$uri $uri/ /index.html";
```

This returns `index.html` for any unmatched route, allowing React Router to handle navigation.

### API Proxy Configuration

```
/api/*     -> http://127.0.0.1:8888/api/*
/health    -> http://127.0.0.1:8888/health
/ws        -> http://127.0.0.1:8888/ws (WebSocket upgrade)
/*         -> static files from frontend build
```

---

## 6. Inter-VM Networking

### nixosTest Virtual Network

nixosTest creates a virtual network automatically:

- Each node gets a hostname matching its name in the `nodes` attribute
- Server is reachable from client as `server` (no IP needed)
- Default network: 10.0.2.x range
- Internet access available for external services

### Hostname Resolution

From the client VM:

```bash
curl http://server/health       # Works via hostname
curl http://server/api/v1/...   # API access
```

### Firewall Rules

Server opens:

- Port 80 (nginx) - Primary entry point
- Port 8888 (backend) - For debugging/direct access

### External Service Access

The VM can reach external services via NAT:

```bash
curl https://llm.decent.town/v1/models  # Works from VM
```

This requires:

1. DNS resolution (configured via `networking.nameservers`)
2. Outbound HTTPS (allowed by default)

---

## 7. Startup Orchestration

### Service Startup Order

```
1. network-online.target
2. postgresql.service
3. alicia-migrate.service (oneshot, blocks until complete)
4. alicia.service (waits for health check in ExecStartPost)
5. nginx.service
```

### Health Check Flow

```python
# In testScript
server.wait_for_unit("postgresql.service")    # DB ready
server.wait_for_unit("alicia.service")        # Backend + migrations done
server.wait_for_unit("nginx.service")         # Frontend ready
server.wait_for_open_port(80)                 # Nginx accepting connections

# Verify end-to-end
server.succeed("curl -sf http://localhost/health | grep -q ok")
```

### Timeout Configuration

| Check | Timeout | Rationale |
|-------|---------|-----------|
| Backend health poll | 30s | Migrations may take time |
| PostgreSQL startup | default | Usually fast |
| Nginx startup | default | Depends on backend |

### Failure Handling

- Migration failure: `alicia-migrate.service` fails, blocking `alicia.service`
- Backend failure: `ExecStartPost` exits non-zero, service marked failed
- Test failure: Python test driver captures logs and screenshots

---

## 8. Integration with Flake

### Adding to flake.nix

```nix
# In flake.nix outputs
checks.${system} = {
  # Existing checks...
  backend-tests = ...;
  frontend-tests = ...;

  # E2E test
  e2e = import ./nix/tests/e2e {
    inherit pkgs;
    alicia = self.packages.${system}.alicia;
  };
};
```

### Running the Test

```bash
# Run e2e test
nix flake check  # Runs all checks including e2e

# Or specifically
nix build .#checks.x86_64-linux.e2e

# Interactive debugging
nix build .#checks.x86_64-linux.e2e.driverInteractive
./result/bin/nixos-test-driver
```

### Test Artifacts

After test completion:

- `./result/log.html` - Detailed execution log
- `./result/*.png` - Screenshots at each step
- `./result/serial*.log` - VM console logs

---

## 9. Configuration Variants

### Minimal Test Configuration

For faster iteration, disable non-essential services:

```nix
# In server.nix for quick tests
environment = {
  # Disable optional services
  ALICIA_ASR_URL = "";      # No speech recognition
  ALICIA_TTS_URL = "";      # No text-to-speech
  ALICIA_LIVEKIT_URL = "";  # No voice
};
```

### With Mock External Services

For offline testing, run mock services in the VM:

```nix
# Add mock LLM server
systemd.services.mock-llm = {
  wantedBy = [ "multi-user.target" ];
  before = [ "alicia.service" ];
  script = ''
    ${pkgs.python3}/bin/python ${./mock-llm-server.py}
  '';
};

# Point backend to mock
environment.ALICIA_LLM_URL = "http://127.0.0.1:8000/v1";
```

---

## 10. Debugging

### Accessing the VM Interactively

```bash
# Build interactive driver
nix build .#checks.x86_64-linux.e2e.driverInteractive
./result/bin/nixos-test-driver

# In Python REPL
>>> server.start()
>>> server.shell_interact()  # Opens shell in VM
```

### Common Issues

| Symptom | Likely Cause | Debug Steps |
|---------|--------------|-------------|
| Backend never healthy | Migration failed | Check `journalctl -u alicia-migrate` |
| Cannot reach external services | DNS/NAT issue | Test with `curl` from VM shell |
| Frontend 404s | Static files missing | Verify `${alicia}/share/alicia/frontend/` |
| WebSocket fails | Proxy config | Check nginx error log |

### Log Locations

```bash
# In VM shell
journalctl -u postgresql    # Database logs
journalctl -u alicia-migrate  # Migration output
journalctl -u alicia        # Backend logs
journalctl -u nginx         # Nginx access/error logs
```

---

## Summary

This architecture provides:

1. **Isolated environment**: Dedicated VM with all dependencies
2. **Production parity**: nginx + static frontend, not dev server
3. **Proper orchestration**: Services start in correct order with health checks
4. **External service access**: Real LLM/LiveKit services for realistic tests
5. **Debug capability**: Interactive access for troubleshooting

The server VM is one half of the e2e test infrastructure. The client VM (documented separately) will run Playwright tests against this server.
