package app

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pmtop/pmtop/internal/config"
	"github.com/pmtop/pmtop/internal/filter"
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

// mode is the current interaction mode of the TUI.
type mode int

const (
	modeTable  mode = iota // default: navigating the port table
	modeSearch             // '/' free-text search input
	modeFilter             // 'f' filter form
	modeDetail             // 'Enter' process detail side panel
	modeSignal             // 'K' signal selection dialog
	modeHelp               // 'F1' full-screen help overlay
)

// filterFields labels the inputs in the filter form, in order.
var filterFields = []string{"Ports", "Protocols", "States", "Process", "PID", "User", "Container", "Local CIDR", "Remote CIDR"}

// Model is the Bubble Tea model for the pmtop TUI.
type Model struct {
	source   DataSource
	keys     KeyMap
	style    *ui.Style
	version  string
	root     bool
	interval time.Duration

	paused   bool
	manual   bool // refresh_interval="manual": no auto-tick
	quitting bool

	full  []netstat.SocketInfo // raw snapshot from the source
	socks []netstat.SocketInfo // filtered + sorted (displayed)
	filt  filter.Filter

	tbl      table.Model
	sortKey  SortKey
	sortAsc  bool

	mode         mode
	searchInput  textinput.Model
	filterInputs []textinput.Model
	filterFocus  int
	filtDraft    filter.Filter // saved filter to restore on cancel

	detail *DetailState
	signal *SignalState
	sender SignalSender
	cfg    config.Config

	// display toggles
	showService bool // 'p' key: show service names instead of port numbers

	// help overlay
	help bool // 'F1' toggles full-screen help

	width, height int

	lastRefresh time.Time
	statusMsg   string
	statusExp   time.Time
	statusPerm  bool // true = status message persists until cleared
	err         error
}

// New returns the initial Model. interval is the auto-refresh period
// (default 2s per FR-02-01).
func New(src DataSource, version string, root bool, interval time.Duration) Model {
	style := ui.NewStyle(false, false)
	tbl := table.New(
		table.WithColumns(ui.BuildColumns(120)),
		table.WithHeight(10),
		table.WithStyles(table.Styles{
			Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7")),
			Selected: lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236")),
		}),
	)
	m := Model{
		source:   src,
		keys:     DefaultKeyMap(),
		style:    style,
		version:  version,
		root:     root,
		interval: interval,
		sortKey:  SortProto,
		sortAsc:  true,
		tbl:      tbl,
	}
	m.searchInput = newSearchInput()
	m.filterInputs = newFilterInputs()
	m.sender = realSender{}
	return m
}

// newSearchInput builds the '/' free-text search field.
func newSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "search process / PID / user / container"
	ti.CharLimit = 64
	ti.Width = 40
	return ti
}

// newFilterInputs builds the filter form fields (one textinput per column).
func newFilterInputs() []textinput.Model {
	inputs := make([]textinput.Model, len(filterFields))
	for i := range inputs {
		ti := textinput.New()
		ti.CharLimit = 80
		ti.Width = 30
		inputs[i] = ti
	}
	return inputs
}

