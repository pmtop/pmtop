package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// tcpFixture mirrors the real /proc/net/tcp layout (little-endian addresses).
const tcpFixture = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 1B01A8C0:B9A6 00000000:0000 0A 00000000:00000000 00:00000000 00000000   999        0 29170 1 0000000000000000 100 0 0 10 0
   1: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 9580 1 0000000000000000 100 0 0 10 0
   2: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 8412 1 0000000000000000 100 0 0 10 0
   3: 0100007F:1F90 0100007F:04D2 01 00000000:00000000 00:00000000 00000000     0        0 11111 1 0000000000000000 100 0 0 10 0
`

func TestParseInetBytes_TCP(t *testing.T) {
	socks, err := ParseInetBytes([]byte(tcpFixture), netstat.ProtocolTCP)
	require.NoError(t, err)
	require.Len(t, socks, 4)

	assert.Equal(t, netstat.ProtocolTCP, socks[0].Protocol)
	assert.Equal(t, "192.168.1.27", socks[0].LocalAddr)
	assert.Equal(t, uint16(47526), socks[0].LocalPort)
	assert.Equal(t, "0.0.0.0", socks[0].RemoteAddr)
	assert.Equal(t, netstat.StateListen, socks[0].State)
	assert.Equal(t, uint32(999), socks[0].UID)
	assert.Equal(t, uint64(29170), socks[0].Inode)

	assert.Equal(t, "0.0.0.0", socks[1].LocalAddr)
	assert.Equal(t, uint16(80), socks[1].LocalPort)
	assert.Equal(t, netstat.StateListen, socks[1].State)
	assert.Equal(t, uint64(9580), socks[1].Inode)

	assert.Equal(t, uint16(22), socks[2].LocalPort)

	// Established row: 127.0.0.1:8080 <-> 127.0.0.1:1234
	assert.Equal(t, "127.0.0.1", socks[3].LocalAddr)
	assert.Equal(t, uint16(8080), socks[3].LocalPort)
	assert.Equal(t, "127.0.0.1", socks[3].RemoteAddr)
	assert.Equal(t, uint16(1234), socks[3].RemotePort)
	assert.Equal(t, netstat.StateEstablished, socks[3].State)
}

func TestParseInetBytes_HeaderAndShortLines(t *testing.T) {
	in := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
short line
0: bad
   0: 0100007F:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 5 1 0
`
	socks, err := ParseInetBytes([]byte(in), netstat.ProtocolTCP)
	require.NoError(t, err)
	require.Len(t, socks, 1)
	assert.Equal(t, uint16(80), socks[0].LocalPort)
}

func TestParseInet_ReadFileError(t *testing.T) {
	fs := newFakeFS()
	_, err := ParseInet(fs, "/proc/net/tcp", netstat.ProtocolTCP)
	assert.Error(t, err)
}

func TestParseInetBytes_UDP6(t *testing.T) {
	// fd00:4f9:f8d1:f52c:: port 53 (0x35) -> little-endian 32-hex.
	// Real sample: F90400FD2CF5D1F8FF290C02592489FE -> fd00:4f9:f8d1:f52c:20c:29ff:fe89:2459
	in := `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: F90400FD2CF5D1F8FF290C02592489FE:0035 00000000000000000000000000000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 4242 1 0
`
	socks, err := ParseInetBytes([]byte(in), netstat.ProtocolUDP6)
	require.NoError(t, err)
	require.Len(t, socks, 1)
	assert.Equal(t, "fd00:4f9:f8d1:f52c:20c:29ff:fe89:2459", socks[0].LocalAddr)
	assert.Equal(t, uint16(53), socks[0].LocalPort)
	assert.Equal(t, netstat.ProtocolUDP6, socks[0].Protocol)
	assert.Equal(t, netstat.StateClose, socks[0].State) // 07 -> Close for UDP
	assert.Equal(t, uint64(4242), socks[0].Inode)
}

func TestDecodeIPv4(t *testing.T) {
	cases := map[string]string{
		"0100007F": "127.0.0.1",
		"00000000": "0.0.0.0",
		"1B01A8C0": "192.168.1.27",
		"FFFFFFFF": "255.255.255.255",
	}
	for hex, want := range cases {
		assert.Equal(t, want, decodeIPv4(hex), hex)
	}
	assert.Equal(t, "", decodeIPv4("bad"), "bad hex")
	assert.Equal(t, "", decodeIPv4("123"), "wrong length")
}

