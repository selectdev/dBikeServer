#!/usr/bin/env bash
# install.sh — dBike Server Enterprise Service Manager
#
# Usage: ./install.sh <command> [options]
#
# Commands:
#   install    Install dBike Server as a boot service (default)
#   uninstall  Remove all services and configurations
#   start      Start the service
#   stop       Stop the service
#   restart    Restart the service
#   status     Show detailed service status
#   logs       Stream live service logs
#   build      Compile the dBike Server binary
#   upgrade    Build and hot-restart the service
#   doctor     Run system diagnostic checks
#   help       Show this help message
#
# Install options:
#   --data DIR    Custom data directory (default: ./data)
#   --no-panel    Skip admin panel installation

set -euo pipefail

# ── Identity ──────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="dbikeserver"
BINARY_PATH="${SCRIPT_DIR}/${BINARY_NAME}"
PANEL_BINARY="${SCRIPT_DIR}/dbikeserver-panel"
LAUNCH_PANEL="${SCRIPT_DIR}/launch-panel.sh"
DATA_DIR="${SCRIPT_DIR}/data"
SERVICE_NAME="dbikeserver"
INSTALL_PANEL=true
SYSTEMD_UNIT_DIR="/etc/systemd/system"
LOG_DIR="/var/log/${SERVICE_NAME}"
VERSION="$(git -C "$SCRIPT_DIR" describe --tags --always 2>/dev/null || echo "dev")"

# ── ANSI ──────────────────────────────────────────────────────────────────────
ESC=$'\033'
RESET="${ESC}[0m"
BOLD="${ESC}[1m"
DIM="${ESC}[2m"
FG_RED="${ESC}[91m"
FG_GREEN="${ESC}[92m"
FG_YELLOW="${ESC}[93m"
FG_CYAN="${ESC}[96m"
FG_WHITE="${ESC}[97m"
FG_GRAY="${ESC}[90m"

# ── Output helpers ─────────────────────────────────────────────────────────────
info()  { printf "${FG_CYAN}  ·${RESET}  %s\n"       "$*"; }
ok()    { printf "${FG_GREEN}  ✔${RESET}  %s\n"       "$*"; }
warn()  { printf "${FG_YELLOW}  !${RESET}  %s\n"      "$*"; }
die()   { printf "${FG_RED}  ✖${RESET}  %s\n" "$*" >&2; exit 1; }
header(){ printf "${BOLD}${FG_WHITE}  %s${RESET}\n"   "$*"; }
sep()   { printf "${FG_GRAY}  %s${RESET}\n" "$(printf '─%.0s' $(seq 1 60))"; }
check_pass() { printf "  ${FG_GREEN}✔${RESET}  %-40s ${FG_GREEN}pass${RESET}\n" "$*"; }
check_warn() { printf "  ${FG_YELLOW}!${RESET}  %-40s ${FG_YELLOW}warn${RESET}\n" "$*"; }
check_fail() { printf "  ${FG_RED}✖${RESET}  %-40s ${FG_RED}fail${RESET}\n" "$*"; }

banner() {
    printf "\n"
    printf "${BOLD}${FG_CYAN}"
    printf "  ╔══════════════════════════════════════════════╗\n"
    printf "  ║          dBike Server  %-18s  ║\n" "${VERSION}"
    printf "  ╚══════════════════════════════════════════════╝\n"
    printf "${RESET}\n"
}

# ── PATH extension ────────────────────────────────────────────────────────────
# When invoked from a non-interactive subprocess (e.g. the panel TUI), PATH may
# omit Go's bin directory. Append common installation locations so 'go' is found.
for _godir in /usr/local/go/bin "$HOME/go/bin" /opt/homebrew/bin /usr/local/bin; do
    [[ ":$PATH:" != *":${_godir}:"* ]] && [[ -d "$_godir" ]] && PATH="${PATH}:${_godir}"
done
export PATH

# ── OS / user detection ───────────────────────────────────────────────────────
detect_os() {
    case "$(uname)" in
        Darwin) echo "macos" ;;
        Linux)  echo "linux" ;;
        *)      die "Unsupported OS: $(uname)" ;;
    esac
}

need_root() {
    [[ "$(id -u)" -eq 0 ]] || die "This command requires root. Run with sudo."
}

real_user() { echo "${SUDO_USER:-${USER:-root}}"; }
real_home() { eval echo "~$(real_user)"; }

OS=$(detect_os)

# ── Argument parsing ──────────────────────────────────────────────────────────
CMD="${1:-install}"
shift || true

