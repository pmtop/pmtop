package app

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestView_RendersChromeAndRows(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "9.9.9", false, 2*time.Second)
	m.width = 120
	m.height = 24
	m.refresh()
	v := m.View()
	assert.Contains(t, v, "pmtop 9.9.9")
	assert.Contains(t, v, "[user]")
	assert.Contains(t, v, "refresh:2s")
	assert.Contains(t, v, "sshd")
	assert.Contains(t, v, "nginx")
}

func TestView_QuittingEmpty(t *testing.T) {
	m := New(&fakeSource{}, "1.0.0", false, 2*time.Second)
	m.quitting = true
	assert.Empty(t, m.View())
}

func TestView_StatusMessage(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.width = 120
	m.refresh()
	m.setStatus("custom-status", time.Hour)
	assert.Contains(t, m.View(), "custom-status")
}

func TestView_HintsContainQuit(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", false, 2*time.Second)
	m.width = 120
	m.refresh()
	v := m.View()
	assert.True(t, strings.Contains(v, "q") || strings.Contains(v, "Quit"), "hints reference quit")
}

func TestView_RootBadge(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", true, 2*time.Second)
	m.width = 120
	m.refresh()
	assert.Contains(t, m.View(), "[root]")
}

func TestView_PausedBadge(t *testing.T) {
	m := New(&fakeSource{socks: sampleSockets()}, "1.0.0", true, 2*time.Second)
	m.width = 120
	m.refresh()
	m.paused = true
	assert.Contains(t, m.View(), "PAUSED")
}

func TestHelpView(t *testing.T) {
	m := New(&fakeSource{}, "1.0.0", false, 2*time.Second)
	h := m.HelpView()
	require.NotEmpty(t, h)
	assert.Contains(t, h, "quit")
	assert.Contains(t, h, "sort")
}
