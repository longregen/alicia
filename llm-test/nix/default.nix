# LLM Test: Minimal VM test for speaches (STT/TTS) + ollama (LLM)
#
# Tests the full pipeline:
# 1. Generate audio with espeak-ng
# 2. Transcribe with whisper (speaches)
# 3. Send to LLM (ollama/tinyllama)
# 4. Convert response to speech (speaches/kokoro)

{ pkgs, lib, speaches, nix-hug }:

pkgs.testers.nixosTest {
  name = "llm-test";

  skipTypeCheck = true;
  skipLint = true;

  nodes = {
    server = import ./server.nix { inherit speaches nix-hug; };
  };

  testScript = ''
    import json
    import os
    from datetime import datetime

    # Artifact directory
    artifact_dir = "/tmp/llm-test-artifacts"
    os.makedirs(artifact_dir, exist_ok=True)

    # Start the server
    start_all()

    # Wait for speaches to be ready
    with subtest("Speaches service startup"):
        print("Waiting for speaches service...")
        server.wait_for_unit("speaches.service")
        server.wait_for_open_port(8000)

        # Health check
        print("Checking speaches health...")
        server.succeed("curl -sf http://localhost:8000/health")
        print("Speaches is healthy")

    # Wait for ollama to be ready
    with subtest("Ollama service startup"):
        print("Waiting for ollama service...")
        server.wait_for_unit("ollama.service")
        server.wait_for_open_port(11434)

        # Wait for model to load (can take a while on first run)
        print("Waiting for tinyllama model to load...")
        for i in range(60):
            try:
                result = server.succeed("curl -sf http://localhost:11434/api/tags | jq -e '.models[] | select(.name | startswith(\"tinyllama\"))'")
                if result.strip():
                    print("tinyllama model is loaded")
                    break
            except:
                pass
            import time
            time.sleep(2)
        else:
            raise Exception("tinyllama model failed to load within timeout")

    # Test 1: Generate test audio
    with subtest("Generate test audio"):
        print("Generating test audio with espeak-ng...")
        server.succeed("/etc/test-scripts/generate-audio.sh 'Hello world' /artifacts/audio/test.wav")
        server.succeed("test -f /artifacts/audio/test.wav")
        print("Test audio generated successfully")

    # Test 2: Transcribe audio with whisper
    with subtest("Transcribe audio with whisper"):
        print("Transcribing audio...")
        transcription = server.succeed("/etc/test-scripts/transcribe.sh /artifacts/audio/test.wav").strip()
        print(f"Transcription: {transcription}")

        # Verify we got something back
        if not transcription or len(transcription) < 3:
            raise Exception(f"Transcription failed or empty: '{transcription}'")

        # Should roughly match "Hello world" (case insensitive, fuzzy)
        if "hello" not in transcription.lower():
            print(f"Warning: Transcription doesn't contain 'hello': {transcription}")

    # Test 3: Send prompt to LLM
    with subtest("LLM inference"):
        print("Sending prompt to LLM...")
        response = server.succeed("/etc/test-scripts/chat.sh 'Say hello in exactly 3 words'").strip()
        print(f"LLM Response: {response}")

        # Verify we got a response
        if not response or len(response) < 2:
            raise Exception(f"LLM response failed or empty: '{response}'")

    # Test 4: Convert text to speech
    with subtest("Text to speech"):
        print("Converting text to speech...")
        server.succeed(
            "curl -sf -X POST 'http://localhost:8000/v1/audio/speech' "
            "-H 'Content-Type: application/json' "
            "-d '{\"input\": \"This is a test response.\", \"model\": \"kokoro\", \"voice\": \"af_heart\"}' "
            "--output /artifacts/audio/response.wav"
        )
        server.succeed("test -f /artifacts/audio/response.wav")
        server.succeed("test -s /artifacts/audio/response.wav")  # Check non-empty
        print("TTS audio generated successfully")

    # Test 5: Full pipeline
    with subtest("Full pipeline"):
        print("Running full pipeline test...")
        server.succeed("/etc/test-scripts/full-pipeline.sh")

    # Collect artifacts
    with subtest("Collect artifacts"):
        print("Copying artifacts...")
        server.copy_from_vm("/artifacts/", artifact_dir)
        print(f"Artifacts saved to {artifact_dir}")

    # Summary
    print("")
    print("=" * 60)
    print("LLM Test Summary: PASSED")
    print("=" * 60)
    print("- Speaches (whisper STT): OK")
    print("- Speaches (kokoro TTS): OK")
    print("- Ollama (tinyllama): OK")
    print("- Full pipeline: OK")
  '';
}