while [[ $# -gt 0 ]]; do
    case "$1" in
        --data)      DATA_DIR="${2:?--data requires a directory}"; shift ;;
        --no-panel)  INSTALL_PANEL=false ;;
        *)           ;;
    esac
    shift
done

# ── Build ─────────────────────────────────────────────────────────────────────
cmd_build() {
    banner
    command -v go &>/dev/null || die "go not found. Install Go or pre-build the binaries."

    info "Building dBike Server binary…"
    if (cd "$SCRIPT_DIR" && go build -ldflags "-X main.Version=${VERSION}" -o "$BINARY_NAME" .); then
        ok "Server binary built: ${BINARY_PATH}"
    else
        warn "Server binary build failed — skipping."
    fi

    info "Building admin panel binary…"
    if (cd "$SCRIPT_DIR" && go build -o dbikeserver-panel ./panel/); then
        ok "Panel binary built: ${PANEL_BINARY}"
    else
        warn "Panel build failed."
    fi

    info "Building launcher binary…"
    if [[ "$OS" == "macos" ]]; then
        if (cd "$SCRIPT_DIR" && go build -o dbikeserver-launcher ./launcher/); then
            ok "Launcher binary built: ${SCRIPT_DIR}/dbikeserver-launcher"
        else
            warn "Launcher build failed."
        fi
    fi
}

ensure_binary() {
    if [[ ! -x "$BINARY_PATH" ]]; then
        info "Binary not found — building…"
        command -v go &>/dev/null || die "go not found. Build the binary manually first."
        (cd "$SCRIPT_DIR" && go build -o "$BINARY_NAME" .) || die "Build failed."
        ok "Binary built."
    fi
}

ensure_panel() {
    if [[ ! -x "$PANEL_BINARY" ]]; then
        info "Panel binary not found — building…"
        command -v go &>/dev/null || die "go not found. Run: make panel"
        (cd "$SCRIPT_DIR" && go build -o dbikeserver-panel ./panel/) || die "Panel build failed."
        ok "Panel binary built."
    fi
}

# ── Linux / systemd ───────────────────────────────────────────────────────────
install_systemd() {
    info "Installing systemd service…"
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
    ok "Service installed and started (systemd)."
}

install_panel_tty1() {
    [[ -x "$PANEL_BINARY" ]] || { warn "dbikeserver-panel not found; skipping TTY1 panel."; return 0; }
    info "Configuring admin panel on TTY1…"
    systemctl mask getty@tty1.service 2>/dev/null || true

    cat > "${SYSTEMD_UNIT_DIR}/${SERVICE_NAME}-panel.service" <<EOF
[Unit]
Description=dBike Admin Panel (TTY1)
After=${SERVICE_NAME}.service
Wants=${SERVICE_NAME}.service

[Service]
Type=simple
ExecStart=${PANEL_BINARY} --watch
StandardInput=tty
StandardOutput=tty
TTYPath=/dev/tty1
TTYReset=yes
TTYVHangup=yes
TTYVTDisallocate=yes
Restart=always
RestartSec=3
Environment=TERM=linux

[Install]
WantedBy=multi-user.target graphical.target
EOF
    systemctl daemon-reload
    systemctl enable --now "${SERVICE_NAME}-panel"
    ok "Admin panel enabled on TTY1."
}

install_panel_desktop() {
    [[ -x "$PANEL_BINARY" ]] || { warn "dbikeserver-panel not found; skipping desktop autostart."; return 0; }
    local autostart_dir
    autostart_dir="$(real_home)/.config/autostart"
    mkdir -p "$autostart_dir"

    local term_exec=""
    if   command -v lxterminal    &>/dev/null; then
        term_exec="lxterminal --fullscreen -e ${PANEL_BINARY} --watch"
    elif command -v xfce4-terminal &>/dev/null; then
        term_exec="xfce4-terminal --fullscreen -e '${PANEL_BINARY} --watch'"
    elif command -v xterm          &>/dev/null; then
        term_exec="xterm -fullscreen -e ${PANEL_BINARY} --watch"
    elif command -v gnome-terminal &>/dev/null; then
        term_exec="gnome-terminal --full-screen -- ${PANEL_BINARY} --watch"
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
    ok "Desktop autostart configured for $(real_user)."
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
    [[ -f "$desktop_entry" ]] && rm -f "$desktop_entry" && ok "Removed desktop autostart entry."
    systemctl daemon-reload
}

svc_is_active_linux() {
    systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null
}

start_linux()   { systemctl start   "$SERVICE_NAME"; ok "Service started."; }
stop_linux()    { systemctl stop    "$SERVICE_NAME"; ok "Service stopped."; }
restart_linux() { systemctl restart "$SERVICE_NAME"; ok "Service restarted."; }

