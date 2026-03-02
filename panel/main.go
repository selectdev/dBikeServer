package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)



type tickMsg time.Time
type statsMsg systemStats
type actionDoneMsg struct {
	label string
	err   error
}
type clearActionMsg struct{}



type model struct {
	width     int
	height    int
	stats     systemStats
	scriptDir string
	watchMode bool
	interval  time.Duration
	action    string 
	ready     bool   
}

func newModel(scriptDir string, watchMode bool, interval time.Duration) model {
	return model{
		scriptDir: scriptDir,
		watchMode: watchMode,
		interval:  interval,
		width:     80,
		height:    24,
	}
}



func (m model) Init() tea.Cmd {
	return collectCmd(m.scriptDir)
}



func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case statsMsg:
		m.stats = systemStats(msg)
		m.ready = true
		if !m.watchMode {
			
			return m, nil
		}
		return m, tea.Tick(m.interval, func(t time.Time) tea.Msg { return tickMsg(t) })

	case tickMsg:
		return m, collectCmd(m.scriptDir)

	case actionDoneMsg:
		if msg.err != nil {
			m.action = fmt.Sprintf("✖  %s failed: %v", msg.label, msg.err)
		} else {
			m.action = fmt.Sprintf("✔  %s complete", msg.label)
		}
		
		return m, tea.Batch(
			collectCmd(m.scriptDir),
			tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return clearActionMsg{} }),
		)

	case clearActionMsg:
		m.action = ""

	case tea.KeyMsg:
		
		if !m.ready {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q", "Q":
			return m, tea.Quit
		case "r", "R":
			return m, collectCmd(m.scriptDir)
		case "s", "S":
			return m, m.toggleServiceCmd()
		case "b", "B":
			m.action = "Building…"
			return m, runAction(m.scriptDir, "build", "Build")
		case "u", "U":
			m.action = "Upgrading…"
			return m, runAction(m.scriptDir, "upgrade", "Upgrade")
		default:
			
			if !m.watchMode {
				return m, tea.Quit
			}
		}
	}

	return m, nil
}



func collectCmd(scriptDir string) tea.Cmd {
	return func() tea.Msg {
		return statsMsg(gatherStats(scriptDir))
	}
}

func (m model) toggleServiceCmd() tea.Cmd {
	if m.stats.svc.status == "running" {
		m.action = "Stopping service…"
		return stopServerCmd(m.stats.svc.pid)
	}
	m.action = "Starting service…"
	return startServerCmd(m.scriptDir)
}


func stopServerCmd(pid int) tea.Cmd {
	return func() tea.Msg {
		var err error
		if pid > 0 {
			err = exec.Command("kill", strconv.Itoa(pid)).Run()
		} else {
			err = exec.Command("pkill", "-x", "dbikeserver").Run()
		}
		return actionDoneMsg{label: "Stop", err: err}
	}
}


func startServerCmd(scriptDir string) tea.Cmd {
	return func() tea.Msg {
		bin := filepath.Join(scriptDir, "dbikeserver")
		cmd := exec.Command(bin)
		cmd.Dir = scriptDir
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		err := cmd.Start()
		return actionDoneMsg{label: "Start", err: err}
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


func runAction(scriptDir, subCmd, label string) tea.Cmd {
	return func() tea.Msg {
		installSh := filepath.Join(scriptDir, "install.sh")
		err := exec.Command("bash", installSh, subCmd).Run()
		return actionDoneMsg{label: label, err: err}
	}
}



func main() {
	var watchMode bool
	var interval int

	flag.BoolVar(&watchMode, "watch", false, "Auto-refresh the panel (use with --interval to tune)")
	flag.IntVar(&interval, "interval", 2, "Refresh interval in seconds (implies --watch)")
	flag.Parse()

	
	if interval != 2 {
		watchMode = true
	}

	
	
	scriptDir := "."
	if exe, err := os.Executable(); err == nil {
		scriptDir = filepath.Dir(exe)
	}

	
	ensureServerRunning(scriptDir)

	m := newModel(scriptDir, watchMode, time.Duration(interval)*time.Second)

	p := tea.NewProgram(m,
		tea.WithAltScreen(), 
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "dbikeserver-panel: %v\n", err)
		os.Exit(1)
	}
}
