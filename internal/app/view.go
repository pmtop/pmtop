package app

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View renders the TUI: top status bar, port table (or filter form/search
// input), and bottom hint/status bar.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	width := m.width
	if width <= 0 {
		width = 120
	}

	top := m.style.StatusBar(m.version, m.root, m.paused, intervalString(m.interval), m.filt.Summary(), width)

	switch m.mode {
	case modeSearch:
		body := m.tbl.View()
		searchLine := "Search: " + m.searchInput.View()
		return top + "\n" + body + "\n" + searchLine
	case modeFilter:
		return top + "\n" + m.filterFormView(width) + "\n" + m.filterFormHints(width)
	}

	body := m.tbl.View()
	bottom := m.hints(width)
	if m.statusMsg != "" && (m.statusExp.IsZero() || time.Now().Before(m.statusExp)) {
		bottom = m.statusMsg
	} else if m.err != nil {
		bottom = "error: " + m.err.Error()
	}
	return top + "\n" + body + "\n" + bottom
}

// filterFormView renders the filter form fields with the focused one emphasized.
func (m Model) filterFormView(width int) string {
	var sb strings.Builder
	sb.WriteString("Filter (Tab=next, Enter=apply, Esc=cancel):\n\n")
	labelStyle := lipgloss.NewStyle().Bold(true)
	focusedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	for i, label := range filterFields {
		prefix := "  "
		style := labelStyle
		if i == m.filterFocus {
			prefix = "▶ "
			style = focusedStyle
		}
		val := m.filterInputs[i].View()
		sb.WriteString(prefix + style.Render(padLabel(label)) + " " + val + "\n")
	}
	return sb.String()
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
	return m.style.HintBar(hint, width)
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

// HelpView renders the full F1 help overlay (used by the help action in M2 it
// is returned as a string for tests/future wiring).
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
