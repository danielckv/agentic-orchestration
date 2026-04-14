#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

PASS=0
FAIL=0

check() {
    local name="$1" cmd="$2" min_version="$3"
    if command -v "$cmd" &>/dev/null; then
        local version
        version=$("$cmd" --version 2>&1 | head -1)
        echo -e "${GREEN}✓${NC} $name found: $version (need $min_version+)"
        ((PASS++))
    else
        echo -e "${RED}✗${NC} $name not found (need $min_version+)"
        ((FAIL++))
    fi
}

echo "=== CAOF Dependency Check ==="
echo ""

check "Go"     go      "1.22"
check "Python"  python3 "3.11"
check "Git"     git     "2.40"
check "tmux"    tmux    "3.3"
check "Make"    make    "any"

# Redis: auto-install if missing
if command -v redis-server &>/dev/null; then
    version=$(redis-server --version 2>&1 | head -1)
    echo -e "${GREEN}✓${NC} Redis found: $version (need 7.0+)"
    ((PASS++))
else
    echo -e "${RED}✗${NC} Redis not found — attempting auto-install..."
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    if source "$SCRIPT_DIR/install-redis.sh"; then
        echo -e "${GREEN}✓${NC} Redis installed: $REDIS_BIN"
        ((PASS++))
    else
        echo -e "${RED}✗${NC} Redis auto-install failed (need 7.0+)"
        ((FAIL++))
    fi
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
    echo -e "${RED}Please install missing dependencies before proceeding.${NC}"
    exit 1
fi

echo -e "${GREEN}All dependencies satisfied.${NC}"
