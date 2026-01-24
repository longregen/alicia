{ pkgs, src, preBuild, version ? "0.1.0" }:

pkgs.buildGoApplication {
  pname = "voice";
  inherit version src preBuild;
  modules = ./../../voice/gomod2nix.toml;
  subPackages = [ "." ];

  nativeBuildInputs = with pkgs; [ pkg-config ];
  buildInputs = with pkgs; [ libopus opusfile libogg ];

  meta = {
    description = "Voice helper - ASR/TTS bridge for LiveKit";
    mainProgram = "voice";
  };
}
