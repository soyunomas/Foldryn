#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

go build -o foldrynd ./cmd/foldrynd
go build -o foldrynctl ./cmd/foldrynctl

install -Dm755 foldrynd "$HOME/.local/bin/foldrynd"
install -Dm755 foldrynctl "$HOME/.local/bin/foldrynctl"
install -Dm644 systemd/foldrynd.service "$HOME/.config/systemd/user/foldrynd.service"

if [ ! -f "$HOME/.config/foldryn/config.toml" ]; then
  install -Dm644 configs/foldryn.example.toml "$HOME/.config/foldryn/config.toml"
fi

systemctl --user daemon-reload

echo "Installed Foldryn:"
"$HOME/.local/bin/foldrynd" -version
echo "Config: $HOME/.config/foldryn/config.toml"
echo "Start manually: ~/.local/bin/foldrynd -config ~/.config/foldryn/config.toml"
echo "Or enable systemd user service: systemctl --user enable --now foldrynd"
