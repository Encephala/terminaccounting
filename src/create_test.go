package main

import (
	"errors"
	"fmt"
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"terminaccounting/view"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateGeneric(t *testing.T) {
	all_apps := []meta.AppType{
		meta.LEDGERSAPP,
		meta.JOURNALSAPP,
		meta.ACCOUNTSAPP,
	}

	for _, app := range all_apps {
		t.Run(string(app), func(t *testing.T) { t.Parallel(); testCreateGenericHelper(t, app) })
	}
}

func testCreateGenericHelper(t *testing.T, app meta.AppType) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(app)

	tw.Execute(t, func(ta *terminaccounting) {
		assert.Equal(t, ta.appManager.apps[ta.appManager.activeApp].Type(), app)
	})

	t.Run("switch to create view", func(t *testing.T) {
		tw.SendText("gc")

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.appManager.currentViewType(), meta.CREATEVIEWTYPE)
		})
	})

	t.Run("enter insert mode", func(t *testing.T) {
		tw.SendText("i")

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.inputMode, meta.INSERTMODE)
		})
	})

	t.Run("set model name", func(t *testing.T) {
		tw.SendText("test").
			Send(tea.KeyMsg{Type: tea.KeyCtrlC})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.inputMode, meta.NORMALMODE)
		})

		tw.AssertViewContains(t, "test")
	})

	t.Run("end commit msg", func(t *testing.T) {
		tw.SendText(":")

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.inputMode, meta.COMMANDMODE)
		})

		tw.SendText("w")

		tw.AssertViewContains(t, ":w")
	})

	t.Run("commit to database", func(t *testing.T) {
		tw.Send(tea.KeyMsg{Type: tea.KeyEnter})

		switch app {
		case meta.LEDGERSAPP:
			ledgers, err := database.SelectLedgers(DB)
			require.Nil(t, err)

			assert.Len(t, ledgers, 1)

			expected := database.Ledger{
				Id:         1,
				Name:       "test",
				Type:       "INCOME",
				Notes:      nil,
				IsAccounts: false,
			}

			assert.Equal(t, expected, ledgers[0])

		case meta.ACCOUNTSAPP:
			accounts, err := database.SelectAccounts(DB)
			require.Nil(t, err)

			assert.Len(t, accounts, 1)

			expected := database.Account{
				Id:          1,
				Name:        "test",
				Type:        database.DEBTOR,
				BankNumbers: nil,
				Notes:       nil,
			}

			assert.Equal(t, expected, accounts[0])

		case meta.JOURNALSAPP:
			journals, err := database.SelectJournals(DB)
			require.Nil(t, err)

			require.Len(t, journals, 1)

			expected := database.Journal{
				Id:    1,
				Name:  "test",
				Type:  database.INCOMEJOURNAL,
				Notes: nil,
			}

			assert.Equal(t, expected, journals[0])

		default:
			panic(fmt.Sprintf("unexpected meta.AppType: %#v", app))
		}
	})
}

func TestCreateGeneric_Msg(t *testing.T) {
	all_apps := []meta.AppType{
		meta.LEDGERSAPP,
		meta.JOURNALSAPP,
		meta.ACCOUNTSAPP,
	}

	for _, app := range all_apps {
		t.Run(string(app), func(t *testing.T) { t.Parallel(); testCreate_ViewSwitch(t, app) })
		t.Run(string(app), func(t *testing.T) { t.Parallel(); testCreate_InsertMode(t, app) })
		t.Run(string(app), func(t *testing.T) { t.Parallel(); testCreate_SetValues(t, app) })
		t.Run(string(app), func(t *testing.T) { t.Parallel(); testCreate_CommitCmd(t, app) })
		t.Run(string(app), func(t *testing.T) { t.Parallel(); testCreate_Commit(t, app) })
	}
}

func testCreate_ViewSwitch(t *testing.T, app meta.AppType) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(app).
		SendText("gc")

	tw.AssertLastMsgsEqual(t, meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE})
}

func testCreate_InsertMode(t *testing.T, app meta.AppType) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(app).
		SwitchView(meta.CREATEVIEWTYPE)

	tw.SendText("i")

	tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE})
}

