final: prev: let 
  mermaidMinJs = builtins.fetchurl {
    url = "https://raw.githubusercontent.com/badboy/mdbook-mermaid/27f72074f5fc8d3b58d75571e08ea33db1ee71e0/src/bin/assets/mermaid.min.js";
    sha256 = "sha256:00s71bf2v66j3l62j9lf5ac2z1s6bjazyx5qis1mwrfrpr9s5zpf";
  };
  
  mermaidInitJs = builtins.fetchurl {
    url = "https://raw.githubusercontent.com/badboy/mdbook-mermaid/27f72074f5fc8d3b58d75571e08ea33db1ee71e0/src/bin/assets/mermaid-init.js";
    sha256 = "sha256:02ihmgj4pijigvmjhp5xmvc7zqavl4d1bamnj7ylqcpnv999jp82";
  };
in {
  mdbookMermaidFiles = prev.stdenv.mkDerivation {
    name = "mdbook-mermaid-files";
    version = "0.15.0";
    dontUnpack = true;
    dontInstall = true;
    
    buildPhase = ''
      mkdir -p $out/share/mdbook-mermaid
      cp ${mermaidMinJs} $out/share/mdbook-mermaid/mermaid.min.js
      cp ${mermaidInitJs} $out/share/mdbook-mermaid/mermaid-init.js
    '';
  };
}
