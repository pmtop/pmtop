package ui

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/pkg/netstat"
)

func TestBuildColumns_FitsWidth(t *testing.T) {
	cols := BuildColumns(120)
	require.Len(t, cols, 8)
	total := 0
	for _, c := range cols {
		total += c.Width
	}
	// 7 separators * 3 + columns must be <= width.
	assert.LessOrEqual(t, total+7*3, 120)
}

func TestBuildColumns_Narrow(t *testing.T) {
	// PRD NFR-08: minimum terminal is 80x24. Columns must fit at 80 cols.
	cols := BuildColumns(80)
	total := 0
	for _, c := range cols {
		total += c.Width
		assert.GreaterOrEqual(t, c.Width, 4, "columns keep a minimum width")
	}
	assert.LessOrEqual(t, total+7*3, 80)
}

func TestRowsFromSockets(t *testing.T) {
	socks := []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 22, State: netstat.StateListen, Inode: 1, PID: 100, ProcessName: "sshd", User: "root"},
		{Protocol: netstat.ProtocolUnix, Path: "/tmp/sock", State: netstat.StateUnconnected, Inode: 2, PID: 0},
	}
	rows := RowsFromSockets(socks, NewStyle())
	require.Len(t, rows, 2)
	assert.Contains(t, rows[0][ColProto], "TCP")
	assert.Contains(t, rows[0][ColProto], "▶") // LISTEN symbol
	assert.Equal(t, "0.0.0.0:22", rows[0][ColLocal])
	assert.Equal(t, "LISTEN", rows[0][ColState])
	assert.Equal(t, "100", rows[0][ColPID])
	assert.Equal(t, "sshd", rows[0][ColProcess])
	assert.Equal(t, "root", rows[0][ColUser])

	// Unix ownerless row.
	assert.Contains(t, rows[1][ColProto], "UNIX")
	assert.Equal(t, "/tmp/sock", rows[1][ColLocal])
	assert.Equal(t, "-", rows[1][ColPID])
	assert.Equal(t, "-", rows[1][ColProcess])
}

func TestRowsFromSockets_ContainerShortID(t *testing.T) {
	socks := []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalPort: 80, State: netstat.StateListen, PID: 1, ContainerID: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
	}
	rows := RowsFromSockets(socks, NewStyle())
	assert.Equal(t, "0123456789ab", rows[0][ColContainer], "container id truncated to 12")
}

func TestRemoteCell(t *testing.T) {
	assert.Equal(t, "*", remoteCell(netstat.SocketInfo{Protocol: netstat.ProtocolTCP, RemoteAddr: "0.0.0.0", RemotePort: 0}))
	assert.Equal(t, "1.2.3.4:5678", remoteCell(netstat.SocketInfo{Protocol: netstat.ProtocolTCP, RemoteAddr: "1.2.3.4", RemotePort: 5678}))
	assert.Equal(t, "-", remoteCell(netstat.SocketInfo{Protocol: netstat.ProtocolUnix}))
}

func TestStateCell(t *testing.T) {
	assert.Equal(t, "LISTEN", stateCell(netstat.SocketInfo{Protocol: netstat.ProtocolTCP, State: netstat.StateListen}))
	assert.Equal(t, "-", stateCell(netstat.SocketInfo{Protocol: netstat.ProtocolUDP, State: netstat.StateUnknown}))
	assert.Equal(t, "-", stateCell(netstat.SocketInfo{Protocol: netstat.ProtocolUnix, State: netstat.StateUnknown}))
}

func TestNoColor(t *testing.T) {
	require.False(t, NoColor(), "NO_COLOR unset by default")
	os.Setenv("NO_COLOR", "1")
	t.Cleanup(func() { os.Unsetenv("NO_COLOR") })
	assert.True(t, NoColor())
}

func TestNewStyle_NoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	t.Cleanup(func() { os.Unsetenv("NO_COLOR") })
	s := NewStyle()
	assert.True(t, s.noColor)
	// styleRow is a no-op when color is disabled.
	row := RowsFromSockets([]netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, State: netstat.StateListen, PID: 1, LocalPort: 80},
	}, s)
	assert.NotContains(t, row[0][ColState], "\x1b") // no ANSI escape codes
}

func TestStatusBar(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	s := NewStyle()
	out := s.StatusBar("1.0.0", true, false, "2s", "TCP,LISTEN", 100)
	assert.Contains(t, out, "pmtop 1.0.0")
	assert.Contains(t, out, "[root]")
	assert.Contains(t, out, "Filter: TCP,LISTEN")
}
