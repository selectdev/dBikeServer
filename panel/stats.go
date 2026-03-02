package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)



type svcInfo struct {
	status   string 
	pid      int
	restarts int    
	uptime   string 
}

type netIface struct {
	name string
	addr string
}

type systemStats struct {
	collectedAt time.Time

	
	svc     svcInfo
	version string

	
	cpuPct    float64
	memUsed   uint64
	memTotal  uint64
	diskUsed  uint64
	diskTotal uint64

	
	ifaces []netIface

	
	hwModel  string
	osName   string
	bootTime string

	
	bleDevice string

	
	scripts []string

	
	logs []string
}

func gatherStats(scriptDir string) systemStats {
	
	type cpuResult struct{ pct float64 }
	type memResult struct{ used, total uint64 }
	type diskResult struct{ used, total uint64 }

	cpuCh := make(chan cpuResult, 1)
	go func() { cpuCh <- cpuResult{cpuPercent()} }()

	var s systemStats
	s.collectedAt = time.Now()
	s.bleDevice = bleDeviceName()
	s.version = buildVersion(scriptDir)
	s.scripts = loadedScripts(scriptDir)
	s.svc = serviceInfo()
	s.ifaces = networkIfaces()
	s.hwModel = hwModel()
	s.osName = osVersion()
	s.bootTime = systemBootTime()
	s.logs = recentLogs(scriptDir, 8)
	s.memUsed, s.memTotal = memInfo()
	s.diskUsed, s.diskTotal = diskInfo()

	r := <-cpuCh
	s.cpuPct = r.pct
	return s
}



func cpuPercent() float64 {
	switch runtime.GOOS {
	case "linux":
		return cpuPercentLinux()
	case "darwin":
		return cpuPercentDarwin()
	}
	return 0
}

type procStat struct{ user, nice, system, idle, iowait, irq, softirq, steal uint64 }

func (p procStat) total() uint64 {
	return p.user + p.nice + p.system + p.idle + p.iowait + p.irq + p.softirq + p.steal
}

func readProcStat() (procStat, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return procStat{}, err
	}
	var s procStat
	for _, line := range strings.SplitN(string(data), "\n", 3) {
		if strings.HasPrefix(line, "cpu ") {
			fmt.Sscanf(line, "cpu %d %d %d %d %d %d %d %d",
				&s.user, &s.nice, &s.system, &s.idle,
				&s.iowait, &s.irq, &s.softirq, &s.steal)
			break
		}
	}
	return s, nil
}

func cpuPercentLinux() float64 {
	s1, err := readProcStat()
	if err != nil {
		return 0
	}
	time.Sleep(200 * time.Millisecond)
	s2, err := readProcStat()
	if err != nil {
		return 0
	}
	totalDelta := s2.total() - s1.total()
	idleDelta := (s2.idle + s2.iowait) - (s1.idle + s1.iowait)
	if totalDelta == 0 {
		return 0
	}
	return float64(totalDelta-idleDelta) / float64(totalDelta) * 100
}

func cpuPercentDarwin() float64 {
	out, err := exec.Command("top", "-l", "1", "-n", "0").Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "CPU usage:") {
			var user, sys float64
			fmt.Sscanf(line, "CPU usage: %f%% user, %f%% sys,", &user, &sys)
			return user + sys
		}
	}
	return 0
}



func memInfo() (used, total uint64) {
	switch runtime.GOOS {
	case "linux":
		return memInfoLinux()
	case "darwin":
		return memInfoDarwin()
	}
	return 0, 1
}

func memInfoLinux() (used, total uint64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 1
	}
	var totalKB, availKB uint64
	for _, line := range strings.Split(string(data), "\n") {
		var v uint64
		if _, err := fmt.Sscanf(line, "MemTotal: %d kB", &v); err == nil {
			totalKB = v
		}
		if _, err := fmt.Sscanf(line, "MemAvailable: %d kB", &v); err == nil {
			availKB = v
		}
	}
	total = totalKB * 1024
	used = (totalKB - availKB) * 1024
	return
}

