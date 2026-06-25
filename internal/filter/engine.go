// Package filter implements the socket filter engine (PRD FR-03).
//
// All parsers are pure functions operating on strings, and the Filter.Match
// predicate is pure over a netstat.SocketInfo — so the whole package is fully
// unit-testable on any platform.
package filter

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// Filter holds a set of ANDed conditions. A zero-value Filter matches everything.
type Filter struct {
	Ports      []uint16           // empty = any port
	Protocols  []netstat.Protocol // empty = any protocol
	States     []netstat.State    // empty = any state
	Process    string             // case-insensitive substring on process name
	PID        int                // 0 = any
	User       string             // case-insensitive substring on user
	Container  string             // case-insensitive substring on container name/id
	LocalCIDR  *net.IPNet         // nil = any local address
	RemoteCIDR *net.IPNet         // nil = any remote address
	Text       string             // free-text: matches process/user/PID/container
}

// IsEmpty reports whether the filter has no active conditions.
func (f Filter) IsEmpty() bool {
	return len(f.Ports) == 0 && len(f.Protocols) == 0 && len(f.States) == 0 &&
		f.Process == "" && f.PID == 0 && f.User == "" && f.Container == "" &&
		f.LocalCIDR == nil && f.RemoteCIDR == nil && f.Text == ""
}

// portSet builds a lookup set of the filter's ports.
func (f Filter) portSet() map[uint16]bool {
	m := make(map[uint16]bool, len(f.Ports))
	for _, p := range f.Ports {
		m[p] = true
	}
	return m
}

