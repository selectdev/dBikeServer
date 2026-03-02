#!/usr/bin/env bash
# launch-panel.sh — macOS admin panel launcher (thin wrapper).
# Opens dbikeserver-panel in a fullscreen terminal window via dbikeserver-launcher.
# Called by the com.dbikeserver.panel LaunchAgent on login.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LAUNCHER="${SCRIPT_DIR}/dbikeserver-launcher"

if [[ ! -x "$LAUNCHER" ]]; then
    printf "Error: dbikeserver-launcher not found.\nRun: ./install.sh build\n" >&2
    exit 1
fi

exec "$LAUNCHER"
