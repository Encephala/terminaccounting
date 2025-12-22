package view

import (
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View interface {
	tea.Model

	AcceptedModels() map[meta.ModelType]struct{}

	MotionSet() meta.MotionSet
	CommandSet() meta.CommandSet

	Reload() View
}

type viewable interface {
	View() string
}

func previousInput(currentInput *int, numInputs int) {
	*currentInput--

	if *currentInput < 0 {
		*currentInput += numInputs
	}
}

func nextInput(currentInput *int, numInputs int) {
	*currentInput++

	*currentInput %= numInputs
}

const (
	NAMEINPUT int = iota
	TYPEINPUT
	NOTEINPUT
)

func renderBoolean(reconciled bool) string {
	if reconciled {
		// Font Awesome checkbox because it's monospace, standard emoji is too wide
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("")
	} else {
		return "□"
	}
}
