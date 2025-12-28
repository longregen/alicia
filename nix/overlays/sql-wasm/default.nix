final: prev:
let
  sqlWasm = builtins.fetchurl {
    url = "https://sql.js.org/dist/sql-wasm.wasm";
    sha256 = "0wrdyqwn8xa5wpjcfjcl5sikavmb9ydb1x9g3wyri4z4hdf1ad07";
  };
in {
  sqlWasmFile = prev.stdenv.mkDerivation {
    name = "sql-wasm";
    version = "1.13.0";
    dontUnpack = true;
    dontInstall = true;

    buildPhase = ''
      mkdir -p $out/share/sql-wasm
      cp ${sqlWasm} $out/share/sql-wasm/sql-wasm.wasm
    '';
  };
}
