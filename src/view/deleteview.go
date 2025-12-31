package view

import (
	"cmp"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type genericDeleteView interface {
	title() string

	inputValues() []string
	inputNames() []string

	getColour() lipgloss.Color
}

func genericDeleteViewView(gdv genericDeleteView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(gdv.getColour()).Padding(0, 1).MarginLeft(2)

	result.WriteString(titleStyle.Render(gdv.title()))
	result.WriteString("\n\n")

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Align(lipgloss.Left)

	names := gdv.inputNames()
	values := gdv.inputValues()

	if len(names) != len(values) {
		panic("what in the fuck")
	}

	// +2 for padding
	maxNameColWidth := len(slices.MaxFunc(names, func(name string, other string) int {
		return cmp.Compare(len(name), len(other))
	})) + 2

	for i := range names {
		result.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			sectionStyle.Width(maxNameColWidth).Render(names[i]),
			" ",
			sectionStyle.Render(values[i]),
		))

		result.WriteString("\n")
	}

	result.WriteString("\n")

	result.WriteString(lipgloss.NewStyle().Italic(true).Render("Run the `:w` command to confirm"))

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}
