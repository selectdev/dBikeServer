

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "dbikeserver-launcher: macOS only")
		os.Exit(1)
	}

	panelBin, err := findPanel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbikeserver-launcher: %v\n", err)
		os.Exit(1)
	}

	
	ensureServerRunning(filepath.Dir(panelBin))

	
	if iterm := iTermAppPath(); iterm != "" {
		if err := launchITerm2(panelBin); err != nil {
			fmt.Fprintf(os.Stderr, "dbikeserver-launcher: iTerm2 launch failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "dbikeserver-launcher: falling back to Terminal.app")
		} else {
			return
		}
	}

	if err := launchTerminalApp(panelBin); err != nil {
		fmt.Fprintf(os.Stderr, "dbikeserver-launcher: Terminal.app launch failed: %v\n", err)
		os.Exit(1)
	}
}




func ensureServerRunning(dir string) {
	if exec.Command("pgrep", "-x", "dbikeserver").Run() == nil {
		return 
	}
	bin := filepath.Join(dir, "dbikeserver")
	if _, err := os.Stat(bin); err != nil {
		return 
	}
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	_ = cmd.Start()
}


func findPanel() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		
		exe, err = filepath.Abs(os.Args[0])
		if err != nil {
			return "", fmt.Errorf("cannot determine executable path: %w", err)
		}
	}
	
	exe, _ = filepath.EvalSymlinks(exe)

	panel := filepath.Join(filepath.Dir(exe), "dbikeserver-panel")
	if _, err := os.Stat(panel); err != nil {
		return "", fmt.Errorf("dbikeserver-panel not found at %s\nRun: ./install.sh build", panel)
	}
	return panel, nil
}


func iTermAppPath() string {
	for _, p := range []string{
		"/Applications/iTerm2.app",
		"/Applications/iTerm.app",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}


func launchITerm2(panel string) error {
	safe := asPath(panel)
	script := fmt.Sprintf(`
tell application "iTerm2"
    activate
    -- Close any existing dBike panel windows.
    set theWindows to (get windows)
    repeat with w in theWindows
        try
            if name of w contains "dbikeserver-panel" then close w
        end try
    end repeat
    -- Open a new window and run the panel.
    set newWin to (create window with default profile)
    tell current session of newWin
        write text "exec %s --watch"
    end tell
    tell newWin
        set fullscreen to true
    end tell
end tell
`, safe)
	return runScript(script)
}


func launchTerminalApp(panel string) error {
	safe := asPath(panel)
	script := fmt.Sprintf(`
tell application "Terminal"
    activate
    -- Close any leftover dBike panel windows.
    set theWindows to (get windows)
    repeat with w in reverse of theWindows
        try
            if name of w contains "dbikeserver-panel" then close w
        end try
    end repeat
    -- Open a new tab/window with the panel.
    set newTab to do script "exec %s --watch"
    delay 0.8
    -- Enter fullscreen (may require Accessibility permission on newer macOS).
    try
        set fullscreen of front window to true
    end try
end tell
`, safe)
	return runScript(script)
}


func runScript(script string) error {
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%v — %s", err, msg)
		}
		return err
	}
	return nil
}


func asPath(p string) string {
	p = strings.ReplaceAll(p, `\`, `\\`)
	p = strings.ReplaceAll(p, `"`, `\"`)
	return p
}