func testCreate_SetValues(t *testing.T, app meta.AppType) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(app).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.INSERTMODE)

	tw.SendText("test")

	tw.AssertViewContains(t, "test")

	tw.Send(tea.KeyMsg{Type: tea.KeyTab})

	tw.Send(tea.KeyMsg{Type: tea.KeyCtrlN}, tea.KeyMsg{Type: tea.KeyCtrlC})

	switch app {
	case meta.LEDGERSAPP:
		tw.AssertViewContains(t, string(database.EXPENSELEDGER))
	case meta.ACCOUNTSAPP:
		tw.AssertViewContains(t, string(database.CREDITOR))
	case meta.JOURNALSAPP:
		tw.AssertViewContains(t, string(database.EXPENSEJOURNAL))
	default:
		panic(fmt.Sprintf("unexpected meta.AppType: %#v", app))
	}
}

func testCreate_CommitCmd(t *testing.T, app meta.AppType) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(app).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.INSERTMODE).
		SendText("test").
		Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyCtrlN}).
		SwitchMode(meta.COMMANDMODE, false)

	tw.SendText("w")
	tw.Send(tea.KeyMsg{Type: tea.KeyEnter})

	require.GreaterOrEqual(t, len(tw.LastCmdResults), 1)
	assert.Equal(t, meta.CommitMsg{}, tw.LastCmdResults[0])
}

func testCreate_Commit(t *testing.T, app meta.AppType) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(app).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.INSERTMODE).
		SendText("test").
		Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyCtrlN}).
		Send(meta.CommitMsg{})

	switch app {
	case meta.LEDGERSAPP:
		ledgers, err := database.SelectLedgers(DB)

		require.Nil(t, err)
		assert.Len(t, ledgers, 1)

		expected := database.Ledger{
			Id:         1,
			Name:       "test",
			Type:       database.EXPENSELEDGER,
			Notes:      nil,
			IsAccounts: false,
		}

		assert.Equal(t, expected, ledgers[0])

	case meta.ACCOUNTSAPP:
		accounts, err := database.SelectAccounts(DB)

		require.Nil(t, err)
		assert.Len(t, accounts, 1)

		expected := database.Account{
			Id:          1,
			Name:        "test",
			Type:        database.CREDITOR,
			BankNumbers: nil,
			Notes:       nil,
		}

		assert.Equal(t, expected, accounts[0])

	case meta.JOURNALSAPP:
		journals, err := database.SelectJournals(DB)

		require.Nil(t, err)
		assert.Len(t, journals, 1)

		expected := database.Journal{
			Id:    1,
			Name:  "test",
			Type:  database.EXPENSEJOURNAL,
			Notes: nil,
		}

		assert.Equal(t, expected, journals[0])

	default:
		panic(fmt.Sprintf("unexpected meta.AppType: %#v", app))
	}
}

func TestCreate_Entries(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	l1, err := (&database.Ledger{Name: "L1", Type: database.EXPENSELEDGER}).Insert(DB)
	require.NoError(t, err)
	j1, err := (&database.Journal{Name: "J1", Type: database.EXPENSEJOURNAL}).Insert(DB)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	tw.GoToTab(meta.ENTRIESAPP)

	t.Run("switch to create view", func(t *testing.T) {
		tw.SendText("gc")

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.appManager.currentViewType(), meta.CREATEVIEWTYPE)
		})
	})

	t.Run("enter insert mode", func(t *testing.T) {
		tw.SendText("i")

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.inputMode, meta.INSERTMODE)
		})
	})

	t.Run("set values", func(t *testing.T) {
		tw.Send(tea.KeyMsg{Type: tea.KeyTab}).
			SendText("Entry Notes")

		tw.AssertViewContains(t, "Entry Notes")

		tw.Send(tea.KeyMsg{Type: tea.KeyTab}).
			// Clear prefilled date
			Send(tea.KeyMsg{Type: tea.KeyCtrlW}).
			SendText("2023-01-01").
			AssertViewContains(t, "2023-01-01")

		tw.Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}).
			SendText("Row 1").
			AssertViewContains(t, "Row 1")

		tw.Send(tea.KeyMsg{Type: tea.KeyTab}).
			SendText("100")

		tw.Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}).
			SendText("Row 2").
			AssertViewContains(t, "Row 2")

		tw.Send(tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyTab}).
			SendText("100").
			Send(tea.KeyMsg{Type: tea.KeyCtrlC})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.inputMode, meta.NORMALMODE)
		})
	})

	t.Run("end commit msg", func(t *testing.T) {
		tw.SendText(":")

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.inputMode, meta.COMMANDMODE)
		})

		tw.SendText("w")

		tw.AssertViewContains(t, ":w")
	})

	t.Run("commit to database", func(t *testing.T) {
		tw.Send(tea.KeyMsg{Type: tea.KeyEnter})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, ta.appManager.currentViewType(), meta.UPDATEVIEWTYPE)
		})

		entries, err := database.SelectEntries(DB)
		require.Nil(t, err)
		assert.Len(t, entries, 1)

		assert.Equal(t, "Entry Notes", entries[0].Notes.Collapse())
		assert.Equal(t, j1, entries[0].Journal)

		rows, err := database.SelectRowsByEntry(DB, entries[0].Id)
		require.Nil(t, err)
		assert.Len(t, rows, 2)

		assert.Equal(t, database.CurrencyValue(10000), rows[0].Value)
		assert.Equal(t, l1, rows[0].Ledger)
		assert.Equal(t, "Row 1", rows[0].Description)

		assert.Equal(t, database.CurrencyValue(-10000), rows[1].Value)
		assert.Equal(t, l1, rows[1].Ledger)
		assert.Equal(t, "Row 2", rows[1].Description)
	})
}

