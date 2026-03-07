package main

import (
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppViewRouting(t *testing.T) {
	appTypes := []meta.AppType{
		meta.LEDGERSAPP,
		meta.ACCOUNTSAPP,
		meta.JOURNALSAPP,
		meta.ENTRIESAPP,
	}

	for _, appType := range appTypes {
		t.Run(string(appType), func(t *testing.T) {
			testAppViewRouting(t, appType)
		})
	}
}

func testAppViewRouting(t *testing.T, appType meta.AppType) {
	// Insert items before creating wrapper so the cache is populated on Init.
	DB := tat.SetupTestEnv(t)
	itemId := insertItemForApp(t, DB, appType)
	detailData := detailDataForApp(appType, itemId)

	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))
	tw.GoToTab(appType)

	t.Run("create view", func(t *testing.T) {
		tw.SwitchView(meta.CREATEVIEWTYPE)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.CREATEVIEWTYPE, ta.appManager.currentViewType())
		})
	})

	t.Run("detail view", func(t *testing.T) {
		tw.SwitchView(meta.DETAILVIEWTYPE, detailData)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.DETAILVIEWTYPE, ta.appManager.currentViewType())
		})
	})

	t.Run("update view", func(t *testing.T) {
		tw.SwitchView(meta.UPDATEVIEWTYPE, itemId)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.UPDATEVIEWTYPE, ta.appManager.currentViewType())
		})
	})

	t.Run("list view", func(t *testing.T) {
		tw.SwitchView(meta.LISTVIEWTYPE)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.LISTVIEWTYPE, ta.appManager.currentViewType())
		})
	})

	t.Run("delete view", func(t *testing.T) {
		tw.SwitchView(meta.DELETEVIEWTYPE, itemId)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.DELETEVIEWTYPE, ta.appManager.currentViewType())
		})
	})
}

func TestAppManager_ReloadViewMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.Send(meta.ReloadViewMsg{})

	assert.Contains(t, tw.LastCmdResults, meta.NotificationMessageMsg{Message: "Refreshed view"})
}

func insertItemForApp(t *testing.T, DB *sqlx.DB, appType meta.AppType) int {
	t.Helper()

	switch appType {
	case meta.LEDGERSAPP:
		ledgerId, err := (&database.Ledger{Name: "Test", Type: database.EXPENSELEDGER}).Insert(DB)
		require.NoError(t, err)
		return ledgerId

	case meta.ACCOUNTSAPP:
		accountId, err := (&database.Account{Name: "Test", Type: database.DEBTOR}).Insert(DB)
		require.NoError(t, err)
		return accountId

	case meta.JOURNALSAPP:
		journalId, err := (&database.Journal{Name: "Test", Type: database.GENERALJOURNAL}).Insert(DB)
		require.NoError(t, err)
		return journalId

	case meta.ENTRIESAPP:
		journalId, err := (&database.Journal{Name: "J", Type: database.GENERALJOURNAL}).Insert(DB)
		require.NoError(t, err)
		ledgerId, err := (&database.Ledger{Name: "L", Type: database.EXPENSELEDGER}).Insert(DB)
		require.NoError(t, err)
		require.NoError(t, database.UpdateCache(DB))
		entry := database.Entry{Journal: journalId}
		entryId, err := entry.Insert(DB, []database.EntryRow{
			{Ledger: ledgerId, Value: 100, Description: "row"},
		})
		require.NoError(t, err)
		return entryId

	default:
		t.Fatalf("unexpected meta.AppType: %#v", appType)
		return 0
	}
}

func TestAppManager_YScroll(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"} {
		_, err := (&database.Ledger{Name: name, Type: database.EXPENSELEDGER}).Insert(DB)
		require.NoError(t, err)
	}

	testHeight := 10
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB)).
		GoToTab(meta.LEDGERSAPP).
		Send(tea.WindowSizeMsg{Width: 80, Height: testHeight})

	t.Run("yscroll=0 shows content from the top", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.yscroll = 0
			assert.Contains(t, ta.appManager.View(), "Alpha")
		})
	})

	t.Run("tab headers stay visible with large yscroll", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			// -3 for the top tabs
			ta.appManager.yscroll = testHeight - 3
			assert.Contains(t, ta.appManager.View(), "Ledgers")
		})
	})

	t.Run("yscroll hides content scrolled past", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			am := ta.appManager

			// Find which line of the raw app content Alpha appears on.
			// The app view is unaffected by yscroll — only appManager.View() clips it.
			appContentLines := strings.Split(am.apps[am.activeApp].View(), "\n")
			alphaLine := -1
			for i, line := range appContentLines {
				if strings.Contains(line, "Alpha") {
					alphaLine = i
					break
				}
			}
			require.GreaterOrEqual(t, alphaLine, 0, "Alpha must appear in app content")

			am.yscroll = alphaLine + 1
			scrolledView := am.View()

			assert.NotContains(t, scrolledView, "Alpha")
			assert.Contains(t, scrolledView, "Ledgers")
		})
	})
}

