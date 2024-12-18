package utils

import (
	"terminaccounting/vim"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Takes a message and builds a tea.Cmd that returns that message
func MessageCmd(message tea.Msg) tea.Cmd {
	return func() tea.Msg { return message }
}

type ClearErrorMsg struct{}

func ClearErrorAfterDelayCmd() tea.Msg {
	time.Sleep(time.Second * 2)

	return ClearErrorMsg{}
}

// When inputting e.g. `j`, this gets captured as a motion,
// and gets propagated through Model.Update() calls as a completed motion.
// When passing the message back to a bubbletea model (i.e. not one I made but one from the bubbletea std),
// it has to be converted back to a keyMsg.
func NavigateMessageToKeyMsg(message vim.CompletedMotionMsg) tea.KeyMsg {
	if message.Type != vim.NAVIGATE {
		panic("I borked something")
	}

	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Alt:   false,
		Paste: false,
	}

	switch message.Data.(vim.Direction) {
	case vim.DOWN:
		keyMsg.Runes = []rune{'j'}

	case vim.UP:
		keyMsg.Runes = []rune{'k'}

	case vim.LEFT:
		keyMsg.Runes = []rune{'h'}

	case vim.RIGHT:
		keyMsg.Runes = []rune{'l'}
	}

	return keyMsg
}
