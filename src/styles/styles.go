package styles

import "github.com/charmbracelet/lipgloss"

func Tab() lipgloss.Style {
	border := lipgloss.NormalBorder()
	border.TopRight = "╮"
	border.TopLeft = "╭"

	tab := lipgloss.NewStyle().
		Border(border, true, true, false, true).
		Width(12).
		AlignHorizontal(lipgloss.Center)

	return tab
}

func Body(width, height int, accentColour lipgloss.Color) lipgloss.Style {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColour).
		Width(width).
		Height(height)

	return style
}
