package vim

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

func Motions() Trie {
	var result Trie

	motions := make([]motionWithValue, 0)

	// Single-stroke/no prefix
	extendMotionsBy(&motions, Motion{}, []motionWithValue{
		{Motion{"h"}, CompletedMotionMsg{Type: NAVIGATE, Data: LEFT}},
		{Motion{"j"}, CompletedMotionMsg{Type: NAVIGATE, Data: DOWN}},
		{Motion{"k"}, CompletedMotionMsg{Type: NAVIGATE, Data: UP}},
		{Motion{"l"}, CompletedMotionMsg{Type: NAVIGATE, Data: RIGHT}},

		{Motion{"i"}, CompletedMotionMsg{Type: SWITCHMODE, Data: INSERTMODE}},
		{Motion{":"}, CompletedMotionMsg{Type: SWITCHMODE, Data: COMMANDMODE}},
		{Motion{"ctrl+c"}, CompletedMotionMsg{Type: SWITCHMODE, Data: NORMALMODE}},

		{Motion{"ctrl+o"}, CompletedMotionMsg{Type: SWITCHVIEW, Data: LISTVIEW}}, // Go back to list view
	})

	// LEADER
	extendMotionsBy(&motions, Motion{LEADER}, []motionWithValue{
		{Motion{"n"}, CompletedMotionMsg{Type: SWITCHVIEW, Data: CREATEVIEW}}, // [n]ew object
	})

	// "g"
	extendMotionsBy(&motions, Motion{"g"}, []motionWithValue{
		{Motion{"t"}, CompletedMotionMsg{Type: SWITCHTAB, Data: RIGHT}},       // [g]oto Next [t]ab
		{Motion{"T"}, CompletedMotionMsg{Type: SWITCHTAB, Data: LEFT}},        // [g]oto Previous [T]ab
		{Motion{"d"}, CompletedMotionMsg{Type: SWITCHVIEW, Data: DETAILVIEW}}, // [g]oto [d]etails
	})

	for _, m := range motions {
		result.Insert(m.path, m.value)
	}

	return result
}

func extendMotionsBy(motions *[]motionWithValue, base Motion, tail []motionWithValue) {
	for _, t := range tail {
		fullPath := append(base, t.path...)
		*motions = append(*motions, motionWithValue{path: fullPath, value: t.value})
	}
}
