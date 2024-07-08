package styles

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

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

type AppStyles struct {
	Foreground, Accent, Background lipgloss.Color
}

type ListViewStyles struct {
	Title lipgloss.Style

	ListDelegateSelectedTitle lipgloss.Style
	ListDelegateSelectedDesc  lipgloss.Style
}

func NewListViewStyles(background, foreground lipgloss.Color) ListViewStyles {
	defaultTitleStyles := list.DefaultStyles().Title
	defaultItemStyles := list.NewDefaultItemStyles()

	result := ListViewStyles{
		Title: defaultTitleStyles.Background(background),

		ListDelegateSelectedTitle: defaultItemStyles.SelectedTitle.
			Foreground(foreground).
			BorderForeground(background),
		ListDelegateSelectedDesc: defaultItemStyles.SelectedDesc.
			Foreground(foreground).
			BorderForeground(background),
	}

	return result
}
