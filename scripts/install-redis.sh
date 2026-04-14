#!/usr/bin/env bash
# Download and build Redis from source if redis-server is not available.
#
# Installs into a local directory (default: ./vendor/redis) so it does not
# require root privileges.  Exports REDIS_BIN pointing at the built binary.
#
# Usage:
#   source scripts/install-redis.sh          # sets REDIS_BIN
#   scripts/install-redis.sh                 # prints path to redis-server
#
set -euo pipefail

REDIS_VERSION="${REDIS_VERSION:-7.2.6}"
VENDOR_DIR="${VENDOR_DIR:-$(cd "$(dirname "$0")/.." && pwd)/vendor/redis}"

# ── Already installed? ──────────────────────────────────────
if command -v redis-server &>/dev/null; then
    REDIS_BIN="$(command -v redis-server)"
    echo "redis-server already available: $REDIS_BIN"
    export REDIS_BIN
    return 0 2>/dev/null || exit 0
fi

# Check vendor dir from a previous run
if [[ -x "$VENDOR_DIR/src/redis-server" ]]; then
    REDIS_BIN="$VENDOR_DIR/src/redis-server"
    echo "redis-server found in vendor: $REDIS_BIN"
    export REDIS_BIN
    return 0 2>/dev/null || exit 0
fi

# ── Pre-flight: need a C compiler ───────────────────────────
if ! command -v make &>/dev/null || ! command -v cc &>/dev/null; then
    echo "ERROR: 'make' and a C compiler (cc) are required to build Redis."
    echo "  macOS:  xcode-select --install"
    echo "  Debian: sudo apt-get install build-essential"
    echo "  RHEL:   sudo yum groupinstall 'Development Tools'"
    return 1 2>/dev/null || exit 1
fi

# ── Download ────────────────────────────────────────────────
TARBALL_URL="https://github.com/redis/redis/archive/refs/tags/${REDIS_VERSION}.tar.gz"
TMPDIR_DL="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_DL"' EXIT

echo "Downloading Redis ${REDIS_VERSION}..."
if command -v curl &>/dev/null; then
    curl -fsSL "$TARBALL_URL" -o "$TMPDIR_DL/redis.tar.gz"
elif command -v wget &>/dev/null; then
    wget -q "$TARBALL_URL" -O "$TMPDIR_DL/redis.tar.gz"
else
    echo "ERROR: curl or wget required to download Redis."
    return 1 2>/dev/null || exit 1
fi

# ── Extract & build ─────────────────────────────────────────
echo "Building Redis ${REDIS_VERSION} (this takes ~1 minute)..."
mkdir -p "$VENDOR_DIR"
tar -xzf "$TMPDIR_DL/redis.tar.gz" --strip-components=1 -C "$VENDOR_DIR"

# Build with minimal output; use available cores
NPROC=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 2)
make -C "$VENDOR_DIR" -j"$NPROC" redis-server redis-cli > "$TMPDIR_DL/build.log" 2>&1 || {
    echo "ERROR: Redis build failed. Build log:"
    tail -30 "$TMPDIR_DL/build.log"
    return 1 2>/dev/null || exit 1
}

REDIS_BIN="$VENDOR_DIR/src/redis-server"
if [[ ! -x "$REDIS_BIN" ]]; then
    echo "ERROR: Build succeeded but redis-server binary not found at $REDIS_BIN"
    return 1 2>/dev/null || exit 1
fi

echo "Redis ${REDIS_VERSION} built successfully: $REDIS_BIN"
export REDIS_BIN
