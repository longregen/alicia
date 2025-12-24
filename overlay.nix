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

  # Build individual packages using shared package definitions
  alicia-backend = final.callPackage ./nix/packages/backend.nix {
    inherit src version;
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

  # Combined package
  alicia = final.callPackage ./nix/packages/alicia.nix {
    inherit version alicia-backend alicia-frontend;
  };

  # Also apply the mdbook-mermaid overlays
  mdbook-mermaid = (import ./nix/overlays/mdbook-mermaid final prev).mdbook-mermaid;
  mdbookMermaidFiles = (import ./nix/overlays/mdbook-mermaid/files.nix final prev).mdbookMermaidFiles;
}
