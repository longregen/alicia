{ config, pkgs, lib, ... }:

let
  testUser = "test";

  # Get the e2e-test directory from the parent of the nix directory
  e2eTestDir = ../.;

  # Pre-built e2e-test package with all npm dependencies
  e2eTestPackage = pkgs.buildNpmPackage {
    pname = "alicia-e2e-tests";
    version = "1.0.0";
    src = e2eTestDir;
    npmDepsHash = "sha256-AcAe8VmnbGZSh9oEUQcwrlWsNraJ1tY0s1i2rfLhK/s=";
    dontNpmBuild = true;
    installPhase = ''
      runHook preInstall
      mkdir -p $out
      cp -r . $out/
      runHook postInstall
    '';
  };
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
    hostName = "client";
    firewall.enable = false;  # Within test network
  };

  # ============================================================================
  # XFCE Desktop Environment
  # ============================================================================

  services.xserver = {
    enable = true;

    # XFCE desktop
    desktopManager.xfce = {
      enable = true;
      enableXfwm = true;
      noDesktop = false;
    };

    # Virtual display configuration
    resolutions = [
      { x = 1920; y = 1080; }
    ];

    # Disable screen blanking for tests
    serverFlagsSection = ''
      Option "BlankTime" "0"
      Option "StandbyTime" "0"
      Option "SuspendTime" "0"
      Option "OffTime" "0"
    '';

    # LightDM display manager
    displayManager.lightdm.enable = true;
  };

  # Display manager autologin (NixOS 24.11+ location)
  services.displayManager = {
    autoLogin = {
      enable = true;
      user = testUser;
    };
    defaultSession = "xfce";
  };

  # ============================================================================
  # Test User Configuration
  # ============================================================================

  users.users.${testUser} = {
    isNormalUser = true;
    home = "/home/${testUser}";
    createHome = true;
    extraGroups = [ "audio" "video" ];
    # Empty password for test environment
    initialPassword = "";
  };

  # ============================================================================
  # Copy e2e-test directory to VM
  # ============================================================================

  # Copy e2e-test files to the test user's home directory
  system.activationScripts.copyE2eTests = ''
    # Create e2e-test directory in test user's home
    mkdir -p /home/${testUser}/e2e-test

    # Copy pre-built e2e-test package with npm dependencies from Nix store
    cp -r ${e2eTestPackage}/* /home/${testUser}/e2e-test/

    # Set ownership (make files writable since they come from Nix store)
    chown -R ${testUser}:users /home/${testUser}/e2e-test
    chmod -R u+w /home/${testUser}/e2e-test

    # Create artifacts directory with proper permissions
    mkdir -p /artifacts
    chown -R ${testUser}:users /artifacts
  '';

  # ============================================================================
  # System Packages
  # ============================================================================

  environment.systemPackages = with pkgs; [
    # Browser - Playwright's bundled chromium
    playwright-driver.browsers

    # Node.js for Playwright
    nodejs_22

    # Playwright system dependencies
    # These are the libraries Playwright needs for Chromium
    nss
    nspr
    at-spi2-atk
    cups
    libdrm
    libxkbcommon
    mesa
    alsa-lib
    pango
    cairo
    expat
    glib
    gtk3

    # Utilities
    xdotool
    scrot
    xclip

    # Debugging tools
    htop
    curl
    jq
  ];

  # ============================================================================
  # Audio Configuration
  # ============================================================================

  # Use PipeWire with PulseAudio compatibility (NixOS 24.11+ default)
  hardware.pulseaudio.enable = false;
  services.pipewire = {
    enable = true;
    pulse.enable = true;  # PulseAudio server emulation
    alsa.enable = true;
  };

  # ============================================================================
  # Fonts
  # ============================================================================

  fonts.packages = with pkgs; [
    noto-fonts
    noto-fonts-color-emoji
    liberation_ttf
    dejavu_fonts
  ];

  # ============================================================================
  # Environment Variables
  # ============================================================================

  environment.variables = {
    DISPLAY = ":0";
    # Playwright settings - use Nix-provided browsers
    PLAYWRIGHT_BROWSERS_PATH = "${pkgs.playwright-driver.browsers}";
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD = "1";
  };

  # ============================================================================
  # Virtualisation Settings
  # ============================================================================

  virtualisation = {
    resolution = {
      x = 1920;
      y = 1080;
    };
    memorySize = 4096;
    cores = 4;
    graphics = true;
    qemu.options = [
      "-vga virtio"
    ];
  };
}