func memInfoDarwin() (used, total uint64) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err == nil {
		total, _ = strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	}
	if total == 0 {
		total = 1
	}

	vmOut, _ := exec.Command("vm_stat").Output()
	var pageSize uint64 = 4096
	var wired, active, compressed uint64

	for _, line := range strings.Split(string(vmOut), "\n") {
		if strings.Contains(line, "page size of") {
			fmt.Sscanf(line, "Mach Virtual Memory Statistics: (page size of %d bytes)", &pageSize)
		}
		var pages uint64
		switch {
		case strings.HasPrefix(line, "Pages wired down:"):
			fmt.Sscanf(strings.TrimPrefix(line, "Pages wired down:"), " %d.", &pages)
			wired = pages
		case strings.HasPrefix(line, "Pages active:"):
			fmt.Sscanf(strings.TrimPrefix(line, "Pages active:"), " %d.", &pages)
			active = pages
		case strings.HasPrefix(line, "Pages occupied by compressor:"):
			fmt.Sscanf(strings.TrimPrefix(line, "Pages occupied by compressor:"), " %d.", &pages)
			compressed = pages
		}
	}
	used = (wired + active + compressed) * pageSize
	return
}



func diskInfo() (used, total uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, 1
	}
	bsize := uint64(stat.Bsize)
	total = stat.Blocks * bsize
	used = (stat.Blocks - stat.Bfree) * bsize
	return
}



func networkIfaces() []netIface {
	nets, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var result []netIface
	for _, n := range nets {
		if n.Flags&net.FlagLoopback != 0 || n.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, _ := n.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				result = append(result, netIface{name: n.Name, addr: ipNet.IP.String()})
			}
		}
	}
	return result
}



func hwModel() string {
	switch runtime.GOOS {
	case "darwin":
		out, _ := exec.Command("sysctl", "-n", "hw.model").Output()
		return strings.TrimSpace(string(out))
	case "linux":
		for _, p := range []string{
			"/proc/device-tree/model",
			"/sys/firmware/devicetree/base/model",
		} {
			if data, err := os.ReadFile(p); err == nil {
				return strings.TrimRight(string(data), "\x00\n")
			}
		}
		data, _ := os.ReadFile("/proc/cpuinfo")
		for _, line := range strings.Split(string(data), "\n") {
			for _, prefix := range []string{"Model name", "Hardware", "Model"} {
				if strings.HasPrefix(line, prefix) {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						return strings.TrimSpace(parts[1])
					}
				}
			}
		}
	}
	return runtime.GOARCH
}

func osVersion() string {
	switch runtime.GOOS {
	case "darwin":
		out, _ := exec.Command("sw_vers", "-productVersion").Output()
		return "macOS " + strings.TrimSpace(string(out))
	case "linux":
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				}
			}
		}
		out, _ := exec.Command("uname", "-sr").Output()
		return strings.TrimSpace(string(out))
	}
	return runtime.GOOS
}

func systemBootTime() string {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
		if err == nil {
			
			if parts := strings.SplitN(string(out), "}", 2); len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	case "linux":
		data, err := os.ReadFile("/proc/uptime")
		if err == nil {
			var uptimeSec float64
			fmt.Sscanf(string(data), "%f", &uptimeSec)
			boot := time.Now().Add(-time.Duration(uptimeSec * float64(time.Second)))
			return boot.Format("2006-01-02 15:04:05")
		}
	}
	return "unknown"
}



func serviceInfo() svcInfo {
	switch runtime.GOOS {
	case "linux":
		return serviceInfoLinux()
	case "darwin":
		return serviceInfoDarwin()
	}
	return svcInfo{status: "unknown", restarts: -1}
}

