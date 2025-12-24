{ pkgs }:

pkgs.writeShellScriptBin "collect-logs" ''
  set -euo pipefail

  ARTIFACT_DIR="''${ARTIFACT_DIR:-/artifacts}"
  LOGS_DIR="$ARTIFACT_DIR/logs"

  mkdir -p "$LOGS_DIR"

  echo "Collecting server logs..."

  # Alicia service logs (JSON format from journald)
  echo "  - alicia service logs (JSON)..."
  ${pkgs.systemd}/bin/journalctl -u alicia \
    --no-pager \
    --output=json \
    > "$LOGS_DIR/backend.jsonl" 2>/dev/null || true

  # Alicia service logs (text format for stderr)
  echo "  - alicia service logs (text)..."
  ${pkgs.systemd}/bin/journalctl -u alicia \
    --no-pager \
    > "$LOGS_DIR/backend-stderr.log" 2>/dev/null || true

  # PostgreSQL logs
  echo "  - PostgreSQL logs..."
  if [ -f /var/log/postgresql/postgresql.log ]; then
    cp /var/log/postgresql/postgresql.log "$LOGS_DIR/"
  fi

  # Nginx access logs
  echo "  - nginx access logs..."
  if [ -f /var/log/nginx/access.log ]; then
    cp /var/log/nginx/access.log "$LOGS_DIR/nginx-access.log"
  fi

  # Nginx error logs
  echo "  - nginx error logs..."
  if [ -f /var/log/nginx/error.log ]; then
    cp /var/log/nginx/error.log "$LOGS_DIR/nginx-error.log"
  fi

  # System journal errors (last hour)
  echo "  - system errors..."
  ${pkgs.systemd}/bin/journalctl --no-pager --since="1 hour ago" \
    --priority=err \
    > "$LOGS_DIR/system-errors.log" 2>/dev/null || true

  # Compress large files (>10MB)
  echo "Compressing large log files..."
  for file in "$LOGS_DIR"/*.log "$LOGS_DIR"/*.jsonl; do
    if [ -f "$file" ] && [ $(stat -c%s "$file") -gt 10485760 ]; then
      echo "  - compressing $(basename "$file")..."
      ${pkgs.gzip}/bin/gzip "$file"
    fi
  done

  echo "Logs collected to $LOGS_DIR"
  ls -lh "$LOGS_DIR"
''
