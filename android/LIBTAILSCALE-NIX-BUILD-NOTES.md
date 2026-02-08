# libtailscale.aar Nix Build Notes

Research notes documenting attempts to build `libtailscale.aar` inside Nix. This is a
record of what was tried, what failed, and what was tried next. It is not a solution
document.

---

## Background

`libtailscale.aar` is a Go library produced by running `gomobile bind` against the
[tailscale-android](https://github.com/nickchal/tailscale-android) repository. The
manual build script (`android/build-libtailscale.sh`) performs these steps:

1. Clones tailscale-android.
2. Uses Tailscale's custom Go toolchain (1.25.6).
3. Installs `gomobile@latest`.
4. Runs `go get golang.org/x/mobile@latest && go mod tidy && go mod download`.
5. Runs `gomobile bind -target android -androidapi 26 -o libtailscale.aar ./libtailscale`.

The goal was to reproduce this build inside a Nix derivation for reproducibility.

### Relevant versions from tailscale-android go.mod

| Dependency | Version |
|---|---|
| Go | 1.25.6 |
| `golang.org/x/mobile` | v0.0.0-20240806205939-81131f6468ab |
| `tailscale.com` | v1.95.0-pre.0.20260128165433-72f736134d74 |

`tailscale.com`'s own go.mod pulls in roughly 470 dependencies (kubernetes, docker,
helm, golangci-lint, etc.).

### Source hashes

| Artifact | Hash |
|---|---|
| tailscale-android (rev `afc295d...`) | `sha256-KpmJuBojaKdX6LcEtnSR1JGT6G1qYszlSKhDT2N5hVE=` |
| Go 1.25.6 (`go.dev/dl/go1.25.6.linux-amd64.tar.gz`) | `sha256-8CK2qteONivLqbC5TQmtWMWnDGujt1gpBfq6v1/gGBo=` |

---

## Attempt 1: buildGoModule with vendorHash

### Approach

Use `pkgs.buildGoModule` to build libtailscale in the standard Nix way.

### Problem: Go version mismatch

nixpkgs shipped Go 1.25.5 but tailscale-android requires 1.25.6. Created a
`go_1_25_6` override:

```nix
go_1_25_6 = pkgs.go.overrideAttrs {
  src = pkgs.fetchurl {
    url = "https://go.dev/dl/go1.25.6.linux-amd64.tar.gz";
    hash = "sha256-8CK2qteONivLqbC5TQmtWMWnDGujt1gpBfq6v1/gGBo=";
  };
};
```

### Problem: buildGoModule FOD tries to compile

`buildGoModule`'s `.goModules` fixed-output derivation (FOD) attempted to compile Go
code during the module-fetch phase, which requires the Android NDK for CGO. Worked
around this by using the `.goModules` attribute directly, which only fetches modules
without compiling.

### Problem: vendor directory does not include tool binaries

The vendor hash produced by `buildGoModule` creates a vendor directory containing only
*imported packages* — not tool binaries like `cmd/gomobile`. Since `gomobile` is a
build tool (not a library import), it was absent from the vendor tree and could not be
built from it.

This ended Attempt 1.

---

## Attempt 2: Custom FOD for module cache + regular derivation for build

### Approach

Split the build into two derivations:

1. **`tailscaleGoModules` (FOD):** Runs `go mod download`, copies the entire
   `GOMODCACHE` to `$out`.
2. **`libtailscale` (regular derivation):** Copies the module cache from the FOD,
   builds gomobile/gobind from it, runs `gomobile bind`.

The build derivation sets `GOPROXY="file://${tailscaleGoModules}/cache/download,off"`
so that Go resolves all modules from the local file cache with network access disabled.

Initial FOD hash: `sha256-FJRl/E4mt9By/z092jIkjeho1amIqjhYlZLYXBA5bdM=`.

---

### Error: gomobile could not find gobind in PATH

**Symptom:** gomobile internally invokes the `gobind` binary and failed because it was
not on PATH.

