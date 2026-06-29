package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/pmtop/pmtop/internal/process"
	"github.com/pmtop/pmtop/internal/ui"
)

// detailPanel renders the process detail panel for the right-upper area.
// It is always visible and follows the current cursor selection.
func (m Model) detailPanel(width, height int) string {
	s, ok := m.currentSocket()
	if !ok || s.PID <= 0 {
		msg := "(no selection)"
		if m.err != nil {
			msg = "error: " + m.err.Error()
		} else if len(m.full) == 0 {
			msg = "(waiting for data...)"
		} else if len(m.socks) == 0 {
			msg = "(no matching sockets)"
		} else if s.PID <= 0 {
			msg = "(ownerless socket)"
		}
		return ui.Box("Process Detail", msg, width, height)
	}

	d := m.detail
	if d == nil {
		return ui.Box("Process Detail", "(loading...)", width, height)
	}

	var lines []string
	if d.err != nil {
		lines = append(lines, "error: "+d.err.Error())
	}
	p := d.proc
	lines = append(lines,
		fmt.Sprintf("PID:      %d", p.PID),
		fmt.Sprintf("PPID:     %d", p.PPID),
		fmt.Sprintf("Name:     %s", p.Name),
		fmt.Sprintf("User:     %s (%d)", p.User, p.UID),
		fmt.Sprintf("Command:  %s", truncateStr(p.Cmdline, 28)),
		fmt.Sprintf("Exe:      %s", truncateStr(p.Exe, 28)),
		fmt.Sprintf("CWD:      %s", truncateStr(p.CWD, 28)),
		fmt.Sprintf("Start:    %s", p.StartTime.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("MEM:      %s RSS  %s VSZ", humanBytes(p.VmRSS), humanBytes(p.VmSize)),
		fmt.Sprintf("CPU:      %.1f%%", d.cpuPct),
	)

	if d.pkgName != "" {
		lines = append(lines, fmt.Sprintf("Package:  %s", truncateStr(d.pkgName, 28)))
	}

	sock := m.currentContainerInfo()
	if sock.Runtime != "" {
		lines = append(lines, fmt.Sprintf("Runtime:  %s %s", sock.Runtime, shortID(sock.ContainerID)))
		if sock.ContainerName != "" {
			lines = append(lines, "  Name:   "+sock.ContainerName)
		}
		if sock.ContainerImage != "" {
			lines = append(lines, "  Image:  "+truncateStr(sock.ContainerImage, 26))
		}
	}

	if len(d.cg.Lines) > 0 {
		lines = append(lines, fmt.Sprintf("Cgroup:   v%d %s", d.cg.Version, truncateStr(d.cg.Lines[0].Path, 22)))
	}

	content := strings.Join(lines, "\n")
	title := "Process Detail (PID " + itoa(d.pid) + ")"
	return ui.Box(title, content, width, height)
}

// signalPanel renders the signal-selection panel for the right-bottom area.
// Returns empty string when no signal operation is active.
func (m Model) signalPanel(width, height int) string {
	if m.signal == nil || height <= 0 {
		return ""
	}
	st := m.signal

	if st.confirm {
		sig := process.Signals[st.sel]
		content := fmt.Sprintf("Confirm: send %s to %s?\nPID: %d\n\n[Enter] yes   [Esc] no", sig.Name, st.name, st.pid)
		return ui.Dialog("Confirm Signal", content, width, height)
	}

	title := fmt.Sprintf("Signal: %s (PID %d)", st.name, st.pid)
	var lines []string
	for i, sig := range process.Signals {
		mark := "○"
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		if i == st.sel {
			mark = "●"
			style = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236")).Foreground(lipgloss.Color("15"))
		}
		line := fmt.Sprintf(" %s %-8s (%d) %s", mark, sig.Name, sig.Num, sig.Desc)
		lines = append(lines, style.Render(line))
	}
	content := strings.Join(lines, "\n")
	return ui.Box(title, content, width, height)
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func truncateStr(s string, max int) string {
	if max <= 0 {
		return s
	}
	w := 0
	for i := range s {
		if w >= max {
			return s[:i] + "…"
		}
		w++
	}
	return s
}

func humanBytes(b uint64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(1024*1024*1024))
	case b >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(1024))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
