# Architecture

Foldryn currently has three main layers:

1. `watcher`: Linux inotify loop, debounce and file-stability checks.
2. `rules`: extension and regex matching.
3. `organizer`: destination rendering, dry-run, move/copy fallback and history.

The daemon is intentionally separate from any future GUI. A Wails UI can later control the daemon through a small local API or Unix socket without changing the file automation core.
