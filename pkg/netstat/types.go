// Package netstat provides pure Go data structures describing network
// sockets, mirroring the information exposed by the Linux /proc/net/* and
// /proc/<pid>/ interfaces. It intentionally contains no I/O logic so it can
// be reused by the collector, the CLI, and external consumers.
package netstat

// Protocol identifies the socket protocol family.
type Protocol string

const (
	ProtocolTCP  Protocol = "tcp"
	ProtocolTCP6 Protocol = "tcp6"
	ProtocolUDP  Protocol = "udp"
	ProtocolUDP6 Protocol = "udp6"
	ProtocolRaw  Protocol = "raw"
	ProtocolRaw6 Protocol = "raw6"
	ProtocolUnix Protocol = "unix"
)

// IsTCP reports whether the protocol is TCP (v4 or v6).
func (p Protocol) IsTCP() bool { return p == ProtocolTCP || p == ProtocolTCP6 }

// IsUDP reports whether the protocol is UDP (v4 or v6).
func (p Protocol) IsUDP() bool { return p == ProtocolUDP || p == ProtocolUDP6 }

// IsIPv6 reports whether the protocol is the IPv6 variant.
func (p Protocol) IsIPv6() bool {
	switch p {
	case ProtocolTCP6, ProtocolUDP6, ProtocolRaw6:
		return true
	default:
		return false
	}
}

// State is a TCP/UDP/Unix socket state as reported by /proc/net.
type State uint8

const (
	StateUnknown State = iota
	StateEstablished
	StateSynSent
	StateSynRecv
	StateFinWait1
	StateFinWait2
	StateTimeWait
	StateClose
	StateCloseWait
	StateLastAck
	StateListen
	StateClosing
	StateUnconnected // Unix-domain: SS_UNCONNECTED
	StateConnecting  // Unix-domain: SS_CONNECTING
	StateConnected   // Unix-domain: SS_CONNECTED
)

// String returns the canonical state name used by ss/netstat.
func (s State) String() string {
	switch s {
	case StateEstablished:
		return "ESTAB"
	case StateSynSent:
		return "SYN_SENT"
	case StateSynRecv:
		return "SYN_RECV"
	case StateFinWait1:
		return "FIN_WAIT1"
	case StateFinWait2:
		return "FIN_WAIT2"
	case StateTimeWait:
		return "TIME_WAIT"
	case StateClose:
		return "CLOSE"
	case StateCloseWait:
		return "CLOSE_WAIT"
	case StateLastAck:
		return "LAST_ACK"
	case StateListen:
		return "LISTEN"
	case StateClosing:
		return "CLOSING"
	case StateUnconnected:
		return "UNCONN"
	case StateConnecting:
		return "CONNECTING"
	case StateConnected:
		return "CONNECTED"
	default:
		return "UNKNOWN"
	}
}

// LongString returns a less abbreviated state name for human display.
func (s State) LongString() string {
	switch s {
	case StateEstablished:
		return "ESTABLISHED"
	case StateUnconnected:
		return "UNCONNECTED"
	default:
		return s.String()
	}
}

// Symbol returns a single-glyph state indicator for colorblind-accessible
// display (PRD FR-10-02, section 6.1). "-" is used for protocols/states
// that have no meaningful state (UDP, raw).
func (s State) Symbol() string {
	switch s {
	case StateListen:
		return "▶"
	case StateEstablished, StateConnected:
		return "●"
	case StateTimeWait:
		return "▲"
	case StateCloseWait:
		return "▼"
	case StateSynSent:
		return "◆"
	case StateClosing:
		return "◀"
	default:
		return "-"
	}
}

// TCPStateFromHex maps a /proc/net/tcp "st" hex code to a State.
func TCPStateFromHex(hex string) State {
	switch hex {
	case "01":
		return StateEstablished
	case "02":
		return StateSynSent
	case "03":
		return StateSynRecv
	case "04":
		return StateFinWait1
	case "05":
		return StateFinWait2
	case "06":
		return StateTimeWait
	case "07":
		return StateClose
	case "08":
		return StateCloseWait
	case "09":
		return StateLastAck
	case "0A", "0a":
		return StateListen
	case "0B", "0b":
		return StateClosing
	default:
		return StateUnknown
	}
}

// UnixStateFromCode maps a /proc/net/unix "St" decimal code to a State.
func UnixStateFromCode(code string) State {
	switch code {
	case "01":
		return StateUnconnected
	case "02":
		return StateConnecting
	case "03":
		return StateConnected
	default:
		return StateUnknown
	}
}

// SocketInfo describes a single socket plus its enriched ownership metadata.
// The socket-level fields (Protocol..Inode/Path/UnixType) are populated by the
// procfs parser; the ownership fields (PID..NetNS) are filled by the collector
// during enrichment and may be zero/empty when unavailable.
type SocketInfo struct {
	// --- socket-level (from /proc/net/*) ---
	Protocol   Protocol
	LocalAddr  string // dotted/colon notation, or "" for unix
	LocalPort  uint16
	RemoteAddr string
	RemotePort uint16
	State      State
	UID        uint32
	Inode      uint64
	RefCount   uint32

	// Unix-domain specific
	UnixType string // STREAM, DGRAM, SEQPACKET, RAW
	Path     string

	// --- enrichment (from /proc/<pid>) ---
	PID          int
	ProcessName  string
	User         string
	Runtime      string // docker, containerd, podman, crio, ""
	ContainerID  string
	ContainerName string
	NetNS         string
}
