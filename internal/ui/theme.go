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
	noColor    bool
	colorblind bool

	// chrome styles
	header    lipgloss.Style
	statusBar lipgloss.Style
	hintBar   lipgloss.Style
	selected  lipgloss.Style
	warn      lipgloss.Style

	// per-state text colors
	stateStyles map[netstat.State]lipgloss.Style
}

// NewStyle returns a Style honoring the NO_COLOR environment variable and
// optional colorblind mode (FR-10-02).
func NewStyle(noColor, colorblind bool) *Style {
	s := &Style{noColor: noColor, colorblind: colorblind}
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

	if colorblind {
		s.stateStyles = map[netstat.State]lipgloss.Style{
			netstat.StateListen:      lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true),
			netstat.StateEstablished: lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
			netstat.StateTimeWait:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
			netstat.StateCloseWait:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
			netstat.StateSynSent:     lipgloss.NewStyle().Foreground(lipgloss.Color("13")),
			netstat.StateClosing:     lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
			netstat.StateFinWait1:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
			netstat.StateFinWait2:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
			netstat.StateConnected:   lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		}
	} else {
		s.stateStyles = map[netstat.State]lipgloss.Style{
			netstat.StateListen:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
			netstat.StateEstablished: lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
			netstat.StateTimeWait:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			netstat.StateCloseWait:   lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
			netstat.StateSynSent:     lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
			netstat.StateClosing:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
			netstat.StateFinWait1:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			netstat.StateFinWait2:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			netstat.StateConnected:   lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		}
	}
	return s
}

// Colorblind reports whether colorblind mode is active.
func (s *Style) Colorblind() bool { return s.colorblind }

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

// StatusBar renders the top status bar, padded to full width to prevent
// residual content from the previous frame.
func (s *Style) StatusBar(version string, root, paused bool, interval string, filterSummary string, width int) string {
	mode := "user"
	if root {
		mode = "root"
	}
	badge := ""
	if paused {
		badge = " [PAUSED]"
	}
	left := "pmtop " + version + " [" + mode + "]" + badge + " " + interval
	right := ""
	if filterSummary != "" {
		right = "Filter: " + filterSummary
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		avail := width - lipgloss.Width(left)
		if avail < 0 {
			avail = 0
		}
		if avail < lipgloss.Width(right) {
			right = lipgloss.NewStyle().MaxWidth(avail).Render(right)
			gap = 0
		}
	}
	content := left + strings.Repeat(" ", gap) + right
	return s.statusBar.Width(width).Render(content)
}

// HintBar renders the bottom key-hint bar, padded to full width.
func (s *Style) HintBar(hints string, width int) string {
	return s.hintBar.Width(width).Render(hints)
}

// Warn renders a warning banner (e.g. restricted-mode notice).
func (s *Style) Warn(text string) string {
	return s.warn.Render(text)
}

// StateColor returns the lipgloss style for a given state, or a no-op style.
func (s *Style) StateColor(st netstat.State) lipgloss.Style {
	if s.noColor {
		return lipgloss.NewStyle()
	}
	if style, ok := s.stateStyles[st]; ok {
		return style
	}
	return lipgloss.NewStyle()
}

// SummaryBar renders a single-line connection count summary with state colors,
// padded to full width.
func (s *Style) SummaryBar(counts []StateCount, width int) string {
	if len(counts) == 0 {
		return strings.Repeat(" ", width)
	}
	var parts []string
	for _, c := range counts {
		label := c.Label
		if label == "" {
			label = c.State.String()
		}
		text := label + ":" + itoa(c.Count)
		if !s.noColor {
			text = s.StateColor(c.State).Render(text)
		}
		parts = append(parts, text)
	}
	line := strings.Join(parts, "  ")
	return lipgloss.NewStyle().MaxWidth(width).Width(width).Render(line)
}

// StateCount pairs a state with its connection count for the summary bar.
type StateCount struct {
	State netstat.State
	Label string
	Count int
}

// TruncateRight truncates s to fit within maxDisplayWidth, appending "…" if
// truncation occurs. Uses lipgloss.MaxWidth for ANSI-safe truncation.
func TruncateRight(s string, maxDisplayWidth int) string {
	if maxDisplayWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxDisplayWidth {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(maxDisplayWidth).Render(s)
}

// itoa is a small allocation-free int -> string converter.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
