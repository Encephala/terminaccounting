package meta

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Takes a message and builds a tea.Cmd that returns that message.
func MessageCmd(message tea.Msg) tea.Cmd {
	return func() tea.Msg { return message }
}

type ClearErrorMsg struct{}

func ClearErrorAfterDelayCmd() tea.Msg {
	// TODO: This does not debounce
	time.Sleep(time.Second * 2)

	return ClearErrorMsg{}
}

type FatalErrorMsg struct {
	Error error
}

type UpdateViewMotionSetMsg *MotionSet

type UpdateViewCommandSetMsg *CommandSet

type AppType int

const (
	LEDGERS AppType = iota
	ENTRIES
	JOURNALS
	ACCOUNTS
)

type ModelType int

const (
	LEDGER ModelType = iota
	ENTRY
	ENTRYROW
	JOURNAL
	ACCOUNT
)

type DataLoadedMsg struct {
	TargetApp AppType

	Model ModelType
	Data  any
}

type NavigateMsg struct {
	Direction
}

type Direction int

const (
	UP Direction = iota
	RIGHT
	DOWN
	LEFT
)

type SwitchModeMsg struct {
	InputMode
}

type Sequence int

const (
	PREVIOUS Sequence = iota
	NEXT
)

type SwitchFocusMsg struct {
	Direction Sequence
}

type SwitchTabMsg struct {
	Direction Sequence
}

type ViewType int

const (
	LISTVIEWTYPE ViewType = iota
	DETAILVIEWTYPE
	CREATEVIEWTYPE
	UPDATEVIEWTYPE
	DELETEVIEWTYPE
)

type SwitchViewMsg struct {
	ViewType
}

type ExecuteCommandMsg struct{}

// For saving a new model to the database
type CommitCreateMsg struct{}

// For comitting the changes from an update view to the database
type CommitUpdateMsg struct{}

// For deleting a model from the database
type CommitDeleteMsg struct{}

// For resetting the value of an active input to the default value
type ResetInputFieldMsg struct{}

// When inputting e.g. `j`, this gets captured as a motion,
// and gets propagated through Model.Update() calls as a Navigate message
// When passing the message back to a bubbletea model (i.e. not one I made but one from the bubbles package),
// it has to be converted back to a keyMsg.
func NavigateMessageToKeyMsg(message NavigateMsg) tea.KeyMsg {
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Alt:   false,
		Paste: false,
	}

	switch message.Direction {
	case DOWN:
		keyMsg.Runes = []rune{'j'}

	case UP:
		keyMsg.Runes = []rune{'k'}

	case LEFT:
		keyMsg.Runes = []rune{'h'}

	case RIGHT:
		keyMsg.Runes = []rune{'l'}
	}

	return keyMsg
}
