#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SRC_DIR="$SCRIPT_DIR/../app/src/main/assets/linux-server"
OUT_DIR="$SCRIPT_DIR/../out"

mkdir -p "$OUT_DIR"

echo "[BUILD] Linux server (amd64)..."
cd "$SRC_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$OUT_DIR/wdtt-server" .

echo "[BUILD] Linux server (arm64)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$OUT_DIR/wdtt-server-arm64" .

echo "[OK] Built:"
ls -lh "$OUT_DIR/wdtt-server"*
