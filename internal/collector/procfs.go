package collector

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// ParseInet parses a /proc/net/{tcp,tcp6,udp,udp6,raw,raw6} file and returns
// one SocketInfo per socket line. proto determines IPv4 vs IPv6 decoding and
// is recorded on each entry.
func ParseInet(fs FS, path string, proto netstat.Protocol) ([]netstat.SocketInfo, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseInetBytes(data, proto)
}

// ParseInetBytes parses the content of an inet /proc/net file.
func ParseInetBytes(data []byte, proto netstat.Protocol) ([]netstat.SocketInfo, error) {
	var out []netstat.SocketInfo
	for _, line := range splitLines(data) {
		f := strings.Fields(line)
		if len(f) < 10 {
			continue // header or malformed
		}
		if f[0] == "sl" || !strings.Contains(f[0], ":") {
			continue
		}
		localAddr, localPort, ok := parseAddrPort(f[1], proto.IsIPv6())
		if !ok {
			continue
		}
		remoteAddr, remotePort, ok := parseAddrPort(f[2], proto.IsIPv6())
		if !ok {
			continue
		}
		state := netstat.TCPStateFromHex(f[3])
		uid, _ := parseUint(f[7])
		inode, _ := parseInode(f[9])

		out = append(out, netstat.SocketInfo{
			Protocol:   proto,
			LocalAddr:  localAddr,
			LocalPort:  localPort,
			RemoteAddr: remoteAddr,
			RemotePort: remotePort,
			State:      state,
			UID:        uid,
			Inode:      inode,
		})
	}
	return out, nil
}

// ParseUnix parses /proc/net/unix and returns one SocketInfo per socket.
func ParseUnix(fs FS, path string) ([]netstat.SocketInfo, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseUnixBytes(data)
}

// ParseUnixBytes parses the content of /proc/net/unix.
func ParseUnixBytes(data []byte) ([]netstat.SocketInfo, error) {
	var out []netstat.SocketInfo
	for _, line := range splitLines(data) {
		if strings.HasPrefix(strings.TrimSpace(line), "Num") {
			continue
		}
		// Need at least 7 fields; path (8th) is optional.
		fields, rest := fieldsN(line, 7)
		if len(fields) < 7 {
			continue
		}
		// fields[0] = "Num:" ; fields[1]=RefCount ; fields[2]=Protocol ;
		// fields[3]=Flags ; fields[4]=Type ; fields[5]=St ; fields[6]=Inode
		inode, _ := parseInode(fields[6])
		ref, _ := parseUint(fields[1])
		path := strings.TrimSpace(rest)
		out = append(out, netstat.SocketInfo{
			Protocol:  netstat.ProtocolUnix,
			State:     netstat.UnixStateFromCode(fields[5]),
			UnixType:  unixTypeFromHex(fields[4]),
			Inode:     inode,
			RefCount:  ref,
			Path:      path,
			LocalAddr: path,
		})
	}
	return out, nil
}

// unixTypeFromHex maps the /proc/net/unix Type field to a human label.
func unixTypeFromHex(hex string) string {
	switch strings.ToLower(hex) {
	case "0001":
		return "STREAM"
	case "0002":
		return "DGRAM"
	case "0003":
		return "RAW"
	case "0005":
		return "SEQPACKET"
	default:
		return hex
	}
}

// parseAddrPort decodes an "address:port" field from /proc/net/inet files.
// For IPv4 the address is 8 hex chars (host byte order); for IPv6 it is 32
// hex chars (4 host-order words). Returns the textual IP and numeric port.
func parseAddrPort(s string, ipv6 bool) (string, uint16, bool) {
	colon := strings.IndexByte(s, ':')
	if colon < 0 {
		return "", 0, false
	}
	addrHex := s[:colon]
	portHex := s[colon+1:]
	port, err := strconv.ParseUint(portHex, 16, 32)
	if err != nil {
		return "", 0, false
	}
	var ip string
	if ipv6 {
		ip = decodeIPv6(addrHex)
	} else {
		ip = decodeIPv4(addrHex)
	}
	if ip == "" {
		return "", 0, false
	}
	return ip, uint16(port), true
}

// decodeIPv4 decodes 8 hex chars (host byte order) to dotted notation.
func decodeIPv4(h string) string {
	if len(h) != 8 {
		return ""
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return ""
	}
	// Host byte order: low byte first.
	return itoaByte(byte(v)) + "." +
		itoaByte(byte(v>>8)) + "." +
		itoaByte(byte(v>>16)) + "." +
		itoaByte(byte(v>>24))
}

