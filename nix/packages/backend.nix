{ pkgs, src, preBuild, migrations, version ? "0.1.0", rev ? "dirty" }:

pkgs.buildGoApplication {
  pname = "alicia-api";
  inherit version src preBuild;
  modules = ./../../api/gomod2nix.toml;
  subPackages = [ "." ];

  nativeBuildInputs = with pkgs; [ pkg-config ];
  buildInputs = with pkgs; [ libopus opusfile libogg ];

  ldflags = [
    "-s" "-w"
    "-X main.version=${version}"
    "-X main.commit=${rev}"
  ];

  postInstall = ''
    mkdir -p $out/share/alicia/migrations
    cp -r ${migrations}/*.sql $out/share/alicia/migrations/
  '';

  meta = {
    description = "Alicia API - HTTP/WebSocket server";
    mainProgram = "api";
  };
}
