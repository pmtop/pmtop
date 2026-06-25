package app

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/pkg/netstat"
)

func TestNew_Defaults(t *testing.T) {
	m := New(&fakeSource{}, "1.0.0", false, 2*time.Second)
	assert.Equal(t, "1.0.0", m.version)
	assert.False(t, m.root)
	assert.Equal(t, 2*time.Second, m.interval)
	assert.Equal(t, SortProto, m.sortKey)
	assert.True(t, m.sortAsc)
	assert.False(t, m.paused)
}

func TestModel_Refresh(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.refresh()
	require.NoError(t, m.err)
	require.Len(t, m.Socks(), 4)
	// Default sort is by proto (tcp first) then port -> first is sshd:22.
	assert.Equal(t, uint16(22), m.Socks()[0].LocalPort)
	assert.Equal(t, "sshd", m.Socks()[0].ProcessName)
	assert.Equal(t, 0, m.Cursor()) // cursor at top
}

func TestModel_Refresh_Error(t *testing.T) {
	src := &fakeSource{err: errors.New("boom")}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.refresh()
	assert.Error(t, m.err)
	assert.Empty(t, m.Socks())
}

func TestModel_PreserveCursor_ByInode(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.refresh()
	// Default sort (proto then port): tcp[100@22, 200@8080, 300@8080], udp[400@53].
	// myapp (inode 300) is at index 2.
	m.tbl.SetCursor(2)
	require.Equal(t, uint64(300), m.Socks()[2].Inode)

	// New snapshot with a different order; cursor should follow inode 300.
	src.socks = []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalPort: 1, Inode: 999, ProcessName: "new"},
		{Protocol: netstat.ProtocolTCP, LocalPort: 2, Inode: 300, ProcessName: "myapp2"},
		{Protocol: netstat.ProtocolTCP, LocalPort: 3, Inode: 100, ProcessName: "sshd"},
	}
	m.refresh()
	// After proto+port sort: port1(999), port2(300), port3(100) -> inode 300 at index 1.
	assert.Equal(t, 1, m.Cursor(), "cursor follows inode 300")
	assert.Equal(t, uint64(300), m.Socks()[m.Cursor()].Inode)
}

func TestModel_PreserveCursor_Clamp(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.refresh()
	m.tbl.SetCursor(3)
	src.socks = []netstat.SocketInfo{{LocalPort: 1, Inode: 1}}
	m.refresh()
	assert.Equal(t, 0, m.Cursor(), "cursor clamped to 0 when old selection gone")
}

func TestModel_ApplySort(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.refresh()
	m.sortKey = SortProcess
	m.applySort()
	assert.Equal(t, "dnsmasq", m.Socks()[0].ProcessName)
}

func TestModel_SetStatus(t *testing.T) {
	m := New(&fakeSource{}, "1.0.0", false, 2*time.Second)
	m.setStatus("hello", time.Second)
	assert.Equal(t, "hello", m.statusMsg)
	assert.True(t, m.statusExp.After(time.Now()))
}

func TestModel_CurrentSocket(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.refresh()
	m.tbl.SetCursor(0)
	s, ok := m.currentSocket()
	require.True(t, ok)
	assert.Equal(t, "sshd", s.ProcessName)

	// Empty model -> no current.
	m2 := New(&fakeSource{}, "1.0.0", false, 2*time.Second)
	_, ok = m2.currentSocket()
	assert.False(t, ok)
}

func TestModel_Resize(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(110, 24)
	assert.Equal(t, 110, m.width)
	assert.Equal(t, 24, m.height)
	// table reserves 1 line for its header; visible height is height-2 minus 1.
	assert.Positive(t, m.tbl.Height())
	assert.Less(t, m.tbl.Height(), m.height)
}

func TestModel_Resize_FloorHeight(t *testing.T) {
	m := New(&fakeSource{}, "1.0.0", false, 2*time.Second)
	m.Resize(80, 4) // height-2 = 2, floored to 3
	assert.Positive(t, m.tbl.Height())
}

func TestModel_RefreshNow_AndErr(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	assert.NoError(t, m.Err())
	assert.Len(t, m.Socks(), 4)

	m2 := New(&fakeSource{err: errors.New("x")}, "1.0.0", false, 2*time.Second)
	m2.RefreshNow()
	assert.Error(t, m2.Err())
}

func TestIntervalString(t *testing.T) {
	assert.Equal(t, "2s", intervalString(2*time.Second))
	assert.Equal(t, "1s", intervalString(time.Second))
	assert.Equal(t, "5s", intervalString(5*time.Second))
	assert.Equal(t, "500ms", intervalString(500*time.Millisecond))
	assert.Equal(t, "3s", intervalString(3*time.Second))
}
