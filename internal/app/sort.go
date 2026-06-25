package app

import (
	"sort"
	"strconv"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// SortKey identifies the column used to sort the socket table (PRD FR-01-05).
type SortKey int

const (
	SortProto SortKey = iota
	SortLocal
	SortPort
	SortRemote
	SortState
	SortPID
	SortProcess
	SortContainer
)

// sortKeys lists the keys in the order 's' cycles through them.
var sortKeys = []SortKey{SortProto, SortPort, SortState, SortPID, SortProcess, SortLocal, SortRemote, SortContainer}

// String returns the column label.
func (k SortKey) String() string {
	switch k {
	case SortProto:
		return "Proto"
	case SortLocal:
		return "Local"
	case SortPort:
		return "Port"
	case SortRemote:
		return "Remote"
	case SortState:
		return "State"
	case SortPID:
		return "PID"
	case SortProcess:
		return "Process"
	case SortContainer:
		return "Container"
	default:
		return "?"
	}
}

// next returns the next sort key in the cycle.
func (k SortKey) next() SortKey {
	for i, sk := range sortKeys {
		if sk == k && i+1 < len(sortKeys) {
			return sortKeys[i+1]
		}
	}
	return sortKeys[0]
}

// SortSockets sorts socks in place by the given column and direction. A stable
// sort is used so that equal keys keep their collection order.
func SortSockets(socks []netstat.SocketInfo, key SortKey, asc bool) {
	less := func(i, j int) bool { return socketLess(socks[i], socks[j], key) }
	if !asc {
		less = func(i, j int) bool { return socketLess(socks[j], socks[i], key) }
	}
	sort.SliceStable(socks, less)
}

// socketLess compares two sockets by the given column.
func socketLess(a, b netstat.SocketInfo, key SortKey) bool {
	switch key {
	case SortProto:
		if a.Protocol != b.Protocol {
			return a.Protocol < b.Protocol
		}
		return a.LocalPort < b.LocalPort
	case SortLocal:
		if a.LocalAddr != b.LocalAddr {
			return a.LocalAddr < b.LocalAddr
		}
		return a.LocalPort < b.LocalPort
	case SortPort:
		return a.LocalPort < b.LocalPort
	case SortRemote:
		if a.RemoteAddr != b.RemoteAddr {
			return a.RemoteAddr < b.RemoteAddr
		}
		return a.RemotePort < b.RemotePort
	case SortState:
		return a.State.String() < b.State.String()
	case SortPID:
		return a.PID < b.PID
	case SortProcess:
		return a.ProcessName < b.ProcessName
	case SortContainer:
		return a.ContainerName < b.ContainerName
	default:
		return false
	}
}

// formatPort returns "addr:port" for inet sockets or the unix path.
func formatPort(s netstat.SocketInfo) string {
	if s.Protocol == netstat.ProtocolUnix {
		return s.Path
	}
	return s.LocalAddr + ":" + strconv.Itoa(int(s.LocalPort))
}
