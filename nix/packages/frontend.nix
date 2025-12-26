{ pkgs
, src
, version ? "0.1.0"
}:

pkgs.buildNpmPackage {
  pname = "alicia-frontend";
  inherit version;
  src = "${src}/frontend";
  npmDepsHash = "sha256-fVhS0tAnWk/kW21nc9ItKh2OEfaPeLzBffdMTAkdi1E=";

  buildPhase = ''
    npm run build
  '';

  installPhase = ''
    mkdir -p $out
    cp -r dist/* $out/
  '';

  meta = {
    description = "Alicia frontend - React/Vite web interface";
    homepage = "https://github.com/longregen/alicia";
  };
}
