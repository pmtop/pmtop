package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pmtop/pmtop/internal/filter"
)

func runesMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestSetFilter(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	m.SetFilter(filter.Filter{Process: "nginx"})
	assert.Len(t, m.Socks(), 1)
	assert.Equal(t, "nginx", m.Socks()[0].ProcessName)
}

func TestSearchMode_RealTimeFilter(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	require.Len(t, m.Socks(), 4)

	// '/' enters search mode.
	mm, _ := m.Update(keyMsg('/'))
	m = mm.(Model)
	require.Equal(t, modeSearch, m.Mode())

	// Typing narrows in real time.
	mm, _ = m.Update(runesMsg("nginx"))
	m = mm.(Model)
	assert.Equal(t, "nginx", m.Filter().Text)
	assert.Len(t, m.Socks(), 1, "only nginx matches text 'nginx'")

	// Esc exits search keeping the filter applied.
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = mm.(Model)
	assert.Equal(t, modeTable, m.Mode())
	assert.Len(t, m.Socks(), 1, "filter retained after leaving search")
}

func TestSearchMode_ExitKeepsFilter_TableEscClears(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('/'))
	m = mm.(Model)
	mm, _ = m.Update(runesMsg("nginx"))
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape}) // exit search
	m = mm.(Model)
	require.Len(t, m.Socks(), 1)

	// Esc in table mode clears all filters (FR-03-08).
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = mm.(Model)
	assert.True(t, m.Filter().IsEmpty())
	assert.Len(t, m.Socks(), 4)
}

func TestFilterForm_Apply(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()

	// 'f' opens the form focused on the Ports field.
	mm, _ := m.Update(keyMsg('f'))
	m = mm.(Model)
	require.Equal(t, modeFilter, m.Mode())

	// Type "22" into the Ports field and apply.
	mm, _ = m.Update(runesMsg("22"))
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)

	assert.Equal(t, modeTable, m.Mode())
	require.NotEmpty(t, m.Filter().Ports)
	assert.Equal(t, uint16(22), m.Filter().Ports[0])
	assert.Len(t, m.Socks(), 1, "port 22 -> sshd only")
	assert.Equal(t, "sshd", m.Socks()[0].ProcessName)
}

func TestFilterForm_Cancel(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	// Pre-set a filter so we can verify cancel restores the draft.
	m.SetFilter(filter.Filter{Process: "nginx"})

	mm, _ := m.Update(keyMsg('f'))
	m = mm.(Model)
	mm, _ = m.Update(runesMsg("53")) // type into Ports (would change filter if applied)
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape}) // cancel
	m = mm.(Model)

	assert.Equal(t, modeTable, m.Mode())
	assert.Equal(t, "nginx", m.Filter().Process, "draft restored on cancel")
	assert.Len(t, m.Socks(), 1)
}

func TestFilterForm_InvalidShowsError(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('f'))
	m = mm.(Model)
	mm, _ = m.Update(runesMsg("abc")) // invalid port
	m = mm.(Model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(Model)
	assert.Equal(t, modeFilter, m.Mode(), "stays in form on parse error")
	assert.Contains(t, m.statusMsg, "filter error")
}

func TestFilterForm_TabCycles(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('f'))
	m = mm.(Model)
	require.Equal(t, 0, m.filterFocus)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = mm.(Model)
	assert.Equal(t, 1, m.filterFocus, "Tab moves to next field")
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = mm.(Model)
	assert.Equal(t, 0, m.filterFocus, "Shift+Tab moves back")
}

func TestView_FilterSummary(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	m.SetFilter(filter.Filter{Ports: []uint16{22}})
	v := m.View()
	assert.Contains(t, v, "port:22")
}

func TestView_SearchMode(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('/'))
	m = mm.(Model)
	assert.Contains(t, m.View(), "Search:")
}

func TestView_FilterForm(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.RefreshNow()
	mm, _ := m.Update(keyMsg('f'))
	m = mm.(Model)
	v := m.View()
	assert.Contains(t, v, "Ports")
	assert.Contains(t, v, "Protocols")
	assert.Contains(t, v, "Enter=apply")
}
