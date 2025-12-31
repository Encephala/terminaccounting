package meta

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	ENTRIESCOLOUR  = lipgloss.Color("#F0F1B2")
	LEDGERSCOLOUR  = lipgloss.Color("#A1EEBD")
	ACCOUNTSCOLOUR = lipgloss.Color("#7BD4EA")
	JOURNALSCOLOUR = lipgloss.Color("#F6D6D6")
)

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

var StatusLineStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("240")).
	Foreground(lipgloss.Color("#00EAEA"))

type ListViewStyles struct {
	Title lipgloss.Style

	ListDelegateSelectedTitle lipgloss.Style
	ListDelegateSelectedDesc  lipgloss.Style
}

func NewListViewStyles(colour lipgloss.Color) ListViewStyles {
	defaultTitleStyles := list.DefaultStyles().Title
	defaultItemStyles := list.NewDefaultItemStyles()

	return ListViewStyles{
		Title: defaultTitleStyles.Background(colour),

		ListDelegateSelectedTitle: defaultItemStyles.SelectedTitle.
			Foreground(colour).
			BorderForeground(colour),
		ListDelegateSelectedDesc: defaultItemStyles.SelectedDesc.
			Foreground(colour).
			BorderForeground(colour),
	}
}

type DetailViewStyles struct {
	Title lipgloss.Style

	Item         lipgloss.Style
	ItemSelected lipgloss.Style
}

func NewDetailViewStyles(colour lipgloss.Color) DetailViewStyles {
	title := list.DefaultStyles().Title
	item := list.NewDefaultItemStyles().NormalDesc.Foreground(lipgloss.ANSIColor(7))

	return DetailViewStyles{
		Title: title.Background(colour),

		Item:         item,
		ItemSelected: item.Foreground(colour),
	}
}

var ModalStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 4)
