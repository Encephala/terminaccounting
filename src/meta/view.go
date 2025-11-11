package meta

import tea "github.com/charmbracelet/bubbletea"

type View interface {
	tea.Model

	AcceptedModels() map[ModelType]struct{}

	MotionSet() *MotionSet
	CommandSet() *CommandSet
}
