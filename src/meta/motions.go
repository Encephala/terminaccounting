package meta

import (
	tea "github.com/charmbracelet/bubbletea"
)

type MotionSet Trie[tea.Msg]

func (ms *MotionSet) get(path Motion) (tea.Msg, bool) {
	asTrie := Trie[tea.Msg](*ms)

	return asTrie.get(path)
}

func (ms *MotionSet) containsPath(path Motion) bool {
	asTrie := Trie[tea.Msg](*ms)
	return asTrie.containsPath(path)
}

func (ms *MotionSet) Insert(path Motion, message tea.Msg) {
	asTrie := (*Trie[tea.Msg])(ms)

	asTrie.Insert(path, message)
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

func (cms *CompleteMotionSet) Get(path Motion) (tea.Msg, bool) {
	if msg, ok := cms.viewMotionSet.get(path); ok {
		return msg, ok
	}

	return cms.globalMotionSet.get(path)
}

func (cms *CompleteMotionSet) ContainsPath(path Motion) bool {
	if cms.viewMotionSet.containsPath(path) {
		return true
	}

	return cms.globalMotionSet.containsPath(path)
}

type motionWithValue struct {
	path  Motion
	value tea.Msg
}

func globalMotions() MotionSet {
	motionsToMake := make([]motionWithValue, 0)

	// Single-stroke/no prefix
	extendMotionsBy(&motionsToMake, Motion{}, []motionWithValue{
		{Motion{"esc"}, tea.KeyMsg{Type: tea.KeyCtrlC}},
		{Motion{"i"}, SwitchModeMsg{InputMode: INSERTMODE}},
		{Motion{":"}, SwitchModeMsg{InputMode: COMMANDMODE, Data: false}}, // false -> not search mode
		{Motion{"ctrl+l"}, ReloadViewMsg{}},
	})

	// LEADER
	extendMotionsBy(&motionsToMake, Motion{LEADER}, []motionWithValue{})

	// "g"
	extendMotionsBy(&motionsToMake, Motion{"g"}, []motionWithValue{
		{Motion{"t"}, SwitchTabMsg{Direction: NEXT}},     // [g]oto Next [t]ab
		{Motion{"T"}, SwitchTabMsg{Direction: PREVIOUS}}, // [g]oto Previous [T]ab
	})

	var motions MotionSet
	for _, m := range motionsToMake {
		motions.Insert(m.path, m.value)
	}

	return motions
}

func extendMotionsBy(motions *[]motionWithValue, base Motion, tail []motionWithValue) {
	for _, t := range tail {
		fullPath := append(base, t.path...)
		*motions = append(*motions, motionWithValue{path: fullPath, value: t.value})
	}
}
