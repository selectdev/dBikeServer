#!/usr/bin/env bash
# launch-panel.sh — opens admin-panel.sh in a fullscreen Terminal.app window (macOS only).
# Called by the com.dbikeserver.panel LaunchAgent on login.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PANEL="${SCRIPT_DIR}/admin-panel.sh"

osascript - "$PANEL" <<'APPLESCRIPT'
on run argv
    set panelScript to item 1 of argv
    tell application "Terminal"
        activate
        -- Close any leftover dBike panel windows from a previous session
        repeat with w in windows
            if name of w contains "admin-panel" then
                close w
            end if
        end repeat
        set w to do script ("exec bash '" & panelScript & "' --watch")
        delay 0.4
        set fullscreen of front window to true
    end tell
end run
APPLESCRIPT