// Init starts the first refresh and the refresh ticker.
func (m Model) Init() tea.Cmd {
	if m.manual {
		return refreshCmd()
	}
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

// refresh performs a data snapshot, applies the filter, re-sorts, rebuilds
// table rows, and preserves the cursor on the same socket when possible
// (FR-02-01). In detail mode, the detail panel data is also updated.
func (m *Model) refresh() {
	prev := m.socks
	prevCursor := m.tbl.Cursor()

	full, err := m.source.Collect()
	m.full = full
	m.err = err
	if err == nil {
		m.rebuild()
		// Don't move cursor while detail panel is open (prevents container info mismatch).
		if m.mode != modeDetail {
			m.preserveCursor(prev, prevCursor)
		}
		// Update detail panel data if open (for CPU% refresh).
		if m.mode == modeDetail && m.detail != nil {
			m.updateDetailData()
		}
	}
	m.lastRefresh = time.Now()
}

// rebuild re-applies the filter to the full snapshot, sorts, and sets rows.
func (m *Model) rebuild() {
	m.socks = filter.Apply(m.full, m.filt)
	SortSockets(m.socks, m.sortKey, m.sortAsc)
	m.tbl.SetRows(ui.RowsFromSockets(m.socks, m.style, ui.RowOptions{
		ShowService: m.showService,
	}))
}

// applySort re-sorts the current (filtered) snapshot, rebuilds rows, and
// updates the column sort indicators.
func (m *Model) applySort() {
	if len(m.socks) > 0 {
		SortSockets(m.socks, m.sortKey, m.sortAsc)
		m.tbl.SetRows(ui.RowsFromSockets(m.socks, m.style, ui.RowOptions{
			ShowService: m.showService,
		}))
		m.clampCursor()
	}
	m.applySortIndicator()
}

// applySortIndicator updates column titles with ▲/▼ for the current sort.
func (m *Model) applySortIndicator() {
	cols := m.tbl.Columns()
	sortCol := ui.SortColumnIndex(int(m.sortKey))
	for i := range cols {
		base := baseColumnTitle(i)
		cols[i].Title = ui.ColumnTitleForSort(base, i == sortCol, m.sortAsc)
	}
	m.tbl.SetColumns(cols)
}

// baseColumnTitle returns the plain title for column index i.
func baseColumnTitle(i int) string {
	switch i {
	case ui.ColProto:
		return "Proto"
	case ui.ColLocal:
		return "Local"
	case ui.ColRemote:
		return "Remote"
	case ui.ColState:
		return "State"
	case ui.ColPID:
		return "PID"
	case ui.ColProcess:
		return "Process"
	case ui.ColUser:
		return "User"
	case ui.ColContainer:
		return "Container"
	default:
		return ""
	}
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
// If dur is zero, the message persists until cleared (used for pause state).
func (m *Model) setStatus(msg string, dur time.Duration) {
	m.statusMsg = msg
	m.statusPerm = dur == 0
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

// Socks returns the current filtered snapshot (for testing/inspection).
func (m Model) Socks() []netstat.SocketInfo { return m.socks }

// Filter returns the active filter (for testing/inspection).
func (m Model) Filter() filter.Filter { return m.filt }

// SetFilter replaces the active filter and rebuilds the view (for tests/CLI).
func (m *Model) SetFilter(f filter.Filter) {
	m.filt = f
	if m.full != nil {
		m.rebuild()
		m.clampCursor()
	}
}

// SetSignalSender replaces the signal sender (for tests to avoid killing real
// processes).
func (m *Model) SetSignalSender(s SignalSender) {
	if s != nil {
		m.sender = s
	}
}

// SetConfig applies runtime configuration (refresh interval, sort, colorblind,
// no-color, etc.) after construction. Wired by cmd/pmtop from layered config
// + flags.
func (m *Model) SetConfig(cfg config.Config) {
	m.cfg = cfg

	// Refresh interval: "manual" means no auto-refresh.
	if strings.ToLower(cfg.RefreshInterval) == "manual" {
		m.manual = true
	} else if d := cfg.Interval(); d > 0 {
		m.interval = d
	}

	// Sort column and direction.
	if cfg.SortColumn != "" {
		if sk, ok := sortKeyFromConfig(cfg.SortColumn); ok {
			m.sortKey = sk
		}
	}
	m.sortAsc = cfg.SortAsc

	// Rebuild style with color/no-color/colorblind settings.
	m.style = ui.NewStyle(cfg.NoColor, cfg.ColorblindMode)
	if cfg.NoColor {
		m.tbl.SetStyles(table.Styles{
			Header:   lipgloss.NewStyle().Bold(true),
			Selected: lipgloss.NewStyle().Bold(true),
		})
	} else {
		m.tbl.SetStyles(table.Styles{
			Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7")),
			Selected: lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236")),
		})
	}

	// Apply sort indicator to current columns.
	m.applySortIndicator()

	// Rebuild rows with new style if data is already loaded.
	if m.full != nil {
		m.rebuild()
		m.clampCursor()
	}
}

// sortKeyFromConfig maps a config sort column name to a SortKey.
func sortKeyFromConfig(s string) (SortKey, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "proto", "protocol":
		return SortProto, true
	case "port":
		return SortPort, true
	case "local":
		return SortLocal, true
	case "remote":
		return SortRemote, true
	case "state":
		return SortState, true
	case "pid":
		return SortPID, true
	case "process":
		return SortProcess, true
	case "container":
		return SortContainer, true
	default:
		return SortProto, false
	}
}

// Err returns the last collection error, if any (for non-interactive use).
func (m Model) Err() error { return m.err }

// Cursor returns the current cursor index (for testing).
func (m Model) Cursor() int { return m.tbl.Cursor() }

// Mode returns the current interaction mode (for testing).
func (m Model) Mode() mode { return m.mode }

// Resize sets the viewport size and rebuilds the table layout. Intended for
// non-interactive rendering (smoke tests, snapshots).
func (m *Model) Resize(width, height int) {
	m.width = width
	m.height = height
	m.tbl.SetWidth(width)
	h := height - 3 // top status (1) + summary (1) + bottom hints (1)
	if h < 3 {
		h = 3
	}
	m.tbl.SetHeight(h)
	m.tbl.SetColumns(ui.BuildColumns(width))
	m.applySortIndicator()
}

// RefreshNow forces a data refresh without waiting for a tick. Intended for
// non-interactive rendering and integration tests.
func (m *Model) RefreshNow() { m.refresh() }
