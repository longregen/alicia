{ config, lib, ... }:

with lib;

let
  cfg = config.services.alicia.nginx;
  aliciaCfg = config.services.alicia;
in
{
  options.services.alicia.nginx = {
    enable = mkEnableOption "nginx reverse proxy for Alicia";

    serverName = mkOption {
      type = types.str;
      default = "localhost";
      example = "alicia.example.com";
      description = "Domain name for the nginx virtual host.";
    };

    frontendPackage = mkOption {
      type = types.nullOr types.package;
      default = null;
      description = "Frontend package to serve. If null, nginx will only proxy to the backend.";
    };

    frontendPath = mkOption {
      type = types.nullOr types.path;
      default = null;
      description = "Path to frontend static files. If null and frontendPackage is set, will use frontendPackage.";
    };

    enableACME = mkOption {
      type = types.bool;
      default = false;
      description = "Enable Let's Encrypt ACME for TLS certificates.";
    };

    forceSSL = mkOption {
      type = types.bool;
      default = false;
      description = "Force redirect HTTP to HTTPS.";
    };

    acmeEmail = mkOption {
      type = types.nullOr types.str;
      default = null;
      example = "admin@example.com";
      description = "Email address for ACME certificate registration.";
    };

    listenAddresses = mkOption {
      type = types.listOf types.str;
      default = [ "0.0.0.0" ];
      description = "IP addresses to listen on.";
    };

    httpPort = mkOption {
      type = types.port;
      default = 80;
      description = "HTTP port to listen on.";
    };

    httpsPort = mkOption {
      type = types.port;
      default = 443;
      description = "HTTPS port to listen on (when TLS is enabled).";
    };

    extraConfig = mkOption {
      type = types.lines;
      default = "";
      description = "Extra nginx configuration for the Alicia virtual host.";
    };

    clientMaxBodySize = mkOption {
      type = types.str;
      default = "10m";
      description = "Maximum allowed size of client request body.";
    };

    proxyTimeout = mkOption {
      type = types.str;
      default = "300s";
      description = "Timeout for proxy connections.";
    };

    websocketSupport = mkOption {
      type = types.bool;
      default = true;
      description = "Enable WebSocket support for LiveKit connections.";
    };

    livekitUrl = mkOption {
      type = types.str;
      default = "ws://localhost:7880";
      description = "LiveKit WebSocket URL for frontend configuration.";
    };
  };

  config = mkIf (aliciaCfg.enable && cfg.enable) {
    assertions = [
      {
        assertion = cfg.enableACME -> cfg.acmeEmail != null;
        message = "services.alicia.nginx.acmeEmail must be set when enableACME is true";
      }
      {
        assertion = cfg.enableACME -> cfg.serverName != "localhost";
        message = "services.alicia.nginx.serverName must be a valid domain when enableACME is true";
      }
    ];

    # Configure ACME if enabled
    security.acme = mkIf cfg.enableACME {
      acceptTerms = true;
      defaults.email = cfg.acmeEmail;
    };

    # Configure nginx
    services.nginx = {
      enable = true;

      recommendedGzipSettings = true;
      recommendedOptimisation = true;
      recommendedProxySettings = true;
      recommendedTlsSettings = true;

      virtualHosts.${cfg.serverName} = {
        enableACME = cfg.enableACME;
        forceSSL = cfg.forceSSL;

        listen = map (addr: {
          addr = addr;
          port = cfg.httpPort;
        }) cfg.listenAddresses
        ++ optionals (cfg.enableACME || cfg.forceSSL) (map (addr: {
          addr = addr;
          port = cfg.httpsPort;
          ssl = true;
        }) cfg.listenAddresses);

        locations = {
          # Proxy API requests to the backend
          "/api/" = {
            proxyPass = "http://${aliciaCfg.host}:${toString aliciaCfg.port}/api/";
            proxyWebsockets = cfg.websocketSupport;
            extraConfig = ''
              proxy_set_header X-Real-IP $remote_addr;
              proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
              proxy_set_header X-Forwarded-Proto $scheme;
              proxy_set_header Host $host;

              proxy_connect_timeout ${cfg.proxyTimeout};
              proxy_send_timeout ${cfg.proxyTimeout};
              proxy_read_timeout ${cfg.proxyTimeout};
              send_timeout ${cfg.proxyTimeout};

              client_max_body_size ${cfg.clientMaxBodySize};
            '';
          };

          # Proxy WebSocket connections for LiveKit
          "/ws" = mkIf aliciaCfg.livekit.enable {
            proxyPass = "http://${aliciaCfg.host}:${toString aliciaCfg.port}/ws";
            proxyWebsockets = true;
            extraConfig = ''
              proxy_set_header X-Real-IP $remote_addr;
              proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
              proxy_set_header X-Forwarded-Proto $scheme;
              proxy_set_header Host $host;

              # WebSocket specific settings
              proxy_http_version 1.1;
              proxy_set_header Upgrade $http_upgrade;
              proxy_set_header Connection "upgrade";

              # Longer timeout for WebSocket connections
              proxy_connect_timeout 7d;
              proxy_send_timeout 7d;
              proxy_read_timeout 7d;
            '';
          };

          # Frontend configuration endpoint
          "/config.json" = mkIf aliciaCfg.livekit.enable {
            extraConfig = ''
              default_type application/json;
              return 200 '{"livekitUrl": "${cfg.livekitUrl}"}';
              add_header Cache-Control "no-store, no-cache, must-revalidate";
            '';
          };

          # Serve frontend static files
          "/" = mkIf (cfg.frontendPackage != null || cfg.frontendPath != null) {
            root = if cfg.frontendPath != null
                   then cfg.frontendPath
                   else "${cfg.frontendPackage}";
            index = "index.html";
            tryFiles = "$uri $uri/ /index.html";
            extraConfig = ''
              # Cache static assets
              location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
                expires 1y;
                add_header Cache-Control "public, immutable";
              }

              # No caching for HTML
              location ~* \.html$ {
                expires -1;
                add_header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0";
              }
            '';
          };
        } // optionalAttrs (cfg.extraConfig != "") {
          "~ ^/custom/" = {
            extraConfig = cfg.extraConfig;
          };
        };

        extraConfig = ''
          # Security headers
          add_header X-Frame-Options "SAMEORIGIN" always;
          add_header X-Content-Type-Options "nosniff" always;
          add_header X-XSS-Protection "1; mode=block" always;
          add_header Referrer-Policy "strict-origin-when-cross-origin" always;
        '';
      };
    };

    # Open firewall ports if needed
    networking.firewall = mkIf aliciaCfg.openFirewall {
      allowedTCPPorts = [ cfg.httpPort ]
        ++ optionals (cfg.enableACME || cfg.forceSSL) [ cfg.httpsPort ];
    };
  };
}
