package vim

type trie struct {
	key string

	isLeaf bool
	value  CompletedMotionMsg

	children []*trie
}

func (t *trie) getChild(key string) (index int, found bool) {
	for i, child := range t.children {
		if child.key == key {
			return i, true
		}
	}

	return 0, false
}

func (t *trie) Get(path []string) (CompletedMotionMsg, bool) {
	if len(path) == 0 {
		return CompletedMotionMsg{}, false
	}

	for _, value := range path {
		if i, ok := t.getChild(value); !ok {
			return CompletedMotionMsg{}, false
		} else {
			t = t.children[i]
		}
	}

	if t.isLeaf {
		return t.value, true
	} else {
		return CompletedMotionMsg{}, false
	}
}

func (t *trie) ContainsPath(path []string) bool {
	for _, value := range path {
		if i, ok := t.getChild(value); !ok {
			return false
		} else {
			t = t.children[i]
		}
	}

	return true
}

func (t *trie) Insert(path []string, value CompletedMotionMsg) (changed bool) {
	changed = false

	for i, key := range path {
		isFinalValue := i == len(path)-1
		if j, ok := t.getChild(key); ok {
			t = t.children[j]

			// Actually, if this happens, we can drop all t's children,
			// as as soon as a motion resolves, it executes
			if isFinalValue {
				// NOTE: When t.isLeaf, this does not check if the t.value is different from the provided value,
				// i.e. it is not possible to update a value in the trie
				if !t.isLeaf {
					t.isLeaf = true
					t.value = value
				}
			}

			continue
		}

		newChild := &trie{
			key:      key,
			isLeaf:   isFinalValue,
			children: []*trie{},
		}
		if isFinalValue {
			newChild.value = value
		}

		t.children = append(t.children, newChild)
		changed = true
		t = newChild
	}

	return changed
}
