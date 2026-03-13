package meta

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// This file is largely analogous to ./motions.go

// Even though a command doesn't have strokes as a Motion does (i.e. ["g", "d"]),
// still split it into its constituent characters for the Trie search
type Command []string

type CompleteCommandSet struct {
	globalCommandSet Trie[tea.Msg]
	viewCommandSet   Trie[tea.Msg]
}

func NewCompleteCommandSet(viewCommandSet Trie[tea.Msg]) CompleteCommandSet {
	return CompleteCommandSet{
		globalCommandSet: globalCommands(),
		viewCommandSet:   viewCommandSet,
	}
}

func (ccs *CompleteCommandSet) Get(path Command) (tea.Msg, bool) {
	if msg, ok := ccs.viewCommandSet.Get(path); ok {
		return msg, ok
	}

	return ccs.globalCommandSet.Get(path)
}

func (ccs *CompleteCommandSet) ContainsPath(path Command) bool {
	if ccs.viewCommandSet.ContainsPath(path) {
		return true
	}

	return ccs.globalCommandSet.ContainsPath(path)
}

func (ccs *CompleteCommandSet) Autocomplete(path Command) []string {
	viewSpecific := ccs.viewCommandSet.Autocomplete(path)

	if viewSpecific != nil {
		return viewSpecific
	}

	return ccs.globalCommandSet.Autocomplete(path)
}

type commandWithValue struct {
	path  Command
	value tea.Msg
}

func globalCommands() Trie[tea.Msg] {
	commandsToMake := make([]commandWithValue, 0)

	extendCommandsBy(&commandsToMake, Command{}, []commandWithValue{
		{Command(strings.Split("quit", "")), QuitMsg{}},
		{Command{"q", "a"}, QuitMsg{All: true}},
		{Command(strings.Split("messages", "")), ShowNotificationsMsg{}},
		{Command(strings.Split("import", "")), ShowBankImporterMsg{}},
		{Command(strings.Split("refreshcache", "")), RefreshCacheMsg{}},
		{Command(strings.Split("debugcache", "")), DebugPrintCacheMsg{}},
	})

	var commands Trie[tea.Msg]
	for _, m := range commandsToMake {
		commands.Insert(m.path, m.value)
	}

	return commands
}

func extendCommandsBy(commands *[]commandWithValue, base Command, tail []commandWithValue) {
	for _, t := range tail {
		fullPath := append(base, t.path...)
		*commands = append(*commands, commandWithValue{path: fullPath, value: t.value})
	}
}
