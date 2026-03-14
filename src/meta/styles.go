package meta

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	ENTRIESCOLOUR  = lipgloss.Color("#7D7E00")
	LEDGERSCOLOUR  = lipgloss.Color("#3E7D56")
	ACCOUNTSCOLOUR = lipgloss.Color("#006B85")
	JOURNALSCOLOUR = lipgloss.Color("#915E5E")
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
	return lipgloss.NewStyle().
		Width(width).
		Height(height)
}

func ModalStyle(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(width).
		Height(height-2). // -2 for padding
		Padding(1, 4)
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
		Title: defaultTitleStyles.Background(colour).Foreground(lipgloss.ANSIColor(7)),

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

var TitleStyle = lipgloss.NewStyle().Margin(1, 0)
