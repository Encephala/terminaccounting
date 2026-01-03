package main

import (
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tatesting"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestCreateLedgerMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	model := newTerminaccounting(DB)

	// Switch ledgers create view
	model.appManager.activeApp = 1
	newModel, cmd := model.Update(tat.KeyMsg("g"))
	newModel, cmd = newModel.Update(tat.KeyMsg("c"))

	var expectedMsg tea.Msg = meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE}
	assert.Equal(t, expectedMsg, cmd())

	newModel, cmd = newModel.Update(cmd())

	newModel, cmd = newModel.Update(tat.KeyMsg("i"))
	newModel, cmd = newModel.Update(cmd())

	model = newModel.(*terminaccounting)
	assert.Equal(t, meta.INSERTMODE, model.inputMode)

	newModel, cmd = newModel.Update(tat.KeyMsg("t"))
	newModel, cmd = newModel.Update(tat.KeyMsg("e"))
	newModel, cmd = newModel.Update(tat.KeyMsg("s"))
	newModel, cmd = newModel.Update(tat.KeyMsg("t"))
	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	newModel, cmd = newModel.Update(cmd())

	model = newModel.(*terminaccounting)
	assert.Equal(t, meta.NORMALMODE, model.inputMode)
	assert.Contains(t, model.View(), "test")

	newModel, cmd = newModel.Update(tat.KeyMsg(":"))
	newModel, cmd = newModel.Update(cmd())

	model = newModel.(*terminaccounting)
	assert.Equal(t, meta.COMMANDMODE, model.inputMode)

	newModel, cmd = newModel.Update(tat.KeyMsg("w"))
	newModel, cmd = newModel.Update(tat.KeyMsg("enter"))

	assert.Equal(t, meta.ExecuteCommandMsg{}, cmd())
	newModel, cmd = newModel.Update(cmd())

	assert.Equal(t, meta.CommitMsg{}, cmd())
	newModel, cmd = newModel.Update(cmd())

	commands := cmd().(tea.BatchMsg)

	assert.Len(t, commands, 2)
	newModel, cmd = newModel.Update(commands[0])
	newModel, cmd = newModel.Update(commands[1])

	assert.Equal(t, meta.SwitchAppViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: 1}, cmd())

	newModel, cmd = newModel.Update(cmd())

	ledger, err := database.SelectLedger(model.DB, 1)
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
		ledger,
	)
}

func TestCreateLedgerIntegration(t *testing.T) {
	wrapper := initWrapper(t)
	defer wrapper.Quit()

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

	model := adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.inputMode == meta.NORMALMODE
	})

	wrapper.Lock()
	assert.Contains(t, model.View(), "test")
	wrapper.Unlock()

	wrapper.Send(tat.KeyMsg(":"))
	adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.inputMode == meta.COMMANDMODE
	})

	wrapper.Send(tat.KeyMsg("w"))
	adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return strings.Contains(ta.View(), ":w")
	})

	wrapper.Send(tat.KeyMsg("enter"))
	model = adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
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
