package main

import (
	"fmt"
	"strings"
	"terminaccounting/meta"

	"github.com/charmbracelet/lipgloss"
)

func (ta *terminaccounting) statusLineView() string {
	var result strings.Builder
	resultLength := 0

	switch ta.inputMode {
	case meta.NORMALMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("10")).Padding(0, 1)
		result.WriteString(modeStyle.Render("NORMAL"))
		resultLength += len("NORMAL") + 2 // Padding

		result.WriteString(meta.StatusLineStyle.Render(" "))
		resultLength += 1

		motionRendered := ta.currentMotion.View()
		result.WriteString(meta.StatusLineStyle.Render(motionRendered))
		resultLength += len(motionRendered)

	case meta.INSERTMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("12")).Padding(0, 1)
		mode := modeStyle.Render("INSERT")
		result.WriteString(mode)
		resultLength += len("INSERT") + 2 // Padding

		result.WriteString(meta.StatusLineStyle.Render(" "))
		resultLength += 1

	case meta.COMMANDMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("208")).Padding(0, 1)
		result.WriteString(modeStyle.Render("COMMAND"))
		resultLength += len("COMMAND") + 2 // Padding

		result.WriteString(meta.StatusLineStyle.Render(" "))
		resultLength += 1

	default:
		panic(fmt.Sprintf("unexpected inputMode: %#v", ta.inputMode))
	}

	if ta.width-resultLength > 0 {
		result.WriteString(meta.StatusLineStyle.Render(strings.Repeat(" ", ta.width-resultLength)))
	}

	return result.String()
}

func (ta *terminaccounting) commandLineView() string {
	var result strings.Builder

	if ta.inputMode == meta.COMMANDMODE {
		commandInputView := ta.commandInput.View()
		result.WriteString(commandInputView)
	}

	if ta.displayNotification {
		notification := ta.notifications[len(ta.notifications)-1]

		result.WriteString(notification.String())
	}

	return result.String()
}
