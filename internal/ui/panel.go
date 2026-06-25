package ui

import "github.com/charmbracelet/lipgloss"

// Box renders a titled, bordered panel of the given width. The content is
// placed inside a rounded border with a bold title.
func Box(title, content string, width int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Width(width)
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
