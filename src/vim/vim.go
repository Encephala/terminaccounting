package vim

const LEADER = " "

type InputMode string

const NORMALMODE InputMode = "NORMAL"
const INSERTMODE InputMode = "INSERT"
const COMMANDMODE InputMode = "COMMAND"

type Motion []string

func (s Motion) Equals(right Motion) bool {
	if len(s) != len(right) {
		return false
	}

	for i, left := range s {
		if right[i] != left {
			return false
		}
	}

	return true
}
