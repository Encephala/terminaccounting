package view

import (
	"cmp"
	"slices"
	"strings"
	"terminaccounting/meta"

	"github.com/charmbracelet/lipgloss"
)

type GenericMutateView interface {
	View

	title() string

	inputs() []viewable
	inputNames() []string

	getActiveInput() *int

	getColours() meta.AppColours
}

func genericMutateViewView(gmv GenericMutateView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(gmv.getColours().Background).Padding(0, 1)
	result.WriteString(titleStyle.Render(gmv.title()))

	result.WriteString("\n\n")

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Left)
	highlightStyle := sectionStyle.Foreground(gmv.getColours().Foreground)

	names := gmv.inputNames()
	inputs := gmv.inputs()

	if len(names) != len(inputs) {
		panic("what in the fuck")
	}

	styles := slices.Repeat([]lipgloss.Style{sectionStyle}, len(names))

	styles[*gmv.getActiveInput()] = highlightStyle

	// +2 for padding
	maxNameColWidth := len(slices.MaxFunc(names, func(name string, other string) int {
		return cmp.Compare(len(name), len(other))
	})) + 2

	for i := range names {
		if names[i] == "" {
			result.WriteString(sectionStyle.Render(inputs[i].View()))
		} else {
			result.WriteString(lipgloss.JoinHorizontal(
				lipgloss.Top,
				sectionStyle.Width(maxNameColWidth).Render(names[i]),
				" ",
				styles[i].Render(inputs[i].View()),
			))
		}

		result.WriteString("\n")
	}

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}
