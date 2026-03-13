package meta

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalCommandsReachable(t *testing.T) {
	ccs := NewCompleteCommandSet(Trie[tea.Msg]{})

	tests := []struct {
		path     string
		expected tea.Msg
	}{
		{"quit", QuitMsg{}},
		{"qa", QuitMsg{All: true}},
		{"messages", ShowNotificationsMsg{}},
		{"import", ShowBankImporterMsg{}},
		{"refreshcache", RefreshCacheMsg{}},
	}

	for _, test := range tests {
		msg, ok := ccs.Get(Command(strings.Split(test.path, "")))
		require.True(t, ok, "expected %q to be a known global command", test.path)
		assert.Equal(t, test.expected, msg, "path: %q", test.path)
	}
}

func TestCompleteCommandSetContainsPath(t *testing.T) {
	ccs := NewCompleteCommandSet(Trie[tea.Msg]{})

	assert.True(t, ccs.ContainsPath(Command{"q"}))
	assert.True(t, ccs.ContainsPath(Command(strings.Split("quit", ""))))
	assert.False(t, ccs.ContainsPath(Command{"x", "y", "z"}))
}

func TestCompleteCommandSetViewTakesPriority(t *testing.T) {
	var viewCommands Trie[tea.Msg]
	viewCommands.Insert(strings.Split("quit", ""), ShowNotificationsMsg{})
	ccs := NewCompleteCommandSet(viewCommands)

	msg, ok := ccs.Get(Command(strings.Split("quit", "")))
	require.True(t, ok)
	assert.Equal(t, ShowNotificationsMsg{}, msg)
}

func TestCompleteCommandSetFallsBackToGlobal(t *testing.T) {
	ccs := NewCompleteCommandSet(Trie[tea.Msg]{})

	msg, ok := ccs.Get(Command(strings.Split("refreshcache", "")))
	require.True(t, ok)
	assert.Equal(t, RefreshCacheMsg{}, msg)
}

func TestCompleteCommandSetAutocomplete(t *testing.T) {
	ccs := NewCompleteCommandSet(Trie[tea.Msg]{})

	result := ccs.Autocomplete(strings.Split("qui", ""))
	assert.Equal(t, strings.Split("quit", ""), result)
}

func TestCompleteCommandSetAutocompleteViewTakesPriority(t *testing.T) {
	var viewCommands Trie[tea.Msg]
	viewCommands.Insert(strings.Split("query", ""), ShowNotificationsMsg{})
	ccs := NewCompleteCommandSet(viewCommands)

	result := ccs.Autocomplete(strings.Split("q", ""))
	assert.Equal(t, strings.Split("query", ""), result)
}
