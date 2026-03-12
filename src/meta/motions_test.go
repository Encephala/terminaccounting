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
		msgs, ok := cms.Get(test.motion)
		require.True(t, ok, "expected %v to be a known normal motion", test.motion)
		require.Len(t, msgs, 1)
		assert.Equal(t, test.expected, msgs[0])
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
	require.Len(t, msgs, 1)
	assert.Equal(t, ShowNotificationsMsg{}, msgs[0])
}

func TestCompleteMotionSetFallsBackToGlobal(t *testing.T) {
	cms := NewCompleteMotionSet(Trie[tea.Msg]{})

	msgs, ok := cms.Get(Motion{"tab"})
	require.True(t, ok)
	require.Len(t, msgs, 1)
	assert.Equal(t, SwitchFocusMsg{Direction: NEXT}, msgs[0])
}

func TestGet_CountPrefixRepeatsMsg(t *testing.T) {
	var viewMotions Trie[tea.Msg]
	viewMotions.Insert(Motion{"j"}, ScrollVerticalMsg{Up: false})
	cms := NewCompleteMotionSet(viewMotions)

	tests := []struct {
		motion        Motion
		expectedCount int
	}{
		{Motion{"5", "j"}, 5},
		{Motion{"1", "j"}, 1},
		{Motion{"9", "j"}, 9},
	}

	for _, test := range tests {
		msgs, ok := cms.Get(test.motion)
		require.True(t, ok, "expected %v to resolve", test.motion)
		require.Len(t, msgs, test.expectedCount)
		for _, msg := range msgs {
			assert.Equal(t, ScrollVerticalMsg{Up: false}, msg)
		}
	}
}

func TestGet_MultiDigitCountPrefix(t *testing.T) {
	var viewMotions Trie[tea.Msg]
	viewMotions.Insert(Motion{"g", "t"}, SwitchTabMsg{Direction: NEXT})
	cms := NewCompleteMotionSet(viewMotions)

	msgs, ok := cms.Get(Motion{"1", "2", "g", "t"})
	require.True(t, ok)
	require.Len(t, msgs, 12)
	for _, msg := range msgs {
		assert.Equal(t, SwitchTabMsg{Direction: NEXT}, msg)
	}
}

func TestGet_CountPrefixNotFoundIfMotionUnknown(t *testing.T) {
	cms := NewCompleteMotionSet(Trie[tea.Msg]{})

	_, ok := cms.Get(Motion{"5", "x"})
	assert.False(t, ok)
}

func TestContainsPath_CountPrefixIsValidPrefix(t *testing.T) {
	var viewMotions Trie[tea.Msg]
	viewMotions.Insert(Motion{"j"}, ScrollVerticalMsg{Up: false})
	cms := NewCompleteMotionSet(viewMotions)

	// A leading digit is always a valid in-progress path in normal mode
	assert.True(t, cms.ContainsPath(Motion{"5"}))
	// Digits followed by a known motion prefix are also valid
	assert.True(t, cms.ContainsPath(Motion{"5", "j"}))
}
