package view

import (
	"fmt"
	"strings"
	"terminaccounting/model"
	"terminaccounting/styles"

	"github.com/charmbracelet/lipgloss"
)

func StatusLineView(m *model.Model) string {
	var result strings.Builder

	statusLineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252"))

	switch m.InputMode {
	case model.NORMALMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("10")).Padding(0, 1)
		result.WriteString(modeStyle.Render("NORMAL"))

		result.WriteString(statusLineStyle.Render(" "))

		convertedStroke := visualMapStroke(m.CurrentStroke)
		joinedStroke := strings.Join(convertedStroke, "")
		result.WriteString(statusLineStyle.Render(joinedStroke))

		numberOfTrailingEmptyCells := m.ViewWidth - len(joinedStroke) - 1
		if numberOfTrailingEmptyCells >= 0 {
			// This has to be in if-statement because on initial render viewWidth is 0,
			// so subtracting 1 leaves negative Repeat count
			result.WriteString(statusLineStyle.Render(strings.Repeat(" ", numberOfTrailingEmptyCells)))
		}

	case model.INSERTMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("12")).Padding(0, 1)
		result.WriteString(modeStyle.Render("INSERT"))

	case model.COMMANDMODE:
		result.WriteString(styles.Command.Render(m.CommandInput.View()))

	default:
		panic(fmt.Sprintf("unexpected inputMode: %#v", m.InputMode))
	}

	return result.String()
}

var specialStrokes = map[string]string{
	model.LEADER: "<leader>",
	"backspace":  "<bs>",
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
