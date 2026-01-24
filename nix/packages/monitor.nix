{ pkgs, src, version ? "0.1.0" }:

pkgs.buildGoApplication {
  pname = "monitor";
  inherit version src;
  modules = ./../../monitor/gomod2nix.toml;
  subPackages = [ "." ];

  meta = {
    description = "Alicia WebSocket Monitor - real-time protocol viewer";
    mainProgram = "monitor";
  };
}
