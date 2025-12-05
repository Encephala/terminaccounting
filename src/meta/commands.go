package meta

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// This file is largely analogous to ./motions.go

type CommandSet Trie[tea.Msg]

// Even though a command doesn't have strokes as a Motion does (i.e. ["g", "d"]),
// still split it into its constituent characters for the Trie search
type Command []string

func (cs *CommandSet) get(path Command) (tea.Msg, bool) {
	asTrie := Trie[tea.Msg](*cs)

	return asTrie.get(path)
}

func (cs *CommandSet) containsPath(path Command) bool {
	asTrie := Trie[tea.Msg](*cs)

	return asTrie.containsPath(path)
}

func (cs *CommandSet) autocomplete(path []string) []string {
	asTrie := Trie[tea.Msg](*cs)

	return asTrie.autocompletion(path)
}

type CompleteCommandSet struct {
	globalCommandSet CommandSet

	ViewCommandSet CommandSet
}

func DefaultCommandSet() CompleteCommandSet {
	return CompleteCommandSet{
		globalCommandSet: globalCommands(),
		ViewCommandSet:   CommandSet{},
	}
}

func (ccs *CompleteCommandSet) Get(path Command) (tea.Msg, bool) {
	if msg, ok := ccs.ViewCommandSet.get(path); ok {
		return msg, ok
	}

	return ccs.globalCommandSet.get(path)
}

func (ccs *CompleteCommandSet) ContainsPath(path Command) bool {
	if ccs.ViewCommandSet.containsPath(path) {
		return true
	}

	return ccs.globalCommandSet.containsPath(path)
}

func (ccs *CompleteCommandSet) Autocomplete(path []string) []string {
	viewSpecific := ccs.ViewCommandSet.autocomplete(path)

	if viewSpecific != nil {
		return viewSpecific
	}

	return ccs.globalCommandSet.autocomplete(path)
}

type commandWithValue struct {
	path  Command
	value tea.Msg
}

func globalCommands() CommandSet {
	commands := make([]commandWithValue, 0)

	extendCommandsBy(&commands, Command{}, []commandWithValue{
		{Command(strings.Split("quit", "")), QuitMsg{}},
		{Command{"q", "a"}, QuitMsg{All: true}},
		{Command(strings.Split("messages", "")), ShowNotificationsMsg{}},
		{Command(strings.Split("import", "")), ShowBankImporterMsg{}},
	})

	var commandsTrie Trie[tea.Msg]
	for _, m := range commands {
		commandsTrie.Insert(m.path, m.value)
	}

	return CommandSet(commandsTrie)
}

func extendCommandsBy(commands *[]commandWithValue, base Command, tail []commandWithValue) {
	for _, t := range tail {
		fullPath := append(base, t.path...)
		*commands = append(*commands, commandWithValue{path: fullPath, value: t.value})
	}
}
