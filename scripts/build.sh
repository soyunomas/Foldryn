#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
go build -o foldrynd ./cmd/foldrynd
go build -o foldrynctl ./cmd/foldrynctl
./foldrynd -version