status_linux() {
    local state pid uptime restarts
    state=$(systemctl is-active "$SERVICE_NAME" 2>/dev/null || echo "inactive")
    pid=$(systemctl show "$SERVICE_NAME" --property=MainPID 2>/dev/null \
        | awk -F= '{print $2}' | grep -v '^0$' || echo "—")
    restarts=$(systemctl show "$SERVICE_NAME" --property=NRestarts 2>/dev/null \
        | awk -F= '{print $2}' || echo "—")

    sep
    header "Service Status"
    sep
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n" "Service"  "$SERVICE_NAME"
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n" "State" \
        "$( [[ "$state" == "active" ]] && echo "${FG_GREEN}● ${state}${RESET}" \
            || echo "${FG_YELLOW}○ ${state}${RESET}" )"
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n" "PID"       "${pid}"
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n" "Restarts"  "${restarts}"
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n" "Data dir"  "${DATA_DIR}"
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n" "Binary"    "${BINARY_PATH}"
    sep
    echo ""
    systemctl status "$SERVICE_NAME" --no-pager 2>/dev/null || true
}

logs_linux() {
    info "Streaming logs (Ctrl+C to stop)…"
    exec journalctl -f -u "$SERVICE_NAME" --output=short-iso
}

# ── macOS / launchd ───────────────────────────────────────────────────────────
install_launchd_daemon() {
    info "Installing LaunchDaemon…"
    mkdir -p "$LOG_DIR"
    local plist="/Library/LaunchDaemons/${SERVICE_NAME}.plist"

    cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-
  "http:
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

install_panel_launchd() {
    [[ -x "$LAUNCH_PANEL" ]] || { warn "launch-panel.sh not found; skipping panel."; return 0; }
    local run_user run_home agent_dir agent_plist
    run_user=$(real_user)
    run_home=$(real_home)
    agent_dir="${run_home}/Library/LaunchAgents"
    agent_plist="${agent_dir}/com.${SERVICE_NAME}.panel.plist"

    info "Installing panel LaunchAgent for user ${run_user}…"
    mkdir -p "$agent_dir"

    cat > "$agent_plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-
  "http:
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.${SERVICE_NAME}.panel</string>

    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>${LAUNCH_PANEL}</string>
    </array>

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
    sudo -u "$run_user" launchctl load -w "$agent_plist" 2>/dev/null \
        || warn "Could not load agent now — will start on next login."
    ok "Panel LaunchAgent installed for ${run_user}."
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

svc_is_active_macos() {
    launchctl list 2>/dev/null | awk "\$3 == \"${SERVICE_NAME}\" && \$1 != \"-\"" | grep -q .
}

start_macos() {
    local plist="/Library/LaunchDaemons/${SERVICE_NAME}.plist"
    [[ -f "$plist" ]] || die "Service not installed. Run: ./install.sh install"
    launchctl load -w "$plist" 2>/dev/null \
        || launchctl bootstrap system "$plist" 2>/dev/null \
        || die "Failed to start service."
    ok "Service started."
}

stop_macos() {
    local plist="/Library/LaunchDaemons/${SERVICE_NAME}.plist"
    [[ -f "$plist" ]] || die "Service not installed."
    launchctl unload -w "$plist" 2>/dev/null \
        || launchctl bootout system "$plist" 2>/dev/null \
        || die "Failed to stop service."
    ok "Service stopped."
}

restart_macos() {
    stop_macos
    sleep 1
    start_macos
    ok "Service restarted."
}

status_macos() {
    local row pid state
    row=$(launchctl list 2>/dev/null | awk "\$3 == \"${SERVICE_NAME}\"")
    pid=$(echo "$row" | awk '{print $1}')
    [[ "$pid" == "-" || -z "$pid" ]] && state="${FG_YELLOW}○ stopped${RESET}" \
        || state="${FG_GREEN}● running (PID ${pid})${RESET}"

    sep
    header "Service Status"
    sep
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n"  "Service"  "$SERVICE_NAME"
    printf "  ${FG_GRAY}%-16s${RESET}  "       "State"
    printf "${state}\n"
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n"  "Data dir"  "${DATA_DIR}"
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n"  "Binary"    "${BINARY_PATH}"
    printf "  ${FG_GRAY}%-16s${RESET}  %s\n"  "Logs"      "${LOG_DIR}/"
    sep
}

logs_macos() {
    local logfile="${LOG_DIR}/stdout.log"
    if [[ -f "$logfile" ]]; then
        info "Streaming logs from ${logfile} (Ctrl+C to stop)…"
        exec tail -f "$logfile"
    else
        warn "Log file not found: ${logfile}"
        warn "Is the service installed? Run: ./install.sh install"
    fi
}

# ── Doctor ────────────────────────────────────────────────────────────────────
cmd_doctor() {
    banner
    header "System Diagnostic Report"
    sep

    local fail=0

    _check() {
        local label="$1"; shift
        if "$@" &>/dev/null; then
            check_pass "$label"
        else
            check_fail "$label"
            fail=1
        fi
    }

    _check_warn() {
        local label="$1"; shift
        if "$@" &>/dev/null; then
            check_pass "$label"
        else
            check_warn "$label"
        fi
    }

    # 1. OS
    case "$(uname)" in Darwin|Linux) check_pass "OS is supported ($(uname))" ;;
        *) check_fail "OS is supported"; fail=1 ;;
    esac

    # 2. Required tools
    _check_warn "bash >= 4" bash -c '[[ "${BASH_VERSINFO[0]}" -ge 4 ]]'
    _check_warn "tput available"   command -v tput
    _check_warn "git available"    command -v git
    _check_warn "Go compiler"      command -v go

    # 3. Bluetooth hardware
    if [[ "$OS" == "linux" ]]; then
        _check_warn "Bluetooth hardware" sh -c 'ls /sys/class/bluetooth/ | grep -q .'
    else
        _check_warn "Bluetooth hardware" sh -c \
            'system_profiler SPBluetoothDataType 2>/dev/null | grep -q "Bluetooth"'
    fi

    # 4. Binaries
    _check "Server binary exists (${BINARY_NAME})"      test -f "$BINARY_PATH"
    _check "Server binary is executable"                test -x "$BINARY_PATH"
    _check_warn "Panel binary exists (dbikeserver-panel)" test -x "$PANEL_BINARY"

    # 5. Directories
    mkdir -p "$DATA_DIR" 2>/dev/null || true
    _check "Data dir writable (${DATA_DIR})"   test -w "$DATA_DIR"
    _check_warn "Scripts dir exists"           test -d "${SCRIPT_DIR}/scripts"

    # 6. Service installation
    if [[ "$OS" == "linux" ]]; then
        _check_warn "Systemd unit installed" \
            test -f "${SYSTEMD_UNIT_DIR}/${SERVICE_NAME}.service"
        _check_warn "Service is active" systemctl is-active --quiet "$SERVICE_NAME"
    else
        _check_warn "LaunchDaemon installed" \
            test -f "/Library/LaunchDaemons/${SERVICE_NAME}.plist"
        _check_warn "Service is active" svc_is_active_macos
    fi

    # 7. Disk space (require 100 MB free)
    local free_kb
    if [[ "$OS" == "linux" ]]; then
        free_kb=$(df "$SCRIPT_DIR" | awk 'NR==2{print $4}')
    else
        free_kb=$(df "$SCRIPT_DIR" | awk 'NR==2{print $4}')
    fi
    if [[ "${free_kb:-0}" -gt 102400 ]]; then
        check_pass "Disk space > 100 MB free"
    else
        check_warn "Disk space > 100 MB free (only $(( ${free_kb:-0} / 1024 )) MB)"
    fi

    sep
    if [[ "$fail" -eq 0 ]]; then
        ok "All critical checks passed."
    else
        warn "One or more critical checks failed. Review the output above."
    fi
    echo ""
}