**Fix:** After building gomobile and gobind, added `$GOPATH/bin` to PATH.

---

### Error: "go: error reading go.mod: missing module declaration"

**Symptom:** `gomobile bind` failed with a go.mod parse error.

**Root cause (discovered through investigation):**

`gomobile bind` works internally as follows:

1. Runs `gobind` to generate Go/Java binding source into a temp directory.
2. Calls `writeGoMod()` which runs `go list -m -json all` to enumerate every module,
   then creates a **synthetic** go.mod (module name `"gobind"`) listing all of those
   modules as requires.
3. Calls `goModTidyAt()` which runs `go mod tidy` in the temp directory.
4. Calls `goBuildAt()` which runs `go build` in the temp directory.

The `go list -m -json all` step was **failing silently**. When it fails,
`getModuleVersions` returns `nil, nil`, so `writeGoMod` writes an empty/malformed
go.mod (no `module` declaration). Then `go mod tidy` fails reading it.

---

### Discovery: gomobile strips environment variables from subprocesses

**This turned out to be the central problem for the entire Attempt 2 approach.**

Investigation of the gomobile source code revealed that `goModTidyAt()` and
`goBuildAt()` set `cmd.Env = env` on their subprocesses, where `env` is a **minimal**
set of variables:

- CGO variables (GOOS, GOARCH, CGO_ENABLED, CC, CXX, CGO_CFLAGS, CGO_LDFLAGS)
- GOPATH
- GOMODCACHE (added by `goModTidyAt`)

Setting `cmd.Env` **replaces** the entire inherited environment. This means GOPROXY,
GONOSUMCHECK, GONOSUMDB, PATH, HOME, and everything else is stripped from the
subprocess.

So even though we set `GOPROXY=file://...` in the parent derivation, the `go`
subprocesses spawned by gomobile never see it.

#### Relevant gomobile source locations

- `goModTidyAt` — `build.go:348-364`
- `goBuildAt` — `build.go:325-346`
- `buildAndroidSO` — `bind_androidapp.go`
- `getModuleVersions` — `bind.go:228-305`
- `writeGoMod` — `bind.go:307-335`

All located under:
`$GOMODCACHE/golang.org/x/mobile@v0.0.0-20240806205939-81131f6468ab/cmd/gomobile/`

---

### Fix: Go wrapper script

Created a `writeShellScriptBin "go"` wrapper that re-exports GOPROXY and related
variables before exec'ing the real go binary:

```nix
goForGomobile = pkgs.writeShellScriptBin "go" ''
  export GOPROXY="file:///.../cache/download,off"
  export GONOSUMCHECK="*"
  export GONOSUMDB="*"
  exec ${go_1_25_6}/bin/go "$@"
'';
```

Because the wrapper uses an absolute Nix store shebang (`#!/nix/store/.../bash`), it
works even when gomobile's `cmd.Env` does not include PATH. When gomobile resolves `go`
from PATH at process start, it finds the wrapper. When it spawns subprocesses with
`cmd.Env = env`, the wrapper's own script body re-exports the needed variables before
exec'ing the real go.

**Result:** This fixed the `writeGoMod` step. `go list -m -json all` succeeded and
produced a valid synthetic go.mod.

---

### Error: `go list -m all` reports "module lookup disabled by GOPROXY=off"

After the wrapper fix, `writeGoMod` worked (synthetic go.mod was valid), but subsequent
steps failed because modules were missing from the file proxy.

**Investigation:**

- `go mod download` (run in the FOD) downloads only modules directly listed in
  tailscale-android's go.mod — approximately 89 modules.
- `go list -m all` returns the **pruned module graph**, which is larger because it
  includes transitive dependencies.
- `tailscale.com`'s go.mod alone has ~470 dependencies. The pruned transitive closure
  includes a subset of those, many of which were not downloaded by `go mod download`.

Checked the download cache: all 89 direct modules had `.info`, `.mod`, and `.zip`
files. But modules pulled in transitively through tailscale.com were absent.

---

### Fix: Download the full transitive graph in the FOD

