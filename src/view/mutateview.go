package view

import (
	"fmt"
	"strings"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type genericMutateView interface {
	View

	title() string

	getInputManager() *inputManager

	getColour() lipgloss.Color
}

func genericMutateViewUpdate(gmv genericMutateView, message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg, meta.SwitchFocusMsg, tea.KeyMsg:
		inputManager := gmv.getInputManager()
		inputManager, cmd := inputManager.Update(message)

		return gmv, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func genericMutateViewView(gmv genericMutateView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(gmv.getColour()).Padding(0, 1)
	result.WriteString(titleStyle.Render(gmv.title()))

	result.WriteString("\n\n")

	result.WriteString(gmv.getInputManager().View(gmv.getColour()))

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}