# ── Install / Uninstall ────────────────────────────────────────────────────────
cmd_install() {
    need_root
    banner
    info "OS: ${OS}  |  Directory: ${SCRIPT_DIR}  |  User: $(real_user)"
    echo ""

    ensure_binary
    mkdir -p "$DATA_DIR"

    case "$OS" in
        linux)
            install_systemd
            if $INSTALL_PANEL; then
                ensure_panel
                install_panel_tty1
                install_panel_desktop
            fi
            ;;
        macos)
            install_launchd_daemon
            if $INSTALL_PANEL; then
                mkdir -p "$LOG_DIR"
                install_panel_launchd
            fi
            ;;
    esac

    echo ""
    ok "Installation complete."
    info "dBike Server starts automatically on boot."
    $INSTALL_PANEL && info "Admin panel opens fullscreen on boot/login."
}

cmd_uninstall() {
    need_root
    banner
    info "Removing dBike Server services…"
    case "$OS" in
        linux) remove_systemd ;;
        macos) remove_launchd ;;
    esac
    ok "All dBike Server services removed."
    echo ""
}

# ── Service control ────────────────────────────────────────────────────────────
cmd_start() {
    need_root
    banner
    case "$OS" in
        linux) start_linux   ;;
        macos) start_macos   ;;
    esac
}

