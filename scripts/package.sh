#!/bin/sh
# Build the v0.1.0-style release tarball: seal-VERSION-linux-x86_64.tar.gz + .sha256.
# Usage: VERSION=0.1.0 sh scripts/package.sh
set -eu
cd "$(dirname "$0")/.."
VERSION="${VERSION:?set VERSION, e.g. VERSION=0.1.0 sh scripts/package.sh}"
[ "$(uname -s)-$(uname -m)" = Linux-x86_64 ] || { echo 'Validated packaging target is Linux x86_64' >&2; exit 1; }
# Pure Go, no cgo: static binary, no host-specific instructions by default.
go build -trimpath -o seal .
name="seal-${VERSION}-linux-x86_64"
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT HUP INT TERM
mkdir -p "$stage/$name" dist
cp seal README.md seal.toml "$stage/$name/"
cp -R seal-out examples "$stage/$name/"
tar -C "$stage" -czf "dist/$name.tar.gz" "$name"
(cd dist && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")
printf 'Created dist/%s.tar.gz and checksum (not published)\n' "$name"