Changed the FOD from just `go mod download` to:

```bash
go mod download
go list -m -json all > /dev/null
```

`go list -m -json all` forces Go to resolve the full pruned module graph, fetching
`.info` files for all transitive modules.

New FOD hash: `sha256-rUgICi1AwEZiYS00x9Jn0JVIB8hnIEPy5HNBH6bJ6tM=`.

---

### Error: .info files present but .mod files missing

**Symptom:** `go mod tidy` (inside gomobile) failed because it needs `.mod` files to
resolve the dependency graph, but `go list -m -json all` only downloaded `.info` files,
not `.mod` files.

**Example:** `4d63.com/gocheckcompilerdirectives/@v/v1.2.1.info` existed in the cache
but `v1.2.1.mod` did not.

---

### Fix: Download .mod files for all modules in the graph

Changed the FOD to:

```bash
go mod download
go list -m -f '{{if not .Main}}{{.Path}}@{{.Version}}{{end}}' all | xargs go mod download
```

This downloads full module data (`.info`, `.mod`, `.zip`) for every module in the
pruned graph.

New FOD hash: `sha256-XIQbY2l7ADiaV4SHAvcXbIROLQ60aLuQUmiFbV/CKfE=`.

---

### Error: `go mod tidy` fails with a DIFFERENT missing module

**Symptom:**

```
go: github.com/Microsoft/hcsshim@v0.12.9: module lookup disabled by GOPROXY=off
```

**Investigation:**

- `hcsshim` v0.12.9 is **not** in the pruned module graph — the pruned graph had
  v0.9.12.
- v0.12.9 comes from `github.com/moby/buildkit@v0.20.2`'s go.mod, a deep transitive
  dependency of tailscale.com's tool dependencies.

**Root cause — module graph expansion feedback loop:**

Downloading more modules in the FOD caused the module graph to **expand** when
evaluated in the build derivation. The mechanism:

1. Go uses lazy module loading (Go 1.17+): it only reads go.mod files for direct
   dependencies, and recursively for dependencies that themselves use go < 1.17.
2. `go mod download` downloads ~89 modules from go.mod.
3. `go list -m all` returns the pruned graph — say ~200 modules. We download those.
4. Now there are ~200 `.mod` files in GOMODCACHE. Go can read them and discover MORE
   transitive dependencies that were previously pruned.
5. The next `go list -m all` (inside gomobile in the build derivation) returns a LARGER
   graph — say ~300 modules. Some of those are not in the cache.
6. Downloading those ~300 would expand the graph further. And so on.

Additionally, the synthetic go.mod generated by gomobile lists every module from
`go list -m all` as a **direct** require. This defeats lazy loading entirely — Go must
read the go.mod of every direct dependency to resolve the graph.

This creates a vicious cycle where downloading more modules causes the build to need
even more modules.

---

### Fix attempt: Patch gomobile to skip `go mod tidy`

Patched `goModTidyAt` in `build.go` via sed to replace the function body. Instead of
running `go mod tidy`, it creates an empty `go.sum`:

```go
func goModTidyAt(at string) error {
    return os.WriteFile(at+"/go.sum", nil, 0644)
}
```

Also added `GOFLAGS=-mod=mod` to the go wrapper so `go build` can update go.sum from
the local cache.

The gomobile source file was patched at:
`$GOMODCACHE/golang.org/x/mobile@v0.0.0-20240806205939-81131f6468ab/cmd/gomobile/build.go`

---

### Error: `undefined: filepath` in patched build.go

The initial sed replacement used `filepath.Join()` but `build.go` does not import
`path/filepath`. Fixed by switching to `at+"/go.sum"` (build.go does import `os`).

---

### Error: `go build` also fails with the same missing module

**Symptom:** After patching `goModTidyAt`, the tidy step was skipped successfully. But
`goBuildAt` (which runs `go build`) also failed with:

```
go: github.com/Microsoft/hcsshim@v0.12.9: module lookup disabled by GOPROXY=off
```

