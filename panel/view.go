package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)



var (
	colAccent   = lipgloss.Color("#00D7FF")
	colMuted    = lipgloss.Color("#777777")
	colWhite    = lipgloss.Color("#E8E8E8")
	colGreen    = lipgloss.Color("#00D700")
	colYellow   = lipgloss.Color("#FFCC00")
	colRed      = lipgloss.Color("#FF4444")
	colDivider  = lipgloss.Color("#2A2A2A")
	colBarGreen = lipgloss.Color("#00AA00")
	colBarAmber = lipgloss.Color("#CCAA00")
	colBarRed   = lipgloss.Color("#CC3333")
	colKeyBg    = colAccent
)

func clr(c lipgloss.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

var (
	sAccent  = clr(colAccent).Bold(true)
	sMuted   = clr(colMuted)
	sWhite   = clr(colWhite).Bold(true)
	sGreen   = clr(colGreen).Bold(true)
	sYellow  = clr(colYellow)
	sRed     = clr(colRed).Bold(true)
	sDiv     = clr(colDivider)
	sLabel   = clr(colMuted)
	sHeavy   = clr(colAccent)
	sKeyHint = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(colKeyBg).
			Bold(true)
)



func (m model) View() string {
	if !m.ready {
		return "\n" + sAccent.Render("  Loading dBike Server panel…")
	}

	w := m.width

	
	mid := w / 2
	leftW := mid - 1
	rightW := w - mid - 1

	leftStr := m.renderLeft(leftW)
	rightStr := m.renderRight(rightW)

	
	leftH := strings.Count(leftStr, "\n") + 1
	rightH := strings.Count(rightStr, "\n") + 1
	panelH := leftH
	if rightH > panelH {
		panelH = rightH
	}
	divLines := make([]string, panelH)
	for i := range divLines {
		divLines[i] = sDiv.Render("│")
	}
	divStr := strings.Join(divLines, "\n")

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftStr, divStr, rightStr)

	
	
	logLines := m.height - panelH - 6
	if logLines < 0 {
		logLines = 0
	}

	var sections []string
	sections = append(sections, m.renderHeader(w))
	sections = append(sections, panels)
	if logLines > 0 {
		sections = append(sections, m.renderLogs(w, logLines))
	} else {
		sections = append(sections, sDiv.Render(strings.Repeat("─", w)))
	}
	sections = append(sections, m.renderFooter(w))
	return strings.Join(sections, "\n")
}



func (m model) renderHeader(w int) string {
	bar := sHeavy.Render(strings.Repeat("━", w))

	titlePart := sAccent.Render(" dBike Server")
	verPart := sMuted.Render("  " + m.stats.version)
	nowPart := sMuted.Render(time.Now().Format("2006-01-02  15:04:05") + " ")

	titleW := lipgloss.Width(titlePart + verPart)
	nowW := lipgloss.Width(nowPart)
	gap := w - titleW - nowW
	if gap < 1 {
		gap = 1
	}

	titleLine := titlePart + verPart + strings.Repeat(" ", gap) + nowPart
	return strings.Join([]string{bar, titleLine, bar}, "\n")
}



func (m model) renderLeft(w int) string {
	s := m.stats
	cw := w - 1 

	var b strings.Builder
	line := func(l string) { b.WriteString(" " + l + "\n") }
	blank := func() { b.WriteString("\n") }
	sep := func() { line(sDiv.Render(strings.Repeat("─", cw))) }

	
	line(sAccent.Render("SERVICE"))
	sep()
	line(lrow("Status", statusDot(s.svc.status), 10))
	if s.svc.pid > 0 {
		line(lrow("PID", sWhite.Render(fmt.Sprintf("%d", s.svc.pid)), 10))
	}
	if s.svc.uptime != "" {
		line(lrow("Uptime", sWhite.Render(s.svc.uptime), 10))
	}
	if s.svc.restarts >= 0 {
		line(lrow("Restarts", sWhite.Render(fmt.Sprintf("%d", s.svc.restarts)), 10))
	}
	if s.version != "" {
		line(lrow("Version", sWhite.Render(trunc(s.version, cw-14)), 10))
	}
	blank()

	
	line(sAccent.Render("BLUETOOTH"))
	sep()
	line(lrow("Device", sWhite.Render(s.bleDevice), 10))
	bleLabel := "Inactive"
	if s.svc.status == "running" {
		bleLabel = "Advertising"
	}
	line(lrow("Status", statusDot2(s.svc.status == "running", bleLabel), 10))
	blank()

	
	line(sAccent.Render("SCRIPTS"))
	sep()
	if len(s.scripts) == 0 {
		line(sMuted.Render("no scripts loaded"))
	} else {
		line(sMuted.Render(fmt.Sprintf("%d script(s) loaded", len(s.scripts))))
		for _, sc := range s.scripts {
			line(sGreen.Render("○ ") + sMuted.Render(sc))
		}
	}

	
	out := strings.TrimRight(b.String(), "\n")
	return out
}



