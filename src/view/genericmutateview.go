package view

import (
	"strings"
	"terminaccounting/meta"

	"github.com/charmbracelet/lipgloss"
)

type GenericMutateView interface {
	View

	title() string

	getInputManager() *inputManager

	getColours() meta.AppColours
}

func genericMutateViewView(gmv GenericMutateView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(gmv.getColours().Background).Padding(0, 1)
	result.WriteString(titleStyle.Render(gmv.title()))

	result.WriteString("\n\n")

	result.WriteString(gmv.getInputManager().View(gmv.getColours().Foreground))

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}
