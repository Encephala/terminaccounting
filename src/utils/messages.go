package utils

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func MessageCmd(message tea.Msg) tea.Cmd {
	return func() tea.Msg { return message }
}

type ClearErrorMsg struct{}

func ClearErrorAfterDelayCmd() tea.Msg {
	time.Sleep(time.Second * 2)

	return ClearErrorMsg{}
}
