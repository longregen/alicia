{ pkgs, lib, self }:

let
  artifactDir = "/tmp/alicia-e2e-artifacts";

  logCollector = import ./log-collector.nix { inherit pkgs; };
in
pkgs.testers.nixosTest {
  name = "alicia-e2e";

  # Skip type checking and linting
  skipTypeCheck = true;
  skipLint = true;

  nodes = {
    # server.nix is curried: first call with extra args, returns standard module
    server = import ./server.nix { inherit self logCollector; };
    client = import ./client.nix;
  };

  testScript = ''
    import json
    import os
    from datetime import datetime
    from typing import TYPE_CHECKING

    if TYPE_CHECKING:
        from test_driver import Machine
        client: Machine
        server: Machine

    # Artifact directory setup
    os.makedirs("${artifactDir}", exist_ok=True)
    os.makedirs("${artifactDir}/screenshots/browser", exist_ok=True)
    os.makedirs("${artifactDir}/screenshots/desktop", exist_ok=True)
    os.makedirs("${artifactDir}/logs", exist_ok=True)
    os.makedirs("${artifactDir}/traces", exist_ok=True)
    os.makedirs("${artifactDir}/dom", exist_ok=True)

    def vnc_screenshot(name: str):
        """Capture desktop screenshot from client VM"""
        timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
        filename = f"{name}-{timestamp}"
        client.succeed("mkdir -p /artifacts/screenshots/desktop && chown -R test:users /artifacts/screenshots")
        client.succeed(f"su - test -c 'DISPLAY=:0 scrot /artifacts/screenshots/desktop/{filename}.png'")
        print(f"Screenshot saved: {filename}.png")
        return f"/artifacts/screenshots/desktop/{filename}.png"

    def collect_server_logs():
        """Collect all server-side logs using log-collector script"""
        server.succeed("collect-logs")

    def run_playwright_tests():
        """Execute Playwright tests on client VM"""
        # Ensure log directory exists before tee
        client.succeed("mkdir -p /artifacts/logs")
        client.succeed(
            "cd /home/test/e2e-test && "
            "NIXOS_TEST=1 "
            "BASE_URL=http://server "
            "ARTIFACT_DIR=/artifacts "
            "npx playwright test tests/smoke.spec.ts "
            "--reporter=html,json "
            "--timeout=60000 "
            "2>&1 | tee /artifacts/logs/playwright-output.log"
        )

    # Start all VMs
    start_all()

    # Wait for server services to be ready
    with subtest("Server services startup"):
        print("Waiting for PostgreSQL...")
        server.wait_for_unit("postgresql.service")

        print("Waiting for Alicia service...")
        server.wait_for_unit("alicia.service")

        print("Waiting for nginx...")
        server.wait_for_unit("nginx.service")

        print("Waiting for port 80...")
        server.wait_for_open_port(80)

    # Verify backend health
    with subtest("Backend health check"):
        print("Checking backend health endpoint...")
        server.succeed("curl -sf http://localhost/health | grep -q ok")
        print("Backend is healthy")

    # Wait for client desktop environment
    with subtest("Client desktop startup"):
        print("Waiting for X server...")
        client.wait_for_x()

        print("Waiting for display manager...")
        client.wait_for_unit("display-manager.service")

        print("Desktop ready")

    # Initial state screenshot
    vnc_screenshot("vnc-before-test")

    # Verify client can reach server
    with subtest("Network connectivity"):
        print("Testing client-to-server connectivity...")
        client.succeed("curl -sf http://server/health | grep -q ok")
        print("Client can reach server")

    # Verify Playwright is ready (npm deps are pre-installed via Nix)
    with subtest("Verify Playwright"):
        print("Verifying Playwright installation...")
        client.succeed("cd /home/test/e2e-test && npx playwright --version")
        print("Playwright is ready")

    vnc_screenshot("vnc-playwright-ready")

    # Run Playwright e2e tests
    test_passed = False
    with subtest("Run Playwright e2e tests"):
        try:
            print("Starting Playwright tests...")
            run_playwright_tests()
            test_passed = True
            print("Playwright tests completed successfully")
        except Exception as e:
            print(f"Playwright tests failed: {e}")
            vnc_screenshot("vnc-on-failure")
            collect_server_logs()
            # Copy artifacts even on failure
            try:
                client.copy_from_vm("/artifacts/", "${artifactDir}/")
                server.copy_from_vm("/artifacts/logs/", "${artifactDir}/logs/")
            except:
                print("Warning: Could not copy all artifacts on failure")
            raise

    # Capture final state
    vnc_screenshot("vnc-final-state")

    # Collect logs regardless of test outcome
    collect_server_logs()

    # Copy artifacts from VMs to host
    with subtest("Collect artifacts"):
        print("Copying artifacts from client VM...")
        client.copy_from_vm("/artifacts/", "${artifactDir}/")

        print("Copying logs from server VM...")
        server.copy_from_vm("/artifacts/logs/", "${artifactDir}/logs/")

        print("Artifacts collected to ${artifactDir}")

    # Generate summary.json
    summary = {
        "timestamp": datetime.now().isoformat(),
        "status": "passed" if test_passed else "failed",
        "server": {
            "services": {
                "postgresql": "running",
                "alicia": "running",
                "nginx": "running"
            }
        },
        "client": {
            "desktop": "xfce",
            "playwright": "installed"
        },
        "artifacts": {
            "logs": os.listdir(f"${artifactDir}/logs") if os.path.exists(f"${artifactDir}/logs") else [],
            "screenshots_browser": os.listdir(f"${artifactDir}/screenshots/browser") if os.path.exists(f"${artifactDir}/screenshots/browser") else [],
            "screenshots_desktop": os.listdir(f"${artifactDir}/screenshots/desktop") if os.path.exists(f"${artifactDir}/screenshots/desktop") else [],
        }
    }

    with open("${artifactDir}/summary.json", "w") as f:
        json.dump(summary, f, indent=2)

    print("Test execution complete. Artifacts saved to ${artifactDir}")
  '';
}