**Root cause:** Same as above. The synthetic go.mod (written by `writeGoMod` via
`getModuleVersions` via `go list -m -json all`) lists ALL modules from the expanded
graph as **direct** requires. With Go's lazy loading, even `go build` needs to read
go.mod files for all direct dependencies to determine the complete module graph.
`hcsshim` v0.12.9 is listed as a direct require in the synthetic go.mod but is not in
the `file://` proxy.

---

### Observation: TODO comment in gomobile source

Line 235 of `bind.go` contains:

```go
// TODO(hyangah): probably we don't need to add all the dependencies.
```

The `getModuleVersions` function runs `go list -m -json all` and adds **every** module
as a require in the synthetic go.mod. Most of these modules are unnecessary for the
actual gobind build (e.g., kubernetes, docker, helm, linter tools from tailscale.com's
dependency tree).

---

### Approach considered but not implemented: Patch getModuleVersions

Idea: replace the `go list -m -json all` call with direct parsing of go.mod using
`modfile.Parse` (which `bind.go` already imports). This would produce a synthetic
go.mod listing only the modules from tailscale-android's go.mod (all ~89 of which are
present in the download cache from `go mod download`).

Would also need to patch `goModTidyAt` to copy `go.sum` from the source directory
instead of running `go mod tidy`.

This was not implemented due to the fragility of multi-line sed replacement of Go
functions.

This ended Attempt 2.

---

## Attempt 3: Single FOD for the entire build

### Approach

Make the `libtailscale` derivation itself a fixed-output derivation (FOD) with network
access. The entire build runs inside a single FOD:

1. `go mod download`
2. Build gomobile/gobind from the module cache.
3. Run `gomobile bind`.

Since the FOD has network access, all of gomobile's internal `go list`, `go mod tidy`,
and `go build` subprocesses can download whatever additional modules they need. No need
for a separate module cache FOD, go wrapper script, or gomobile source patches.

### Potential issue: reproducibility of .aar output

FODs require bit-identical output for the hash to match across builds. `.aar` files are
ZIP archives which may contain timestamps or other non-deterministic data, which would
cause the hash to differ between builds. In practice this did not block us — the hash
was obtained on first build and subsequent builds matched.

### Error: `javac` not found

**Symptom:** Build ran successfully through `go mod download`, `go build` of
gomobile/gobind, and into `gomobile bind`. gomobile generated bindings and started
compiling Java sources, then failed:

```
gomobile: javac -d /build/gomobile-work-207667746/javac-output -source 1.8 -target 1.8
  -bootclasspath .../android-35/android.jar ... failed: exec: "javac": executable file
  not found in $PATH
```

**Root cause:** The derivation's `nativeBuildInputs` had `go_1_25_6`, `androidSdk`, and
`pkgs.cacert`, but not a JDK. gomobile needs `javac` to compile the generated Java
binding classes.

**Fix:** Added `pkgs.jdk17` to `nativeBuildInputs`.

---

### Result: libtailscale.aar builds successfully

After adding JDK17, the build completed in under 2 minutes. The FOD hash was obtained
from the hash mismatch error:

```
sha256-PuzEtBzyTyXWRLGjp2FJEfS4W/6A9c3kjKqS5az3mSU=
```

Output: 62MB `libtailscale.aar` containing native `.so` files for all Android
architectures (arm, arm64, x86, x86_64) plus Java class files.

Final derivation:

