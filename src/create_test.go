package main

import (
	"fmt"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	all_apps := []meta.AppType{
		meta.LEDGERSAPP,
		meta.JOURNALSAPP,
		meta.ACCOUNTSAPP,
	}

	for _, app := range all_apps {
		t.Run(string(app), func(t *testing.T) { t.Parallel(); testCreateGeneric(t, app) })
	}
}

func testCreateGeneric(t *testing.T, app meta.AppType) {
	tw, DB := initWrapper(t)

	tw.RunAsync(t).
		GoToTab(app)

	adaptedWait(t, tw, func(ta *terminaccounting) bool {
		return ta.appManager.apps[ta.appManager.activeApp].Type() == app
	})

	t.Run("switch to create view", func(t *testing.T) {
		tw.SendText("gc")

		adaptedWait(t, tw, func(ta *terminaccounting) bool {
			return ta.appManager.currentViewType() == meta.CREATEVIEWTYPE
		})
	})

	t.Run("enter insert mode", func(t *testing.T) {
		tw.SendText("i")

		adaptedWait(t, tw, func(ta *terminaccounting) bool {
			return ta.inputMode == meta.INSERTMODE
		})
	})

	t.Run("set model name", func(t *testing.T) {
		tw.SendText("test").
			Send(tea.KeyMsg{Type: tea.KeyCtrlC})

		adaptedWait(t, tw, func(ta *terminaccounting) bool {
			return ta.inputMode == meta.NORMALMODE
		})

		tw.AssertViewContains(t, "test")
	})

	t.Run("end commit msg", func(t *testing.T) {
		tw.SendText(":")

		adaptedWait(t, tw, func(ta *terminaccounting) bool {
			return ta.inputMode == meta.COMMANDMODE
		})

		tw.SendText("w")

		adaptedWait(t, tw, func(ta *terminaccounting) bool {
			return strings.Contains(ta.View(), ":w")
		})
	})

	t.Run("commit to database", func(t *testing.T) {
		tw.Send(tea.KeyMsg{Type: tea.KeyEnter})

		switch app {
		case meta.LEDGERSAPP:
			adaptedWait(t, tw, func(ta *terminaccounting) bool {
				return ta.appManager.currentViewType() == meta.UPDATEVIEWTYPE
			})
			// tw.Wait(t, func(tea.Model) bool {
			// 	ledgers, _ := database.SelectLedgers(DB)
			// 	return len(ledgers) > 0
			// })

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
			tw.Wait(t, func(tea.Model) bool {
				accounts, _ := database.SelectAccounts(DB)
				return len(accounts) > 0
			})

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
			tw.Wait(t, func(tea.Model) bool {
				journals, _ := database.SelectJournals(DB)
				return len(journals) > 0
			})

			journals, err := database.SelectJournals(DB)

			require.Nil(t, err)
			assert.Len(t, journals, 1)

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
