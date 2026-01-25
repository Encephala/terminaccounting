package main

import (
	"errors"
	"fmt"
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

		lastCmdResults := tw.GetLastCmdResults()
		assert.Equal(t, meta.ExecuteCommandMsg{}, lastCmdResults[0])
		assert.Equal(t, meta.CommitMsg{}, lastCmdResults[1])
		assert.Equal(t, meta.NotificationMessageMsg{Message: fmt.Sprintf("Successfully created Entry %q", "1")}, lastCmdResults[2])
	})
}

func TestCreate_Entries_Msg(t *testing.T) {
	t.Run("ViewSwitch", testCreateEntries_ViewSwitch)
	t.Run("InsertMode", testCreateEntries_InsertMode)
	t.Run("SetValues", testCreateEntries_SetValues)
	t.Run("CommitCmd", testCreateEntries_CommitCmd)
	t.Run("Commit", testCreateEntries_Commit)
}

func testCreateEntries_ViewSwitch(t *testing.T) {
	tw, _ := initWrapper(t, false)

	tw.GoToTab(meta.ENTRIESAPP).
		SendText("gc")

	tw.AssertLastCmdsEqual(t, meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE})
}

func testCreateEntries_InsertMode(t *testing.T) {
	tw, _ := initWrapper(t, false)

	tw.GoToTab(meta.ENTRIESAPP).
		SwitchView(meta.CREATEVIEWTYPE)

	tw.SendText("i")

	tw.AssertLastCmdsEqual(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE})
}

func testCreateEntries_SetValues(t *testing.T) {
	tw, DB := initWrapper(t, false)

	_, err := (&database.Journal{Name: "J1", Type: database.EXPENSEJOURNAL}).Insert(DB)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	tw.GoToTab(meta.ENTRIESAPP).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.INSERTMODE)

	tw.Send(tea.KeyMsg{Type: tea.KeyTab})
	tw.SendText("test notes")
	tw.AssertViewContains(t, "test notes")

	tw.AssertViewContains(t, "0")

	// Switch to entry rows and then to row 0 description
	tw.Send(tea.KeyMsg{Type: tea.KeyTab})
	tw.Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab})

	tw.SendText("row description")
	tw.AssertViewContains(t, "row description")
}

func testCreateEntries_CommitCmd(t *testing.T) {
	tw, _ := initWrapper(t, false)

	tw.GoToTab(meta.ENTRIESAPP).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.COMMANDMODE, false)

	tw.SendText("w")
	tw.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tw.AssertLastCmdsEqual(t,
		meta.ExecuteCommandMsg{},
		meta.CommitMsg{},
		errors.New("no journal selected (none available)"),
	)
}

func testCreateEntries_Commit(t *testing.T) {
	tw, DB := initWrapper(t, false)

	l1, err := (&database.Ledger{Name: "L1", Type: database.EXPENSELEDGER}).Insert(DB)
	require.NoError(t, err)
	j1, err := (&database.Journal{Name: "J1", Type: database.EXPENSEJOURNAL}).Insert(DB)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	tw.GoToTab(meta.ENTRIESAPP).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.INSERTMODE).
		Send(tea.KeyMsg{Type: tea.KeyTab}).
		SendText("My Entry").
		Send(tea.KeyMsg{Type: tea.KeyTab}).
		SendText("2023-01-01").
		Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}).
		SendText("Row 1").
		Send(tea.KeyMsg{Type: tea.KeyTab}).
		SendText("100").
		SwitchMode(meta.NORMALMODE).
		SendText("j_").
		SwitchMode(meta.INSERTMODE).
		Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}).
		SendText("Row 2").
		Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}).
		SendText("100").
		SwitchMode(meta.NORMALMODE).
		Send(meta.CommitMsg{})

	entries, err := database.SelectEntries(DB)
	require.Nil(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, j1, entries[0].Journal)
	assert.Equal(t, "My Entry", entries[0].Notes.Collapse())

	rows, err := database.SelectRowsByEntry(DB, entries[0].Id)
	require.Nil(t, err)
	assert.Len(t, rows, 2)
	assert.Equal(t, database.CurrencyValue(10000), rows[0].Value)
	assert.Equal(t, l1, rows[0].Ledger)
	assert.Equal(t, database.CurrencyValue(-10000), rows[1].Value)
	assert.Equal(t, l1, rows[1].Ledger)
}
