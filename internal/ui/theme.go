package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/pmtop/pmtop/pkg/netstat"
)

// Style holds lipgloss styles for the TUI. When color is disabled (NO_COLOR),
// styleRow returns rows unchanged and relies on the state symbols.
type Style struct {
	noColor bool

	// chrome styles
	header     lipgloss.Style
	statusBar  lipgloss.Style
	hintBar    lipgloss.Style
	selected   lipgloss.Style
	warn       lipgloss.Style

	// per-state text colors
	stateStyles map[netstat.State]lipgloss.Style
}

// NewStyle returns a Style honoring the NO_COLOR environment variable.
func NewStyle() *Style {
	s := &Style{noColor: NoColor()}
	if s.noColor {
		s.header = lipgloss.NewStyle().Bold(true)
		s.statusBar = lipgloss.NewStyle()
		s.hintBar = lipgloss.NewStyle()
		s.selected = lipgloss.NewStyle().Bold(true)
		s.warn = lipgloss.NewStyle().Bold(true)
		return s
	}
	s.header = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	s.statusBar = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	s.hintBar = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	s.selected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	s.warn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))

	s.stateStyles = map[netstat.State]lipgloss.Style{
		netstat.StateListen:     lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		netstat.StateEstablished: lipgloss.NewStyle().Foreground(lipgloss.Color("4")), // blue
		netstat.StateTimeWait:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
		netstat.StateCloseWait:  lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		netstat.StateSynSent:    lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		netstat.StateClosing:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		netstat.StateFinWait1:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		netstat.StateFinWait2:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		netstat.StateConnected:  lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
	}
	return s
}

// styleRow applies state coloring to the Proto and State cells of a row.
func (s *Style) styleRow(row table.Row, sock netstat.SocketInfo) table.Row {
	if s.noColor {
		return row
	}
	if st, ok := s.stateStyles[sock.State]; ok {
		if len(row) > ColProto {
			row[ColProto] = st.Render(row[ColProto])
		}
		if len(row) > ColState {
			row[ColState] = st.Render(row[ColState])
		}
	}
	return row
}

// StatusBar renders the top status bar.
func (s *Style) StatusBar(version string, root, paused bool, interval string, filterSummary string, width int) string {
	mode := "user"
	if root {
		mode = "root"
	}
	badge := ""
	if paused {
		badge = " [PAUSED]"
	}
	left := "pmtop " + version + " [" + mode + "]" + badge + " refresh:" + interval
	right := ""
	if filterSummary != "" {
		right = "Filter: " + filterSummary
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return s.statusBar.Render(left + strings.Repeat(" ", gap) + right)
}

// HintBar renders the bottom key-hint bar.
func (s *Style) HintBar(hints string, width int) string {
	return s.hintBar.Render(hints)
}

// Warn renders a warning banner (e.g. restricted-mode notice).
func (s *Style) Warn(text string) string {
	return s.warn.Render(text)
}
