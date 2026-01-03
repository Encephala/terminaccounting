package main

import (
	"errors"
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tatesting"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initWrapper(t *testing.T) *tat.TestWrapper {
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

	result := tat.InitIntegrationTest(t, newTerminaccounting(DB))
	return &result
}

func adaptedWait(t *testing.T, wrapper *tat.TestWrapper, condition func(*terminaccounting) bool) *terminaccounting {
	t.Helper()

	genericCondition := func(m tea.Model) bool {
		return condition(m.(*terminaccounting))
	}

	return wrapper.Wait(genericCondition).(*terminaccounting)
}

func adaptedWaitQuit(t *testing.T, wrapper *tat.TestWrapper, condition func(*terminaccounting) bool) *terminaccounting {
	t.Helper()

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

func TestSwitchApp(t *testing.T) {
	wrapper := initWrapper(t)
	defer wrapper.Quit()

	// Next tab
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("t"))

	model := adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.activeApp == 1
	})

	wrapper.Lock()
	assert.Equal(t, 1, model.appManager.activeApp)
	wrapper.Unlock()

	// Wrap tabs backwards
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("T"))
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("T"))

	model = adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.activeApp == 3
	})

	wrapper.Lock()
	assert.Equal(t, 3, model.appManager.activeApp)
	wrapper.Unlock()

	// Wrap tabs forwards
	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("t"))

	model = adaptedWait(t, wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.activeApp == 0
	})

	wrapper.Lock()
	assert.Equal(t, 0, model.appManager.activeApp)
	wrapper.Unlock()
}
