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

	CurrentMotionSet() *vim.MotionSet
}

type FatalErrorMsg struct {
	Error error
}

type UpdateViewMotionSetMsg *vim.MotionSet

type DataLoadedMsg struct {
	TargetApp string
	ActualApp string // for asserting

	Model string
	Items []list.Item
}