func serviceInfoLinux() svcInfo {
	out, err := exec.Command("systemctl", "show", "dbikeserver",
		"--property=ActiveState,MainPID,NRestarts,ActiveEnterTimestamp",
		"--no-pager").Output()
	if err != nil {
		
		return pgrepFallback()
	}
	var info svcInfo
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "ActiveState":
			if parts[1] == "active" {
				info.status = "running"
			} else {
				info.status = parts[1]
			}
		case "MainPID":
			pid, _ := strconv.Atoi(parts[1])
			if pid > 0 {
				info.pid = pid
			}
		case "NRestarts":
			info.restarts, _ = strconv.Atoi(parts[1])
		case "ActiveEnterTimestamp":
			if parts[1] != "" && parts[1] != "n/a" {
				
				for _, layout := range []string{
					"Mon 2006-01-02 15:04:05 MST",
					"Mon 2006-01-02 15:04:05 UTC",
				} {
					if t, err := time.Parse(layout, parts[1]); err == nil {
						info.uptime = formatUptime(time.Since(t))
						break
					}
				}
			}
		}
	}
	
	if info.status != "running" {
		if fb := pgrepFallback(); fb.status == "running" {
			return fb
		}
	}
	return info
}

func serviceInfoDarwin() svcInfo {
	
	out, err := exec.Command("launchctl", "list", "dbikeserver").Output()
	info := svcInfo{restarts: -1}
	if err == nil {
		text := string(out)
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			
			if strings.HasPrefix(line, `"PID"`) {
				var pid int
				fmt.Sscanf(line, `"PID" = %d;`, &pid)
				info.pid = pid
			}
		}
		if info.pid > 0 {
			info.status = "running"
			return info
		}
		
		row, _ := exec.Command("sh", "-c",
			`launchctl list 2>/dev/null | awk '$3=="dbikeserver"'`).Output()
		fields := strings.Fields(strings.TrimSpace(string(row)))
		if len(fields) >= 1 && fields[0] != "-" && fields[0] != "" {
			info.pid, _ = strconv.Atoi(fields[0])
		}
		if info.pid > 0 {
			info.status = "running"
			return info
		}
	}
	
	if fb := pgrepFallback(); fb.status == "running" {
		return fb
	}
	info.status = "stopped"
	return info
}



func pgrepFallback() svcInfo {
	out, err := exec.Command("pgrep", "-x", "dbikeserver").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return svcInfo{status: "stopped", restarts: -1}
	}
	
	first := strings.Fields(strings.TrimSpace(string(out)))[0]
	pid, _ := strconv.Atoi(first)
	return svcInfo{status: "running", pid: pid, restarts: -1}
}



func recentLogs(scriptDir string, n int) []string {
	ns := strconv.Itoa(n)
	var out []byte
	switch runtime.GOOS {
	case "linux":
		out, _ = exec.Command("journalctl", "-u", "dbikeserver",
			"-n", ns, "--no-pager", "--output=short").Output()
	case "darwin":
		logFile := "/var/log/dbikeserver/stdout.log"
		out, _ = exec.Command("tail", "-n", ns, logFile).Output()
	}
	if len(out) == 0 {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}



func loadedScripts(scriptDir string) []string {
	files, err := filepath.Glob(filepath.Join(scriptDir, "scripts", "*.tengo"))
	if err != nil {
		return nil
	}
	var names []string
	for _, f := range files {
		names = append(names, strings.TrimSuffix(filepath.Base(f), ".tengo"))
	}
	return names
}



func bleDeviceName() string {
	if name := os.Getenv("DBIKE_BLE_NAME"); name != "" {
		return name
	}
	return "dBike-Go"
}

func buildVersion(scriptDir string) string {
	out, err := exec.Command("git", "-C", scriptDir, "describe", "--tags", "--always").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return "dev"
}



func formatUptime(d time.Duration) string {
	d = d.Round(time.Minute)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}
