package utils

import tea "github.com/charmbracelet/bubbletea"

func MessageCommand(message tea.Msg) tea.Cmd {
	return func() tea.Msg { return message }
}