```nix
libtailscale = pkgs.stdenv.mkDerivation {
  pname = "libtailscale";
  version = "0-unstable-2026-01-28";
  src = tailscaleAndroidSrc;

  nativeBuildInputs = [ go_1_25_6 androidSdk pkgs.jdk17 pkgs.cacert ];

  buildPhase = ''
    export HOME=$TMPDIR
    export GOPATH=$TMPDIR/go
    export GOMODCACHE=$TMPDIR/modcache
    export GOCACHE=$TMPDIR/go-cache
    export GOWORK=off
    export GOTOOLCHAIN=local
    export SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt
    export ANDROID_HOME="${androidSdk}/share/android-sdk"
    export ANDROID_NDK_HOME="$ANDROID_HOME/ndk/26.1.10909125"

    mkdir -p $GOPATH/bin

    go mod download
    go build -o $GOPATH/bin/gomobile golang.org/x/mobile/cmd/gomobile
    go build -o $GOPATH/bin/gobind golang.org/x/mobile/cmd/gobind
    export PATH="$GOPATH/bin:$PATH"

    gomobile bind \
      -target android \
      -androidapi 26 \
      -o libtailscale.aar \
      ./libtailscale
  '';

  installPhase = ''
    mkdir -p $out
    cp libtailscale.aar $out/
  '';

  outputHashAlgo = "sha256";
  outputHashMode = "recursive";
  outputHash = "sha256-PuzEtBzyTyXWRLGjp2FJEfS4W/6A9c3kjKqS5az3mSU=";
};
```

---

## Android APK build with libtailscale

### Error: `OutOfMemoryError: Java heap space` during APK packaging

**Symptom:** After libtailscale built successfully, building the full Android APK
(`.#android`) failed during the `packageDebug` Gradle task:

```
Caused by: java.lang.OutOfMemoryError: Java heap space
    at com.android.zipflinger.NoCopyByteArrayOutputStream.<init>(NoCopyByteArrayOutputStream.java:27)
    at com.android.zipflinger.Compressor.deflate(Compressor.java:35)
    at com.android.zipflinger.BytesSource.build(BytesSource.java:89)
```

The 62MB libtailscale.aar contains native `.so` files for 4 architectures. The APK
packager's zip compressor ran out of heap space while deflating these large binaries.

**Fix attempt 1:** Added `-Xmx2g` to `GRADLE_OPTS`. Same OOM error.

**Fix:** Changed to `-Xmx4g` in `GRADLE_OPTS`:

```nix
export GRADLE_OPTS="-Xmx4g -Djava.io.tmpdir=$TMPDIR -Dorg.gradle.native.dir=$TMPDIR/native"
```

With 4GB heap, the Gradle build completed successfully. All APK variants were produced:

| APK | Size |
|-----|------|
| `app-arm64-v8a-debug.apk` | 101 MB |
| `app-armeabi-v7a-debug.apk` | 99 MB |
| `app-universal-debug.apk` | 205 MB |
| `app-x86_64-debug.apk` | 105 MB |

### Note: Signing key mismatch on reinstall

When installing the new APK on an emulator that already had a previous build installed,
`adb install -r` failed with `INSTALL_FAILED_UPDATE_INCOMPATIBLE` because the Nix build
generates a fresh debug keystore each time. Required `adb uninstall com.alicia.assistant`
before installing.

---

## E2E Tests

### First run: 3/4 tests pass, VPN registration fails

Ran the full E2E test suite (`run-maestro-tests.sh`) which:
1. Builds APK via Nix (cached)
2. Installs on emulator
3. Starts Headscale for VPN tests
4. Runs onboarding, voice_interaction, conversations, vpn_registration tests

Results:
- onboarding: PASSED
- voice_interaction: PASSED
- conversations: PASSED
- vpn_registration: FAILED

---

### Error: `authKeyInput` element not found

**Symptom:** In `vpn_registration.yaml`, after entering the server URL, the test tried
to tap `authKeyInput` and failed:

```
Element not found: Id matching regex: com.alicia.assistant:id/authKeyInput
```

**Screenshot (vpn_04_server_url):** The soft keyboard was open and covering the bottom
half of the screen. The auth key input field is below the server URL input in a
ScrollView, but the keyboard was occluding it. Maestro could not find the element because
it was not visible/accessible.

**Fix:** Added `hideKeyboard` and `scrollUntilVisible` steps before tapping authKeyInput
and registerButton:

