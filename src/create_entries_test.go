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

func TestCreate_Entries_Motions(t *testing.T) {
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
		tw.AssertLastCmdsEqual(t,
			view.CreateEntryRowMsg{After: true},
			meta.SwitchModeMsg{InputMode: meta.INSERTMODE},
		)
	})

	t.Run("insert row before", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SendText("O")
		tw.AssertLastCmdsEqual(t,
			view.CreateEntryRowMsg{After: false},
			meta.SwitchModeMsg{InputMode: meta.INSERTMODE},
		)
	})

	t.Run("delete row", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SendText("dd")
		tw.AssertLastCmdsEqual(t, view.DeleteEntryRowMsg{})
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
			tw.AssertLastCmdsEqual(t, meta.NavigateMsg{Direction: direction})
		}
	})

	t.Run("jump horizontal", func(t *testing.T) {
		tw.SendText("$")
		tw.AssertLastCmdsEqual(t, meta.JumpHorizontalMsg{ToEnd: true})

		tw.SendText("_")
		tw.AssertLastCmdsEqual(t, meta.JumpHorizontalMsg{ToEnd: false})
	})

	t.Run("jump vertical", func(t *testing.T) {
		tw.SendText("G")
		tw.AssertLastCmdsEqual(t, meta.JumpVerticalMsg{ToEnd: true})

		tw.SendText("gg")
		tw.AssertLastCmdsEqual(t, meta.JumpVerticalMsg{ToEnd: false})
	})

	t.Run("write error", func(t *testing.T) {
		tw.SwitchMode(meta.COMMANDMODE, false).
			SendText("w").
			Send(tea.KeyMsg{Type: tea.KeyEnter})

		results := tw.GetLastCmdResults()
		assert.Equal(t, results[:2], []tea.Msg{meta.ExecuteCommandMsg{}, meta.CommitMsg{}})

		// Error because debit/credit is not filled in in both rows
		// CBA to test the actual error itself
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

		tw.AssertLastCmdsEqual(t,
			meta.ExecuteCommandMsg{},
			meta.CommitMsg{},
			meta.NotificationMessageMsg{},
		)
	})
}
