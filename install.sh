#!/bin/sh
# One-command install for the Annalist + Paldron + Docket suite.
# Builds from local checkouts when present, else prints release URLs.
# Usage: ./install.sh [--prefix ~/.local]
set -e
PREFIX="${1:-$HOME/.local}"
BIN="$PREFIX/bin"
mkdir -p "$BIN"
have() { [ -d "$1" ]; }

if have "$HOME/projects/annalist"; then
  (cd "$HOME/projects/annalist" && zig build -Doptimize=ReleaseSafe 2>/dev/null || zig build)
  cp "$HOME/projects/annalist/zig-out/bin/annalist" "$BIN/"
  echo "installed: annalist"
else
  echo "missing: ~/projects/annalist (release: https://github.com/GregDixonMXN/annalist/releases)"
fi

if have "$HOME/projects/paldron"; then
  (cd "$HOME/projects/paldron" && go build -o "$BIN/paldron" ./cmd/paldron)
  echo "installed: paldron"
else
  echo "missing: ~/projects/paldron (https://github.com/GregDixonMXN/paldron)"
fi

go build -o "$BIN/docket" .
echo "installed: docket"
echo "PATH: $BIN (annalist run -- paldron exec -- <cmd>)"
