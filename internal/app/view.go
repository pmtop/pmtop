package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/pmtop/pmtop/internal/ui"
	"github.com/pmtop/pmtop/pkg/netstat"
)

// View renders the TUI: top status bar, summary line, port table (or filter
// form/search input/help overlay), and bottom hint/status bar.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	width := m.width
	if width <= 0 {
		width = 120
	}
	height := m.height
	if height <= 0 {
		height = 24
	}

	top := m.style.StatusBar(m.version, m.root, m.paused, intervalString(m.interval), m.filt.Summary(), width)
	summary := m.summaryLine(width)

	switch m.mode {
	case modeSearch:
		body := m.tbl.View()
		searchLine := "Search: " + m.searchInput.View()
		bottom := m.style.HintBar("[Enter]confirm [Esc]cancel", width)
		return top + "\n" + summary + "\n" + body + "\n" + searchLine + "\n" + bottom
	case modeFilter:
		return top + "\n" + summary + "\n" + m.filterFormView(width) + "\n" + m.filterFormHints(width)
	case modeDetail:
		return top + "\n" + m.detailView(width, height) + "\n" + m.style.HintBar("[Esc]close [↑/↓]scroll [K]signal", width)
	case modeSignal:
		return top + "\n" + summary + "\n" + m.signalView(width) + "\n" + m.style.HintBar("[↑/↓]choose [Enter]confirm [Esc]cancel", width)
	case modeHelp:
		return top + "\n" + m.helpView(width, height) + "\n" + m.style.HintBar("[Esc/F1]close", width)
	}

	body := m.tbl.View()
	bottom := m.bottomBar(width)
	return top + "\n" + summary + "\n" + body + "\n" + bottom
}

// summaryLine renders a single-line connection count summary with state colors.
func (m Model) summaryLine(width int) string {
	if len(m.full) == 0 {
		return ""
	}
	counts := countStates(m.full)
	return m.style.SummaryBar(counts, width)
}

// countStates tallies sockets by state for the summary bar.
func countStates(socks []netstat.SocketInfo) []ui.StateCount {
	type bucket struct {
		state netstat.State
		label string
	}
	order := []bucket{
		{netstat.StateListen, "LISTEN"},
		{netstat.StateEstablished, "ESTAB"},
		{netstat.StateTimeWait, "TIME_WAIT"},
		{netstat.StateCloseWait, "CLOSE_WAIT"},
		{netstat.StateSynSent, "SYN_SENT"},
		{netstat.StateFinWait1, "FIN_WAIT1"},
		{netstat.StateFinWait2, "FIN_WAIT2"},
		{netstat.StateClose, "CLOSED"},
		{netstat.StateClosing, "CLOSING"},
		{netstat.StateLastAck, "LAST_ACK"},
	}
	tcp := 0
	udp := 0
	unix := 0
	other := 0
	stateCounts := make(map[netstat.State]int)
	for _, s := range socks {
		stateCounts[s.State]++
		switch {
		case s.Protocol.IsTCP():
			tcp++
		case s.Protocol.IsUDP():
			udp++
		case s.Protocol == netstat.ProtocolUnix:
			unix++
		default:
			other++
		}
	}
	var counts []ui.StateCount
	for _, b := range order {
		if c := stateCounts[b.state]; c > 0 {
			counts = append(counts, ui.StateCount{State: b.state, Label: b.label, Count: c})
		}
	}
	if tcp > 0 {
		counts = append(counts, ui.StateCount{State: netstat.StateUnknown, Label: fmt.Sprintf("TCP:%d", tcp), Count: tcp})
	}
	if udp > 0 {
		counts = append(counts, ui.StateCount{State: netstat.StateUnknown, Label: fmt.Sprintf("UDP:%d", udp), Count: udp})
	}
	if unix > 0 {
		counts = append(counts, ui.StateCount{State: netstat.StateUnknown, Label: fmt.Sprintf("UNIX:%d", unix), Count: unix})
	}
	if other > 0 {
		counts = append(counts, ui.StateCount{State: netstat.StateUnknown, Label: fmt.Sprintf("RAW:%d", other), Count: other})
	}
	return counts
}

// bottomBar renders the bottom hint bar with an optional status message on
// the right side (V5 fix: status no longer replaces hints).
func (m Model) bottomBar(width int) string {
	hints := m.hints(width)
	if m.err != nil {
		errMsg := "error: " + m.err.Error()
		hints = truncateOrPad(hints, width-lipgloss.Width(errMsg)-2, "")
		return m.style.HintBar(hints+errMsg, width)
	}
	if m.statusMsg != "" && (m.statusPerm || (!m.statusExp.IsZero() && time.Now().Before(m.statusExp))) {
		status := m.statusMsg
		hints = truncateOrPad(hints, width-lipgloss.Width(status)-2, "")
		return m.style.HintBar(hints+status, width)
	}
	return m.style.HintBar(hints, width)
}

