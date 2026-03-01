#!/usr/bin/env bash
# install.sh — dBike Server boot installer
#
# Supports:
#   • Linux  + systemd  — headless (TTY1 fullscreen) and desktop (autostart)
#   • macOS  + launchd  — fullscreen Terminal.app window on login
#
# Usage:
#   sudo ./install.sh           — install & enable boot service + admin panel
#   sudo ./install.sh --remove  — remove all installed services
#   sudo ./install.sh --status  — show service status

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="dbikeserver"
BINARY_PATH="${SCRIPT_DIR}/${BINARY_NAME}"
DATA_DIR="${SCRIPT_DIR}/data"
SERVICE_NAME="dbikeserver"
ADMIN_PANEL="${SCRIPT_DIR}/admin-panel.sh"
LAUNCH_PANEL="${SCRIPT_DIR}/launch-panel.sh"

# ── Helpers ───────────────────────────────────────────────────────────────────
info()  { printf "\033[96m  •\033[0m  %s\n" "$*"; }
ok()    { printf "\033[92m  ✔\033[0m  %s\n" "$*"; }
warn()  { printf "\033[93m  !\033[0m  %s\n" "$*"; }
die()   { printf "\033[91m  ✖\033[0m  %s\n" "$*" >&2; exit 1; }

detect_os() {
    case "$(uname)" in
        Darwin) echo "macos" ;;
        Linux)  echo "linux" ;;
        *)      die "Unsupported OS: $(uname)" ;;
    esac
}

need_root() {
    [[ "$(id -u)" -eq 0 ]] || die "Run this script with sudo."
}

# The real logged-in user (not root) and their home directory.
real_user() { echo "${SUDO_USER:-${USER:-root}}"; }
real_home() { eval echo "~$(real_user)"; }

# ── Build ─────────────────────────────────────────────────────────────────────
build_binary() {
    if [[ ! -x "$BINARY_PATH" ]]; then
        info "Binary not found — building…"
        command -v go &>/dev/null || die "go not found; install Go or build manually first."
        (cd "$SCRIPT_DIR" && go build -o "$BINARY_NAME" .) || die "Build failed."
        ok "Binary built: $BINARY_PATH"
    else
        ok "Binary found: $BINARY_PATH"
    fi
}

# ── Linux / systemd ───────────────────────────────────────────────────────────
SYSTEMD_UNIT_DIR="/etc/systemd/system"

install_systemd() {
    info "Installing dbikeserver systemd service…"
    local run_user run_group
    run_user=$(real_user)
    run_group=$(id -gn "$run_user" 2>/dev/null || echo "$run_user")

    cat > "${SYSTEMD_UNIT_DIR}/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=dBike BLE IPC Server
After=network.target bluetooth.target
Wants=bluetooth.target

[Service]
Type=simple
User=${run_user}
Group=${run_group}
WorkingDirectory=${SCRIPT_DIR}
ExecStart=${BINARY_PATH} --db ${DATA_DIR}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${SERVICE_NAME}

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable --now "$SERVICE_NAME"
    ok "dbikeserver service enabled and started."
}

# Fullscreen admin panel on TTY1 (headless / embedded devices).
# Masks getty@tty1 so the login prompt never appears there.
install_panel_tty1() {
    [[ -x "$ADMIN_PANEL" ]] || return 0
    local unit="${SYSTEMD_UNIT_DIR}/${SERVICE_NAME}-panel.service"
    info "Configuring fullscreen admin panel on TTY1…"

    # Prevent the getty login prompt from competing with our panel.
    systemctl mask getty@tty1.service 2>/dev/null || true

    cat > "$unit" <<EOF
[Unit]
Description=dBike Admin Panel (TTY1 fullscreen)
After=${SERVICE_NAME}.service
Wants=${SERVICE_NAME}.service

[Service]
Type=simple
ExecStart=/bin/bash ${ADMIN_PANEL} --watch
StandardInput=tty
StandardOutput=tty
TTYPath=/dev/tty1
TTYReset=yes
TTYVHangup=yes
TTYVTDisallocate=yes
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target graphical.target
EOF

    systemctl daemon-reload
    systemctl enable --now "${SERVICE_NAME}-panel"
    ok "Admin panel service enabled — fullscreen on TTY1."
}

