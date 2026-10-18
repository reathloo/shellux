#!/usr/bin/env python3
"""Exercise concurrent resize and Focus -> Snapshot through the real watcher.

Run with a race-enabled binary to check accesses across the rendering and metric
goroutines. Spotify is replaced locally; no playback session is controlled.
"""
from datetime import datetime, timedelta, timezone
import fcntl
import json
import os
from pathlib import Path
import pty
import re
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time

BINARY = str(Path(sys.argv[1]).resolve())


def check_watch(settings, resize=False):
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        config = root / "config.json"
        if settings.get("selected_profile") == "focus":
            settings = {**settings, "focus_until": (datetime.now(timezone.utc) + timedelta(seconds=3)).isoformat()}
        config.write_text(json.dumps(settings))
        fake = root / "osascript"
        fake.write_text("#!/bin/sh\nexit 0\n")
        fake.chmod(0o755)
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 140, 0, 0))
        env = {**os.environ, "TERM": "xterm-256color", "XDG_CONFIG_HOME": directory,
               "XDG_DATA_HOME": directory, "PATH": directory + os.pathsep + os.environ["PATH"],
               "GORACE": "atexit_sleep_ms=0"}
        env.pop("NO_COLOR", None)
        process = subprocess.Popen([BINARY, "--watch", "--in-place", "--interval", "20ms", "--config", str(config)],
                                   cwd=directory, env=env, stdin=slave, stdout=slave, stderr=slave)
        os.close(slave)
        raw = bytearray()

        def drain(seconds):
            end = time.monotonic() + seconds
            while time.monotonic() < end:
                if select.select([master], [], [], max(0, min(0.02, end-time.monotonic())))[0]:
                    try:
                        raw.extend(os.read(master, 65536))
                    except OSError:
                        return

        try:
            deadline = time.monotonic() + 10
            while b"S H E L L U X" not in raw and time.monotonic() < deadline:
                drain(0.1)
            assert b"S H E L L U X" in raw, bytes(raw)[-1500:]
            if resize:
                for index in range(80):
                    columns, rows = (70, 15) if index % 2 else (140, 40)
                    fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack("HHHH", rows, columns, 0, 0))
                    process.send_signal(signal.SIGWINCH)
                    drain(0.04)
            else:
                deadline = time.monotonic() + 8
                while time.monotonic() < deadline:
                    drain(0.1)
                    plain = re.sub(rb"\x1b\[[0-?]*[ -/]*[@-~]", b"", bytes(raw))
                    if b"SNAPSHOT" in plain and re.search(rb"load\s+1m [0-9]", plain, re.IGNORECASE):
                        break
                else:
                    raise AssertionError(("Focus did not capture and display Snapshot metrics", plain[-3000:]))
                drain(0.2)
                captured = len(raw)
                drain(0.5)
                assert len(raw) == captured, "Snapshot continued drawing after capture"
            race = raw.find(b"WARNING: DATA RACE")
            assert race < 0, bytes(raw[race:race+3000])
        finally:
            if process.poll() is None:
                process.terminate()
            drain(0.3)
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
            os.close(master)
        assert process.returncode == 0, (process.returncode, bytes(raw)[-9000:])


if __name__ == "__main__":
    check_watch({"profile_selection": "auto", "language": "en"}, resize=True)
    print("watch: concurrent profile resize passed", flush=True)
    check_watch({"profile_selection": "manual", "selected_profile": "focus", "focus_minutes": 1,
                 "focus_previous_selection": "manual", "focus_previous_profile": "snapshot", "language": "en"})
    print("watch: Focus to Snapshot capture passed", flush=True)
