#!/usr/bin/env python3
"""Check automatic Coding transitions in real sourced Zsh and Bash PTYs."""
import fcntl
from datetime import datetime, timedelta, timezone
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

REPO = Path(__file__).resolve().parents[1]
BINARY = str(Path(sys.argv[1]).resolve()) if len(sys.argv) == 2 else str(REPO / "shellux")


def check(shell):
    with tempfile.TemporaryDirectory() as temporary:
        root = Path(temporary)
        project = root / "project"
        project.mkdir()
        (project / "go.mod").write_text("module example.test/project\n")
        settings = root / "shellux" / "config.json"
        settings.parent.mkdir()
        settings.write_text(json.dumps({"profile_selection": "auto", "style": "aurora", "background": "#123456"}))
        original = settings.read_bytes()
        pid, fd = pty.fork()
        if pid == 0:
            os.chdir(root)
            os.environ.update(XDG_CONFIG_HOME=temporary, XDG_DATA_HOME=temporary,
                              SHELLUX_RELEASE_BASE_URL="https://127.0.0.1:1",
                              SHELLUX_BIN=BINARY, TERM="xterm-256color")
            os.environ.pop("NO_COLOR", None)
            argv = ["zsh", "-f"] if shell == "zsh" else ["bash", "--noprofile", "--norc", "-i"]
            os.execvp(shell, argv)
        raw = bytearray()

        def drain(seconds=0.05):
            deadline = time.monotonic() + seconds
            while time.monotonic() < deadline:
                ready, _, _ = select.select([fd], [], [], min(0.02, max(0, deadline - time.monotonic())))
                if ready:
                    try:
                        data = os.read(fd, 65536)
                    except OSError:
                        return
                    raw.extend(data)
                    if b"\x1b[6n" in data:
                        os.write(fd, b"\x1b[1;1R")

        def command(line, expected):
            start = len(raw)
            os.write(fd, line.encode() + b"\n")
            deadline = time.monotonic() + 12
            while time.monotonic() < deadline:
                drain()
                output = bytes(raw[start:])
                if expected in output and b"SX> " in output:
                    return output
            raise AssertionError((shell, line, expected, bytes(raw[start:])[-1500:]))

        def resize(columns, rows, expected):
            start = len(raw)
            fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, columns, 0, 0))
            os.kill(pid, signal.SIGWINCH)
            deadline = time.monotonic() + 8
            while time.monotonic() < deadline:
                drain()
                if expected in raw[start:]:
                    drain(0.35)
                    return bytes(raw[start:])
            raise AssertionError((shell, "resize", columns, rows, expected, bytes(raw[start:])[-1500:]))

        try:
            fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 140, 0, 0))
            os.kill(pid, signal.SIGWINCH)
            drain(0.2)
            command("PS1=$'SX\\x3e '; PS2='MORE> '; source " + str(REPO / ("shell/shellux." + shell)), b"RESOURCES")
            resize(70, 15, b" Shellux")
            resize(140, 40, b"RESOURCES")
            enter_output = command("cd project", b"CODING")
            assert b"\x1b]11;#123456\x07" in enter_output, "Coding changed the selected background"
            resize(70, 15, b" Shellux")
            resize(140, 40, b"CODING")
            exit_output = command("cd ..", b"RESOURCES")
            assert b"\x1b]11;#123456\x07" in exit_output, "saved background changed after Coding"
            assert settings.read_bytes() == original, "profile transition changed saved settings"
            settings.write_text(json.dumps({"profile_selection": "manual", "selected_profile": "coding", "style": "aurora", "background": "#123456"}))
            enter_output = command("shellux reload", b"CODING")
            assert b"\x1b]11;#123456\x07" in enter_output
            settings.write_text(json.dumps({"profile_selection": "manual", "selected_profile": "presentation", "style": "aurora", "background": "#123456"}))
            presentation_output = command("shellux reload", b"RESOURCES")
            assert b"\x1b]11;#123456\x07" in presentation_output
            presentation_header = presentation_output[presentation_output.rfind(b"UPTIME"):]
            for sensitive in (b"PATH", b"BRANCH", b"SPOTIFY", b"PRIVATE"):
                assert sensitive not in presentation_header, (shell, "Presentation showed", sensitive)
            settings.write_text(json.dumps({"profile_selection": "manual", "selected_profile": "snapshot", "style": "aurora", "background": "#123456"}))
            snapshot_output = command("shellux reload", b"SNAPSHOT")
            assert b"captured" in snapshot_output.lower() and b"shellux v" in snapshot_output.lower()
            drain(0.5)
            stable_start = len(raw)
            drain(1.5)
            assert b"SNAPSHOT" not in raw[stable_start:], (shell, "Snapshot kept redrawing", bytes(raw[stable_start:])[-500:])
            assert b"SNAPSHOT" in command("shellux reload", b"SNAPSHOT"), (shell, "Snapshot did not recapture")
            focus_until = (datetime.now(timezone.utc) + timedelta(seconds=7)).isoformat()
            settings.write_text(json.dumps({"profile_selection": "manual", "selected_profile": "focus", "focus_minutes": 1,
                                            "focus_until": focus_until, "focus_previous_selection": "manual",
                                            "style": "aurora", "background": "#123456"}))
            focus_output = command("shellux reload", b"SPOTIFY")
            drain(0.2)
            focus_output += bytes(raw[-1000:])
            assert b"PROGRESS" in focus_output, (shell, "Focus hid playback progress")
            assert b"\x1b]11;#000000\x07" in focus_output, (shell, "Focus did not set a black background")
            narrow_focus = resize(80, 24, b"SPOTIFY")
            narrow_plain = re.sub(rb"\x1b\[[0-?]*[ -/]*[@-~]", b"", narrow_focus)
            assert b"######" in narrow_plain and narrow_plain.rfind(b"SPOTIFY") < narrow_plain.rfind(b"######"), (shell, "Focus clock did not stack below the menu", narrow_plain[-1800:])
            resize(140, 40, b"SPOTIFY")
            expiry_start = len(raw)
            deadline = time.monotonic() + 8
            while time.monotonic() < deadline:
                drain()
                expired = bytes(raw[expiry_start:])
                if b"RESOURCES" in expired and b"\x1b]11;#123456\x07" in expired:
                    break
            else:
                raise AssertionError((shell, "Focus did not restore layout and background", bytes(raw[expiry_start:])[-1200:]))
            settings.write_text(json.dumps({"profile_selection": "manual", "selected_profile": "", "style": "aurora", "background": "#123456"}))
            exit_output = command("shellux reload", b"RESOURCES")
            assert b"\x1b]11;#123456\x07" in exit_output
            settings.write_bytes(original)
            command("shellux reload", b"RESOURCES")
            resize(70, 15, b" Shellux")
            command("printf 'ACTIVE=%s\\n' \"$(command \"$SHELLUX_BIN\" profile resolve)\"", b"ACTIVE=minimal")
            print(shell + ": automatic profile PTY transitions passed")
        finally:
            try:
                os.write(fd, b"\x03exit\n")
                drain()
                os.kill(pid, signal.SIGHUP)
            except (OSError, ProcessLookupError):
                pass
            os.close(fd)
            os.waitpid(pid, 0)


for candidate in ("zsh", "bash"):
    if subprocess.run(["sh", "-c", "command -v " + candidate], stdout=subprocess.DEVNULL).returncode == 0:
        check(candidate)
