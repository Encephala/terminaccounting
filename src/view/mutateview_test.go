package view

import (
	"testing"

	"terminaccounting/meta"
	"terminaccounting/tat"

	"github.com/stretchr/testify/assert"
)

func testGenericMutateView_Generic(t *testing.T, v genericMutateView, expectedTitle string, expectedInputNames []string) {
	tw := tat.NewTestWrapperSpecific(View(v))

	// Generic Rendering
	t.Run("Rendering", func(t *testing.T) {
		tw.AssertViewContains(t, expectedTitle)
		for _, name := range expectedInputNames {
			tw.AssertViewContains(t, name)
		}
	})

	// Focus Navigation
	t.Run("Focus Navigation", func(t *testing.T) {
		im := v.getInputManager()
		assert.Equal(t, 0, im.activeInput, "Initial active input should be 0")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
		assert.Equal(t, 1, im.activeInput, "Active input should be 1 after NEXT")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
		assert.Equal(t, 0, im.activeInput, "Active input should be 0 after PREVIOUS")

		// Test looping
		im.activeInput = len(im.inputs) - 1

		tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
		assert.Equal(t, 0, im.activeInput, "Active input should loop to 0 after NEXT from last input")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
		assert.Equal(t, len(im.inputs)-1, im.activeInput, "Active input should loop to last input after PREVIOUS from 0")
	})

	// Input Delegation
	t.Run("Input Delegation", func(t *testing.T) {
		im := v.getInputManager()
		// Ensure we are at the first input (Name)
		im.activeInput = 0
		im.inputs[0].focus()

		tw.SendText("test")

		val := im.inputs[0].value()
		assert.Equal(t, "test", val)
	})
}

func TestAccountsCreateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewAccountsCreateView(DB)
	testGenericMutateView_Generic(t, v, "Creating new account", []string{"Name", "Type", "Bank numbers", "Notes"})
}

func TestLedgersCreateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewLedgersCreateView(DB)
	testGenericMutateView_Generic(t, v, "Create new Ledger", []string{"Name", "Type", "Notes", "Is accounts ledger?"})
}

func TestJournalsCreateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewJournalsCreateView(DB)
	testGenericMutateView_Generic(t, v, "Creating new journal", []string{"Name", "Type", "Notes"})
}
