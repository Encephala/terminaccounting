package main

import (
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestCreateLedger_ViewSwitch(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	model := newTerminaccounting(DB)

	model.appManager.activeApp = 1

	newModel, cmd := model.Update(tat.KeyMsg("g"))
	assert.Nil(t, cmd)

	newModel, cmd = newModel.Update(tat.KeyMsg("c"))

	assert.Equal(t, meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE}, cmd())

}

func TestCreateLedger_InsertMode(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	model := newTerminaccounting(DB)

	model.appManager.activeApp = 1

	newModel, cmd := model.Update(meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE})

	newModel, cmd = newModel.Update(tat.KeyMsg("i"))

	assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.INSERTMODE}, cmd())
}

func TestCreateLedger_SetValues(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	model := newTerminaccounting(DB)

	model.appManager.activeApp = 1

	newModel, cmd := model.Update(meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE})
	newModel, cmd = newModel.Update(meta.SwitchModeMsg{InputMode: meta.INSERTMODE})

	for _, char := range "test" {
		newModel, cmd = newModel.Update(tat.KeyMsg(string(char)))
	}
	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	newModel, cmd = newModel.Update(cmd())

	assert.Contains(t, newModel.View(), "test")

	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyTab})

	assert.Equal(t, meta.SwitchFocusMsg{Direction: meta.NEXT}, cmd())

	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tat.KeyMsg("i"))
	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	newModel, cmd = newModel.Update(cmd())

	assert.Contains(t, newModel.View(), "EXPENSE")
}

func TestCreateLedger_CommitCmd(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	model := newTerminaccounting(DB)

	model.appManager.activeApp = 1

	newModel, cmd := model.Update(meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE})
	newModel, cmd = newModel.Update(meta.SwitchModeMsg{InputMode: meta.INSERTMODE})

	for _, char := range "test" {
		newModel, cmd = newModel.Update(tat.KeyMsg(string(char)))
	}

	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyTab})
	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tat.KeyMsg(":"))
	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tat.KeyMsg("w"))
	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, meta.ExecuteCommandMsg{}, cmd())

	newModel, cmd = newModel.Update(cmd())

	assert.Equal(t, meta.CommitMsg{}, cmd())
}

func TestCreateLedger_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	model := newTerminaccounting(DB)

	model.appManager.activeApp = 1

	newModel, cmd := model.Update(meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE})
	newModel, cmd = newModel.Update(meta.SwitchModeMsg{InputMode: meta.INSERTMODE})

	for _, char := range "test" {
		newModel, _ = newModel.Update(tat.KeyMsg(string(char)))
	}

	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyTab})
	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tat.KeyMsg(":"))
	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tat.KeyMsg("w"))
	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	newModel, cmd = newModel.Update(cmd()) // Handle ExecuteCommandMsg

	newModel, cmd = newModel.Update(cmd()) // Handle CommitMsg

	ledger, err := database.SelectLedger(model.DB, 1)
	assert.Nil(t, err)
	assert.Equal(
		t,
		database.Ledger{
			Id:         1,
			Name:       "test",
			Type:       database.EXPENSELEDGER,
			Notes:      nil,
			IsAccounts: false,
		},
		ledger,
	)
}

func TestCreateLedger(t *testing.T) {
	wrapper := initWrapper(t).RunAsync()

	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("t"))
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("c"))

	adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.activeApp == 1 && ta.appManager.currentViewAllowsInsertMode()
	})

	wrapper.Send(tat.KeyMsg("i"))

	adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.inputMode == meta.INSERTMODE
	})

	wrapper.Send(tat.KeyMsg("t"), tat.KeyMsg("e"), tat.KeyMsg("s"), tat.KeyMsg("t"))

	wrapper.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.inputMode == meta.NORMALMODE
	})

	wrapper.AssertViewContains("test")

	wrapper.Send(tat.KeyMsg(":"))
	adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.inputMode == meta.COMMANDMODE
	})

	wrapper.Send(tat.KeyMsg("w"))
	adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return strings.Contains(ta.View(), ":w")
	})

	wrapper.Send(tat.KeyMsg("enter"))
	model := adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.currentViewType() == meta.UPDATEVIEWTYPE
	})

	ledgers, err := database.SelectLedgers(model.DB)
	assert.Nil(t, err)
	assert.Len(t, ledgers, 1)

	newLedger, err := database.SelectLedger(model.DB, 1)
	assert.Nil(t, err)

	assert.Equal(
		t,
		database.Ledger{
			Id:         1,
			Name:       "test",
			Type:       database.INCOMELEDGER,
			Notes:      nil,
			IsAccounts: false,
		},
		newLedger,
	)
}
