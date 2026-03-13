package meta

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalMotionsReachable(t *testing.T) {
	cms := NewCompleteMotionSet(Trie[tea.Msg]{})

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
		msg, ok := cms.Get(test.motion)
		require.True(t, ok, "expected %v to be a known normal motion", test.motion)
		assert.Equal(t, test.expected, msg)
	}
}

func TestCompleteMotionSetContainsPath(t *testing.T) {
	cms := NewCompleteMotionSet(Trie[tea.Msg]{})

	assert.True(t, cms.ContainsPath(Motion{"g"}))
	assert.True(t, cms.ContainsPath(Motion{"g", "t"}))
	assert.False(t, cms.ContainsPath(Motion{"x", "y", "z"}))
}

func TestCompleteMotionSetViewTakesPriority(t *testing.T) {
	var viewMotions Trie[tea.Msg]
	viewMotions.Insert(Motion{"i"}, ShowNotificationsMsg{})
	cms := NewCompleteMotionSet(viewMotions)

	msg, ok := cms.Get(Motion{"i"})
	require.True(t, ok)
	assert.Equal(t, ShowNotificationsMsg{}, msg)
}

func TestCompleteMotionSetFallsBackToGlobal(t *testing.T) {
	cms := NewCompleteMotionSet(Trie[tea.Msg]{})

	msg, ok := cms.Get(Motion{"ctrl+l"})
	require.True(t, ok)
	assert.Equal(t, ReloadViewMsg{}, msg)
}