cmd_stop() {
    need_root
    banner
    case "$OS" in
        linux) stop_linux    ;;
        macos) stop_macos    ;;
    esac
}

cmd_restart() {
    need_root
    banner
    case "$OS" in
        linux) restart_linux ;;
        macos) restart_macos ;;
    esac
}

cmd_status() {
    banner
    case "$OS" in
        linux) status_linux  ;;
        macos) status_macos  ;;
    esac
}

cmd_logs() {
    banner
    case "$OS" in
        linux) logs_linux    ;;
        macos) logs_macos    ;;
    esac
}

# ── Upgrade ────────────────────────────────────────────────────────────────────
cmd_upgrade() {
    banner
    command -v go &>/dev/null || die "go not found."

    local server_ok=false
    info "Building dBike Server binary…"
    if (cd "$SCRIPT_DIR" && go build -ldflags "-X main.Version=${VERSION}" -o "$BINARY_NAME" .); then
        ok "Server binary built."
        server_ok=true
    else
        warn "Server binary build failed — will not restart service."
    fi

    info "Building admin panel binary…"
    if (cd "$SCRIPT_DIR" && go build -o dbikeserver-panel ./panel/); then
        ok "Panel binary built."
    else
        warn "Panel build failed."
    fi

    if $server_ok; then
        info "Restarting service…"
        case "$OS" in
            linux)
                if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
                    need_root; restart_linux
                elif pgrep -x "$BINARY_NAME" &>/dev/null; then
                    pkill -x "$BINARY_NAME" || true
                    sleep 1
                    "$BINARY_PATH" &
                    ok "Server restarted (direct process)."
                else
                    info "Service not running — skipping restart."
                fi
                ;;
            macos)
                if svc_is_active_macos; then
                    need_root; restart_macos
                elif pgrep -x "$BINARY_NAME" &>/dev/null; then
                    pkill -x "$BINARY_NAME" || true
                    sleep 1
                    "$BINARY_PATH" &
                    ok "Server restarted (direct process)."
                else
                    info "Service not running — skipping restart."
                fi
                ;;
        esac
    fi

    ok "Upgrade complete."
}

# ── Help ──────────────────────────────────────────────────────────────────────
cmd_help() {
    banner
    printf "${BOLD}${FG_WHITE}  Usage:${RESET}  ./install.sh <command> [options]\n\n"

    printf "${BOLD}${FG_CYAN}  Commands:${RESET}\n"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "install"   "Install dBike Server as a boot service"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "uninstall" "Remove all services and configurations"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "start"     "Start the service"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "stop"      "Stop the service"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "restart"   "Restart the service"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "status"    "Show detailed service status"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "logs"      "Stream live service logs"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "build"     "Compile the dBike Server binary"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "upgrade"   "Build and hot-restart the service"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "doctor"    "Run system diagnostic checks"
    printf "  ${FG_GREEN}%-12s${RESET}  %s\n" "help"      "Show this help message"

    printf "\n${BOLD}${FG_CYAN}  Install options:${RESET}\n"
    printf "  ${FG_YELLOW}%-16s${RESET}  %s\n" "--data <dir>"   "Custom data directory (default: ./data)"
    printf "  ${FG_YELLOW}%-16s${RESET}  %s\n" "--no-panel"     "Skip admin panel installation"

    printf "\n${BOLD}${FG_CYAN}  Examples:${RESET}\n"
    printf "  ${FG_GRAY}sudo ./install.sh install${RESET}\n"
    printf "  ${FG_GRAY}sudo ./install.sh install --data /mnt/data --no-panel${RESET}\n"
    printf "  ${FG_GRAY}sudo ./install.sh restart${RESET}\n"
    printf "  ${FG_GRAY}     ./install.sh doctor${RESET}\n"
    printf "  ${FG_GRAY}     ./install.sh logs${RESET}\n"
    printf "\n"
}

# ── Dispatch ──────────────────────────────────────────────────────────────────
case "$CMD" in
    install)           cmd_install   ;;
    uninstall|remove)  cmd_uninstall ;;
    start)             cmd_start     ;;
    stop)              cmd_stop      ;;
    restart)           cmd_restart   ;;
    status)            cmd_status    ;;
    logs)              cmd_logs      ;;
    build)             cmd_build     ;;
    upgrade)           cmd_upgrade   ;;
    doctor)            cmd_doctor    ;;
    help|--help|-h)    cmd_help      ;;
    *)
        printf "${FG_RED}  Unknown command: %s${RESET}\n\n" "$CMD" >&2
        cmd_help
        exit 2
        ;;
esac
