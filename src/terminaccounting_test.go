package main

import (
	"errors"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitchModesMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

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

	t.Run("switch command mode", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SendText(":")

		tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: false})
	})

	t.Run("switch search mode", func(t *testing.T) {
		tw.SwitchMode(meta.NORMALMODE).
			SwitchView(meta.LISTVIEWTYPE).
			SendText("/")

		tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: true}, meta.UpdateSearchMsg{Query: ""})
	})
}

func TestSwitchModesMsg_ModalBlocksInsertMode(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.Send(meta.ShowTextModalMsg{Text: []string{"message"}})
	tw.SendText("i")

	tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE}, errors.New("current view doesn't allow insert mode"))
}

func TestNotifications_Error(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	errorMsg := "something went wrong"
	tw.Send(errors.New(errorMsg))

	tw.Execute(t, func(ta *terminaccounting) {
		require.Len(t, ta.notifications, 1)
		assert.Equal(t, errorMsg, ta.notifications[0].Text)
		assert.True(t, ta.notifications[0].IsError)
		assert.True(t, ta.displayNotification)
	})
	tw.AssertViewContains(t, errorMsg)
}

func TestNotifications_NotificationMessageMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	notificationMsg := "all good"
	tw.Send(meta.NotificationMessageMsg{Message: notificationMsg})

	tw.Execute(t, func(ta *terminaccounting) {
		require.Len(t, ta.notifications, 1)
		assert.Equal(t, notificationMsg, ta.notifications[0].Text)
		assert.False(t, ta.notifications[0].IsError)
		assert.True(t, ta.displayNotification)
	})
	tw.AssertViewContains(t, notificationMsg)
}

func TestShowNotificationsMsg(t *testing.T) {
	t.Run("no notifications returns error", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.Send(meta.ShowNotificationsMsg{})

		tw.AssertLastMsgsEqual(t, errors.New("no messages to show"))
	})

	t.Run("with notifications shows modal", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.Send(errors.New("first error")).
			Send(meta.NotificationMessageMsg{Message: "second message"}).
			Send(meta.ShowNotificationsMsg{})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.True(t, ta.showModal)
			require.Len(t, ta.notifications, 2)
		})
	})
}

func TestExecuteCommand_InvalidCommand(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.SwitchMode(meta.COMMANDMODE, false).
		SendText("zzz").
		Send(tea.KeyMsg{Type: tea.KeyEnter})

	tw.Execute(t, func(ta *terminaccounting) {
		require.NotEmpty(t, ta.notifications)
		lastNotification := ta.notifications[len(ta.notifications)-1]
		assert.Contains(t, lastNotification.Text, "invalid command")
		assert.True(t, lastNotification.IsError)
	})
}

func TestHandleKeyMsg_InvalidMotion(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	// "x" is not a defined motion in list view normal mode
	tw.SendText("x")

	tw.Execute(t, func(ta *terminaccounting) {
		require.Len(t, ta.notifications, 1)
		assert.Contains(t, ta.notifications[0].Text, "invalid motion")
		assert.True(t, ta.notifications[0].IsError)
	})
}

func TestTryCompleteCommandMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.SwitchMode(meta.COMMANDMODE, false).
		SendText("qui").
		Send(tea.KeyMsg{Type: tea.KeyTab})

	tw.Execute(t, func(ta *terminaccounting) {
		assert.Equal(t, "quit", ta.commandInput.Value())
	})
}

func TestRefreshCacheMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.Send(meta.RefreshCacheMsg{})

	tw.Execute(t, func(ta *terminaccounting) {
		assert.Empty(t, ta.notifications)
	})
}

func TestExecuteCommand_Quit(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB), tea.QuitMsg{})

	tw.SwitchMode(meta.COMMANDMODE, false).
		SendText("quit").
		Send(tea.KeyMsg{Type: tea.KeyEnter})

	tw.AssertLastMsgsEqual(t, meta.QuitMsg{}, tea.QuitMsg{})
}

func TestExecuteCommand_QuitAll(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB), tea.QuitMsg{})

	tw.SwitchMode(meta.COMMANDMODE, false).
		SendText("qa").
		Send(tea.KeyMsg{Type: tea.KeyEnter})

	tw.AssertLastMsgsEqual(t, meta.QuitMsg{All: true}, tea.QuitMsg{})
}

func TestExecuteCommand_Messages(t *testing.T) {
	t.Run("with no notifications returns error", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SwitchMode(meta.COMMANDMODE, false).
			SendText("messages").
			Send(tea.KeyMsg{Type: tea.KeyEnter})

		tw.AssertLastMsgsEqual(t, meta.ShowNotificationsMsg{}, errors.New("no messages to show"))
	})

	t.Run("with notifications opens modal", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.Send(errors.New("an error"))
		tw.SwitchMode(meta.COMMANDMODE, false).
			SendText("messages").
			Send(tea.KeyMsg{Type: tea.KeyEnter})

		tw.Execute(t, func(ta *terminaccounting) {
			assert.True(t, ta.showModal)
		})
	})
}

