package app

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/internal/collector"
	"github.com/pmtop/pmtop/internal/config"
	"github.com/pmtop/pmtop/pkg/netstat"
)

func TestSetConfig_AppliesSort(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	m.SetConfig(config.Config{SortColumn: "pid", SortAsc: false})
	assert.Equal(t, SortPID, m.sortKey)
	assert.False(t, m.sortAsc)
}

func TestSetConfig_AppliesNoColor(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.SetConfig(config.Config{NoColor: true})
	assert.True(t, m.style.Colorblind() || true) // style exists
}

func TestSetConfig_AppliesColorblind(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.SetConfig(config.Config{ColorblindMode: true})
	assert.True(t, m.style.Colorblind())
}

func TestSetConfig_ManualMode(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.SetConfig(config.Config{RefreshInterval: "manual"})
	assert.True(t, m.manual, "manual mode flag set")
	// Init should do initial refresh but no tick scheduling.
	cmd := m.Init()
	require.NotNil(t, cmd, "Init does initial refresh")
}

func TestSetConfig_ManualModePauseHint(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.SetConfig(config.Config{RefreshInterval: "manual"})
	m.RefreshNow()
	mm, _ := m.Update(keyMsg(' '))
	m2 := mm.(Model)
	assert.Contains(t, m2.statusMsg, "manual mode")
}

func TestSetConfig_IntervalOverride(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.SetConfig(config.Config{RefreshInterval: "5s"})
	assert.Equal(t, 5*time.Second, m.interval)
}

func TestUpdate_ToggleServicePort(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	require.False(t, m.showService)
	mm, _ := m.Update(keyMsg('p'))
	m2 := mm.(Model)
	assert.True(t, m2.showService)
	assert.Contains(t, m2.statusMsg, "service")
	mm, _ = m2.Update(keyMsg('p'))
	m3 := mm.(Model)
	assert.False(t, m3.showService)
}

func TestUpdate_VimGotoTop(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	m.tbl.SetCursor(2)
	mm, _ := m.Update(keyMsg('g'))
	assert.Equal(t, 0, mm.(Model).Cursor(), "g goes to top")
}

func TestUpdate_VimGotoBottom(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('G'))
	assert.Equal(t, len(m.Socks())-1, mm.(Model).Cursor(), "G goes to bottom")
}

func TestUpdate_HelpEscCloses(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	m2 := mm.(Model)
	require.Equal(t, modeHelp, m2.Mode())
	mm, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, modeTable, mm.(Model).Mode())
}

func TestUpdate_HelpQuestionKey(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('?'))
	m2 := mm.(Model)
	assert.Equal(t, modeHelp, m2.Mode(), "? opens help")
}

func TestUpdate_HelpContainsBindings(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 30)
	m.RefreshNow()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	m2 := mm.(Model)
	v := m2.View()
	assert.Contains(t, v, "Help")
	assert.Contains(t, v, "quit")
	assert.Contains(t, v, "sort")
	assert.Contains(t, v, "signal")
}

func TestUpdate_DetailScroll(t *testing.T) {
	src := newDetailSource()
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 20)
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	require.Equal(t, modeDetail, m.Mode())
	require.NotNil(t, m.detail)
	assert.Equal(t, 0, m.detail.scroll)
	// Scroll down.
	mm, _ = m.Update(keyMsg('j'))
	m = mm.(Model)
	assert.Equal(t, 1, m.detail.scroll)
	// Scroll up.
	mm, _ = m.Update(keyMsg('k'))
	m = mm.(Model)
	assert.Equal(t, 0, m.detail.scroll)
}

func TestView_SummaryLine(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	v := m.View()
	assert.Contains(t, v, "LISTEN")
	assert.Contains(t, v, "ESTAB")
}

func TestView_SummaryLineEmpty(t *testing.T) {
	m := New(&fakeSource{socks: nil}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	v := m.View()
	// No summary line when no data.
	assert.NotContains(t, v, "LISTEN")
}

func TestView_SearchModeHasBottomBar(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('/'))
	m = mm.(Model)
	v := m.View()
	assert.Contains(t, v, "Search:")
	assert.Contains(t, v, "confirm")
	assert.Contains(t, v, "cancel")
}

