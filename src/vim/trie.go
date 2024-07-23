package vim

type trie struct {
	value  string
	isLeaf bool

	children []*trie
}

func (t *trie) getChild(value string) (index int, found bool) {
	for i, child := range t.children {
		if child.value == value {
			return i, true
		}
	}

	return 0, false
}

func (t *trie) Get(path []string) bool {
	if len(path) == 0 {
		return true
	}

	for _, value := range path {
		if i, ok := t.getChild(value); !ok {
			return false
		} else {
			t = t.children[i]
		}
	}

	if t.isLeaf {
		return true
	} else {
		return false
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

func (t *trie) Insert(path []string) (changed bool) {
	changed = false

	for i, value := range path {
		isFinalValue := i == len(path)-1
		if j, ok := t.getChild(value); ok {
			t = t.children[j]
			continue
		}

		newChild := &trie{
			value:    value,
			isLeaf:   isFinalValue,
			children: []*trie{},
		}
		t.children = append(t.children, newChild)
		changed = true
		t = newChild
	}

	return changed
}
