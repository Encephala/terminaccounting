package meta

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Takes a message and builds a tea.Cmd that returns that message.
func MessageCmd(message tea.Msg) tea.Cmd {
	return func() tea.Msg { return message }
}

type CloseViewMsg struct{}

type NotificationMessageMsg struct {
	Message string
}

type ShowNotificationsMsg struct{}

type ShowModalMsg struct {
	Message string
}

type CloseModalMsg struct{}

type FatalErrorMsg struct {
	Error error
}

type UpdateViewMotionSetMsg *MotionSet

type UpdateViewCommandSetMsg *CommandSet

type AppType string

const (
	LEDGERSAPP  AppType = "LEDGERS"
	ENTRIESAPP  AppType = "ENTRIES"
	JOURNALSAPP AppType = "JOURNALS"
	ACCOUNTSAPP AppType = "ACCOUNTS"
)

type ModelType string

const (
	LEDGERMODEL   ModelType = "LEDGER"
	ENTRYMODEL    ModelType = "ENTRY"
	ENTRYROWMODEL ModelType = "ENTRYROW"
	JOURNALMODEL  ModelType = "JOURNAL"
	ACCOUNTMODEL  ModelType = "ACCOUNT"
)

type DataLoadedMsg struct {
	TargetApp AppType

	Model ModelType
	Data  any
}

type NavigateMsg struct {
	Direction
}

type Direction string

const (
	UP    Direction = "UP"
	RIGHT Direction = "RIGHT"
	DOWN  Direction = "DOWN"
	LEFT  Direction = "LEFT"
)

// Jumping to start or end of a line (row)
// For $ and _ motions
type JumpHorizontalMsg struct {
	ToEnd bool
}

// For gg and G motions
type JumpVerticalMsg struct {
	ToEnd bool
}

type SwitchModeMsg struct {
	InputMode
	// vim treats search as command mode, so I am too
	// Data = false -> command, true -> search
	Data any
}

type Sequence string

const (
	PREVIOUS Sequence = "PREVIOUS"
	NEXT     Sequence = "NEXT"
)

type SwitchFocusMsg struct {
	Direction Sequence
}

type SwitchTabMsg struct {
	Direction Sequence
}

type ViewType string

const (
	LISTVIEWTYPE   ViewType = "LIST VIEW"
	DETAILVIEWTYPE ViewType = "DETAIL VIEW"
	CREATEVIEWTYPE ViewType = "CREATE VIEW"
	UPDATEVIEWTYPE ViewType = "UPDATE VIEW"
	DELETEVIEWTYPE ViewType = "DELETE VIEW"
)

// To switch to specific View (in specific App if provided)
type SwitchViewMsg struct {
	App      *AppType
	ViewType ViewType
	Data     any
}

type ExecuteCommandMsg struct{}

type TryCompleteCommandMsg struct{}

type ResetSearchMsg struct{}

type UpdateSearchMsg struct {
	Query string
}

// For comitting the changes from a create/update/delete view to the database
type CommitMsg struct{}

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
