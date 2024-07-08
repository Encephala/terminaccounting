package meta

import (
	"terminaccounting/styles"

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
	Type  string
	Items []interface{}
}
