{ pkgs, src, preBuild, version ? "0.1.0" }:

pkgs.buildGoApplication {
  pname = "agent";
  inherit version src preBuild;
  modules = ./../../agent/gomod2nix.toml;
  subPackages = [ "." ];

  meta = {
    description = "Alicia Agent - AI assistant backend";
    mainProgram = "agent";
  };
}
