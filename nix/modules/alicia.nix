{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.services.alicia;
in
{
  imports = [
    ./alicia-nginx.nix
  ];

  options.services.alicia = {
    enable = mkEnableOption "Alicia voice assistant service";

    package = mkOption {
      type = types.package;
      default = pkgs.alicia or (throw "Alicia package not found. Make sure to overlay it in your configuration.");
      defaultText = literalExpression "pkgs.alicia";
      description = "The Alicia package to use.";
    };

    user = mkOption {
      type = types.str;
      default = "alicia";
      description = "User account under which Alicia runs.";
    };

    group = mkOption {
      type = types.str;
      default = "alicia";
      description = "Group under which Alicia runs.";
    };

    dataDir = mkOption {
      type = types.path;
      default = "/var/lib/alicia";
      description = "Data directory for Alicia.";
    };

    host = mkOption {
      type = types.str;
      default = "0.0.0.0";
      description = "The host address to bind the server to.";
    };

    port = mkOption {
      type = types.port;
      default = 8080;
      description = "The port to run the HTTP server on.";
    };

    # Database configuration
    database = {
      url = mkOption {
        type = types.nullOr types.str;
        default = null;
        description = "PostgreSQL connection URL. If null, uses SQLite in dataDir.";
      };

      urlFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "File containing the PostgreSQL connection URL (for secrets management).";
      };

      autoMigrate = mkOption {
        type = types.bool;
        default = true;
        description = "Automatically run database migrations on startup.";
      };

      failOnMigrationError = mkOption {
        type = types.bool;
        default = true;
        description = ''
          Whether to fail service startup if a database migration fails.
          When true (default), any migration error will prevent the service from starting.
          When false, migration errors are logged but the service continues to start.
          Setting this to false is not recommended for production.
        '';
      };
    };

    # LLM configuration
    llm = {
      url = mkOption {
        type = types.str;
        default = "http://localhost:8000/v1";
        description = "URL of the LLM server (vLLM/LiteLLM).";
      };

      model = mkOption {
        type = types.str;
        default = "Qwen/Qwen3-8B-AWQ";
        description = "LLM model name to use.";
      };

      apiKeyFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "File containing the LLM API key.";
      };

      maxTokens = mkOption {
        type = types.int;
        default = 4096;
        description = "Maximum number of tokens for LLM responses.";
      };

      temperature = mkOption {
        type = types.float;
        default = 0.7;
        description = "Temperature parameter for LLM (0.0 to 2.0).";
      };
    };

    # LiveKit configuration
    livekit = {
      enable = mkEnableOption "LiveKit real-time voice support";

      url = mkOption {
        type = types.str;
        default = "ws://localhost:7880";
        description = "LiveKit server WebSocket URL.";
      };

      apiKeyFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "File containing the LiveKit API key.";
      };

      apiSecretFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "File containing the LiveKit API secret.";
      };
    };

    # ASR (Automatic Speech Recognition) configuration
    asr = {
      url = mkOption {
        type = types.nullOr types.str;
        default = null;
        description = "URL of the ASR service (Whisper via speaches).";
      };

      apiKeyFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "File containing the ASR API key.";
      };

      model = mkOption {
        type = types.str;
        default = "whisper-large-v3";
        description = "ASR model name to use.";
      };
    };

    # TTS (Text-to-Speech) configuration
    tts = {
      url = mkOption {
        type = types.nullOr types.str;
        default = null;
        description = "URL of the TTS service (Kokoro via speaches).";
      };

      apiKeyFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "File containing the TTS API key.";
      };

      model = mkOption {
        type = types.str;
        default = "kokoro";
        description = "TTS model name to use.";
      };

      voice = mkOption {
        type = types.str;
        default = "af_sarah";
        description = "TTS voice to use.";
      };
    };

    # Embedding configuration
    embedding = {
      url = mkOption {
        type = types.nullOr types.str;
        default = null;
        description = "URL of the embedding service.";
      };

      apiKeyFile = mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "File containing the embedding API key.";
      };

      model = mkOption {
        type = types.str;
        default = "text-embedding-3-small";
        description = "Embedding model name to use.";
      };

      dimensions = mkOption {
        type = types.int;
        default = 1536;
        description = "Embedding dimensions.";
      };
    };

    environmentFile = mkOption {
      type = types.nullOr types.path;
      default = null;
      description = "Environment file containing additional variables.";
    };

    openFirewall = mkOption {
      type = types.bool;
      default = false;
      description = "Whether to open the firewall for the Alicia port.";
    };

    mode = mkOption {
      type = types.enum [ "server" "agent" "both" ];
      default = "both";
      description = "Which services to run: server only, agent only, or both";
    };
  };

  config = mkIf cfg.enable {
    # Assertions to validate configuration
    assertions = [
      {
        assertion = cfg.llm.url != "" && cfg.llm.url != null;
        message = "services.alicia.llm.url must be set to a valid LLM server URL";
      }
      {
        assertion = !cfg.livekit.enable || (cfg.livekit.url != "" && cfg.livekit.url != null);
        message = "services.alicia.livekit.url must be set when LiveKit is enabled";
      }
      {
        assertion = !cfg.livekit.enable || (cfg.livekit.apiKeyFile != null && cfg.livekit.apiSecretFile != null);
        message = "services.alicia.livekit.apiKeyFile and apiSecretFile must be set when LiveKit is enabled";
      }
      {
        assertion = !cfg.database.autoMigrate || (cfg.database.url != null || cfg.database.urlFile != null);
        message = "services.alicia.database.url or database.urlFile must be provided when autoMigrate is enabled (migrations require PostgreSQL)";
      }
    ];


    # Create user and group
    users.users.${cfg.user} = {
      isSystemUser = true;
      group = cfg.group;
      home = cfg.dataDir;
      createHome = true;
      description = "Alicia voice assistant service user";
    };

    users.groups.${cfg.group} = {};

    # Create data directory
    systemd.tmpfiles.rules = [
      "d '${cfg.dataDir}' 0750 ${cfg.user} ${cfg.group} - -"
    ];

    # systemd service
    systemd.services.alicia = mkIf (cfg.mode == "server" || cfg.mode == "both") {
      description = "Alicia Voice Assistant Server";
      wantedBy = [ "multi-user.target" ];
      after = [ "network-online.target" ]
        ++ optional (cfg.database.url != null && (hasInfix "localhost" cfg.database.url || hasInfix "/run/postgresql" cfg.database.url)) "postgresql.service";
      wants = [ "network-online.target" ];

      serviceConfig = {
        Type = "simple";
        User = cfg.user;
        Group = cfg.group;
        WorkingDirectory = cfg.dataDir;

        # Note: ExecStart is generated from the script block below

        # Health check - verify service is responding after startup
        ExecStartPost = "${pkgs.bash}/bin/bash -c 'for i in {1..30}; do ${pkgs.curl}/bin/curl -sf http://localhost:${toString cfg.port}/health && exit 0; sleep 1; done; exit 1'";

        # Restart policy
        Restart = "on-failure";
        RestartSec = "10s";

        # Security hardening
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadWritePaths = [ cfg.dataDir ];
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictAddressFamilies = [ "AF_UNIX" "AF_INET" "AF_INET6" ];
        RestrictNamespaces = true;
        LockPersonality = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        RemoveIPC = true;
        PrivateMounts = true;

        # System call filtering
        SystemCallFilter = [ "@system-service" ];
        SystemCallErrorNumber = "EPERM";
        SystemCallArchitectures = "native";

        # Capabilities
        CapabilityBoundingSet = "";
        AmbientCapabilities = "";

        # Resource limits
        LimitNOFILE = "65536";

        # Load secrets via LoadCredential
        LoadCredential =
          (optional (cfg.database.urlFile != null) "db-url:${cfg.database.urlFile}")
          ++ (optional (cfg.llm.apiKeyFile != null) "llm-api-key:${cfg.llm.apiKeyFile}")
          ++ (optional (cfg.livekit.apiKeyFile != null) "livekit-api-key:${cfg.livekit.apiKeyFile}")
          ++ (optional (cfg.livekit.apiSecretFile != null) "livekit-api-secret:${cfg.livekit.apiSecretFile}")
          ++ (optional (cfg.asr.apiKeyFile != null) "asr-api-key:${cfg.asr.apiKeyFile}")
          ++ (optional (cfg.tts.apiKeyFile != null) "tts-api-key:${cfg.tts.apiKeyFile}")
          ++ (optional (cfg.embedding.apiKeyFile != null) "embedding-api-key:${cfg.embedding.apiKeyFile}");
      } // (optionalAttrs (cfg.database.autoMigrate && (cfg.database.url != null || cfg.database.urlFile != null)) {
        # Run database migrations before starting the service
        ExecStartPre = pkgs.writeShellScript "alicia-migrate" ''
          set -e

          # Get database URL from credentials or environment
          ${optionalString (cfg.database.urlFile != null) ''
            DB_URL=$(cat "$CREDENTIALS_DIRECTORY/db-url")
          ''}
          ${optionalString (cfg.database.url != null && cfg.database.urlFile == null) ''
            DB_URL="${cfg.database.url}"
          ''}

          echo "Running database migrations..."

          # Track migration status
          MIGRATION_FAILED=0

          # Run each migration in order
          for migration in ${cfg.package}/share/alicia/migrations/*_*.up.sql; do
            if [ -f "$migration" ]; then
              echo "Running migration: $(basename $migration)"

              if ! ${pkgs.postgresql}/bin/psql "$DB_URL" -f "$migration" 2>&1; then
                echo "ERROR: Migration failed: $(basename $migration)" >&2
                MIGRATION_FAILED=1

                ${optionalString cfg.database.failOnMigrationError ''
                  echo "ERROR: Aborting due to migration failure (failOnMigrationError=true)" >&2
                  exit 1
                ''}
                ${optionalString (!cfg.database.failOnMigrationError) ''
                  echo "WARNING: Continuing despite migration failure (failOnMigrationError=false)" >&2
                ''}
              fi
            fi
          done

          if [ $MIGRATION_FAILED -eq 0 ]; then
            echo "Database migrations completed successfully"
          else
            echo "Database migrations completed with errors"
          fi
        '';
      });

      environment = {
        # Server configuration
        ALICIA_SERVER_HOST = cfg.host;
        ALICIA_SERVER_PORT = toString cfg.port;
        ALICIA_STATIC_DIR = "${cfg.package}/share/alicia/frontend";

        # Database configuration
        ALICIA_DB_PATH = mkIf (cfg.database.url == null && cfg.database.urlFile == null)
          "${cfg.dataDir}/alicia.db";
        ALICIA_POSTGRES_URL = mkIf (cfg.database.url != null) cfg.database.url;

        # LLM configuration
        ALICIA_LLM_URL = cfg.llm.url;
        ALICIA_LLM_MODEL = cfg.llm.model;
        ALICIA_LLM_MAX_TOKENS = toString cfg.llm.maxTokens;
        ALICIA_LLM_TEMPERATURE = toString cfg.llm.temperature;

        # LiveKit configuration
      } // optionalAttrs cfg.livekit.enable {
        ALICIA_LIVEKIT_URL = cfg.livekit.url;
      } // optionalAttrs (cfg.asr.url != null) {
        # ASR configuration
        ALICIA_ASR_URL = cfg.asr.url;
        ALICIA_ASR_MODEL = cfg.asr.model;
      } // optionalAttrs (cfg.tts.url != null) {
        # TTS configuration
        ALICIA_TTS_URL = cfg.tts.url;
        ALICIA_TTS_MODEL = cfg.tts.model;
        ALICIA_TTS_VOICE = cfg.tts.voice;
      } // optionalAttrs (cfg.embedding.url != null) {
        # Embedding configuration
        ALICIA_EMBEDDING_URL = cfg.embedding.url;
        ALICIA_EMBEDDING_MODEL = cfg.embedding.model;
        ALICIA_EMBEDDING_DIMENSIONS = toString cfg.embedding.dimensions;
      };

      # Script to load credentials and start service
      script = ''
        ${optionalString (cfg.database.urlFile != null) ''
          export ALICIA_POSTGRES_URL=$(cat "$CREDENTIALS_DIRECTORY/db-url")
        ''}
        ${optionalString (cfg.llm.apiKeyFile != null) ''
          export ALICIA_LLM_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/llm-api-key")
        ''}
        ${optionalString (cfg.livekit.enable && cfg.livekit.apiKeyFile != null) ''
          export ALICIA_LIVEKIT_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/livekit-api-key")
        ''}
        ${optionalString (cfg.livekit.enable && cfg.livekit.apiSecretFile != null) ''
          export ALICIA_LIVEKIT_API_SECRET=$(cat "$CREDENTIALS_DIRECTORY/livekit-api-secret")
        ''}
        ${optionalString (cfg.asr.url != null && cfg.asr.apiKeyFile != null) ''
          export ALICIA_ASR_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/asr-api-key")
        ''}
        ${optionalString (cfg.tts.url != null && cfg.tts.apiKeyFile != null) ''
          export ALICIA_TTS_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/tts-api-key")
        ''}
        ${optionalString (cfg.embedding.url != null && cfg.embedding.apiKeyFile != null) ''
          export ALICIA_EMBEDDING_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/embedding-api-key")
        ''}

        exec ${cfg.package}/bin/alicia serve
      '';
    };

    # Alicia Agent service for LiveKit voice support
    systemd.services.alicia-agent = mkIf (cfg.livekit.enable && (cfg.mode == "agent" || cfg.mode == "both")) {
      description = "Alicia Voice Assistant Agent (LiveKit)";
      wantedBy = [ "multi-user.target" ];
      after = [ "network-online.target" ]
        ++ optional (cfg.database.url != null && (hasInfix "localhost" cfg.database.url || hasInfix "/run/postgresql" cfg.database.url)) "postgresql.service"
        ++ optional (cfg.mode == "both") "alicia.service";
      wants = [ "network-online.target" ];

      serviceConfig = {
        Type = "simple";
        User = cfg.user;
        Group = cfg.group;
        WorkingDirectory = cfg.dataDir;

        # Note: ExecStart is generated from the script block below

        # Restart policy
        Restart = "on-failure";
        RestartSec = "10s";

        # Security hardening (same as main service)
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadWritePaths = [ cfg.dataDir ];
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictAddressFamilies = [ "AF_UNIX" "AF_INET" "AF_INET6" ];
        RestrictNamespaces = true;
        LockPersonality = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        RemoveIPC = true;
        PrivateMounts = true;

        # System call filtering
        SystemCallFilter = [ "@system-service" ];
        SystemCallErrorNumber = "EPERM";
        SystemCallArchitectures = "native";

        # Capabilities
        CapabilityBoundingSet = "";
        AmbientCapabilities = "";

        # Resource limits
        LimitNOFILE = "65536";

        # Load secrets via LoadCredential
        LoadCredential =
          (optional (cfg.database.urlFile != null) "db-url:${cfg.database.urlFile}")
          ++ (optional (cfg.llm.apiKeyFile != null) "llm-api-key:${cfg.llm.apiKeyFile}")
          ++ (optional (cfg.livekit.apiKeyFile != null) "livekit-api-key:${cfg.livekit.apiKeyFile}")
          ++ (optional (cfg.livekit.apiSecretFile != null) "livekit-api-secret:${cfg.livekit.apiSecretFile}")
          ++ (optional (cfg.asr.apiKeyFile != null) "asr-api-key:${cfg.asr.apiKeyFile}")
          ++ (optional (cfg.tts.apiKeyFile != null) "tts-api-key:${cfg.tts.apiKeyFile}")
          ++ (optional (cfg.embedding.apiKeyFile != null) "embedding-api-key:${cfg.embedding.apiKeyFile}");
      };

      environment = {
        # Database configuration
        ALICIA_DB_PATH = mkIf (cfg.database.url == null && cfg.database.urlFile == null)
          "${cfg.dataDir}/alicia.db";
        ALICIA_POSTGRES_URL = mkIf (cfg.database.url != null) cfg.database.url;

        # LLM configuration
        ALICIA_LLM_URL = cfg.llm.url;
        ALICIA_LLM_MODEL = cfg.llm.model;
        ALICIA_LLM_MAX_TOKENS = toString cfg.llm.maxTokens;
        ALICIA_LLM_TEMPERATURE = toString cfg.llm.temperature;

        # LiveKit configuration (required for agent)
        ALICIA_LIVEKIT_URL = cfg.livekit.url;
      } // optionalAttrs (cfg.asr.url != null) {
        # ASR configuration
        ALICIA_ASR_URL = cfg.asr.url;
        ALICIA_ASR_MODEL = cfg.asr.model;
      } // optionalAttrs (cfg.tts.url != null) {
        # TTS configuration
        ALICIA_TTS_URL = cfg.tts.url;
        ALICIA_TTS_MODEL = cfg.tts.model;
        ALICIA_TTS_VOICE = cfg.tts.voice;
      } // optionalAttrs (cfg.embedding.url != null) {
        # Embedding configuration
        ALICIA_EMBEDDING_URL = cfg.embedding.url;
        ALICIA_EMBEDDING_MODEL = cfg.embedding.model;
        ALICIA_EMBEDDING_DIMENSIONS = toString cfg.embedding.dimensions;
      };

      # Script to load credentials and start agent
      script = ''
        ${optionalString (cfg.database.urlFile != null) ''
          export ALICIA_POSTGRES_URL=$(cat "$CREDENTIALS_DIRECTORY/db-url")
        ''}
        ${optionalString (cfg.llm.apiKeyFile != null) ''
          export ALICIA_LLM_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/llm-api-key")
        ''}
        ${optionalString (cfg.livekit.apiKeyFile != null) ''
          export ALICIA_LIVEKIT_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/livekit-api-key")
        ''}
        ${optionalString (cfg.livekit.apiSecretFile != null) ''
          export ALICIA_LIVEKIT_API_SECRET=$(cat "$CREDENTIALS_DIRECTORY/livekit-api-secret")
        ''}
        ${optionalString (cfg.asr.url != null && cfg.asr.apiKeyFile != null) ''
          export ALICIA_ASR_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/asr-api-key")
        ''}
        ${optionalString (cfg.tts.url != null && cfg.tts.apiKeyFile != null) ''
          export ALICIA_TTS_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/tts-api-key")
        ''}
        ${optionalString (cfg.embedding.url != null && cfg.embedding.apiKeyFile != null) ''
          export ALICIA_EMBEDDING_API_KEY=$(cat "$CREDENTIALS_DIRECTORY/embedding-api-key")
        ''}

        exec ${cfg.package}/bin/alicia agent
      '';
    };

    # Optionally open firewall
    networking.firewall = mkIf cfg.openFirewall {
      allowedTCPPorts = [ cfg.port ];
    };
  };
}
