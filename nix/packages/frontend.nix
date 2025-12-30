{ pkgs
, src
, version ? "0.1.0"
}:

pkgs.buildNpmPackage {
  pname = "alicia-frontend";
  inherit version;
  src = "${src}/frontend";
  npmDepsHash = "sha256-ECSazdjVj5GkIhdm+fGL+8+iG+i7IW04hQQuLTGjMFE=";

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
