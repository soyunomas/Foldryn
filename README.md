# Foldryn

Foldryn is a lightweight, rule-based file automation daemon for Linux.

It acts as a background service that continuously watches specified directories (like your Downloads folder) and automatically organizes incoming files into corresponding subdirectories based on their file extensions or names, using customizable templates.

## Features

- **Dependency-Free:** Built entirely with the Go standard library, natively leveraging Linux `inotify` for maximum performance and minimal overhead.
- **Smart Placeholders:** Automatically categorizes files into folders using placeholders like `{year}`, `{month}`, `{day}`, `{basename}`, and `{ext}`.
- **Ghost File Immunity:** Intelligently ignores 0-byte placeholder files created by web browsers at the start of a download, only acting when the actual file finishes downloading.
- **Temporary File Filtering:** Automatically ignores `.part`, `.crdownload`, `.tmp`, and other common ephemeral files to prevent errors and duplicates.
- **Dry-Run Mode:** Test your rules safely before actually moving any files.
- **Desktop Notifications:** Get notified via `notify-send` when a file is successfully processed or if an error occurs.
- **History Tracking:** Logs all actions to a local JSON Lines file so you can always check where a file was moved.

## Requirements

- Linux operating system (uses `inotify`)
- Go 1.20+ (for building from source)

## Building from Source

Clone the repository and compile the daemon and the CLI tool:

```bash
go build -o foldrynd ./cmd/foldrynd
go build -o foldrynctl ./cmd/foldrynctl
```

Check the version:
```bash
./foldrynd -version
```

## Quick Local Install

We provide a script to easily install the binaries to `~/.local/bin`, copy the default configuration, and set up a systemd user service.

```bash
./scripts/install-local.sh
```

## Configuration

The configuration file is located at `~/.config/foldryn/config.toml`. Here is an example of how to configure it:

```toml
[app]
dry_run = true # Set to false to start moving files
history_path = "~/.local/share/foldryn/history.jsonl"
notifications = true
settle_delay_ms = 1500

[[watch]]
path = "~/Downloads"
recursive = false

[[rules]]
name = "Images"
enabled = true
extensions = [".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"]
destination = "~/Downloads/Images/{year}/{month}"
rename = "{basename}{ext}"

[[rules]]
name = "Disk Images"
enabled = true
extensions = [".iso", ".img", ".vdi", ".vmdk"]
destination = "~/Downloads/ISOs"
rename = "{basename}{ext}"
```

## Running

**Run manually:**
```bash
~/.local/bin/foldrynd -config ~/.config/foldryn/config.toml
```

**Run as a systemd user service (Recommended):**
```bash
systemctl --user daemon-reload
systemctl --user enable --now foldrynd
```

Check the logs of the service:
```bash
journalctl --user -u foldrynd -f
```

## Viewing History

You can view the history of moved files using the CLI tool:

```bash
foldrynctl history -n 20
```
