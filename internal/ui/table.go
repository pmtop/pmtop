// Package ui contains the rendering primitives for the pmtop TUI: the port
// table, status bars, and dialogs. It depends only on lipgloss/bubbles and the
// netstat data types, keeping it fully unit-testable.
package ui

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// Column identifiers (indices into the column set).
const (
	ColProto = iota
	ColLocal
	ColRemote
	ColState
	ColPID
	ColProcess
	ColUser
	ColContainer
)

// BuildColumns returns the table column set fit to the given total width.
// Narrow terminals progressively shrink the wider text columns.
func BuildColumns(width int) []table.Column {
	// Fixed-ish widths for short columns; flexible for text columns.
	proto := 9
	state := 11
	pid := 7
	local := 22
	remote := 22
	process := 16
	user := 10
	container := 12

	fixed := proto + state + pid
	// 7 column separators (3 chars each " │ " in bubbles table) + padding.
	separators := 7 * 3
	avail := width - fixed - separators
	if avail < 40 {
		avail = 40
	}
	// Distribute available width across the 4 text columns.
	local = clamp(local, avail/5)
	remote = clamp(remote, avail/5)
	process = clamp(process, avail/3)
	user = clamp(user, avail/6)
	container = clamp(container, avail/6)
	// Recompute to fit within avail by trimming the largest first.
	for totalWidth(fixed, separators, local, remote, process, user, container) > width && width > 0 {
		switch {
		case process > 6:
			process--
		case remote > 6:
			remote--
		case local > 6:
			local--
		case container > 4:
			container--
		case user > 4:
			user--
		default:
			goto done
		}
	}
done:
	return []table.Column{
		{Title: "Proto", Width: proto},
		{Title: "Local", Width: local},
		{Title: "Remote", Width: remote},
		{Title: "State", Width: state},
		{Title: "PID", Width: pid},
		{Title: "Process", Width: process},
		{Title: "User", Width: user},
		{Title: "Container", Width: container},
	}
}

func totalWidth(fixed, sep, local, remote, process, user, container int) int {
	return fixed + sep + local + remote + process + user + container
}

func clamp(v, max int) int {
	if v > max {
		return max
	}
	if v < 4 {
		return 4
	}
	return v
}

// protoCell renders the protocol + state symbol cell, e.g. "TCP ▶".
func protoCell(s netstat.SocketInfo) string {
	return strings.ToUpper(string(s.Protocol)) + " " + s.State.Symbol()
}

// stateCell renders the state name (or "-" for stateless protocols).
func stateCell(s netstat.SocketInfo) string {
	if s.Protocol == netstat.ProtocolUnix && s.State == netstat.StateUnknown {
		return "-"
	}
	if !s.Protocol.IsTCP() && s.State == netstat.StateUnknown {
		return "-"
	}
	return s.State.String()
}

// localCell renders "addr:port" (or the unix path, truncated by the table).
func localCell(s netstat.SocketInfo) string {
	if s.Protocol == netstat.ProtocolUnix {
		if s.Path == "" {
			return "(anonymous)"
		}
		return s.Path
	}
	return s.LocalAddr + ":" + strconv.Itoa(int(s.LocalPort))
}

// remoteCell renders the remote endpoint.
func remoteCell(s netstat.SocketInfo) string {
	if s.Protocol == netstat.ProtocolUnix {
		return "-"
	}
	if s.RemoteAddr == "" || (s.RemoteAddr == "0.0.0.0" && s.RemotePort == 0) {
		return "*"
	}
	return s.RemoteAddr + ":" + strconv.Itoa(int(s.RemotePort))
}

// pidCell renders the PID or "-".
func pidCell(s netstat.SocketInfo) string {
	if s.PID == 0 {
		return "-"
	}
	return strconv.Itoa(s.PID)
}

// containerCell renders the container name, short id, or "-".
func containerCell(s netstat.SocketInfo) string {
	if s.ContainerName != "" {
		return s.ContainerName
	}
	if s.ContainerID != "" {
		if len(s.ContainerID) > 12 {
			return s.ContainerID[:12]
		}
		return s.ContainerID
	}
	return "-"
}

// RowsFromSockets converts sockets into table rows in column order.
func RowsFromSockets(socks []netstat.SocketInfo, style *Style) []table.Row {
	rows := make([]table.Row, 0, len(socks))
	for _, s := range socks {
		var pid, proc, user, container string
		if s.PID == 0 {
			pid, proc, user, container = "-", "-", "-", "-"
		} else {
			pid = pidCell(s)
			proc = s.ProcessName
			if proc == "" {
				proc = "-"
			}
			user = s.User
			if user == "" {
				user = strconv.Itoa(int(s.UID))
			}
			container = containerCell(s)
		}
		row := table.Row{
			protoCell(s),
			localCell(s),
			remoteCell(s),
			stateCell(s),
			pid,
			proc,
			user,
			container,
		}
		if style != nil {
			row = style.styleRow(row, s)
		}
		rows = append(rows, row)
	}
	return rows
}

// NoColor reports whether color output is disabled (NO_COLOR env or
// colorblind_mode). When true, callers rely on the state symbols instead.
func NoColor() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}
