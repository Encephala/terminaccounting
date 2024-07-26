package vim

import (
	"fmt"
)

type MotionSet struct {
	Normal  Trie
	Insert  Trie
	Command Trie
}

func (ms *MotionSet) get(mode InputMode, path Motion) (CompletedMotionMsg, bool) {
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
	MotionSet MotionSet

	ViewMotionSet *MotionSet
}

func (cms *CompleteMotionSet) Get(mode InputMode, path Motion) (CompletedMotionMsg, bool) {
	if cms.ViewMotionSet != nil {
		if msg, ok := cms.ViewMotionSet.get(mode, path); ok {
			return msg, ok
		}
	}

	return cms.MotionSet.get(mode, path)
}

func (cms *CompleteMotionSet) ContainsPath(mode InputMode, path Motion) bool {
	if cms.ViewMotionSet != nil {
		if cms.ViewMotionSet.containsPath(mode, path) {
			return true
		}
	}

	return cms.MotionSet.containsPath(mode, path)
}

type CompletedMotionMsg struct {
	Type completedMotionType
	Data interface{}
}

type completedMotionType int

const (
	NAVIGATE completedMotionType = iota
	SWITCHMODE
	SWITCHTAB
	SWITCHVIEW
	EXECUTECOMMAND
)

type motionWithValue struct {
	path  Motion
	value CompletedMotionMsg
}

type Direction int

const (
	UP Direction = iota
	RIGHT
	DOWN
	LEFT
)

type View int

const (
	LISTVIEW View = iota
	DETAILVIEW
	CREATEVIEW
)

func GlobalMotions() MotionSet {
	normalMotions := make([]motionWithValue, 0)

	// Single-stroke/no prefix
	extendMotionsBy(&normalMotions, Motion{}, []motionWithValue{
		{Motion{"h"}, CompletedMotionMsg{Type: NAVIGATE, Data: LEFT}},
		{Motion{"j"}, CompletedMotionMsg{Type: NAVIGATE, Data: DOWN}},
		{Motion{"k"}, CompletedMotionMsg{Type: NAVIGATE, Data: UP}},
		{Motion{"l"}, CompletedMotionMsg{Type: NAVIGATE, Data: RIGHT}},

		{Motion{"i"}, CompletedMotionMsg{Type: SWITCHMODE, Data: INSERTMODE}},
		{Motion{":"}, CompletedMotionMsg{Type: SWITCHMODE, Data: COMMANDMODE}},
	})

	// LEADER
	extendMotionsBy(&normalMotions, Motion{LEADER}, []motionWithValue{
		{Motion{"n"}, CompletedMotionMsg{Type: SWITCHVIEW, Data: CREATEVIEW}}, // [n]ew object
	})

	// "g"
	extendMotionsBy(&normalMotions, Motion{"g"}, []motionWithValue{
		{Motion{"t"}, CompletedMotionMsg{Type: SWITCHTAB, Data: RIGHT}}, // [g]oto Next [t]ab
		{Motion{"T"}, CompletedMotionMsg{Type: SWITCHTAB, Data: LEFT}},  // [g]oto Previous [T]ab
	})

	var normal Trie
	for _, m := range normalMotions {
		normal.Insert(m.path, m.value)
	}

	insertMotions := []motionWithValue{
		{Motion{"ctrl+c"}, CompletedMotionMsg{Type: SWITCHMODE, Data: NORMALMODE}},
	}

	var insert Trie
	for _, m := range insertMotions {
		insert.Insert(m.path, m.value)
	}

	commandMotions := []motionWithValue{
		{Motion{"enter"}, CompletedMotionMsg{Type: EXECUTECOMMAND}},
		{Motion{"ctrl+c"}, CompletedMotionMsg{Type: SWITCHMODE, Data: NORMALMODE}},
	}

	var command Trie
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
