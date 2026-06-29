package ui

import "github.com/charmbracelet/lipgloss"

// Box renders a titled, bordered panel of the given width. If maxHeight > 0,
// the content is truncated to fit within that height.
func Box(title, content string, width, maxHeight int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Width(width)
	if maxHeight > 0 {
		border = border.MaxHeight(maxHeight)
	}
	titled := lipgloss.NewStyle().Bold(true).Render(title)
	return border.Render(titled + "\n" + content)
}

// Dialog renders a centered modal-style box (sharper border) for confirmations.
func Dialog(title, content string, width int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 1).
		Width(width)
	titled := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).Render(title)
	return border.Render(titled + "\n" + content)
}

// BoxWithHeight renders a bordered panel with an explicit height for scrolling.
// The content is the visible portion after applying scrollOffset.
func BoxWithHeight(title, content string, width, height int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Width(width).
		Height(height)
	titled := lipgloss.NewStyle().Bold(true).Render(title)
	return border.Render(titled + "\n" + content)
}
