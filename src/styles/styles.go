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
		// -2s for the borders
		Width(width - 2).
		Height(height - 2)

	return style
}

func Command() lipgloss.Style {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF"))

	return style
}
