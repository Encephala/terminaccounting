package main

import (
	"errors"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntries_Create_Motions(t *testing.T) {
	tw, DB := initWrapper(t, false)

	_, err := (&database.Ledger{Name: "L1", Type: database.EXPENSELEDGER}).Insert(DB)
	require.NoError(t, err)
	_, err = (&database.Journal{Name: "J1", Type: database.EXPENSEJOURNAL}).Insert(DB)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	tw.GoToTab(meta.ENTRIESAPP).
		SwitchView(meta.CREATEVIEWTYPE).
		// Navigate to entryrows input
		Send(tea.KeyMsg{Type: tea.KeyTab}).
		Send(tea.KeyMsg{Type: tea.KeyTab})

	t.Run("insert row after", func(t *testing.T) {
		tw.SendText("o")
		results := tw.GetLastCmdResults()
		require.Len(t, results, 2)
		assert.Equal(t, view.CreateEntryRowMsg{After: true}, results[0])
		assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE}, results[1])
	})

	t.Run("insert row before", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SendText("O")
		results := tw.GetLastCmdResults()
		require.Len(t, results, 2)
		assert.Equal(t, view.CreateEntryRowMsg{After: false}, results[0])
		assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE}, results[1])
	})

	t.Run("delete row", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SendText("dd")
		results := tw.GetLastCmdResults()
		require.Len(t, results, 1)
		assert.Equal(t, view.DeleteEntryRowMsg{}, results[0])
	})

	t.Run("navigation", func(t *testing.T) {
		motions := map[string]meta.Direction{
			"h": meta.LEFT,
			"j": meta.DOWN,
			"k": meta.UP,
			"l": meta.RIGHT,
		}

		for key, direction := range motions {
			tw.SendText(key)
			results := tw.GetLastCmdResults()
			require.Len(t, results, 1)
			assert.Equal(t, meta.NavigateMsg{Direction: direction}, results[0])
		}
	})

	t.Run("jump horizontal", func(t *testing.T) {
		tw.SendText("$")
		results := tw.GetLastCmdResults()
		require.Len(t, results, 1)
		assert.Equal(t, meta.JumpHorizontalMsg{ToEnd: true}, results[0])

		tw.SendText("_")
		results = tw.GetLastCmdResults()
		require.Len(t, results, 1)
		assert.Equal(t, meta.JumpHorizontalMsg{ToEnd: false}, results[0])
	})

	t.Run("jump vertical", func(t *testing.T) {
		tw.SendText("G")
		results := tw.GetLastCmdResults()
		require.Len(t, results, 1)
		assert.Equal(t, meta.JumpVerticalMsg{ToEnd: true}, results[0])

		tw.SendText("gg")
		results = tw.GetLastCmdResults()
		require.Len(t, results, 1)
		assert.Equal(t, meta.JumpVerticalMsg{ToEnd: false}, results[0])
	})

	t.Run("write error", func(t *testing.T) {
		tw.SwitchMode(meta.COMMANDMODE, false).
			SendText("w").
			Send(tea.KeyMsg{Type: tea.KeyEnter})

		results := tw.GetLastCmdResults()
		require.Equal(t, len(results), 3)
		assert.Equal(t, meta.ExecuteCommandMsg{}, results[0])
		assert.Equal(t, meta.CommitMsg{}, results[1])
		// Error because debit/credit is not filled in in both rows
		assert.IsType(t, errors.New(""), results[2])
	})

	t.Run("write success", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SendText("gg").
			SendText("llll")

		// Row 0: Debit 10
		tw.SwitchMode(meta.INSERTMODE).
			SendText("420").
			SwitchMode(meta.NORMALMODE)

		// Row 1: Credit 5
		tw.SendText("jl").
			SwitchMode(meta.INSERTMODE).
			SendText("69").
			SwitchMode(meta.NORMALMODE)

		// Row 2: Credit 5
		tw.SendText("j").
			SwitchMode(meta.INSERTMODE).
			SendText("351").
			SwitchMode(meta.NORMALMODE)

		tw.SwitchMode(meta.COMMANDMODE, false).
			SendText("w").
			Send(tea.KeyMsg{Type: tea.KeyEnter})

		results := tw.GetLastCmdResults()
		require.GreaterOrEqual(t, len(results), 3)
		assert.Equal(t, meta.ExecuteCommandMsg{}, results[0])
		assert.Equal(t, meta.CommitMsg{}, results[1])
		assert.IsType(t, meta.NotificationMessageMsg{}, results[2])
	})
}