// decodeIPv6 decodes 32 hex chars (4 host-order words) to colon notation.
func decodeIPv6(h string) string {
	if len(h) != 32 {
		return ""
	}
	var raw [16]byte
	for i := 0; i < 4; i++ {
		word, err := strconv.ParseUint(h[i*8:(i+1)*8], 16, 32)
		if err != nil {
			return ""
		}
		raw[i*4+0] = byte(word)
		raw[i*4+1] = byte(word >> 8)
		raw[i*4+2] = byte(word >> 16)
		raw[i*4+3] = byte(word >> 24)
	}
	return formatIPv6(raw)
}

// itoaByte is a small allocation-free byte-to-string integer converter.
func itoaByte(b byte) string {
	if b == 0 {
		return "0"
	}
	var buf [3]byte
	n := 0
	for b > 0 {
		buf[n] = '0' + b%10
		b /= 10
		n++
	}
	// reverse
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf[:n])
}

// formatIPv6 renders a 16-byte address using net.IP-equivalent logic without
// importing net (keeps this package dependency-light and fully deterministic).
func formatIPv6(raw [16]byte) string {
	// Build the 8 groups.
	groups := make([]uint16, 8)
	for i := 0; i < 8; i++ {
		groups[i] = uint16(raw[i*2])<<8 | uint16(raw[i*2+1])
	}

	// IPv4-mapped (::ffff:a.b.c.d)
	isV4Mapped := groups[0] == 0 && groups[1] == 0 && groups[2] == 0 && groups[3] == 0 &&
		groups[4] == 0 && groups[5] == 0xffff
	if isV4Mapped {
		a, b, c, d := raw[12], raw[13], raw[14], raw[15]
		return "::ffff:" + itoaByte(a) + "." + itoaByte(b) + "." + itoaByte(c) + "." + itoaByte(d)
	}

	// Find longest run of zero groups for "::" compression.
	bestStart, bestLen := -1, 0
	curStart, curLen := -1, 0
	for i, g := range groups {
		if g == 0 {
			if curStart < 0 {
				curStart = i
			}
			curLen++
			if curLen > bestLen {
				bestLen = curLen
				bestStart = curStart
			}
		} else {
			curStart, curLen = -1, 0
		}
	}
	// Only compress runs of length >= 2.
	if bestLen < 2 {
		bestStart = -1
	}
	return normalizeIPv6(groups, bestStart, bestLen)
}

// normalizeIPv6 is a clear, correct renderer with "::" compression. It
// tracks whether the previously emitted token was "::" so that no spurious
// leading colon is added to the following group.
func normalizeIPv6(groups []uint16, zstart, zlen int) string {
	var sb strings.Builder
	i := 0
	afterDoubleColon := false
	for i < 8 {
		if i == zstart && zlen >= 2 {
			sb.WriteString("::")
			i += zlen
			afterDoubleColon = true
			continue
		}
		if i > 0 && !afterDoubleColon {
			sb.WriteByte(':')
		}
		afterDoubleColon = false
		sb.WriteString(strconv.FormatUint(uint64(groups[i]), 16))
		i++
	}
	return sb.String()
}

// parseUint parses a base-10 uint32.
func parseUint(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// parseInode parses an inode field which is decimal in /proc/net/*, with a
// hex fallback for robustness against unusual kernels.
func parseInode(s string) (uint64, error) {
	if v, err := strconv.ParseUint(s, 10, 64); err == nil {
		return v, nil
	}
	return strconv.ParseUint(s, 16, 64)
}

// splitLines splits on \n, dropping a trailing empty line and \r.
func splitLines(data []byte) []string {
	data = bytes.TrimRight(data, "\n")
	if len(data) == 0 {
		return nil
	}
	parts := strings.Split(string(data), "\n")
	for i := range parts {
		parts[i] = strings.TrimRight(parts[i], "\r")
	}
	return parts
}

// fieldsN splits line into up to n whitespace-delimited fields and returns the
// remainder (untouched) after the nth field. Used for /proc/net/unix where the
// Path field may contain spaces.
func fieldsN(line string, n int) ([]string, string) {
	line = strings.TrimLeft(line, " \t")
	fields := make([]string, 0, n)
	i := 0
	for len(fields) < n {
		// skip leading whitespace
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) {
			break
		}
		start := i
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			i++
		}
		fields = append(fields, line[start:i])
	}
	remainder := ""
	if i < len(line) {
		remainder = strings.TrimLeft(line[i:], " \t")
	}
	return fields, remainder
}
