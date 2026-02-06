#!/usr/bin/env python3
"""Rewind the emulator's WAV audio input to the beginning."""
import socket, sys, time

def main():
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 5556
    token_path = "/home/usr/.emulator_console_auth_token"

    with open(token_path) as f:
        token = f.read().strip()

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect(("localhost", port))
    s.settimeout(3)

    # Read banner
    s.recv(4096)

    # Auth
    s.send(f"auth {token}\r\n".encode())
    time.sleep(0.3)
    resp = s.recv(4096).decode()
    if "OK" not in resp:
        print(f"Auth failed: {resp}", file=sys.stderr)
        sys.exit(1)

    # Rewind audio
    s.send(b"avd rewindaudio\r\n")
    time.sleep(0.3)
    resp = s.recv(4096).decode()
    if "OK" in resp:
        print("Audio rewound to beginning")
    else:
        print(f"Rewind failed: {resp}", file=sys.stderr)
        sys.exit(1)

    s.send(b"quit\r\n")
    s.close()

if __name__ == "__main__":
    main()
