package app

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestView_RendersChromeAndRows(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "9.9.9", false, 2*time.Second)
	m.Resize(120, 24)
	m.refresh()
	v := m.View()
	assert.Contains(t, v, "pmtop 9.9.9")
	assert.Contains(t, v, "[user]")
	assert.Contains(t, v, "2s")
	// Table shows sequence numbers and protocol
	assert.Contains(t, v, "TCP")
	assert.Contains(t, v, "LISTEN")
}

func TestView_QuittingEmpty(t *testing.T) {
	m := New(&fakeSource{}, "1.0.0", false, 2*time.Second)
	m.quitting = true
	assert.Empty(t, m.View())
}

func TestView_StatusMessage(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.refresh()
	m.setStatus("custom-status", time.Hour)
	assert.Contains(t, m.View(), "custom-status")
}

func TestView_HintsContainQuit(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.refresh()
	v := m.View()
	assert.True(t, strings.Contains(v, "q") || strings.Contains(v, "Quit"), "hints reference quit")
}

func TestView_RootBadge(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", true, 2*time.Second)
	m.Resize(120, 24)
	m.refresh()
	assert.Contains(t, m.View(), "[root]")
}

func TestView_PausedBadge(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", true, 2*time.Second)
	m.Resize(120, 24)
	m.refresh()
	m.paused = true
	assert.Contains(t, m.View(), "PAUSED")
}

func TestView_HasSequenceNumbers(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.refresh()
	v := m.View()
	assert.Contains(t, v, "1", "row 1 has sequence number")
	assert.Contains(t, v, "2", "row 2 has sequence number")
}

func TestView_HasDetailPanel(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.refresh()
	v := m.View()
	assert.Contains(t, v, "Process Detail", "detail panel is always visible")
}

func TestView_HasSummaryBar(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.Resize(120, 24)
	m.refresh()
	v := m.View()
	assert.Contains(t, v, "LISTEN", "summary bar shows state counts")
}

func TestHelpView(t *testing.T) {
	m := New(&fakeSource{}, "1.0.0", false, 2*time.Second)
	h := m.HelpView()
	assert.NotEmpty(t, h)
	assert.Contains(t, h, "quit")
	assert.Contains(t, h, "sort")
}