func TestAppManager_YScrollState(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB)).
		Send(tea.WindowSizeMsg{Width: 80, Height: 20})

	t.Run("scroll down increases yscroll", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.yscroll = 0
		})

		tw.Send(meta.ScrollVerticalMsg{Up: false})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 1, ta.appManager.yscroll)
		})
	})

	t.Run("scroll up decreases yscroll", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.yscroll = 5
		})

		tw.Send(meta.ScrollVerticalMsg{Up: true})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 4, ta.appManager.yscroll)
		})
	})

	t.Run("scroll up at 0 stays at 0", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.yscroll = 0
		})

		tw.Send(meta.ScrollVerticalMsg{Up: true})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 0, ta.appManager.yscroll)
		})
	})

	t.Run("scroll down to end", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.yscroll = 0
		})

		tw.Send(meta.ScrollVerticalMsg{Up: false, ToEnd: true})

		tw.Execute(t, func(ta *terminaccounting) {
			// terminaccounting passes height-2 to appManager, so am.height = 18
			// bodyHeight = 18 - 3 - 1 = 14, max yscroll = bodyHeight - 1 = 13
			assert.Equal(t, 13, ta.appManager.yscroll)
		})
	})

	t.Run("scroll up to start", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.yscroll = 420
		})

		tw.Send(meta.ScrollVerticalMsg{Up: true, ToEnd: true})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 0, ta.appManager.yscroll)
		})
	})
}

func TestAppManager_XScrollState(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB)).
		Send(tea.WindowSizeMsg{Width: 80, Height: 20})

	t.Run("scroll right increases xscroll", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.xscroll = 0
		})

		tw.Send(meta.ScrollHorizontalMsg{Left: false})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 1, ta.appManager.xscroll)
		})
	})

	t.Run("scroll left decreases xscroll", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.xscroll = 5
		})

		tw.Send(meta.ScrollHorizontalMsg{Left: true})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 4, ta.appManager.xscroll)
		})
	})

	t.Run("scroll left at 0 stays at 0", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.xscroll = 0
		})

		tw.Send(meta.ScrollHorizontalMsg{Left: true})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 0, ta.appManager.xscroll)
		})
	})

	t.Run("scroll right to end", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.xscroll = 0
		})

		tw.Send(meta.ScrollHorizontalMsg{Left: false, ToEnd: true})

		tw.Execute(t, func(ta *terminaccounting) {
			// bodyWidth = am.width, max xscroll = bodyWidth - 1 = 79
			assert.Equal(t, 79, ta.appManager.xscroll)
		})
	})

	t.Run("scroll left to start", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.xscroll = 420
		})

		tw.Send(meta.ScrollHorizontalMsg{Left: true, ToEnd: true})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 0, ta.appManager.xscroll)
		})
	})
}

func TestAppManager_XScroll(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"} {
		_, err := (&database.Ledger{Name: name, Type: database.EXPENSELEDGER}).Insert(DB)
		require.NoError(t, err)
	}

	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB)).
		GoToTab(meta.LEDGERSAPP).
		Send(tea.WindowSizeMsg{Width: 80, Height: 20})

	t.Run("xscroll=0 shows content from the left", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.xscroll = 0
			assert.Contains(t, ta.appManager.View(), "Alpha")
		})
	})

	t.Run("tab headers stay visible with large xscroll", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			ta.appManager.xscroll = 69
			assert.Contains(t, ta.appManager.View(), "Ledgers")
		})
	})

	t.Run("xscroll hides content scrolled past", func(t *testing.T) {
		tw.Execute(t, func(ta *terminaccounting) {
			am := ta.appManager

			// Find which column of the raw app content Alpha starts at.
			// Strip ANSI codes to get the visual column index.
			appContentLines := strings.Split(am.apps[am.activeApp].View(), "\n")
			alphaCol := -1
			for _, line := range appContentLines {
				col := strings.Index(ansi.Strip(line), "Alpha")
				if col >= 0 {
					alphaCol = col
					break
				}
			}
			require.GreaterOrEqual(t, alphaCol, 0, "Alpha must appear in app content")

			am.xscroll = alphaCol + len("Alpha")
			scrolledView := am.View()

			assert.NotContains(t, scrolledView, "Alpha")
			assert.Contains(t, scrolledView, "Ledgers")
		})
	})
}

// Each app's detail view only reads the ID from the struct, so only that field needs to be set.
func detailDataForApp(appType meta.AppType, id int) any {
	switch appType {
	case meta.LEDGERSAPP:
		return database.Ledger{Id: id}
	case meta.ACCOUNTSAPP:
		return database.Account{Id: id}
	case meta.JOURNALSAPP:
		return database.Journal{Id: id}
	case meta.ENTRIESAPP:
		return database.Entry{Id: id}
	default:
		panic("unexpected meta.AppType")
	}
}
