package main

import (
	"errors"
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func initWrapper(t *testing.T) (*tat.TestWrapper, *sqlx.DB) {
	t.Helper()

	DB := tat.SetupTestEnv(t)

	setUp, err := database.DatabaseTableIsSetUp(DB, "ledgers")
	require.Nil(t, err)
	require.True(t, setUp)

	setUp, err = database.DatabaseTableIsSetUp(DB, "accounts")
	require.Nil(t, err)
	require.True(t, setUp)

	setUp, err = database.DatabaseTableIsSetUp(DB, "entries")
	require.Nil(t, err)
	require.True(t, setUp)

	setUp, err = database.DatabaseTableIsSetUp(DB, "entryrows")
	require.Nil(t, err)
	require.True(t, setUp)

	setUp, err = database.DatabaseTableIsSetUp(DB, "journals")
	require.Nil(t, err)
	require.True(t, setUp)

	return tat.NewTestWrapperBuilder(newTerminaccounting(DB)).RunSync(t), DB
}

func adaptedAssert(
	t *testing.T,
	tw *tat.TestWrapper,
	condition func(*terminaccounting) bool,
) {
	t.Helper()

	tw.Assert(t, func(model tea.Model) bool {
		return condition(model.(*terminaccounting))
	})
}

func TestSwitchModesMsg(t *testing.T) {
	tw, _ := initWrapper(t)

	t.Run("cannot go insert mode on list view", func(t *testing.T) {
		tw.SendText("i")

		tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE}, errors.New("current view doesn't allow insert mode"))
	})

	t.Run("switch insert mode", func(t *testing.T) {
		// Switch to ledgers create view
		tw.SwitchTab(meta.NEXT).
			SwitchView(meta.CREATEVIEWTYPE).
			SendText("i")

		tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE})
	})

	t.Run("switch back normal mode", func(t *testing.T) {
		tw.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

		tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.NORMALMODE})
	})

	t.Run("switch search mode", func(t *testing.T) {
		tw.SwitchView(meta.LISTVIEWTYPE).
			SendText("/")

		tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: true}, meta.UpdateSearchMsg{Query: ""})
	})
}

func TestSwitchApp(t *testing.T) {
	tw, _ := initWrapper(t)

	testCases := []struct {
		name              string
		inputs            []string
		expectedActiveApp int
	}{
		{"switch tab simple", []string{"gt"}, 1},
		{"wrap backwards", []string{"gT", "gT"}, 3},
		{"wrap forwards", []string{"gt"}, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, input := range tc.inputs {
				tw.SendText(input)
			}

			adaptedAssert(t, tw, func(ta *terminaccounting) bool {
				return ta.appManager.activeApp == tc.expectedActiveApp
			})
		})
	}
}
