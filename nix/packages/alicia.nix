{ pkgs
, alicia-backend
, alicia-frontend
, version ? "0.1.0"
}:

pkgs.stdenv.mkDerivation {
  pname = "alicia-full";
  inherit version;

  dontUnpack = true;

  installPhase = ''
    mkdir -p $out/bin
    mkdir -p $out/share/alicia/frontend
    mkdir -p $out/share/alicia/migrations

    # Copy backend binary
    cp ${alicia-backend}/bin/api $out/bin/alicia-server

    # Copy frontend static files
    cp -r ${alicia-frontend}/* $out/share/alicia/frontend/

    # Copy database migrations
    if [ -d "${alicia-backend}/share/alicia/migrations" ]; then
      if [ -n "$(ls -A ${alicia-backend}/share/alicia/migrations 2>/dev/null)" ]; then
        cp -r ${alicia-backend}/share/alicia/migrations/* $out/share/alicia/migrations/
        echo "Copied $(ls ${alicia-backend}/share/alicia/migrations | wc -l) migration files"
      fi
    fi
  '';

  meta = {
    description = "Alicia - A real-time voice assistant with web interface";
    homepage = "https://github.com/longregen/alicia";
    mainProgram = "alicia-server";
  };
}
