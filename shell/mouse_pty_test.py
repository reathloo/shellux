#!/usr/bin/env python3
"""Check Zsh mouse routing without controlling a real Spotify session."""

import os
from pathlib import Path
import pty
import select
import signal
import tempfile
import time


ROOT = Path(__file__).resolve().parents[1]


def drain(fd, seconds=0.05):
    end = time.monotonic() + seconds
    output = bytearray()
    while time.monotonic() < end:
        if select.select([fd], [], [], min(0.02, end - time.monotonic()))[0]:
            try:
                output.extend(os.read(fd, 65536))
            except OSError:
                break
    return bytes(output)


def wait_for(fd, expected, seconds=4):
    end = time.monotonic() + seconds
    output = bytearray()
    while time.monotonic() < end:
        output.extend(drain(fd))
        if expected in output:
            return bytes(output)
    raise AssertionError(f"missing {expected!r}: {bytes(output)[-400:]!r}")


def main():
    with tempfile.TemporaryDirectory() as directory:
        fake = Path(directory, "shellux-fake")
        log = Path(directory, "calls")
        hover_log = Path(directory, "hover-path")
        fake.write_text(
            "#!/bin/sh\n"
            'printf "%s\\n" "$*" >> "$SHELLUX_MOUSE_LOG"\n'
            'case "$1" in --help) echo clear-scrollback;; --watch) printf "%s\\n" "$SHELLUX_HOVER_FILE" > "$SHELLUX_HOVER_LOG"; exec sleep 30;; profile) echo default;; esac\n'
        )
        fake.chmod(0o755)
        pid, fd = pty.fork()
        if pid == 0:
            os.environ.update(
                SHELLUX_BIN=str(fake), SHELLUX_MOUSE_LOG=str(log), SHELLUX_HOVER_LOG=str(hover_log),
                XDG_CONFIG_HOME=directory, TERM="xterm-256color",
            )
            os.execlp("zsh", "zsh", "-f")
        try:
            drain(fd, 0.2)
            os.write(fd, f"PROMPT='SX> '; source {ROOT / 'shell/shellux.zsh'}\n".encode())
            wait_for(fd, b"\x1b[?1003h\x1b[?1006h")
            end = time.monotonic() + 4
            while time.monotonic() < end and not hover_log.exists():
                drain(fd)
            hover_file = Path(hover_log.read_text().strip())
            os.write(fd, b"\x1b[<35;20;1M")
            end = time.monotonic() + 4
            while time.monotonic() < end and hover_file.read_text().strip() != "20 1":
                drain(fd)
            assert hover_file.read_text().strip() == "20 1"
            os.write(fd, b"\x1b[<35;20;2M")
            end = time.monotonic() + 4
            while time.monotonic() < end and hover_file.read_text().strip() != "0 0":
                drain(fd)
            assert hover_file.read_text().strip() == "0 0"
            assert "mouse " not in log.read_text(), "hover triggered a click"
            os.write(fd, b"\x1b[<0;20;5M\x1b[<0;20;5m")
            end = time.monotonic() + 4
            while time.monotonic() < end and log.read_text().count("mouse 20 5") != 1:
                drain(fd)
            assert log.read_text().count("mouse 20 5") == 1
            os.write(fd, b"echo INPUT_OK\n")
            output = wait_for(fd, b"INPUT_OK\r\n")
            assert b"\x1b[?1003l\x1b[?1006l" in output, "mouse mode stayed active during a command"
            assert log.read_text().count("mouse 20 5") == 1, "release triggered a second action"
            os.write(fd, b"shellux off\n")
            end = time.monotonic() + 4
            while time.monotonic() < end and hover_file.exists():
                drain(fd)
            assert not hover_file.exists(), "hover state file survived shellux off"
        finally:
            os.kill(pid, signal.SIGKILL)
            os.waitpid(pid, 0)
            os.close(fd)
    print("zsh dashboard mouse PTY test passed")


if __name__ == "__main__":
    main()
