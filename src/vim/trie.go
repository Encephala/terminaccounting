package vim

type Trie[T any] struct {
	key string

	isLeaf bool
	value  T

	children []*Trie[T]
}

func (t *Trie[T]) getChild(key string) (index int, found bool) {
	for i, child := range t.children {
		if child.key == key {
			return i, true
		}
	}

	return 0, false
}

func (t *Trie[T]) get(path Motion) (T, bool) {
	var zeroValue T

	if len(path) == 0 {
		return zeroValue, false
	}

	for _, value := range path {
		if i, ok := t.getChild(value); !ok {
			return zeroValue, false
		} else {
			t = t.children[i]
		}
	}

	if t.isLeaf {
		return t.value, true
	} else {
		return zeroValue, false
	}
}

func (t *Trie[T]) containsPath(path Motion) bool {
	for _, value := range path {
		if i, ok := t.getChild(value); !ok {
			return false
		} else {
			t = t.children[i]
		}
	}

	return true
}

func (t *Trie[T]) Insert(path Motion, value T) (changed bool) {
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

		newChild := &Trie[T]{
			key:      key,
			isLeaf:   isFinalValue,
			children: []*Trie[T]{},
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
