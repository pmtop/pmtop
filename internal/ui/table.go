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

// Column identifiers (indices into the column set). The left table shows only
// these 5 columns; PID/Process/User/Container live in the right detail panel.
const (
	ColSeq = iota // 0: row sequence number
	ColProto      // 1: protocol + state symbol
	ColLocal      // 2: local address:port
	ColRemote     // 3: remote address:port
	ColState      // 4: state name
)

// NumColumns is the total number of columns in the left table.
const NumColumns = 5

// BuildColumns returns the table column set fit to the given total width.
// The left table is narrower than the terminal (terminal - rightPanelWidth),
// so columns are compact: #, Proto, Local, Remote, State.
func BuildColumns(width int) []table.Column {
	seq := 4
	proto := 6
	state := 9
	fixed := seq + proto + state
	separators := (NumColumns - 1) * 3
	avail := width - fixed - separators
	if avail < 8 {
		avail = 8
	}
	local := avail / 2
	remote := avail - local
	// Trim to fit within width.
	for fixed+separators+local+remote > width && width > 0 {
		switch {
		case remote > 4:
			remote--
		case local > 4:
			local--
		default:
			goto done
		}
	}
done:
	return []table.Column{
		{Title: "#", Width: seq},
		{Title: "Proto", Width: proto},
		{Title: "Local", Width: local},
		{Title: "Remote", Width: remote},
		{Title: "State", Width: state},
	}
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
			return "(anon)"
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

// RowOptions controls row rendering behavior (display modes).
type RowOptions struct {
	ShowService bool // p key: show service names instead of port numbers
}

// RowsFromSockets converts sockets into table rows with sequence numbers.
// Each row has exactly NumColumns elements.
func RowsFromSockets(socks []netstat.SocketInfo, style *Style, opts RowOptions) []table.Row {
	rows := make([]table.Row, 0, len(socks))
	for i, s := range socks {
		row := table.Row{
			strconv.Itoa(i + 1),            // #
			protoCell(s),                   // Proto
			localCell(s, opts.ShowService),  // Local
			remoteCell(s, opts.ShowService), // Remote
			stateCell(s),                   // State
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
		return base + "▲"
	}
	return base + "▼"
}

// SortColumnIndex maps an app-level sort key index to the table column index.
// Returns -1 if the sort key has no corresponding column (PID/Process/Container
// are in the detail panel, not the table).
func SortColumnIndex(sortKeyIdx int) int {
	switch sortKeyIdx {
	case 0: // SortProto
		return ColProto
	case 1: // SortLocal
		return ColLocal
	case 2: // SortPort
		return ColLocal
	case 3: // SortRemote
		return ColRemote
	case 4: // SortState
		return ColState
	default: // SortPID, SortProcess, SortContainer — not in table
		return -1
	}
}
