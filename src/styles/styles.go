package styles

import "github.com/charmbracelet/lipgloss"

func Tab() lipgloss.Style {
	tabBorder := lipgloss.NormalBorder()
	tabBorder.TopRight = "╮"
	tabBorder.TopLeft = "╭"

	tab := lipgloss.NewStyle().
		Border(tabBorder)

	return tab
}

func ActiveTab() lipgloss.Style {
	return Tab().BorderForeground(lipgloss.Color("#A1EEBD60"))
}
