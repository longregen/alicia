#!/usr/bin/env python3
"""Minimal mock of the Alicia API for E2E testing.

Handles POST /api/v1/vpn/auth-key by returning a pre-configured
Headscale pre-auth key and server URL. Mimics the real Go backend endpoint.
"""

import http.server
import json
import os
import sys

# The pre-auth key to hand out (created by start-headscale.sh)
HEADSCALE_PREAUTH_KEY = os.environ.get("HEADSCALE_PREAUTH_KEY", "")
# The URL the Android emulator uses to reach Headscale
HEADSCALE_URL = os.environ.get("HEADSCALE_URL", "http://10.0.2.2:8080")
PORT = int(os.environ.get("MOCK_API_PORT", "8181"))


class MockApiHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path == "/api/v1/vpn/auth-key":
            self.send_json(200, {
                "server_url": HEADSCALE_URL,
                "auth_key": HEADSCALE_PREAUTH_KEY,
            })
        else:
            self.send_error(404, "Not found")

    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"status":"ok"}')
        else:
            self.send_error(404, "Not found")

    def send_json(self, code, data):
        body = json.dumps(data).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        sys.stderr.write(f"[mock-api] {fmt % args}\n")


if __name__ == "__main__":
    if not HEADSCALE_PREAUTH_KEY:
        print("HEADSCALE_PREAUTH_KEY not set", file=sys.stderr)
        sys.exit(1)

    server = http.server.HTTPServer(("0.0.0.0", PORT), MockApiHandler)
    print(f"Mock Alicia API listening on :{PORT}", file=sys.stderr)
    print(f"  Headscale URL: {HEADSCALE_URL}", file=sys.stderr)
    print(f"  Pre-auth key:  {HEADSCALE_PREAUTH_KEY[:12]}...", file=sys.stderr)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    server.server_close()
