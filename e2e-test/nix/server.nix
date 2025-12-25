# Curried function: first takes extra args, then returns a standard NixOS module
{ self, logCollector }:
{ config, pkgs, lib, ... }:

let
  # PostgreSQL with pgvector extension
  postgresWithVector = pkgs.postgresql_17.withPackages (p: [ p.pgvector ]);

  # Reference to alicia packages from flake
  alicia = self.packages.${pkgs.system}.alicia;

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
    hostName = "server";

    # nixosTest provides virtual network automatically
    # Server will be reachable at hostname "server" from client VM

    # Disable firewall in test environment to avoid nf_conntrack table exhaustion
    # This is safe because the test runs in an isolated VM network
    firewall.enable = false;

    # Ensure external DNS resolution works
    nameservers = [ "8.8.8.8" "1.1.1.1" ];
  };

  # ============================================================================
  # Kernel Parameters
  # ============================================================================

  # Increase connection tracking table size to prevent packet drops during tests
  # Also increase hash table size and timeout settings
  boot.kernel.sysctl = {
    "net.netfilter.nf_conntrack_max" = 524288;
    "net.nf_conntrack_max" = 524288;
    # Reduce timeouts to free up entries faster
    "net.netfilter.nf_conntrack_tcp_timeout_established" = 3600;
    "net.netfilter.nf_conntrack_tcp_timeout_time_wait" = 30;
    "net.netfilter.nf_conntrack_tcp_timeout_close_wait" = 30;
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
    # Depend on postgresql.target which includes the setup scripts (initialScript)
    # This ensures the alicia user exists before migrations run
    after = [ "postgresql.target" ];
    requires = [ "postgresql.target" ];
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
    after = [ "network-online.target" "postgresql.target" "alicia-migrate.service" ];
    requires = [ "postgresql.target" "alicia-migrate.service" ];
    wants = [ "network-online.target" ];
    wantedBy = [ "multi-user.target" ];

    environment = {
      # Server configuration
      ALICIA_SERVER_HOST = "127.0.0.1";
      ALICIA_SERVER_PORT = "9999";

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

    serviceConfig = {
      Type = "simple";
      User = "alicia";
      Group = "alicia";
      WorkingDirectory = "/var/lib/alicia";

      # Environment file with API keys
      EnvironmentFile = "/etc/alicia/env";

      # Main service command
      ExecStart = "${alicia}/bin/alicia serve";

      # Health check: wait for /health to respond
      ExecStartPost = pkgs.writeShellScript "alicia-healthcheck" ''
        for i in $(seq 1 30); do
          if ${pkgs.curl}/bin/curl -sf http://127.0.0.1:9999/health >/dev/null 2>&1; then
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

      locations = {
        # API proxy to backend
        "/api/" = {
          proxyPass = "http://127.0.0.1:9999";
          proxyWebsockets = true;
          extraConfig = ''
            proxy_connect_timeout 300s;
            proxy_send_timeout 300s;
            proxy_read_timeout 300s;
            client_max_body_size 10m;
          '';
        };

        # Health endpoint proxy
        "/health" = {
          proxyPass = "http://127.0.0.1:9999/health";
        };

        # WebSocket proxy for real-time features
        "/ws" = {
          proxyPass = "http://127.0.0.1:9999/ws";
          proxyWebsockets = true;
          extraConfig = ''
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_read_timeout 86400;
          '';
        };

        # Frontend configuration endpoint (LiveKit URL for client)
        # Use = exact match to avoid alias traversal issues
        "= /config.json" = {
          alias = pkgs.writeText "config.json" ''{"livekitUrl": "${livekitUrl}"}'';
          extraConfig = ''
            default_type application/json;
            add_header X-Frame-Options "SAMEORIGIN" always;
            add_header X-Content-Type-Options "nosniff" always;
            add_header X-XSS-Protection "1; mode=block" always;
            add_header Cache-Control "no-store, no-cache, must-revalidate" always;
          '';
        };

        # SPA fallback: serve index.html for all unmatched routes
        "/" = {
          tryFiles = "$uri $uri/ /index.html";
          extraConfig = ''
            # Cache static assets (include security headers to avoid gixy warning)
            location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
              expires 1y;
              add_header X-Frame-Options "SAMEORIGIN" always;
              add_header X-Content-Type-Options "nosniff" always;
              add_header X-XSS-Protection "1; mode=block" always;
              add_header Cache-Control "public, immutable" always;
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

  # ============================================================================
  # Debugging Tools (available in test VM)
  # ============================================================================

  environment.systemPackages = with pkgs; [
    curl
    jq
    postgresql_17  # psql client
    netcat
    htop
    logCollector  # collect-logs script for artifact collection
  ];

  # ============================================================================
  # Artifact Directory (for log collection)
  # ============================================================================

  system.activationScripts.createArtifactDir = ''
    mkdir -p /artifacts/logs
    chmod 777 /artifacts
    chmod 777 /artifacts/logs
  '';
}
