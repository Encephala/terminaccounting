package meta

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type MotionSet struct {
	Normal  Trie[tea.Msg]
	Insert  Trie[tea.Msg]
	Command Trie[tea.Msg]
}

func (ms *MotionSet) get(mode InputMode, path Motion) (tea.Msg, bool) {
	switch mode {
	case NORMALMODE:
		return ms.Normal.get(path)

	case INSERTMODE:
		return ms.Insert.get(path)

	case COMMANDMODE:
		return ms.Command.get(path)

	default:
		panic(fmt.Sprintf("unexpected vim.InputMode: %#v", mode))
	}
}

func (ms *MotionSet) containsPath(mode InputMode, path Motion) bool {
	switch mode {
	case NORMALMODE:
		return ms.Normal.containsPath(path)

	case INSERTMODE:
		return ms.Insert.containsPath(path)

	case COMMANDMODE:
		return ms.Command.containsPath(path)

	default:
		panic(fmt.Sprintf("unexpected vim.InputMode: %#v", mode))
	}
}

type CompleteMotionSet struct {
	globalMotionSet MotionSet
	viewMotionSet   MotionSet
}

func NewCompleteMotionSet(viewMotionSet MotionSet) CompleteMotionSet {
	return CompleteMotionSet{
		globalMotionSet: globalMotions(),
		viewMotionSet:   viewMotionSet,
	}
}

func (cms *CompleteMotionSet) Get(mode InputMode, path Motion) (tea.Msg, bool) {
	if msg, ok := cms.viewMotionSet.get(mode, path); ok {
		return msg, ok
	}

	return cms.globalMotionSet.get(mode, path)
}

func (cms *CompleteMotionSet) ContainsPath(mode InputMode, path Motion) bool {
	if cms.viewMotionSet.containsPath(mode, path) {
		return true
	}

	return cms.globalMotionSet.containsPath(mode, path)
}

type motionWithValue struct {
	path  Motion
	value tea.Msg
}

func globalMotions() MotionSet {
	normalMotions := make([]motionWithValue, 0)

	// Single-stroke/no prefix
	extendMotionsBy(&normalMotions, Motion{}, []motionWithValue{
		{Motion{"esc"}, tea.KeyMsg{Type: tea.KeyCtrlC}},
		{Motion{"i"}, SwitchModeMsg{InputMode: INSERTMODE}},
		{Motion{":"}, SwitchModeMsg{InputMode: COMMANDMODE, Data: false}}, // false -> not search mode
		{Motion{"ctrl+l"}, ReloadViewMsg{}},
	})

	// LEADER
	extendMotionsBy(&normalMotions, Motion{LEADER}, []motionWithValue{})

	// "g"
	extendMotionsBy(&normalMotions, Motion{"g"}, []motionWithValue{
		{Motion{"t"}, SwitchTabMsg{Direction: NEXT}},     // [g]oto Next [t]ab
		{Motion{"T"}, SwitchTabMsg{Direction: PREVIOUS}}, // [g]oto Previous [T]ab
	})

	var normal Trie[tea.Msg]
	for _, m := range normalMotions {
		normal.Insert(m.path, m.value)
	}

	insertMotions := []motionWithValue{
		{Motion{"esc"}, tea.KeyMsg{Type: tea.KeyCtrlC}},
		{Motion{"tab"}, SwitchFocusMsg{Direction: NEXT}},
		{Motion{"shift+tab"}, SwitchFocusMsg{Direction: PREVIOUS}},
	}

	var insert Trie[tea.Msg]
	for _, m := range insertMotions {
		insert.Insert(m.path, m.value)
	}

	commandMotions := []motionWithValue{
		{Motion{"esc"}, tea.KeyMsg{Type: tea.KeyCtrlC}},
		{Motion{"enter"}, ExecuteCommandMsg{}},
		{Motion{"tab"}, TryCompleteCommandMsg{}},
	}

	var command Trie[tea.Msg]
	for _, m := range commandMotions {
		command.Insert(m.path, m.value)
	}

	return MotionSet{
		Normal:  normal,
		Insert:  insert,
		Command: command,
	}
}

func extendMotionsBy(motions *[]motionWithValue, base Motion, tail []motionWithValue) {
	for _, t := range tail {
		fullPath := append(base, t.path...)
		*motions = append(*motions, motionWithValue{path: fullPath, value: t.value})
	}
}
