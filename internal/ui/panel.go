package ui

import "github.com/charmbracelet/lipgloss"

// Box renders a titled, bordered panel. When height > 0, the box is forced to
// exactly that height (padded if short, truncated if tall). This is critical
// for split layouts where each panel must occupy a fixed number of rows.
func Box(title, content string, width, height int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Width(width)
	if height > 0 {
		style = style.Height(height).MaxHeight(height)
	}
	titled := lipgloss.NewStyle().Bold(true).Render(title)
	return style.Render(titled + "\n" + content)
}

// Dialog renders a modal-style box with a sharper border for confirmations.
// When height > 0, the box is forced to exactly that height.
func Dialog(title, content string, width, height int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 1).
		Width(width)
	if height > 0 {
		style = style.Height(height).MaxHeight(height)
	}
	titled := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).Render(title)
	return style.Render(titled + "\n" + content)
}
