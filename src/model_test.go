package main

import (
	"errors"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/tatesting"
	tat "terminaccounting/tatesting"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initWrapper(t *testing.T) tatesting.TestWrapper {
	t.Helper()

	DB := tat.SetupTestEnv(t)

	setUp, err := database.DatabaseTableIsSetUp(DB, "ledgers")
	require.Nil(t, err)
	require.True(t, setUp)

	setUp, err = database.DatabaseTableIsSetUp(DB, "accounts")
	require.Nil(t, err)
	require.True(t, setUp)

	setUp, err = database.DatabaseTableIsSetUp(DB, "entries")
	require.Nil(t, err)
	require.True(t, setUp)

	setUp, err = database.DatabaseTableIsSetUp(DB, "entryrows")
	require.Nil(t, err)
	require.True(t, setUp)

	setUp, err = database.DatabaseTableIsSetUp(DB, "journals")
	require.Nil(t, err)
	require.True(t, setUp)

	return tat.InitIntegrationTest(t, newTerminaccounting(DB))
}

func adaptedWait(wrapper tat.TestWrapper, condition func(*terminaccounting) bool) *terminaccounting {
	genericCondition := func(m tea.Model) bool {
		return condition(m.(*terminaccounting))
	}

	return wrapper.Wait(genericCondition).(*terminaccounting)
}

func adaptedWaitQuit(wrapper tat.TestWrapper, condition func(*terminaccounting) bool) *terminaccounting {
	genericCondition := func(m tea.Model) bool {
		return condition(m.(*terminaccounting))
	}

	return wrapper.WaitQuit(genericCondition).(*terminaccounting)
}

func TestSwitchModesMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	model := newTerminaccounting(DB)

	newModel, cmd := model.Update(tat.KeyMsg("i"))

	expectedMsg := meta.SwitchModeMsg{InputMode: meta.INSERTMODE}
	assert.Equal(t, expectedMsg, cmd())

	newModel, cmd = newModel.Update(cmd())
	assert.Equal(t, errors.New("current view doesn't allow insert mode"), cmd())

	// Switch to ledgers create view
	model = newModel.(*terminaccounting)
	model.appManager.activeApp = 1
	newModel, cmd = model.Update(meta.SwitchAppViewMsg{ViewType: meta.CREATEVIEWTYPE})
	assert.True(t, model.appManager.apps[model.appManager.activeApp].CurrentViewAllowsInsertMode())

	newModel, cmd = newModel.Update(tat.KeyMsg("i"))
	assert.Equal(t, expectedMsg, cmd())

	newModel, cmd = newModel.Update(cmd())
	assert.Nil(t, cmd)

	model = newModel.(*terminaccounting)
	assert.Equal(t, meta.INSERTMODE, model.inputMode)

	newModel, cmd = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.NORMALMODE}, cmd())

	newModel, cmd = newModel.Update(cmd())

	model = newModel.(*terminaccounting)
	assert.Equal(t, meta.NORMALMODE, model.inputMode)

	newModel, cmd = newModel.Update(meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	newModel, cmd = newModel.Update(tat.KeyMsg("/"))
	assert.Equal(t, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: true}, cmd())

	newModel, cmd = newModel.Update(cmd())

	model = newModel.(*terminaccounting)
	assert.Equal(t, meta.COMMANDMODE, model.inputMode)
	assert.True(t, model.currentCommandIsSearch)
}

func TestSwitchAppMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	model := newTerminaccounting(DB)

	newModel, cmd := model.Update(tat.KeyMsg("g"))
	model = newModel.(*terminaccounting)

	assert.Nil(t, cmd)

	newModel, cmd = newModel.Update(tat.KeyMsg("t"))

	expectedMsg := meta.SwitchTabMsg{Direction: meta.NEXT}
	assert.Equal(t, expectedMsg, cmd())

	newModel, cmd = newModel.Update(cmd())
	model = newModel.(*terminaccounting)

	assert.Equal(t, 1, model.appManager.activeApp)
}

func TestSwitchApp(t *testing.T) {
	wrapper := initWrapper(t)

	// Next tab
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("t"))

	model := adaptedWait(wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.activeApp == 1
	})

	assert.Equal(t, 1, model.appManager.activeApp)

	// Wrap tabs backwards
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("T"))
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("T"))

	model = adaptedWait(wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.activeApp == 3
	})

	assert.Equal(t, 3, model.appManager.activeApp)

	// Wrap tabs forwards
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("t"))

	model = adaptedWait(wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.activeApp == 0
	})

	assert.Equal(t, 0, model.appManager.activeApp)
}
