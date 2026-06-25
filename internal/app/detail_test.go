package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/internal/collector"
	"github.com/pmtop/pmtop/internal/process"
	"github.com/pmtop/pmtop/pkg/netstat"
)

func newDetailSource() *fakeSource {
	src := &fakeSource{socks: sampleSockets()}
	src.proc = map[int]collector.ProcessInfo{
		100: {PID: 100, PPID: 1, Name: "sshd", Cmdline: "/usr/sbin/sshd -D", Exe: "/usr/sbin/sshd", User: "root", UID: 0, VmRSS: 10 * 1024 * 1024},
	}
	src.cg = map[int]collector.CgroupInfo{
		100: {Version: 2, Runtime: "", Lines: []collector.CgroupLine{{HierarchyID: "0", Path: "/system.slice/sshd.service"}}},
	}
	return src
}

func TestUpdate_EnterOpensDetail(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	m.tbl.SetCursor(0) // sshd (PID 100)

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	require.Equal(t, modeDetail, m.Mode())
	require.NotNil(t, m.detail)
	assert.Equal(t, 100, m.detail.pid)
	assert.Equal(t, "sshd", m.detail.proc.Name)
}

func TestUpdate_DetailEscCloses(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	require.Equal(t, modeDetail, m.Mode())

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = mm.(Model)
	assert.Equal(t, modeTable, m.Mode())
	assert.Nil(t, m.detail)
}

func TestView_DetailContainsFields(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 30)
	m.RefreshNow()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	v := m.View()
	assert.Contains(t, v, "Process Detail")
	assert.Contains(t, v, "sshd")
	assert.Contains(t, v, "/usr/sbin/sshd -D")
	assert.Contains(t, v, "root")
}

func TestUpdate_EnterOnNoPID(t *testing.T) {
	// unix socket with PID 0
	src := &fakeSource{socks: []netstat.SocketInfo{
		{Protocol: netstat.ProtocolUnix, Path: "/tmp/sock", Inode: 5, State: netstat.StateUnconnected, PID: 0},
	}}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	assert.Equal(t, modeTable, m.Mode(), "no detail for PID 0")
	assert.Contains(t, m.statusMsg, "no process")
}

func TestUpdate_KillOpensSignal(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	m.tbl.SetCursor(0)

	mm, _ := m.Update(keyMsg('K'))
	m = mm.(Model)
	require.Equal(t, modeSignal, m.Mode())
	require.NotNil(t, m.signal)
	assert.Equal(t, 100, m.signal.pid)
	assert.Equal(t, "sshd", m.signal.name)
	// SIGTERM is the default selection.
	assert.Equal(t, "SIGTERM", process.Signals[m.signal.sel].Name)
}

func TestUpdate_SignalSelectionAndConfirm(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	sender := &fakeSender{}
	m.SetSignalSender(sender)
	m.RefreshNow()
	m.tbl.SetCursor(0)

	// Open signal dialog.
	mm, _ := m.Update(keyMsg('K'))
	m = mm.(Model)
	// Move down once to SIGKILL (index 3; default is SIGTERM at index 2).
	mm, _ = m.Update(keyMsg('j'))
	m = mm.(Model)
	assert.Equal(t, "SIGKILL", process.Signals[m.signal.sel].Name)

	// Enter -> confirmation step.
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	assert.True(t, m.signal.confirm, "Enter opens confirmation")
	assert.Contains(t, m.View(), "Confirm")

	// Esc -> back to selection (not exit).
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = mm.(Model)
	assert.False(t, m.signal.confirm)

	// Enter -> confirm, Enter -> send.
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	assert.Equal(t, modeTable, m.Mode())
	require.Len(t, sender.sent, 1)
	assert.Equal(t, 100, sender.sent[0].pid)
	assert.Equal(t, "SIGKILL", sender.sent[0].sig.Name)
	assert.Contains(t, m.statusMsg, "sent SIGKILL")
}

func TestUpdate_SignalEscCancels(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.SetSignalSender(&fakeSender{})
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(keyMsg('K'))
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = mm.(Model)
	assert.Equal(t, modeTable, m.Mode())
	assert.Nil(t, m.signal)
}

func TestUpdate_SignalSendFailure(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.SetSignalSender(&fakeSender{fail: errPerm})
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(keyMsg('K'))
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // send
	m = mm.(Model)
	assert.Contains(t, m.statusMsg, "failed")
}

func TestUpdate_KillOnNoPID(t *testing.T) {
	src := &fakeSource{socks: []netstat.SocketInfo{
		{Protocol: netstat.ProtocolUnix, Path: "/x", Inode: 5, PID: 0},
	}}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('K'))
	m = mm.(Model)
	assert.Equal(t, modeTable, m.Mode())
	assert.Contains(t, m.statusMsg, "no process")
}

// errPerm is a stand-in permission error for the fake sender.
var errPerm = permErr("permission denied")

type permErr string

func (e permErr) Error() string { return string(e) }

func TestHumanBytes(t *testing.T) {
	assert.Equal(t, "512 B", humanBytes(512))
	assert.Contains(t, humanBytes(2048), "KB")
	assert.Contains(t, humanBytes(10*1024*1024), "MB")
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "123", itoa(123))
	assert.Equal(t, "-7", itoa(-7))
}

func TestUpdate_DetailKillOpensSignal(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.SetSignalSender(&fakeSender{})
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // open detail
	m = mm.(Model)
	require.Equal(t, modeDetail, m.Mode())

	mm, _ = m.Update(keyMsg('K')) // signal from detail
	m = mm.(Model)
	assert.Equal(t, modeSignal, m.Mode())
	assert.Equal(t, 100, m.signal.pid)
}

func TestUpdate_DetailNoOpKey(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	mm, _ = m.Update(keyMsg('x')) // unrelated key
	m = mm.(Model)
	assert.Equal(t, modeDetail, m.Mode(), "unrelated key keeps detail open")
}

func TestUpdate_DetailError(t *testing.T) {
	src := newDetailSource()
	src.procErr = errPerm
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 30)
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	assert.Equal(t, modeDetail, m.Mode())
	assert.Contains(t, m.View(), "error")
}

func TestUpdate_SignalUp(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.SetSignalSender(&fakeSender{})
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(keyMsg('K'))
	m = mm.(Model)
	start := m.signal.sel
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = mm.(Model)
	assert.Equal(t, start-1, m.signal.sel, "Up moves selection up")
}

func TestUpdate_HelpNotAvailable(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	assert.Contains(t, mm.(Model).statusMsg, "not available")
}