# Autostart entry for Linux desktop environments (LXDE, GNOME, XFCE, etc.).
# Finds the best available fullscreen-capable terminal emulator.
install_panel_desktop() {
    [[ -x "$ADMIN_PANEL" ]] || return 0

    local autostart_dir
    autostart_dir="$(real_home)/.config/autostart"
    mkdir -p "$autostart_dir"

    # Find a terminal emulator that supports fullscreen.
    local term_exec=""
    if   command -v lxterminal    &>/dev/null; then
        term_exec="lxterminal --fullscreen -e bash ${ADMIN_PANEL} --watch"
    elif command -v xfce4-terminal &>/dev/null; then
        term_exec="xfce4-terminal --fullscreen -e 'bash ${ADMIN_PANEL} --watch'"
    elif command -v xterm          &>/dev/null; then
        term_exec="xterm -fullscreen -e bash ${ADMIN_PANEL} --watch"
    elif command -v gnome-terminal &>/dev/null; then
        term_exec="gnome-terminal --full-screen -- bash ${ADMIN_PANEL} --watch"
    else
        warn "No supported terminal emulator found; skipping desktop autostart."
        return 0
    fi

    cat > "${autostart_dir}/dbikeserver-panel.desktop" <<EOF
[Desktop Entry]
Name=dBike Admin Panel
Comment=Fullscreen dBike server status display
Exec=${term_exec}
Type=Application
X-GNOME-Autostart-enabled=true
X-GNOME-Autostart-Delay=3
EOF

    chown -R "$(real_user)" "$autostart_dir"
    ok "Desktop autostart entry installed for $(real_user)."
}

remove_systemd() {
    info "Removing systemd services…"
    systemctl unmask getty@tty1.service 2>/dev/null || true
    for svc in "${SERVICE_NAME}-panel" "${SERVICE_NAME}"; do
        if systemctl list-unit-files "${svc}.service" &>/dev/null; then
            systemctl disable --now "$svc" 2>/dev/null || true
            rm -f "${SYSTEMD_UNIT_DIR}/${svc}.service"
            ok "Removed ${svc}.service"
        fi
    done

    local desktop_entry
    desktop_entry="$(real_home)/.config/autostart/dbikeserver-panel.desktop"
    if [[ -f "$desktop_entry" ]]; then
        rm -f "$desktop_entry"
        ok "Removed desktop autostart entry."
    fi

    systemctl daemon-reload
}

status_systemd() {
    systemctl status "$SERVICE_NAME" --no-pager 2>/dev/null || true
}

# ── macOS / launchd ───────────────────────────────────────────────────────────
LOG_DIR="/var/log/${SERVICE_NAME}"

# System daemon — runs the BLE server in the background.
install_launchd_daemon() {
    info "Installing dbikeserver LaunchDaemon…"
    mkdir -p "$LOG_DIR"
    local plist="/Library/LaunchDaemons/${SERVICE_NAME}.plist"

    cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${SERVICE_NAME}</string>

    <key>ProgramArguments</key>
    <array>
        <string>${BINARY_PATH}</string>
        <string>--db</string>
        <string>${DATA_DIR}</string>
    </array>

    <key>WorkingDirectory</key>
    <string>${SCRIPT_DIR}</string>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>StandardOutPath</key>
    <string>${LOG_DIR}/stdout.log</string>

    <key>StandardErrorPath</key>
    <string>${LOG_DIR}/stderr.log</string>

    <key>ThrottleInterval</key>
    <integer>5</integer>
</dict>
</plist>
EOF

    launchctl load -w "$plist" 2>/dev/null \
        || launchctl bootstrap system "$plist" 2>/dev/null \
        || warn "Could not start daemon now — will start on next boot."
    ok "LaunchDaemon installed. Logs: ${LOG_DIR}/"
}