func TestExecuteCommand_Import(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.SwitchMode(meta.COMMANDMODE, false).
		SendText("import").
		Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Without an accounts ledger configured the importer immediately closes and errors
	tw.Execute(t, func(ta *terminaccounting) {
		require.NotEmpty(t, ta.notifications)
		assert.Contains(t, ta.notifications[len(ta.notifications)-1].Text, "accounts ledger")
	})
}

func TestExecuteCommand_CacheCommands(t *testing.T) {
	t.Run("refreshcache updates cache without error", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SwitchMode(meta.COMMANDMODE, false).
			SendText("refreshcache").
			Send(tea.KeyMsg{Type: tea.KeyEnter})

		tw.AssertLastMsgsEqual(t, meta.RefreshCacheMsg{})
	})

	t.Run("debugcache logs cache without error", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SwitchMode(meta.COMMANDMODE, false).
			SendText("debugcache").
			Send(tea.KeyMsg{Type: tea.KeyEnter})

		tw.AssertLastMsgsEqual(t, meta.DebugPrintCacheMsg{})
	})
}

func TestExecuteCommand_EmptyCommand(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.SwitchMode(meta.COMMANDMODE, false).
		Send(tea.KeyMsg{Type: tea.KeyEnter})

	tw.Execute(t, func(ta *terminaccounting) {
		assert.Empty(t, ta.notifications)
		assert.Equal(t, meta.NORMALMODE, ta.inputMode)
	})
}

func TestExecuteCommand_SearchMode(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.SwitchMode(meta.COMMANDMODE, true).
		SendText("hello").
		Send(tea.KeyMsg{Type: tea.KeyEnter})

	tw.AssertLastMsgsEqual(t, meta.UpdateSearchMsg{Query: "hello"})
}

func TestQuitMsg_ClosesModalWhenOpen(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.Send(meta.ShowTextModalMsg{Text: []string{"a message"}})
	tw.Send(meta.QuitMsg{})

	tw.Execute(t, func(ta *terminaccounting) {
		assert.False(t, ta.showModal)
	})
}

func TestFatalErrorMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB), tea.QuitMsg{})

	fatalErr := errors.New("something fatal")
	tw.Send(meta.FatalErrorMsg{Error: fatalErr})

	tw.Execute(t, func(ta *terminaccounting) {
		assert.Equal(t, fatalErr, ta.fatalError)
	})
	tw.AssertLastMsgsEqual(t, tea.QuitMsg{})
}

func TestHandleCtrlC_SearchMode(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.SwitchMode(meta.COMMANDMODE, true).
		Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	tw.AssertLastMsgsEqual(t, meta.SwitchModeMsg{InputMode: meta.NORMALMODE}, meta.UpdateSearchMsg{Query: ""})
}

func TestHandleCtrlC_ClearsCurrentMotion(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	// "g" is the start of multi-char motions (gt, gT) — pressing ctrl+c should clear the in-progress motion
	tw.SendText("g")
	tw.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	tw.Execute(t, func(ta *terminaccounting) {
		assert.Empty(t, ta.currentMotion)
	})
}

func TestRepeatingMotionUnit(t *testing.T) {
	t.Run("5gt switches tab 5 times", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SendText("5gt")

		// Starting at tab 0, 5 forward switches: 0→1→2→3→0→1 = tab 1
		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 1, ta.appManager.activeApp)
		})
	})

	t.Run("3gT switches tab backward 3 times", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SendText("3gT")

		// Starting at tab 0, 3 backward switches: 0→3→2→1 = tab 1
		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 1, ta.appManager.activeApp)
		})
	})

	t.Run("multi-digit count 12gt switches tab 12 times", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SendText("12gt")

		// Starting at tab 0, 12 forward switches: 12 % 4 = 0, final tab = 0
		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 0, ta.appManager.activeApp)
		})
	})

	t.Run("count of 1 acts same as no count", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SendText("1gt")

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 1, ta.appManager.activeApp)
		})
	})

	t.Run("count digit does not produce invalid motion notification", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SendText("5gt")

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Empty(t, ta.notifications)
		})
	})

	t.Run("ctrl+c clears pending count", func(t *testing.T) {
		DB := tat.SetupTestEnv(t)
		tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

		tw.SendText("5")
		tw.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
		tw.SendText("gt")

		// Count was cancelled, should switch only once
		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, 1, ta.appManager.activeApp)
		})
	})
}

func TestSwitchApp(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

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

			tw.Execute(t, func(ta *terminaccounting) {
				assert.Equal(t, ta.appManager.activeApp, tc.expectedActiveApp)
			})
		})
	}
}