// Match reports whether a socket satisfies all active conditions (FR-03-07).
func (f Filter) Match(s netstat.SocketInfo) bool {
	if len(f.Ports) > 0 && !f.portSet()[s.LocalPort] {
		return false
	}
	if len(f.Protocols) > 0 && !containsProto(f.Protocols, s.Protocol) {
		return false
	}
	if len(f.States) > 0 {
		ok := false
		for _, st := range f.States {
			if st == s.State {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if f.PID != 0 && s.PID != f.PID {
		return false
	}
	if f.Process != "" && !ciContains(s.ProcessName, f.Process) {
		return false
	}
	if f.User != "" && !ciContains(s.User, f.User) {
		return false
	}
	if f.Container != "" && !ciContains(s.ContainerName, f.Container) && !ciContains(s.ContainerID, f.Container) {
		return false
	}
	if f.LocalCIDR != nil && !cidrContains(f.LocalCIDR, s.LocalAddr) {
		return false
	}
	if f.RemoteCIDR != nil && !cidrContains(f.RemoteCIDR, s.RemoteAddr) {
		return false
	}
	if f.Text != "" {
		if !ciContains(s.ProcessName, f.Text) &&
			!ciContains(s.User, f.Text) &&
			!ciContains(s.ContainerName, f.Text) &&
			!ciContains(s.ContainerID, f.Text) &&
			strconv.Itoa(s.PID) != f.Text {
			return false
		}
	}
	return true
}

// Apply returns the subset of sockets matching the filter, preserving order.
func Apply(socks []netstat.SocketInfo, f Filter) []netstat.SocketInfo {
	if f.IsEmpty() {
		out := make([]netstat.SocketInfo, len(socks))
		copy(out, socks)
		return out
	}
	out := make([]netstat.SocketInfo, 0, len(socks))
	for _, s := range socks {
		if f.Match(s) {
			out = append(out, s)
		}
	}
	return out
}

// Summary returns a compact, human-readable description of active filters for
// the filter bar (PRD FR-03-08).
func (f Filter) Summary() string {
	if f.IsEmpty() {
		return ""
	}
	var parts []string
	if len(f.Protocols) > 0 {
		names := make([]string, len(f.Protocols))
		for i, p := range f.Protocols {
			names[i] = string(p)
		}
		parts = append(parts, "proto:"+strings.Join(names, ","))
	}
	if len(f.States) > 0 {
		names := make([]string, len(f.States))
		for i, s := range f.States {
			names[i] = s.String()
		}
		parts = append(parts, "state:"+strings.Join(names, ","))
	}
	if len(f.Ports) > 0 {
		parts = append(parts, "port:"+PortRangeString(f.Ports))
	}
	if f.Process != "" {
		parts = append(parts, "proc:"+f.Process)
	}
	if f.PID != 0 {
		parts = append(parts, "pid:"+strconv.Itoa(f.PID))
	}
	if f.User != "" {
		parts = append(parts, "user:"+f.User)
	}
	if f.Container != "" {
		parts = append(parts, "container:"+f.Container)
	}
	if f.LocalCIDR != nil {
		parts = append(parts, "local:"+f.LocalCIDR.String())
	}
	if f.RemoteCIDR != nil {
		parts = append(parts, "remote:"+f.RemoteCIDR.String())
	}
	if f.Text != "" {
		parts = append(parts, "text:"+f.Text)
	}
	return strings.Join(parts, " ")
}

// --- helpers ---

func containsProto(list []netstat.Protocol, p netstat.Protocol) bool {
	for _, x := range list {
		if x == p {
			return true
		}
	}
	return false
}

// ciContains is a case-insensitive substring test.
func ciContains(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

// cidrContains reports whether addr falls under n. Non-IP or mismatched
// families return false. The address may include a zone; it is stripped.
func cidrContains(n *net.IPNet, addr string) bool {
	if n == nil || addr == "" {
		return false
	}
	ip := parseIP(addr)
	if ip == nil {
		return false
	}
	return n.Contains(ip)
}

// parseIP parses an address that may be "1.2.3.4", "::1", or "fe80::1%eth0".
func parseIP(addr string) net.IP {
	if i := strings.IndexByte(addr, '%'); i >= 0 {
		addr = addr[:i]
	}
	return net.ParseIP(addr)
}

// ParsePorts parses a comma-separated list of ports and ranges (FR-03-01):
// "80", "80,443", "8080-8090", "80,8080-8090,9000". Returns a deduplicated,
// sorted list.
func ParsePorts(input string) ([]uint16, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	seen := map[uint16]bool{}
	var out []uint16
	for _, seg := range strings.Split(input, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		if strings.Contains(seg, "-") {
			bounds := strings.SplitN(seg, "-", 2)
			lo, err := parsePort(bounds[0])
			if err != nil {
				return nil, err
			}
			hi, err := parsePort(bounds[1])
			if err != nil {
				return nil, err
			}
			if hi < lo {
				lo, hi = hi, lo
			}
			for p := lo; p <= hi; p++ {
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
		} else {
			p, err := parsePort(seg)
			if err != nil {
				return nil, err
			}
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func parsePort(s string) (uint16, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid port %q: %w", s, err)
	}
	if v > 65535 {
		return 0, fmt.Errorf("port out of range: %d", v)
	}
	return uint16(v), nil
}

// PortRangeString renders a port list compactly, collapsing consecutive runs
// into "lo-hi" ranges.
func PortRangeString(ports []uint16) string {
	if len(ports) == 0 {
		return ""
	}
	sorted := append([]uint16(nil), ports...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	var parts []string
	lo, hi := sorted[0], sorted[0]
	for _, p := range sorted[1:] {
		if p == hi+1 {
			hi = p
			continue
		}
		parts = append(parts, rangeStr(lo, hi))
		lo, hi = p, p
	}
	parts = append(parts, rangeStr(lo, hi))
	return strings.Join(parts, ",")
}

func rangeStr(lo, hi uint16) string {
	if lo == hi {
		return strconv.Itoa(int(lo))
	}
	return strconv.Itoa(int(lo)) + "-" + strconv.Itoa(int(hi))
}

// ParseProtocols parses a comma-separated protocol list (case-insensitive):
// "tcp", "tcp,udp", "tcp,udp,unix". Unknown tokens are rejected.
func ParseProtocols(input string) ([]netstat.Protocol, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	var out []netstat.Protocol
	for _, tok := range strings.Split(input, ",") {
		tok = strings.ToLower(strings.TrimSpace(tok))
		switch tok {
		case "tcp":
			out = append(out, netstat.ProtocolTCP)
		case "tcp6":
			out = append(out, netstat.ProtocolTCP6)
		case "udp":
			out = append(out, netstat.ProtocolUDP)
		case "udp6":
			out = append(out, netstat.ProtocolUDP6)
		case "raw":
			out = append(out, netstat.ProtocolRaw)
		case "raw6":
			out = append(out, netstat.ProtocolRaw6)
		case "unix", "ux":
			out = append(out, netstat.ProtocolUnix)
		default:
			return nil, fmt.Errorf("unknown protocol %q", tok)
		}
	}
	return out, nil
}

// ParseStates parses a comma-separated state list (case-insensitive), accepting
// both short (ss-style) and long names, e.g. "LISTEN", "ESTAB", "TIME_WAIT".
func ParseStates(input string) ([]netstat.State, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	var out []netstat.State
	for _, tok := range strings.Split(input, ",") {
		tok = strings.ToUpper(strings.TrimSpace(tok))
		st, ok := stateByName[tok]
		if !ok {
			return nil, fmt.Errorf("unknown state %q", tok)
		}
		out = append(out, st)
	}
	return out, nil
}

// stateByName maps state names (short and long) to State values.
var stateByName = map[string]netstat.State{
	"ESTAB": netstat.StateEstablished, "ESTABLISHED": netstat.StateEstablished,
	"SYN_SENT": netstat.StateSynSent,
	"SYN_RECV": netstat.StateSynRecv,
	"FIN_WAIT1": netstat.StateFinWait1,
	"FIN_WAIT2": netstat.StateFinWait2,
	"TIME_WAIT": netstat.StateTimeWait,
	"CLOSE": netstat.StateClose,
	"CLOSE_WAIT": netstat.StateCloseWait,
	"LAST_ACK": netstat.StateLastAck,
	"LISTEN": netstat.StateListen,
	"CLOSING": netstat.StateClosing,
	"UNCONN": netstat.StateUnconnected, "UNCONNECTED": netstat.StateUnconnected,
	"CONNECTING": netstat.StateConnecting,
	"CONNECTED": netstat.StateConnected,
}

// ParseCIDR parses a CIDR or bare IP (treated as /32 or /128).
func ParseCIDR(input string) (*net.IPNet, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	if !strings.Contains(input, "/") {
		ip := parseIP(input)
		if ip == nil {
			return nil, fmt.Errorf("invalid CIDR %q", input)
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
	}
	_, n, err := net.ParseCIDR(input)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", input, err)
	}
	return n, nil
}
