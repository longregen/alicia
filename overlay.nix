# Alicia overlay for non-flake users
#
# This overlay provides the alicia, alicia-backend, and alicia-frontend packages.
#
# Prerequisites:
# - gomod2nix overlay must be applied before this overlay
# - See https://github.com/nix-community/gomod2nix for setup instructions
#
# Usage in configuration.nix:
#
#   let
#     aliciaSrc = builtins.fetchGit {
#       url = "https://github.com/longregen/alicia.git";
#       ref = "main";
#     };
#     gomod2nixSrc = builtins.fetchGit {
#       url = "https://github.com/nix-community/gomod2nix.git";
#       ref = "master";
#     };
#   in {
#     nixpkgs.overlays = [
#       (import "${gomod2nixSrc}/overlay.nix")
#       (import "${aliciaSrc}/overlay.nix")
#     ];
#   }
#
final: prev:
let
  # Version information
  version = "0.1.0";

  # Source is the current directory (where overlay.nix is located)
  src = ./.;

  # Silero VAD v5 model for Go backend
  # Uses the same model as the frontend (from @ricky0123/vad-web) for consistency
  silero-vad-model = final.stdenv.mkDerivation {
    pname = "silero-vad-model";
    version = "5.0";

    src = final.fetchurl {
      url = "https://cdn.jsdelivr.net/npm/@ricky0123/vad-web@0.0.22/dist/silero_vad_v5.onnx";
      sha256 = "sha256-JiOilT9v89LB5hdAxs23FoEzR5smff7xFKSjzFvdeI8=";
    };

    dontUnpack = true;

    installPhase = ''
      mkdir -p $out/share/silero-vad
      cp $src $out/share/silero-vad/silero_vad.onnx
    '';

    meta = {
      description = "Silero VAD v5 ONNX model for voice activity detection";
      homepage = "https://github.com/snakers4/silero-vad";
      license = final.lib.licenses.mit;
    };
  };

  # Build individual packages using shared package definitions
  alicia-backend = final.callPackage ./nix/packages/backend.nix {
    inherit src version silero-vad-model;
    rev = "overlay";
  };

  alicia-frontend = final.callPackage ./nix/packages/frontend.nix {
    inherit src version;
  };

in
{
  # Export the packages
  alicia-backend = alicia-backend;
  alicia-frontend = alicia-frontend;

  # Silero VAD model
  inherit silero-vad-model;

  # Combined package
  alicia = final.callPackage ./nix/packages/alicia.nix {
    inherit version alicia-backend alicia-frontend;
  };

  # Also apply the mdbook-mermaid overlays
  mdbook-mermaid = (import ./nix/overlays/mdbook-mermaid final prev).mdbook-mermaid;
  mdbookMermaidFiles = (import ./nix/overlays/mdbook-mermaid/files.nix final prev).mdbookMermaidFiles;
}