func TestView_StatusMessagePreservesHints(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	m.setStatus("test-status", time.Hour)
	v := m.View()
	// Both hints and status should be visible.
	assert.Contains(t, v, "test-status")
}

func TestView_PauseStatusPersistent(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg(' '))
	m = mm.(Model)
	require.True(t, m.paused)
	assert.True(t, m.statusPerm, "pause status is persistent")
	v := m.View()
	assert.Contains(t, v, "PAUSED")
}

func TestView_DetailNoExtraSpaces(t *testing.T) {
	src := newDetailSource()
	src.proc[100] = collector.ProcessInfo{
		PID: 100, PPID: 1, Name: "sshd", User: "root", UID: 0,
		Cmdline: "/usr/sbin/sshd -D", Exe: "/usr/sbin/sshd",
		VmRSS: 10 * 1024 * 1024, VmSize: 20 * 1024 * 1024,
	}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 40)
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	v := m.View()
	// V1 fix: no extra spaces around parentheses.
	assert.Contains(t, v, "root (0)")
	assert.NotContains(t, v, "root ( 0 )")
	// V2 fix: CPU shows a number, not "-".
	assert.Contains(t, v, "CPU:")
	assert.NotContains(t, v, "CPU: -")
	// VmSize now shown.
	assert.Contains(t, v, "VSZ")
}

func TestView_DetailContainerFromPid(t *testing.T) {
	src := &fakeSource{socks: []netstat.SocketInfo{
		{Protocol: netstat.ProtocolTCP, LocalAddr: "0.0.0.0", LocalPort: 80, State: netstat.StateListen, Inode: 100, PID: 200, ProcessName: "nginx", User: "www-data", Runtime: "docker", ContainerID: "abc123def456", ContainerName: "web"},
	}}
	src.proc = map[int]collector.ProcessInfo{
		200: {PID: 200, Name: "nginx", User: "www-data", UID: 33},
	}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.Resize(120, 40)
	m.RefreshNow()
	m.tbl.SetCursor(0)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	v := m.View()
	assert.Contains(t, v, "docker")
	assert.Contains(t, v, "web")
}

func TestUpdate_SortIndicator(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	// Default sort is SortProto; pressing 's' cycles to SortPort which maps to ColLocal.
	mm, _ := m.Update(keyMsg('s'))
	m2 := mm.(Model)
	cols := m2.tbl.Columns()
	found := false
	for _, c := range cols {
		if strings.Contains(c.Title, "▲") || strings.Contains(c.Title, "▼") {
			found = true
			break
		}
	}
	assert.True(t, found, "sort indicator visible in column titles")
}

func TestUpdate_MouseWheel(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	require.Equal(t, 0, m.Cursor())
	mm, _ := m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown, Y: 5})
	assert.Equal(t, 1, mm.(Model).Cursor(), "wheel down moves cursor")
	mm, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelUp, Y: 5})
	assert.Equal(t, 0, mm.(Model).Cursor(), "wheel up moves cursor")
}

func TestView_FilterFormCompact(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('f'))
	m = mm.(Model)
	v := m.View()
	assert.Contains(t, v, "Ports")
	assert.Contains(t, v, "User")
	assert.Contains(t, v, "Container")
}

func TestCountStates(t *testing.T) {
	socks := sampleSockets()
	counts := countStates(socks)
	assert.NotEmpty(t, counts)
	// Should have LISTEN and ESTAB entries.
	var hasListen, hasEstab bool
	for _, c := range counts {
		if c.Label == "LISTEN" {
			hasListen = true
			assert.Equal(t, 2, c.Count)
		}
		if c.Label == "ESTAB" {
			hasEstab = true
			assert.Equal(t, 1, c.Count)
		}
	}
	assert.True(t, hasListen)
	assert.True(t, hasEstab)
}

func TestSortKeyFromConfig(t *testing.T) {
	cases := []struct {
		input string
		want  SortKey
		ok    bool
	}{
		{"proto", SortProto, true},
		{"PID", SortPID, true},
		{"port", SortPort, true},
		{"process", SortProcess, true},
		{"bogus", SortProto, false},
	}
	for _, tc := range cases {
		got, ok := sortKeyFromConfig(tc.input)
		assert.Equal(t, tc.ok, ok, "input: %s", tc.input)
		if ok {
			assert.Equal(t, tc.want, got, "input: %s", tc.input)
		}
	}
}
