# Deployment Guide

This guide covers deploying Alicia on a NixOS system with production-ready configuration including PostgreSQL, nginx reverse proxy, and proper secrets management.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Detailed Setup](#detailed-setup)
  - [1. NixOS Configuration](#1-nixos-configuration)
  - [2. PostgreSQL Setup](#2-postgresql-setup)
  - [3. Secrets Management](#3-secrets-management)
  - [4. External Services](#4-external-services)
  - [5. Nginx Configuration](#5-nginx-configuration)
  - [6. Frontend Deployment](#6-frontend-deployment)
- [Configuration Reference](#configuration-reference)
- [Secrets Management Patterns](#secrets-management-patterns)
- [Troubleshooting](#troubleshooting)
- [Production Checklist](#production-checklist)

## Prerequisites

### System Requirements

- **Operating System**: NixOS 23.11 or later
- **Memory**: Minimum 2GB RAM (4GB+ recommended)
- **Storage**: At least 10GB free space
- **CPU**: x86_64 or aarch64 architecture

### External Services

Alicia requires several external services to function:

1. **LLM Service** (Required)
   - vLLM, LiteLLM, or any OpenAI-compatible API
   - Used for conversation and reasoning

2. **PostgreSQL** (Required)
   - Provided by NixOS module
   - Must have pgvector extension enabled

3. **LiveKit Server** (Optional, for real-time voice)
   - Can be self-hosted or use LiveKit Cloud
   - Required for WebRTC-based voice features

4. **ASR Service** (Optional, for speech recognition)
   - Speaches, OpenAI Whisper API, or compatible service
   - Required for voice input

5. **TTS Service** (Optional, for speech synthesis)
   - Speaches, OpenAI TTS API, or compatible service
   - Required for voice output

6. **Embedding Service** (Optional, for RAG/vector search)
   - Required for memory and context retrieval features

## Quick Start

For a minimal local deployment:

```bash
# 1. Clone the repository
git clone https://github.com/longregen/alicia.git
cd alicia

# 2. Copy the example configuration
sudo mkdir -p /etc/nixos
sudo cp nix/examples/configuration.nix /etc/nixos/alicia.nix

# 3. Include in your main configuration
echo 'imports = [ ./alicia.nix ];' | sudo tee -a /etc/nixos/configuration.nix

# 4. Create secrets directory
sudo mkdir -p /var/secrets/alicia
sudo chmod 750 /var/secrets/alicia

# 5. Configure database connection
echo "postgresql://alicia@localhost/alicia?sslmode=disable" | \
  sudo tee /var/secrets/alicia/database-url

# 6. Configure LLM API (adjust URL and key as needed)
echo "http://localhost:8000/v1" > /tmp/llm-url
echo "your-api-key-here" | sudo tee /var/secrets/alicia/llm-api-key

# 7. Apply the configuration
sudo nixos-rebuild switch

# 8. Check service status
sudo systemctl status alicia
```

## Detailed Setup

### 1. NixOS Configuration

#### Using Flakes (Recommended)

Add Alicia to your NixOS flake:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    alicia.url = "github:longregen/alicia";
  };

  outputs = { self, nixpkgs, alicia, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        alicia.nixosModules.default
        ./configuration.nix
      ];
    };
  };
}
```

#### Using Channels

If you're not using flakes:

```nix
# In your configuration.nix
{ config, pkgs, ... }:

let
  aliciaSrc = builtins.fetchGit {
    url = "https://github.com/longregen/alicia.git";
    ref = "main";
  };

  # Alicia requires gomod2nix for building the Go backend
  gomod2nixSrc = builtins.fetchGit {
    url = "https://github.com/nix-community/gomod2nix.git";
    ref = "master";
  };
in
{
  imports = [
    "${aliciaSrc}/nix/modules/alicia.nix"
  ];

  # Apply overlays in order: gomod2nix first, then alicia
  nixpkgs.overlays = [
    (import "${gomod2nixSrc}/overlay.nix")
    (import "${aliciaSrc}/overlay.nix")
  ];

  # ... rest of configuration
}
```

### 2. PostgreSQL Setup

The Alicia module includes PostgreSQL configuration, but you can customize it:

```nix
services.postgresql = {
  enable = true;
  package = pkgs.postgresql_17;

  # Enable pgvector extension
  extraPlugins = with pkgs.postgresql_17.pkgs; [
    pgvector
  ];

  # Database and user setup
  ensureDatabases = [ "alicia" ];
  ensureUsers = [
    {
      name = "alicia";
      ensureDBOwnership = true;
    }
  ];

  # Enable pgvector extension
  initialScript = pkgs.writeText "init.sql" ''
    CREATE EXTENSION IF NOT EXISTS vector;
  '';

  # Performance tuning (adjust based on your hardware)
  settings = {
    # For a system with 4GB RAM:
    shared_buffers = "1GB";
    effective_cache_size = "3GB";
    maintenance_work_mem = "256MB";

    # Connection settings
    max_connections = 100;

    # WAL settings
    wal_level = "replica";
    max_wal_size = "2GB";

    # For SSD storage
    random_page_cost = 1.1;
  };
};
```

#### Manual Database Setup

If you need to set up the database manually:

```bash
# Become postgres user
sudo -u postgres psql

# Create database and user
CREATE DATABASE alicia;
CREATE USER alicia WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE alicia TO alicia;

# Connect to the database
\c alicia

# Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

# Grant permissions
GRANT ALL ON SCHEMA public TO alicia;

# Exit
\q
```

#### Database Backups

Set up automated backups:

```nix
services.postgresqlBackup = {
  enable = true;
  databases = [ "alicia" ];
  startAt = "daily";
  location = "/var/backup/postgresql";
  compression = "zstd";
};
```

### 3. Secrets Management

Alicia supports multiple approaches to secrets management.

#### Option 1: File-Based Secrets (Simple)

Create secret files manually:

```bash
# Create secrets directory
sudo mkdir -p /var/secrets/alicia
sudo chmod 750 /var/secrets/alicia

# Database URL
echo "postgresql://alicia:password@localhost/alicia?sslmode=disable" | \
  sudo tee /var/secrets/alicia/database-url

# LLM API Key
echo "sk-your-api-key-here" | \
  sudo tee /var/secrets/alicia/llm-api-key

# LiveKit credentials (if using LiveKit)
echo "your-livekit-api-key" | \
  sudo tee /var/secrets/alicia/livekit-api-key
echo "your-livekit-api-secret" | \
  sudo tee /var/secrets/alicia/livekit-api-secret

# Secure the files
sudo chown -R root:alicia /var/secrets/alicia
sudo chmod 640 /var/secrets/alicia/*
```

Configure in NixOS:

```nix
services.alicia = {
  database.urlFile = "/var/secrets/alicia/database-url";
  llm.apiKeyFile = "/var/secrets/alicia/llm-api-key";
  livekit.apiKeyFile = "/var/secrets/alicia/livekit-api-key";
  livekit.apiSecretFile = "/var/secrets/alicia/livekit-api-secret";
};
```

#### Option 2: sops-nix (Recommended for Production)

[sops-nix](https://github.com/Mic92/sops-nix) provides encrypted secrets management.

1. **Install sops-nix**:

```nix
{
  inputs.sops-nix.url = "github:Mic92/sops-nix";

  outputs = { self, nixpkgs, sops-nix, alicia, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      modules = [
        sops-nix.nixosModules.sops
        alicia.nixosModules.default
        ./configuration.nix
      ];
    };
  };
}
```

2. **Create secrets file**:

```bash
# Create a sops configuration
cat > .sops.yaml <<EOF
keys:
  - &admin_key age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
creation_rules:
  - path_regex: secrets/.*\.yaml$
    key_groups:
      - age:
          - *admin_key
EOF

# Create secrets
mkdir -p secrets
sops secrets/alicia.yaml
```

3. **Configure secrets**:

```yaml
# secrets/alicia.yaml
database_url: postgresql://alicia:password@localhost/alicia?sslmode=disable
llm_api_key: sk-your-api-key-here
livekit_api_key: your-livekit-key
livekit_api_secret: your-livekit-secret
```

4. **Use in NixOS configuration**:

```nix
{
  sops.defaultSopsFile = ../secrets/alicia.yaml;
  sops.age.keyFile = "/var/lib/sops-nix/key.txt";

  sops.secrets = {
    alicia-db-url = {
      owner = "alicia";
      group = "alicia";
    };
    alicia-llm-key = {
      owner = "alicia";
      group = "alicia";
    };
    alicia-livekit-key = {
      owner = "alicia";
      group = "alicia";
    };
    alicia-livekit-secret = {
      owner = "alicia";
      group = "alicia";
    };
  };

  services.alicia = {
    database.urlFile = config.sops.secrets.alicia-db-url.path;
    llm.apiKeyFile = config.sops.secrets.alicia-llm-key.path;
    livekit.apiKeyFile = config.sops.secrets.alicia-livekit-key.path;
    livekit.apiSecretFile = config.sops.secrets.alicia-livekit-secret.path;
  };
}
```

#### Option 3: agenix

[agenix](https://github.com/ryantm/agenix) is another encrypted secrets solution.

1. **Setup**:

```nix
{
  inputs.agenix.url = "github:ryantm/agenix";

  outputs = { self, nixpkgs, agenix, alicia, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      modules = [
        agenix.nixosModules.default
        alicia.nixosModules.default
        ./configuration.nix
      ];
    };
  };
}
```

2. **Create secrets**:

```bash
# Create secrets directory
mkdir -p secrets

# Create age keys
ssh-keyscan myhost >> secrets/keys.txt

# Create secret files
agenix -e secrets/database-url.age
agenix -e secrets/llm-api-key.age
```

3. **Configure**:

```nix
{
  age.secrets = {
    alicia-db-url.file = ../secrets/database-url.age;
    alicia-llm-key.file = ../secrets/llm-api-key.age;
  };

  services.alicia = {
    database.urlFile = config.age.secrets.alicia-db-url.path;
    llm.apiKeyFile = config.age.secrets.alicia-llm-key.path;
  };
}
```

### 4. External Services

#### LLM Service Setup

**Option A: vLLM with Docker**

```bash
# Create a docker-compose.yml
cat > docker-compose.yml <<EOF
version: '3.8'
services:
  vllm:
    image: vllm/vllm-openai:latest
    ports:
      - "8000:8000"
    volumes:
      - ./models:/root/.cache/huggingface
    command: >
      --model Qwen/Qwen3-8B-AWQ
      --dtype auto
      --api-key your-api-key-here
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
EOF

docker-compose up -d
```

**Option B: vLLM with NixOS systemd**

```nix
virtualisation.docker.enable = true;

systemd.services.vllm = {
  description = "vLLM OpenAI-compatible API";
  after = [ "docker.service" "network.target" ];
  wantedBy = [ "multi-user.target" ];

  serviceConfig = {
    Type = "simple";
    ExecStart = ''
      ${pkgs.docker}/bin/docker run --rm --gpus all \
        -p 8000:8000 \
        -v /var/lib/vllm/models:/root/.cache/huggingface \
        vllm/vllm-openai:latest \
        --model Qwen/Qwen3-8B-AWQ \
        --dtype auto
    '';
    Restart = "always";
    RestartSec = "10s";
  };
};
```

**Option C: LiteLLM as a Proxy**

```nix
services.litellm = {
  enable = true;
  settings = {
    model_list = [
      {
        model_name = "gpt-4";
        litellm_params = {
          model = "openai/gpt-4";
          api_key = "sk-your-openai-key";
        };
      }
    ];
  };
};
```

#### LiveKit Setup

**Option A: LiveKit Cloud**

Sign up at [livekit.io](https://livekit.io) and use their hosted service:

```nix
services.alicia.livekit = {
  enable = true;
  url = "wss://your-instance.livekit.cloud";
  apiKeyFile = "/var/secrets/alicia/livekit-api-key";
  apiSecretFile = "/var/secrets/alicia/livekit-api-secret";
};
```

**Option B: Self-Hosted LiveKit**

```nix
# Note: This is a simplified example - you'll need a proper LiveKit NixOS package
services.livekit = {
  enable = true;
  settings = {
    port = 7880;
    rtc = {
      port_range_start = 50000;
      port_range_end = 60000;
    };
  };
};

# Open firewall for LiveKit
networking.firewall = {
  allowedTCPPorts = [ 7880 ];
  allowedUDPPortRanges = [
    { from = 50000; to = 60000; }
  ];
};
```

### 5. Nginx Configuration

#### Basic Setup (HTTP only)

```nix
services.alicia.nginx = {
  enable = true;
  serverName = "localhost";
  httpPort = 80;
};
```

#### Production Setup (HTTPS with Let's Encrypt)

```nix
services.alicia.nginx = {
  enable = true;
  serverName = "alicia.example.com";

  # Enable automatic TLS with Let's Encrypt
  enableACME = true;
  forceSSL = true;
  acmeEmail = "admin@example.com";

  # Ports
  httpPort = 80;
  httpsPort = 443;

  # Proxy settings
  clientMaxBodySize = "100m";  # For large audio uploads
  proxyTimeout = "600s";       # For long-running requests

  # WebSocket support (required for LiveKit)
  websocketSupport = true;
};
```

#### Custom Nginx Configuration

```nix
services.alicia.nginx = {
  enable = true;
  serverName = "alicia.example.com";

  extraConfig = ''
    # Rate limiting
    limit_req_zone $binary_remote_addr zone=alicia:10m rate=10r/s;
    limit_req zone=alicia burst=20 nodelay;

    # Additional security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';" always;
  '';
};
```

### 6. Frontend Deployment

#### Option 1: Build Frontend with Nix

Create `nix/packages/frontend.nix`:

```nix
{ pkgs, lib, stdenv, buildNpmPackage }:

buildNpmPackage {
  pname = "alicia-frontend";
  version = "0.1.0";

  src = ../../frontend;

  npmDepsHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

  buildPhase = ''
    npm run build
  '';

  installPhase = ''
    mkdir -p $out
    cp -r dist/* $out/
  '';

  meta = with lib; {
    description = "Alicia web frontend";
    homepage = "https://github.com/longregen/alicia";
  };
}
```

Add to flake.nix:

```nix
packages = {
  alicia = ...;
  alicia-frontend = pkgs.callPackage ./nix/packages/frontend.nix {};
};
```

Use in configuration:

```nix
services.alicia.nginx = {
  enable = true;
  frontendPackage = aliciaFlake.packages.${pkgs.system}.alicia-frontend;
};
```

#### Option 2: Manual Frontend Deployment

```bash
# Build frontend
cd frontend
npm install
npm run build

# Copy to server
sudo mkdir -p /var/www/alicia-frontend
sudo cp -r dist/* /var/www/alicia-frontend/
sudo chown -R nginx:nginx /var/www/alicia-frontend
```

Configure nginx:

```nix
services.alicia.nginx = {
  enable = true;
  frontendPath = "/var/www/alicia-frontend";
};
```

## Configuration Reference

### Complete Service Options

```nix
services.alicia = {
  enable = true;
  package = pkgs.alicia;
  user = "alicia";
  group = "alicia";
  dataDir = "/var/lib/alicia";
  host = "127.0.0.1";
  port = 8080;

  database = {
    url = null;  # Direct database URL
    urlFile = "/var/secrets/alicia/database-url";  # Path to file containing URL
    autoMigrate = true;  # Auto-run migrations on startup
  };

  llm = {
    url = "http://localhost:8000/v1";
    model = "Qwen/Qwen3-8B-AWQ";
    apiKeyFile = "/var/secrets/alicia/llm-api-key";
    maxTokens = 4096;
    temperature = 0.7;
  };

  livekit = {
    enable = true;
    url = "ws://localhost:7880";
    apiKeyFile = "/var/secrets/alicia/livekit-api-key";
    apiSecretFile = "/var/secrets/alicia/livekit-api-secret";
  };

  asr = {
    url = "http://localhost:8001/v1";
    apiKeyFile = "/var/secrets/alicia/asr-api-key";
    model = "whisper-large-v3";
  };

  tts = {
    url = "http://localhost:8002/v1";
    apiKeyFile = "/var/secrets/alicia/tts-api-key";
    model = "kokoro";
    voice = "af_sarah";
  };

  embedding = {
    url = "http://localhost:8003/v1";
    apiKeyFile = "/var/secrets/alicia/embedding-api-key";
    model = "text-embedding-3-small";
    dimensions = 1536;
  };

  openFirewall = false;

  nginx = {
    enable = true;
    serverName = "alicia.example.com";
    frontendPackage = null;
    frontendPath = null;
    enableACME = false;
    forceSSL = false;
    acmeEmail = null;
    listenAddresses = [ "0.0.0.0" ];
    httpPort = 80;
    httpsPort = 443;
    extraConfig = "";
    clientMaxBodySize = "10m";
    proxyTimeout = "300s";
    websocketSupport = true;
  };
};
```

## Secrets Management Patterns

### Environment Variables

Secrets are loaded from files using systemd's `LoadCredential`:

```nix
# In the service configuration, secrets are automatically loaded
# and made available as environment variables:

ALICIA_POSTGRES_URL      # From database.urlFile
ALICIA_LLM_API_KEY       # From llm.apiKeyFile
ALICIA_LIVEKIT_API_KEY   # From livekit.apiKeyFile
ALICIA_LIVEKIT_API_SECRET # From livekit.apiSecretFile
ALICIA_ASR_API_KEY       # From asr.apiKeyFile
ALICIA_TTS_API_KEY       # From tts.apiKeyFile
ALICIA_EMBEDDING_API_KEY # From embedding.apiKeyFile
```

### Secret Rotation

To rotate secrets:

```bash
# 1. Update the secret file
echo "new-secret-value" | sudo tee /var/secrets/alicia/llm-api-key

# 2. Restart the service
sudo systemctl restart alicia
```

### Backup Secrets

**Important**: Always backup your secrets securely:

```bash
# Backup secrets directory
sudo tar -czf alicia-secrets-backup.tar.gz -C /var/secrets alicia
sudo chmod 600 alicia-secrets-backup.tar.gz

# Store in a secure location (encrypted storage, password manager, etc.)
```

## Troubleshooting

### Service Won't Start

Check the service logs:

```bash
sudo journalctl -u alicia -f
```

Common issues:

1. **Database connection failed**
   - Check database is running: `sudo systemctl status postgresql`
   - Verify connection string in `/var/secrets/alicia/database-url`
   - Test connection: `psql "$(sudo cat /var/secrets/alicia/database-url)"`

2. **Permission denied on secrets**
   - Check file ownership: `ls -la /var/secrets/alicia/`
   - Fix permissions: `sudo chown root:alicia /var/secrets/alicia/*`
   - Fix mode: `sudo chmod 640 /var/secrets/alicia/*`

3. **LLM service unreachable**
   - Check LLM service is running
   - Verify URL in configuration
   - Test: `curl http://localhost:8000/v1/models`

### Database Issues

Reset database (WARNING: This deletes all data):

```bash
# Stop service
sudo systemctl stop alicia

# Drop and recreate database
sudo -u postgres psql <<EOF
DROP DATABASE IF EXISTS alicia;
CREATE DATABASE alicia;
GRANT ALL PRIVILEGES ON DATABASE alicia TO alicia;
\c alicia
CREATE EXTENSION IF NOT EXISTS vector;
EOF

# Restart service (will run migrations)
sudo systemctl start alicia
```

### Nginx Issues

Check nginx configuration:

```bash
sudo nginx -t
```

View nginx logs:

```bash
sudo journalctl -u nginx -f
```

### ACME/Let's Encrypt Issues

Common problems:

1. **Certificate request failed**
   - Ensure domain points to your server
   - Check firewall allows port 80
   - Verify `acmeEmail` is set
   - Check logs: `sudo journalctl -u acme-${domain} -f`

2. **Certificate renewal failed**
   - Check nginx is not blocking ACME challenge
   - Verify domain is still valid
   - Manual renewal: `sudo systemctl start acme-${domain}`

### Performance Issues

1. **High memory usage**
   - Check PostgreSQL settings
   - Monitor with: `htop` or `sudo systemctl status alicia`

2. **Slow database queries**
   - Check pgvector is installed: `sudo -u postgres psql alicia -c '\dx'`
   - Analyze queries: `sudo -u postgres psql alicia`
   - Run `EXPLAIN ANALYZE` on slow queries

3. **WebSocket disconnections**
   - Increase nginx timeouts
   - Check network stability
   - Review LiveKit logs

### Debug Mode

Enable debug logging:

```nix
# Add to environment variables
systemd.services.alicia.environment = {
  ALICIA_LOG_LEVEL = "debug";
};
```

## Production Checklist

Before deploying to production:

- [ ] Use encrypted secrets management (sops-nix or agenix)
- [ ] Enable HTTPS with Let's Encrypt
- [ ] Set up automated PostgreSQL backups
- [ ] Configure firewall properly
- [ ] Set up monitoring (Prometheus, Grafana)
- [ ] Configure log aggregation
- [ ] Set up health checks
- [ ] Test disaster recovery procedures
- [ ] Document your specific configuration
- [ ] Set up alerts for service failures
- [ ] Review security hardening options
- [ ] Test with non-root users
- [ ] Verify all secrets are not in version control
- [ ] Set up rate limiting in nginx
- [ ] Test WebSocket connections
- [ ] Verify database migrations work
- [ ] Test backup and restore procedures

### Monitoring Setup Example

```nix
# Prometheus monitoring
services.prometheus = {
  enable = true;
  scrapeConfigs = [
    {
      job_name = "alicia";
      static_configs = [{
        targets = [ "localhost:8080" ];
      }];
    }
  ];
};

# Grafana for visualization
services.grafana = {
  enable = true;
  settings = {
    server.http_port = 3000;
  };
};
```

### Health Checks

Add systemd health checks:

```nix
systemd.services.alicia = {
  serviceConfig = {
    # Restart if service becomes unhealthy
    Restart = "on-failure";
    RestartSec = "10s";

    # Health check
    ExecStartPost = "${pkgs.curl}/bin/curl -f http://localhost:8080/health || exit 1";
  };
};
```

## Support

For additional help:

- GitHub Issues: https://github.com/longregen/alicia/issues
- Documentation: https://github.com/longregen/alicia/tree/main/docs
- NixOS Manual: https://nixos.org/manual/nixos/stable/

## License

See the main repository LICENSE file for details.
