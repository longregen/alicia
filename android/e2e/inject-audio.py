#!/usr/bin/env python3
"""Inject a WAV file into the Android emulator's virtual microphone via gRPC."""

import argparse
import struct
import time
import wave
import grpc
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "proto"))
import emulator_controller_pb2 as pb
import emulator_controller_pb2_grpc as pb_grpc


def read_raw_pcm(path):
    """Read WAV and return raw PCM bytes + metadata."""
    with wave.open(path, "rb") as w:
        channels = w.getnchannels()
        sample_width = w.getsampwidth()
        rate = w.getframerate()
        # Read all frames (nframes may be INT_MAX for streaming WAVs)
        data = w.readframes(rate * 60)  # up to 60s
    return data, channels, sample_width, rate


def audio_packets(wav_path, chunk_ms=30):
    """Yield AudioPacket messages from a WAV file."""
    data, channels, sample_width, rate = read_raw_pcm(wav_path)

    if sample_width != 2:
        raise ValueError(f"Expected 16-bit audio, got {sample_width * 8}-bit")

    fmt = pb.AudioFormat(
        samplingRate=rate,
        channels=pb.AudioFormat.Mono if channels == 1 else pb.AudioFormat.Stereo,
        format=pb.AudioFormat.AUD_FMT_S16,
    )

    chunk_bytes = int(rate * channels * sample_width * chunk_ms / 1000)
    offset = 0
    ts = int(time.time() * 1_000_000)

    while offset < len(data):
        chunk = data[offset : offset + chunk_bytes]
        pkt = pb.AudioPacket(
            format=fmt,
            timestamp=ts,
            audio=chunk,
        )
        yield pkt
        offset += chunk_bytes
        ts += chunk_ms * 1000
        time.sleep(chunk_ms / 1000)

    # Send trailing silence to flush the buffer (~300ms)
    silence = b"\x00" * chunk_bytes
    for _ in range(10):
        yield pb.AudioPacket(format=fmt, timestamp=ts, audio=silence)
        ts += chunk_ms * 1000
        time.sleep(chunk_ms / 1000)


def get_grpc_token():
    """Read the gRPC token from the emulator's advertising file."""
    import glob
    for ini in glob.glob("/run/user/*/avd/running/pid_*.ini"):
        with open(ini) as f:
            for line in f:
                if line.startswith("grpc.token="):
                    return line.strip().split("=", 1)[1]
    return None


def main():
    parser = argparse.ArgumentParser(description="Inject audio into emulator mic")
    parser.add_argument("wav", help="Path to WAV file")
    parser.add_argument("--host", default="localhost", help="Emulator gRPC host")
    parser.add_argument("--port", type=int, default=8556, help="Emulator gRPC port")
    args = parser.parse_args()

    addr = f"{args.host}:{args.port}"
    print(f"Connecting to emulator gRPC at {addr}...")
    channel = grpc.insecure_channel(addr)
    stub = pb_grpc.EmulatorControllerStub(channel)

    print(f"Injecting audio from {args.wav}...")
    try:
        stub.injectAudio(audio_packets(args.wav))
        print("Audio injection complete.")
    except grpc.RpcError as e:
        print(f"gRPC error: {e.code()} - {e.details()}")
        sys.exit(1)


if __name__ == "__main__":
    main()
