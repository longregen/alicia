{
  pkgs,
  src,
  version ? "0.1.0",
  vad-dependencies,
}:

pkgs.buildNpmPackage {
  pname = "alicia-web";
  inherit version src;
  npmDepsHash = "sha256-ZphWwy3nCTqoetAOvdg8i7hiYYOL5/kgshBfYADkUNk=";

  postPatch = ''
    rm -f public/models/* public/onnx/* public/js/lib/* public/sql-wasm.wasm 2>/dev/null || true
    mkdir -p public/models public/onnx public/js/lib
    cp ${vad-dependencies}/models/* public/models/
    cp ${vad-dependencies}/onnx/* public/onnx/
    cp ${vad-dependencies}/js/lib/* public/js/lib/
    cp ${pkgs.sqlWasmFile}/share/sql-wasm/sql-wasm.wasm public/sql-wasm.wasm
  '';

  buildPhase = ''
    runHook preBuild
    npm run build
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p $out
    cp -r dist/* $out/
    chmod -R u+w $out
    cp ${pkgs.sqlWasmFile}/share/sql-wasm/sql-wasm.wasm $out/
    runHook postInstall
  '';

  meta = {
    description = "Alicia web - React/Vite web interface";
    homepage = "https://github.com/longregen/alicia";
  };
}
