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

// NumColumns is the total number of columns in the table.
const NumColumns = 8

// BuildColumns returns the table column set fit to the given total width.
// Narrow terminals progressively shrink the wider text columns.
func BuildColumns(width int) []table.Column {
	proto := 9
	state := 11
	pid := 7
	local := 22
	remote := 22
	process := 16
	user := 10
	container := 12

	fixed := proto + state + pid
	separators := (NumColumns - 1) * 3
	avail := width - fixed - separators
	if avail < 40 {
		avail = 40
	}
	local = clamp(local, avail/5)
	remote = clamp(remote, avail/5)
	process = clamp(process, avail/3)
	user = clamp(user, avail/6)
	container = clamp(container, avail/6)
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

// totalWidth computes the full rendered width of all columns plus separators.
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
// When showService is true, known ports are replaced with their service name.
func localCell(s netstat.SocketInfo, showService bool) string {
	if s.Protocol == netstat.ProtocolUnix {
		if s.Path == "" {
			return "(anonymous)"
		}
		return s.Path
	}
	return formatEndpoint(s.LocalAddr, s.LocalPort, showService)
}

// remoteCell renders the remote endpoint.
func remoteCell(s netstat.SocketInfo, showService bool) string {
	if s.Protocol == netstat.ProtocolUnix {
		return "-"
	}
	if s.RemoteAddr == "" || (s.RemoteAddr == "0.0.0.0" && s.RemotePort == 0) {
		return "*"
	}
	return formatEndpoint(s.RemoteAddr, s.RemotePort, showService)
}

// formatEndpoint renders "addr:port" or "addr:service" when showService is true.
func formatEndpoint(addr string, port uint16, showService bool) string {
	if showService {
		if name, ok := serviceName(port); ok {
			return addr + ":" + name
		}
	}
	return addr + ":" + strconv.Itoa(int(port))
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

// RowOptions controls row rendering behavior (display modes).
type RowOptions struct {
	ShowService bool // p key: show service names instead of port numbers
}

// RowsFromSockets converts sockets into table rows in column order.
func RowsFromSockets(socks []netstat.SocketInfo, style *Style, opts RowOptions) []table.Row {
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
			localCell(s, opts.ShowService),
			remoteCell(s, opts.ShowService),
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

// NoColor reports whether color output is disabled (NO_COLOR env).
func NoColor() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}

// serviceName maps common port numbers to their service names.
var serviceNameMap = map[uint16]string{
	20: "ftp-data", 21: "ftp", 22: "ssh", 23: "telnet", 25: "smtp",
	53: "dns", 67: "dhcp", 68: "dhcp", 80: "http", 110: "pop3",
	143: "imap", 443: "https", 465: "smtps", 587: "submission",
	993: "imaps", 995: "pop3s", 3306: "mysql", 5432: "pgsql",
	6379: "redis", 8080: "http-alt", 8443: "https-alt",
	9090: "prom", 9200: "es", 11211: "memcache", 27017: "mongo",
}

// serviceName returns the service name for a known port.
func serviceName(port uint16) (string, bool) {
	name, ok := serviceNameMap[port]
	return name, ok
}

// ColumnTitleForSort returns the title for a column given the sort key index
// and direction, appending a ▲ or ▼ indicator to the sorted column.
func ColumnTitleForSort(base string, isSorted bool, asc bool) string {
	if !isSorted {
		return base
	}
	if asc {
		return base + " ▲"
	}
	return base + " ▼"
}

// SortColumnIndex maps an app-level sort key index to the table column index.
// Returns -1 if the sort key has no corresponding column.
func SortColumnIndex(sortKeyIdx int) int {
	switch sortKeyIdx {
	case 0: // SortProto
		return ColProto
	case 1: // SortLocal
		return ColLocal
	case 2: // SortPort
		return ColLocal // port sorts by local column
	case 3: // SortRemote
		return ColRemote
	case 4: // SortState
		return ColState
	case 5: // SortPID
		return ColPID
	case 6: // SortProcess
		return ColProcess
	case 7: // SortContainer
		return ColContainer
	default:
		return -1
	}
}
