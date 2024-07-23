package main

import (
	"fmt"
	"strings"
	"terminaccounting/styles"
	"terminaccounting/vim"

	"github.com/charmbracelet/lipgloss"
)

func statusLineView(m *model) string {
	var result strings.Builder

	statusLineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252"))

	switch m.inputMode {
	case vim.NORMALMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("10")).Padding(0, 1)
		result.WriteString(modeStyle.Render("NORMAL"))

		result.WriteString(statusLineStyle.Render(" "))

		convertedStroke := visualMapStroke(m.currentStroke)
		joinedStroke := strings.Join(convertedStroke, "")
		result.WriteString(statusLineStyle.Render(joinedStroke))

		numberOfTrailingEmptyCells := m.viewWidth - len(joinedStroke) - 1
		if numberOfTrailingEmptyCells >= 0 {
			// This has to be in if-statement because on initial render viewWidth is 0,
			// so subtracting 1 leaves negative Repeat count
			result.WriteString(statusLineStyle.Render(strings.Repeat(" ", numberOfTrailingEmptyCells)))
		}

	case vim.INSERTMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("12")).Padding(0, 1)
		result.WriteString(modeStyle.Render("INSERT"))

	case vim.COMMANDMODE:
		result.WriteString(styles.Command.Render(m.commandInput.View()))

	default:
		panic(fmt.Sprintf("unexpected inputMode: %#v", m.inputMode))
	}

	return result.String()
}

var specialStrokes = map[string]string{
	vim.LEADER:  "<leader>",
	"backspace": "<bs>",
}

func visualMapStroke(stroke []string) []string {
	result := make([]string, len(stroke))

	for i, s := range stroke {
		mapped, ok := specialStrokes[s]
		if ok {
			result[i] = mapped
		} else {
			result[i] = s
		}
	}

	return result
}