```yaml
- inputText: "${HEADSCALE_URL}"
- takeScreenshot: screenshots/vpn_04_server_url
- hideKeyboard

- scrollUntilVisible:
    element:
      id: "com.alicia.assistant:id/authKeyInput"
    direction: DOWN
    timeout: 5000
- tapOn:
    id: "com.alicia.assistant:id/authKeyInput"
```

Same pattern applied before tapping `registerButton`.

---

### Second run: registration itself fails

After the keyboard/scroll fix, the test progressed further. All input steps completed.
But the registration step failed.

**Screenshot (vpn_07_registered):** Shows "Registration failed" toast at the bottom of
the screen. The Connect button is visible but it's the **default** state ("Not connected
to any exit node"), not a post-registration state. The test's `extendedWaitUntil` for
`connectButton` succeeded because the button was already there by default, not because
registration completed.

The test then tapped Connect, which started "Setting up secure tunnel" but never reached
"Connected" (vpn_08_connecting screenshot shows this state).

**Investigation started but incomplete:** Attempted to check logcat for registration
error details. The logcat showed libtailscale initialization logs (VpnManager init,
backend started, health warnings about unstable version) but the actual registration
attempt logs were not found in the initial search. The registration failure likely
involves:
- The `setControlUrl` or `loginWithAuthKey` calls in `VpnSettingsActivity.kt`
- Communication between the app and Headscale at `http://10.0.2.2:8080`
- Possible issue with the Tailscale client accepting an HTTP (non-HTTPS) control URL

Investigation was interrupted at this point.

---

## Appendix: gomobile bind internal flow (Android)

For reference, the full internal flow of `gomobile bind -target android`:

1. `buildEnvInit()` — creates a temp work directory (`/tmp/gomobile-work-NNNN/`).
2. `gobind` — generates Go and Java binding source code into `$WORK/src/gobind/`.
3. For each architecture (arm, arm64, amd64, 386):
   1. Copies `$WORK/src/` to `$WORK/src-android-<arch>/`.
   2. `writeGoMod()` calls `getModuleVersions()` which runs `go list -m -json all`,
      then creates a synthetic go.mod (module `"gobind"`) with all enumerated modules
      as requires.
   3. `goModTidyAt()` runs `go mod tidy` in the temp directory to create go.sum.
   4. `goBuildAt()` runs `go build -buildmode=c-shared` to produce `libgojni.so`.
4. Packages everything into an `.aar` file (a ZIP containing JNI `.so` files and Java
   class files).

### Environment variable propagation in detail

gomobile constructs `androidEnv[arch]` containing:
- `GOOS=android`
- `GOARCH=<arch>`
- `CGO_ENABLED=1`
- `CC`, `CXX` (NDK clang paths)
- `CGO_CFLAGS`, `CGO_LDFLAGS`

`goModTidyAt` appends `GOMODCACHE` (from `go env GOMODCACHE`).
`buildAndroidSO` appends `GOPATH`.

All subprocess calls use `cmd.Env = env`, which **replaces** the entire process
environment. GOPROXY, GONOSUMCHECK, GONOSUMDB, PATH, HOME, and all other variables are
lost.

The go wrapper script workaround (Attempt 2) addresses this by embedding the needed
environment variables directly in the wrapper, so they are re-exported regardless of
what gomobile sets in `cmd.Env`.

### Module graph expansion in detail

Go 1.17+ uses lazy module loading: it only reads go.mod files for direct dependencies
(and recursively for any dependency whose own go.mod uses `go < 1.17`).

- `go mod download` downloads modules listed in the current module's go.mod and go.sum
  (~89 modules for tailscale-android).
- `go list -m all` returns the pruned module graph — modules actually needed for the
  build. The size of this set depends on which `.mod` files are available in
  GOMODCACHE.
- When more `.mod` files are present, Go can read them and discover additional
  transitive dependencies that were previously pruned, expanding the graph.
- The gomobile-generated synthetic go.mod lists everything from `go list -m all` as
  direct requires, which defeats lazy loading and forces Go to read go.mod for every
  listed module.
- This creates a feedback loop: downloading more modules causes the graph to expand,
  which requires downloading even more modules.
