package meta

import (
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type App interface {
	tea.Model

	Name() string

	Colours() styles.AppColours
}

type FatalErrorMsg struct {
	Error error
}

type SetActiveViewMsg struct {
	ViewType ViewType
	View     tea.Model
}

type DataLoadedMsg struct {
	TargetApp string
	ActualApp string // for asserting

	Model string
	Items []list.Item
}
