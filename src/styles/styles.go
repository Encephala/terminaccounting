package styles

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var ENTRIESCOLOURS = AppColours{
	Foreground: "#F0F1B2D0",
	Accent:     "#F0F1B280",
	Background: "#EBECABFF",
}

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

type AppColours struct {
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

	return ListViewStyles{
		Title: defaultTitleStyles.Background(background),

		ListDelegateSelectedTitle: defaultItemStyles.SelectedTitle.
			Foreground(foreground).
			BorderForeground(background),
		ListDelegateSelectedDesc: defaultItemStyles.SelectedDesc.
			Foreground(foreground).
			BorderForeground(background),
	}
}

type DetailViewStyles struct {
	Title lipgloss.Style

	ListDelegateSelectedTitle lipgloss.Style
	ListDelegateSelectedDesc  lipgloss.Style
}

func NewDetailViewStyles(background lipgloss.Color) DetailViewStyles {
	defaultTitleStyles := list.DefaultStyles().Title
	defaultItemStyles := list.NewDefaultItemStyles()

	return DetailViewStyles{
		Title: defaultTitleStyles.Background(background),

		ListDelegateSelectedTitle: defaultItemStyles.SelectedTitle.
			Foreground(ENTRIESCOLOURS.Foreground).
			BorderForeground(ENTRIESCOLOURS.Background),
		ListDelegateSelectedDesc: defaultItemStyles.SelectedDesc.
			Foreground(ENTRIESCOLOURS.Foreground).
			BorderForeground(ENTRIESCOLOURS.Background),
	}
}
