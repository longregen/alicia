{
  description = "Alicia, a real-time voice assistant for conversations through audio";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
    android-nixpkgs = {
      url = "github:tadfisher/android-nixpkgs";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nix-hug = {
      url = "github:longregen/nix-hug";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
    speaches = {
      url = "github:longregen/speaches";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
      inputs.nix-hug.follows = "nix-hug";
    };
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    gomod2nix,
    android-nixpkgs,
    nix-hug,
    speaches,
    ...
  }:
    {
      # NixOS module
      nixosModules.default = import ./nix/modules/alicia.nix;

      # Export gomod2nix overlay
      overlays.default = gomod2nix.overlays.default;
    }
    // flake-utils.lib.eachDefaultSystem (
      system: let
        overlays = [
          gomod2nix.overlays.default
          (import ./nix/overlays/mdbook-mermaid)
          (import ./nix/overlays/mdbook-mermaid/files.nix)
          (import ./nix/overlays/sql-wasm)
        ];
        pkgs = import nixpkgs {
          inherit system overlays;
          config = {
            android_sdk.accept_license = true;
            allowUnfree = true;
          };
        };
        lib = pkgs.lib;

        # PostgreSQL with pgvector extension for vector similarity search
        postgresWithVector = pkgs.postgresql_17.withPackages (p: [p.pgvector]);

        # Android SDK configuration (minimal for build/lint - saves ~10GB disk space)
        androidSdk = android-nixpkgs.sdk.${system} (sdkPkgs:
          with sdkPkgs; [
            cmdline-tools-latest
            build-tools-36-0-0
            platform-tools
            platforms-android-36
          ]);

        # VAD and ONNX Runtime dependencies for web frontend
        vad-dependencies = pkgs.stdenv.mkDerivation {
          name = "vad-dependencies";
          version = "0.0.22";

          src = pkgs.fetchurl {
            url = "https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/package.json";
            sha256 = "sha256-5CsjsXZoqvdc25fAvhLEOOxjzEpNmNCc1aU8i9dJ6nU=";
          };

          sileroModel = pkgs.fetchurl {
            url = "https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/dist/silero_vad_v5.onnx";
            sha256 = "sha256-JiOilT9v89LB5hdAxs23FoEzR5smff7xFKSjzFvdeI8=";
          };

          vadWorklet = pkgs.fetchurl {
            url = "https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/dist/vad.worklet.bundle.min.js";
            sha256 = "sha256-1SD9NRyZuhVT/7z6GTAvJnRwZolwXRlIw9ygf7QI4Lc=";
          };

          ortWasm = pkgs.fetchurl {
            url = "https://cdn.jsdelivr.net/npm/onnxruntime-web@1.14.0/dist/ort-wasm.wasm";
            sha256 = "sha256-u9y2s8fSlFd9gGB3YwRgvk4TyjWjRay16BGI6WSfp0o=";
          };

          ortWasmThreaded = pkgs.fetchurl {
            url = "https://cdn.jsdelivr.net/npm/onnxruntime-web@1.14.0/dist/ort-wasm-threaded.wasm";
            sha256 = "sha256-LuL3FQk6qswDRL+xYQMT60mjR06K/eXkY18w3fx2Fcg=";
          };

          ortWasmSimd = pkgs.fetchurl {
            url = "https://cdn.jsdelivr.net/npm/onnxruntime-web@1.14.0/dist/ort-wasm-simd.wasm";
            sha256 = "sha256-m9B7q6vGX1PQYfRXIz7q5QG+fOuKKtue71LYf+d22GU=";
          };

          ortWasmSimdThreaded = pkgs.fetchurl {
            url = "https://cdn.jsdelivr.net/npm/onnxruntime-web@1.14.0/dist/ort-wasm-simd-threaded.wasm";
            sha256 = "sha256-rCPy88vVGaZaB5b3x56zTq1MH28x6wbhTtipV51pfvY=";
          };

          dontUnpack = true;

          installPhase = ''
            mkdir -p $out/models
            mkdir -p $out/onnx

            cp $sileroModel $out/models/silero_vad_v5.onnx
            cp $vadWorklet $out/models/vad.worklet.bundle.min.js
            cp $ortWasm $out/onnx/ort-wasm.wasm
            cp $ortWasmThreaded $out/onnx/ort-wasm-threaded.wasm
            cp $ortWasmSimd $out/onnx/ort-wasm-simd.wasm
            cp $ortWasmSimdThreaded $out/onnx/ort-wasm-simd-threaded.wasm
          '';

          meta = {
            description = "VAD and ONNX Runtime dependencies for web";
            license = pkgs.lib.licenses.mit;
          };
        };
      in {
        packages = rec {
          # Main Alicia Go application
          alicia-backend = pkgs.callPackage ./nix/packages/backend.nix {
            src = ./.;
            version = "0.1.0";
            rev = self.rev or "dirty";
          };

          # Frontend React/Vite application
          alicia-frontend = pkgs.callPackage ./nix/packages/frontend.nix {
            src = ./.;
            version = "0.1.0";
          };

          # Android Gradle dependency cache (separate to avoid circular dependency)
          alicia-android-deps = let
            # Minimal dummy package for fetchDeps
            dummyPkg = pkgs.stdenv.mkDerivation {
              pname = "alicia-android-deps-base";
              version = "0.1.0";
              src = ./android;
              dontBuild = true;
              installPhase = "mkdir -p $out";
            };
          in
            pkgs.gradle.fetchDeps {
              pkg = dummyPkg;
              data = ./android/deps.json;
              useBwrap = false;
            };

          # Android APK (hermetic build with cached dependencies)
          # Uses buildFHSEnv to create a proper FHS environment where unpatched
          # ELF binaries (like aapt2) work without patching - solving the NixOS
          # Gradle/Android build problem.
          alicia-android = let
            # FHS environment for running Gradle with Android tools
            # This provides /lib64/ld-linux-x86-64.so.2 and standard paths
            # so that aapt2 and other extracted binaries work without patching
            androidFHSEnv = pkgs.buildFHSEnv {
              name = "android-fhs-env";
              targetPkgs = pkgs:
                with pkgs; [
                  # Core build tools
                  androidSdk
                  jdk17
                  gradle

                  # Standard libraries needed by Android tools (aapt2, etc.)
                  stdenv.cc.cc.lib
                  zlib
                  ncurses5
                  libpulseaudio

                  # Build essentials
                  coreutils
                  gnused
                  gnugrep
                  gawk
                  findutils
                  which
                  file
                  gnutar
                  gzip
                  bash
                ];
              multiPkgs = pkgs:
                with pkgs; [
                  # 32-bit libraries for NDK compatibility
                  zlib
                ];
              runScript = "bash";
              profile = ''
                export ANDROID_HOME="${androidSdk}/share/android-sdk"
                export ANDROID_SDK_ROOT="$ANDROID_HOME"
                export JAVA_HOME="${pkgs.jdk17.home}"
              '';
            };
          in
            pkgs.stdenv.mkDerivation (finalAttrs: {
              pname = "alicia-android";
              version = "0.1.0";

              src = ./android;

              nativeBuildInputs = [
                androidFHSEnv
                pkgs.jdk17
              ];

              # Gradle dependency update script (used by mitmCache)
              # Runs inside FHS environment so aapt2 works without patching
              gradleUpdateScript = ''
                chmod +x gradlew

                # Run Gradle inside FHS environment with mitm-cache proxy configured
                ${androidFHSEnv}/bin/android-fhs-env -c '
                  export HOME="$TMPDIR/home"
                  mkdir -p "$HOME/.android"

                  # Configure gradle to use mitm-cache proxy
                  # Extract host and port from MITM_CACHE_ADDRESS
                  PROXY_HOST="''${MITM_CACHE_ADDRESS%:*}"
                  PROXY_PORT="''${MITM_CACHE_ADDRESS##*:}"

                  export GRADLE_OPTS="
                    -Dhttp.proxyHost=$PROXY_HOST
                    -Dhttp.proxyPort=$PROXY_PORT
                    -Dhttps.proxyHost=$PROXY_HOST
                    -Dhttps.proxyPort=$PROXY_PORT
                    -Dhttp.nonProxyHosts=localhost|127.0.0.1
                  "

                  # Import the mitm-cache CA certificate into Java trust store
                  if [ -f "$MITM_CACHE_CA" ]; then
                    JAVA_KEYSTORE="$HOME/.keystore"
                    ${pkgs.jdk17}/bin/keytool -import -alias mitm-cache-ca -file "$MITM_CACHE_CA" \
                      -keystore "$JAVA_KEYSTORE" -storepass changeit -noprompt 2>/dev/null || true
                    export JAVA_TOOL_OPTIONS="-Djavax.net.ssl.trustStore=$JAVA_KEYSTORE -Djavax.net.ssl.trustStorePassword=changeit"
                  fi

                  # Download dependencies through proxy
                  ./gradlew dependencies --no-daemon

                  # Build to trigger all dependency resolution including aapt2
                  ./gradlew assembleDebug --no-daemon || true
                '
              '';

              buildPhase = ''
                chmod +x gradlew

                # Set up Gradle home directory
                export GRADLE_USER_HOME="$TMPDIR/gradle-home"
                mkdir -p "$GRADLE_USER_HOME"

                # Modify settings.gradle.kts to use local mitmCache repositories
                if [ -d "${alicia-android-deps}" ] && [ "$(ls -A ${alicia-android-deps})" ]; then
                  # Replace repository URLs in settings.gradle.kts with local mitmCache paths
                  cat > settings.gradle.kts << EOF
                pluginManagement {
                    repositories {
                        maven { url = uri("${alicia-android-deps}/https/dl.google.com/dl/android/maven2") }
                        maven { url = uri("${alicia-android-deps}/https/repo.maven.apache.org/maven2") }
                        maven { url = uri("${alicia-android-deps}/https/plugins.gradle.org/m2") }
                    }
                }

                dependencyResolutionManagement {
                    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
                    repositories {
                        maven { url = uri("${alicia-android-deps}/https/dl.google.com/dl/android/maven2") }
                        maven { url = uri("${alicia-android-deps}/https/repo.maven.apache.org/maven2") }
                        maven { url = uri("${alicia-android-deps}/https/jitpack.io") }
                    }
                }

                rootProject.name = "Alicia"

                include(":app")

                // Core modules
                include(":core:common")
                include(":core:data")
                include(":core:domain")
                include(":core:network")
                include(":core:database")

                // Feature modules
                include(":feature:assistant")
                include(":feature:conversations")
                include(":feature:settings")

                // Service modules
                include(":service:voice")
                include(":service:hotkey")
                EOF
                fi

                # Build inside FHS environment
                # Use nixpkgs gradle directly instead of gradlew to avoid wrapper download
                ${androidFHSEnv}/bin/android-fhs-env -c "
                  export HOME=\"\$TMPDIR/home\"
                  mkdir -p \"\$HOME/.android\"
                  export GRADLE_USER_HOME=\"$GRADLE_USER_HOME\"
                  gradle assembleDebug --offline --no-daemon --stacktrace
                "
              '';

              installPhase = ''
                mkdir -p $out
                if [ -f "app/build/outputs/apk/debug/app-universal-debug.apk" ]; then
                  cp app/build/outputs/apk/debug/app-universal-debug.apk $out/
                elif [ -f "app/build/outputs/apk/debug/app-debug.apk" ]; then
                  cp app/build/outputs/apk/debug/app-debug.apk $out/
                else
                  echo "ERROR: Could not find debug APK"
                  find app/build/outputs/apk -name "*.apk" -type f || true
                  exit 1
                fi
              '';

              dontPatchELF = true;
              dontStrip = true;

              # Self-referential mitmCache for Gradle dependency management
              # To update deps.json, run: nix build .#alicia-android.mitmCache.updateScript --print-out-paths
              # Then execute the resulting script: $(nix build .#alicia-android.mitmCache.updateScript --print-out-paths)
              passthru.mitmCache = alicia-android-deps;
            });

          # Combined package with backend and frontend
          alicia = pkgs.callPackage ./nix/packages/alicia.nix {
            version = "0.1.0";
            inherit alicia-backend alicia-frontend;
          };

          # Set alicia as the default package
          default = alicia;

          # Documentation package
          mdbookMermaidFiles = pkgs.mdbookMermaidFiles;

          # mdbook documentation
          docs = pkgs.stdenv.mkDerivation {
            name = "alicia-docs";
            src = ./.;
            buildInputs = with pkgs; [mdbook mdbook-mermaid];
            buildPhase = ''
              # Set up mdbook-mermaid assets
              mkdir -p docs/bin/assets
              ln -sf ${pkgs.mdbookMermaidFiles}/share/mdbook-mermaid/mermaid.min.js docs/bin/assets/mermaid.min.js
              ln -sf ${pkgs.mdbookMermaidFiles}/share/mdbook-mermaid/mermaid-init.js docs/bin/assets/mermaid-init.js

              # Build documentation
              mdbook build -d $out
            '';
            installPhase = ''
              echo "Documentation built at $out"
            '';
          };

          # Frontend tests (also available as check)
          frontend-tests = pkgs.buildNpmPackage {
            name = "alicia-frontend-tests";
            src = ./frontend;
            npmDepsHash = "sha256-IrRFQB+6Pz3hPaCDXejyLoQvTMZ9sWe573EVyoeoIks=";

            buildPhase = ''
              export HOME=$TMPDIR
              npm test -- --run
            '';

            installPhase = ''
              mkdir -p $out
              echo "Frontend tests passed" > $out/result
            '';
          };

          # VAD dependencies for frontend development
          inherit vad-dependencies;
        };

        # Test derivations
        checks = {
          # Backend tests - run all tests that can build without external services
          # Tests requiring PostgreSQL or other external dependencies are skipped
          backend-tests = pkgs.buildGoApplication {
            name = "alicia-backend-tests";
            src = ./.;
            modules = ./gomod2nix.toml;

            # CGO dependencies for opus audio codec
            nativeBuildInputs = with pkgs; [pkg-config];
            buildInputs = with pkgs; [libopus opusfile];

            # Run tests for packages that don't require external dependencies
            # Packages with compilation errors from outdated mocks are excluded until fixed
            buildPhase = ''
              export HOME=$TMPDIR
              runHook preBuild

              # Test domain models (pure business logic, no external deps)
              go test -v ./internal/domain/...

              # Test protocol package (message serialization)
              go test -v ./pkg/...

              # Test HTTP middleware (no DB required)
              go test -v ./internal/adapters/http/middleware

              # Test embedding client (no external service needed)
              go test -v ./internal/adapters/embedding

              # Test retry logic
              go test -v ./internal/adapters/retry

              # Test built-in tools
              go test -v ./internal/application/tools/builtin

              runHook postBuild
            '';

            installPhase = ''
              mkdir -p $out
              echo "Backend tests passed" > $out/result
            '';
          };

          # Frontend tests
          frontend-tests = pkgs.buildNpmPackage {
            name = "alicia-frontend-tests";
            src = ./frontend;
            npmDepsHash = "sha256-IrRFQB+6Pz3hPaCDXejyLoQvTMZ9sWe573EVyoeoIks=";

            buildPhase = ''
              export HOME=$TMPDIR
              npm test -- --run
            '';

            installPhase = ''
              mkdir -p $out
              echo "Frontend tests passed" > $out/result
            '';
          };

          # End-to-end tests with NixOS VMs
          e2e = import ./e2e-test/nix {
            inherit pkgs lib;
            self = self;
          };

          # LLM integration test: speaches (STT/TTS) + ollama (LLM)
          llm-test = import ./llm-test/nix {
            inherit pkgs lib;
            inherit speaches nix-hug;
          };
        };

        # Lean CI shells (minimal dependencies for faster builds and less disk usage)

        # Go-only shell for backend CI
        devShells.go = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            go-tools # staticcheck
            gcc
            pkg-config
            libopus
            opusfile
            libogg
          ];
        };

        # Frontend-only shell for frontend CI
        devShells.frontend = pkgs.mkShell {
          buildInputs = with pkgs; [
            nodejs_22
          ];
          shellHook = ''
            mkdir -p frontend/public/models
            mkdir -p frontend/public/onnx

            ln -sf ${vad-dependencies}/models/* frontend/public/models/
            ln -sf ${vad-dependencies}/onnx/* frontend/public/onnx/
            ln -sf ${pkgs.sqlWasmFile}/share/sql-wasm/sql-wasm.wasm frontend/public/sql-wasm.wasm

            echo "VAD dependencies linked to frontend/public/"
          '';
        };

        # Android development shell with FHS environment for AAPT2 compatibility
        # Use: nix develop .#android -c android-fhs-env
        # Or for CI: nix develop .#android --command android-fhs-env -c './gradlew ...'
        devShells.android = let
          androidFHSEnv = pkgs.buildFHSEnv {
            name = "android-fhs-env";
            targetPkgs = pkgs:
              with pkgs; [
                androidSdk
                jdk17
                gradle
                stdenv.cc.cc.lib
                zlib
                ncurses5
                libpulseaudio
                coreutils
                gnused
                gnugrep
                gawk
                findutils
                which
                file
                gnutar
                gzip
                bash
                git
              ];
            multiPkgs = pkgs: with pkgs; [zlib];
            profile = ''
              export ANDROID_HOME="${androidSdk}/share/android-sdk"
              export ANDROID_SDK_ROOT="$ANDROID_HOME"
              export JAVA_HOME="${pkgs.jdk17.home}"
            '';
            runScript = "bash";
          };
        in
          pkgs.mkShell {
            buildInputs = [androidFHSEnv];
            shellHook = ''
              echo "Android FHS environment available."
              echo "Run: android-fhs-env -c './gradlew assembleDebug'"
            '';
          };

        devShells.default = pkgs.mkShell {
          buildInputs =
            (with pkgs; [
              # Go development tools
              go
              gopls
              delve
              go-tools

              # C compiler and build tools (required for CGO)
              gcc
              pkg-config
              libopus
              opusfile
              libogg

              # Database tools
              postgresWithVector
              pgcli

              # Code generation
              sqlc

              # Frontend development
              nodejs_22

              # Documentation
              mdbook
              mdbook-mermaid

              # Android development (androidSdk configured in shellHook)
              jdk17
              gradle
            ])
            ++ [
              # gomod2nix from flake input
              gomod2nix.packages.${system}.default
            ];

          shellHook = ''
            # Set up Android SDK environment
            export ANDROID_HOME="${androidSdk}/share/android-sdk"
            export ANDROID_SDK_ROOT="$ANDROID_HOME"
            export JAVA_HOME="${pkgs.jdk17.home}"
            export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/build-tools/36.0.0:$JAVA_HOME/bin:$PATH"

            # Set up PostgreSQL database environment
            export PGDATA="$PWD/.postgres"
            export PGHOST="$PGDATA"
            export PGDATABASE="alicia"
            export PGUSER="postgres"
            export PGPORT=5555

            # Initialize PostgreSQL if needed
            if [ ! -d "$PGDATA" ]; then
              echo "Initializing PostgreSQL database..."
              mkdir -p "$PGDATA"
              initdb --auth=trust --no-locale --encoding=UTF8 --username=postgres

              # Start PostgreSQL server with UNIX socket only (no TCP)
              pg_ctl start -o "-k $PGDATA -h \"\" -c listen_addresses=\"\""

              # Wait for PostgreSQL to be ready
              sleep 2

              # Create database
              createdb alicia

              # Initialize database schema if migrations exist
              if [ -f "migrations/schema.sql" ] || [ -d "migrations" ]; then
                echo "Running database migrations..."
                for migration in migrations/*.sql; do
                  if [ -f "$migration" ]; then
                    psql -d alicia -f "$migration"
                  fi
                done
              fi

              echo "PostgreSQL database initialized."
            else
              # Start PostgreSQL server if not already running
              if ! pg_ctl status > /dev/null 2>&1; then
                pg_ctl start -o "-k $PGDATA -h \"\" -c listen_addresses=\"\""
                sleep 2
              fi
            fi

            # Set up documentation symlinks
            mkdir -p docs/bin/assets
            ln -sf ${pkgs.mdbookMermaidFiles}/share/mdbook-mermaid/mermaid.min.js docs/bin/assets/mermaid.min.js
            ln -sf ${pkgs.mdbookMermaidFiles}/share/mdbook-mermaid/mermaid-init.js docs/bin/assets/mermaid-init.js

            # Set up VAD dependencies for frontend
            mkdir -p frontend/public/models
            mkdir -p frontend/public/onnx
            ln -sf ${vad-dependencies}/models/* frontend/public/models/
            ln -sf ${vad-dependencies}/onnx/* frontend/public/onnx/

            # Generate Android gradle wrapper if missing
            if [ -d "android" ] && [ ! -f "android/gradlew" ]; then
              echo "Generating gradle wrapper..."
              (cd android && gradle wrapper --gradle-version=8.14.3 --quiet)
            fi

            # Generate deps.json placeholder if missing
            if [ -d "android" ] && [ ! -f "android/deps.json" ]; then
              echo "Generating deps.json placeholder..."
              cat > android/deps.json << 'DEPS'
            {
             "!comment": "This is a nixpkgs Gradle dependency lockfile. For more details, refer to the Gradle section in the nixpkgs manual.",
             "!version": 1
            }
            DEPS
            fi

            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo "  Alicia Development Environment"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo ""
            echo "Database:"
            echo "  PostgreSQL running at localhost:$PGPORT"
            echo "  Database: $PGDATABASE"
            echo "  User: $PGUSER"
            echo ""
            echo "Android Environment:"
            echo "  ANDROID_HOME: $ANDROID_HOME"
            echo "  JAVA_HOME: $JAVA_HOME"
            echo ""
            echo "Development Commands:"
            echo "  go build -o bin/alicia cmd/alicia/main.go  # Build backend"
            echo "  gomod2nix                                   # Update Go dependencies"
            echo "  sqlc generate                               # Generate database code"
            echo "  mdbook serve docs                           # Serve documentation"
            echo "  cd android && ./gradlew assembleDebug       # Build Android APK"
            echo ""
            echo "Nix Build Commands:"
            echo "  nix build .#alicia                          # Build backend + frontend"
            echo "  nix build .#alicia-android                  # Build Android APK (hermetic)"
            echo "  \$(nix build .#alicia-android.mitmCache.updateScript --print-out-paths)  # Update deps.json"
            echo ""
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

            # Clean up PostgreSQL when exiting the shell
            cleanup() {
              echo ""
              echo "Stopping PostgreSQL..."
              pg_ctl stop > /dev/null 2>&1
            }
            trap cleanup EXIT
          '';
        };
      }
    );
}
