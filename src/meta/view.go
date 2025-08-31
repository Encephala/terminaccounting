package meta

import tea "github.com/charmbracelet/bubbletea"

type View interface {
	tea.Model

	MotionSet() *MotionSet
	CommandSet() *CommandSet
}
