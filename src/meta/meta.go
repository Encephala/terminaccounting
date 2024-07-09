package meta

import (
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type App interface {
	tea.Model

	Name() string

	Styles() styles.AppStyles
}

type FatalErrorMsg struct {
	Error error
}

type DataLoadedMsg struct {
	Model string
	Items []list.Item
}
