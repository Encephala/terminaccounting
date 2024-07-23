package meta

import (
	"terminaccounting/styles"
	"terminaccounting/vim"

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

type CompletedMotionMsg vim.Stroke

type SetActiveViewMsg struct {
	ViewType ViewType
	View     tea.Model
}

type DataLoadedMsg struct {
	Model string
	Items []list.Item
}
