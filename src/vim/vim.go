package vim

const LEADER = " "

type InputMode string

const NORMALMODE InputMode = "NORMAL"
const INSERTMODE InputMode = "INSERT"
const COMMANDMODE InputMode = "COMMAND"

type Stroke []string

func (s Stroke) Equals(right Stroke) bool {
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

type motionKeysType []string

func (mk motionKeysType) Contains(key string) bool {
	for _, m := range mk {
		if m == key {
			return true
		}
	}

	return false
}

var MotionKeys = motionKeysType{"j", "k"}
