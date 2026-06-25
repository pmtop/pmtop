package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/pkg/netstat"
)

func TestSortSockets_ByPort(t *testing.T) {
	s := sampleSockets()
	SortSockets(s, SortPort, true)
	require.Equal(t, uint16(22), s[0].LocalPort)
	require.Equal(t, uint16(53), s[1].LocalPort)
	require.Equal(t, uint16(8080), s[2].LocalPort)
}

func TestSortSockets_ByPortDesc(t *testing.T) {
	s := sampleSockets()
	SortSockets(s, SortPort, false)
	assert.Equal(t, uint16(8080), s[0].LocalPort)
	assert.Equal(t, uint16(22), s[len(s)-1].LocalPort)
}

func TestSortSockets_ByProcess(t *testing.T) {
	s := sampleSockets()
	SortSockets(s, SortProcess, true)
	assert.Equal(t, "dnsmasq", s[0].ProcessName)
	assert.Equal(t, "sshd", s[len(s)-1].ProcessName)
}

func TestSortSockets_ByPID(t *testing.T) {
	s := sampleSockets()
	SortSockets(s, SortPID, true)
	assert.Equal(t, 100, s[0].PID)
	assert.Equal(t, 400, s[len(s)-1].PID)
}

func TestSortSockets_ByProto(t *testing.T) {
	s := sampleSockets()
	SortSockets(s, SortProto, true)
	// "tcp" < "udp" lexicographically.
	assert.Equal(t, netstat.ProtocolTCP, s[0].Protocol)
	assert.Equal(t, netstat.ProtocolUDP, s[len(s)-1].Protocol)
}

func TestSortSockets_ByState(t *testing.T) {
	s := sampleSockets()
	SortSockets(s, SortState, true)
	// CLOSE < ESTAB < LISTEN lexicographically.
	assert.Equal(t, netstat.StateClose, s[0].State)
	assert.Equal(t, netstat.StateListen, s[len(s)-1].State)
}

func TestSortSockets_Stable(t *testing.T) {
	s := []netstat.SocketInfo{
		{LocalPort: 80, ProcessName: "a"},
		{LocalPort: 80, ProcessName: "b"},
		{LocalPort: 80, ProcessName: "c"},
	}
	SortSockets(s, SortPort, true)
	assert.Equal(t, "a", s[0].ProcessName)
	assert.Equal(t, "b", s[1].ProcessName)
	assert.Equal(t, "c", s[2].ProcessName)
}

func TestSortKeyNext_Cycle(t *testing.T) {
	k := SortProto
	seen := map[SortKey]bool{}
	for i := 0; i < len(sortKeys)+2; i++ {
		seen[k] = true
		k = k.next()
	}
	// After cycling, every defined key should appear.
	for _, sk := range sortKeys {
		assert.True(t, seen[sk], "key %s not visited", sk)
	}
}

func TestSortKeyString(t *testing.T) {
	assert.Equal(t, "Port", SortPort.String())
	assert.Equal(t, "Process", SortProcess.String())
}

func TestFormatPort(t *testing.T) {
	assert.Equal(t, "0.0.0.0:22", formatPort(netstat.SocketInfo{Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 22}))
	assert.Equal(t, "/tmp/sock", formatPort(netstat.SocketInfo{Protocol: netstat.ProtocolUnix, Path: "/tmp/sock"}))
}
