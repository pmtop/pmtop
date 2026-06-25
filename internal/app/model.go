package app

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pmtop/pmtop/internal/ui"
	"github.com/pmtop/pmtop/pkg/netstat"
)

// DataSource abstracts socket collection so the TUI can be tested without /proc.
// *collector.Collector satisfies this interface.
type DataSource interface {
	Collect() ([]netstat.SocketInfo, error)
}

// tickMsg is emitted on each refresh interval.
type tickMsg time.Time

// Model is the Bubble Tea model for the pmtop TUI.
type Model struct {
	source   DataSource
	keys     KeyMap
	style    *ui.Style
	version  string
	root     bool
	interval time.Duration

	paused   bool
	quitting bool

	socks   []netstat.SocketInfo
	tbl     table.Model
	sortKey SortKey
	sortAsc bool

	width, height int

	lastRefresh time.Time
	statusMsg   string
	statusExp   time.Time
	err         error
}

// New returns the initial Model. interval is the auto-refresh period
// (default 2s per FR-02-01).
func New(src DataSource, version string, root bool, interval time.Duration) Model {
	style := ui.NewStyle()
	tbl := table.New(
		table.WithColumns(ui.BuildColumns(120)),
		table.WithHeight(10),
	)
	return Model{
		source:  src,
		keys:    DefaultKeyMap(),
		style:   style,
		version: version,
		root:    root,
		interval: interval,
		sortKey: SortProto,
		sortAsc: true,
		tbl:     tbl,
	}
}

// Init starts the first refresh and the refresh ticker.
func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshCmd(), tickCmd(m.interval))
}

// refreshCmd returns a command that triggers an immediate data refresh.
func refreshCmd() tea.Cmd {
	return func() tea.Msg { return refreshMsg{} }
}

// refreshMsg requests a data snapshot.
type refreshMsg struct{}

// tickCmd schedules the next tick after duration d.
func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// refresh performs a data snapshot, re-sorts, rebuilds table rows, and
// preserves the cursor on the same socket when possible (FR-02-01).
func (m *Model) refresh() {
	prev := m.socks
	prevCursor := m.tbl.Cursor()

	socks, err := m.source.Collect()
	m.socks = socks
	m.err = err
	if err == nil {
		SortSockets(m.socks, m.sortKey, m.sortAsc)
		m.tbl.SetRows(ui.RowsFromSockets(m.socks, m.style))
		m.preserveCursor(prev, prevCursor)
	}
	m.lastRefresh = time.Now()
}

// applySort re-sorts the current snapshot and rebuilds rows without refetching.
func (m *Model) applySort() {
	if len(m.socks) == 0 {
		return
	}
	SortSockets(m.socks, m.sortKey, m.sortAsc)
	m.tbl.SetRows(ui.RowsFromSockets(m.socks, m.style))
	m.clampCursor()
}

// preserveCursor keeps the selection on the same socket across a refresh by
// matching inode, falling back to PID+endpoint, then to a clamped index.
func (m *Model) preserveCursor(prev []netstat.SocketInfo, prevCursor int) {
	if len(m.socks) == 0 {
		m.tbl.SetCursor(0)
		return
	}
	if prevCursor < 0 || prevCursor >= len(prev) {
		m.clampCursor()
		return
	}
	target := prev[prevCursor]
	if target.Inode != 0 {
		for i, s := range m.socks {
			if s.Inode == target.Inode {
				m.tbl.SetCursor(i)
				return
			}
		}
	}
	for i, s := range m.socks {
		if s.PID == target.PID && s.LocalAddr == target.LocalAddr && s.LocalPort == target.LocalPort {
			m.tbl.SetCursor(i)
			return
		}
	}
	m.clampCursor()
}

// clampCursor keeps the cursor within the row range.
func (m *Model) clampCursor() {
	if len(m.socks) == 0 {
		return
	}
	c := m.tbl.Cursor()
	if c < 0 {
		c = 0
	}
	if c >= len(m.socks) {
		c = len(m.socks) - 1
	}
	m.tbl.SetCursor(c)
}

// setStatus shows msg for dur (e.g. 3s for signal feedback per FR-06-04).
func (m *Model) setStatus(msg string, dur time.Duration) {
	m.statusMsg = msg
	m.statusExp = time.Now().Add(dur)
}

// currentSocket returns the selected socket, if any.
func (m Model) currentSocket() (netstat.SocketInfo, bool) {
	c := m.tbl.Cursor()
	if c < 0 || c >= len(m.socks) {
		return netstat.SocketInfo{}, false
	}
	return m.socks[c], true
}

// Socks returns the current snapshot (for testing/inspection).
func (m Model) Socks() []netstat.SocketInfo { return m.socks }

// Err returns the last collection error, if any (for non-interactive use).
func (m Model) Err() error { return m.err }

// Cursor returns the current cursor index (for testing).
func (m Model) Cursor() int { return m.tbl.Cursor() }

// Resize sets the viewport size and rebuilds the table layout. Intended for
// non-interactive rendering (smoke tests, snapshots).
func (m *Model) Resize(width, height int) {
	m.width = width
	m.height = height
	m.tbl.SetWidth(width)
	h := height - 2
	if h < 3 {
		h = 3
	}
	m.tbl.SetHeight(h)
	m.tbl.SetColumns(ui.BuildColumns(width))
}

// RefreshNow forces a data refresh without waiting for a tick. Intended for
// non-interactive rendering and integration tests.
func (m *Model) RefreshNow() { m.refresh() }
