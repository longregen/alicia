{ pkgs
, src
, version ? "0.1.0"
, rev ? "dirty"
, silero-vad-model ? null
}:

pkgs.buildGoApplication {
  pname = "alicia";
  inherit version;
  inherit src;
  modules = "${src}/gomod2nix.toml";
  subPackages = [ "cmd/alicia" ];

  # CGO dependencies for opus audio codec and ONNX runtime for VAD
  nativeBuildInputs = with pkgs; [ pkg-config ];
  buildInputs = with pkgs; [
    libopus
    opusfile
    libogg
    onnxruntime  # Required for silero-vad-go voice activity detection
  ];

  # Add build-time version information
  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
    "-X main.commit=${rev}"
    "-X main.date=1970-01-01T00:00:00Z"
  ];

  # Copy database migrations and VAD model to the output
  postInstall = ''
    mkdir -p $out/share/alicia/migrations
    cp -r ${src}/migrations/*.sql $out/share/alicia/migrations/

    # Copy Silero VAD model if provided
    ${pkgs.lib.optionalString (silero-vad-model != null) ''
      mkdir -p $out/share/alicia/models
      cp ${silero-vad-model}/share/silero-vad/silero_vad.onnx $out/share/alicia/models/
    ''}
  '';

  meta = {
    description = "Alicia - A real-time voice assistant for conversations through audio";
    homepage = "https://github.com/longregen/alicia";
    mainProgram = "alicia";
  };
}
