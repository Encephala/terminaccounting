package main

import (
	"errors"
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initWrapper(t *testing.T, async bool) (*tat.TestWrapper, *sqlx.DB) {
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

	if async {
		return tat.NewTestWrapperBuilder(newTerminaccounting(DB)).RunAsync(t), DB
	} else {
		return tat.NewTestWrapperBuilder(newTerminaccounting(DB)).RunSync(t), DB
	}
}

func adaptedWait(t *testing.T, wrapper *tat.TestWrapper, condition func(*terminaccounting) bool) {
	t.Helper()

	genericCondition := func(m tea.Model) bool {
		return condition(m.(*terminaccounting))
	}

	wrapper.Wait(t, genericCondition)
}

func adaptedAssertEqual(
	t *testing.T,
	wrapper *tat.TestWrapper,
	actualGetter func(*terminaccounting) any,
	expected any,
) {
	t.Helper()

	genericGetter := func(m tea.Model) any {
		return actualGetter(m.(*terminaccounting))
	}

	wrapper.AssertEqual(t, genericGetter, expected)
}

func TestSwitchModesMsg(t *testing.T) {
	tw, _ := initWrapper(t, false)

	t.Run("cannot go insert mode on list view", func(t *testing.T) {
		tw.SendText("i")

		lastCmdResults := tw.GetLastCmdResults()
		require.Len(t, lastCmdResults, 2)
		assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE}, lastCmdResults[0])
		assert.Equal(t, errors.New("current view doesn't allow insert mode"), lastCmdResults[1])
	})

	t.Run("switch insert mode", func(t *testing.T) {
		// Switch to ledgers create view
		tw.SwitchTab(meta.NEXT).
			SwitchView(meta.CREATEVIEWTYPE).
			SendText("i")

		lastCmdResults := tw.GetLastCmdResults()
		require.Len(t, lastCmdResults, 1)
		assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE}, lastCmdResults[0])
	})

	t.Run("switch back normal mode", func(t *testing.T) {
		tw.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

		lastCmdResults := tw.GetLastCmdResults()
		require.Len(t, lastCmdResults, 1)
		assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.NORMALMODE}, lastCmdResults[0])
	})

	t.Run("switch search mode", func(t *testing.T) {
		tw.SwitchView(meta.LISTVIEWTYPE).
			SendText("/")

		lastCmdResults := tw.GetLastCmdResults()
		require.Len(t, lastCmdResults, 2)
		assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: true}, lastCmdResults[0])
		assert.Equal(t, meta.UpdateSearchMsg{Query: ""}, lastCmdResults[1])
	})
}

func TestSwitchApp(t *testing.T) {
	tw, _ := initWrapper(t, true)

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

			adaptedWait(t, tw, func(ta *terminaccounting) bool {
				return ta.appManager.activeApp == tc.expectedActiveApp
			})
		})
	}
}