// truncateOrPad ensures s fits within maxW by truncating (with "…") or padding.
func truncateOrPad(s string, maxW int, pad string) string {
	w := lipgloss.Width(s)
	if w > maxW {
		return ui.TruncateRight(s, maxW)
	}
	if w < maxW && pad != "" {
		return s + strings.Repeat(pad, maxW-w)
	}
	return s
}

// filterFormView renders the filter form fields in a compact 2-column layout.
func (m Model) filterFormView(width int) string {
	var sb strings.Builder
	sb.WriteString("Filter (Tab=next, Enter=apply, Esc=cancel):\n\n")

	labelStyle := lipgloss.NewStyle().Bold(true)
	focusedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

	colWidth := width / 2
	if colWidth < 35 {
		colWidth = 35
	}

	leftFields := []int{0, 1, 2, 3, 4}       // Ports, Protocols, States, Process, PID
	rightFields := []int{5, 6, 7, 8}          // User, Container, Local CIDR, Remote CIDR
	maxRows := len(leftFields)
	if len(rightFields) > maxRows {
		maxRows = len(rightFields)
	}

	for row := 0; row < maxRows; row++ {
		var left, right string
		if row < len(leftFields) {
			left = m.renderFilterField(leftFields[row], colWidth, labelStyle, focusedStyle)
		}
		if row < len(rightFields) {
			right = m.renderFilterField(rightFields[row], colWidth, labelStyle, focusedStyle)
		}
		// Pad left to colWidth for alignment.
		leftW := lipgloss.Width(left)
		if leftW < colWidth {
			left += strings.Repeat(" ", colWidth-leftW)
		}
		sb.WriteString(left)
		sb.WriteString(right)
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderFilterField renders a single filter field with label and input.
func (m Model) renderFilterField(idx int, colWidth int, labelStyle, focusedStyle lipgloss.Style) string {
	label := filterFields[idx]
	prefix := "  "
	style := labelStyle
	if idx == m.filterFocus {
		prefix = "▶ "
		style = focusedStyle
	}
	inputW := colWidth - 14 // label(12) + prefix(2)
	if inputW < 10 {
		inputW = 10
	}
	m.filterInputs[idx].Width = inputW
	val := m.filterInputs[idx].View()
	return prefix + style.Render(padLabel(label)) + " " + val
}

// filterFormHints renders the bottom hint for the filter form.
func (m Model) filterFormHints(width int) string {
	return m.style.HintBar("[Tab]next [Shift+Tab]prev [Enter]apply [Esc]cancel", width)
}

// padLabel right-pads a field label for aligned form output.
func padLabel(l string) string {
	const w = 12
	if len(l) >= w {
		return l
	}
	return l + strings.Repeat(" ", w-len(l))
}

// hints renders the bottom key-hint bar from the keymap.
func (m Model) hints(width int) string {
	var parts []string
	for _, b := range m.keys.ShortHelp() {
		h := b.Help()
		k := h.Key
		if k == "" {
			continue
		}
		if i := strings.IndexByte(k, ' '); i > 0 {
			k = k[:i]
		}
		parts = append(parts, "["+k+"]"+h.Desc)
	}
	hint := strings.Join(parts, " ")
	return hint
}

// intervalString renders the refresh interval as a short label.
func intervalString(d time.Duration) string {
	switch {
	case d <= 500*time.Millisecond:
		return "500ms"
	case d == time.Second:
		return "1s"
	case d == 2*time.Second:
		return "2s"
	case d == 5*time.Second:
		return "5s"
	default:
		return d.String()
	}
}

// helpView renders the full F1 help overlay (full-screen).
func (m Model) helpView(width, height int) string {
	var sb strings.Builder
	sb.WriteString("pmtop key bindings\n\n")
	for _, group := range m.keys.FullHelp() {
		for _, b := range group {
			h := b.Help()
			sb.WriteString("  ")
			sb.WriteString(padKey(h.Key))
			sb.WriteString("  ")
			sb.WriteString(h.Desc)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return ui.Box("Help (F1 / ?)", sb.String(), width, height-2)
}

// HelpView renders the full help as a string (for tests/future wiring).
func (m Model) HelpView() string {
	var sb strings.Builder
	for _, group := range m.keys.FullHelp() {
		for _, b := range group {
			h := b.Help()
			sb.WriteString("  ")
			sb.WriteString(padKey(h.Key))
			sb.WriteString("  ")
			sb.WriteString(h.Desc)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// padKey left-pads a key spec to a fixed width for aligned help output.
func padKey(k string) string {
	const w = 10
	if len(k) >= w {
		return k
	}
	return k + strings.Repeat(" ", w-len(k))
}
