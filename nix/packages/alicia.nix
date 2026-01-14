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
    mkdir -p $out/share/alicia/models

    # Copy backend binary
    cp ${alicia-backend}/bin/alicia $out/bin/

    # Copy frontend static files
    cp -r ${alicia-frontend}/* $out/share/alicia/frontend/

    # Copy database migrations
    # Note: This will fail if no migrations exist, which is intentional
    # as migrations are required for the database auto-migration feature
    if [ -d "${alicia-backend}/share/alicia/migrations" ]; then
      if [ -n "$(ls -A ${alicia-backend}/share/alicia/migrations 2>/dev/null)" ]; then
        cp -r ${alicia-backend}/share/alicia/migrations/* $out/share/alicia/migrations/
        echo "Copied $(ls ${alicia-backend}/share/alicia/migrations | wc -l) migration files"
      else
        echo "WARNING: Migrations directory exists but is empty" >&2
      fi
    else
      echo "ERROR: Migrations directory not found in backend package" >&2
      echo "Expected: ${alicia-backend}/share/alicia/migrations" >&2
      exit 1
    fi

    # Copy Silero VAD model for voice activity detection
    if [ -d "${alicia-backend}/share/alicia/models" ]; then
      if [ -n "$(ls -A ${alicia-backend}/share/alicia/models 2>/dev/null)" ]; then
        cp -r ${alicia-backend}/share/alicia/models/* $out/share/alicia/models/
        echo "Copied VAD model files"
      fi
    fi
  '';

  meta = {
    description = "Alicia - A real-time voice assistant with web interface";
    homepage = "https://github.com/longregen/alicia";
    mainProgram = "alicia";
  };
}