# User-level LaunchAgent — opens a fullscreen Terminal.app window on login.
install_panel_launchd() {
    [[ -x "$LAUNCH_PANEL" ]] || { warn "launch-panel.sh not found; skipping panel."; return 0; }

    local run_user run_home
    run_user=$(real_user)
    run_home=$(real_home)
    local agent_dir="${run_home}/Library/LaunchAgents"
    local agent_plist="${agent_dir}/com.${SERVICE_NAME}.panel.plist"

    info "Installing admin panel LaunchAgent for user ${run_user}…"
    mkdir -p "$agent_dir"

    cat > "$agent_plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.${SERVICE_NAME}.panel</string>

    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>${LAUNCH_PANEL}</string>
    </array>

    <!-- Wait a few seconds for the desktop to be ready -->
    <key>StartInterval</key>
    <integer>0</integer>

    <key>RunAtLoad</key>
    <true/>

    <key>StandardOutPath</key>
    <string>${LOG_DIR}/panel-stdout.log</string>

    <key>StandardErrorPath</key>
    <string>${LOG_DIR}/panel-stderr.log</string>
</dict>
</plist>
EOF

    chown "${run_user}" "$agent_plist"
    # Load as the real user, not root.
    sudo -u "$run_user" launchctl load -w "$agent_plist" 2>/dev/null \
        || warn "Could not load agent now — will start on next login."
    ok "Panel LaunchAgent installed for ${run_user} — fullscreen Terminal.app on login."
}

remove_launchd() {
    info "Removing launchd services…"

    local daemon_plist="/Library/LaunchDaemons/${SERVICE_NAME}.plist"
    if [[ -f "$daemon_plist" ]]; then
        launchctl unload -w "$daemon_plist" 2>/dev/null \
            || launchctl bootout system "$daemon_plist" 2>/dev/null || true
        rm -f "$daemon_plist"
        ok "Removed LaunchDaemon."
    fi

    local agent_plist
    agent_plist="$(real_home)/Library/LaunchAgents/com.${SERVICE_NAME}.panel.plist"
    if [[ -f "$agent_plist" ]]; then
        sudo -u "$(real_user)" launchctl unload -w "$agent_plist" 2>/dev/null || true
        rm -f "$agent_plist"
        ok "Removed panel LaunchAgent."
    fi
}

status_launchd() {
    launchctl list 2>/dev/null | grep "$SERVICE_NAME" \
        || echo "  (no dbikeserver services loaded)"
}

# ── Dispatch ──────────────────────────────────────────────────────────────────
OS=$(detect_os)
ACTION="${1:---install}"

case "$ACTION" in
    --remove)
        need_root
        info "Removing dBike Server services…"
        case "$OS" in
            linux) remove_systemd ;;
            macos) remove_launchd ;;
        esac
        ok "All dBike Server services removed."
        ;;

    --status)
        info "OS: $OS"
        echo ""
        case "$OS" in
            linux) status_systemd ;;
            macos) status_launchd ;;
        esac
        ;;

    --install|*)
        need_root
        info "Installing dBike Server…"
        info "OS: $OS  |  Directory: $SCRIPT_DIR  |  User: $(real_user)"
        echo ""

        build_binary

        case "$OS" in
            linux)
                install_systemd
                install_panel_tty1      # always — works on any Linux device
                install_panel_desktop   # additionally, if a desktop env is present
                ;;
            macos)
                install_launchd_daemon
                install_panel_launchd
                ;;
        esac

        echo ""
        ok "Installation complete."
        info "dBike Server starts automatically on boot."
        info "Admin panel opens fullscreen on boot/login."
        ;;
esac
