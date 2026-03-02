#!/usr/bin/env bash
# admin-panel.sh — dBike Server admin panel display
# Shows a fullscreen status screen on boot or on demand.
# Usage: ./admin-panel.sh [--watch]  (--watch refreshes every 5s)

WATCH=false
[[ "${1:-}" == "--watch" ]] && WATCH=true

# ── Terminal setup ────────────────────────────────────────────────────────────
ESC=$'\033'
RESET="${ESC}[0m"
BOLD="${ESC}[1m"
DIM="${ESC}[2m"
FG_WHITE="${ESC}[97m"
FG_CYAN="${ESC}[96m"
FG_YELLOW="${ESC}[93m"
FG_GREEN="${ESC}[92m"
FG_GRAY="${ESC}[90m"
BG_BLACK="${ESC}[40m"

hide_cursor() { printf "${ESC}[?25l"; }
show_cursor() { printf "${ESC}[?25h"; }
clear_screen() { printf "${ESC}[2J${ESC}[H"; }
move_to()      { printf "${ESC}[$1;$2H"; }  # move_to row col

# ── System info helpers ───────────────────────────────────────────────────────
get_hardware_model() {
    if [[ "$(uname)" == "Darwin" ]]; then
        sysctl -n hw.model 2>/dev/null || echo "Unknown Mac"
    elif [[ -f /proc/device-tree/model ]]; then
        tr -d '\0' < /proc/device-tree/model 2>/dev/null || echo "Unknown"
    elif [[ -f /sys/firmware/devicetree/base/model ]]; then
        tr -d '\0' < /sys/firmware/devicetree/base/model 2>/dev/null || echo "Unknown"
    elif [[ -f /proc/cpuinfo ]]; then
        grep -m1 "Model" /proc/cpuinfo 2>/dev/null | awk -F': ' '{print $2}' || echo "Unknown"
    else
        uname -m
    fi
}

get_boot_time() {
    if [[ "$(uname)" == "Darwin" ]]; then
        # Parse kern.boottime: "{ sec = 1234567890, usec = 0 } Day Mon DD HH:MM:SS YYYY"
        sysctl -n kern.boottime 2>/dev/null \
            | sed 's/.*} 
            | xargs -I{} date -j -f "%a %b %d %T %Y" "{}" "+%Y-%m-%d %H:%M:%S" 2>/dev/null \
            || date
    elif [[ -f /proc/uptime ]]; then
        uptime_sec=$(awk '{print int($1)}' /proc/uptime)
        boot_epoch=$(( $(date +%s) - uptime_sec ))
        date -d "@${boot_epoch}" "+%Y-%m-%d %H:%M:%S" 2>/dev/null \
            || date -r "${boot_epoch}" "+%Y-%m-%d %H:%M:%S" 2>/dev/null \
            || echo "Unknown"
    else
        echo "Unknown"
    fi
}

get_uptime() {
    if [[ "$(uname)" == "Darwin" ]]; then
        uptime 2>/dev/null | sed 's/.*up /up /' | sed 's/, [0-9]* user.*
    else
        uptime -p 2>/dev/null || uptime 2>/dev/null | awk -F',' '{print $1}' | sed 's/.*up /up /'
    fi
}

get_build_info() {
    # Try git describe, then git short hash, then binary mod time
    local dir
    dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if git -C "$dir" rev-parse --short HEAD &>/dev/null; then
        local commit
        commit=$(git -C "$dir" rev-parse --short HEAD)
        local tag
        tag=$(git -C "$dir" describe --tags --always 2>/dev/null || echo "$commit")
        echo "$tag"
    elif [[ -x "$dir/dbikeserver" ]]; then
        echo "built $(date -r "$dir/dbikeserver" "+%Y-%m-%d" 2>/dev/null || echo "unknown")"
    else
        echo "dev"
    fi
}

get_service_status() {
    if systemctl is-active --quiet dbikeserver 2>/dev/null; then
        echo "${FG_GREEN}● running${RESET}"
    elif launchctl list 2>/dev/null | grep -q "dbikeserver"; then
        echo "${FG_GREEN}● running${RESET}"
    else
        echo "${FG_YELLOW}○ stopped${RESET}"
    fi
}

get_ip() {
    if [[ "$(uname)" == "Darwin" ]]; then
        ipconfig getifaddr en0 2>/dev/null \
            || ipconfig getifaddr en1 2>/dev/null \
            || echo "—"
    else
        hostname -I 2>/dev/null | awk '{print $1}' || echo "—"
    fi
}

# ── ASCII art icon ────────────────────────────────────────────────────────────
# A small bicycle rendered in ASCII
ICON_LINES=(
    "  o--o  "
    " /|  |\ "
    "o-+--+-o"
    "  |  |  "
    "  o  o  "
)

