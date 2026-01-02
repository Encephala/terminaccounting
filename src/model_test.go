package main

import (
	"terminaccounting/meta"
	"terminaccounting/tatesting"
	tat "terminaccounting/tatesting"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func initWrapper(t *testing.T) tatesting.TestWrapper {
	t.Helper()

	DB := tat.SetupTestDB(t)
	return tat.InitIntegrationTest(t, newTerminaccounting(DB))
}

func adaptedWaitQuit(wrapper tat.TestWrapper, condition func(*terminaccounting) bool) *terminaccounting {
	genericCondition := func(m tea.Model) bool {
		return condition(m.(*terminaccounting))
	}

	return wrapper.WaitQuit(genericCondition).(*terminaccounting)
}

func TestSwitchAppUnit(t *testing.T) {
	DB := tat.SetupTestDB(t)
	model := newTerminaccounting(DB)

	newModel, cmd := model.Update(tat.KeyMsg("g"))
	model = newModel.(*terminaccounting)

	assert.Nil(t, cmd)

	newModel, cmd = model.Update(tat.KeyMsg("t"))
	model = newModel.(*terminaccounting)

	assert.Equal(t, cmd(), meta.SwitchTabMsg{Direction: meta.NEXT})
}

func TestSwitchAppIntegration(t *testing.T) {
	wrapper := initWrapper(t)

	wrapper.Send(tat.KeyMsg("g"), tat.KeyMsg("t"))
	model := adaptedWaitQuit(wrapper, func(ta *terminaccounting) bool {
		return ta.appManager.activeApp == 1
	})

	assert.Equal(t, model.appManager.activeApp, 1)
}
