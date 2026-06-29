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

// RightPanelWidth is the fixed width of the right detail+signal panel.
const RightPanelWidth = 40

// DataSource abstracts socket collection so the TUI can be tested without /proc.
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
	modeSignal             // 'F9' signal selection in right-bottom panel
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
	manual   bool
	quitting bool

	full  []netstat.SocketInfo
	socks []netstat.SocketInfo
	filt  filter.Filter

	tbl     table.Model
	sortKey SortKey
	sortAsc bool

	mode         mode
	searchInput  textinput.Model
	filterInputs []textinput.Model
	filterFocus  int
	filtDraft    filter.Filter

	detail *DetailState
	signal *SignalState
	sender SignalSender
	cfg    config.Config

	showService bool
	help        bool

	width, height int

	lastRefresh time.Time
	statusMsg   string
	statusExp   time.Time
	statusPerm  bool
	err         error
}

// New returns the initial Model.
func New(src DataSource, version string, root bool, interval time.Duration) Model {
	style := ui.NewStyle(false, false)
	tbl := table.New(
		table.WithColumns(ui.BuildColumns(80)),
		table.WithHeight(10),
		table.WithStyles(table.Styles{
			Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7")),
			Selected: lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("237")).Foreground(lipgloss.Color("15")),
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

func newSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "search process / PID / user / container"
	ti.CharLimit = 64
	ti.Width = 40
	return ti
}

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

func refreshCmd() tea.Cmd {
	return func() tea.Msg { return refreshMsg{} }
}

type refreshMsg struct{}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// refresh performs a data snapshot, applies the filter, re-sorts, rebuilds
// table rows, preserves the cursor, and updates the detail panel.
func (m *Model) refresh() {
	prev := m.socks
	prevCursor := m.tbl.Cursor()

	full, err := m.source.Collect()
	m.full = full
	m.err = err
	if err == nil {
		m.rebuild()
		if m.mode != modeSignal {
			m.preserveCursor(prev, prevCursor)
			m.updateDetailPanel()
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

// applySort re-sorts the current snapshot, rebuilds rows, and updates sort
// indicators.
func (m *Model) applySort() {
	if len(m.socks) > 0 {
		SortSockets(m.socks, m.sortKey, m.sortAsc)
		m.tbl.SetRows(ui.RowsFromSockets(m.socks, m.style, ui.RowOptions{
			ShowService: m.showService,
		}))
		m.clampCursor()
		m.updateDetailPanel()
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

func baseColumnTitle(i int) string {
	switch i {
	case ui.ColSeq:
		return "#"
	case ui.ColProto:
		return "Proto"
	case ui.ColLocal:
		return "Local"
	case ui.ColRemote:
		return "Remote"
	case ui.ColState:
		return "State"
	default:
		return ""
	}
}

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

func (m *Model) setStatus(msg string, dur time.Duration) {
	m.statusMsg = msg
	m.statusPerm = dur == 0
	m.statusExp = time.Now().Add(dur)
}

func (m Model) currentSocket() (netstat.SocketInfo, bool) {
	c := m.tbl.Cursor()
	if c < 0 || c >= len(m.socks) {
		return netstat.SocketInfo{}, false
	}
	return m.socks[c], true
}

// Socks returns the current filtered snapshot.
func (m Model) Socks() []netstat.SocketInfo { return m.socks }

// Filter returns the active filter.
func (m Model) Filter() filter.Filter { return m.filt }

// SetFilter replaces the active filter and rebuilds the view.
func (m *Model) SetFilter(f filter.Filter) {
	m.filt = f
	if m.full != nil {
		m.rebuild()
		m.clampCursor()
		m.updateDetailPanel()
	}
}

// SetSignalSender replaces the signal sender (for tests).
func (m *Model) SetSignalSender(s SignalSender) {
	if s != nil {
		m.sender = s
	}
}

// SetConfig applies runtime configuration after construction.
func (m *Model) SetConfig(cfg config.Config) {
	m.cfg = cfg
	if strings.ToLower(cfg.RefreshInterval) == "manual" {
		m.manual = true
	} else if d := cfg.Interval(); d > 0 {
		m.interval = d
	}
	if cfg.SortColumn != "" {
		if sk, ok := sortKeyFromConfig(cfg.SortColumn); ok {
			m.sortKey = sk
		}
	}
	m.sortAsc = cfg.SortAsc
	m.style = ui.NewStyle(cfg.NoColor, cfg.ColorblindMode)
	if cfg.NoColor {
		m.tbl.SetStyles(table.Styles{
			Header:   lipgloss.NewStyle().Bold(true),
			Selected: lipgloss.NewStyle().Bold(true),
		})
	} else {
		m.tbl.SetStyles(table.Styles{
			Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7")),
			Selected: lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("237")).Foreground(lipgloss.Color("15")),
		})
	}
	m.applySortIndicator()
	if m.full != nil {
		m.rebuild()
		m.clampCursor()
		m.updateDetailPanel()
	}
}

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

// Err returns the last collection error.
func (m Model) Err() error { return m.err }

// Cursor returns the current cursor index.
func (m Model) Cursor() int { return m.tbl.Cursor() }

// Mode returns the current interaction mode.
func (m Model) Mode() mode { return m.mode }

// Detail returns the current detail state (for testing).
func (m Model) Detail() *DetailState { return m.detail }

// availableHeight returns the body height (excluding status, summary, bottom).
func (m Model) availableHeight() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	h = h - 3 // status(1) + summary(1) + bottom(1)
	if h < 3 {
		h = 3
	}
	return h
}

// leftPaneWidth returns the width available for the left table.
func (m Model) leftPaneWidth() int {
	w := m.width
	if w <= 0 {
		w = 120
	}
	w = w - RightPanelWidth
	if w < 20 {
		w = 20
	}
	return w
}

// Resize sets the viewport size and rebuilds the table layout.
func (m *Model) Resize(width, height int) {
	m.width = width
	m.height = height
	leftW := m.leftPaneWidth()
	availH := m.availableHeight()
	m.tbl.SetWidth(leftW)
	m.tbl.SetHeight(availH)
	m.tbl.SetColumns(ui.BuildColumns(leftW))
	m.applySortIndicator()
}

// RefreshNow forces a data refresh.
func (m *Model) RefreshNow() { m.refresh() }
