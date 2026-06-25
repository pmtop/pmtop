package filter

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/pkg/netstat"
)

func TestParsePorts(t *testing.T) {
	cases := []struct {
		in   string
		want []uint16
	}{
		{"80", []uint16{80}},
		{"80,443", []uint16{80, 443}},
		{"8080-8082", []uint16{8080, 8081, 8082}},
		{"80,8080-8081,9000", []uint16{80, 8080, 8081, 9000}},
		{"8082-8080", []uint16{8080, 8081, 8082}}, // reversed range normalized
		{"80,80", []uint16{80}},             // dedup
		{"", nil},
		{"  53  ,  80  ", []uint16{53, 80}}, // whitespace tolerant
	}
	for _, c := range cases {
		got, err := ParsePorts(c.in)
		require.NoError(t, err, c.in)
		assert.Equal(t, c.want, got, c.in)
	}
}

func TestParsePorts_Errors(t *testing.T) {
	_, err := ParsePorts("abc")
	assert.Error(t, err)
	_, err = ParsePorts("80-abc")
	assert.Error(t, err)
	_, err = ParsePorts("70000")
	assert.Error(t, err)
}

func TestPortRangeString(t *testing.T) {
	assert.Equal(t, "80", PortRangeString([]uint16{80}))
	assert.Equal(t, "80,443", PortRangeString([]uint16{443, 80}))
	assert.Equal(t, "8080-8082,9000", PortRangeString([]uint16{8080, 8081, 8082, 9000}))
	assert.Equal(t, "", PortRangeString(nil))
}

func TestParseProtocols(t *testing.T) {
	p, err := ParseProtocols("tcp,udp,unix")
	require.NoError(t, err)
	assert.Equal(t, []netstat.Protocol{netstat.ProtocolTCP, netstat.ProtocolUDP, netstat.ProtocolUnix}, p)

	p, err = ParseProtocols("TCP6")
	require.NoError(t, err)
	assert.Equal(t, []netstat.Protocol{netstat.ProtocolTCP6}, p)

	_, err = ParseProtocols("foo")
	assert.Error(t, err)
}

func TestParseStates(t *testing.T) {
	s, err := ParseStates("LISTEN,ESTAB")
	require.NoError(t, err)
	assert.Equal(t, []netstat.State{netstat.StateListen, netstat.StateEstablished}, s)

	s, err = ParseStates("time_wait,close_wait")
	require.NoError(t, err)
	assert.Equal(t, []netstat.State{netstat.StateTimeWait, netstat.StateCloseWait}, s)

	_, err = ParseStates("BOGUS")
	assert.Error(t, err)
}

func TestParseCIDR(t *testing.T) {
	n, err := ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)
	assert.True(t, n.Contains(net.ParseIP("192.168.1.50")))
	assert.False(t, n.Contains(net.ParseIP("10.0.0.1")))

	// Bare IP -> /32.
	n, err = ParseCIDR("1.2.3.4")
	require.NoError(t, err)
	assert.True(t, n.Contains(net.ParseIP("1.2.3.4")))

	// Bare IPv6 -> /128.
	n, err = ParseCIDR("::1")
	require.NoError(t, err)
	assert.True(t, n.Contains(net.ParseIP("::1")))

	_, err = ParseCIDR("not-a-cidr")
	assert.Error(t, err)
	_, err = ParseCIDR("10.0.0.0/33")
	assert.Error(t, err)
}

func TestFilter_Match(t *testing.T) {
	socks := []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 22, State: netstat.StateListen, PID: 100, ProcessName: "sshd", User: "root"},
		{Protocol: netstat.ProtocolTCP, LocalAddr: "127.0.0.1", LocalPort: 8080, State: netstat.StateListen, PID: 200, ProcessName: "nginx", User: "www-data", ContainerID: "abc123def456"},
		{Protocol: netstat.ProtocolUDP, LocalAddr: "0.0.0.0", LocalPort: 53, State: netstat.StateClose, PID: 300, ProcessName: "dnsmasq", User: "dnsmasq"},
	}

	// Empty filter matches all.
	out := Apply(socks, Filter{})
	assert.Len(t, out, 3)

	// Port filter.
	out = Apply(socks, Filter{Ports: []uint16{8080}})
	assert.Len(t, out, 1)
	assert.Equal(t, "nginx", out[0].ProcessName)

	// Protocol filter.
	out = Apply(socks, Filter{Protocols: []netstat.Protocol{netstat.ProtocolUDP}})
	assert.Len(t, out, 1)
	assert.Equal(t, "dnsmasq", out[0].ProcessName)

	// State filter.
	out = Apply(socks, Filter{States: []netstat.State{netstat.StateListen}})
	assert.Len(t, out, 2)

	// Process fuzzy (case-insensitive).
	out = Apply(socks, Filter{Process: "NGIN"})
	assert.Len(t, out, 1)

	// PID filter.
	out = Apply(socks, Filter{PID: 300})
	assert.Len(t, out, 1)

	// User fuzzy.
	out = Apply(socks, Filter{User: "root"})
	assert.Len(t, out, 1)

	// Container fuzzy (matches id).
	out = Apply(socks, Filter{Container: "abc123"})
	assert.Len(t, out, 1)

	// CIDR on local address.
	cidr, _ := ParseCIDR("127.0.0.0/8")
	out = Apply(socks, Filter{LocalCIDR: cidr})
	assert.Len(t, out, 1)
	assert.Equal(t, "nginx", out[0].ProcessName)

	// Text free-form matches PID string.
	out = Apply(socks, Filter{Text: "200"})
	assert.Len(t, out, 1)
	// Text matches process name.
	out = Apply(socks, Filter{Text: "ssh"})
	assert.Len(t, out, 1)
}

func TestFilter_Combined(t *testing.T) {
	socks := []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalPort: 8080, State: netstat.StateListen, ProcessName: "nginx", User: "root"},
		{Protocol: netstat.ProtocolTCP, LocalPort: 8080, State: netstat.StateEstablished, ProcessName: "nginx", User: "root"},
		{Protocol: netstat.ProtocolTCP, LocalPort: 8080, State: netstat.StateListen, ProcessName: "apache", User: "root"},
	}
	// port 8080 AND state LISTEN AND process nginx -> 1 match (FR-03-07 AND logic).
	out := Apply(socks, Filter{
		Ports:   []uint16{8080},
		States:  []netstat.State{netstat.StateListen},
		Process: "nginx",
	})
	assert.Len(t, out, 1)
}

func TestFilter_IsEmpty_AndSummary(t *testing.T) {
	assert.True(t, Filter{}.IsEmpty())
	f := Filter{Ports: []uint16{80, 443}, Protocols: []netstat.Protocol{netstat.ProtocolTCP}, Process: "ng"}
	assert.False(t, f.IsEmpty())
	s := f.Summary()
	assert.Contains(t, s, "proto:tcp")
	assert.Contains(t, s, "port:80,443")
	assert.Contains(t, s, "proc:ng")
	assert.Equal(t, "", Filter{}.Summary())
}

func TestCIDRContains_ZoneAndBogus(t *testing.T) {
	n, _ := ParseCIDR("fe80::/16")
	assert.True(t, cidrContains(n, "fe80::1%eth0"), "zone stripped before match")
	assert.False(t, cidrContains(n, "not-an-ip"))
	assert.False(t, cidrContains(nil, "1.2.3.4"))
}

func TestCIContains(t *testing.T) {
	assert.True(t, ciContains("Nginx", "ngi"))
	assert.False(t, ciContains("nginx", ""))
	assert.False(t, ciContains("nginx", "xyz"))
}
