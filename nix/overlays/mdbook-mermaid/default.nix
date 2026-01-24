final: prev: {
  mdbook-mermaid = prev.mdbook-mermaid.overrideAttrs (finalAttrs: oldAttrs: let
    version = "0.15.0";
  in {
    inherit version;
    src = prev.fetchFromGitHub {
      owner = "badboy";
      repo = "mdbook-mermaid";
      tag = "v${version}";
      hash = "sha256-+Dk3wW1pLWVfJy+OC648BQ5rZrHYqPdjV2hfJSIV6m0=";
    };
    cargoDeps = prev.rustPlatform.importCargoLock {
      lockFile = ./Cargo.lock;
    };
    postPatch = ''
      substituteInPlace Cargo.toml --replace-fail '0.4.36' '0.4.49'
      cp ${./Cargo.lock} Cargo.lock
    '';
  });
}