# ── Render ────────────────────────────────────────────────────────────────────
render() {
    local COLS ROWS
    COLS=$(tput cols 2>/dev/null || echo 80)
    ROWS=$(tput lines 2>/dev/null || echo 24)

    local hw build boot uptime_str ip svc_status
    hw=$(get_hardware_model)
    build=$(get_build_info)
    boot=$(get_boot_time)
    uptime_str=$(get_uptime)
    ip=$(get_ip)
    svc_status=$(get_service_status)

    clear_screen
    # Fill background black
    printf "${BG_BLACK}${FG_WHITE}"
    for (( r=1; r<=ROWS; r++ )); do
        move_to "$r" 1
        printf "%-${COLS}s" ""
    done

    # ── Top bar ──────────────────────────────────────────────────────────────
    move_to 1 1
    printf "${BG_BLACK}${BOLD}${FG_CYAN}%-${COLS}s${RESET}" ""

    # Admin Panel label (top-left)
    move_to 1 2
    printf "${BG_BLACK}${BOLD}${FG_CYAN} dBike  ADMIN PANEL${RESET}"

    # Current time (top-right)
    local now
    now=$(date "+%Y-%m-%d  %H:%M:%S")
    local time_col=$(( COLS - ${#now} - 2 ))
    move_to 1 "$time_col"
    printf "${BG_BLACK}${FG_GRAY}${now}${RESET}"

    # ── Separator ────────────────────────────────────────────────────────────
    move_to 2 1
    printf "${BG_BLACK}${FG_CYAN}"
    printf '%0.s─' $(seq 1 "$COLS")
    printf "${RESET}"

    # ── Bike icon (left, rows 4-8) ───────────────────────────────────────────
    local icon_start_row=4
    for i in "${!ICON_LINES[@]}"; do
        move_to $(( icon_start_row + i )) 4
        printf "${BG_BLACK}${FG_CYAN}${BOLD}%s${RESET}" "${ICON_LINES[$i]}"
    done

    move_to $(( icon_start_row + ${#ICON_LINES[@]} + 1 )) 4
    printf "${BG_BLACK}${FG_WHITE}${BOLD}  dBike Server${RESET}"

    # ── Stats panel (right side) ─────────────────────────────────────────────
    local label_col=$(( COLS / 2 ))
    local val_col=$(( label_col + 18 ))

    local stat_start_row=4
    # Row: Build Info
    move_to "$stat_start_row" "$label_col"
    printf "${BG_BLACK}${FG_GRAY}%-16s${RESET}" "Build Info"
    move_to "$stat_start_row" "$val_col"
    printf "${BG_BLACK}${FG_WHITE}${BOLD}%s${RESET}" "$build"

    # Row: Boot Time
    move_to $(( stat_start_row + 2 )) "$label_col"
    printf "${BG_BLACK}${FG_GRAY}%-16s${RESET}" "Boot Time"
    move_to $(( stat_start_row + 2 )) "$val_col"
    printf "${BG_BLACK}${FG_WHITE}${BOLD}%s${RESET}" "$boot"

    # Row: Uptime
    move_to $(( stat_start_row + 4 )) "$label_col"
    printf "${BG_BLACK}${FG_GRAY}%-16s${RESET}" "Uptime"
    move_to $(( stat_start_row + 4 )) "$val_col"
    printf "${BG_BLACK}${FG_WHITE}${BOLD}%s${RESET}" "$uptime_str"

    # Row: Hardware Model
    move_to $(( stat_start_row + 6 )) "$label_col"
    printf "${BG_BLACK}${FG_GRAY}%-16s${RESET}" "Hardware Model"
    move_to $(( stat_start_row + 6 )) "$val_col"
    printf "${BG_BLACK}${FG_WHITE}${BOLD}%s${RESET}" "$hw"

    # Row: IP Address
    move_to $(( stat_start_row + 8 )) "$label_col"
    printf "${BG_BLACK}${FG_GRAY}%-16s${RESET}" "IP Address"
    move_to $(( stat_start_row + 8 )) "$val_col"
    printf "${BG_BLACK}${FG_WHITE}${BOLD}%s${RESET}" "$ip"

    # Row: Service Status
    move_to $(( stat_start_row + 10 )) "$label_col"
    printf "${BG_BLACK}${FG_GRAY}%-16s${RESET}" "Service"
    move_to $(( stat_start_row + 10 )) "$val_col"
    printf "${BG_BLACK}${BOLD}${svc_status}${RESET}"

    # ── Divider between icon area and stats ──────────────────────────────────
    local mid=$(( COLS / 2 - 1 ))
    for r in $(seq 3 $(( stat_start_row + 12 ))); do
        move_to "$r" "$mid"
        printf "${BG_BLACK}${FG_GRAY}│${RESET}"
    done

    # ── Bottom bar ───────────────────────────────────────────────────────────
    local bottom=$(( ROWS - 1 ))
    move_to "$bottom" 1
    printf "${BG_BLACK}${FG_CYAN}"
    printf '%0.s─' $(seq 1 "$COLS")
    printf "${RESET}"

    move_to "$ROWS" 2
    if $WATCH; then
        printf "${BG_BLACK}${FG_GRAY} Press Ctrl+C to exit  •  Auto-refreshing every 5s${RESET}"
    else
        printf "${BG_BLACK}${FG_GRAY} Run with --watch to auto-refresh${RESET}"
    fi

    # Park cursor off-screen
    move_to "$ROWS" "$COLS"
}

# ── Main ─────────────────────────────────────────────────────────────────────
trap 'show_cursor; printf "${RESET}"; tput rmcup 2>/dev/null; exit' INT TERM EXIT

tput smcup 2>/dev/null || true   # save terminal state
hide_cursor

if $WATCH; then
    while true; do
        render
        sleep 5
    done
else
    render
    # Block until keypress
    read -rsn1 -t 0 _dummy 2>/dev/null || true
    read -rsn1 _key 2>/dev/null || true
fi