func TestCreate_Entries_Motions(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

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
		tw.AssertLastMsgsEqual(t,
			view.CreateEntryRowMsg{After: true},
			meta.SwitchModeMsg{InputMode: meta.INSERTMODE},
		)
	})

	t.Run("insert row before", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SendText("O")
		tw.AssertLastMsgsEqual(t,
			view.CreateEntryRowMsg{After: false},
			meta.SwitchModeMsg{InputMode: meta.INSERTMODE},
		)
	})

	t.Run("delete row", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SendText("dd")
		tw.AssertLastMsgsEqual(t, view.DeleteEntryRowMsg{})
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
			tw.AssertLastMsgsEqual(t, meta.NavigateMsg{Direction: direction})
		}
	})

	t.Run("jump horizontal", func(t *testing.T) {
		tw.SendText("$")
		tw.AssertLastMsgsEqual(t, meta.JumpHorizontalMsg{ToEnd: true})

		tw.SendText("_")
		tw.AssertLastMsgsEqual(t, meta.JumpHorizontalMsg{ToEnd: false})
	})

	t.Run("jump vertical", func(t *testing.T) {
		tw.SendText("G")
		tw.AssertLastMsgsEqual(t, meta.JumpVerticalMsg{Down: true})

		tw.SendText("gg")
		tw.AssertLastMsgsEqual(t, meta.JumpVerticalMsg{Down: false})
	})

	t.Run("write error", func(t *testing.T) {
		tw.SwitchMode(meta.COMMANDMODE, false).
			SendText("w").
			Send(tea.KeyMsg{Type: tea.KeyEnter})

		require.Len(t, tw.LastCmdResults, 2)
		assert.Equal(t, tw.LastCmdResults[0], meta.CommitMsg{})

		// Error because debit/credit is not filled in in both rows
		// CBA to test the actual error message itself
		assert.IsType(t, errors.New(""), tw.LastCmdResults[1])
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

		require.GreaterOrEqual(t, len(tw.LastCmdResults), 2)
		assert.Equal(t, meta.CommitMsg{}, tw.LastCmdResults[0])
		assert.Equal(t,
			meta.NotificationMessageMsg{Message: fmt.Sprintf("Successfully created Entry %q", "1")},
			tw.LastCmdResults[1],
		)
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
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(meta.ENTRIESAPP).
		SendText("gc")

	tw.AssertLastMsgsEqual(t, meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE})
}

func testCreateEntries_InsertMode(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(meta.ENTRIESAPP).
		SwitchView(meta.CREATEVIEWTYPE)

	tw.SendText("i")

	tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE})
}

func testCreateEntries_SetValues(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

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
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.GoToTab(meta.ENTRIESAPP).
		SwitchView(meta.CREATEVIEWTYPE).
		SwitchMode(meta.COMMANDMODE, false)

	tw.SendText("w")
	tw.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tw.AssertLastMsgsEqual(t,
		meta.CommitMsg{},
		errors.New("no journal selected (none available)"),
	)
}

func testCreateEntries_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

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
