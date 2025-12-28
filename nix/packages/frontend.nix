{ pkgs
, src
, version ? "0.1.0"
}:

pkgs.buildNpmPackage {
  pname = "alicia-frontend";
  inherit version;
  src = "${src}/frontend";
  npmDepsHash = "sha256-vQfigFhmPtGHBdBq9iVv7ElZqFUCv/uNZ3EYG3M2RPU=";

  buildPhase = ''
    npm run build
  '';

  installPhase = ''
    mkdir -p $out
    cp -r dist/* $out/
    cp ${pkgs.sqlWasmFile}/share/sql-wasm/sql-wasm.wasm $out/
  '';

  meta = {
    description = "Alicia frontend - React/Vite web interface";
    homepage = "https://github.com/longregen/alicia";
  };
}
