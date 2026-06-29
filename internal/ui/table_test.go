package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/pkg/netstat"
)

func TestBuildColumns_FitsWidth(t *testing.T) {
	cols := BuildColumns(80) // left pane width
	require.Len(t, cols, NumColumns)
	total := 0
	for _, c := range cols {
		total += c.Width
	}
	assert.LessOrEqual(t, total+(NumColumns-1)*3, 80)
}

func TestBuildColumns_Narrow(t *testing.T) {
	cols := BuildColumns(40)
	total := 0
	for _, c := range cols {
		total += c.Width
		assert.GreaterOrEqual(t, c.Width, 4, "columns keep a minimum width")
	}
	assert.LessOrEqual(t, total+(NumColumns-1)*3, 40)
}

func TestRowsFromSockets(t *testing.T) {
	socks := []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 22, State: netstat.StateListen, Inode: 1, PID: 100, ProcessName: "sshd", User: "root"},
		{Protocol: netstat.ProtocolUnix, Path: "/tmp/sock", State: netstat.StateUnconnected, Inode: 2, PID: 0},
	}
	rows := RowsFromSockets(socks, NewStyle(false, false), RowOptions{})
	require.Len(t, rows, 2)
	// Row 0: sequence number, proto, local, remote, state
	assert.Equal(t, "1", rows[0][ColSeq])
	assert.Contains(t, rows[0][ColProto], "TCP")
	assert.Contains(t, rows[0][ColProto], "▶")
	assert.Equal(t, "0.0.0.0:22", rows[0][ColLocal])
	assert.Equal(t, "LISTEN", rows[0][ColState])

	// Row 1: unix socket
	assert.Equal(t, "2", rows[1][ColSeq])
	assert.Contains(t, rows[1][ColProto], "UNIX")
	assert.Equal(t, "/tmp/sock", rows[1][ColLocal])
}

func TestRowsFromSockets_ShowService(t *testing.T) {
	socks := []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 22, State: netstat.StateListen, PID: 1},
		{Protocol: netstat.ProtocolTCP, LocalAddr: "1.2.3.4", LocalPort: 9999, RemoteAddr: "5.6.7.8", RemotePort: 443, State: netstat.StateEstablished, PID: 2},
	}
	rows := RowsFromSockets(socks, NewStyle(false, false), RowOptions{ShowService: true})
	assert.Contains(t, rows[0][ColLocal], "ssh")
	assert.Contains(t, rows[1][ColRemote], "https")

	rowsPort := RowsFromSockets(socks, NewStyle(false, false), RowOptions{})
	assert.Contains(t, rowsPort[0][ColLocal], "22")
	assert.Contains(t, rowsPort[1][ColRemote], "443")
}

func TestRemoteCell(t *testing.T) {
	assert.Equal(t, "*", remoteCell(netstat.SocketInfo{Protocol: netstat.ProtocolTCP, RemoteAddr: "0.0.0.0", RemotePort: 0}, false))
	assert.Equal(t, "1.2.3.4:5678", remoteCell(netstat.SocketInfo{Protocol: netstat.ProtocolTCP, RemoteAddr: "1.2.3.4", RemotePort: 5678}, false))
	assert.Equal(t, "-", remoteCell(netstat.SocketInfo{Protocol: netstat.ProtocolUnix}, false))
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
	s := NewStyle(true, false)
	assert.True(t, s.noColor)
	row := RowsFromSockets([]netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, State: netstat.StateListen, PID: 1, LocalPort: 80},
	}, s, RowOptions{})
	assert.NotContains(t, row[0][ColState], "\x1b")
}

func TestNewStyle_Colorblind(t *testing.T) {
	s := NewStyle(false, true)
	assert.True(t, s.Colorblind())
	assert.False(t, s.noColor)
	assert.NotNil(t, s.stateStyles)
}

func TestStatusBar(t *testing.T) {
	s := NewStyle(false, false)
	out := s.StatusBar("1.0.0", true, false, "2s", "TCP,LISTEN", 100)
	assert.Contains(t, out, "pmtop 1.0.0")
	assert.Contains(t, out, "[root]")
	assert.Contains(t, out, "Filter: TCP,LISTEN")
}

func TestStatusBar_Truncation(t *testing.T) {
	s := NewStyle(false, false)
	out := s.StatusBar("1.0.0", true, false, "2s", strings.Repeat("X", 200), 80)
	assert.LessOrEqual(t, lipgloss.Width(out), 80)
}

func TestSummaryBar(t *testing.T) {
	s := NewStyle(false, false)
	counts := []StateCount{
		{State: netstat.StateListen, Label: "LISTEN", Count: 12},
		{State: netstat.StateEstablished, Label: "ESTAB", Count: 8},
	}
	out := s.SummaryBar(counts, 100)
	assert.Contains(t, out, "LISTEN:12")
	assert.Contains(t, out, "ESTAB:8")
}

func TestColumnTitleForSort(t *testing.T) {
	assert.Equal(t, "Proto", ColumnTitleForSort("Proto", false, true))
	assert.Equal(t, "Proto▲", ColumnTitleForSort("Proto", true, true))
	assert.Equal(t, "Proto▼", ColumnTitleForSort("Proto", true, false))
}

func TestSortColumnIndex(t *testing.T) {
	assert.Equal(t, ColProto, SortColumnIndex(0))
	assert.Equal(t, ColLocal, SortColumnIndex(1))
	assert.Equal(t, ColState, SortColumnIndex(4))
	assert.Equal(t, -1, SortColumnIndex(5))  // SortPID not in table
	assert.Equal(t, -1, SortColumnIndex(99))
}
