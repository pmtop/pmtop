package app

import (
	"fmt"
	"strings"

	"github.com/pmtop/pmtop/internal/process"
	"github.com/pmtop/pmtop/internal/ui"
)

// detailView renders the process detail side panel (PRD 6.2) with scroll
// support and height constraint.
func (m Model) detailView(width, height int) string {
	if m.detail == nil {
		return ui.Box("Process Detail", "(no selection)", width, 0)
	}
	d := m.detail
	var lines []string

	if d.err != nil {
		lines = append(lines, "error: "+d.err.Error())
	}
	p := d.proc
	lines = append(lines,
		fmt.Sprintf("PID:        %d", p.PID),
		fmt.Sprintf("PPID:       %d", p.PPID),
		fmt.Sprintf("Name:       %s", p.Name),
		fmt.Sprintf("User:       %s (%d)", p.User, p.UID),
		fmt.Sprintf("Command:    %s", p.Cmdline),
		fmt.Sprintf("Exe Path:   %s", p.Exe),
		fmt.Sprintf("CWD:        %s", p.CWD),
		fmt.Sprintf("Start:      %s", p.StartTime.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("MEM:        %s (RSS)  %s (VSZ)", humanBytes(p.VmRSS), humanBytes(p.VmSize)),
		fmt.Sprintf("CPU:        %.1f%%", d.cpuPct),
	)

	if d.pkgName != "" {
		lines = append(lines, fmt.Sprintf("Package:    %s (dpkg/rpm)", d.pkgName))
	} else if d.pkgErr != nil {
		lines = append(lines, "Package:    -")
	}

	// Container association.
	s := m.currentContainerInfo()
	if s.Runtime != "" {
		lines = append(lines, fmt.Sprintf("Container:  %s %s", s.Runtime, shortID(s.ContainerID)))
		if s.ContainerName != "" {
			lines = append(lines, "  Name: "+s.ContainerName)
		}
		if s.ContainerImage != "" {
			lines = append(lines, "  Image: "+s.ContainerImage)
		}
		if s.ContainerStatus != "" {
			lines = append(lines, "  Status: "+s.ContainerStatus)
		}
	} else {
		lines = append(lines, "Container:  -")
	}

	if len(d.cg.Lines) > 0 {
		lines = append(lines, fmt.Sprintf("Cgroup:     v%d %s", d.cg.Version, d.cg.Lines[0].Path))
	}

	// Apply scroll offset.
	maxContentHeight := height - 4 // status(1) + box border(2) + hints(1)
	if maxContentHeight < 3 {
		maxContentHeight = 3
	}
	scroll := d.scroll
	if scroll < 0 {
		scroll = 0
	}
	if scroll > len(lines)-maxContentHeight && len(lines) > maxContentHeight {
		scroll = len(lines) - maxContentHeight
	}
	end := scroll + maxContentHeight
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[scroll:end]
	content := strings.Join(visible, "\n")

	if scroll > 0 || end < len(lines) {
		content += fmt.Sprintf("\n[%d-%d/%d]", scroll+1, end, len(lines))
	}

	return ui.Box("Process Detail (PID "+itoa(d.pid)+")", content, width, height-2)
}

// signalView renders the signal-selection dialog and optional confirmation
// (PRD 6.3, FR-06-01..04).
func (m Model) signalView(width int) string {
	if m.signal == nil {
		return ""
	}
	st := m.signal
	title := fmt.Sprintf("Send signal to %s (PID: %d)", st.name, st.pid)
	var sb strings.Builder
	for i, sig := range process.Signals {
		mark := "○"
		if i == st.sel {
			mark = "●"
		}
		fmt.Fprintf(&sb, "  %s %-8s (%d)  — %s\n", mark, sig.Name, sig.Num, sig.Desc)
	}
	if st.confirm {
		sig := process.Signals[st.sel]
		confirm := fmt.Sprintf("Confirm: send %s to %s (PID %d)?\n[Enter] yes   [Esc] no", sig.Name, st.name, st.pid)
		return ui.Dialog(title, sb.String()+"\n"+confirm, width)
	}
	return ui.Box(title, sb.String(), width, 0)
}

// shortID returns the first 12 chars of a container id, or the whole id.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// humanBytes renders a byte count as a compact human-readable string.
func humanBytes(b uint64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1024*1024*1024))
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1024))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
