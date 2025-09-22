package main

import (
	"fmt"
	"strings"
	"terminaccounting/meta"

	"github.com/acarl005/stripansi"
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

		result.WriteString(meta.StatusLineStyle.Render(" "))
		resultLength += 1

		motionRendered := m.currentMotion.View()
		result.WriteString(meta.CommandStyle.Render(motionRendered))
		resultLength += len(motionRendered)

	case meta.INSERTMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("12")).Padding(0, 1)
		mode := modeStyle.Render("INSERT")
		result.WriteString(mode)
		resultLength += 8 // INSERT + padding

		result.WriteString(meta.StatusLineStyle.Render(" "))
		resultLength += 1

	case meta.COMMANDMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("208")).Padding(0, 1)
		result.WriteString(modeStyle.Render("COMMAND"))
		resultLength += 9 // COMMAND + padding

		result.WriteString(meta.StatusLineStyle.Render(" "))
		resultLength += 1

		commandInputView := meta.CommandStyle.Render(m.commandInput.View())
		result.WriteString(commandInputView)
		resultLength += len(m.commandInput.Value()) + 1 + 1 // +1 for the commandInput.Prompt, and for its cursor

	default:
		panic(fmt.Sprintf("unexpected inputMode: %#v", m.inputMode))
	}

	if m.displayMessage {
		messageRendered := meta.StatusLineStyle.Render(truncate.StringWithTail(
			m.messages[len(m.messages)-1],
			uint(m.viewWidth-resultLength),
			"...",
		))
		result.WriteString(messageRendered)
		resultLength += len(stripansi.Strip(messageRendered))
	}

	if m.viewWidth-resultLength > 0 {
		result.WriteString(meta.StatusLineStyle.Render(strings.Repeat(" ", m.viewWidth-resultLength)))
	}

	return result.String()
}
