package main

import (
	"errors"
	"terminaccounting/meta"
	"terminaccounting/view"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntries_Create_Motions(t *testing.T) {
	tw, _ := initWrapper(t, false)

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

	t.Run("write", func(t *testing.T) {
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
}
