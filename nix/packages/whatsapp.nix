{ pkgs, src, preBuild, version ? "0.1.0" }:

pkgs.buildGoApplication {
  pname = "whatsapp";
  inherit version src preBuild;
  modules = ./../../whatsapp/gomod2nix.toml;
  subPackages = [ "." ];

  meta = {
    description = "Alicia WhatsApp - WhatsApp bridge for AI assistant";
    mainProgram = "whatsapp";
  };
}
