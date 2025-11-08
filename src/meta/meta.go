// Global types and behaviour that is generic for each App
package meta

import (
	tea "github.com/charmbracelet/bubbletea"
)

type App interface {
	tea.Model

	Name() string

	Colours() AppColours

	CurrentMotionSet() *MotionSet
	CurrentCommandSet() *CommandSet

	AcceptedModels() map[ModelType]struct{}

	MakeLoadListCmd() tea.Cmd
	MakeLoadRowsCmd(modelId int) tea.Cmd
}