func TestDecodeIPv6(t *testing.T) {
	// ::1: 16 zero bytes except last = 01. As 4 little-endian words the last
	// word is 0x01000000 -> "01000000".
	assert.Equal(t, "::1", decodeIPv6("00000000000000000000000001000000"))
	// ::ffff:127.0.0.1 (v4-mapped): bytes ...00 00 ff ff 7f 00 00 01.
	// word2 = 0xffff0000 -> "FFFF0000", word3 = 0x0100007f -> "0100007F".
	assert.Equal(t, "::ffff:127.0.0.1", decodeIPv6("0000000000000000FFFF00000100007F"))
	// fe80::20c:29ff:fe89:2459 (from real VM sample)
	assert.Equal(t, "fe80::20c:29ff:fe89:2459", decodeIPv6("000080FE00000000FF290C02592489FE"))
	assert.Equal(t, "", decodeIPv6("short"), "bad length")
}

func TestParseUnixBytes(t *testing.T) {
	in := `Num       RefCount Protocol Flags    Type St Inode Path
0000000000000000: 00000003 00000000 00000000 0002 03 1838361
0000000000000000: 00000002 00000000 00000000 0001 01 1838348 /var/run/docker.sock
0000000000000000: 00000001 00000000 00000000 0005 01 999 /has space/path
`
	socks, err := ParseUnixBytes([]byte(in))
	require.NoError(t, err)
	require.Len(t, socks, 3)

	assert.Equal(t, netstat.ProtocolUnix, socks[0].Protocol)
	assert.Equal(t, "DGRAM", socks[0].UnixType)
	assert.Equal(t, netstat.StateConnected, socks[0].State) // 03
	assert.Equal(t, uint64(1838361), socks[0].Inode)
	assert.Empty(t, socks[0].Path)

	assert.Equal(t, "STREAM", socks[1].UnixType)
	assert.Equal(t, netstat.StateUnconnected, socks[1].State) // 01
	assert.Equal(t, "/var/run/docker.sock", socks[1].Path)
	assert.Equal(t, "/var/run/docker.sock", socks[1].LocalAddr)

	assert.Equal(t, "SEQPACKET", socks[2].UnixType)
	assert.Equal(t, "/has space/path", socks[2].Path, "path with spaces preserved")
}

func TestParseUnix_ReadFileError(t *testing.T) {
	fs := newFakeFS()
	_, err := ParseUnix(fs, "/proc/net/unix")
	assert.Error(t, err)
}

func TestUnixTypeFromHex(t *testing.T) {
	assert.Equal(t, "STREAM", unixTypeFromHex("0001"))
	assert.Equal(t, "DGRAM", unixTypeFromHex("0002"))
	assert.Equal(t, "RAW", unixTypeFromHex("0003"))
	assert.Equal(t, "SEQPACKET", unixTypeFromHex("0005"))
	// Unknown types are returned unchanged (original case preserved).
	assert.Equal(t, "00FF", unixTypeFromHex("00FF"))
}

func TestParseAddrPort_Bad(t *testing.T) {
	_, _, ok := parseAddrPort("noport", false)
	assert.False(t, ok)
	_, _, ok = parseAddrPort("zz:gg", false)
	assert.False(t, ok)
	_, _, ok = parseAddrPort("badhex:0016", false)
	assert.False(t, ok)
}

func TestSplitLines(t *testing.T) {
	out := splitLines([]byte("a\nb\n"))
	assert.Len(t, out, 2)
	assert.Equal(t, "a", out[0])
	assert.Equal(t, "b", out[1])

	assert.Empty(t, splitLines([]byte("")))
	// CRLF stripping
	out = splitLines([]byte("a\r\nb\r\n"))
	assert.Equal(t, "a", out[0])
	assert.Equal(t, "b", out[1])
}

func TestFieldsN(t *testing.T) {
	fields, rest := fieldsN("a b c d e f g h i j", 7)
	assert.Len(t, fields, 7)
	assert.Equal(t, "a", fields[0])
	assert.Equal(t, "g", fields[6])
	assert.Equal(t, "h i j", rest)

	// fewer than n fields
	fields, rest = fieldsN("a b", 7)
	assert.Len(t, fields, 2)
	assert.Empty(t, rest)
}
