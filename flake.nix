{
  description = "Alicia - a real-time voice assistant";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
    android-nixpkgs = {
      url = "github:tadfisher/android-nixpkgs";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, gomod2nix, android-nixpkgs, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        overlays = [
          gomod2nix.overlays.default
          (import ./nix/overlays/sql-wasm)
        ];
        pkgs = import nixpkgs { inherit system overlays; };
        lib = pkgs.lib;

        postgresWithVector = pkgs.postgresql_17.withPackages (p: [ p.pgvector p.pgsql-http ]);

        db-migrate = pkgs.writeShellScriptBin "db-migrate" ''
          for f in api/migrations/*.sql; do
            psql -q -d alicia -f "$f" 2>/dev/null || true
          done
          echo "Migrations applied."
        '';

        goSrcFilter = src: name: lib.cleanSourceWith {
          inherit src name;
          filter = path: type:
            let baseName = builtins.baseNameOf path;
            in type == "directory" ||
               lib.hasSuffix ".go" baseName ||
               baseName == "go.mod" ||
               baseName == "go.sum" ||
               baseName == "gomod2nix.toml";
        };

        # Pre-filtered local module sources (shared across packages)
        sharedSrc = goSrcFilter ./shared "shared-src";
        langfuseSrc = goSrcFilter ./pkg/langfuse "langfuse-src";
        otelSrc = goSrcFilter ./pkg/otel "otel-src";

        # Vendor fixup: copy local replace modules into the vendor directory
        vendorLocalModules = deps: ''
          if [ -L vendor ]; then
            target=$(readlink -f vendor)
            rm vendor
            cp -rL "$target" vendor
          fi
          chmod -R u+w vendor || true
          mkdir -p vendor/github.com/longregen/alicia/pkg vendor/github.com/longregen/alicia
          ${lib.concatStringsSep "\n" (map (d: "cp -r ${d.src} vendor/github.com/longregen/alicia/${d.dst}") deps)}
        '';

        mkMcpPackage = { pname, subdir }: let
          assembledSrc = pkgs.runCommand "${pname}-src" {} ''
            mkdir -p $out/${subdir} $out/pkg/langfuse $out/pkg/otel $out/shared
            cp ${./mcp/go.mod} $out/go.mod
            cp ${./mcp/go.sum} $out/go.sum
            cp -r ${goSrcFilter ./mcp/${subdir} "mcp-${subdir}-src"}/* $out/${subdir}/
            cp -r ${langfuseSrc}/* $out/pkg/langfuse/
            cp -r ${otelSrc}/* $out/pkg/otel/
            cp -r ${sharedSrc}/* $out/shared/
          '';
        in pkgs.buildGoApplication {
          inherit pname;
          version = "0.1.0";
          src = assembledSrc;
          modules = ./mcp/gomod2nix.toml;
          subPackages = [subdir];
          preBuild = vendorLocalModules [
            { src = "pkg/langfuse"; dst = "pkg/langfuse"; }
            { src = "pkg/otel"; dst = "pkg/otel"; }
            { src = "shared"; dst = "shared"; }
          ];
        };

        # Android SDK and build tooling
        androidSdk = android-nixpkgs.sdk.${system} (sdkPkgs: with sdkPkgs; [
          cmdline-tools-latest
          build-tools-35-0-0
          platform-tools
          platforms-android-35
          platforms-android-24
          emulator
          ndk-26-1-10909125
        ]);

        sileroVadModel = pkgs.fetchurl {
          url = "https://github.com/snakers4/silero-vad/raw/master/src/silero_vad/data/silero_vad.onnx";
          sha256 = "1qw8hyfjfrac2xz2ns4895dv5pp8hndnyzg6jhm2k7jhyhi3l58s";
        };

        voskModelSmallEn = pkgs.fetchzip {
          url = "https://alphacephei.com/vosk/models/vosk-model-small-en-us-0.15.zip";
          sha256 = "1rl65n2maayggnzi811x6zingkd1ny2z7p0fvcbfaprbz5khz2h8";
        };

        androidFhsEnv = pkgs.buildFHSEnv {
          name = "android-fhs-env";
          targetPkgs = pkgs: with pkgs; [
            androidSdk
            jdk17
            gradle
            glibc
            zlib
            ncurses5
            stdenv.cc.cc.lib
          ];
          runScript = "bash";
          profile = ''
            export ANDROID_HOME="${androidSdk}/share/android-sdk"
            export ANDROID_SDK_ROOT="$ANDROID_HOME"
            export JAVA_HOME="${pkgs.jdk17.home}"
            export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/build-tools/35.0.0:$JAVA_HOME/bin:$PATH"
          '';
        };

        androidBuildScript = pkgs.writeShellScriptBin "build-apk" ''
          set -e

          export ANDROID_HOME="${androidSdk}/share/android-sdk"
          export ANDROID_SDK_ROOT="$ANDROID_HOME"
          export JAVA_HOME="${pkgs.jdk17.home}"
          export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/build-tools/35.0.0:$JAVA_HOME/bin:$PATH"

          # Ensure model assets are present
          if [ ! -f app/src/main/assets/silero_vad.onnx ]; then
            echo "Copying Silero VAD model..."
            cp ${sileroVadModel} app/src/main/assets/silero_vad.onnx
          fi
          if [ ! -d app/src/main/assets/vosk-models/small-en-us/am ]; then
            echo "Copying Vosk model..."
            mkdir -p app/src/main/assets/vosk-models/small-en-us
            cp -r ${voskModelSmallEn}/. app/src/main/assets/vosk-models/small-en-us/
          fi

          echo "Building ARM v7 debug APK..."
          chmod +x ./gradlew
          ./gradlew assembleDebug \
            -Pandroid.injected.abi=armeabi-v7a \
            --parallel \
            --build-cache \
            --no-daemon \
            --stacktrace

          echo "Build complete!"
          APK_PATH=""
          if [ -f "app/build/outputs/apk/debug/app-armeabi-v7a-debug.apk" ]; then
            APK_PATH="app/build/outputs/apk/debug/app-armeabi-v7a-debug.apk"
          elif [ -f "app/build/outputs/apk/armeabi-v7a/debug/app-armeabi-v7a-debug.apk" ]; then
            APK_PATH="app/build/outputs/apk/armeabi-v7a/debug/app-armeabi-v7a-debug.apk"
          elif [ -f "app/build/outputs/apk/debug/app-debug.apk" ]; then
            APK_PATH="app/build/outputs/apk/debug/app-debug.apk"
          fi

          if [ -n "$APK_PATH" ]; then
            echo "APK location: $APK_PATH"
            ls -lh "$APK_PATH"
          else
            echo "ERROR: APK not found at expected locations!"
            find app/build/outputs/apk -name "*.apk" -type f || true
            exit 1
          fi
        '';
      in
      {
        packages = {
          api = let
            src = pkgs.runCommand "api-src" {} ''
              mkdir -p $out/pkg/langfuse $out/pkg/otel $out/shared
              cp -r ${goSrcFilter ./api "api-src"}/* $out/
              cp -r ${langfuseSrc}/* $out/pkg/langfuse/
              cp -r ${otelSrc}/* $out/pkg/otel/
              cp -r ${sharedSrc}/* $out/shared/
            '';
          in pkgs.callPackage ./nix/packages/backend.nix {
            inherit src;
            migrations = ./api/migrations;
            version = "0.1.0";
            rev = self.rev or "dirty";
            preBuild = vendorLocalModules [
              { src = "pkg/langfuse"; dst = "pkg/langfuse"; }
              { src = "pkg/otel"; dst = "pkg/otel"; }
              { src = "shared"; dst = "shared"; }
            ];
          };

          web = let
            src = lib.cleanSourceWith {
              src = ./web;
              name = "web-src";
              filter = path: type:
                let baseName = builtins.baseNameOf path;
                in baseName != ".env" && baseName != ".env.local";
            };
          in pkgs.callPackage ./nix/packages/frontend.nix {
            inherit src;
            version = "0.1.0";
            vad-dependencies = pkgs.stdenv.mkDerivation {
              name = "vad-dependencies";
              dontUnpack = true;
              sileroModel = pkgs.fetchurl {
                url = "https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/dist/silero_vad_v5.onnx";
                sha256 = "sha256-JiOilT9v89LB5hdAxs23FoEzR5smff7xFKSjzFvdeI8=";
              };
              vadWorklet = pkgs.fetchurl {
                url = "https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/dist/vad.worklet.bundle.min.js";
                sha256 = "sha256-1SD9NRyZuhVT/7z6GTAvJnRwZolwXRlIw9ygf7QI4Lc=";
              };
              vadBundle = pkgs.fetchurl {
                url = "https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/dist/bundle.min.js";
                sha256 = "sha256-XMxdF+8E89clng0bMXsjqGj6Nm5WjBHTK5VvfX026cc=";
              };
              ortJs = pkgs.fetchurl {
                url = "https://cdn.jsdelivr.net/npm/onnxruntime-web@1.14.0/dist/ort.min.js";
                sha256 = "sha256-v3I5nuDnSSIrx8DdHPPaQbZvx++J9DcSeUQZnWBez/4=";
              };
              ortWasm = pkgs.fetchurl {
                url = "https://cdn.jsdelivr.net/npm/onnxruntime-web@1.14.0/dist/ort-wasm.wasm";
                sha256 = "sha256-u9y2s8fSlFd9gGB3YwRgvk4TyjWjRay16BGI6WSfp0o=";
              };
              ortWasmSimd = pkgs.fetchurl {
                url = "https://cdn.jsdelivr.net/npm/onnxruntime-web@1.14.0/dist/ort-wasm-simd.wasm";
                sha256 = "sha256-m9B7q6vGX1PQYfRXIz7q5QG+fOuKKtue71LYf+d22GU=";
              };
              installPhase = ''
                mkdir -p $out/models $out/onnx $out/js/lib
                cp $sileroModel $out/models/silero_vad_v5.onnx
                cp $vadWorklet $out/models/vad.worklet.bundle.min.js
                cp $ortWasm $out/onnx/ort-wasm.wasm
                cp $ortWasmSimd $out/onnx/ort-wasm-simd.wasm
                cp $ortJs $out/js/lib/ort.js
                cp $vadBundle $out/js/lib/vad.bundle.min.js
              '';
            };
          };

          voice = let
            src = pkgs.runCommand "voice-src" {} ''
              mkdir -p $out/pkg/otel $out/shared
              cp -r ${goSrcFilter ./voice "voice-src"}/* $out/
              cp -r ${otelSrc}/* $out/pkg/otel/
              cp -r ${sharedSrc}/* $out/shared/
            '';
          in pkgs.callPackage ./nix/packages/voice.nix {
            inherit src;
            version = "0.1.0";
            preBuild = vendorLocalModules [
              { src = "pkg/otel"; dst = "pkg/otel"; }
              { src = "shared"; dst = "shared"; }
            ];
          };

          agent = let
            src = pkgs.runCommand "agent-src" {} ''
              mkdir -p $out/pkg/langfuse $out/pkg/otel $out/shared
              cp -r ${goSrcFilter ./agent "agent-src"}/* $out/
              cp -r ${langfuseSrc}/* $out/pkg/langfuse/
              cp -r ${otelSrc}/* $out/pkg/otel/
              cp -r ${sharedSrc}/* $out/shared/
            '';
          in pkgs.callPackage ./nix/packages/agent.nix {
            inherit src;
            version = "0.1.0";
            preBuild = vendorLocalModules [
              { src = "pkg/langfuse"; dst = "pkg/langfuse"; }
              { src = "pkg/otel"; dst = "pkg/otel"; }
              { src = "shared"; dst = "shared"; }
            ];
          };

          monitor = pkgs.callPackage ./nix/packages/monitor.nix {
            src = goSrcFilter ./monitor "monitor-src";
            version = "0.1.0";
          };

          mcp-garden = mkMcpPackage { pname = "mcp-garden"; subdir = "garden"; };
          mcp-web = let
            unwrapped = mkMcpPackage { pname = "mcp-web"; subdir = "web"; };
          in pkgs.runCommand "mcp-web" {
            nativeBuildInputs = [ pkgs.makeWrapper ];
          } ''
            mkdir -p $out/bin
            makeWrapper ${unwrapped}/bin/web $out/bin/web \
              --prefix PATH : "${lib.makeBinPath [ pkgs.chromium pkgs.firefox ]}"
          '';
          mcp-deno-calc = mkMcpPackage { pname = "mcp-deno-calc"; subdir = "deno-calc"; };

          android = pkgs.stdenv.mkDerivation rec {
            pname = "alicia-assistant";
            version = "1.0.0";

            src = ./android;

            nativeBuildInputs = [
              androidSdk
              pkgs.jdk17
              pkgs.gradle
            ];

            mitmCache = pkgs.gradle.fetchDeps {
              pkg = self.packages.${system}.android;
              data = ./android/deps.json;
              useBwrap = false;
            };

            buildPhase = ''
              export HOME=$(mktemp -d)
              export ANDROID_HOME="${androidSdk}/share/android-sdk"
              export ANDROID_SDK_ROOT="$ANDROID_HOME"
              export JAVA_HOME="${pkgs.jdk17.home}"
              export GRADLE_USER_HOME=$(mktemp -d)
              export TMPDIR=$(mktemp -d)
              export GRADLE_OPTS="-Djava.io.tmpdir=$TMPDIR -Dorg.gradle.native.dir=$TMPDIR/native"
              mkdir -p $TMPDIR/native

              mkdir -p app/src/main/assets
              cp ${sileroVadModel} app/src/main/assets/silero_vad.onnx
              mkdir -p app/src/main/assets/vosk-models/small-en-us
              cp -r ${voskModelSmallEn}/. app/src/main/assets/vosk-models/small-en-us/

              cp -a $mitmCache/. $GRADLE_USER_HOME/
              chmod -R u+w $GRADLE_USER_HOME

              gradle assembleDebug \
                -Pandroid.injected.abi=armeabi-v7a \
                -Pandroid.aapt2FromMavenOverride="$ANDROID_HOME/build-tools/35.0.0/aapt2" \
                --no-daemon \
                --stacktrace
            '';

            installPhase = ''
              mkdir -p $out
              find app/build/outputs/apk -name "*.apk" -exec cp {} $out/ \;
            '';
          };

          default = self.packages.${system}.api;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go development
            go
            go-tools
            gopls
            delve
            gcc
            pkg-config
            libopus
            opusfile
            libogg

            # Node.js development
            nodejs_22

            # Database
            postgresWithVector
            pgcli

            # Tools
            watchexec
            gomod2nix.packages.${system}.default
          ] ++ [
            db-migrate
            self.packages.${system}.mcp-garden
            self.packages.${system}.mcp-web
            self.packages.${system}.mcp-deno-calc
            deno
          ];

          shellHook = ''
            export PGDATA="$PWD/.postgres"
            export PGHOST="$PGDATA"
            export PGDATABASE="alicia"
            export PGUSER="postgres"
            export PGPORT=5555
            export DATABASE_URL="postgres://postgres@localhost:5555/alicia?host=$PGDATA&sslmode=disable"

            if [ ! -d "$PGDATA" ]; then
              echo "Initializing database..."
              mkdir -p "$PGDATA"
              initdb --auth=trust --no-locale --encoding=UTF8 --username=postgres >/dev/null
              echo "host all all 0.0.0.0/0 trust" >> "$PGDATA/pg_hba.conf"
              pg_ctl start -l "$PGDATA/postgres.log" -o "-k $PGDATA -p $PGPORT -c listen_addresses='0.0.0.0'" >/dev/null
              sleep 2
              createdb alicia
              db-migrate
            else
              if ! pg_ctl status > /dev/null 2>&1; then
                pg_ctl start -l "$PGDATA/postgres.log" -o "-k $PGDATA -p $PGPORT -c listen_addresses='0.0.0.0'" >/dev/null
                sleep 2
              fi
            fi

            echo ""
            echo "Alicia Development Shell"
            echo "========================"
            echo ""
            echo "Subprojects:"
            echo "  api/    - Go backend (cd api && go build ./...)"
            echo "  web/    - React frontend (cd web && npm install && npm run dev)"
            echo "  agent/  - Go agent (cd agent && go build ./...)"
            echo "  voice/  - Go voice helper (cd voice && go build ./...)"
            echo "  mcp/    - MCP tools (cd mcp && go build ./...)"
            echo ""
            echo "Database: PostgreSQL at localhost:$PGPORT"
            echo ""

            cleanup() {
              echo "Stopping PostgreSQL..."
              pg_ctl stop > /dev/null 2>&1 || true
            }
            trap cleanup EXIT
          '';
        };

        devShells.android = pkgs.mkShell {
          buildInputs = [
            androidSdk
            pkgs.jdk17
            pkgs.gradle
            androidBuildScript
            androidFhsEnv
            pkgs.maestro
            pkgs.go
            pkgs.gomobile
          ];

          shellHook = ''
            export ANDROID_HOME="${androidSdk}/share/android-sdk"
            export ANDROID_SDK_ROOT="$ANDROID_HOME"
            export ANDROID_NDK_HOME="$ANDROID_HOME/ndk/26.1.10909125"
            export JAVA_HOME="${pkgs.jdk17.home}"
            export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$ANDROID_HOME/build-tools/35.0.0:$JAVA_HOME/bin:$PATH"

            # Maestro settings
            export MAESTRO_CLI_NO_ANALYTICS=1
            export MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED=true

            cat > local.properties << EOF
sdk.dir=$ANDROID_HOME
EOF

            echo ""
            echo "  Alicia Voice Assistant - Android Build Environment"
            echo ""
            echo "Commands:"
            echo "  ./gradlew assembleDebug    - Build debug APK"
            echo "  build-apk                  - Build ARM v7 debug APK"
            echo "  maestro test e2e/          - Run E2E tests (device required)"
            echo "  ./build-libtailscale.sh     - Build libtailscale.aar from source"
            echo ""
            echo "NixOS users: If AAPT2 fails, use the FHS environment:"
            echo "  android-fhs-env -c './gradlew assembleDebug'"
            echo ""
          '';
        };

        apps.android-e2e = let
          e2eTests = pkgs.stdenv.mkDerivation {
            name = "alicia-e2e-tests";
            src = ./android/e2e;
            installPhase = ''
              mkdir -p $out
              cp -r *.yaml $out/ 2>/dev/null || true
            '';
          };
          e2eScript = pkgs.writeShellScriptBin "android-e2e" ''
            set -e

            export PATH="${pkgs.maestro}/bin:${androidSdk}/share/android-sdk/platform-tools:$PATH"
            export ANDROID_HOME="${androidSdk}/share/android-sdk"
            export ANDROID_SDK_ROOT="$ANDROID_HOME"
            export MAESTRO_CLI_NO_ANALYTICS=1
            export MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED=true

            # Check device
            if ! adb devices | grep -q "device$"; then
              echo "Error: No Android device connected"
              echo "Connect a device or start an emulator first"
              exit 1
            fi

            # Install APK if not already installed or if --install flag
            APK_PATH="${self.packages.${system}.android}/app-arm64-v8a-debug.apk"
            if [ "$1" = "--install" ] || ! adb shell pm list packages | grep -q "com.alicia.assistant"; then
              echo "Installing APK..."
              adb install -r "$APK_PATH"
            fi

            # Run tests
            echo "Running E2E tests..."
            cd ${e2eTests}

            if [ -n "$1" ] && [ "$1" != "--install" ]; then
              maestro test "$1"
            else
              for test in *.yaml; do
                echo "=== Running $test ==="
                maestro test "$test" || exit 1
              done
            fi

            echo "All tests passed!"
          '';
        in {
          type = "app";
          program = "${e2eScript}/bin/android-e2e";
        };
      }
    );
}
