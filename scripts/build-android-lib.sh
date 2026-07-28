#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SRC_DIR="$SCRIPT_DIR/../app/src/main/assets/android-client"
OUT_DIR="$SCRIPT_DIR/../app/src/main/jniLibs"

declare -A TARGETS
TARGETS=(
    ["arm64"]="aarch64-linux-android21-clang"
    ["arm"]="armv7a-linux-androideabi21-clang"
    ["x86_64"]="x86_64-linux-android21-clang"
    ["x86"]="i686-linux-android21-clang"
)

for arch in "${!TARGETS[@]}"; do
    cc="${TARGETS[$arch]}"
    echo "[BUILD] android-client ($arch)..."
    cd "$SRC_DIR"
    CGO_ENABLED=1 \
    GOOS=android \
    GOARCH="$arch" \
    CC="$cc" \
    go build -buildmode=c-shared -ldflags="-s -w" \
        -o "$OUT_DIR/$arch/libclient.so" .
done

echo "[OK] Built libclient.so for all architectures"
find "$OUT_DIR" -name "libclient.so" -exec ls -lh {} \;
