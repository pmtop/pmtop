package app

import (
	"fmt"
	"strings"

	"github.com/pmtop/pmtop/internal/process"
	"github.com/pmtop/pmtop/internal/ui"
)

// detailView renders the process detail side panel (PRD 6.2).
func (m Model) detailView(width int) string {
	if m.detail == nil {
		return ui.Box("Process Detail", "(no selection)", width)
	}
	d := m.detail
	var sb strings.Builder
	w := fmt.Fprintln

	if d.err != nil {
		_, _ = w(&sb, "error: ", d.err)
	}
	p := d.proc
	_, _ = w(&sb, "PID:        ", p.PID)
	_, _ = w(&sb, "PPID:       ", p.PPID)
	_, _ = w(&sb, "Name:       ", p.Name)
	_, _ = w(&sb, "User:       ", p.User, " (", p.UID, ")")
	_, _ = w(&sb, "Command:    ", p.Cmdline)
	_, _ = w(&sb, "Exe Path:   ", p.Exe)
	_, _ = w(&sb, "CWD:        ", p.CWD)
	_, _ = w(&sb, "Start:      ", p.StartTime)
	_, _ = w(&sb, "MEM:        ", humanBytes(p.VmRSS), "   CPU: -")
	if d.pkgName != "" {
		_, _ = w(&sb, "Package:    ", d.pkgName, " (dpkg/rpm)")
	} else if d.pkgErr != nil {
		_, _ = w(&sb, "Package:    -")
	}
	// Container association.
	s := m.currentContainerInfo()
	if s.Runtime != "" {
		_, _ = w(&sb, "Container:  ", s.Runtime, " ", shortID(s.ContainerID))
		if s.ContainerName != "" {
			_, _ = w(&sb, "  Name: ", s.ContainerName)
		}
		if s.ContainerImage != "" {
			_, _ = w(&sb, "  Image: ", s.ContainerImage)
		}
		if s.ContainerStatus != "" {
			_, _ = w(&sb, "  Status: ", s.ContainerStatus)
		}
	} else {
		_, _ = w(&sb, "Container:  -")
	}
	if len(d.cg.Lines) > 0 {
		_, _ = w(&sb, "Cgroup:     ", d.cg.Version, " ", d.cg.Lines[0].Path)
	}
	return ui.Box("Process Detail (PID "+itoa(d.pid)+")", sb.String(), width)
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
	if st.result != "" {
		sb.WriteString("\n")
		sb.WriteString(st.result)
	}
	return ui.Box(title, sb.String(), width)
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
