package main

import (
	"fmt"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

func statusLineView(m *model) string {
	var result strings.Builder
	resultLength := 0

	switch m.inputMode {
	case meta.NORMALMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("10")).Padding(0, 1)
		result.WriteString(modeStyle.Render("NORMAL"))
		resultLength += 8 // NORMAL + padding

		result.WriteString(styles.StatusLine.Render(" "))
		resultLength += 1

		motionVisual := m.currentMotion.View()
		result.WriteString(styles.StatusLine.Render(motionVisual))
		resultLength += len(motionVisual)

	case meta.INSERTMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("12")).Padding(0, 1)
		mode := modeStyle.Render("INSERT")
		result.WriteString(mode)
		resultLength += 8 // INSERT + padding

		result.WriteString(styles.StatusLine.Render(" "))
		resultLength += 1

	case meta.COMMANDMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("208")).Padding(0, 1)
		result.WriteString(modeStyle.Render("COMMAND"))
		resultLength += 9 // COMMAND + padding

		result.WriteString(styles.StatusLine.Render(" "))
		resultLength += 1

		commandInputView := styles.Command.Render(m.commandInput.View())
		result.WriteString(commandInputView)
		resultLength += len(m.commandInput.Value()) + 1 + 1 // +1 for the commandInput.Prompt, and for its cursor

	default:
		panic(fmt.Sprintf("unexpected inputMode: %#v", m.inputMode))
	}

	maxErrorLength := 24

	numberOfEmptyCells := m.viewWidth - resultLength
	if m.displayedError != nil {
		numberOfEmptyCells -= min(len(m.displayedError.Error()), maxErrorLength) + 1 // +1 for right padding of the error
	}
	if numberOfEmptyCells >= 0 {
		result.WriteString(styles.StatusLine.Render(strings.Repeat(" ", numberOfEmptyCells)))
	}

	if m.displayedError != nil {
		result.WriteString(styles.StatusLineError.Render(
			truncate.StringWithTail(
				m.displayedError.Error(),
				uint(maxErrorLength),
				"...",
			),
		))
	}

	return result.String()
}
