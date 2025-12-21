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

type activeInput int

func (input *activeInput) previous(numInputs int) {
	*input--

	if *input < 0 {
		*input += activeInput(numInputs)
	}
}

func (input *activeInput) next(numInputs int) {
	*input++

	*input %= activeInput(numInputs)
}

const (
	NAMEINPUT activeInput = iota
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
