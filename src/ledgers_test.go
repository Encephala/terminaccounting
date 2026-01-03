package main

import (
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tatesting"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestCreateLedgerMsg(t *testing.T) {
	DB := tat.SetupTestDB(t)
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
