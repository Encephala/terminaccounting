package meta

import tea "github.com/charmbracelet/bubbletea"

// This file is largely analogous to ./motions.go

type CommandSet struct {
	Commands Trie[tea.Msg]
}

// Even though a command doesn't have strokes as a Motion does (i.e. "ctrl+o"),
// still split it into its constituent characters for the Trie search
type Command []string

func (cs *CommandSet) get(path Command) (tea.Msg, bool) {
	return cs.Commands.get(path)
}

func (cs *CommandSet) containsPath(path Command) bool {
	return cs.Commands.containsPath(path)
}

type CompleteCommandSet struct {
	GlobalCommandSet CommandSet

	ViewCommandSet *CommandSet
}

func (cms *CompleteCommandSet) Get(path Command) (tea.Msg, bool) {
	if cms.ViewCommandSet != nil {
		if msg, ok := cms.ViewCommandSet.get(path); ok {
			return msg, ok
		}
	}

	return cms.GlobalCommandSet.get(path)
}

func (cms *CompleteCommandSet) ContainsPath(path Command) bool {
	if cms.ViewCommandSet != nil {
		if cms.ViewCommandSet.containsPath(path) {
			return true
		}
	}

	return cms.GlobalCommandSet.containsPath(path)
}

type commandWithValue struct {
	path  Command
	value tea.Msg
}

func GlobalCommands() CommandSet {
	commands := make([]commandWithValue, 0)

	extendCommandsBy(&commands, Command{}, []commandWithValue{
		{Command{"q"}, tea.QuitMsg{}},
	})

	var commandsTrie Trie[tea.Msg]
	for _, m := range commands {
		commandsTrie.Insert(m.path, m.value)
	}

	return CommandSet{commandsTrie}
}

func extendCommandsBy(commands *[]commandWithValue, base Command, tail []commandWithValue) {
	for _, t := range tail {
		fullPath := append(base, t.path...)
		*commands = append(*commands, commandWithValue{path: fullPath, value: t.value})
	}
}
