package view

import (
	"fmt"
	"strings"
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type genericMutateView interface {
	View

	title() string

	getInputManager() *inputManager
}

func genericMutateViewUpdate(gmv genericMutateView, message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg, meta.SwitchFocusMsg, tea.KeyMsg:
		inputManager := gmv.getInputManager()
		inputManager, cmd := inputManager.Update(message)

		return gmv, cmd

	case meta.UpdateSearchMsg:
		inputManager := gmv.getInputManager()

		picker := inputManager.inputs[inputManager.activeInput].(*inputAdapter[itempicker.Model]).model
		picker, cmd := picker.Update(itempicker.FuzzySelectMsg{Query: message.Query})

		inputManager.inputs[inputManager.activeInput] = newAdapterFrom(picker)

		return gmv, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func genericMutateViewView(gmv genericMutateView, colour lipgloss.Color) string {
	var result strings.Builder

	result.WriteString(meta.TitleStyle.Render(gmv.title()))
	result.WriteString("\n")

	result.WriteString(gmv.getInputManager().View(colour))

	return result.String()
}
