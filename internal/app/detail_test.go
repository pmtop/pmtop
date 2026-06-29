package app

import (
	"os"
	"strings"
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

// Detail panel is always visible — verify it shows after refresh.
func TestDetailPanel_AutoUpdate(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 30)
	m.RefreshNow()
	m.tbl.SetCursor(0) // sshd (PID 100)
	m.updateDetailPanel()
	require.NotNil(t, m.detail)
	assert.Equal(t, 100, m.detail.pid)
	assert.Equal(t, "sshd", m.detail.proc.Name)
}

func TestView_DetailContainsFields(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 30)
	m.RefreshNow()
	m.tbl.SetCursor(0)
	m.updateDetailPanel()
	v := m.View()
	assert.Contains(t, v, "Process Detail")
	assert.Contains(t, v, "sshd")
	assert.Contains(t, v, "/usr/sbin/sshd -D")
	assert.Contains(t, v, "root")
}

func TestDetailPanel_NoPID(t *testing.T) {
	src := &fakeSource{socks: []netstat.SocketInfo{
		{Protocol: netstat.ProtocolUnix, Path: "/tmp/sock", Inode: 5, State: netstat.StateUnconnected, PID: 0},
	}}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 30)
	m.RefreshNow()
	m.updateDetailPanel()
	assert.Nil(t, m.detail, "no detail for ownerless socket")
	v := m.View()
	assert.Contains(t, v, "ownerless socket")
}

// F9 opens signal panel.
func TestUpdate_F9OpensSignal(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	m.tbl.SetCursor(0)

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF9})
	m = mm.(Model)
	require.Equal(t, modeSignal, m.Mode())
	require.NotNil(t, m.signal)
	assert.Equal(t, 100, m.signal.pid)
	assert.Equal(t, "sshd", m.signal.name)
	assert.Equal(t, "SIGTERM", process.Signals[m.signal.sel].Name)
}

func TestUpdate_SignalSelectionAndConfirm(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 30)
	sender := &fakeSender{}
	m.SetSignalSender(sender)
	m.RefreshNow()
	m.tbl.SetCursor(0)

	// Open signal dialog with F9.
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF9})
	m = mm.(Model)
	// Move down once to SIGKILL.
	mm, _ = m.Update(keyMsg('j'))
	m = mm.(Model)
	assert.Equal(t, "SIGKILL", process.Signals[m.signal.sel].Name)

	// Enter -> confirmation step.
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	assert.True(t, m.signal.confirm, "Enter opens confirmation")
	assert.Contains(t, m.View(), "Confirm")

	// Esc -> back to selection.
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
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF9})
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
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF9})
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	assert.Contains(t, m.statusMsg, "failed")
}

func TestUpdate_KillOnNoPID(t *testing.T) {
	src := &fakeSource{socks: []netstat.SocketInfo{
		{Protocol: netstat.ProtocolUnix, Path: "/x", Inode: 5, PID: 0},
	}}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF9})
	m = mm.(Model)
	assert.Equal(t, modeTable, m.Mode())
	assert.Contains(t, m.statusMsg, "no process")
}

var errPerm = permErr("permission denied")

type permErr string

func (e permErr) Error() string { return string(e) }

func TestHumanBytes(t *testing.T) {
	assert.Equal(t, "512B", humanBytes(512))
	assert.Contains(t, humanBytes(2048), "KB")
	assert.Contains(t, humanBytes(10*1024*1024), "MB")
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "123", itoa(123))
	assert.Equal(t, "-7", itoa(-7))
}

func TestUpdate_SignalUp(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.SetSignalSender(&fakeSender{})
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF9})
	m = mm.(Model)
	start := m.signal.sel
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = mm.(Model)
	assert.Equal(t, start-1, m.signal.sel, "Up moves selection up")
}

func TestUpdate_HelpFromTable(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	m2 := mm.(Model)
	assert.Equal(t, modeHelp, m2.Mode(), "F1 opens help overlay")
}

func TestUpdate_ExportWritesFile(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('e'))
	m = mm.(Model)
	assert.Contains(t, m.statusMsg, "exported pmtop-export-")
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	var found bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			found = true
			break
		}
	}
	assert.True(t, found, "export created a json file")
}

func TestUpdate_ExportEmpty(t *testing.T) {
	m := New(&fakeSource{socks: nil}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('e'))
	assert.Contains(t, mm.(Model).statusMsg, "nothing to export")
}
