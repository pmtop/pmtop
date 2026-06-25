package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func updateKey(m Model, r rune) Model {
	mm, _ := m.Update(keyMsg(r))
	return mm.(Model)
}

func TestUpdate_NavDownUp(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	require.Equal(t, 0, m.Cursor())

	m = updateKey(m, 'j')
	assert.Equal(t, 1, m.Cursor(), "j moves down")
	m = updateKey(m, 'j')
	assert.Equal(t, 2, m.Cursor())
	m = updateKey(m, 'k')
	assert.Equal(t, 1, m.Cursor(), "k moves up")
}

func TestUpdate_NavHomeEnd(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, len(m.Socks())-1, mm.(Model).Cursor())

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	assert.Equal(t, 0, mm.(Model).Cursor())
}

func TestUpdate_SortCycle(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	first := m.sortKey
	m = updateKey(m, 's')
	assert.NotEqual(t, first, m.sortKey, "s cycles sort key")
	assert.Contains(t, m.statusMsg, "sort:")
}

func TestUpdate_SortDir(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	asc := m.sortAsc
	m = updateKey(m, 'S')
	assert.NotEqual(t, asc, m.sortAsc, "S toggles direction")
}

func TestUpdate_Pause(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	m = updateKey(m, ' ')
	assert.True(t, m.paused, "space pauses")
	assert.Contains(t, m.statusMsg, "PAUSED")
	m = updateKey(m, ' ')
	assert.False(t, m.paused, "space resumes")
}

func TestUpdate_Refresh(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.refresh()
	calls := src.calls
	m = updateKey(m, 'r')
	assert.Equal(t, calls+1, src.calls, "r triggers refresh")
	assert.Contains(t, m.statusMsg, "refresh")
}

func TestUpdate_TickRefreshesWhenNotPaused(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	mm, _ := m.Update(tickMsg{})
	assert.Equal(t, 1, src.calls, "tick triggers refresh")
	assert.False(t, mm.(Model).paused)
}

func TestUpdate_TickSkipsWhenPaused(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	m.paused = true
	mm, _ := m.Update(tickMsg{})
	assert.Equal(t, 0, src.calls, "no refresh while paused")
	assert.True(t, mm.(Model).paused)
}

func TestUpdate_Quit(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	mm, cmd := m.Update(keyMsg('q'))
	assert.True(t, mm.(Model).quitting)
	require.NotNil(t, cmd)

	// ctrl+c also quits
	m2 := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	mm2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, mm2.(Model).quitting)
}

func TestUpdate_EmptyNoNav(t *testing.T) {
	m := New(&fakeSource{socks: nil}, "1.0.0", false, 2*time.Second)
	mm, _ := m.Update(keyMsg('j'))
	assert.Equal(t, 0, mm.(Model).Cursor(), "no nav on empty model")
}

func TestUpdate_WindowSize(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m2 := mm.(Model)
	assert.Equal(t, 100, m2.width)
	assert.Equal(t, 24, m2.height)
	// table height = 24 - 2 = 22
	assert.Equal(t, 22, m2.height-2)
}

func TestUpdate_NotAvailableHint(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	// F1 (help) is not yet implemented; shows a hint.
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	assert.Contains(t, mm.(Model).statusMsg, "not available")
}

func TestUpdate_FilterFormEnters(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	mm, _ := m.Update(keyMsg('f'))
	assert.Equal(t, modeFilter, mm.(Model).Mode())
}

func TestUpdate_SearchEnters(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	mm, _ := m.Update(keyMsg('/'))
	assert.Equal(t, modeSearch, mm.(Model).Mode())
}

func TestUpdate_RefreshMsg(t *testing.T) {
	src := &fakeSource{socks: sampleSockets()}
	m := New(src, "1.0.0", false, 2*time.Second)
	mm, _ := m.Update(refreshMsg{})
	assert.Equal(t, 1, src.calls)
	assert.Len(t, mm.(Model).Socks(), 4)
}

func TestUpdate_NoMatchKey(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.refresh()
	// 'x' is not bound to anything; state should be unchanged.
	mm, _ := m.Update(keyMsg('x'))
	assert.Equal(t, m.Cursor(), mm.(Model).Cursor())
	assert.Empty(t, mm.(Model).statusMsg)
}

func TestUpdate_NonKeyMsgPassesThrough(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	mm, _ := m.Update("some-non-key-msg")
	assert.Equal(t, m, mm)
}
