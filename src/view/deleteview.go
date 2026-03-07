package view

import (
	"cmp"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type genericDeleteView interface {
	inputValues() []string
	inputNames() []string
}

func genericDeleteViewView(gdv genericDeleteView, width, height int) string {
	var result strings.Builder

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
	greatestNameColWidth := len(slices.MaxFunc(names, func(name string, other string) int {
		return cmp.Compare(len(name), len(other))
	})) + 2

	// -4 for MarginLeft and implicit MarginRight, -2 for padding, -1 for PaddingRight
	remainingValueWidth := width - 4 - greatestNameColWidth - 2 - 1

	for i := range names {
		// -4 for padding and border on both sides
		truncatedValue := ansi.Truncate(values[i], remainingValueWidth-4, "...")

		result.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			sectionStyle.Width(greatestNameColWidth).MarginRight(1).Render(names[i]),
			sectionStyle.Render(truncatedValue),
		))

		result.WriteString("\n")
	}

	result.WriteString("\n")

	result.WriteString(lipgloss.NewStyle().Italic(true).Render("Run the `:w` command to confirm"))

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}