func (m model) renderRight(w int) string {
	s := m.stats
	cw := w - 1

	var b strings.Builder
	line := func(l string) { b.WriteString(" " + l + "\n") }
	blank := func() { b.WriteString("\n") }
	sep := func() { line(sDiv.Render(strings.Repeat("─", cw))) }

	
	line(sAccent.Render("SYSTEM"))
	sep()

	barW := cw - 18
	if barW < 8 {
		barW = 8
	}

	line(metricRow("CPU", s.cpuPct, barW,
		fmt.Sprintf("%4.0f%%", s.cpuPct), ""))

	if s.memTotal > 0 {
		memPct := float64(s.memUsed) / float64(s.memTotal) * 100
		usedMB := s.memUsed / 1024 / 1024
		totalMB := s.memTotal / 1024 / 1024
		line(metricRow("Memory", memPct, barW,
			fmt.Sprintf("%4.0f%%", memPct),
			fmt.Sprintf("  %d/%d MB", usedMB, totalMB)))
	}

	if s.diskTotal > 0 {
		diskPct := float64(s.diskUsed) / float64(s.diskTotal) * 100
		usedGB := float64(s.diskUsed) / 1024 / 1024 / 1024
		totalGB := float64(s.diskTotal) / 1024 / 1024 / 1024
		line(metricRow("Disk", diskPct, barW,
			fmt.Sprintf("%4.0f%%", diskPct),
			fmt.Sprintf("  %.1f/%.1f GB", usedGB, totalGB)))
	}
	blank()

	
	line(sAccent.Render("NETWORK"))
	sep()
	if len(s.ifaces) == 0 {
		line(sMuted.Render("no interfaces"))
	} else {
		for _, iface := range s.ifaces {
			line(sLabel.Render(fmt.Sprintf("%-6s", iface.name)) +
				"  " + sWhite.Render(iface.addr))
		}
	}
	blank()

	
	line(sAccent.Render("HARDWARE"))
	sep()
	if s.hwModel != "" {
		line(lrow("Model", sWhite.Render(trunc(s.hwModel, cw-14)), 10))
	}
	if s.osName != "" {
		line(lrow("OS", sWhite.Render(trunc(s.osName, cw-14)), 10))
	}
	if s.bootTime != "" {
		line(lrow("Booted", sWhite.Render(trunc(s.bootTime, cw-14)), 10))
	}

	out := strings.TrimRight(b.String(), "\n")
	return out
}



func (m model) renderLogs(w, maxLines int) string {
	var b strings.Builder
	b.WriteString(sDiv.Render(strings.Repeat("─", w)) + "\n")
	b.WriteString(" " + sAccent.Render("RECENT LOGS") + "\n")

	logs := m.stats.logs
	if len(logs) > maxLines {
		logs = logs[len(logs)-maxLines:]
	}
	if len(logs) == 0 {
		b.WriteString(sMuted.Render("  (no log entries found)") + "\n")
	} else {
		for _, l := range logs {
			b.WriteString(sMuted.Render(" "+trunc(l, w-2)) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}



func (m model) renderFooter(w int) string {
	bar := sHeavy.Render(strings.Repeat("━", w))

	keys := " " + keyHint("Q") + " Quit  " +
		keyHint("R") + " Refresh  " +
		keyHint("S") + " Start/Stop  " +
		keyHint("B") + " Build  " +
		keyHint("U") + " Upgrade"

	var right string
	if m.action != "" {
		right = sYellow.Render(m.action) + " "
	} else if m.watchMode {
		right = sMuted.Render(fmt.Sprintf("%ds ↻ ", int(m.interval.Seconds())))
	} else {
		right = sMuted.Render("press any key to exit ")
	}

	keysW := lipgloss.Width(keys)
	rightW := lipgloss.Width(right)
	gap := w - keysW - rightW
	if gap < 0 {
		gap = 0
	}
	footerLine := keys + strings.Repeat(" ", gap) + right

	return strings.Join([]string{bar, footerLine}, "\n")
}




func lrow(label, value string, labelW int) string {
	return sLabel.Render(fmt.Sprintf("%-*s", labelW, label)) + "  " + value
}


func metricRow(label string, pct float64, barW int, pctStr, extra string) string {
	l := sLabel.Render(fmt.Sprintf("%-7s", label))
	bar := progressBar(pct, barW)
	p := sMuted.Render(pctStr)
	return l + "  " + bar + "  " + p + sMuted.Render(extra)
}


func progressBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(math.Round(pct / 100.0 * float64(width)))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	var color lipgloss.Color
	switch {
	case pct > 85:
		color = colBarRed
	case pct > 65:
		color = colBarAmber
	default:
		color = colBarGreen
	}
	return lipgloss.NewStyle().Foreground(color).Render(bar)
}


func statusDot(status string) string {
	switch status {
	case "running":
		return sGreen.Render("● RUNNING")
	case "stopped", "inactive", "failed":
		return sRed.Render("● " + strings.ToUpper(status))
	default:
		return sYellow.Render("○ " + strings.ToUpper(status))
	}
}

func statusDot2(ok bool, label string) string {
	if ok {
		return sGreen.Render("● " + label)
	}
	return sMuted.Render("○ " + label)
}


func keyHint(k string) string {
	return sKeyHint.Render("[" + k + "]")
}


func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(runes[:n-1]) + "…"
}
