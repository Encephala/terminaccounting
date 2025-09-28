package main

import (
	"fmt"
	"strings"
	"terminaccounting/meta"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

func (ta *terminaccounting) statusLineView() string {
	var result strings.Builder
	resultLength := 0

	switch ta.inputMode {
	case meta.NORMALMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("10")).Padding(0, 1)
		result.WriteString(modeStyle.Render("NORMAL"))
		resultLength += 8 // NORMAL + padding

		result.WriteString(meta.StatusLineStyle.Render(" "))
		resultLength += 1

		motionRendered := ta.currentMotion.View()
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

		commandInputView := meta.CommandStyle.Render(ta.commandInput.View())
		result.WriteString(commandInputView)
		resultLength += len(ta.commandInput.Value()) + 1 + 1 // +1 for the commandInput.Prompt, and for its cursor

	default:
		panic(fmt.Sprintf("unexpected inputMode: %#v", ta.inputMode))
	}

	if ta.displayNotification {
		notification := ta.notifications[len(ta.notifications)-1]

		if notification.isError {
			result.WriteString(meta.StatusLineErrorStyle.Render(truncate.StringWithTail(
				notification.text,
				uint(ta.viewWidth-resultLength),
				"...",
			)))
		} else {
			result.WriteString(meta.StatusLineStyle.Render(truncate.StringWithTail(
				ta.notifications[len(ta.notifications)-1].text,
				uint(ta.viewWidth-resultLength),
				"...",
			)))
		}

		resultLength += len(notification.text)
	}

	if ta.viewWidth-resultLength > 0 {
		result.WriteString(meta.StatusLineStyle.Render(strings.Repeat(" ", ta.viewWidth-resultLength)))
	}

	return result.String()
}
