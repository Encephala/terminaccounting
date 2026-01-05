package main

import (
	"terminaccounting/database"
	"terminaccounting/meta"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLedger_ViewSwitch(t *testing.T) {
	tw, _ := initWrapper(t)

	tw.SwitchTab(meta.NEXT)

	tw.SendText("gc")

	lastCmdResults := tw.GetLastCmdResults()

	require.Len(t, lastCmdResults, 1)
	assert.Equal(t, meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE}, lastCmdResults[0])
}

func TestCreateLedger_InsertMode(t *testing.T) {
	tw, _ := initWrapper(t)

	tw.SwitchTab(meta.NEXT).
		SwitchView(meta.CREATEVIEWTYPE)

	tw.SendText("i")

	lastCmdResults := tw.GetLastCmdResults()

	require.Len(t, lastCmdResults, 1)
	assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE}, lastCmdResults[0])
}

func TestCreateLedger_SetValues(t *testing.T) {
	tw, _ := initWrapper(t)

	tw.SwitchTab(meta.NEXT).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.INSERTMODE)

	tw.SendText("test")

	tw.AssertViewContains(t, "test")

	tw.Send(tea.KeyMsg{Type: tea.KeyTab})

	lastCmdResults := tw.GetLastCmdResults()
	require.Len(t, lastCmdResults, 1)
	assert.Equal(t, meta.SwitchFocusMsg{Direction: meta.NEXT}, lastCmdResults[0])

	tw.Send(tea.KeyMsg{Type: tea.KeyCtrlN}, tea.KeyMsg{Type: tea.KeyCtrlC})

	tw.AssertViewContains(t, "EXPENSE")
}

func TestCreateLedger_CommitCmd(t *testing.T) {
	tw, _ := initWrapper(t)

	tw.SwitchTab(meta.NEXT).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.INSERTMODE).
		SendText("test").
		Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyCtrlN}).
		SwitchMode(meta.COMMANDMODE, false)

	tw.SendText("w")
	tw.Send(tea.KeyMsg{Type: tea.KeyEnter})

	lastCmdResults := tw.GetLastCmdResults()
	// The two messages I want to test for, but also the notification/view switch etc. from handling CommitMsg
	require.Greater(t, len(lastCmdResults), 2)
	assert.Equal(t, meta.ExecuteCommandMsg{}, lastCmdResults[0])
	assert.Equal(t, meta.CommitMsg{}, lastCmdResults[1])
}

func TestCreateLedger_Commit(t *testing.T) {
	tw, DB := initWrapper(t)

	tw.SwitchTab(meta.NEXT).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.INSERTMODE).
		SendText("test").
		Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyCtrlN}).
		Send(meta.CommitMsg{})

	ledger, err := database.SelectLedger(DB, 1)
	require.Nil(t, err)
	assert.Equal(
		t,
		database.Ledger{
			Id:         1,
			Name:       "test",
			Type:       database.EXPENSELEDGER,
			Notes:      nil,
			IsAccounts: false,
		},
		ledger,
	)
}
