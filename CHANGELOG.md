# Changelog

## 0.3.0

- Replaced `fsnotify` with direct Linux `inotify` via the Go standard library.
- Removed all third-party Go dependencies.
- Added hard filtering of browser temporary files before any `stat` call.
- Added path-level debounce.
- Added file stability waiting before organizing.
- Missing paths caused by rename/delete races are now ignored silently.
- Added `scripts/install-local.sh` to avoid accidentally running an old binary.
- Switched history storage to JSON Lines for a dependency-free baseline.

## 0.2.1

- Added temp suffix filtering and debounce around `fsnotify` events.

## 0.2.0

- Ignored missing paths caused by rename/delete races.
