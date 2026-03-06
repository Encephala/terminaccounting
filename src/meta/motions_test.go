package meta

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalNormalMotionsReachable(t *testing.T) {
	cms := NewCompleteMotionSet(MotionSet{})

	tests := []struct {
		motion   Motion
		expected tea.Msg
	}{
		{Motion{"i"}, SwitchModeMsg{InputMode: INSERTMODE}},
		{Motion{":"}, SwitchModeMsg{InputMode: COMMANDMODE, Data: false}},
		{Motion{"g", "t"}, SwitchTabMsg{Direction: NEXT}},
		{Motion{"g", "T"}, SwitchTabMsg{Direction: PREVIOUS}},
		{Motion{"esc"}, tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, test := range tests {
		msg, ok := cms.Get(NORMALMODE, test.motion)
		require.True(t, ok, "expected %v to be a known normal motion", test.motion)
		assert.Equal(t, test.expected, msg)
	}
}

func TestGlobalInsertMotionsReachable(t *testing.T) {
	cms := NewCompleteMotionSet(MotionSet{})

	tests := []struct {
		motion   Motion
		expected tea.Msg
	}{
		{Motion{"esc"}, tea.KeyMsg{Type: tea.KeyCtrlC}},
		{Motion{"tab"}, SwitchFocusMsg{Direction: NEXT}},
		{Motion{"shift+tab"}, SwitchFocusMsg{Direction: PREVIOUS}},
	}

	for _, test := range tests {
		msg, ok := cms.Get(INSERTMODE, test.motion)
		require.True(t, ok, "expected %v to be a known insert motion", test.motion)
		assert.Equal(t, test.expected, msg)
	}
}

func TestGlobalCommandMotionsReachable(t *testing.T) {
	cms := NewCompleteMotionSet(MotionSet{})

	tests := []struct {
		motion   Motion
		expected tea.Msg
	}{
		{Motion{"esc"}, tea.KeyMsg{Type: tea.KeyCtrlC}},
		{Motion{"enter"}, ExecuteCommandMsg{}},
		{Motion{"tab"}, TryCompleteCommandMsg{}},
	}

	for _, test := range tests {
		msg, ok := cms.Get(COMMANDMODE, test.motion)
		require.True(t, ok, "expected %v to be a known command-mode motion", test.motion)
		assert.Equal(t, test.expected, msg)
	}
}

func TestCompleteMotionSetContainsPath(t *testing.T) {
	cms := NewCompleteMotionSet(MotionSet{})

	assert.True(t, cms.ContainsPath(NORMALMODE, Motion{"g"}))
	assert.True(t, cms.ContainsPath(NORMALMODE, Motion{"g", "t"}))
	assert.False(t, cms.ContainsPath(NORMALMODE, Motion{"x", "y", "z"}))
}

func TestCompleteMotionSetViewTakesPriority(t *testing.T) {
	var viewNormal Trie[tea.Msg]
	viewNormal.Insert(Motion{"i"}, ShowNotificationsMsg{})
	cms := NewCompleteMotionSet(MotionSet{Normal: viewNormal})

	msg, ok := cms.Get(NORMALMODE, Motion{"i"})
	require.True(t, ok)
	assert.Equal(t, ShowNotificationsMsg{}, msg)
}

func TestCompleteMotionSetFallsBackToGlobal(t *testing.T) {
	cms := NewCompleteMotionSet(MotionSet{})

	msg, ok := cms.Get(INSERTMODE, Motion{"tab"})
	require.True(t, ok)
	assert.Equal(t, SwitchFocusMsg{Direction: NEXT}, msg)
}