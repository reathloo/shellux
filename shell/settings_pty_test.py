#!/usr/bin/env python3
"""Real sourced-shell/settings lifecycle checks. Uses only Python's standard library.
Run from the repository: python3 shell/settings_pty_test.py /path/to/shellux
This checks terminal protocol and processes, not Terminal.app's native rendering.
"""
import fcntl
import json
import os
from pathlib import Path
import pty
import re
import select
import signal
import shutil
import struct
import subprocess
import sys
import tempfile
import termios
import time

REPO = Path(__file__).resolve().parents[1]
BINARY = str(Path(sys.argv[1]).resolve()) if len(sys.argv) == 2 else str(REPO / "shellux")


def check_cli():
    for args in (["settings"], ["settings", "extra"]):
        result = subprocess.run([BINARY, *args], input=b"", capture_output=True)
        assert result.returncode == 2, (args, result.returncode)
        assert b"\x1b" not in result.stdout, "redirected CLI entered terminal mode"
    # Separately reject redirected stdin even when stdout is a terminal.
    master, slave = pty.openpty()
    try:
        result = subprocess.run([BINARY, "settings"], input=b"", stdout=slave, stderr=subprocess.PIPE)
        assert result.returncode == 2
        result = subprocess.run([BINARY, "settings"], stdin=slave, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        assert result.returncode == 2
    finally:
        os.close(master)
        os.close(slave)


def check_shell(shell):
    with tempfile.TemporaryDirectory() as directory:
        data_dir = str(Path(directory) / "data")
        fixture_dir = Path(data_dir) / "shellux/themes"
        fixture_dir.mkdir(parents=True)
        catalog = json.loads((REPO / "themes/catalog.json").read_text())
        entry = next(e for e in catalog["themes"] if e["name"] == "orb")
        shutil.copyfile(REPO / "themes" / entry["package"], fixture_dir / Path(entry["package"]).name)
        (fixture_dir / "orb.installed.json").write_text(json.dumps({"format": 1, "themes": [entry]}))
        test_env = {**os.environ, "XDG_CONFIG_HOME": directory, "XDG_DATA_HOME": data_dir, "LC_ALL": "C",
                    "SHELLUX_THEME_CATALOG_URL": "https://127.0.0.1:1/catalog.json"}
        names = subprocess.check_output([BINARY, "animations"], env=test_env, text=True).splitlines()[1:]
        orb_index = [n.strip() for n in names].index("orb")
        pid, fd = pty.fork()
        if pid == 0:
            os.chdir(REPO)
            os.environ.update(test_env)
            os.environ.update(TERM="xterm-256color", SHELLUX_BIN=BINARY)
            # Exercise actual terminal colors even if the test runner disables them.
            os.environ.pop("NO_COLOR", None)
            argv = ["zsh", "-f"] if shell == "zsh" else ["bash", "--noprofile", "--norc", "-i"]
            os.execvp(shell, argv)
        raw = bytearray()
        config_path = Path(directory) / "shellux/config.json"

        def resize(columns, rows):
            fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, columns, 0, 0))
            os.kill(pid, signal.SIGWINCH)

        def drain(seconds=0.1):
            deadline = time.monotonic() + seconds
            while time.monotonic() < deadline:
                ready, _, _ = select.select([fd], [], [], max(0, min(0.02, deadline - time.monotonic())))
                if ready:
                    try:
                        data = os.read(fd, 65536)
                    except OSError:
                        return
                    raw.extend(data)
                    if b"\x1b[6n" in data:
                        os.write(fd, b"\x1b[1;1R")

        def wait_for(needle, start, timeout=8):
            deadline = time.monotonic() + timeout
            while time.monotonic() < deadline:
                drain(0.03)
                if needle in raw[start:]:
                    drain(0.1)
                    return
            raise AssertionError((shell, "timeout", needle, bytes(raw[start:])[-1200:]))

        def command(text):
            start = len(raw)
            os.write(fd, text.encode() + b"\n")
            wait_for(b"SX> ", start)
            return bytes(raw[start:])

        def renderers():
            processes = subprocess.check_output(["ps", "-axo", "command="], text=True)
            return len(re.findall(r"--watch --in-place --parent-pid " + str(pid) + r"\b", processes))

        def open_ui():
            before = termios.tcgetattr(fd)
            start = len(raw)
            os.write(fd, b"shellux settings\n")
            wait_for(b"\x1b[?1049h", start)
            drain(0.3)
            assert renderers() == 0, (shell, "renderer running during settings")
            assert b"Settings" in raw[start:], (shell, "UI did not render")
            return start, before

        def close_ui(keys, start, before, count):
            os.write(fd, keys)
            wait_for(b"SX> ", start)
            assert b"\x1b[?1049l" in raw[start:], (shell, "alternate screen retained")
            assert termios.tcgetattr(fd) == before, (shell, "terminal mode not restored")
            assert renderers() == count, (shell, "wrong renderer count", renderers(), count)
            return bytes(raw[start:])

        try:
            resize(140, 40)
            drain(0.2)
            command("PS1=$'SX\\x3e '; PS2='MORE> '; source " + str(REPO / ("shell/shellux." + shell)))
            assert renderers() == 1
            # Open from a visible header, cancel without reload or config write.
            start, before = open_ui()
            output = close_ui(b"\x1b", len(raw), before, 0)
            assert b"\x1b[3J" not in output, "cancellation cleared scrollback"
            assert not config_path.exists()
            assert b"EXIT_0" in command("printf 'EXIT_%s\\n' $?")
            # Dirty cancellation defaults to continue editing; then discard.
            start, before = open_ui()
            os.write(fd, b"\t ")
            drain()
            os.write(fd, b"\x1b")
            wait_for(b"Discard unsaved changes?", start)
            os.write(fd, b"\r")  # default: continue editing
            drain()
            assert b"\x1b[?1049l" not in raw[start:]
            os.write(fd, b"\x1b")
            drain(0.2)
            output = close_ui(b"\x1b[C\r", len(raw), before, 0)
            assert not config_path.exists()
            assert b"\x1b[3J" not in output
            # Save from paused state and verify the entire write and one renderer.
            start, before = open_ui()
            output = close_ui(b"\t \x13", len(raw), before, 1)
            assert b"\x1b[3J" in output, "save did not clear scrollback"
            saved = config_path.read_bytes()
            assert json.loads(saved)["visible"]["platform"] is False
            # Repeated opening and Ctrl+C: no reload, identical file, exit 130.
            for _ in range(2):
                start, before = open_ui()
                close_ui(b"\x03", len(raw), before, 0)
                assert config_path.read_bytes() == saved
                assert b"EXIT_130" in command("printf 'EXIT_%s\\n' $?")
            # Resize during a dirty draft, keep it, save after enlarging.
            start, before = open_ui()
            os.write(fd, b"\t ")
            drain()
            resize(50, 12)
            wait_for(b"64", start)
            assert renderers() == 0
            resize(100, 32)
            drain(0.25)
            close_ui(b"\x13", len(raw), before, 1)
            assert json.loads(config_path.read_bytes())["visible"]["platform"] is True
            # Real text editing and independent appearance selectors.
            start, before = open_ui()
            os.write(fd, b"\x1b[B\x1b[B\x1b[B\r\r")  # General -> interval
            drain(0.2)
            os.write(fd, b"\x01\x0b500ms\r")  # Ctrl+A, Ctrl+K, duration, Enter
            drain(0.1)
            close_ui(b"\x13", len(raw), before, 1)
            assert json.loads(config_path.read_bytes())["refresh_interval"] == "500ms"
            start, before = open_ui()
            os.write(fd, b"\x1b[B\x1b[B\r\r")  # Appearance -> animation
            drain(0.1)
            os.write(fd, b"\x1b[A" * 50 + b"\x1b[B" * orb_index + b"\r")  # start at top, then orb
            drain(0.1)
            style_start = len(raw)
            os.write(fd, b"\x1b[B\r")  # style selector
            # Bubble Tea may redraw only the changed portion of the title, so
            # wait for a stable selector entry instead of the full heading.
            wait_for(b"luxury", style_start)
            os.write(fd, b"\x1b[A" * 50 + b"\x1b[B")  # top -> neon
            drain(0.1)
            os.write(fd, b"\r")
            wait_for(b"38;5;201", style_start)
            assert b"38;5;201" in raw[style_start:], ("UI did not immediately adopt neon accents", bytes(raw[style_start:])[-2000:])
            assert b"38;5;45" in raw[style_start:], "UI did not immediately adopt neon border colors"
            os.write(fd, b"\x1b[B\r" + b"\x1b[B" * 30 + b"\r")  # custom background
            drain(0.1)
            os.write(fd, b"\x01\x0b#123456\r")
            drain(0.1)
            output = close_ui(b"\x13", len(raw), before, 1)
            appearance = json.loads(config_path.read_bytes())
            assert (appearance["animation"], appearance["style"], appearance["background"]) == ("orb", "neon", "#123456")
            assert b"\x1b]11;" in output, "saved background was not applied"
            start, before = open_ui()
            theme_list = subprocess.check_output([BINARY, "themes"], env=test_env, text=True)
            theme_count = len(re.findall(r"^\s+\S+\s+\((?:default|built-in|custom|download|installed)\)", theme_list, re.MULTILINE))
            os.write(fd, b"\x1b[B\r" + b"\x1b[B" * (theme_count + 1) + b"\r")  # Themes -> new, after random
            drain(0.1)
            os.write(fd, b"my-ui-theme\r")
            drain(0.1)
            close_ui(b"\x13", len(raw), before, 1)
            assert json.loads(config_path.read_bytes())["themes"]["my-ui-theme"]["background"] == "#123456"
            # New metric details: keyboard selection in a narrow window, saved
            # through the same UI transaction and exactly one renderer afterward.
            report = subprocess.check_output([BINARY, "show"], env=test_env, text=True)
            menu_rows = re.findall(r"^  (\S+)\s+(?:on|off)(.*)$", report, re.MULTILINE)
            names = [name for name, _ in menu_rows]
            parts = {name: re.findall(r"(\w+): (?:on|off)", details) for name, details in menu_rows}
            for item, part, label, key in (
                ("cpu", "top", b"Top process", "cpu.top"),
                ("ram", "pressure", b"Pressure", "ram.pressure"),
                ("battery", "bar", b"Bar", "battery.bar"),
                ("battery", "remaining", b"Remaining", "battery.remaining"),
                ("status", "connection", b"Connection", "network-status.connection"),
                ("status", "name", b"Network name", "network-status.name"),
                ("date", "timezone", b"Timezone", "date.timezone"),
                ("date", "utc", b"UTC offset", "date.utc"),
                ("traffic", "latency", b"Latency", "network-traffic.latency"),
            ):
                resize(64, 18)
                start, before = open_ui()
                os.write(fd, b"\t" + b"\x1b[B" * names.index(item) + b"\x1b[C" * (parts[item].index(part) + 1))
                drain(0.2)
                plain = re.sub(rb"\x1b\[[0-?]*[ -/]*[@-~]", b"", bytes(raw[start:]))
                # Bubble Tea may reuse the leading 'S' from the previous screen
                # line, so the raw incremental stream need not contain the whole
                # 'Selected:' prefix. The full label plus persisted key verifies
                # focus; model tests check the complete rendered screen.
                assert label in plain, (shell, "component focus missing", item, part, plain[-800:])
                close_ui(b" \x13", len(raw), before, 1)
                assert json.loads(config_path.read_bytes())["visible"][key] is True, (item, part, "component not saved")
            resize(140, 40)
            # Enable date by freeing the shell slot; keep the two newly saved
            # details and the normal date value together in one header entry.
            start, before = open_ui()
            os.write(fd, b"\t" + b"\x1b[B" * names.index("shell") + b" " + b"\x1b[B" * (names.index("date") - names.index("shell")) + b" ")
            drain(0.2)
            output = close_ui(b"\x13", len(raw), before, 1)
            enabled = json.loads(config_path.read_bytes())["visible"]
            assert enabled["date"] is True and enabled["shell"] is False
            assert enabled["date.timezone"] is True and enabled["date.utc"] is True
            assert b"UTC" in output, "saved date did not show its UTC offset"
            assert "↳ TOP".encode() in output, "top process was not a separate indented row"
            assert b"load" in output, "CPU load label was not written out"
            assert re.search(rb"\([^\r\n]*[0-9]+%\)", output), "CPU/process percentage parentheses missing"
            # Each Traffic detail is independent and does not consume another slot.
            for part, label in (("upload", b"Upload"), ("download", b"Download"), ("latency", b"Latency")):
                for expected in (False, True):
                    start, before = open_ui()
                    os.write(fd, b"\t" + b"\x1b[B" * names.index("traffic") + b"\x1b[C" * (parts["traffic"].index(part) + 1))
                    drain(0.2)
                    assert label in raw[start:], ("Traffic component focus missing", part)
                    output = close_ui(b" \x13", len(raw), before, 1)
                    enabled = json.loads(config_path.read_bytes())["visible"]
                    assert enabled["network-traffic." + part] is expected
                    assert enabled["volume"] is True, "Traffic detail used a separate menu slot"
            plain = re.sub(rb"\x1b\[[0-?]*[ -/]*[@-~]", b"", output)
            traffic_line = re.search(rb"TRAFFIC[^\r\n]*UP [^\r\n]*DOWN [^\r\n]*(?:[0-9]+\.[0-9] ms|N/A)", plain)
            assert traffic_line, "Latency did not appear inline after upload/download"
            assert b"Gateway" not in traffic_line.group() and "↳ PING".encode() not in plain, "Old Ping row returned"
            # Explicitly off stays off on save and cancel.
            command("shellux off")
            start, before = open_ui()
            close_ui(b"\t \x13", len(raw), before, 0)
            assert b"ENABLED_0" in command("printf 'ENABLED_%s\\n' $shellux_enabled")
            # External config rewrite cannot be overwritten by the draft.
            start, before = open_ui()
            external = config_path.read_bytes() + b" "
            config_path.write_bytes(external)
            os.write(fd, b"\x13")
            wait_for(b"changed externally", start)
            assert b"\x1b[?1049l" not in raw[start:]
            assert config_path.read_bytes() == external
            close_ui(b"\x1b", len(raw), before, 0)
            # Signal interruption also restores terminal mode and returns 130.
            start, before = open_ui()
            children = subprocess.check_output(["ps", "-axo", "pid=,ppid=,command="], text=True).splitlines()
            ui = [int(line.split()[0]) for line in children if len(line.split()) > 2 and int(line.split()[1]) == pid and line.endswith(" settings")]
            assert len(ui) == 1, (shell, "cannot identify UI child", ui)
            exit_start = len(raw)
            os.kill(ui[0], signal.SIGINT)
            wait_for(b"SX> ", exit_start)
            assert b"\x1b[?1049l" in raw[exit_start:]
            assert termios.tcgetattr(fd) == before
            assert b"EXIT_130" in command("printf 'EXIT_%s\\n' $?")
            assert renderers() == 0
            print(shell + ": settings PTY lifecycle passed")
        finally:
            try:
                os.write(fd, b"\x03exit\n")
                drain(0.1)
                os.kill(pid, signal.SIGHUP)
            except (OSError, ProcessLookupError):
                pass
            os.close(fd)
            os.waitpid(pid, 0)


check_cli()
for test_shell in ("zsh", "bash"):
    if subprocess.run(["sh", "-c", "command -v " + test_shell], stdout=subprocess.DEVNULL).returncode == 0:
        check_shell(test_shell)
