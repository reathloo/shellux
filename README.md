# Shellux

An animated system dashboard for **Zsh and Bash on macOS and Linux**. Shellux
keeps system stats, project context, music, and your own visual style above the
prompt you already use.

![Shellux dashboard with a gold wordmark, system stats, and Spotify playback](docs/images/shellux-header.png)

[Install](#installation) · [Customize](#customize-shellux) ·
[Profiles](#profiles) · [Commands](#everyday-commands) ·
[Development](#development)

## Why Shellux?

Shellux turns unused terminal space into a live, configurable header without
replacing your prompt or shell workflow.

| | What you get |
| --- | --- |
| **System at a glance** | CPU, memory, storage, battery, network, load, swap, and other optional local metrics. |
| **Context when it matters** | Git, runtime, container, SSH, and last-command details can appear only in the relevant profile. |
| **A dashboard that adapts** | Automatic layouts for coding, small terminals, and remote sessions, plus manual focus and presentation modes. |
| **Your own look** | Animated themes, menu styles, backgrounds, custom colors, favorites, and saved layouts. |
| **Terminal-native controls** | Configure Shellux with keyboard, mouse, or touchpad; use interactive Spotify controls on macOS. |

## Installation

Install the latest release:

```sh
curl -fsSL https://raw.githubusercontent.com/reathloo/shellux/main/scripts/install-release.sh | sh
```

The installer detects your operating system and processor architecture,
verifies the downloaded archive with SHA-256, installs Shellux under
`~/.local`, and adds the integration to Zsh or Bash. Open a new terminal tab
when it finishes.

<details>
<summary><strong>Manual installation</strong></summary>

Download the archive and `checksums.txt` from
[GitHub Releases](https://github.com/reathloo/shellux/releases), verify the
archive, extract it, and run:

```sh
./install.sh
```

The manual installer prints the shell configuration lines instead of changing
your configuration automatically.

</details>

## Quick start

Open the visual settings menu:

```sh
shellux settings
```

Use the arrow keys to navigate, `Tab` to switch sections, and `Space` or
`Enter` to toggle or select. Press `Ctrl+S` to save or `Esc` to cancel.

> [!IMPORTANT]
> Changes take effect only after you save. Saving restarts the header and clears
> scrollback; canceling restores the previous settings and terminal background.

![Shellux settings with navigation and individually configurable menu items](docs/images/shellux-settings.png)

## Customize Shellux

The settings menu groups related controls instead of exposing one long list:

| Section | Purpose |
| --- | --- |
| **Menu items** | Choose rows and their independent detail switches; reorder them with `M`. |
| **Menu layouts** | Select automatic or manual profile handling, apply presets, and save or delete custom layouts. |
| **Themes** | Select, preview, favorite, install, update, or uninstall theme packages. |
| **Appearance** | Combine animations, menu styles, colors, and terminal backgrounds. |
| **General** | Change language and refresh behavior. |
| **Doctor** | Scan the local installation and offer confirmation-gated repairs. |

Shellux supports up to **11 visible menu slots**. Empty categories disappear
automatically. A saved layout contains row visibility, detail switches, and
their order; it does not change your theme, colors, language, or refresh
interval.

### Reorder items

In **Menu items**, select a section, row, or detail switch and press `M`:

- `↑` / `↓` moves a section or row.
- `←` / `→` moves a detail switch.
- `Enter` leaves move mode.
- `Shift+←` / `Shift+→` moves the selected switch directly.

### Themes and appearance

- Press `Enter` in **Appearance** to preview an animation, style, or background.
- Press `F` in **Themes** to mark a theme as a favorite.
- Press `U` to uninstall an inactive downloaded theme after confirmation.
- Installed catalog themes are checked for verified updates when **Themes** opens.
- Downloaded themes stay available offline; the live header itself does not
  access the network.

The default theme, general styles, and basic colors are built in. Additional
animation previews require the corresponding theme package to be installed.

## Profiles

Shellux uses one global profile mode: **Automatic** or **Manual**. Manual mode
keeps the selected profile in every directory; Automatic mode chooses a profile
from the current terminal context.

| Profile | Selection | Designed for |
| --- | --- | --- |
| **Standard** | Default | Your normal saved dashboard. |
| **Coding** | Automatic or manual | Git repository, runtimes, containers, and command context. |
| **Minimal** | Automatic or manual | Terminals narrower than 110 columns or shorter than 18 rows. |
| **SSH / Remote** | Automatic or manual | Incoming SSH sessions and remote diagnostics. |
| **Presentation** | Manual | Screen sharing with sensitive dashboard fields hidden. |
| **Focus** | Manual | A 1–240 minute timer beside the normal resource dashboard. |
| **Snapshot** | Manual | A fixed, local diagnostic view refreshed only on request. |

> [!NOTE]
> In Automatic mode, a small terminal selects **Minimal** first, an incoming SSH
> session selects **SSH / Remote** next, and a recognized project selects
> **Coding**. Profiles change layout and visible information, not the saved theme.

<details>
<summary><strong>Profile behavior in detail</strong></summary>

### Coding

Coding activates inside a Git repository or a project containing `go.mod`,
`package.json`, `pyproject.toml`, `Cargo.toml`, `pom.xml`, or a Gradle build
file. It can show local Git status, installed runtimes, Docker and Kubernetes
contexts, and the last command's result. Docker and Kubernetes alone do not
trigger the profile.

### Minimal

Minimal shows time, path, an optional Git branch, CPU, and memory with a compact
pulsing indicator. The regular layout returns as soon as the terminal has
enough room.

### SSH / Remote

Remote activates when `SSH_CONNECTION`, `SSH_CLIENT`, or `SSH_TTY` is set.
`SSH_AUTH_SOCK` alone does not count. The preset can show account, host,
endpoints, TTY, terminal type, path, connection status, uptime, CPU, and memory.
Missing values appear as `N/A`; Shellux does not contact the server to fill
them in.

### Presentation

Presentation keeps only date, time, uptime, CPU, memory, and connection status.
It hides paths, host and network names, IP addresses, Git and tool context,
logged-in users, and media metadata. Your prompt and existing terminal output
may still contain private information.

### Focus

Choose 15, 25, 30, 45, or 60 minutes, or enter a custom duration from 1 to 240
minutes. The timer uses an absolute deadline, survives reopening the terminal,
and restores the previous layout when it expires. Selecting another layout ends
the session early.

### Snapshot

Snapshot freezes the time, Shellux version, platform, shell, uptime, CPU,
memory, load, and connection status. Hostname, path, IP, and media details stay
hidden. Run `shellux reload` to capture a new local view; nothing is shared
automatically.

</details>

## Dashboard components

Rows and their detail switches are independently configurable. Optional rows
start hidden, and unavailable values appear as `N/A`.

<details>
<summary><strong>System and resource metrics</strong></summary>

- CPU, RAM, network traffic, and battery can show an optional eight-step
  sparkline for roughly the last minute.
- Optional rows include disk read/write throughput, gateway-ping loss with
  sample counts, distinct local users, microphone/camera activity, load
  averages, swap usage, and the GPU model or core count reported by macOS.
- Volume can show remaining free space with an independent, default-off switch.
- Gateway-ping loss can also mean ICMP is blocked; it is not a measurement of
  all network traffic.
- GPU utilization is not estimated. Media activity currently reports `N/A` on
  Linux when the platform does not expose the required information.
- Thermal reports a state, not a temperature in degrees Celsius. Optional macOS
  setup with `shellux thermal enable` requires administrator privileges.

</details>

<details>
<summary><strong>Coding data</strong></summary>

The **CODING** section contains five default-off rows:

- Git branch and repository.
- Local ahead/behind counts for the configured upstream.
- Go, Node, Python, and Java versions.
- Docker and Kubernetes contexts.
- Last command duration and exit code.

Git comparisons use local refs and never fetch. Shellux probes optional tools
only when their rows are enabled. Last-command data requires the sourced Zsh or
Bash integration and has whole-second resolution.

</details>

<details>
<summary><strong>Spotify and mouse interaction</strong></summary>

On macOS, **SESSION → Spotify** shows the current artist and title in a scrolling
21-column field. The default-off **Album** switch adds the fixed album name.
The **Progress** row can show playback time, a seekable progress bar, play/pause,
and previous/next controls.

In sourced Zsh, click playback controls directly while the prompt is idle.
Hovering a letter in **S H E L L U X** highlights it. Mouse reporting is active
only while Shellux owns the prompt; hold `Shift` when selecting terminal text if
your terminal reserves normal dragging for mouse reports.

Bash Readline cannot safely route dashboard clicks, so use `shellux spotify` as
the keyboard fallback. Its `T` shortcut toggles playback.

</details>

## Language

Under **General → Language**, choose **Automatic (system)**, **English**, or
**Deutsch** for the dashboard and settings. Automatic follows the system or
terminal locale and falls back to English. Command names remain unchanged.

## Doctor

`shellux doctor` checks the installed Zsh/Bash integration, downloaded theme
packages, owner permissions on Shellux files, and orphaned live-renderer
processes.

The scan is local, read-only, and does not access the network. Repairs are shown
only for targets Shellux can identify safely and always require confirmation.
Theme repair may download a verified package; permission repair never uses
administrator privileges. Open a new terminal tab after repairing shell
integration.

## Everyday commands

| Command | Action |
| --- | --- |
| `shx` | Bring back the header and clear scrollback. |
| `shellux settings` | Open the visual settings menu. |
| `shellux reload` | Restart the live header. |
| `shellux off` / `shellux on` | Hide or restore the header in the current tab. |
| `shellux status` | Show the current theme and refresh settings. |
| `shellux doctor` | Check the local installation and renderer state. |
| `shellux spotify` | Open keyboard-driven Spotify controls. |
| `shellux help` | List every command and customization option. |

The header pauses while commands run so output remains readable and scrollable.
Bring it back with `shx` or `shellux reload`.

## Platform notes

- Some metrics depend on the operating system, hardware, and permissions.
- Spotify requires the desktop app and may request macOS Automation permission.
- Background colors require a terminal with OSC 11 support.
- Shellux preserves your existing prompt; prompt appearance and behavior remain
  the responsibility of your shell configuration.

## Development

Development requires **Go 1.27.2 or later**. On macOS, install the Xcode Command
Line Tools as well.

From a cloned repository:

```sh
go mod verify
go run ./tools/theme-pack -check
go test ./...
go test -race ./...
go vet ./...
sh shell/shellux_test.sh
sh scripts/install_test.sh
sh -n scripts/install.sh scripts/install-release.sh
```

[CI](.github/workflows/ci.yml) also checks formatting, builds Shellux, and tests
shell integration, profile transitions, mouse routing, concurrent rendering,
and the interactive settings menu on macOS and Linux.

To publish your own theme packages, see the [theme guide](themes/README.md).

Shellux is source-available under the
[PolyForm Noncommercial License 1.0.0](LICENSE). Commercial use or sale requires
separate permission from the copyright holder.
