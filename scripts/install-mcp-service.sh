#!/usr/bin/env bash
# Install the CAOF MCP server as a systemd service.
#
# Usage:
#   System-wide:  sudo ./scripts/install-mcp-service.sh --system
#   Per-user:     ./scripts/install-mcp-service.sh [--user]
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
MCP_DIR="$PROJECT_DIR/mcp"

MODE="${1:---user}"

# ── Helper ───────────────────────────────────────────────────

setup_venv() {
    local target="$1"
    echo "Setting up Python venv at $target/.venv ..."
    python3 -m venv "$target/.venv"
    "$target/.venv/bin/pip" install --upgrade pip -q
    "$target/.venv/bin/pip" install -e "$target" -q
    echo "Done."
}

create_env_file() {
    local env_file="$1"
    if [[ -f "$env_file" ]]; then
        echo "Env file already exists: $env_file"
        return
    fi
    mkdir -p "$(dirname "$env_file")"
    cat > "$env_file" <<'EOF'
# CAOF MCP environment — add your API keys here
# CAOF_INFERENCE_PROVIDER=anthropic
# CAOF_INFERENCE_MODEL=claude-sonnet-4-6
# CAOF_INFERENCE_API_KEY=sk-ant-...
EOF
    echo "Created env template: $env_file"
}

# ── System-wide install ──────────────────────────────────────

install_system() {
    echo "Installing system-wide service..."
    sudo mkdir -p /opt/caof/mcp
    sudo cp -r "$MCP_DIR"/* /opt/caof/mcp/
    sudo cp "$MCP_DIR/pyproject.toml" /opt/caof/mcp/
    setup_venv /opt/caof/mcp

    sudo cp "$PROJECT_DIR/systemd/caof-mcp.service" /etc/systemd/system/
    sudo mkdir -p /etc/caof
    create_env_file /etc/caof/mcp.env

    sudo systemctl daemon-reload
    echo ""
    echo "Installed. Next steps:"
    echo "  1. Edit /etc/caof/mcp.env with your API keys"
    echo "  2. sudo systemctl enable --now caof-mcp"
    echo "  3. sudo journalctl -u caof-mcp -f"
}

# ── Per-user install ─────────────────────────────────────────

install_user() {
    local dest="$HOME/caof-mcp"
    echo "Installing per-user service..."

    mkdir -p "$dest"
    cp -r "$MCP_DIR"/* "$dest/"
    cp "$MCP_DIR/pyproject.toml" "$dest/"
    # Also copy config dir so inference.py can find provider configs
    cp -r "$PROJECT_DIR/config" "$dest/"
    setup_venv "$dest"

    mkdir -p "$HOME/.config/systemd/user"
    # Write a concrete user unit (not a template)
    cat > "$HOME/.config/systemd/user/caof-mcp.service" <<UNIT
[Unit]
Description=CAOF MCP Server
After=network.target

[Service]
Type=simple
WorkingDirectory=$dest
ExecStart=$dest/.venv/bin/python -m server
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
EnvironmentFile=-$HOME/.config/caof/mcp.env

[Install]
WantedBy=default.target
UNIT

    create_env_file "$HOME/.config/caof/mcp.env"

    systemctl --user daemon-reload
    echo ""
    echo "Installed. Next steps:"
    echo "  1. Edit ~/.config/caof/mcp.env with your API keys"
    echo "  2. systemctl --user enable --now caof-mcp"
    echo "  3. journalctl --user -u caof-mcp -f"
    echo ""
    echo "Task output goes to: ~/caof-tasks/"
}

# ── Main ─────────────────────────────────────────────────────

case "$MODE" in
    --system) install_system ;;
    --user)   install_user ;;
    *)
        echo "Usage: $0 [--user|--system]"
        exit 1
        ;;
esac
