package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/pmtop/pmtop/internal/ui"
	"github.com/pmtop/pmtop/pkg/netstat"
)

// View renders the TUI with a split layout:
//   ┌──────────────────────────────────────┬─────────────────┐
//   │ status bar (full width)              │                 │
//   │ summary bar (full width)             │                 │
//   ├──────────────────────────────────────┬─────────────────┤
//   │ left: socket table                   │ right: detail   │
//   │                                      │       + signal  │
//   ├──────────────────────────────────────┴─────────────────┤
//   │ bottom hint bar (full width)                           │
//   └────────────────────────────────────────────────────────┘
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
	bottom := m.bottomBar(width)

	switch m.mode {
	case modeSearch:
		return m.viewSearch(width, height, top, summary, bottom)
	case modeFilter:
		return m.viewFilter(width, height, top, summary, bottom)
	case modeHelp:
		return m.viewHelp(width, height, top, summary, bottom)
	default:
		return m.viewSplit(width, height, top, summary, bottom)
	}
}

// viewSplit renders the default split layout (modeTable and modeSignal).
func (m Model) viewSplit(width, height int, top, summary, bottom string) string {
	availH := m.availableHeight()
	leftW := m.leftPaneWidth()
	rightW := RightPanelWidth
	if leftW+rightW > width {
		rightW = width - leftW
		if rightW < 20 {
			rightW = 20
		}
	}

	// Right panel split: detail (upper) + signal (lower).
	var detailH, signalH int
	if m.mode == modeSignal && m.signal != nil {
		detailH = availH * 3 / 5
		if detailH < 5 {
			detailH = 5
		}
		signalH = availH - detailH
	} else {
		detailH = availH
		signalH = 0
	}

	// Left pane: table.
	leftPane := m.tbl.View()

	// Right pane: detail upper + signal lower.
	rightUpper := m.detailPanel(rightW, detailH)
	rightLower := m.signalPanel(rightW, signalH)

	var rightPane string
	if rightLower != "" {
		rightPane = lipgloss.JoinVertical(lipgloss.Left, rightUpper, rightLower)
	} else {
		rightPane = rightUpper
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	result := top + "\n" + summary + "\n" + body + "\n" + bottom
	return ensureHeight(result, width, height)
}

// viewSearch renders the split layout with a search input line.
func (m Model) viewSearch(width, height int, top, summary, bottom string) string {
	availH := m.availableHeight() - 1 // search line takes 1 row
	leftW := m.leftPaneWidth()
	rightW := RightPanelWidth
	if leftW+rightW > width {
		rightW = width - leftW
		if rightW < 20 {
			rightW = 20
		}
	}

	leftPane := m.tbl.View()
	rightUpper := m.detailPanel(rightW, availH)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightUpper)

	searchLine := "Search: " + m.searchInput.View()
	result := top + "\n" + summary + "\n" + body + "\n" + searchLine + "\n" + bottom
	return ensureHeight(result, width, height)
}

// viewFilter renders the filter form (full width, no split).
func (m Model) viewFilter(width, height int, top, summary, bottom string) string {
	form := m.filterFormView(width)
	result := top + "\n" + summary + "\n" + form + "\n" + bottom
	return ensureHeight(result, width, height)
}

// viewHelp renders the full-screen help overlay.
func (m Model) viewHelp(width, height int, top, summary, bottom string) string {
	availH := m.availableHeight()
	helpBox := m.helpView(width, availH)
	result := top + "\n" + summary + "\n" + helpBox + "\n" + bottom
	return ensureHeight(result, width, height)
}

// ensureHeight guarantees the output is exactly height lines, each padded to
// width visible characters. This prevents residual content from the previous
// frame when the terminal doesn't clear the screen (Bubble Tea AltScreen).
func ensureHeight(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		w := lipgloss.Width(lines[i])
		if w < width {
			lines[i] = lines[i] + strings.Repeat(" ", width-w)
		}
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

// summaryLine renders a single-line connection count summary.
func (m Model) summaryLine(width int) string {
	if len(m.full) == 0 {
		return strings.Repeat(" ", width)
	}
	counts := countStates(m.full)
	return m.style.SummaryBar(counts, width)
}

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
	tcp, udp, unix, other := 0, 0, 0, 0
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

// bottomBar renders the bottom hint bar with an optional status message.
func (m Model) bottomBar(width int) string {
	hints := m.hints(width)
	if m.err != nil {
		errMsg := "error: " + m.err.Error()
		hints = truncateOrPad(hints, width-lipgloss.Width(errMsg)-2)
		return m.style.HintBar(hints+"  "+errMsg, width)
	}
	if m.statusMsg != "" && (m.statusPerm || (!m.statusExp.IsZero() && time.Now().Before(m.statusExp))) {
		status := m.statusMsg
		hints = truncateOrPad(hints, width-lipgloss.Width(status)-2)
		return m.style.HintBar(hints+"  "+status, width)
	}
	return m.style.HintBar(hints, width)
}

func truncateOrPad(s string, maxW int) string {
	w := lipgloss.Width(s)
	if w > maxW && maxW > 0 {
		return ui.TruncateRight(s, maxW)
	}
	return s
}

func (m Model) filterFormView(width int) string {
	var sb strings.Builder
	sb.WriteString("Filter (Tab=next, Enter=apply, Esc=cancel):\n\n")

	labelStyle := lipgloss.NewStyle().Bold(true)
	focusedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

	colWidth := width / 2
	if colWidth < 35 {
		colWidth = 35
	}

	leftFields := []int{0, 1, 2, 3, 4}
	rightFields := []int{5, 6, 7, 8}
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

func (m Model) renderFilterField(idx int, colWidth int, labelStyle, focusedStyle lipgloss.Style) string {
	label := filterFields[idx]
	prefix := "  "
	style := labelStyle
	if idx == m.filterFocus {
		prefix = "▶ "
		style = focusedStyle
	}
	inputW := colWidth - 14
	if inputW < 10 {
		inputW = 10
	}
	m.filterInputs[idx].Width = inputW
	val := m.filterInputs[idx].View()
	return prefix + style.Render(padLabel(label)) + " " + val
}

func padLabel(l string) string {
	const w = 12
	if len(l) >= w {
		return l
	}
	return l + strings.Repeat(" ", w-len(l))
}

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
	return strings.Join(parts, " ")
}

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
	return ui.Box("Help (F1 / ?)", sb.String(), width, height)
}

// HelpView renders the full help as a string (for tests).
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

func padKey(k string) string {
	const w = 10
	if len(k) >= w {
		return k
	}
	return k + strings.Repeat(" ", w-len(k))
}
