package meta

import "strings"

// Very configurable yes, just change the source code
const LEADER = " "

type InputMode string

const NORMALMODE InputMode = "NORMAL"
const INSERTMODE InputMode = "INSERT"
const COMMANDMODE InputMode = "COMMAND"

type Motion []string

func (m Motion) Equals(right Motion) bool {
	if len(m) != len(right) {
		return false
	}

	for i, left := range m {
		if right[i] != left {
			return false
		}
	}

	return true
}

var specialStrokes = map[string]string{
	LEADER:      "<leader>",
	"backspace": "<bs>",
	"enter":     "<enter>",
}

// Replaces special strokes like LEADER, "backspace" with more visually pleasing variants
// for the purpose of rendering the motion.
// Then, joins the individual strokes into a string
func (m Motion) View() string {
	result := make([]string, len(m))

	for i, s := range m {
		mapped, ok := specialStrokes[s]
		if ok {
			result[i] = mapped
		} else {
			result[i] = s
		}
	}

	return strings.Join(result, "")
}
