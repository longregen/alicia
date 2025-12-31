# LLM Test Server: speaches (STT/TTS) + ollama (LLM inference)
# Curried function: first takes extra args, then returns a standard NixOS module
{ speaches, nix-hug }:
{ config, pkgs, lib, ... }:

let
  # Use CPU-only speaches for VM testing
  speachesPkg = speaches.packages.${pkgs.system}.speaches-cpu;

  # Pre-built models that don't have hash issues
  kokoroModel = speaches.packages.${pkgs.system}.kokoro-82m;
  sileroModel = speaches.packages.${pkgs.system}.silero-vad;
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
    firewall.enable = false;

    # Enable external network access for model downloads
    nameservers = [ "8.8.8.8" "1.1.1.1" ];
  };

  # Enable NAT for external network access
  virtualisation.forwardPorts = [];  # Placeholder for VM network

  # ============================================================================
  # Speaches Service (STT/TTS)
  # ============================================================================

  # Import speaches NixOS module (x86_64-linux for VM tests)
  imports = [ speaches.nixosModules.x86_64-linux.default ];

  services.speaches = {
    enable = true;
    package = speachesPkg;
    port = 8000;
    host = "0.0.0.0";
    enableCuda = false;  # CPU only for VM test
    enablePiper = false; # Use kokoro only
    enableOtel = false;  # No telemetry needed

    # Models will be downloaded at runtime
    # Pre-built kokoro and silero are available but whisper needs download
  };

  # ============================================================================
  # Ollama Service (LLM inference)
  # Using ollama instead of vLLM for simpler CPU-only setup
  # ============================================================================

  services.ollama = {
    enable = true;
    host = "0.0.0.0";
    port = 11434;

    # Load a small model on startup
    loadModels = [ "tinyllama" ];
  };

  # ============================================================================
  # Test Audio Generation (espeak for TTS to create test audio)
  # ============================================================================

  environment.systemPackages = with pkgs; [
    curl
    jq
    espeak-ng      # TTS for generating test audio
    ffmpeg         # Audio conversion
    sox            # Audio manipulation
  ];

  # ============================================================================
  # Artifact Directory
  # ============================================================================

  system.activationScripts.createArtifactDir = ''
    mkdir -p /artifacts/logs /artifacts/audio
    chmod 777 /artifacts /artifacts/logs /artifacts/audio
  '';

  # ============================================================================
  # Test Scripts
  # ============================================================================

  environment.etc."test-scripts/generate-audio.sh" = {
    mode = "0755";
    text = ''
      #!/usr/bin/env bash
      # Generate test audio using espeak-ng
      TEXT="''${1:-Hello, this is a test of the speech recognition system.}"
      OUTPUT="''${2:-/artifacts/audio/test.wav}"

      espeak-ng -w "$OUTPUT" "$TEXT"
      echo "Generated audio: $OUTPUT"
    '';
  };

  environment.etc."test-scripts/transcribe.sh" = {
    mode = "0755";
    text = ''
      #!/usr/bin/env bash
      # Send audio to speaches for transcription
      AUDIO_FILE="''${1:-/artifacts/audio/test.wav}"

      curl -s -X POST "http://localhost:8000/v1/audio/transcriptions" \
        -H "Content-Type: multipart/form-data" \
        -F "file=@$AUDIO_FILE" \
        -F "model=whisper-base" \
        | jq -r '.text'
    '';
  };

  environment.etc."test-scripts/chat.sh" = {
    mode = "0755";
    text = ''
      #!/usr/bin/env bash
      # Send prompt to ollama for response
      PROMPT="''${1:-Hello, how are you?}"

      curl -s "http://localhost:11434/api/generate" \
        -d "{\"model\": \"tinyllama\", \"prompt\": \"$PROMPT\", \"stream\": false}" \
        | jq -r '.response'
    '';
  };

  environment.etc."test-scripts/full-pipeline.sh" = {
    mode = "0755";
    text = ''
      #!/usr/bin/env bash
      set -euo pipefail

      echo "=== LLM Test Pipeline ==="

      # Step 1: Generate audio from text
      TEST_TEXT="What is the capital of France?"
      echo "1. Generating audio from text: $TEST_TEXT"
      /etc/test-scripts/generate-audio.sh "$TEST_TEXT" /artifacts/audio/test.wav

      # Step 2: Transcribe audio back to text
      echo "2. Transcribing audio..."
      TRANSCRIPTION=$(/etc/test-scripts/transcribe.sh /artifacts/audio/test.wav)
      echo "   Transcription: $TRANSCRIPTION"

      # Step 3: Send to LLM
      echo "3. Sending to LLM..."
      RESPONSE=$(/etc/test-scripts/chat.sh "$TRANSCRIPTION")
      echo "   LLM Response: $RESPONSE"

      # Step 4: Convert response to speech (using speaches TTS)
      echo "4. Converting response to speech..."
      curl -s -X POST "http://localhost:8000/v1/audio/speech" \
        -H "Content-Type: application/json" \
        -d "{\"input\": \"$RESPONSE\", \"model\": \"kokoro\", \"voice\": \"af_heart\"}" \
        --output /artifacts/audio/response.wav

      echo "=== Pipeline Complete ==="
      echo "Test audio: /artifacts/audio/test.wav"
      echo "Response audio: /artifacts/audio/response.wav"
    '';
  };
}
