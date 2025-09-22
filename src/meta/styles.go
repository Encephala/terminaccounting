package meta

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var ENTRIESCOLOURS = AppColours{
	Foreground: "#F0F1B2D0",
	Accent:     "#F0F1B280",
	Background: "#EBECABFF",
}

var tabBorder = lipgloss.Border{
	Top:         "─",
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "┴",
	BottomRight: "┴",
}

func TabStyle(activeTabAccentColour lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(tabBorder).
		BorderBottomForeground(activeTabAccentColour).
		Width(12).
		AlignHorizontal(lipgloss.Center)
}

var activeTabBorder = lipgloss.Border{
	Top:         "─",
	Bottom:      " ",
	Left:        "│",
	Right:       "│",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "┘",
	BottomRight: "└",
}

func ActiveTabStyle(accentColour lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(activeTabBorder).
		BorderForeground(accentColour).
		Width(12).
		AlignHorizontal(lipgloss.Center)
}

func BodyStyle(width, height int) lipgloss.Style {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height)

	return style
}

var CommandStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#00FFFF")).
	Background(lipgloss.Color("240"))

var StatusLineStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("240")).
	Foreground(lipgloss.Color("252"))

var StatusLineErrorStyle = StatusLineStyle.
	Foreground(lipgloss.Color("9"))

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

	Item         lipgloss.Style
	ItemSelected lipgloss.Style
}

func NewDetailViewStyles(colours AppColours) DetailViewStyles {
	// TODO: Update to match new listdelegate
	title := list.DefaultStyles().Title
	item := list.NewDefaultItemStyles().NormalDesc.Foreground(lipgloss.ANSIColor(7))

	return DetailViewStyles{
		Title: title.Background(colours.Background),

		Item:         item,
		ItemSelected: item.Foreground(ENTRIESCOLOURS.Foreground),
	}
}

var LEDGERSCOLOURS = AppColours{
	Foreground: "#A1EEBDD0",
	Background: "#A1EEBD60",
	Accent:     "#A1EEBDFF",
}
