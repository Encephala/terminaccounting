package meta

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func isDigit(str string) bool {
	if len(str) != 1 {
		return false
	}

	return str[0] >= '0' && str[0] <= '9'
}

type CompleteMotionSet struct {
	globalMotionSet Trie[tea.Msg]
	viewMotionSet   Trie[tea.Msg]
}

func NewCompleteMotionSet(viewMotionSet Trie[tea.Msg]) CompleteMotionSet {
	return CompleteMotionSet{
		globalMotionSet: globalMotions(),
		viewMotionSet:   viewMotionSet,
	}
}

func (cms *CompleteMotionSet) Get(path Motion) ([]tea.Msg, bool) {
	var prefix []string

	for len(path) > 0 && isDigit(path[0]) {
		prefix = append(prefix, path[0])
		path = path[1:]
	}

	var count int
	if len(prefix) == 0 {
		count = 1
	} else {
		var err error
		count, err = strconv.Atoi(strings.Join(prefix, ""))
		if err != nil {
			panic(fmt.Sprintf("errrr %s", err.Error()))
		}
	}

	if msg, ok := cms.viewMotionSet.Get(path); ok {
		return slices.Repeat([]tea.Msg{msg}, count), true
	}

	msg, ok := cms.globalMotionSet.Get(path)
	if !ok {
		return nil, false
	}

	return slices.Repeat([]tea.Msg{msg}, count), true
}

func (cms *CompleteMotionSet) ContainsPath(path Motion) bool {
	for len(path) > 0 && isDigit(path[0]) {
		path = path[1:]
	}

	if cms.viewMotionSet.ContainsPath(path) {
		return true
	}

	return cms.globalMotionSet.ContainsPath(path)
}

type motionWithValue struct {
	path  Motion
	value tea.Msg
}

func globalMotions() Trie[tea.Msg] {
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

	var motions Trie[tea.Msg]
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
