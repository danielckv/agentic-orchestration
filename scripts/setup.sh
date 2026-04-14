#!/usr/bin/env bash
# CAOF Setup Script — run this after extracting the release archive.
#
# This script:
#   1. Detects platform and copies the correct binary
#   2. Sets up the MCP server Python venv
#   3. Optionally installs the systemd service
#   4. Creates config directories
#
# Usage:
#   ./setup.sh                  # Interactive setup
#   ./setup.sh --no-service     # Skip systemd service install
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="${CAOF_INSTALL_DIR:-/opt/caof}"
SKIP_SERVICE=false

for arg in "$@"; do
    case "$arg" in
        --no-service) SKIP_SERVICE=true ;;
    esac
done

# ── Detect platform ──────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY_NAME="caof-${OS}-${ARCH}"
if [[ ! -f "$SCRIPT_DIR/bin/$BINARY_NAME" ]]; then
    echo "ERROR: Binary not found: bin/$BINARY_NAME"
    echo "Available binaries:"
    ls "$SCRIPT_DIR/bin/" 2>/dev/null || echo "  (none)"
    exit 1
fi

echo "=== CAOF Setup ==="
echo "Platform: ${OS}/${ARCH}"
echo "Install dir: ${INSTALL_DIR}"
echo ""

# ── Ensure Redis is available ────────────────────────────────

if ! command -v redis-server &>/dev/null; then
    echo "Redis not found — installing from source..."
    VENDOR_DIR="$INSTALL_DIR/vendor/redis" source "$SCRIPT_DIR/install-redis.sh"
    # Symlink vendor redis-server so it's on PATH
    sudo ln -sf "$REDIS_BIN" /usr/local/bin/redis-server
    sudo ln -sf "${REDIS_BIN%server}cli" /usr/local/bin/redis-cli
    echo "  Linked: /usr/local/bin/redis-server -> $REDIS_BIN"
else
    echo "Redis found: $(redis-server --version 2>&1 | head -1)"
fi
echo ""

# ── Install binary ───────────────────────────────────────────

echo "Installing caof binary..."
sudo mkdir -p "$INSTALL_DIR/bin"
sudo cp "$SCRIPT_DIR/bin/$BINARY_NAME" "$INSTALL_DIR/bin/caof"
sudo chmod +x "$INSTALL_DIR/bin/caof"

# Symlink to /usr/local/bin if not already there
if [[ ! -f /usr/local/bin/caof ]]; then
    sudo ln -sf "$INSTALL_DIR/bin/caof" /usr/local/bin/caof
    echo "  Linked: /usr/local/bin/caof -> $INSTALL_DIR/bin/caof"
fi

echo "  Binary installed: $INSTALL_DIR/bin/caof"
"$INSTALL_DIR/bin/caof" version || true

# ── Install config ───────────────────────────────────────────

echo ""
echo "Installing config..."
sudo mkdir -p "$INSTALL_DIR/config"
sudo cp -r "$SCRIPT_DIR/config/"* "$INSTALL_DIR/config/"
echo "  Config installed: $INSTALL_DIR/config/"

# ── Install MCP server ───────────────────────────────────────

echo ""
echo "Setting up MCP server..."
sudo mkdir -p "$INSTALL_DIR/mcp"
sudo cp -r "$SCRIPT_DIR/mcp/"* "$INSTALL_DIR/mcp/"
sudo cp "$SCRIPT_DIR/mcp/pyproject.toml" "$INSTALL_DIR/mcp/"

# Create venv and install deps
if command -v python3 &>/dev/null; then
    sudo python3 -m venv "$INSTALL_DIR/mcp/.venv"
    sudo "$INSTALL_DIR/mcp/.venv/bin/pip" install --upgrade pip -q
    sudo "$INSTALL_DIR/mcp/.venv/bin/pip" install -e "$INSTALL_DIR/mcp" -q
    echo "  MCP venv ready: $INSTALL_DIR/mcp/.venv/"
else
    echo "  WARNING: python3 not found — install Python 3.11+ and re-run"
fi

# ── Env file ─────────────────────────────────────────────────

echo ""
sudo mkdir -p /etc/caof
if [[ ! -f /etc/caof/mcp.env ]]; then
    sudo tee /etc/caof/mcp.env > /dev/null <<'EOF'
# CAOF MCP environment — add your API keys here
# CAOF_INFERENCE_PROVIDER=anthropic
# CAOF_INFERENCE_MODEL=claude-sonnet-4-6
# CAOF_INFERENCE_API_KEY=sk-ant-...
EOF
    echo "Created env template: /etc/caof/mcp.env"
    echo "  ** Edit this file with your API keys before starting the service **"
else
    echo "Env file already exists: /etc/caof/mcp.env"
fi

# ── systemd service ──────────────────────────────────────────

if [[ "$SKIP_SERVICE" == "false" ]] && command -v systemctl &>/dev/null; then
    echo ""
    echo "Installing systemd service..."

    # Write service unit pointing at the install dir
    sudo tee /etc/systemd/system/caof-mcp.service > /dev/null <<UNIT
[Unit]
Description=CAOF MCP Server
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR/mcp
ExecStart=$INSTALL_DIR/mcp/.venv/bin/python -m server
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
EnvironmentFile=-/etc/caof/mcp.env

NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/home /root /tmp $INSTALL_DIR
PrivateTmp=true

[Install]
WantedBy=multi-user.target
UNIT

    sudo systemctl daemon-reload
    echo "  Service installed: caof-mcp.service"
    echo ""
    echo "To start:"
    echo "  sudo systemctl enable --now caof-mcp"
    echo "  sudo journalctl -u caof-mcp -f"
else
    if [[ "$SKIP_SERVICE" == "true" ]]; then
        echo ""
        echo "Skipped systemd service install (--no-service)."
    else
        echo ""
        echo "systemd not available — run MCP manually:"
        echo "  cd $INSTALL_DIR/mcp && .venv/bin/python -m server"
    fi
fi

# ── Done ─────────────────────────────────────────────────────

echo ""
echo "=== Setup complete ==="
echo ""
echo "  caof binary:  $INSTALL_DIR/bin/caof  (also /usr/local/bin/caof)"
echo "  config:       $INSTALL_DIR/config/"
echo "  MCP server:   $INSTALL_DIR/mcp/"
echo "  task output:  ~/caof-tasks/"
echo "  env file:     /etc/caof/mcp.env"
echo ""
echo "Quick start:"
echo "  1. Edit /etc/caof/mcp.env with your API keys"
echo "  2. sudo systemctl enable --now caof-mcp"
echo "  3. caof version"
